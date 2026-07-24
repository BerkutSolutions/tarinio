import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.virtual-patches-api-readback", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(3 * 60_000);
  const siteID = e2eID(testInfo, "e2e-vp-ui");
  const upstreamID = `${siteID}-upstream`;
  const pattern = "^/e2e-vp-ui$";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    return JSON.parse(result.body);
  };
  const cleanup = new CleanupLedger();
  cleanup.add("site " + siteID, () => api(`/api/sites/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  cleanup.add("upstream " + upstreamID, () => api(`/api/upstreams/${upstreamID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));

  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: `${siteID}.test`, enabled: true, listen_http: true, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
    expect([200, 201], result.body).toContain(result.status);

    await openPage(page, `/services/${siteID}`, page.locator("#service-editor-form"));
    await page.locator('[data-wizard-tab="virtualpatches"]').click();
    await page.locator("#vp-pattern").fill(pattern);
    await page.locator("#vp-target").selectOption("uri");
    await page.locator("#vp-action").selectOption("block");
    await page.locator("#vp-add-btn").click();
    await expect.poll(async () => {
      const patches = await list(`/api/virtual-patches/${siteID}`);
      return Array.isArray(patches) && patches.some((patch: { pattern?: string; target?: string; action?: string }) => patch.pattern === pattern && patch.target === "uri" && patch.action === "block");
    }).toBe(true);
    await openPage(page, `/services/${siteID}`, page.locator("#service-editor-form"));
    await page.locator('[data-wizard-tab="virtualpatches"]').click();
    await expect(page.locator("#virtual-patches-list")).toContainText(pattern);
    await page.locator("[data-vp-delete]").click();
    await expect.poll(async () => (await list(`/api/virtual-patches/${siteID}`)).length).toBe(0);
  } finally {
    await cleanup.run();
  }
});
