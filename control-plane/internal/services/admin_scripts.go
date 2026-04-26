package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type AdminScriptField struct {
	Name           string `json:"name"`
	Label          string `json:"label"`
	LabelKey       string `json:"label_key,omitempty"`
	Type           string `json:"type"`
	Placeholder    string `json:"placeholder,omitempty"`
	PlaceholderKey string `json:"placeholder_key,omitempty"`
	HelpText       string `json:"help_text,omitempty"`
	HelpTextKey    string `json:"help_text_key,omitempty"`
	DefaultValue   string `json:"default_value,omitempty"`
	Required       bool   `json:"required,omitempty"`
}

type AdminScriptDefinition struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	TitleKey       string             `json:"title_key,omitempty"`
	Description    string             `json:"description"`
	DescriptionKey string             `json:"description_key,omitempty"`
	FileName       string             `json:"file_name"`
	Fields         []AdminScriptField `json:"fields"`
}

type AdminScriptCatalog struct {
	Scripts []AdminScriptDefinition `json:"scripts"`
}

type AdminScriptRunResult struct {
	RunID          string `json:"run_id"`
	ScriptID       string `json:"script_id"`
	Title          string `json:"title"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at"`
	Status         string `json:"status"`
	ExitCode       int    `json:"exit_code"`
	ArchiveName    string `json:"archive_name,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	ConsoleLogName string `json:"console_log_name,omitempty"`
	Error          string `json:"error,omitempty"`
}

type adminScriptRunRecord struct {
	RunID          string `json:"run_id"`
	ScriptID       string `json:"script_id"`
	Title          string `json:"title"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at"`
	Status         string `json:"status"`
	ExitCode       int    `json:"exit_code"`
	ArchiveName    string `json:"archive_name,omitempty"`
	ArchivePath    string `json:"archive_path,omitempty"`
	ConsoleLogName string `json:"console_log_name,omitempty"`
	ConsoleLogPath string `json:"console_log_path,omitempty"`
	Error          string `json:"error,omitempty"`
}

type AdminScriptService struct {
	scriptsRoot string
	runsRoot    string
	catalog     map[string]AdminScriptDefinition
}

