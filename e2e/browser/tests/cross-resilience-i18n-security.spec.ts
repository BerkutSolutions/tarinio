import { expect, test } from "../fixtures/auth";
import { requiredE2EEnv } from "../support/env";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  for (let attempt = 0; attempt < 3; attempt += 1) {
    try {
      return await page.evaluate(async ({ requestPath, requestInit }) => {
        const response = await fetch(requestPath, { ...requestInit, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(requestInit.headers || {}) } });
        return { status: response.status, body: await response.text() };
      }, { requestPath: path, requestInit: init });
    } catch (error) {
      if (!String(error).includes("Execution context was destroyed") || attempt === 2) throw error;
      await page.waitForLoadState("domcontentloaded");
    }
  }
  throw new Error(`API request did not execute: ${path}`);
}

test("cross-resilience-429-403-5xx-malformed-and-slow-api", async ({ authenticatedPage: source }, testInfo) => {
  testInfo.annotations.push({
    type: "evidence-scope",
    description: "Injected responses below prove browser UI resilience only; live WAF evidence is asserted separately before interception.",
  });
  const liveEvents = await api(source, "/api/events?limit=1&offset=0");
  expect(liveEvents.status, liveEvents.body).toBe(200);
  expect(Array.isArray(JSON.parse(liveEvents.body)?.events)).toBe(true);
  const states = [
    { name: "429", status: 429, body: '{"error":"rate limited"}' },
    { name: "403", status: 403, body: '{"error":"forbidden"}' },
    { name: "500", status: 500, body: '{"error":"backend failed"}' },
    { name: "malformed", status: 200, body: '{"events":"invalid"}' },
    { name: "slow", status: 200, body: '{"events":[],"total":0}', delay: 700 },
  ];
  for (const state of states) {
    const page = await source.context().newPage();
    try {
      await page.route("**/api/events**", async (route) => {
        if (state.delay) await new Promise((resolve) => setTimeout(resolve, state.delay));
        await route.fulfill({ status: state.status, contentType: "application/json", body: state.body });
      });
      await page.goto("/events", { waitUntil: "domcontentloaded" });
      if (state.name === "slow") await expect(page.locator("#events-status")).toContainText(/loading|загруз|laden|učit|加载/i);
      await expect(page.locator("#events-status")).not.toBeEmpty();
      await expect(page.locator("nav")).toBeVisible();
      await expect(page.locator("#events-filters")).toBeVisible();
    } finally { await page.close(); }
  }
});

test("cross-i18n-all-locales-core-labels-errors-and-modals", async ({ authenticatedPage: page }) => {
  const original = JSON.parse((await api(page, "/api/settings/runtime")).body).language || "en";
  const localizedEventTitles: Record<string, string> = {
    en: "Event details",
    ru: "Детали события",
    de: "Veranstaltungsdetails",
    sr: "Детаљи догађаја",
    zh: "活动详情",
  };
  const readySelectors: Record<string, string> = {
    "/dashboard": "#dashboard-page",
    "/events": "#events-filters",
    "/activity": "#audit-results",
    "/settings/general": "#settings-language-save",
  };
  const staticScopeSelectors: Record<string, string> = {
    "/dashboard": ".dashboard-toolbar",
    "/events": "#events-filters",
    "/activity": "#audit-filters",
    "/settings/general": "#settings-page",
  };
  try {
    for (const language of ["en", "ru", "de", "sr", "zh"]) {
      const update = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ language }) });
      expect(update.status, update.body).toBe(200);
      for (const path of ["/dashboard", "/events", "/activity", "/settings/general"]) {
        await page.goto(path, { waitUntil: "domcontentloaded" });
        await expect(page.locator(readySelectors[path]), `${language} ${path} must finish rendering`).toBeVisible();
        if (path === "/settings/general") await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
        await expect(page.locator("nav")).toBeVisible();
        const visibleText = `${await page.locator("nav").innerText()}\n${await page.locator(staticScopeSelectors[path]).innerText()}`;
        expect(visibleText, `${language} ${path}`).not.toMatch(/\b(?:app|common|events|activity|settings)\.[a-z][a-z0-9_.-]+\b/);
      }
      await page.goto("/events", { waitUntil: "domcontentloaded" });
      const row = page.locator("[data-event-row]").first();
      await expect(row, `${language} requires a real Events row`).toBeVisible();
      await row.click();
      const modal = page.locator("#events-detail-modal");
      await expect(modal).toBeVisible();
      await expect(page.locator("#events-detail-title")).toHaveText(localizedEventTitles[language]);
      await expect(page.locator("#events-detail-content tr")).toHaveCount(11);
      await modal.press("Escape");
      await expect(modal).toBeHidden();
    }
  } finally {
    const restore = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ language: original }) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("cross-security-headers-cookies-and-no-secret-leak", async ({ authenticatedPage: page, browser }) => {
  const consoleMessages: string[] = [];
  const anonymous = await browser.newContext({ ignoreHTTPSErrors: true });
  const loginPage = await anonymous.newPage();
  try {
    const response = await loginPage.goto("/login", { waitUntil: "domcontentloaded" });
    expect(response).not.toBeNull();
    const headers = response!.headers();
    expect(headers["x-content-type-options"]?.toLowerCase()).toBe("nosniff");
    expect(headers["x-frame-options"]?.toUpperCase()).toMatch(/DENY|SAMEORIGIN/);
    expect(headers["content-security-policy"]).toContain("default-src 'self'");
    expect(headers["referrer-policy"]).toBeTruthy();
    const cookies = await page.context().cookies();
    const sessions = cookies.filter((cookie) => cookie.name.startsWith("waf_session"));
    expect(sessions.length).toBeGreaterThan(0);
    for (const cookie of sessions) {
      expect(cookie.httpOnly, cookie.name).toBe(true);
      expect(cookie.secure, cookie.name).toBe(true);
      expect(cookie.sameSite, cookie.name).toBe("Strict");
    }
    page.on("console", (message) => consoleMessages.push(message.text()));
    await page.goto("/settings/secrets", { waitUntil: "domcontentloaded" });
    await page.goto("/tls", { waitUntil: "domcontentloaded" });
    const body = await page.locator("body").innerText();
    const logs = consoleMessages.join("\n");
    for (const secret of [requiredE2EEnv("WAF_E2E_PASSWORD"), ...sessions.map((cookie) => cookie.value)]) {
      expect(body).not.toContain(secret);
      expect(logs).not.toContain(secret);
    }
  } finally {
    await anonymous.close();
  }
});
