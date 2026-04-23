package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultRuntimeRoot = "/var/lib/waf"
const defaultHealthAddr = "127.0.0.1:8081"
const runtimeAuthHeader = "X-WAF-Runtime-Token"
const bootstrapUIUpstream = "http://ui:80"

var errActivePointerMissing = errors.New("active/current.json is required")
var runtimeProm = newRuntimeMetrics()

type activePointer struct {
	RevisionID    string `json:"revision_id"`
	CandidatePath string `json:"candidate_path"`
}

type runtimeStatus struct {
	mu               sync.RWMutex
	pointerLoaded    bool
	bundleValidated  bool
	nginxRunning     bool
	activeRevisionID string
	activeBundlePath string
}

func (s *runtimeStatus) setActiveBundle(pointer *activePointer, candidatePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pointerLoaded = true
	s.bundleValidated = true
	s.activeRevisionID = pointer.RevisionID
	s.activeBundlePath = candidatePath
	runtimeProm.setActiveRevision(pointer.RevisionID)
	runtimeProm.setReady(true)
}

func (s *runtimeStatus) setBundleUnavailable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pointerLoaded = false
	s.bundleValidated = false
	s.activeRevisionID = ""
	s.activeBundlePath = ""
	runtimeProm.setActiveRevision("")
	runtimeProm.setReady(false)
}

func (s *runtimeStatus) setNginxRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nginxRunning = running
	runtimeProm.setReady(running && s.pointerLoaded && s.bundleValidated && s.activeRevisionID != "" && s.activeBundlePath != "")
}

func (s *runtimeStatus) live() bool {
	return true
}

func (s *runtimeStatus) ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pointerLoaded && s.bundleValidated && s.nginxRunning && s.activeRevisionID != "" && s.activeBundlePath != ""
}

type runtimeProcess struct {
	mu          sync.Mutex
	runtimeRoot string
	crsPath     string
	modulePaths []string
	crsManager  *crsManager
	status      *runtimeStatus
	cmd         *exec.Cmd
	exitCh      chan error
}

func newRuntimeProcess(runtimeRoot, crsPath string, status *runtimeStatus, manager *crsManager, modulePaths ...string) *runtimeProcess {
	return &runtimeProcess{
		runtimeRoot: runtimeRoot,
		crsPath:     crsPath,
		modulePaths: modulePaths,
		crsManager:  manager,
		status:      status,
		exitCh:      make(chan error, 1),
	}
}

func (p *runtimeProcess) bootCurrent() error {
	pointer, candidatePath, err := p.loadCurrentBundle()
	if err != nil {
		runtimeProm.recordBundleLoad("failed")
		return err
	}
	if err := prepareRuntimeLayout(candidatePath, p.crsPath); err != nil {
		runtimeProm.recordBundleLoad("failed")
		return err
	}
	p.status.setActiveBundle(pointer, candidatePath)
	if err := p.startOrReloadLocked(); err != nil {
		runtimeProm.recordBundleLoad("failed")
		return err
	}
	runtimeProm.recordBundleLoad("succeeded")
	return nil
}

func (p *runtimeProcess) bootBootstrapUI() error {
	if err := writeBootstrapNginxConfig("/etc/waf/nginx", bootstrapUIUpstream); err != nil {
		return err
	}
	return p.startOrReloadLocked()
}

func (p *runtimeProcess) reloadCurrent() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pointer, candidatePath, err := p.loadCurrentBundle()
	if err != nil {
		runtimeProm.recordReload("failed")
		return err
	}
	if err := prepareRuntimeLayout(candidatePath, p.crsPath); err != nil {
		runtimeProm.recordReload("failed")
		return err
	}
	p.status.setActiveBundle(pointer, candidatePath)
	if err := p.startOrReloadLocked(); err != nil {
		runtimeProm.recordReload("failed")
		return err
	}
	runtimeProm.recordReload("succeeded")
	return nil
}

func (p *runtimeProcess) setCRSPath(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.crsPath = strings.TrimSpace(path)
}

func (p *runtimeProcess) loadCurrentBundle() (*activePointer, string, error) {
	active, err := loadActivePointer(p.runtimeRoot)
	if err != nil {
		if errors.Is(err, errActivePointerMissing) {
			p.status.setBundleUnavailable()
		}
		return nil, "", err
	}

	candidatePath, err := resolveCandidatePath(p.runtimeRoot, active.CandidatePath)
	if err != nil {
		return nil, "", err
	}
	if err := validateCandidateBundle(candidatePath); err != nil {
		return nil, "", err
	}
	return active, candidatePath, nil
}

