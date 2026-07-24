//go:build e2e

package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type registryEvent struct {
	ID                string         `json:"id"`
	Type              string         `json:"type"`
	Severity          string         `json:"severity"`
	SiteID            string         `json:"site_id"`
	SourceComponent   string         `json:"source_component"`
	OccurredAt        string         `json:"occurred_at"`
	Summary           string         `json:"summary"`
	Details           map[string]any `json:"details"`
	RelatedRevisionID string         `json:"related_revision_id"`
	RelatedJobID      string         `json:"related_job_id"`
}

type registryEventsPage struct {
	Events               []registryEvent `json:"events"`
	Total, Limit, Offset int
}

type registryAuditEvent struct {
	ID                string         `json:"id"`
	ActorUserID       string         `json:"actor_user_id"`
	ActorIP           string         `json:"actor_ip"`
	Action            string         `json:"action"`
	ResourceType      string         `json:"resource_type"`
	ResourceID        string         `json:"resource_id"`
	SiteID            string         `json:"site_id"`
	RelatedRevisionID string         `json:"related_revision_id"`
	RelatedJobID      string         `json:"related_job_id"`
	Status            string         `json:"status"`
	OccurredAt        string         `json:"occurred_at"`
	Summary           string         `json:"summary"`
	Details           map[string]any `json:"details_json"`
	Hash              string         `json:"hash"`
}

type registryAuditPage struct {
	Items                []registryAuditEvent `json:"items"`
	Total, Limit, Offset int
}

