package main

import (
	"archive/tar"
	"compress/gzip"
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
}

type crsStateFile struct {
	ActiveVersion           string `json:"active_version"`
	ActivePath              string `json:"active_path"`
	HourlyAutoUpdateEnabled bool   `json:"hourly_auto_update_enabled"`
	FirstStartComplete      bool   `json:"first_start_complete"`
}

type crsReleasePayload struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	TarballURL string `json:"tarball_url"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

type crsManager struct {
	mu sync.Mutex

	rootDir         string
	statePath       string
	systemPath      string
	httpClient      *http.Client
	activePath      string
	activeVersion   string
	latestVersion   string
	latestRelease   string
	lastCheckedAt   string
	hourlyAuto      bool
	firstStartDone  bool
	lastError       string
}

func newCRSManager(rootDir, systemPath string) *crsManager {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = "/var/lib/waf/crs"
	}
	return &crsManager{
		rootDir:    rootDir,
		statePath:  filepath.Join(rootDir, "state.json"),
		systemPath: strings.TrimSpace(systemPath),
		httpClient: &http.Client{Timeout: 45 * time.Second},
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
	m.lastError = ""
	_ = m.saveStateLocked()
}

func (m *crsManager) CheckForUpdates(_ bool) (crsStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	release, err := m.fetchLatestReleaseLocked()
	if err != nil {
		m.lastError = err.Error()
		_ = m.saveStateLocked()
		return m.statusLocked(), err
	}
	m.latestVersion = release.Version
	m.latestRelease = release.ReleaseURL
	m.lastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	m.lastError = ""
	_ = m.saveStateLocked()
	return m.statusLocked(), nil
}

func (m *crsManager) UpdateToLatest(_ bool) (crsStatus, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	release, err := m.fetchLatestReleaseLocked()
	if err != nil {
		m.lastError = err.Error()
		_ = m.saveStateLocked()
		return m.statusLocked(), false, err
	}
	m.latestVersion = release.Version
	m.latestRelease = release.ReleaseURL
	m.lastCheckedAt = time.Now().UTC().Format(time.RFC3339)

	if !isVersionGreater(release.Version, m.activeVersion) {
		m.firstStartDone = true
		m.lastError = ""
		_ = m.saveStateLocked()
		return m.statusLocked(), false, nil
	}

	versionPath := filepath.Join(m.rootDir, "releases", sanitizeReleaseVersion(release.Version))
	if !isValidCRSPath(versionPath) {
		if err := m.downloadAndExtractLocked(release.DownloadURL, versionPath); err != nil {
			m.lastError = err.Error()
			_ = m.saveStateLocked()
			return m.statusLocked(), false, err
		}
	}

	m.activeVersion = release.Version
	m.activePath = versionPath
	m.firstStartDone = true
	m.lastError = ""
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

type latestCRSRelease struct {
	Version    string
	ReleaseURL string
	DownloadURL string
}

func (m *crsManager) fetchLatestReleaseLocked() (latestCRSRelease, error) {
	req, err := http.NewRequest(http.MethodGet, crsGitHubLatestReleaseAPI, nil)
	if err != nil {
		return latestCRSRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", crsUserAgent)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return latestCRSRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return latestCRSRelease{}, fmt.Errorf("github release API returned status %d", resp.StatusCode)
	}
	var payload crsReleasePayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return latestCRSRelease{}, err
	}
	version := normalizeVersion(payload.TagName)
	if version == "" {
		return latestCRSRelease{}, errors.New("latest CRS release has empty tag_name")
	}
	downloadURL := strings.TrimSpace(payload.TarballURL)
	for _, asset := range payload.Assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		url := strings.TrimSpace(asset.URL)
		if url == "" {
			continue
		}
		if strings.HasSuffix(name, ".tar.gz") && strings.Contains(name, "coreruleset") {
			downloadURL = url
			break
		}
	}
	if downloadURL == "" {
		return latestCRSRelease{}, errors.New("latest CRS release has no downloadable archive")
	}
	return latestCRSRelease{
		Version:     version,
		ReleaseURL:  strings.TrimSpace(payload.HTMLURL),
		DownloadURL: downloadURL,
	}, nil
}

func (m *crsManager) downloadAndExtractLocked(url, targetDir string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("crs download url is required")
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", crsUserAgent)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("crs archive download failed: status %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
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
		dest := filepath.Join(targetDir, filepath.FromSlash(relPath))
		if !isPathWithinRoot(dest, targetDir) {
			return fmt.Errorf("invalid tar path: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, reader); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	if !isValidCRSPath(targetDir) {
		return errors.New("downloaded CRS archive is missing rules/*.conf")
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
