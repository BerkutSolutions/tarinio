package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type crsReleasePayload struct {
	TagName string            `json:"tag_name"`
	HTMLURL string            `json:"html_url"`
	Assets  []crsReleaseAsset `json:"assets"`
}

type crsReleaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

type latestCRSRelease struct {
	Version     string
	ReleaseURL  string
	DownloadURL string
	Digest      string
}

func (m *crsManager) fetchLatestReleaseLocked() (latestCRSRelease, error) {
	req, err := http.NewRequest(http.MethodGet, crsGitHubLatestReleaseAPI, nil)
	if err != nil {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseUnavailable, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", crsUserAgent)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseUnavailable, fmt.Errorf("GitHub release API returned status %d", resp.StatusCode))
	}

	var payload crsReleasePayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseInvalid, err)
	}
	version := normalizeVersion(payload.TagName)
	if version == "" {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseInvalid, errors.New("latest CRS release has empty tag_name"))
	}
	asset, ok := selectCRSReleaseAsset(payload.Assets, version)
	if !ok {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseInvalid, errors.New("latest CRS release has no matching minimal archive"))
	}
	if !isOfficialCRSReleaseURL(payload.HTMLURL, version) || !isOfficialCRSAssetURL(asset.URL, version) {
		return latestCRSRelease{}, newCRSError(crsErrorReleaseInvalid, errors.New("latest CRS release asset URL is not official"))
	}

	digest := strings.TrimSpace(m.trustedDigests[version])
	if digest == "" {
		var err error
		digest, err = parseCRSReleaseDigest(asset.Digest)
		if err != nil {
			return latestCRSRelease{}, newCRSError(crsErrorDigestInvalid, err)
		}
	}
	return latestCRSRelease{
		Version:     version,
		ReleaseURL:  strings.TrimSpace(payload.HTMLURL),
		DownloadURL: strings.TrimSpace(asset.URL),
		Digest:      digest,
	}, nil
}

func selectCRSReleaseAsset(assets []crsReleaseAsset, version string) (crsReleaseAsset, bool) {
	expectedName := "coreruleset-" + normalizeVersion(version) + "-minimal.tar.gz"
	for _, asset := range assets {
		if strings.EqualFold(strings.TrimSpace(asset.Name), expectedName) {
			return asset, true
		}
	}
	return crsReleaseAsset{}, false
}

func isOfficialCRSAssetURL(value, version string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.User != nil || !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Host, "github.com") {
		return false
	}
	expectedPath := "/coreruleset/coreruleset/releases/download/v" + normalizeVersion(version) + "/coreruleset-" + normalizeVersion(version) + "-minimal.tar.gz"
	return parsed.RawQuery == "" && parsed.Fragment == "" && parsed.EscapedPath() == expectedPath
}

func isOfficialCRSReleaseURL(value, version string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.User != nil || !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Host, "github.com") {
		return false
	}
	expectedPath := "/coreruleset/coreruleset/releases/tag/v" + normalizeVersion(version)
	return parsed.RawQuery == "" && parsed.Fragment == "" && parsed.EscapedPath() == expectedPath
}

func parseCRSReleaseDigest(value string) (string, error) {
	digest := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(digest), "sha256:") {
		return "", errors.New("release asset has no SHA-256 digest")
	}
	digest = strings.ToLower(strings.TrimSpace(digest[len("sha256:"):]))
	if len(digest) != 64 {
		return "", errors.New("release asset has invalid SHA-256 digest length")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", errors.New("release asset has invalid SHA-256 digest")
	}
	return digest, nil
}
