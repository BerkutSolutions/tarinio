import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { runtimeBaseURL } from "../support/env";

test("services.enable-disable-runtime", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-toggle");
  const upstreamID = siteID + "-upstream";
  const host = siteID + ".test";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const readSite = async () => {
    const response = await api("/api/sites");
    expect(response.status).toBe(200);
    const payload = JSON.parse(response.body);
    const sites = Array.isArray(payload) ? payload : payload?.items || [];
    return sites.find((site: { id?: string }) => site.id === siteID) || null;
  };
  const runtimeResponse = async () => {
    const response = await page.context().request.get(`${runtimeBaseURL()}/`, { headers: { Host: host }, failOnStatusCode: false, timeout: 30_000 });
    return {
      status: response.status(),
      body: await response.text(),
      contentType: response.headers()["content-type"] || "",
      revision: response.headers()["x-waf-runtime-revision"] || "",
    };
  };
  const cleanup = new CleanupLedger();
  cleanup.add("easy profile " + siteID, () => api("/api/easy-site-profiles/" + siteID + "?auto_apply=false", { method: "DELETE" }), async () => {
    const result = await api("/api/easy-site-profiles");
    return !JSON.parse(result.body).some((item: { site_id?: string }) => item.site_id === siteID);
  });
  cleanup.add("upstream " + upstreamID, () => api("/api/upstreams/" + upstreamID + "?auto_apply=false", { method: "DELETE" }), async () => {
    const result = await api("/api/upstreams");
    return !JSON.parse(result.body).some((item: { id?: string }) => item.id === upstreamID);
  });
  cleanup.add("site " + siteID, () => api("/api/sites/" + siteID + "?auto_apply=false", { method: "DELETE" }), async () => (await readSite()) === null);
  try {
    let response = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: host, enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201]).toContain(response.status);
    response = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/" }) });
    expect([200, 201]).toContain(response.status);

    await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await page.locator("#services-search").fill(siteID);
    const toggle = page.locator(`[data-toggle-site="${siteID}"]`);
    await expect(toggle).toBeVisible({ timeout: 30_000 });
    await toggle.click();
    await expect.poll(async () => Boolean((await readSite())?.enabled), { timeout: 120_000 }).toBe(false);
    await expect(page.locator(`[data-toggle-site="${siteID}"]`)).toHaveAttribute("data-toggle-enabled", "0", { timeout: 120_000 });
    await expect.poll(async () => {
      const runtime = await runtimeResponse();
      return {
        status: runtime.status,
        contentType: runtime.contentType,
        hasRevision: runtime.revision !== "",
        isUnknownHostPage: /<title>421 Misdirected Request<\/title>/i.test(runtime.body),
        reachedUpstream: /upstream-echo|"path"|"headers"/i.test(runtime.body),
      };
    }, { timeout: 120_000 }).toEqual({
      status: 200,
      contentType: "text/html",
      hasRevision: true,
      isUnknownHostPage: true,
      reachedUpstream: false,
    });

    await page.locator(`[data-toggle-site="${siteID}"]`).click();
    await expect.poll(async () => Boolean((await readSite())?.enabled), { timeout: 120_000 }).toBe(true);
    await expect(page.locator(`[data-toggle-site="${siteID}"]`)).toHaveAttribute("data-toggle-enabled", "1", { timeout: 120_000 });
    await expect.poll(async () => {
      const runtime = await runtimeResponse();
      return runtime.status === 200 && /upstream-echo|request|headers/i.test(runtime.body);
    }, { timeout: 120_000 }).toBe(true);
  } finally {
    await cleanup.run();
  }
});

test("services.list-resilience", async ({ authenticatedPage: page }) => {
  const emptyPage = await page.context().newPage();
  try {
    for (const endpoint of ["sites", "upstreams", "tls-configs", "access-policies"]) {
      await emptyPage.route(`**/api/${endpoint}**`, (route) => route.fulfill({ status: 200, contentType: "application/json", body: "[]" }));
    }
    await emptyPage.goto("/services?e2e-state=empty", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(emptyPage.locator(".waf-table .waf-empty").filter({ hasText: /no services|нет сервис/i })).toBeVisible({ timeout: 30_000 });
  } finally { await emptyPage.close(); }

  const errorPage = await page.context().newPage();
  try {
    await errorPage.route("**/api/sites**", (route) => route.fulfill({ status: 503, contentType: "application/json", body: '{"error":"synthetic services outage"}' }));
    await errorPage.goto("/services?e2e-state=error", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(errorPage.locator(".alert").last()).toContainText("synthetic services outage", { timeout: 30_000 });
  } finally { await errorPage.close(); }
});
