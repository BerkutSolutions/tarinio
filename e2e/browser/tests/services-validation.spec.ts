import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.validation-reference-matrix", async ({ authenticatedPage: page }, testInfo) => {
  const prefix = e2eID(testInfo, "e2e-validation");
  const siteA = prefix + "-a";
  const siteB = prefix + "-b";
  const upstreamA = siteA + "-upstream";
  const uiSite = prefix + "-ui";
  const uiUpstream = uiSite + "-upstream";
  const hostA = siteA + ".example.test";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30_000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally {
      window.clearTimeout(timer);
    }
  }, { path, init });
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
  };
  const cleanup = new CleanupLedger();
  cleanup.add("UI upstream " + uiUpstream, () => api(`/api/upstreams/${encodeURIComponent(uiUpstream)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === uiUpstream));
  cleanup.add("UI site " + uiSite, () => api(`/api/sites/${encodeURIComponent(uiSite)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === uiSite));
  cleanup.add("upstream " + upstreamA, () => api(`/api/upstreams/${encodeURIComponent(upstreamA)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamA));
  for (const id of [siteA, siteB]) cleanup.add("site " + id, () => api(`/api/sites/${encodeURIComponent(id)}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === id));

  const reject = async (path: string, payload: object, method = "POST") => {
    const result = await api(path, { method, body: JSON.stringify(payload) });
    expect(result.status, result.body).toBe(400);
    expect(JSON.parse(result.body)?.error).toBeTruthy();
  };
  const validUpstream = { id: upstreamA, site_id: siteA, host: "upstream-echo", port: 8888, scheme: "http" };

  try {
    await test.step("site API validation", async () => {
      await reject("/api/sites?auto_apply=false", { primary_host: hostA, enabled: true });
      await reject("/api/sites?auto_apply=false", { id: siteA, primary_host: "", enabled: true });
      let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteA, primary_host: hostA, enabled: true }) });
      expect([200, 201], result.body).toContain(result.status);
      await reject("/api/sites?auto_apply=false", { id: siteB, primary_host: hostA.toUpperCase(), enabled: true });
      result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteB, primary_host: siteB + ".example.test", enabled: true }) });
      expect([200, 201], result.body).toContain(result.status);
    });

    await test.step("upstream API validation and ownership", async () => {
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, id: "" });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, site_id: "" });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, site_id: prefix + "-missing" });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, host: "" });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, port: 0 });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, port: 65536 });
      await reject("/api/upstreams?auto_apply=false", { ...validUpstream, scheme: "ftp" });
      const result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify(validUpstream) });
      expect([200, 201], result.body).toContain(result.status);
      await reject(`/api/upstreams/${encodeURIComponent(upstreamA)}?auto_apply=false`, { ...validUpstream, site_id: siteB }, "PUT");
      expect((await list("/api/upstreams")).find((item: { id?: string }) => item.id === upstreamA)?.site_id).toBe(siteA);
    });

    const uiCases: Array<[string, string, string]> = [
      ["site id", "#service-id", ""],
      ["primary host", "#service-host", ""],
      ["upstream host", "#service-upstream-host", ""],
      ["port lower bound", "#service-upstream-port", "0"],
      ["port upper bound", "#service-upstream-port", "65536"],
      ["scheme", "#service-upstream-scheme", "ftp"],
    ];
    for (const [label, selector, value] of uiCases) {
      await test.step("UI validation: " + label, async () => {
        await openPage(page, "/services/new", page.locator("#service-editor-form"));
        await page.locator("#service-id").fill(uiSite);
        await page.locator("#service-host").fill(uiSite + ".example.test");
        await page.locator('[data-wizard-tab="upstream"]').click();
        await expect(page.locator('[data-tab-panel="upstream"]')).toBeVisible();
        await page.locator("#service-upstream-host").fill("upstream-echo");
        await page.locator("#service-upstream-port").fill("8888");
        await page.locator("#service-editor-form").evaluate((form, invalid) => {
          const input = form.querySelector(invalid.selector) as HTMLInputElement | HTMLSelectElement | null;
          if (!input) throw new Error(`missing validation input ${invalid.selector}`);
          if (input instanceof HTMLSelectElement && !Array.from(input.options).some((option) => option.value === invalid.value)) {
            input.add(new Option(invalid.value, invalid.value));
          }
          input.value = invalid.value;
          form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
        }, { selector, value });
        await expect(page.locator("#sites-feedback"), label).not.toBeEmpty({ timeout: 10000 });
        expect((await list("/api/sites")).some((item: { id?: string }) => item.id === uiSite), label).toBe(false);
      });
    }
  } finally {
    await cleanup.run();
  }
});
