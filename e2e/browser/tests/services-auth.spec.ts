import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage, waitForAPIResponse } from "../support/waits";

test("services.basic-auth-mask-reveal-preview", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(3 * 60_000);
  page.setDefaultTimeout(15_000);
  const siteID = e2eID(testInfo, "e2e-auth-ui");
  const upstreamID = siteID + "-upstream";
  const username = "browser-user";
  const password = "Browser-secret-123!";
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
    return Array.isArray(payload) ? payload : payload?.items || [];
  };
  const profile = async () => JSON.parse((await api(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`)).body);
  const cleanup = new CleanupLedger();
  cleanup.add("site " + siteID, () => api(`/api/sites/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  cleanup.add("upstream " + upstreamID, () => api(`/api/upstreams/${upstreamID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add("profile " + siteID, () => api(`/api/easy-site-profiles/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));

  try {
    await test.step("seed site and upstream", async () => {
      let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: siteID + ".test", enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }) });
      expect([200, 201], result.body).toContain(result.status);
      result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
      expect([200, 201], result.body).toContain(result.status);
    });

    await test.step("edit Basic Auth users and settings", async () => {
      await openPage(page, `/services/${siteID}`, "#service-editor-form");
      await page.locator('[data-wizard-tab="antibot"]').click();
      await expect(page.locator('[data-tab-panel="antibot"]')).toBeVisible();
      await page.locator("#service-antibot-enabled").check();
      await page.locator("#service-antibot-challenge").selectOption("cookie");
      await page.locator("#service-antibot-session-ttl").selectOption("15");
      await page.locator("#service-use-auth-basic").check();
      await page.locator("#service-auth-order").selectOption("antibot_first");
      await page.locator("#service-auth-basic-template").selectOption("v6");
      await page.locator("#service-auth-basic-session-ttl").selectOption("5");
      await page.locator('[data-auth-user-username="0"]').fill(username);
      await page.locator('[data-auth-user-password="0"]').fill(password);
      await page.locator('[data-auth-user-enabled="0"]').check();
      await page.locator("[data-auth-user-add]").click();
      await expect(page.locator("[data-auth-user-username]")).toHaveCount(2);
      await page.locator('[data-auth-user-remove="1"]').click();
      await expect(page.locator("[data-auth-user-username]")).toHaveCount(1);
    });

    await test.step("preview template v6", async () => {
      const popupPromise = page.waitForEvent("popup", { timeout: 15_000 });
      await page.locator("#auth-basic-template-preview-btn").click();
      const popup = await popupPromise;
      await expect.poll(() => new URL(popup.url()).pathname, { timeout: 15_000 }).toBe("/api/error-pages/preview/auth-v6");
      await expect(popup.locator("body")).toHaveClass(/v6/);
      await popup.close();
    });

    await test.step("save and read back profile", async () => {
      await page.locator("#service-editor-form button[type=submit]").click();
      await expect.poll(async () => {
        const auth = (await profile())?.security_auth_basic;
        return auth?.use_auth_basic === true && auth?.auth_order === "antibot_first" && auth?.auth_basic_template === "v6" && auth?.session_inactivity_minutes === 5 && auth?.users?.[0]?.username === username && auth?.users?.[0]?.password_length === password.length;
      }, { timeout: 90_000 }).toBe(true);
    });

    await test.step("reload mask, reveal and hide", async () => {
      await openPage(page, `/services/${siteID}`, "#service-editor-form");
      await page.locator('[data-wizard-tab="antibot"]').click();
      await expect(page.locator("#service-antibot-session-ttl")).toHaveValue("15");
      await expect(page.locator("#service-auth-order")).toHaveValue("antibot_first");
      const passwordInput = page.locator('[data-auth-user-password="0"]');
      await expect(passwordInput).toHaveAttribute("data-auth-user-password-stored", "true");
      expect((await passwordInput.getAttribute("placeholder"))?.length).toBe(password.length);
      const revealResponsePromise = waitForAPIResponse(page, "POST", `/api/easy-site-profiles/${siteID}/auth-password/reveal`);
      await page.locator('[data-auth-user-toggle="0"]').click();
      const revealResponse = await revealResponsePromise;
      expect(revealResponse.status(), await revealResponse.text()).toBe(200);
      await expect(passwordInput).toHaveAttribute("type", "text");
      await expect(passwordInput).toHaveValue(password);
      await page.locator('[data-auth-user-toggle="0"]').click();
      await expect(passwordInput).toHaveAttribute("type", "password");
      const audit = await api("/api/audit?limit=500");
      expect(audit.status, audit.body).toBe(200);
      const auditItems = JSON.parse(audit.body)?.items || [];
      expect(auditItems).toEqual(expect.arrayContaining([expect.objectContaining({
        action: "easysiteprofile.auth_password.reveal",
        resource_id: siteID,
        status: "succeeded",
      })]));
    });
  } finally {
    await cleanup.run();
  }
});
