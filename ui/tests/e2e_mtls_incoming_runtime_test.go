//go:build e2e

package tests

import (
	"bytes"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EIncomingMTLSClientCertificateRuntime(t *testing.T) {
	base := strings.TrimRight(os.Getenv("WAF_E2E_BASE_URL"), "/")
	httpsURL := strings.TrimRight(os.Getenv("WAF_E2E_RUNTIME_HTTPS_URL"), "/")
	fixtureURL := strings.TrimRight(os.Getenv("WAF_E2E_MTLS_FIXTURE_URL"), "/")
	if base == "" || httpsURL == "" || fixtureURL == "" {
		t.Fatal("WAF_E2E_BASE_URL, WAF_E2E_RUNTIME_HTTPS_URL and WAF_E2E_MTLS_FIXTURE_URL are required")
	}
	client, requestBase, hostOverride := newE2EClientAndBase(t, base)
	loginE2EUser(t, client, requestBase, hostOverride)
	getFixture := func(path string) []byte {
		response, err := http.Get(fixtureURL + path)
		if err != nil {
			t.Fatalf("read mTLS fixture %s: %v", path, err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("mTLS fixture %s status=%d body=%s", path, response.StatusCode, body)
		}
		return body
	}
	ca, caKey := getFixture("/__e2e/ca.pem"), getFixture("/__e2e/ca.key")
	clientCert, clientKey := getFixture("/__e2e/client.pem"), getFixture("/__e2e/client.key")
	siteID := e2eUniqueID(t, "e2e-incoming-mtls")
	upstreamID, certID := siteID+"-upstream", siteID+"-tls"
	host := siteID + ".test"
	caID := siteID + "-client-ca"
	uploadedCA := uploadE2EMTLSMaterial(t, client, requestBase, hostOverride, caID, ca, caKey)
	t.Cleanup(func() {
		for _, path := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false", "/api/tls-configs/" + siteID + "?auto_apply=false", "/api/certificates/" + certID} {
			response := requestE2EJSON(t, client, http.MethodDelete, requestBase+path, hostOverride, nil)
			_ = response.Body.Close()
		}
	})
	for _, item := range []struct {
		path string
		body map[string]any
	}{
		{"/api/sites?auto_apply=false", map[string]any{"id": siteID, "primary_host": host, "enabled": true, "listen_http": true, "listen_https": true, "use_easy_config": true, "default_upstream_id": upstreamID}},
		{"/api/upstreams?auto_apply=false", map[string]any{"id": upstreamID, "site_id": siteID, "scheme": "http", "host": "upstream-echo", "port": 8888}},
	} {
		response := postJSON(t, client, requestBase+item.path, hostOverride, item.body)
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
			t.Fatalf("create %s status=%d body=%s", item.path, response.StatusCode, body)
		}
	}
	issued := postJSON(t, client, requestBase+"/api/certificates/self-signed/issue", hostOverride, map[string]any{"certificate_id": certID, "common_name": host, "san_list": []string{}})
	assertE2EStatus(t, issued, "issue WAF TLS certificate", http.StatusCreated, http.StatusOK)
	bound := postJSON(t, client, requestBase+"/api/tls-configs?auto_apply=false", hostOverride, map[string]any{"site_id": siteID, "certificate_id": certID})
	assertE2EStatus(t, bound, "bind WAF TLS certificate", http.StatusCreated, http.StatusOK)
	profile := e2eGetProfile(t, client, requestBase, hostOverride, siteID)
	front := mapGetOrCreate(profile, "front_service")
	front["mtls_enabled"] = true
	front["mtls_optional"] = false
	front["mtls_verify_depth"] = 2
	front["mtls_client_ca_ref"] = e2eMTLSMaterialRef(t, uploadedCA, "certificate_ref")
	e2ePutProfileWithoutAutoApply(t, client, requestBase, hostOverride, siteID, profile)
	revisionID := e2eCompileAndApply(t, client, requestBase, hostOverride)
	if revisionID == "" {
		t.Fatal("compile/apply incoming mTLS revision failed")
	}
	assertE2EArtifactActive(t, revisionID, "nginx/sites/"+siteID+".conf", "ssl_verify_client on", "ssl_client_certificate /etc/waf/tls/materials/")
	withoutCertificate := incomingMTLSClient(t, host, nil)
	assertIncomingMTLSRejected(t, withoutCertificate, httpsURL, host)
	certificate, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		t.Fatalf("load fixture client certificate: %v", err)
	}
	withCertificate := incomingMTLSClient(t, host, &certificate)
	assertIncomingMTLSPassed(t, withCertificate, httpsURL, host)
	front["mtls_enabled"] = false
	front["mtls_client_ca_ref"] = ""
	e2ePutProfileWithoutAutoApply(t, client, requestBase, hostOverride, siteID, profile)
	if revisionID = e2eCompileAndApply(t, client, requestBase, hostOverride); revisionID == "" {
		t.Fatal("compile/apply mTLS disable revision failed")
	}
	assertE2EArtifactActiveWithout(t, revisionID, "nginx/sites/"+siteID+".conf", "ssl_verify_client on", "ssl_client_certificate ")
	assertIncomingMTLSPassed(t, withoutCertificate, httpsURL, host)
}

func uploadE2EMTLSMaterial(t *testing.T, client *http.Client, requestBase, hostOverride, id string, certificate, privateKey []byte) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("certificate_id", id)
	_ = writer.WriteField("common_name", id)
	for _, file := range []struct {
		name    string
		content []byte
	}{{"certificate_file", certificate}, {"private_key_file", privateKey}} {
		part, err := writer.CreateFormFile(file.name, file.name+".pem")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(file.content)
	}
	_ = writer.Close()
	request, err := http.NewRequest(http.MethodPost, requestBase+"/api/certificate-materials/upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if hostOverride != "" {
		request.Host = hostOverride
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("upload %s status=%d body=%s", id, response.StatusCode, raw)
	}
	return decodeE2EMap(t, raw)
}

func incomingMTLSClient(t *testing.T, serverName string, certificate *tls.Certificate) *http.Client {
	t.Helper()
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ServerName: serverName}
	if certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*certificate}
	}
	return &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{Proxy: nil, TLSClientConfig: tlsConfig}}
}

func assertIncomingMTLSRejected(t *testing.T, client *http.Client, endpoint, host string) {
	t.Helper()
	request, _ := http.NewRequest(http.MethodGet, endpoint+"/incoming-mtls", nil)
	request.Host = host
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 400 {
		t.Fatalf("mTLS without client certificate unexpectedly passed: status=%d", response.StatusCode)
	}
}

func assertIncomingMTLSPassed(t *testing.T, client *http.Client, endpoint, host string) {
	t.Helper()
	request, _ := http.NewRequest(http.MethodGet, endpoint+"/incoming-mtls", nil)
	request.Host = host
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("mTLS request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("mTLS request status=%d body=%s", response.StatusCode, body)
	}
}
