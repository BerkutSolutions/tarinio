package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type crsRoundTripper func(*http.Request) (*http.Response, error)

func (fn crsRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetchLatestCRSReleaseUsesOfficialReleaseDigestWithoutEnvironment(t *testing.T) {
	t.Setenv(crsTrustedDigestsEnv, "")
	manager := newCRSManager(t.TempDir(), "")
	manager.httpClient = &http.Client{Transport: crsRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v4.29.0","html_url":"https://github.com/coreruleset/coreruleset/releases/tag/v4.29.0","assets":[{"name":"coreruleset-4.29.0-minimal.tar.gz","browser_download_url":"https://github.com/coreruleset/coreruleset/releases/download/v4.29.0/coreruleset-4.29.0-minimal.tar.gz","digest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	release, err := manager.fetchLatestReleaseLocked()
	if err != nil {
		t.Fatalf("fetch release: %v", err)
	}
	const expectedDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if release.Digest != expectedDigest {
		t.Fatalf("expected official release digest, got %q", release.Digest)
	}
	if release.DownloadURL != "https://github.com/coreruleset/coreruleset/releases/download/v4.29.0/coreruleset-4.29.0-minimal.tar.gz" {
		t.Fatalf("expected release asset URL, got %q", release.DownloadURL)
	}
}

func TestFetchLatestCRSReleaseRejectsUntrustedAssetURL(t *testing.T) {
	manager := newCRSManager(t.TempDir(), "")
	manager.httpClient = &http.Client{Transport: crsRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v4.29.0","html_url":"https://github.com/coreruleset/coreruleset/releases/tag/v4.29.0","assets":[{"name":"coreruleset-4.29.0-minimal.tar.gz","browser_download_url":"https://example.invalid/coreruleset-4.29.0-minimal.tar.gz","digest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	_, err := manager.fetchLatestReleaseLocked()
	if crsErrorCode(err) != crsErrorReleaseInvalid {
		t.Fatalf("expected official source validation error, got %v", err)
	}
}

func TestFetchLatestCRSReleaseRejectsMissingOfficialDigest(t *testing.T) {
	manager := newCRSManager(t.TempDir(), "")
	manager.httpClient = &http.Client{Transport: crsRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v4.29.0","html_url":"https://github.com/coreruleset/coreruleset/releases/tag/v4.29.0","assets":[{"name":"coreruleset-4.29.0-minimal.tar.gz","browser_download_url":"https://github.com/coreruleset/coreruleset/releases/download/v4.29.0/coreruleset-4.29.0-minimal.tar.gz"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	_, err := manager.fetchLatestReleaseLocked()
	if crsErrorCode(err) != crsErrorDigestInvalid {
		t.Fatalf("expected missing official digest error, got %v", err)
	}
}