var (
	adminScriptSincePattern       = regexp.MustCompile(`^[0-9]+[smhdw]$`)
	adminScriptContainerPattern   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,127}$`)
	adminScriptPathPattern        = regexp.MustCompile(`^[a-zA-Z0-9/_\.-]{1,256}$`)
	adminScriptBinaryPathPattern  = regexp.MustCompile(`^[a-zA-Z0-9/_\.-]{1,128}$`)
	adminScriptStatusCodePattern  = regexp.MustCompile(`^[0-9\s,;|]{1,256}$`)
	adminScriptUnsafeShellPattern = regexp.MustCompile(`[\"'\\$` + "`" + `|&<>]`)
)

func NewAdminScriptService(revisionStoreDir string, scriptsRoot string) *AdminScriptService {
	definitions := []AdminScriptDefinition{
		{
			ID:             "collect-waf-errors",
			TitleKey:       "administration.scripts.collectErrors.title",
			Title:          "WAF Error Collector",
			DescriptionKey: "administration.scripts.collectErrors.description",
			Description:    "Collect runtime/control-plane error, warning, notice and timeout diagnostics into a downloadable archive.",
			FileName:       "collect-waf-errors.sh",
			Fields: []AdminScriptField{
				{Name: "SINCE", Label: "Since", LabelKey: "administration.scripts.field.since", Type: "text", DefaultValue: "24h", Placeholder: "24h", HelpText: "Docker log range, for example 30m, 6h, 24h.", HelpTextKey: "administration.scripts.field.sinceHelp"},
				{Name: "FILTER_HOST", Label: "Host Filter", LabelKey: "administration.scripts.field.hostFilter", Type: "text", Placeholder: "example.com", HelpText: "Optional host/domain to highlight in collected logs.", HelpTextKey: "administration.scripts.field.hostFilterHelp"},
				{Name: "FILTER_IP", Label: "Client IPs", LabelKey: "administration.scripts.field.clientIps", Type: "textarea", Placeholder: "203.0.113.10\n198.51.100.20", HelpText: "Optional IPs. One per line or space-separated.", HelpTextKey: "administration.scripts.field.clientIpsHelp"},
				{Name: "FILTER_URI", Label: "Request URI / Path", Type: "textarea", Placeholder: "/api/2/envelope\n/login", HelpText: "Optional URI/path filters. One per line or space-separated."},
				{Name: "RUNTIME_CONTAINER", Label: "Runtime Container", LabelKey: "administration.scripts.field.runtimeContainer", Type: "text", DefaultValue: "tarinio-runtime", HelpText: "Docker container name for runtime logs.", HelpTextKey: "administration.scripts.field.runtimeContainerHelp"},
				{Name: "CONTROL_PLANE_CONTAINER", Label: "Control-Plane Container", LabelKey: "administration.scripts.field.controlPlaneContainer", Type: "text", DefaultValue: "tarinio-control-plane", HelpText: "Docker container name for control-plane logs.", HelpTextKey: "administration.scripts.field.controlPlaneContainerHelp"},
				{Name: "DDOS_MODEL_CONTAINER", Label: "DDoS Model Container", LabelKey: "administration.scripts.field.ddosModelContainer", Type: "text", DefaultValue: "tarinio-sentinel"},
				{Name: "UI_CONTAINER", Label: "UI Container", LabelKey: "administration.scripts.field.uiContainer", Type: "text", DefaultValue: "tarinio-ui"},
			},
		},
		{
			ID:             "collect-waf-events",
			TitleKey:       "administration.scripts.collectEvents.title",
			Title:          "WAF Events Collector",
			DescriptionKey: "administration.scripts.collectEvents.description",
			Description:    "Collect API events, bans, audit trail and runtime access/log slices with optional filters.",
			FileName:       "collect-waf-events.sh",
			Fields: []AdminScriptField{
				{Name: "SINCE", Label: "Since", LabelKey: "administration.scripts.field.since", Type: "text", DefaultValue: "24h", Placeholder: "24h", HelpText: "Docker log range, for example 30m, 6h, 24h.", HelpTextKey: "administration.scripts.field.sinceHelp"},
				{Name: "FILTER_IP", Label: "Client IPs", LabelKey: "administration.scripts.field.clientIps", Type: "textarea", Placeholder: "1.1.1.1\n2.2.2.2", HelpText: "Optional IP filters. One per line or space-separated.", HelpTextKey: "administration.scripts.field.clientIpsHelp"},
				{Name: "FILTER_SITE", Label: "Sites / Hosts", LabelKey: "administration.scripts.field.sitesHosts", Type: "textarea", Placeholder: "site-a.example.com\napi.example.com", HelpText: "Optional site or host filters.", HelpTextKey: "administration.scripts.field.sitesHostsHelp"},
				{Name: "FILTER_URI", Label: "Request URI / Path", Type: "textarea", Placeholder: "/api/2/envelope\n/api/store", HelpText: "Optional URI/path filters. One per line or space-separated."},
				{Name: "FILTER_STATUS", Label: "HTTP Status Codes", LabelKey: "administration.scripts.field.httpStatus", Type: "text", Placeholder: "403 429 503", HelpText: "Optional status filters.", HelpTextKey: "administration.scripts.field.httpStatusHelp"},
				{Name: "RUNTIME_CONTAINER", Label: "Runtime Container", LabelKey: "administration.scripts.field.runtimeContainer", Type: "text", DefaultValue: "tarinio-runtime", HelpText: "Docker container name for runtime logs.", HelpTextKey: "administration.scripts.field.runtimeContainerHelp"},
				{Name: "CONTROL_PLANE_CONTAINER", Label: "Control-Plane Container", LabelKey: "administration.scripts.field.controlPlaneContainer", Type: "text", DefaultValue: "tarinio-control-plane", HelpText: "Docker container name for control-plane logs.", HelpTextKey: "administration.scripts.field.controlPlaneContainerHelp"},
				{Name: "WAF_CLI_BIN", Label: "CLI Binary", LabelKey: "administration.scripts.field.cliBinary", Type: "text", DefaultValue: "waf-cli", HelpText: "Leave default for built-in control-plane CLI.", HelpTextKey: "administration.scripts.field.cliBinaryHelp"},
				{Name: "DEPLOY_DIR", Label: "Deploy Directory", LabelKey: "administration.scripts.field.deployDir", Type: "text", DefaultValue: "/opt/tarinio/deploy/compose/default", HelpText: "Used only when the script falls back to docker compose.", HelpTextKey: "administration.scripts.field.deployDirHelp"},
			},
		},
	}

	catalog := make(map[string]AdminScriptDefinition, len(definitions))
	for _, item := range definitions {
		catalog[item.ID] = item
	}
	return &AdminScriptService{
		scriptsRoot: strings.TrimSpace(scriptsRoot),
		runsRoot:    filepath.Join(revisionStoreDir, "script-runs"),
		catalog:     catalog,
	}
}

func (s *AdminScriptService) Catalog() AdminScriptCatalog {
	items := make([]AdminScriptDefinition, 0, len(s.catalog))
	for _, item := range s.catalog {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	return AdminScriptCatalog{Scripts: items}
}

func (s *AdminScriptService) Run(ctx context.Context, scriptID string, input map[string]string) (AdminScriptRunResult, error) {
	if s == nil {
		return AdminScriptRunResult{}, errors.New("admin script service is nil")
	}
	definition, ok := s.catalog[strings.TrimSpace(scriptID)]
	if !ok {
		return AdminScriptRunResult{}, errors.New("script not found")
	}
	scriptPath, err := s.resolveScriptPath(definition.FileName)
	if err != nil {
		return AdminScriptRunResult{}, err
	}
	if err := os.MkdirAll(s.runsRoot, 0o755); err != nil {
		return AdminScriptRunResult{}, err
	}

	runID := fmt.Sprintf("%s-%d", definition.ID, time.Now().UTC().UnixNano())
	runDir := filepath.Join(s.runsRoot, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return AdminScriptRunResult{}, err
	}

	startedAt := time.Now().UTC()
	env, err := s.buildEnvironment(definition, input, runDir)
	if err != nil {
		return AdminScriptRunResult{}, err
	}
	consoleLogPath := filepath.Join(runDir, "console.log")
	cmdCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		cmdCtx, cancel = context.WithTimeout(ctx, 4*time.Minute)
		defer cancel()
	}
	cmd := exec.CommandContext(cmdCtx, "bash", scriptPath)
	cmd.Env = env
	cmd.Dir = filepath.Dir(scriptPath)
	output, runErr := cmd.CombinedOutput()
	if writeErr := os.WriteFile(consoleLogPath, output, 0o644); writeErr != nil {
		return AdminScriptRunResult{}, writeErr
	}

	finishedAt := time.Now().UTC()
	record := adminScriptRunRecord{
		RunID:          runID,
		ScriptID:       definition.ID,
		Title:          definition.Title,
		StartedAt:      startedAt.Format(time.RFC3339Nano),
		FinishedAt:     finishedAt.Format(time.RFC3339Nano),
		Status:         "succeeded",
		ExitCode:       0,
		ConsoleLogName: filepath.Base(consoleLogPath),
		ConsoleLogPath: consoleLogPath,
	}
	if runErr != nil {
		record.Status = "failed"
		record.ExitCode = 1
		record.Error = strings.TrimSpace(runErr.Error())
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			record.ExitCode = exitErr.ExitCode()
		}
	}

	if archivePath, archiveName := findArchive(runDir); archivePath != "" {
		record.ArchivePath = archivePath
		record.ArchiveName = archiveName
	}
	if err := s.writeRunRecord(runDir, record); err != nil {
		return AdminScriptRunResult{}, err
	}

	return AdminScriptRunResult{
		RunID:          record.RunID,
		ScriptID:       record.ScriptID,
		Title:          record.Title,
		StartedAt:      record.StartedAt,
		FinishedAt:     record.FinishedAt,
		Status:         record.Status,
		ExitCode:       record.ExitCode,
		ArchiveName:    record.ArchiveName,
		DownloadURL:    buildRunDownloadURL(record.RunID),
		ConsoleLogName: record.ConsoleLogName,
		Error:          record.Error,
	}, nil
}

func (s *AdminScriptService) Download(runID string) (string, []byte, error) {
	if s == nil {
		return "", nil, errors.New("admin script service is nil")
	}
	record, err := s.readRunRecord(runID)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(record.ArchivePath) == "" {
		return "", nil, errors.New("archive is not available for this run")
	}
	content, err := os.ReadFile(record.ArchivePath)
	if err != nil {
		return "", nil, err
	}
	return record.ArchiveName, content, nil
}

func (s *AdminScriptService) buildEnvironment(definition AdminScriptDefinition, input map[string]string, runDir string) ([]string, error) {
	env := append([]string(nil), os.Environ()...)
	env = append(env, "NON_INTERACTIVE=1", "OUT_BASE_DIR="+runDir)
	for _, field := range definition.Fields {
		value := strings.TrimSpace(input[field.Name])
		if value == "" {
			value = strings.TrimSpace(field.DefaultValue)
		}
		if field.Required && value == "" {
			return nil, fmt.Errorf("%s is required", field.Label)
		}
		if value == "" {
			continue
		}
		safeValue, err := validateAdminScriptFieldValue(field.Name, value)
		if err != nil {
			return nil, err
		}
		env = append(env, field.Name+"="+safeValue)
	}
	return env, nil
}

func validateAdminScriptFieldValue(fieldName, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsAny(trimmed, "\r\n") && fieldName != "FILTER_IP" && fieldName != "FILTER_SITE" && fieldName != "FILTER_URI" {
		return "", fmt.Errorf("%s contains unsupported line breaks", fieldName)
	}
	if adminScriptUnsafeShellPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%s contains unsupported characters", fieldName)
	}

	switch fieldName {
	case "SINCE":
		if !adminScriptSincePattern.MatchString(trimmed) {
			return "", errors.New("SINCE must match <number><s|m|h|d|w>, for example 24h")
		}
	case "RUNTIME_CONTAINER", "CONTROL_PLANE_CONTAINER", "DDOS_MODEL_CONTAINER", "UI_CONTAINER", "CLICKHOUSE_CONTAINER", "OPENSEARCH_CONTAINER", "VAULT_CONTAINER":
		if !adminScriptContainerPattern.MatchString(trimmed) {
			return "", fmt.Errorf("%s must be a safe container name", fieldName)
		}
	case "DEPLOY_DIR":
		if !adminScriptPathPattern.MatchString(trimmed) || strings.Contains(trimmed, "..") {
			return "", errors.New("DEPLOY_DIR must be a safe absolute or relative path without traversal")
		}
	case "WAF_CLI_BIN":
		if !adminScriptBinaryPathPattern.MatchString(trimmed) {
			return "", errors.New("WAF_CLI_BIN must be a safe binary path")
		}
	case "FILTER_STATUS":
		if !adminScriptStatusCodePattern.MatchString(trimmed) {
			return "", errors.New("FILTER_STATUS supports only status codes and separators")
		}
	case "FILTER_IP", "FILTER_SITE", "FILTER_URI", "FILTER_HOST":
		if len(trimmed) > 2000 {
			return "", fmt.Errorf("%s is too long", fieldName)
		}
	}

	return trimmed, nil
}

func (s *AdminScriptService) resolveScriptPath(fileName string) (string, error) {
	root := strings.TrimSpace(s.scriptsRoot)
	if root == "" {
		return "", errors.New("scripts root is not configured")
	}
	path := filepath.Join(root, fileName)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("script %s is unavailable: %w", fileName, err)
	}
	return path, nil
}

func findArchive(runDir string) (string, string) {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return "", ""
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", ""
	}
	name := names[len(names)-1]
	return filepath.Join(runDir, name), name
}

func buildRunDownloadURL(runID string) string {
	return "/api/administration/scripts/runs/" + runID + "/download"
}

func (s *AdminScriptService) writeRunRecord(runDir string, record adminScriptRunRecord) error {
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(filepath.Join(runDir, "run.json"), content, 0o644)
}

func (s *AdminScriptService) readRunRecord(runID string) (adminScriptRunRecord, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return adminScriptRunRecord{}, errors.New("run id is required")
	}
	if strings.Contains(runID, "..") || strings.ContainsAny(runID, `/\`) {
		return adminScriptRunRecord{}, errors.New("invalid run id")
	}
	content, err := os.ReadFile(filepath.Join(s.runsRoot, runID, "run.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return adminScriptRunRecord{}, errors.New("run not found")
		}
		return adminScriptRunRecord{}, err
	}
	var record adminScriptRunRecord
	if err := json.Unmarshal(content, &record); err != nil {
		return adminScriptRunRecord{}, err
	}
	return record, nil
}
