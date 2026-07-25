import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.antibot-template-preview-and-readback", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(4 * 60_000);
  const siteID = e2eID(testInfo, "e2e-antibot-preview");
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
  const cleanup = new CleanupLedger();
  cleanup.add(`profile ${siteID}`, () => api(`/api/easy-site-profiles/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add(`upstream ${upstreamID}`, () => api(`/api/upstreams/${upstreamID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add(`site ${siteID}`, () => api(`/api/sites/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: `${siteID}.test`, enabled: true, listen_http: true, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
    expect([200, 201], result.body).toContain(result.status);

    await openPage(page, `/services/${siteID}`, "#service-editor-form [data-wizard-tab=\"antibot\"]");
    await expect(page.locator('[data-wizard-tab="antibot"]')).toBeEnabled();
    await page.locator('[data-wizard-tab="antibot"]').click();
    await page.locator("#service-antibot-enabled").check();
    await page.locator("#service-antibot-challenge-template").selectOption("v4");

    await page.locator("#service-antibot-challenge").selectOption("cookie");
    await page.locator("#antibot-template-preview-btn").click();
    await expect(page.locator("#app-toast-container .app-toast").last()).toBeVisible();

    await page.locator("#service-antibot-challenge").selectOption("javascript");
    let popupPromise = page.waitForEvent("popup");
    await page.locator("#antibot-template-preview-btn").click();
    let popup = await popupPromise;
    await expect.poll(() => new URL(popup.url()).pathname).toBe("/api/error-pages/preview/antibot-v4");
    await popup.close();

    await page.locator("#service-antibot-challenge").selectOption("captcha");
    popupPromise = page.waitForEvent("popup");
    await page.locator("#antibot-template-preview-btn").click();
    popup = await popupPromise;
    await expect.poll(() => new URL(popup.url()).pathname).toBe("/api/error-pages/preview/captcha-v4");
    await popup.close();

    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => {
      const profileResult = await api(`/api/easy-site-profiles/${siteID}`);
      if (profileResult.status !== 200) return false;
      const profile = JSON.parse(profileResult.body)?.security_antibot;
      return profile?.antibot_challenge === "captcha" && profile?.antibot_challenge_template === "v4";
    }, { timeout: 120_000 }).toBe(true);
  } finally {
    await cleanup.run();
  }
});
