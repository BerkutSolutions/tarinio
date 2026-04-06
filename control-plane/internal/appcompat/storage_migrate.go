package appcompat

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var legacyControlPlaneSubdirs = []string{
	"revisions",
	"revision-snapshots",
	"events",
	"audits",
	"jobs",
	"roles",
	"users",
	"sessions",
	"passkeys",
	"sites",
	"upstreams",
	"certificates",
	"certificate-materials",
	"tlsconfigs",
	"wafpolicies",
	"accesspolicies",
	"ratelimitpolicies",
	"easysiteprofiles",
	"antiddos",
	"tls-auto-renew",
}

var moduleLegacyDirs = map[string][]string{
	"dashboard":      {"events", "jobs", "revisions"},
	"sites":          {"sites", "upstreams", "easysiteprofiles", "accesspolicies", "ratelimitpolicies", "wafpolicies"},
	"antiddos":       {"antiddos"},
	"owaspcrs":       {"wafpolicies"},
	"tls":            {"certificates", "certificate-materials", "tlsconfigs", "tls-auto-renew"},
	"requests":       {"events"},
	"events":         {"events"},
	"bans":           {"accesspolicies", "sites"},
	"administration": {"users", "roles", "sessions", "passkeys", "audits"},
	"activity":       {"audits"},
	"settings":       {"users", "roles", "tls-auto-renew"},
	"profile":        {"users", "sessions", "passkeys"},
}

type LegacyScan struct {
	PendingTransfers []string `json:"pending_transfers"`
	LegacyCandidates []string `json:"legacy_candidates"`
}

func LegacyCandidates(runtimeRoot string) []string {
	root := strings.TrimSpace(runtimeRoot)
	if root == "" {
		return nil
	}
	return []string{
		filepath.Join(root, "data", "control-plane"),
		filepath.Join(root, "storage", "control-plane"),
		filepath.Join(root, "controlplane"),
	}
}

func ScanLegacyLayout(runtimeRoot string, revisionStoreDir string) LegacyScan {
	out := LegacyScan{
		PendingTransfers: make([]string, 0),
		LegacyCandidates: make([]string, 0),
	}
	dstRoot := strings.TrimSpace(revisionStoreDir)
	if dstRoot == "" {
		return out
	}
	for _, base := range LegacyCandidates(runtimeRoot) {
		info, err := os.Stat(base)
		if err != nil || !info.IsDir() {
			continue
		}
		out.LegacyCandidates = append(out.LegacyCandidates, base)
		for _, item := range legacyControlPlaneSubdirs {
			src := filepath.Join(base, item)
			srcInfo, srcErr := os.Stat(src)
			if srcErr != nil || !srcInfo.IsDir() {
				continue
			}
			dst := filepath.Join(dstRoot, item)
			dstInfo, dstErr := os.Stat(dst)
			if dstErr == nil && dstInfo.IsDir() {
				continue
			}
			out.PendingTransfers = append(out.PendingTransfers, item)
		}
	}
	return out
}

func PendingLegacyByModule(runtimeRoot string, revisionStoreDir string) map[string][]string {
	scan := ScanLegacyLayout(runtimeRoot, revisionStoreDir)
	pendingSet := make(map[string]struct{}, len(scan.PendingTransfers))
	for _, item := range scan.PendingTransfers {
		pendingSet[item] = struct{}{}
	}
	result := make(map[string][]string, len(moduleLegacyDirs))
	for moduleID, dirs := range moduleLegacyDirs {
		missing := make([]string, 0, len(dirs))
		for _, dir := range dirs {
			if _, ok := pendingSet[dir]; ok {
				missing = append(missing, dir)
			}
		}
		sort.Strings(missing)
		result[moduleID] = missing
	}
	return result
}

func EnsureLegacyDataTransferred(runtimeRoot string, revisionStoreDir string) error {
	dstRoot := strings.TrimSpace(revisionStoreDir)
	if dstRoot == "" {
		return fmt.Errorf("revision store dir is required")
	}
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return fmt.Errorf("create revision store root: %w", err)
	}
	for _, base := range LegacyCandidates(runtimeRoot) {
		baseInfo, err := os.Stat(base)
		if err != nil || !baseInfo.IsDir() {
			continue
		}
		for _, item := range legacyControlPlaneSubdirs {
			src := filepath.Join(base, item)
			srcInfo, srcErr := os.Stat(src)
			if srcErr != nil || !srcInfo.IsDir() {
				continue
			}
			dst := filepath.Join(dstRoot, item)
			if dstInfo, dstErr := os.Stat(dst); dstErr == nil && dstInfo.IsDir() {
				continue
			}
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("legacy transfer %s -> %s: %w", src, dst, err)
			}
		}
	}
	return nil
}

func EnsureLegacyModuleTransferred(runtimeRoot string, revisionStoreDir string, moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return fmt.Errorf("module id is required")
	}
	dirs, ok := moduleLegacyDirs[moduleID]
	if !ok {
		return fmt.Errorf("unknown module id: %s", moduleID)
	}
	return ensureLegacyDirsTransferred(runtimeRoot, revisionStoreDir, dirs)
}

func ensureLegacyDirsTransferred(runtimeRoot string, revisionStoreDir string, onlyDirs []string) error {
	dstRoot := strings.TrimSpace(revisionStoreDir)
	if dstRoot == "" {
		return fmt.Errorf("revision store dir is required")
	}
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return fmt.Errorf("create revision store root: %w", err)
	}
	allowed := make(map[string]struct{}, len(onlyDirs))
	for _, item := range onlyDirs {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}
	for _, base := range LegacyCandidates(runtimeRoot) {
		baseInfo, err := os.Stat(base)
		if err != nil || !baseInfo.IsDir() {
			continue
		}
		for _, item := range legacyControlPlaneSubdirs {
			if _, ok := allowed[item]; !ok {
				continue
			}
			src := filepath.Join(base, item)
			srcInfo, srcErr := os.Stat(src)
			if srcErr != nil || !srcInfo.IsDir() {
				continue
			}
			dst := filepath.Join(dstRoot, item)
			if dstInfo, dstErr := os.Stat(dst); dstErr == nil && dstInfo.IsDir() {
				continue
			}
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("legacy transfer %s -> %s: %w", src, dst, err)
			}
		}
	}
	return nil
}

func copyDir(src string, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
