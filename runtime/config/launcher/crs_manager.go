package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	crsGitHubLatestReleaseAPI = "https://api.github.com/repos/coreruleset/coreruleset/releases/latest"
	crsUserAgent              = "tarinio-runtime-crs-updater"
	fallbackCRSVersion        = "system"
)

type crsStatus struct {
	ActiveVersion           string `json:"active_version"`
	ActivePath              string `json:"active_path"`
	SystemFallbackPath      string `json:"system_fallback_path"`
	LatestVersion           string `json:"latest_version"`
	LatestReleaseURL        string `json:"latest_release_url"`
	LastCheckedAt           string `json:"last_checked_at"`
	HasUpdate               bool   `json:"has_update"`
	HourlyAutoUpdateEnabled bool   `json:"hourly_auto_update_enabled"`
	FirstStartPending       bool   `json:"first_start_pending"`
	LastError               string `json:"last_error,omitempty"`
	LastErrorCode           string `json:"last_error_code,omitempty"`
}

type crsStateFile struct {
	ActiveVersion           string `json:"active_version"`
	ActivePath              string `json:"active_path"`
	HourlyAutoUpdateEnabled bool   `json:"hourly_auto_update_enabled"`
	FirstStartComplete      bool   `json:"first_start_complete"`
}

type crsManager struct {
	mu sync.Mutex

	rootDir        string
	statePath      string
	systemPath     string
	httpClient     *http.Client
	trustedDigests map[string]string
	activePath     string
	activeVersion  string
	latestVersion  string
	latestRelease  string
	lastCheckedAt  string
	hourlyAuto     bool
	firstStartDone bool
	lastError      string
	lastErrorCode  string
}

func newCRSManager(rootDir, systemPath string) *crsManager {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = "/var/lib/waf/crs"
	}
	trustedDigests := defaultCRSTrustedDigests()
	if digests, err := loadCRSTrustedDigests(); err == nil {
		trustedDigests = digests
	}
	return &crsManager{
		rootDir:        rootDir,
		statePath:      filepath.Join(rootDir, "state.json"),
		systemPath:     strings.TrimSpace(systemPath),
		httpClient:     &http.Client{Timeout: 45 * time.Second},
		trustedDigests: trustedDigests,
	}
}

func (m *crsManager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Join(m.rootDir, "releases"), 0o755); err != nil {
		return fmt.Errorf("create crs root: %w", err)
	}

	var state crsStateFile
	if raw, err := os.ReadFile(m.statePath); err == nil {
		_ = json.Unmarshal(raw, &state)
	}
	m.hourlyAuto = state.HourlyAutoUpdateEnabled
	m.firstStartDone = state.FirstStartComplete
	m.activeVersion = strings.TrimSpace(state.ActiveVersion)
	m.activePath = strings.TrimSpace(state.ActivePath)

	if !isValidCRSPath(m.activePath) {
		m.activePath = m.systemPath
		m.activeVersion = fallbackCRSVersion
	}
	if strings.TrimSpace(m.activePath) == "" {
		m.activePath = m.systemPath
	}
	if strings.TrimSpace(m.activeVersion) == "" {
		m.activeVersion = fallbackCRSVersion
	}
	return m.saveStateLocked()
}

func (m *crsManager) Status() crsStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return crsStatus{
		ActiveVersion:           m.activeVersion,
		ActivePath:              m.activePath,
		SystemFallbackPath:      m.systemPath,
		LatestVersion:           m.latestVersion,
		LatestReleaseURL:        m.latestRelease,
		LastCheckedAt:           m.lastCheckedAt,
		HasUpdate:               isVersionGreater(m.latestVersion, m.activeVersion),
		HourlyAutoUpdateEnabled: m.hourlyAuto,
		FirstStartPending:       !m.firstStartDone,
		LastError:               m.lastError,
		LastErrorCode:           m.lastErrorCode,
	}
}

func (m *crsManager) ActivePath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activePath
}

func (m *crsManager) IsFirstStart() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.firstStartDone
}

func (m *crsManager) HourlyAutoUpdateEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hourlyAuto
}

func (m *crsManager) SetHourlyAutoUpdate(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hourlyAuto = enabled
	m.clearLastErrorLocked()
	_ = m.saveStateLocked()
}

func (m *crsManager) CheckForUpdates(_ bool) (crsStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	release, err := m.fetchLatestReleaseLocked()
	if err != nil {
		m.recordErrorLocked(err)
		_ = m.saveStateLocked()
		return m.statusLocked(), err
	}
	m.latestVersion = release.Version
	m.latestRelease = release.ReleaseURL
	m.lastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	m.clearLastErrorLocked()
	_ = m.saveStateLocked()
	return m.statusLocked(), nil
}

func (m *crsManager) UpdateToLatest(_ bool) (crsStatus, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	release, err := m.fetchLatestReleaseLocked()
	if err != nil {
		m.recordErrorLocked(err)
		_ = m.saveStateLocked()
		return m.statusLocked(), false, err
	}
	m.latestVersion = release.Version
	m.latestRelease = release.ReleaseURL
	m.lastCheckedAt = time.Now().UTC().Format(time.RFC3339)

	if !isVersionGreater(release.Version, m.activeVersion) {
		m.firstStartDone = true
		m.clearLastErrorLocked()
		_ = m.saveStateLocked()
		return m.statusLocked(), false, nil
	}

	versionPath := filepath.Join(m.rootDir, "releases", sanitizeReleaseVersion(release.Version))
	if !isValidCRSPath(versionPath) {
		if err := m.downloadAndExtractLocked(release.DownloadURL, release.Digest, versionPath); err != nil {
			m.recordErrorLocked(err)
			_ = m.saveStateLocked()
			return m.statusLocked(), false, err
		}
	}

	m.activeVersion = release.Version
	m.activePath = versionPath
	m.firstStartDone = true
	m.clearLastErrorLocked()
	_ = m.saveStateLocked()
	return m.statusLocked(), true, nil
}

