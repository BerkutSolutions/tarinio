import { expect, test } from "../fixtures/auth";
import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

test("activity.api-filter-pagination-and-critical-mutations", async ({ authenticatedPage: page }, testInfo) => {
  const suffix = `${testInfo.project.name}-${Date.now()}`.replace(/[^a-z0-9-]/gi, "-").toLowerCase();
  const siteID = `e2e-audit-${suffix}`;
  const userID = `e2e-audit-user-${suffix}`;
  const certificateID = `e2e-audit-cert-${suffix}`;
  const fixtureDir = resolve("test-results", `audit-certfixture-${suffix}`);
  const originalAntiDDoS = JSON.parse((await api(page, "/api/anti-ddos/settings")).body);
  await page.goto("/activity", { waitUntil: "domcontentloaded" });
  try {
    expect((await api(page, "/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: `${siteID}.test`, enabled: true, listen_http: true }) })).status).toBe(201);
    expect((await api(page, `/api/sites/${encodeURIComponent(siteID)}/ban?auto_apply=false`, { method: "POST", body: JSON.stringify({ ip: "203.0.113.211" }) })).status).toBe(200);
    expect((await api(page, `/api/sites/${encodeURIComponent(siteID)}/unban?auto_apply=false`, { method: "POST", body: JSON.stringify({ ip: "203.0.113.211" }) })).status).toBe(200);
    expect((await api(page, "/api/revisions/compile", { method: "POST", body: "{}" })).status).toBe(201);
    expect((await api(page, "/api/administration/users", { method: "POST", body: JSON.stringify({ id: userID, username: userID, email: `${userID}@example.test`, password: "E2e-Audit-1234!", role_ids: ["auditor"], is_active: true }) })).status).toBe(201);
    expect((await api(page, `/api/administration/users/${encodeURIComponent(userID)}`, { method: "DELETE" })).status).toBe(200);
    for (let i = 0; i < 26; i++) {
      const settings = { ...originalAntiDDoS, model_enabled: i % 2 === 0 ? !Boolean(originalAntiDDoS.model_enabled) : Boolean(originalAntiDDoS.model_enabled) };
      expect((await api(page, "/api/anti-ddos/settings?auto_apply=false", { method: "PUT", body: JSON.stringify(settings) })).status).toBe(200);
    }
    mkdirSync(fixtureDir, { recursive: true });
    const [certPath, keyPath] = execFileSync("go", ["run", "./support/certfixture", fixtureDir, certificateID], { cwd: resolve("."), encoding: "utf8", windowsHide: true }).trim().split(/\r?\n/);
    await page.goto("/tls", { waitUntil: "domcontentloaded" });
    await page.locator("#upload-certificate-id").fill(certificateID);
    await page.locator("#upload-common-name").fill(`${certificateID}.example.test`);
    await page.locator("#certificate-file").setInputFiles(certPath);
    await page.locator("#private-key-file").setInputFiles(keyPath);
    await page.locator("#certificate-upload-form button[type=submit]").click();
    await expect.poll(async () => {
      const payload = JSON.parse((await api(page, "/api/certificates")).body);
      const certificates = Array.isArray(payload) ? payload : Array.isArray(payload?.certificates) ? payload.certificates : payload?.items;
      return Array.isArray(certificates) && certificates.some((item: { id?: string }) => item.id === certificateID);
    }, { timeout: 30_000 }).toBe(true);

    const result = await api(page, "/api/audit?limit=500&offset=0");
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    expect(Array.isArray(payload.items)).toBe(true);
    expect(payload.limit).toBe(500);
    expect(payload.offset).toBe(0);
    expect(payload.total).toBeGreaterThanOrEqual(payload.items.length);
    const item = payload.items[0];
    for (const key of ["id", "action", "resource_type", "resource_id", "status", "occurred_at", "summary", "hash"]) expect(item).toHaveProperty(key);
    const query = new URLSearchParams({ action: item.action, actor_user_id: item.actor_user_id || "", resource_type: item.resource_type, resource_id: item.resource_id, site_id: item.site_id || "", status: item.status, from: item.occurred_at, to: item.occurred_at, limit: "1", offset: "0" });
    expect(JSON.parse((await api(page, `/api/audit?${query}`)).body).items).toEqual(expect.arrayContaining([expect.objectContaining({ id: item.id })]));
    const categorized = JSON.parse((await api(page, "/api/audit?category=config&limit=25&offset=0")).body);
    expect(categorized.total).toBeGreaterThan(25);
    expect(categorized.items.every((entry: { action: string }) => !entry.action.startsWith("auth.") && !entry.action.startsWith("revision."))).toBe(true);
    for (const path of ["/api/audit?from=bad", "/api/audit?status=bad", "/api/audit?category=bad", "/api/audit?offset=-1"]) expect((await api(page, path)).status).toBe(400);
    for (const action of ["accesspolicy.unban", "revision.compile_request", "antiddos.settings.upsert", "certificate.upload", "administration.user.delete"]) {
      const actionResult = await api(page, `/api/audit?action=${encodeURIComponent(action)}&limit=1&offset=0`);
      expect(actionResult.status, actionResult.body).toBe(200);
      expect(JSON.parse(actionResult.body).items, `missing real audit action ${action}`).toEqual(expect.arrayContaining([expect.objectContaining({ action })]));
    }
  } finally {
    expect((await api(page, "/api/anti-ddos/settings?auto_apply=false", { method: "PUT", body: JSON.stringify(originalAntiDDoS) })).status).toBe(200);
    expect((await api(page, `/api/certificates/${encodeURIComponent(certificateID)}`, { method: "DELETE" })).status).toBeLessThan(300);
    expect((await api(page, `/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" })).status).toBeLessThan(300);
  }
});

test("activity.presets-submit-pagination-keyboard", async ({ authenticatedPage: page }) => {
  await page.goto("/activity", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#audit-results")).toBeVisible();
  await page.locator("[data-preset=hour]").click();
  await expect(page.locator("#audit-from")).not.toHaveValue("");
  await expect(page.locator("#audit-to")).not.toHaveValue("");
  await page.locator("[data-preset=day]").click();
  await expect(page.locator("#audit-from")).not.toHaveValue("");
  await page.locator("[data-preset='']").click();
  await expect(page.locator("#audit-from")).toHaveValue("");
  await page.locator("#audit-category").selectOption("config");
  await page.locator("#audit-status").selectOption("succeeded");
  await page.locator("#audit-limit").selectOption("25");
  await page.locator("#audit-actor").focus();
  await page.locator("#audit-actor").press("Enter");
  await expect(page.locator("#audit-page-info")).not.toHaveText("-");
  const next = page.locator("#audit-next");
  await expect(next).toBeEnabled();
  const before = await page.locator("#audit-page-info").textContent();
  await next.click();
  await expect(page.locator("#audit-page-info")).not.toHaveText(before || "");
  await expect(page.locator("#audit-prev")).toBeEnabled();
  await page.locator("#audit-prev").click();
  await page.locator("#audit-reset").click();
  await expect(page.locator("#audit-category")).toHaveValue("");
  await expect(page.locator("#audit-status")).toHaveValue("");
});

test("activity.loading-empty-error-malformed", async ({ authenticatedPage: source }) => {
  for (const state of ["loading", "empty", "error", "malformed"] as const) {
    const page = await source.context().newPage();
    try {
      if (state === "loading") await page.route("**/api/audit**", async (route) => { await new Promise((resolve) => setTimeout(resolve, 600)); await route.fulfill({ status: 200, contentType: "application/json", body: '{"items":[],"total":0,"limit":50,"offset":0}' }); });
      if (state === "empty") await page.route("**/api/audit**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: '{"items":[],"total":0,"limit":50,"offset":0}' }));
      if (state === "error") await page.route("**/api/audit**", (route) => route.fulfill({ status: 503, contentType: "application/json", body: '{"error":"unavailable"}' }));
      if (state === "malformed") await page.route("**/api/audit**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: '{"items":"bad","total":"bad"}' }));
      await page.goto("/activity", { waitUntil: "domcontentloaded" });
      if (state === "loading") await expect(page.locator("#audit-results")).toContainText(/loading|загруз|laden|učit|加载/i);
      else if (state === "error" || state === "malformed") await expect(page.locator("#audit-results")).not.toBeEmpty();
      else await expect(page.locator("#audit-results")).toContainText(/no data|нет данных|данных пока нет|keine|nema|没有/i);
      await expect(page.locator("nav")).toBeVisible();
    } finally { await page.close(); }
  }
});