func (p *runtimeProcess) startOrReloadLocked() error {
	if err := applyL4Guard(); err != nil {
		return err
	}
	globalDirective := p.nginxGlobalDirective("/etc/waf/nginx/nginx.conf", true)
	if p.cmd == nil || p.cmd.Process == nil || p.cmd.ProcessState != nil {
		args := []string{"-p", "/etc/waf/nginx", "-c", "nginx.conf"}
		if strings.TrimSpace(globalDirective) != "" {
			args = append(args, "-g", globalDirective)
		}
		cmd := exec.Command("nginx", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		p.cmd = cmd
		p.status.setNginxRunning(true)
		go p.waitForExit(cmd)
		return nil
	}
	return p.cmd.Process.Signal(syscall.SIGHUP)
}

func writeBootstrapNginxConfig(rootDir, uiUpstream string) error {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return errors.New("bootstrap nginx root is required")
	}
	uiUpstream = strings.TrimSpace(uiUpstream)
	if uiUpstream == "" {
		return errors.New("bootstrap ui upstream is required")
	}
	confDir := filepath.Join(rootDir, "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return err
	}
	nginxConf := `worker_processes auto;
events {
    worker_connections 1024;
}
http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    access_log    /var/log/nginx/access.log;
    error_log     /var/log/nginx/error.log;
    include /etc/waf/nginx/conf.d/*.conf;
}
`
	bootstrapConf := fmt.Sprintf(`server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    absolute_redirect off;
    resolver 127.0.0.11 ipv6=off valid=30s;

    location / {
        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
`, uiUpstream)
	if err := os.WriteFile(filepath.Join(rootDir, "nginx.conf"), []byte(nginxConf), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(confDir, "bootstrap.conf"), []byte(bootstrapConf), 0o644); err != nil {
		return err
	}
	return nil
}

func (p *runtimeProcess) nginxGlobalDirective(configPath string, daemonOff bool) string {
	directives := make([]string, 0, len(p.modulePaths)+1)
	content, err := os.ReadFile(configPath)
	configText := ""
	if err == nil {
		configText = string(content)
	}
	if !strings.Contains(configText, "pid ") {
		directives = append(directives, "pid /etc/waf/nginx/nginx.pid;")
	}
	for _, modulePath := range p.modulePaths {
		modulePath = strings.TrimSpace(modulePath)
		if modulePath == "" {
			continue
		}
		moduleBase := filepath.Base(modulePath)
		if strings.Contains(configText, modulePath) || (moduleBase != "" && strings.Contains(configText, moduleBase)) {
			continue
		}
		directives = append(directives, fmt.Sprintf("load_module %s;", modulePath))
	}
	if daemonOff {
		directives = append(directives, "daemon off;")
	}
	return strings.Join(directives, " ")
}

func (p *runtimeProcess) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()
	p.status.setNginxRunning(false)
	p.mu.Lock()
	if p.cmd == cmd {
		p.cmd = nil
	}
	p.mu.Unlock()
	p.exitCh <- err
}