func (m *crsManager) statusLocked() crsStatus {
	return crsStatus{
		ActiveVersion:           m.activeVersion,
		ActivePath:              m.activePath,
		SystemFallbackPath:      m.systemPath,
		LatestVersion:           m.latestVersion,
		LatestReleaseURL:        m.latestRelease,
		LastCheckedAt:           m.lastCheckedAt,
		HasUpdate:               isVersionGreater(m.latestVersion, m.activeVersion),
		HourlyAutoUpdateEnabled: m.hourlyAuto,
		FirstStartPending:       !m.firstStartDone,
		LastError:               m.lastError,
		LastErrorCode:           m.lastErrorCode,
	}
}

func (m *crsManager) saveStateLocked() error {
	state := crsStateFile{
		ActiveVersion:           m.activeVersion,
		ActivePath:              m.activePath,
		HourlyAutoUpdateEnabled: m.hourlyAuto,
		FirstStartComplete:      m.firstStartDone,
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(m.statePath, raw, 0o644)
}

func (m *crsManager) downloadAndExtractLocked(url, expectedDigest, targetDir string) error {
	if strings.TrimSpace(url) == "" {
		return newCRSError(crsErrorArchiveDownload, errors.New("CRS download URL is required"))
	}
	if len(expectedDigest) != 64 {
		return newCRSError(crsErrorDigestInvalid, errors.New("CRS archive SHA-256 is required"))
	}
	pendingDir, err := os.MkdirTemp(filepath.Dir(targetDir), ".crs-pending-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(pendingDir)
	if err := os.Chmod(pendingDir, 0o700); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", crsUserAgent)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return newCRSError(crsErrorArchiveDownload, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newCRSError(crsErrorArchiveDownload, fmt.Errorf("CRS archive download returned status %d", resp.StatusCode))
	}

	hash := sha256.New()
	gz, err := gzip.NewReader(io.TeeReader(resp.Body, hash))
	if err != nil {
		return newCRSError(crsErrorArchiveInvalid, err)
	}
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return newCRSError(crsErrorArchiveInvalid, err)
		}
		relPath := strings.TrimSpace(header.Name)
		if relPath == "" {
			continue
		}
		parts := strings.Split(relPath, "/")
		if len(parts) > 1 {
			relPath = strings.Join(parts[1:], "/")
		}
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			continue
		}
		dest := filepath.Join(pendingDir, filepath.FromSlash(relPath))
		if !isPathWithinRoot(dest, pendingDir) {
			return newCRSError(crsErrorArchiveInvalid, fmt.Errorf("invalid tar path: %s", header.Name))
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return newCRSError(crsErrorArchiveInvalid, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return newCRSError(crsErrorArchiveInvalid, err)
			}
			file, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return newCRSError(crsErrorArchiveInvalid, err)
			}
			if _, err := io.Copy(file, reader); err != nil {
				_ = file.Close()
				return newCRSError(crsErrorArchiveInvalid, err)
			}
			if err := file.Close(); err != nil {
				return newCRSError(crsErrorArchiveInvalid, err)
			}
		}
	}
	if err := gz.Close(); err != nil {
		return newCRSError(crsErrorArchiveInvalid, err)
	}
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return newCRSError(crsErrorArchiveDownload, err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(expectedDigest) {
		return newCRSError(crsErrorArchiveDigest, errors.New("downloaded CRS archive digest does not match expected SHA-256"))
	}
	if !isValidCRSPath(pendingDir) {
		return newCRSError(crsErrorArchiveInvalid, errors.New("downloaded CRS archive is missing rules/*.conf"))
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return newCRSError(crsErrorArchiveInvalid, err)
	}
	if err := os.Rename(pendingDir, targetDir); err != nil {
		return err
	}
	return nil
}

func isValidCRSPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	rulesDir := filepath.Join(path, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".conf") {
			return true
		}
	}
	return false
}

func sanitizeReleaseVersion(version string) string {
	clean := normalizeVersion(version)
	if clean == "" {
		return "latest"
	}
	out := make([]rune, 0, len(clean))
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "latest"
	}
	return string(out)
}

func isPathWithinRoot(path, root string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath = filepath.Clean(absPath)
	absRoot = filepath.Clean(absRoot)
	if absPath == absRoot {
		return true
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isVersionGreater(latest, current string) bool {
	lParts := parseVersionParts(normalizeVersion(latest))
	cParts := parseVersionParts(normalizeVersion(current))
	maxLen := len(lParts)
	if len(cParts) > maxLen {
		maxLen = len(cParts)
	}
	for i := 0; i < maxLen; i++ {
		lv := 0
		cv := 0
		if i < len(lParts) {
			lv = lParts[i]
		}
		if i < len(cParts) {
			cv = cParts[i]
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return false
}

func parseVersionParts(value string) []int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			out = append(out, 0)
			continue
		}
		digits := make([]rune, 0, len(part))
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			digits = append(digits, r)
		}
		if len(digits) == 0 {
			out = append(out, 0)
			continue
		}
		v := 0
		for _, r := range digits {
			v = v*10 + int(r-'0')
		}
		out = append(out, v)
	}
	return slices.Clip(out)
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	return value
}
