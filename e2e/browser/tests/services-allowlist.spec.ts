import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.allowlist-easy-and-raw-persistence", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(4 * 60_000);
  page.setDefaultTimeout(20_000);
  const siteID = e2eID(testInfo, "e2e-allowlist");
  const upstreamID = siteID + "-upstream";
  const allowCIDR = "198.51.100.17/32";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const items = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : payload?.items || [];
  };
  const accessPolicy = async () => (await items("/api/access-policies"))
    .find((item: { site_id?: string }) => item.site_id === siteID);

  const cleanup = new CleanupLedger();
  cleanup.add("access policy", async () => {
    const policy = await accessPolicy();
    if (policy?.id) await api("/api/access-policies/" + encodeURIComponent(policy.id) + "?auto_apply=false", { method: "DELETE" });
  }, async () => !(await accessPolicy()));
  cleanup.add("profile", () => api("/api/easy-site-profiles/" + siteID + "?auto_apply=false", { method: "DELETE" }), async () => !(await items("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add("upstream", () => api("/api/upstreams/" + upstreamID + "?auto_apply=false", { method: "DELETE" }), async () => !(await items("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add("site", () => api("/api/sites/" + siteID + "?auto_apply=false", { method: "DELETE" }), async () => !(await items("/api/sites")).some((item: { id?: string }) => item.id === siteID));

  try {
    let result = await api("/api/sites?auto_apply=false", {
      method: "POST",
      body: JSON.stringify({ id: siteID, primary_host: siteID + ".test", enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }),
    });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", {
      method: "POST",
      body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }),
    });
    expect([200, 201], result.body).toContain(result.status);

    await openPage(page, "/services/" + siteID, "#service-editor-form");
    await page.locator('[data-wizard-tab="traffic"]').click();
    await page.locator("#service-use-allowlist").check();
    await page.locator("#list-input-access_allowlist").fill(allowCIDR);
    await page.locator('[data-list-add="access_allowlist"]').click();
    await expect(page.locator('[data-list-field="access_allowlist"]')).toContainText(allowCIDR);
    await page.locator("#service-editor-form button[type=submit]").click();

    await expect.poll(async () => (await accessPolicy())?.allowlist || [], { timeout: 90_000 }).toContain(allowCIDR);
    await openPage(page, "/services/" + siteID, "#service-editor-form");
    await page.locator('[data-wizard-tab="traffic"]').click();
    await expect(page.locator("#service-use-allowlist")).toBeChecked();
    await expect(page.locator('[data-list-field="access_allowlist"]')).toContainText(allowCIDR);

    await page.locator('[data-mode-tab="raw"]').click();
    await expect(page.locator("#service-raw-env")).toBeVisible();
    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => (await accessPolicy())?.allowlist || [], { timeout: 90_000 }).toContain(allowCIDR);

    await openPage(page, "/services/" + siteID, "#service-editor-form");
    await page.locator('[data-wizard-tab="traffic"]').click();
    await expect(page.locator("#service-use-allowlist")).toBeChecked();
    await expect(page.locator('[data-list-field="access_allowlist"]')).toContainText(allowCIDR);
  } finally {
    await cleanup.run();
  }
});