func (s *runtimeStatus) handlers(process *runtimeProcess, securitySource *securityEventSource, requestSource *requestStreamSource) http.Handler {
	mux := http.NewServeMux()
	requireRuntimeAccess := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !runtimeRequestAuthorized(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		runtimeProm.setLive(true)
		if !s.live() {
			http.Error(w, "not live", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if !s.ready() {
			runtimeProm.setReady(false)
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		runtimeProm.setReady(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}))
	mux.Handle("/metrics", runtimeProm)
	mux.HandleFunc("/reload", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := process.reloadCurrent(); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.HandleFunc("/security-events", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := securitySource.next()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"events": items,
		})
	}))
	mux.HandleFunc("/security-events/probe", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := securitySource.probe(); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/requests", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		items, err := requestSource.latest(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, items)
	}))
	mux.HandleFunc("/requests/probe", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := requestSource.probe(r.URL.Query()); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/requests/indexes", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			if err := requestSource.deleteIndex(r.URL.Query()); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload, err := requestSource.indexes(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}))
	mux.HandleFunc("/crs/status", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if process.crsManager == nil {
			http.Error(w, "crs manager not configured", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, process.crsManager.Status())
	}))
	mux.HandleFunc("/crs/check-updates", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if process.crsManager == nil {
			http.Error(w, "crs manager not configured", http.StatusServiceUnavailable)
			return
		}
		dryRun := false
		if body, err := decodeJSONBody(r); err == nil && body != nil {
			if raw, ok := body["dry_run"].(bool); ok {
				dryRun = raw
			}
		}
		status, err := process.crsManager.CheckForUpdates(dryRun)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, status)
	}))
	mux.HandleFunc("/crs/update", requireRuntimeAccess(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if process.crsManager == nil {
			http.Error(w, "crs manager not configured", http.StatusServiceUnavailable)
			return
		}
		body, err := decodeJSONBody(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		update := false
		if body != nil {
			if raw, ok := body["enable_hourly_auto_update"].(bool); ok {
				update = true
				process.crsManager.SetHourlyAutoUpdate(raw)
			}
		}
		if update {
			writeJSON(w, http.StatusOK, process.crsManager.Status())
			return
		}
		status, changed, err := process.crsManager.UpdateToLatest(false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if changed {
			process.setCRSPath(status.ActivePath)
			if err := process.reloadCurrent(); err != nil && !errors.Is(err, errActivePointerMissing) {
				http.Error(w, fmt.Sprintf("updated but reload failed: %v", err), http.StatusBadGateway)
				return
			}
		}
		writeJSON(w, http.StatusOK, status)
	}))
	return mux
}

func runtimeRequestAuthorized(r *http.Request) bool {
	if r == nil {
		return false
	}
	expectedToken := strings.TrimSpace(os.Getenv("WAF_RUNTIME_API_TOKEN"))
	if expectedToken != "" {
		return subtleConstantTimeEqual(strings.TrimSpace(r.Header.Get(runtimeAuthHeader)), expectedToken)
	}
	return requestFromLoopback(r)
}

func subtleConstantTimeEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var diff byte
	for i := 0; i < len(left); i++ {
		diff |= left[i] ^ right[i]
	}
	return diff == 0
}

func requestFromLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "waf-runtime-launcher: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	runtimeRoot := strings.TrimSpace(os.Getenv("WAF_RUNTIME_ROOT"))
	if runtimeRoot == "" {
		runtimeRoot = defaultRuntimeRoot
	}
	healthAddr := strings.TrimSpace(os.Getenv("WAF_RUNTIME_HEALTH_ADDR"))
	if healthAddr == "" {
		healthAddr = defaultHealthAddr
	}
	status := &runtimeStatus{}

	systemCRSPath, err := selectFirstExisting(
		"/usr/share/modsecurity-crs",
		"/etc/modsecurity/crs",
	)
	if err != nil {
		return err
	}
	crsStateRoot := strings.TrimSpace(os.Getenv("WAF_CRS_STATE_ROOT"))
	if crsStateRoot == "" {
		crsStateRoot = "/etc/waf/modsecurity/crs-state"
	}
	manager := newCRSManager(crsStateRoot, systemCRSPath)
	if initErr := manager.Init(); initErr != nil {
		fmt.Fprintf(os.Stderr, "waf-runtime-launcher crs init warning: %v\n", initErr)
	}
	if manager.IsFirstStart() {
		if status, changed, updateErr := manager.UpdateToLatest(false); updateErr != nil {
			fmt.Fprintf(os.Stderr, "waf-runtime-launcher crs first-start update warning: %v\n", updateErr)
		} else if changed {
			fmt.Fprintf(os.Stdout, "waf-runtime-launcher: CRS first-start update applied (%s)\n", status.ActiveVersion)
		}
	}
	crsPath := manager.ActivePath()

	modsecurityModulePath, err := selectFirstExisting(
		"/usr/lib/nginx/modules/ngx_http_modsecurity_module.so",
		"/usr/lib/nginx/modules/ngx_http_modsecurity.so",
	)
	if err != nil {
		return err
	}
	geoIPModulePath, err := selectFirstExisting(
		"/usr/lib/nginx/modules/ngx_http_geoip_module.so",
		"/usr/lib/nginx/modules/ngx_http_geoip_module-debug.so",
	)
	if err != nil {
		return err
	}

	process := newRuntimeProcess(runtimeRoot, crsPath, status, manager, modsecurityModulePath, geoIPModulePath)
	securitySource := newSecurityEventSource("/var/log/nginx/access.log")
	requestSource := newRequestStreamSource(
		"/var/log/nginx/access.log",
		50000,
		requestArchiveRoot(runtimeRoot),
		30,
		withRequestOpenSearch(
			filepath.Join(runtimeRoot, "control-plane", "settings", "runtime_settings.json"),
			strings.TrimSpace(os.Getenv("CONTROL_PLANE_SECURITY_PEPPER")),
		),
		withRequestClickHouse(
			filepath.Join(runtimeRoot, "control-plane", "settings", "runtime_settings.json"),
			strings.TrimSpace(os.Getenv("CONTROL_PLANE_SECURITY_PEPPER")),
		),
	)
	requestSource.startBackgroundIngest(3 * time.Second)
	if err := startHealthServer(healthAddr, status, process, securitySource, requestSource); err != nil {
		return err
	}
	startPeriodicL4GuardRefresh()
	startPeriodicCRSUpdate(manager, process)
	if err := process.bootCurrent(); err != nil {
		if !errors.Is(err, errActivePointerMissing) {
			return err
		}
		if fallbackErr := process.bootBootstrapUI(); fallbackErr != nil {
			return fallbackErr
		}
	}

	return <-process.exitCh
}

