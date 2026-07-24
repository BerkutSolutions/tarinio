import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { runtimeBaseURL } from "../support/env";
import { openPage } from "../support/waits";

test("services.custom-error-pages-controls-preview-runtime", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(5 * 60_000);
  page.setDefaultTimeout(15_000);
  const siteID = e2eID(testInfo, "e2e-errors");
  const upstreamID = `${siteID}-upstream`;
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 60_000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : (payload?.items || []);
  };
  const activeRevisionID = async () => {
    const result = await api("/api/revisions");
    expect(result.status, result.body).toBe(200);
    return String((JSON.parse(result.body)?.revisions || []).find((item: { is_active?: boolean }) => item.is_active)?.id || "");
  };
  const runtime = async (path: string) => {
    const response = await page.context().request.get(`${runtimeBaseURL()}${path}`, { headers: { Host: `${siteID}.test` }, failOnStatusCode: false, timeout: 30_000 });
    return { status: response.status(), body: await response.text(), serverTiming: response.headers()["server-timing"] || "" };
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
    await page.locator("#service-security-mode").selectOption("block");
    await page.locator('[data-wizard-tab="modsec"]').click();
    await page.locator("#service-use-modsecurity").check();
    await page.locator("#service-use-modsecurity-crs-plugins").uncheck();
    await page.locator("#service-use-modsecurity-custom-configuration").check();
    await page.locator("#service-modsecurity-custom-path").fill("modsec/e2e-browser-errors.conf");
    await page.locator("#service-modsecurity-custom-content").fill([
      'SecRule REQUEST_URI "@rx ^/e2e-error-404$" "id:100041,phase:2,deny,status:404,log"',
      'SecRule REQUEST_URI "@rx ^/e2e-error-500$" "id:100042,phase:2,deny,status:500,log"',
    ].join("\n"));
    await page.locator('[data-wizard-tab="errorpages"]').click();
    const checkboxes = page.locator(".waf-ep-page-cb");
    const pageCount = await checkboxes.count();
    expect(pageCount).toBeGreaterThan(40);
    await page.locator("#ep-disable-all").click();
    await expect(page.locator(".waf-ep-page-cb:checked")).toHaveCount(0);
    await page.locator("#ep-enable-all").click();
    await expect(page.locator(".waf-ep-page-cb:checked")).toHaveCount(pageCount);
    await page.locator('[data-ep-slug="404"]').uncheck();

    const popupPromise = page.waitForEvent("popup", { timeout: 15_000 });
    await page.locator('[data-error-page-slug="404"]').click();
    const popup = await popupPromise;
    await expect.poll(() => new URL(popup.url()).pathname).toBe("/api/error-pages/preview/404");
    await expect(popup.locator("body")).not.toBeEmpty();
    await popup.close();

    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => {
      const readback = await api(`/api/easy-site-profiles/${siteID}`);
      if (readback.status !== 200) return false;
      const profile = JSON.parse(readback.body);
      return profile.use_custom_error_pages === true && Array.isArray(profile.disabled_error_pages) && profile.disabled_error_pages.includes("404");
    }, { timeout: 90_000 }).toBe(true);

    const compiled = await api("/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [siteID] }) });
    expect(compiled.status, compiled.body).toBe(201);
    const revisionID = String(JSON.parse(compiled.body)?.revision?.id || "");
    const applied = await api(`/api/revisions/${revisionID}/apply`, { method: "POST", body: "{}" });
    expect(applied.status, applied.body).toBe(201);
    await expect.poll(activeRevisionID, { timeout: 120_000 }).toBe(revisionID);
    const disabled404 = await runtime("/e2e-error-404");
    expect(disabled404.status).toBe(404);
    expect(disabled404.body.toLowerCase()).not.toContain("<!doctype html");
    const branded500 = await runtime("/e2e-error-500");
    expect(branded500.status).toBe(500);
    expect(branded500.body.toLowerCase()).toMatch(/<!doctype html|<html/);
    expect(branded500.body.length).toBeGreaterThan(200);
    expect(branded500.serverTiming).toContain("rid");
  } finally {
    await cleanup.run();
  }
});