func TestE2EEventsAPIFilterPaginationDetailContract(t *testing.T) {
	client, baseURL, host := registryE2ESession(t)
	from := time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
	var revisions map[string]any
	registryGetJSON(t, client, baseURL+"/api/revisions", host, &revisions)
	items, ok := revisions["revisions"].([]any)
	if !ok {
		t.Fatalf("revision catalog schema mismatch: %#v", revisions)
	}
	activeID := ""
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if active, _ := item["is_active"].(bool); active {
			activeID = strings.TrimSpace(fmt.Sprint(item["id"]))
			break
		}
	}
	if activeID == "" {
		t.Fatal("disposable E2E stack has no active revision")
	}

	t.Run("RealActiveRevisionApplySeed", func(t *testing.T) {
		resp := requestE2EJSON(t, client, http.MethodPost, baseURL+"/api/revisions/"+url.PathEscape(activeID)+"/apply", host, map[string]any{})
		registryRequireStatusAndClose(t, resp, http.StatusCreated)
	})

	siteID := fmt.Sprintf("e2e-events-%d", time.Now().UTC().UnixNano())
	upstreamID := siteID + "-upstream"
	siteHost := siteID + ".test"
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/upstreams/" + upstreamID + "?auto_apply=false", "/api/sites/" + siteID + "?auto_apply=false"} {
			resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+endpoint, host, nil)
			_ = resp.Body.Close()
		}
		e2eCompileAndApply(t, client, baseURL, host)
	})
	t.Run("RealSiteScopedRuntimeEventSeed", func(t *testing.T) {
		createE2EModSecuritySite(t, client, baseURL, host, siteID, upstreamID, siteHost)
		profile := e2eGetProfile(t, client, baseURL, host, siteID)
		modsecurity := mapGetOrCreate(profile, "security_modsecurity")
		modsecurity["use_modsecurity"] = true
		modsecurity["use_modsecurity_crs_plugins"] = false
		modsecurity["use_modsecurity_custom_configuration"] = true
		modsecurity["custom_configuration"] = map[string]any{
			"path":    "modsec/e2e-events-site-filter.conf",
			"content": `SecRule REQUEST_URI "@streq /events-site-filter" "id:100901,phase:2,deny,status:403,log"`,
		}
		e2ePutProfileWithoutAutoApply(t, client, baseURL, host, siteID, profile)
		if revisionID := e2eCompileAndApply(t, client, baseURL, host); revisionID == "" {
			t.Fatal("compile/apply did not produce a revision for the Events site-scoped seed")
		}
		runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
		if runtimeURL == "" {
			t.Fatal("WAF_E2E_RUNTIME_URL is required for the Events site-scoped seed")
		}
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/events-site-filter", nil)
		if err != nil {
			t.Fatalf("build runtime seed request: %v", err)
		}
		req.Host = siteHost
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			t.Fatalf("send runtime seed request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("runtime seed status=%d, want 403", resp.StatusCode)
		}
		from = time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
	})

	var selected registryEvent
	t.Run("ExactSchemaAndStableDetail", func(t *testing.T) {
		page := registryGetEvents(t, client, baseURL, host, url.Values{"from": {from}, "limit": {"500"}})
		for _, event := range page.Events {
			if event.Type == "apply_succeeded" && event.RelatedRevisionID == activeID {
				selected = event
				break
			}
		}
		if page.Total != len(page.Events) || page.Limit != 500 || page.Offset != 0 {
			t.Fatalf("events page metadata mismatch: %+v", page)
		}
		registryRequireEventSchema(t, selected)
		query := url.Values{"type": {selected.Type}, "severity": {selected.Severity}, "from": {selected.OccurredAt}, "to": {selected.OccurredAt}, "limit": {"500"}}
		detailPage := registryGetEvents(t, client, baseURL, host, query)
		matched := false
		for _, event := range detailPage.Events {
			if event.Type != selected.Type || event.Severity != selected.Severity || event.OccurredAt != selected.OccurredAt {
				t.Fatalf("combined event filter leaked a row: %+v", event)
			}
			if event.ID == selected.ID {
				if !reflect.DeepEqual(event, selected) {
					t.Fatalf("event detail changed between reads: first=%+v second=%+v", selected, event)
				}
				matched = true
			}
		}
		if !matched {
			t.Fatalf("stable event detail %s is absent from exact boundary query", selected.ID)
		}
	})

	t.Run("SiteFilter", func(t *testing.T) {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			sitePage := registryGetEvents(t, client, baseURL, host, url.Values{"site_id": {siteID}, "limit": {"500"}})
			for _, event := range sitePage.Events {
				if event.SiteID != siteID {
					t.Fatalf("site filter leaked row: %+v", event)
				}
				if event.Type == "security_access" && event.Details["path"] == "/events-site-filter" {
					return
				}
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Fatalf("site filter did not return the real runtime security event for %s", siteID)
	})

	t.Run("DeterministicPaginationAndValidation", func(t *testing.T) {
		first := registryGetEvents(t, client, baseURL, host, url.Values{"from": {from}, "limit": {"1"}, "offset": {"0"}})
		second := registryGetEvents(t, client, baseURL, host, url.Values{"from": {from}, "limit": {"1"}, "offset": {"1"}})
		if first.Total < 2 || second.Total != first.Total || len(first.Events) != 1 || len(second.Events) != 1 || first.Events[0].ID == second.Events[0].ID {
			t.Fatalf("unstable events pagination: first=%+v second=%+v", first, second)
		}
		if first.Events[0].OccurredAt < second.Events[0].OccurredAt {
			t.Fatalf("events are not newest-first: %s before %s", first.Events[0].OccurredAt, second.Events[0].OccurredAt)
		}
		resp := getWithAuth(t, client, baseURL+"/api/events?from=not-rfc3339", host)
		registryRequireStatusAndClose(t, resp, http.StatusBadRequest)
	})
}

