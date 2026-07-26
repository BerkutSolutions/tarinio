import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.toggles-conditional-fields-dynamic-lists-readback", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(5 * 60_000);
  page.setDefaultTimeout(10_000);
  const siteID = e2eID(testInfo, "e2e-dynamic");
  const upstreamID = `${siteID}-upstream`;
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : (payload?.items || []);
  };
  const addListValue = async (field: string, value: string) => {
    await page.locator(`#list-input-${field}`).fill(value);
    await page.locator(`[data-list-add="${field}"]`).click();
    await expect(page.locator(`[data-list-field="${field}"]`)).toContainText(value);
  };
  const cleanup = new CleanupLedger();
  cleanup.add(`profile ${siteID}`, () => api(`/api/easy-site-profiles/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add(`upstream ${upstreamID}`, () => api(`/api/upstreams/${upstreamID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add(`site ${siteID}`, () => api(`/api/sites/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: `${siteID}.test`, enabled: true, listen_http: true, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
    expect([200, 201], result.body).toContain(result.status);

    await openPage(page, `/services/${siteID}`, page.locator("#service-editor-form"));
    await page.locator("#service-security-mode").selectOption("monitor");
    await page.locator("#service-profile").selectOption("api");
    await page.locator("#service-ca-server").selectOption("custom");
    await page.locator('[data-wizard-tab="upstream"]').click();
    if (!(await page.locator("#service-use-reverse-proxy").isChecked())) await page.locator("#service-use-reverse-proxy").check();
    await page.locator("#service-upstream-scheme").selectOption("https");
    await page.locator("#service-upstream-host").fill("mtls-upstream");
    await page.locator("#service-upstream-port").fill("8443");
    await expect(page.locator("#service-health-check-path")).toBeDisabled();
    await page.locator("#service-health-check-enabled").check();
    await expect(page.locator("#service-health-check-path")).toBeEnabled();
    await page.locator("#service-health-check-path").fill("/e2e-health");
    await page.locator("#service-reverse-proxy-websocket").check();

    await page.locator('[data-wizard-tab="httpheaders"]').click();
    await page.locator("#service-use-cors").check();
    await addListValue("cors_allowed_origins", "https://dynamic.example.test");

    await page.locator('[data-wizard-tab="traffic"]').click();
    await page.locator("#service-use-limit-req").check();
    await page.locator("#service-limit-req-rate").fill("7");
    await page.locator("#service-limit-req-rate-unit").selectOption("r/m");
    await page.locator("#service-use-blacklist").check();
    await addListValue("blacklist_user_agent", "e2e-dynamic-agent");
    const userAgentEditor = page.locator('[data-list-field="blacklist_user_agent"]');
    await userAgentEditor.locator("details").first().locator("summary").click();
    const template = userAgentEditor.locator('[data-list-template-apply="blacklist_user_agent"]').first();
    await template.click();
    await expect(page.locator('[data-list-template-remove="blacklist_user_agent"]')).toHaveCount(1);
    await page.locator('[data-list-field="blacklist_user_agent"] .waf-list-selected-dropdown summary').click();
    await page.locator('[data-list-template-remove="blacklist_user_agent"]').click();
    await expect(page.locator('[data-list-template-remove="blacklist_user_agent"]')).toHaveCount(0);

    await page.locator('[data-wizard-tab="blocking"]').click();
    await page.locator("#service-ban-escalation-enabled").check();
    await page.locator("#service-ban-escalation-scope").selectOption("current_site");
    await page.locator("#service-ban-stage-input").fill("5m");
    await page.locator("[data-ban-stage-add]").click();
    await expect(page.locator("[data-ban-stage-remove]")).not.toHaveCount(0);

    await page.locator('[data-wizard-tab="antibot"]').click();
    await page.locator("#service-antibot-enabled").check();
    await page.locator("#service-antibot-challenge-template").selectOption("v5");
    await page.locator("#service-antibot-challenge").selectOption("javascript");
    await page.locator("#service-antibot-escalation-enabled").check();
    await page.locator("#service-antibot-escalation-mode").selectOption("cookie");
    await page.locator("#service-auth-mode").selectOption("basic_or_token");
    await page.locator("#service-auth-order").selectOption("antibot_first");
    await page.locator("#service-auth-basic-template").selectOption("v9");
    await page.locator("#service-auth-basic-session-ttl").selectOption("30");
    await page.locator("[data-antibot-rule-add]").click();
    await expect(page.locator("[data-antibot-rule-remove]")).toHaveCount(1);
    await page.locator("[data-antibot-rule-remove]").click();
    await expect(page.locator("[data-antibot-rule-remove]")).toHaveCount(0);

    await page.locator('[data-wizard-tab="geo"]').click();
    await page.locator("[data-geo-tw-add]").click();
    await expect(page.locator("[data-geo-tw-remove]")).toHaveCount(1);
    await page.locator("[data-geo-tw-remove]").click();
    await expect(page.locator("[data-geo-tw-remove]")).toHaveCount(0);

    await page.locator('[data-wizard-tab="modsec"]').click();
    await page.locator("[data-modsec-exclusion-add]").click();
    await expect(page.locator("[data-modsec-exclusion-remove]")).toHaveCount(1);
    await page.locator("[data-modsec-exclusion-remove]").click();
    await expect(page.locator("[data-modsec-exclusion-remove]")).toHaveCount(0);

    await page.locator('[data-wizard-tab="websocket"]').click();
    await expect(page.locator("#service-use-ws-inspection")).toBeEnabled();
    await page.locator("#service-use-ws-inspection").check();
    await addListValue("ws_block_patterns", "(?i)e2e-block");

    await page.locator('[data-wizard-tab="upstream"]').click();
    await expect(page.locator("#service-health-check-enabled")).toBeChecked();
    await expect(page.locator("#service-health-check-path")).toHaveValue("/e2e-health");
    await page.locator('[data-wizard-tab="websocket"]').click();
    await expect(page.locator("#service-use-ws-inspection")).toBeChecked();
    await expect(page.locator('[data-list-field="ws_block_patterns"]')).toContainText("(?i)e2e-block");

    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => {
      const persisted = JSON.parse((await api(`/api/easy-site-profiles/${siteID}`)).body);
      return Boolean(persisted?.updated_at) && persisted?.http_headers?.use_cors === true;
    }, { timeout: 60_000 }).toBe(true);
    const profileResult = await api(`/api/easy-site-profiles/${siteID}`);
    const profile = JSON.parse(profileResult.body);
    expect(profile?.upstream_routing, profileResult.body).toMatchObject({ health_check_enabled: true, health_check_path: "/e2e-health" });
    expect(profile?.http_headers?.use_cors, profileResult.body).toBe(true);
    expect(profile?.http_headers?.cors_allowed_origins, profileResult.body).toContain("https://dynamic.example.test");
    expect(profile?.security_behavior_and_limits?.blacklist_user_agent, profileResult.body).toContain("e2e-dynamic-agent");
    expect(profile?.security_behavior_and_limits?.ban_escalation_stages_seconds, profileResult.body).toContain(300);
    expect(profile?.security_websocket?.use_ws_inspection, profileResult.body).toBe(true);
    expect(profile?.security_websocket?.ws_block_patterns, profileResult.body).toContain("(?i)e2e-block");
    expect(profile?.front_service, profileResult.body).toMatchObject({ security_mode: "monitor", profile: "api", certificate_authority_server: "custom" });
    expect(profile?.security_behavior_and_limits, profileResult.body).toMatchObject({ limit_req_rate: "7r/m", ban_escalation_scope: "current_site" });
    expect(profile?.security_antibot, profileResult.body).toMatchObject({ antibot_challenge: "javascript", antibot_challenge_template: "v5", challenge_escalation_mode: "cookie" });
    expect(profile?.security_auth_basic, profileResult.body).toMatchObject({ auth_mode: "basic_or_token", auth_order: "antibot_first", auth_basic_template: "v9", session_inactivity_minutes: 30 });
    const upstreamResult = await api("/api/upstreams");
    expect(upstreamResult.status, upstreamResult.body).toBe(200);
    expect(JSON.parse(upstreamResult.body)).toEqual(expect.arrayContaining([expect.objectContaining({ id: upstreamID, scheme: "https", host: "mtls-upstream", port: 8443 })]));
    await openPage(page, `/services/${siteID}`, page.locator("#service-editor-form"));
    for (const [selector, value] of Object.entries({
      "#service-security-mode": "monitor", "#service-profile": "api", "#service-ca-server": "custom", "#service-upstream-scheme": "https",
      "#service-limit-req-rate-unit": "r/m", "#service-ban-escalation-scope": "current_site", "#service-antibot-challenge-template": "v5",
      "#service-antibot-challenge": "javascript", "#service-antibot-escalation-mode": "cookie", "#service-auth-mode": "basic_or_token",
      "#service-auth-order": "antibot_first", "#service-auth-basic-template": "v9", "#service-auth-basic-session-ttl": "30",
    })) await expect(page.locator(selector)).toHaveValue(value);
  } finally {
    await cleanup.run();
  }
});