func requestArchiveRoot(runtimeRoot string) string {
	if configured := strings.TrimSpace(os.Getenv("WAF_REQUEST_ARCHIVE_ROOT")); configured != "" {
		return configured
	}
	return filepath.Join(runtimeRoot, "requests-archive")
}

func startPeriodicCRSUpdate(manager *crsManager, process *runtimeProcess) {
	if manager == nil || process == nil {
		return
	}
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			if !manager.HourlyAutoUpdateEnabled() {
				continue
			}
			status, changed, err := manager.UpdateToLatest(false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "waf-runtime-launcher periodic crs update failed: %v\n", err)
				continue
			}
			if !changed {
				continue
			}
			process.setCRSPath(status.ActivePath)
			if err := process.reloadCurrent(); err != nil && !errors.Is(err, errActivePointerMissing) {
				fmt.Fprintf(os.Stderr, "waf-runtime-launcher periodic crs reload failed: %v\n", err)
			}
		}
	}()
}

func startPeriodicL4GuardRefresh() {
	enabled := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_ENABLED"))
	if strings.EqualFold(enabled, "false") || enabled == "0" {
		return
	}
	intervalSeconds := 5
	if raw := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_REAPPLY_INTERVAL_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			intervalSeconds = parsed
		}
	}
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	go func() {
		for range ticker.C {
			if err := applyL4Guard(); err != nil {
				fmt.Fprintf(os.Stderr, "waf-runtime-launcher periodic l4 guard reapply failed: %v\n", err)
			}
		}
	}()
}

func startHealthServer(addr string, status *runtimeStatus, process *runtimeProcess, securitySource *securityEventSource, requestSource *requestStreamSource) error {
	server := &http.Server{
		Addr:    addr,
		Handler: runtimeProm.instrument(status.handlers(process, securitySource, requestSource)),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "waf-runtime-launcher health server: %v\n", err)
		}
	}()
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSONBody(r *http.Request) (map[string]any, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func loadActivePointer(runtimeRoot string) (*activePointer, error) {
	// active/current.json is the only authoritative selector for the active bundle.
	content, err := os.ReadFile(filepath.Join(runtimeRoot, "active", "current.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errActivePointerMissing
		}
		return nil, fmt.Errorf("read active pointer: %w", err)
	}

	var pointer activePointer
	if err := json.Unmarshal(content, &pointer); err != nil {
		return nil, fmt.Errorf("decode active pointer: %w", err)
	}
	if strings.TrimSpace(pointer.RevisionID) == "" {
		return nil, errors.New("active pointer revision_id is required")
	}
	if strings.TrimSpace(pointer.CandidatePath) == "" {
		return nil, errors.New("active pointer candidate_path is required")
	}

	return &pointer, nil
}

func resolveCandidatePath(runtimeRoot, candidatePath string) (string, error) {
	candidatePath = strings.TrimSpace(candidatePath)
	if candidatePath == "" {
		return "", errors.New("candidate path is required")
	}
	if filepath.IsAbs(candidatePath) {
		return candidatePath, nil
	}
	return filepath.Join(runtimeRoot, filepath.FromSlash(candidatePath)), nil
}

func validateCandidateBundle(candidatePath string) error {
	required := []string{
		filepath.Join(candidatePath, "manifest.json"),
		filepath.Join(candidatePath, "nginx", "nginx.conf"),
		filepath.Join(candidatePath, "modsecurity", "modsecurity.conf"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("required bundle file missing: %s", path)
			}
			return fmt.Errorf("stat bundle file %s: %w", path, err)
		}
	}
	return nil
}

