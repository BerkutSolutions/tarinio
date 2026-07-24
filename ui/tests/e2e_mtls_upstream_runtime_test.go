//go:build e2e

package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EUpstreamMTLSMaterialUploadCompileApplyRuntime(t *testing.T) {
	base := strings.TrimRight(os.Getenv("WAF_E2E_BASE_URL"), "/")
	runtimeURL := strings.TrimRight(os.Getenv("WAF_E2E_RUNTIME_URL"), "/")
	fixtureURL := strings.TrimRight(os.Getenv("WAF_E2E_MTLS_FIXTURE_URL"), "/")
	if base == "" || runtimeURL == "" || fixtureURL == "" {
		t.Fatal("WAF_E2E_BASE_URL, WAF_E2E_RUNTIME_URL and WAF_E2E_MTLS_FIXTURE_URL are required")
	}
	client, requestBase, hostOverride := newE2EClientAndBase(t, base)
	loginE2EUser(t, client, requestBase, hostOverride)
	getFixture := func(path string) []byte {
		resp, err := http.Get(fixtureURL + path)
		if err != nil {
			t.Fatalf("fixture %s: %v", path, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("fixture %s status=%d", path, resp.StatusCode)
		}
		return body
	}
	ca, caKey := getFixture("/__e2e/ca.pem"), getFixture("/__e2e/ca.key")
	clientCert, clientKey := getFixture("/__e2e/client.pem"), getFixture("/__e2e/client.key")
	siteID, upstreamID := e2eUniqueID(t, "e2e-mtls"), ""
	upstreamID = siteID + "-upstream"
	host := siteID + ".test"
	caID, clientID := siteID+"-ca", siteID+"-client"
	upload := func(id string, cert, key []byte) map[string]any {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("certificate_id", id)
		_ = w.WriteField("common_name", id)
		for _, f := range []struct {
			n string
			b []byte
		}{{"certificate_file", cert}, {"private_key_file", key}} {
			p, _ := w.CreateFormFile(f.n, f.n+".pem")
			_, _ = p.Write(f.b)
		}
		_ = w.Close()
		req, err := http.NewRequest(http.MethodPost, requestBase+"/api/certificate-materials/upload", &body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("upload %s status=%d body=%s", id, resp.StatusCode, raw)
		}
		return decodeE2EMap(t, raw)
	}
	caResult, clientResult := upload(caID, ca, caKey), upload(clientID, clientCert, clientKey)
	t.Cleanup(func() {
		for _, p := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false", "/api/certificates/" + caID, "/api/certificates/" + clientID} {
			r := requestE2EJSON(t, client, http.MethodDelete, requestBase+p, hostOverride, nil)
			_ = r.Body.Close()
		}
	})
	for _, item := range []struct {
		path    string
		payload map[string]any
	}{{"/api/sites?auto_apply=false", map[string]any{"id": siteID, "primary_host": host, "enabled": true, "listen_http": true, "use_easy_config": true, "default_upstream_id": upstreamID}}, {"/api/upstreams?auto_apply=false", map[string]any{"id": upstreamID, "site_id": siteID, "scheme": "https", "host": "mtls-upstream", "port": 8443}}} {
		r := postJSON(t, client, requestBase+item.path, hostOverride, item.payload)
		raw, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if r.StatusCode != http.StatusCreated && r.StatusCode != http.StatusOK {
			t.Fatalf("create status=%d body=%s", r.StatusCode, raw)
		}
	}
	profile := e2eGetProfile(t, client, requestBase, hostOverride, siteID)
	upstream := mapGetOrCreate(profile, "upstream_routing")
	upstream["upstream_mtls_enabled"] = true
	upstream["upstream_mtls_cert_ref"] = e2eMTLSMaterialRef(t, clientResult, "certificate_ref")
	upstream["upstream_mtls_key_ref"] = e2eMTLSMaterialRef(t, clientResult, "private_key_ref")
	upstream["upstream_mtls_ca_ref"] = e2eMTLSMaterialRef(t, caResult, "certificate_ref")
	upstream["reverse_proxy_ssl_sni"] = true
	upstream["reverse_proxy_ssl_sni_name"] = "mtls-upstream"
	e2ePutProfileWithoutAutoApply(t, client, requestBase, hostOverride, siteID, profile)
	revisionID := e2eCompileAndApply(t, client, requestBase, hostOverride)
	assertE2EArtifactActive(t, revisionID, "nginx/easy/"+siteID+".conf", "proxy_ssl_certificate", "proxy_ssl_verify on", "proxy_ssl_name mtls-upstream")
	req, _ := http.NewRequest(http.MethodGet, runtimeURL+"/mtls", nil)
	req.Host = host
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(raw), "mtls-upstream-ok") {
		t.Fatalf("mTLS upstream runtime status=%d body=%s", resp.StatusCode, raw)
	}
}

func decodeE2EMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func e2eMTLSMaterialRef(t *testing.T, value map[string]any, key string) string {
	t.Helper()
	material, _ := value["material"].(map[string]any)
	ref, _ := material[key].(string)
	if strings.TrimSpace(ref) == "" {
		t.Fatalf("upload response missing material.%s: %#v", key, value)
	}
	return ref
}
