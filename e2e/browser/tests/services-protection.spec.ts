import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { runtimeBaseURL } from "../support/env";

test("services.rate-limit-modsecurity-runtime", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-protection");
  const upstreamID = siteID + "-upstream";
  const host = siteID + ".test";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30_000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
  };
  const readProfile = async () => {
    const result = await api(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`);
    return result.status === 200 ? JSON.parse(result.body) : null;
  };
  const runtimeStatus = async (path: string) => {
    const response = await page.context().request.get(`${runtimeBaseURL()}${path}`, { headers: { Host: host }, failOnStatusCode: false, timeout: 30_000 });
    return response.status();
  };
  const activeRevisionID = async () => {
    const result = await api("/api/revisions");
    expect(result.status, result.body).toBe(200);
    const revisions = JSON.parse(result.body)?.revisions || [];
    return String(revisions.find((item: { is_active?: boolean; id?: string }) => item.is_active)?.id || "");
  };
  const cleanup = new CleanupLedger();
  cleanup.add("site " + siteID, () => api(`/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  cleanup.add("upstream " + upstreamID, () => api(`/api/upstreams/${encodeURIComponent(upstreamID)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add("profile " + siteID, () => api(`/api/easy-site-profiles/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));

  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: host, enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
    expect([200, 201], result.body).toContain(result.status);

    await page.goto(`/services/${encodeURIComponent(siteID)}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(page.locator("#service-editor-form")).toBeVisible({ timeout: 30_000 });
    await page.locator("#service-security-mode").selectOption("block");

    await page.locator('[data-wizard-tab="traffic"]').click();
    await expect(page.locator('[data-tab-panel="traffic"]')).toBeVisible();
    await page.locator("#service-use-limit-req").check();
    await page.locator("#service-limit-req-url").fill("/");
    await page.locator("#service-limit-req-rate").fill("1");
    await page.locator("#service-limit-req-rate-unit").selectOption("r/s");

    await page.locator('[data-wizard-tab="modsec"]').click();
    await expect(page.locator('[data-tab-panel="modsec"]')).toBeVisible();
    await page.locator("#service-use-modsecurity").check();
    await page.locator("#service-use-modsecurity-crs-plugins").uncheck();
    await page.locator("#service-use-modsecurity-custom-configuration").check();
    await page.locator("#service-modsecurity-custom-path").fill("modsec/e2e-browser-protection.conf");
    await page.locator("#service-modsecurity-custom-content").fill('SecRule REQUEST_URI "@rx ^/e2e-browser-block$" "id:100031,phase:2,deny,status:403,log"');
    await page.locator("#service-editor-form button[type=submit]").click();

    await expect.poll(async () => {
      const profile = await readProfile();
      return profile?.security_behavior_and_limits?.use_limit_req === true &&
        profile?.security_behavior_and_limits?.limit_req_rate === "1r/s" &&
        profile?.security_modsecurity?.use_modsecurity === true &&
        String(profile?.security_modsecurity?.custom_configuration?.content || "").includes("100031");
    }, { timeout: 120_000 }).toBe(true);

    const compiled = await api("/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [siteID] }) });
    expect(compiled.status, compiled.body).toBe(201);
    const revisionID = String(JSON.parse(compiled.body)?.revision?.id || "");
    expect(revisionID).toBeTruthy();
    const applied = await api(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, { method: "POST", body: "{}" });
    expect(applied.status, applied.body).toBe(201);
    await expect.poll(activeRevisionID, { timeout: 120_000 }).toBe(revisionID);

    await expect.poll(() => runtimeStatus("/e2e-browser-block"), { timeout: 120_000 }).toBe(403);
    const statuses: number[] = [];
    for (let index = 0; index < 30 && !statuses.includes(429); index += 1) statuses.push(await runtimeStatus("/"));
    expect(statuses, `runtime did not enforce request limit: ${statuses.join(",")}`).toContain(429);
  } finally {
    await cleanup.run();
  }
});