func prepareRuntimeLayout(candidatePath, crsPath string) error {
	if err := os.MkdirAll("/etc/waf/nginx", 0o755); err != nil {
		return fmt.Errorf("create /etc/waf/nginx: %w", err)
	}
	if err := os.MkdirAll("/etc/waf/modsecurity", 0o755); err != nil {
		return fmt.Errorf("create /etc/waf/modsecurity: %w", err)
	}
	if err := os.MkdirAll("/etc/waf/l4guard", 0o755); err != nil {
		return fmt.Errorf("create /etc/waf/l4guard: %w", err)
	}
	if err := os.MkdirAll("/etc/waf", 0o755); err != nil {
		return fmt.Errorf("create /etc/waf: %w", err)
	}

	if err := relink(filepath.Join(candidatePath, "nginx", "nginx.conf"), "/etc/waf/nginx/nginx.conf"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "conf.d"), "/etc/waf/nginx/conf.d"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "access"), "/etc/waf/nginx/access"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "ratelimits"), "/etc/waf/nginx/ratelimits"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "easy-locations"), "/etc/waf/nginx/easy-locations"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "easy"), "/etc/waf/nginx/easy"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "auth-basic"), "/etc/waf/nginx/auth-basic"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "nginx", "sites"), "/etc/waf/nginx/sites"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "modsecurity", "modsecurity.conf"), "/etc/waf/modsecurity/modsecurity.conf"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "modsecurity", "crs-setup.conf"), "/etc/waf/modsecurity/crs-setup.conf"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "modsecurity", "sites"), "/etc/waf/modsecurity/sites"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "modsecurity", "easy"), "/etc/waf/modsecurity/easy"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "l4guard"), "/etc/waf/l4guard"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "modsecurity", "crs-overrides"), "/etc/waf/modsecurity/crs-overrides"); err != nil {
		return err
	}
	if err := relink(crsPath, "/etc/waf/modsecurity/coreruleset"); err != nil {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "tls"), "/etc/waf/tls"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := relink(filepath.Join(candidatePath, "errors"), "/etc/waf/errors"); err != nil && !os.IsNotExist(err) {
		return err
	}
	// /etc/waf/current is a derived convenience view of the bundle selected by
	// active/current.json. It is not an authority input for revision selection.
	if err := relink(candidatePath, "/etc/waf/current"); err != nil {
		return err
	}

	return nil
}

func relink(target, linkPath string) error {
	if _, err := os.Stat(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return fmt.Errorf("create link parent for %s: %w", linkPath, err)
	}

	tempLink := linkPath + ".tmp"
	_ = os.RemoveAll(tempLink)

	if err := os.Symlink(target, tempLink); err != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", linkPath, target, err)
	}
	if err := os.RemoveAll(linkPath); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("remove existing path %s: %w", linkPath, err)
	}
	if err := os.Rename(tempLink, linkPath); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("replace symlink %s -> %s: %w", linkPath, target, err)
	}
	return nil
}

func applyL4Guard() error {
	enabled := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_ENABLED"))
	if strings.EqualFold(enabled, "false") || enabled == "0" {
		return nil
	}
	cmd := exec.Command("/usr/local/bin/waf-runtime-l4-guard", "bootstrap")
	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details != "" {
			return fmt.Errorf("apply l4 guard: %w: %s", err, details)
		}
		return fmt.Errorf("apply l4 guard: %w", err)
	}
	if len(output) > 0 {
		_, _ = os.Stdout.Write(output)
	}
	return nil
}

func selectFirstExisting(paths ...string) (string, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("none of the required paths exist: %s", strings.Join(paths, ", "))
}
