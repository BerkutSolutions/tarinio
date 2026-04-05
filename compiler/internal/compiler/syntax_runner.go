package compiler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func hasGlobPattern(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

// CommandExecutor isolates external command execution for syntax validation.
type CommandExecutor interface {
	Run(name string, args []string, workdir string) error
}

// RuntimeSyntaxRunner validates one compiled revision bundle without requiring
// control-plane state or domain-model access.
type RuntimeSyntaxRunner struct {
	NginxBinary string
	Executor    CommandExecutor
}

// Validate materializes the bundle, checks included files, and runs nginx -t.
func (r RuntimeSyntaxRunner) Validate(bundle *RevisionBundle) error {
	if err := ValidateRevisionBundle(bundle); err != nil {
		return err
	}
	if r.Executor == nil {
		return errors.New("command executor is required")
	}

	nginxBinary := strings.TrimSpace(r.NginxBinary)
	if nginxBinary == "" {
		nginxBinary = "nginx"
	}

	tempRoot, err := os.MkdirTemp("", "waf-syntax-*")
	if err != nil {
		return fmt.Errorf("create syntax validation temp root: %w", err)
	}
	defer os.RemoveAll(tempRoot)

	bundleRoot := filepath.Join(tempRoot, "bundle")
	if err := materializeBundle(bundleRoot, bundle.Files); err != nil {
		return err
	}
	if err := validateIncludedFiles(bundleRoot); err != nil {
		return err
	}

	nginxRoot := filepath.Join(bundleRoot, "nginx")
	args := []string{"-t", "-p", nginxRoot, "-c", "nginx.conf"}
	moduleDirectives := make([]string, 0, 2)
	if modulePath, ok := firstExistingPath(
		"/usr/lib/nginx/modules/ngx_http_modsecurity_module.so",
		"/usr/lib/nginx/modules/ngx_http_modsecurity.so",
	); ok {
		moduleDirectives = append(moduleDirectives, fmt.Sprintf("load_module %s;", modulePath))
	}
	if modulePath, ok := firstExistingPath(
		"/usr/lib/nginx/modules/ngx_http_geoip_module.so",
		"/usr/lib/nginx/modules/ngx_http_geoip_module-debug.so",
	); ok {
		moduleDirectives = append(moduleDirectives, fmt.Sprintf("load_module %s;", modulePath))
	}
	if len(moduleDirectives) > 0 {
		args = append(args, "-g", strings.Join(moduleDirectives, " "))
	}

	run := func() error {
		if err := r.Executor.Run(nginxBinary, args, nginxRoot); err != nil {
			return fmt.Errorf("nginx syntax validation failed: %w", err)
		}
		return nil
	}

	if runtime.GOOS == "windows" {
		return run()
	}

	if err := withTemporaryEtcWAF(bundleRoot, run); err != nil {
		return err
	}

	return nil
}

func withTemporaryEtcWAF(bundleRoot string, run func() error) error {
	const liveEtcWAF = "/etc/waf"

	shadowRoot := bundleRoot
	backupPath := ""
	if _, err := os.Lstat(liveEtcWAF); err == nil {
		backupPath = liveEtcWAF + ".bak"
		_ = os.RemoveAll(backupPath)
		if err := os.Rename(liveEtcWAF, backupPath); err != nil {
			return fmt.Errorf("prepare temporary /etc/waf backup: %w", err)
		}
	}

	if err := os.Symlink(shadowRoot, liveEtcWAF); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, liveEtcWAF)
		}
		return fmt.Errorf("prepare temporary /etc/waf shadow: %w", err)
	}

	defer func() {
		_ = os.Remove(liveEtcWAF)
		if backupPath != "" {
			_ = os.Rename(backupPath, liveEtcWAF)
		}
	}()

	return run()
}

func materializeBundle(root string, files []BundleFile) error {
	for _, file := range files {
		targetPath := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create bundle directory for %s: %w", file.Path, err)
		}
		if err := os.WriteFile(targetPath, file.Content, 0o644); err != nil {
			return fmt.Errorf("write bundle file %s: %w", file.Path, err)
		}
	}
	return nil
}

func validateIncludedFiles(bundleRoot string) error {
	nginxRoot := filepath.Join(bundleRoot, "nginx")
	required := []string{
		filepath.Join(nginxRoot, "nginx.conf"),
	}

	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("required nginx file missing: %s", filepath.ToSlash(path))
		}
	}

	seen := make(map[string]struct{})
	queue := []string{filepath.Join(nginxRoot, "nginx.conf")}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		content, err := os.ReadFile(current)
		if err != nil {
			return fmt.Errorf("read nginx config %s: %w", filepath.ToSlash(current), err)
		}

		refs, err := extractReferencedFiles(bundleRoot, current, string(content))
		if err != nil {
			return err
		}
		for _, ref := range refs {
			matches, err := filepath.Glob(ref)
			if err != nil {
				return fmt.Errorf("invalid include pattern %s: %w", filepath.ToSlash(ref), err)
			}
			if len(matches) == 0 {
				if hasGlobPattern(ref) {
					continue
				}
				return fmt.Errorf("included file pattern has no matches: %s", filepath.ToSlash(ref))
			}
			for _, match := range matches {
				if _, err := os.Stat(match); err != nil {
					return fmt.Errorf("included file missing: %s", filepath.ToSlash(match))
				}
				if filepath.Ext(match) == ".conf" {
					queue = append(queue, match)
				}
			}
		}
	}

	return nil
}

func extractReferencedFiles(bundleRoot, currentPath, content string) ([]string, error) {
	var refs []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "include "):
			ref, ok := extractDirectiveValue(trimmed, "include")
			if ok {
				refs = append(refs, mapBundleReference(bundleRoot, currentPath, ref))
			}
		case strings.HasPrefix(trimmed, "modsecurity_rules_file "):
			ref, ok := extractDirectiveValue(trimmed, "modsecurity_rules_file")
			if ok {
				refs = append(refs, mapBundleReference(bundleRoot, currentPath, ref))
			}
		}
	}
	return refs, nil
}

func extractDirectiveValue(line, directive string) (string, bool) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, directive))
	rest = strings.TrimSuffix(rest, ";")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	return rest, true
}

func mapBundleReference(bundleRoot, currentPath, ref string) string {
	slashRef := filepath.ToSlash(ref)
	if strings.HasPrefix(slashRef, "/etc/waf/") {
		trimmed := strings.TrimPrefix(slashRef, "/etc/waf/")
		return filepath.Join(bundleRoot, filepath.FromSlash(trimmed))
	}

	osRef := filepath.FromSlash(ref)
	if filepath.IsAbs(osRef) {
		return osRef
	}
	return filepath.Join(filepath.Dir(currentPath), osRef)
}

func firstExistingPath(paths ...string) (string, bool) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}
