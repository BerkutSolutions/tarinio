//go:build e2e

package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type e2eBansPolicy struct {
	ID       string   `json:"id"`
	SiteID   string   `json:"site_id"`
	Enabled  bool     `json:"enabled"`
	DenyList []string `json:"denylist"`
}

type e2eRevisionContractItem struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Checksum string `json:"checksum"`
	IsActive bool   `json:"is_active"`
}

type e2eRevisionContractCatalog struct {
	Revisions []e2eRevisionContractItem `json:"revisions"`
}

func TestE2EBansReadContract(t *testing.T) {
	client, baseURL, host := e2eContractSession(t)
	before := e2eReadBansPolicies(t, client, baseURL, host)

	var sites []map[string]any
	e2eDecodeStatus(t, getWithAuth(t, client, baseURL+"/api/sites", host), http.StatusOK, &sites)
	if len(sites) == 0 {
		t.Fatal("bans read contract requires at least one disposable-stack site")
	}
	for index, site := range sites {
		if strings.TrimSpace(e2eString(site["id"])) == "" || strings.TrimSpace(e2eString(site["primary_host"])) == "" {
			t.Fatalf("site %d misses bans identity fields: %#v", index, site)
		}
	}
	for index, policy := range before {
		if strings.TrimSpace(policy.ID) == "" || strings.TrimSpace(policy.SiteID) == "" {
			t.Fatalf("access policy %d misses id/site_id: %#v", index, policy)
		}
		if policy.DenyList == nil {
			t.Fatalf("access policy %s must expose a denylist array", policy.ID)
		}
	}
	var events struct {
		Events []map[string]any `json:"events"`
		Total  int              `json:"total"`
	}
	e2eDecodeStatus(t, getWithAuth(t, client, baseURL+"/api/events?limit=1&offset=0", host), http.StatusOK, &events)
	if events.Events == nil || events.Total < len(events.Events) {
		t.Fatalf("events read contract is inconsistent: total=%d items=%d", events.Total, len(events.Events))
	}
	after := e2eReadBansPolicies(t, client, baseURL, host)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("bans read endpoints mutated access policies: before=%#v after=%#v", before, after)
	}
}

func TestE2EBansDurationAndUnauthorizedSiteErrors(t *testing.T) {
	client, baseURL, host := e2eContractSession(t)
	var sites []map[string]any
	e2eDecodeStatus(t, getWithAuth(t, client, baseURL+"/api/sites", host), http.StatusOK, &sites)
	if len(sites) == 0 {
		t.Fatal("ban validation requires an existing disposable-stack site")
	}
	siteID := strings.TrimSpace(e2eString(sites[0]["id"]))
	if siteID == "" {
		t.Fatalf("first site has no id: %#v", sites[0])
	}
	before := e2eReadBansPolicies(t, client, baseURL, host)

	t.Run("InvalidAddress", func(t *testing.T) {
		resp := postJSON(t, client, baseURL+"/api/sites/"+siteID+"/ban", host, map[string]any{"ip": "not-an-ip", "duration_seconds": -1})
		e2eRequireErrorStatus(t, resp, http.StatusBadRequest)
	})
	t.Run("MissingSite", func(t *testing.T) {
		missingSite := e2eUniqueID(t, "missing-ban")
		resp := postJSON(t, client, baseURL+"/api/sites/"+missingSite+"/ban", host, map[string]any{"ip": "203.0.113.199"})
		e2eRequireErrorStatus(t, resp, http.StatusNotFound)
	})

	after := e2eReadBansPolicies(t, client, baseURL, host)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected ban requests mutated access policies: before=%#v after=%#v", before, after)
	}
}

func TestE2ERevisionsValidationDeleteActiveAndMissingErrors(t *testing.T) {
	client, baseURL, host := e2eContractSession(t)
	before := e2eReadRevisionContract(t, client, baseURL, host)
	activeID := ""
	for _, revision := range before.Revisions {
		if revision.IsActive {
			activeID = revision.ID
			break
		}
	}
	if activeID == "" {
		t.Fatal("revision validation requires an active revision")
	}

	t.Run("DeleteActive", func(t *testing.T) {
		resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+"/api/revisions/"+activeID, host, nil)
		e2eRequireErrorStatus(t, resp, http.StatusConflict)
	})
	t.Run("ApplyMissing", func(t *testing.T) {
		missingRevision := e2eUniqueID(t, "missing-rev")
		resp := postJSON(t, client, baseURL+"/api/revisions/"+missingRevision+"/apply", host, map[string]any{})
		e2eRequireErrorStatus(t, resp, http.StatusNotFound)
	})
	t.Run("DeleteMissing", func(t *testing.T) {
		missingRevision := e2eUniqueID(t, "missing-del")
		resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+"/api/revisions/"+missingRevision, host, nil)
		e2eRequireErrorStatus(t, resp, http.StatusNotFound)
	})

	after := e2eReadRevisionContract(t, client, baseURL, host)
	if !reflect.DeepEqual(after.Revisions, before.Revisions) {
		t.Fatalf("rejected revision operations mutated catalog: before=%#v after=%#v", before.Revisions, after.Revisions)
	}
	for _, revision := range after.Revisions {
		if revision.ID == activeID && revision.IsActive {
			return
		}
	}
	t.Fatalf("active revision %s changed after rejected operations", activeID)
}

func e2eContractSession(t *testing.T) (*http.Client, string, string) {
	t.Helper()
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Fatal("WAF_E2E_BASE_URL is required for tagged E2E contracts")
	}
	client, requestBaseURL, host := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, host)
	return client, requestBaseURL, host
}

func e2eReadBansPolicies(t *testing.T, client *http.Client, baseURL, host string) []e2eBansPolicy {
	t.Helper()
	var policies []e2eBansPolicy
	e2eDecodeStatus(t, getWithAuth(t, client, baseURL+"/api/access-policies", host), http.StatusOK, &policies)
	if policies == nil {
		policies = []e2eBansPolicy{}
	}
	return policies
}

func e2eReadRevisionContract(t *testing.T, client *http.Client, baseURL, host string) e2eRevisionContractCatalog {
	t.Helper()
	var catalog e2eRevisionContractCatalog
	e2eDecodeStatus(t, getWithAuth(t, client, baseURL+"/api/revisions", host), http.StatusOK, &catalog)
	if catalog.Revisions == nil {
		t.Fatal("revision catalog must expose a revisions array")
	}
	return catalog
}

func e2eDecodeStatus(t *testing.T, resp *http.Response, expected int, output any) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		t.Fatalf("%s %s: status=%d want=%d body=%s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, expected, mustReadBody(t, resp.Body))
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		t.Fatalf("decode %s response: %v", resp.Request.URL.Path, err)
	}
}

func e2eRequireErrorStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		t.Fatalf("%s %s: status=%d want=%d body=%s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, expected, mustReadBody(t, resp.Body))
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.Error) == "" {
		t.Fatalf("%s %s returned no structured error: error=%q decode=%v", resp.Request.Method, resp.Request.URL.Path, payload.Error, err)
	}
}

func e2eString(value any) string {
	text, _ := value.(string)
	return text
}