func TestE2EActivityAPIFilterPaginationAndCriticalMutations(t *testing.T) {
	client, baseURL, host := registryE2ESession(t)
	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	siteID, userID, ip := "e2e-audit-"+suffix, "e2e-audit-user-"+suffix, "203.0.113.211"
	from := time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
	var revisionID string
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/access-policies/" + siteID + "-access?auto_apply=false", "/api/sites/" + siteID + "?auto_apply=false", "/api/revisions/" + revisionID, "/api/administration/users/" + userID} {
			if strings.HasSuffix(endpoint, "/") {
				continue
			}
			resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+endpoint, host, nil)
			_ = resp.Body.Close()
		}
	})

	t.Run("CriticalMutationsProduceAuditEvidence", func(t *testing.T) {
		registryRequireStatusAndClose(t, postJSON(t, client, baseURL+"/api/sites?auto_apply=false", host, map[string]any{"id": siteID, "primary_host": siteID + ".test", "enabled": true, "listen_http": true}), http.StatusCreated)
		registryRequireStatusAndClose(t, requestE2EJSON(t, client, http.MethodPost, baseURL+"/api/sites/"+siteID+"/ban?auto_apply=false", host, map[string]any{"ip": ip}), http.StatusOK)
		registryRequireStatusAndClose(t, requestE2EJSON(t, client, http.MethodPost, baseURL+"/api/sites/"+siteID+"/unban?auto_apply=false", host, map[string]any{"ip": ip}), http.StatusOK)
		compile := requestE2EJSON(t, client, http.MethodPost, baseURL+"/api/revisions/compile", host, map[string]any{})
		registryRequireStatus(t, compile, http.StatusCreated)
		var compiled struct {
			Revision struct {
				ID string `json:"id"`
			} `json:"revision"`
		}
		registryDecodeBody(t, compile, &compiled)
		revisionID = compiled.Revision.ID
		if revisionID == "" {
			t.Fatal("compile returned empty revision id")
		}
		registryRequireStatusAndClose(t, postJSON(t, client, baseURL+"/api/administration/users", host, map[string]any{"id": userID, "username": userID, "email": userID + "@example.test", "password": "E2e-Audit-1234!", "role_ids": []string{"auditor"}, "is_active": true}), http.StatusCreated)
		registryRequireStatusAndClose(t, requestE2EJSON(t, client, http.MethodDelete, baseURL+"/api/administration/users/"+userID, host, nil), http.StatusOK)
		var settings map[string]any
		registryGetJSON(t, client, baseURL+"/api/anti-ddos/settings", host, &settings)
		registryRequireStatusAndClose(t, requestE2EJSON(t, client, http.MethodPut, baseURL+"/api/anti-ddos/settings?auto_apply=false", host, settings), http.StatusOK)

		for action, resourceID := range map[string]string{"accesspolicy.unban": siteID, "revision.compile_request": revisionID, "antiddos.settings.upsert": "global", "administration.user.delete": userID} {
			query := url.Values{"action": {action}, "resource_id": {resourceID}, "status": {"succeeded"}, "from": {from}, "limit": {"10"}}
			page := registryGetAudit(t, client, baseURL, host, query)
			if len(page.Items) == 0 {
				t.Fatalf("critical audit evidence is absent for %s/%s", action, resourceID)
			}
			if page.Items[0].Action != action || page.Items[0].ResourceID != resourceID {
				t.Fatalf("critical audit mismatch for %s: %+v", action, page.Items[0])
			}
			registryRequireAuditSchema(t, page.Items[0])
		}
	})

	t.Run("ExactFilterMatrixAndDetails", func(t *testing.T) {
		query := url.Values{"action": {"accesspolicy.unban"}, "resource_type": {"accesspolicy"}, "resource_id": {siteID}, "site_id": {siteID}, "status": {"succeeded"}, "category": {"config"}, "from": {from}, "limit": {"10"}}
		page := registryGetAudit(t, client, baseURL, host, query)
		if len(page.Items) != 1 {
			t.Fatalf("exact audit filter expected one row, got %+v", page)
		}
		item := page.Items[0]
		if item.Details["ip"] != ip {
			t.Fatalf("audit detail IP mismatch: %+v", item)
		}
		if item.ActorUserID == "" || item.ActorIP == "" {
			t.Fatalf("audit actor fields are empty: %+v", item)
		}
		query.Set("actor_user_id", item.ActorUserID)
		query.Set("actor_ip", item.ActorIP)
		query.Set("to", item.OccurredAt)
		filtered := registryGetAudit(t, client, baseURL, host, query)
		if len(filtered.Items) != 1 || !reflect.DeepEqual(filtered.Items[0], item) {
			t.Fatalf("audit exact readback changed: %+v", filtered)
		}
	})

	t.Run("DeterministicPaginationAndValidation", func(t *testing.T) {
		first := registryGetAudit(t, client, baseURL, host, url.Values{"from": {from}, "limit": {"1"}, "offset": {"0"}})
		second := registryGetAudit(t, client, baseURL, host, url.Values{"from": {from}, "limit": {"1"}, "offset": {"1"}})
		if first.Total < 4 || second.Total != first.Total || len(first.Items) != 1 || len(second.Items) != 1 || first.Items[0].ID == second.Items[0].ID {
			t.Fatalf("unstable audit pagination: first=%+v second=%+v", first, second)
		}
		for _, raw := range []string{"from=bad", "status=bad", "category=bad", "offset=-1"} {
			resp := getWithAuth(t, client, baseURL+"/api/audit?"+raw, host)
			registryRequireStatusAndClose(t, resp, http.StatusBadRequest)
		}
	})
}

func registryE2ESession(t *testing.T) (*http.Client, string, string) {
	t.Helper()
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if base == "" {
		t.Fatal("WAF_E2E_BASE_URL is required")
	}
	client, requestBase, host := newE2EClientAndBase(t, base)
	loginE2EUser(t, client, requestBase, host)
	return client, requestBase, host
}

func registryGetEvents(t *testing.T, c *http.Client, base, host string, q url.Values) registryEventsPage {
	var out registryEventsPage
	registryGetJSON(t, c, base+"/api/events?"+q.Encode(), host, &out)
	return out
}
func registryGetAudit(t *testing.T, c *http.Client, base, host string, q url.Values) registryAuditPage {
	var out registryAuditPage
	registryGetJSON(t, c, base+"/api/audit?"+q.Encode(), host, &out)
	return out
}
func registryGetJSON(t *testing.T, c *http.Client, endpoint, host string, out any) {
	t.Helper()
	resp := getWithAuth(t, c, endpoint, host)
	registryRequireStatus(t, resp, http.StatusOK)
	registryDecodeBody(t, resp, out)
}
func registryDecodeBody(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", resp.Request.URL, err)
	}
}
func registryRequireStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("%s %s: want=%d got=%d body=%s", resp.Request.Method, resp.Request.URL, want, resp.StatusCode, mustReadBody(t, resp.Body))
	}
}
func registryRequireStatusAndClose(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	registryRequireStatus(t, resp, want)
	_ = resp.Body.Close()
}

func registryRequireEventSchema(t *testing.T, e registryEvent) {
	t.Helper()
	if e.ID == "" || e.Type == "" || e.Severity == "" || e.SourceComponent == "" || e.OccurredAt == "" || e.Summary == "" || e.RelatedRevisionID == "" || e.RelatedJobID == "" {
		t.Fatalf("incomplete event schema: %+v", e)
	}
	if _, err := time.Parse(time.RFC3339Nano, e.OccurredAt); err != nil {
		t.Fatalf("invalid event time %q: %v", e.OccurredAt, err)
	}
}
func registryRequireAuditSchema(t *testing.T, e registryAuditEvent) {
	t.Helper()
	if e.ID == "" || e.Action == "" || e.ResourceType == "" || e.ResourceID == "" || e.Status != "succeeded" || e.OccurredAt == "" || e.Summary == "" || e.Hash == "" {
		t.Fatalf("incomplete audit schema: %+v", e)
	}
	if _, err := time.Parse(time.RFC3339Nano, e.OccurredAt); err != nil {
		t.Fatalf("invalid audit time %q: %v", e.OccurredAt, err)
	}
}
