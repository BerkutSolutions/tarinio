import { expect, test } from "../fixtures/auth";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

async function status(page: import("@playwright/test").Page) {
  const result = await api(page, "/api/owasp-crs/status");
  expect(result.status, result.body).toBe(200);
  return JSON.parse(result.body);
}

test("owasp-crs.check-update-hourly-persistence-restore", async ({ authenticatedPage: page }) => {
  await page.goto("/owasp-crs", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("#owasp-crs-check")).toBeVisible({ timeout: 30_000 });
  const original = await status(page);
  try {
    await page.locator("#owasp-crs-check").click();
    await expect(page.locator(".waf-crs-console")).toContainText(/done|latest|version|готов|版本|Version/i, { timeout: 60_000 });
    await expect(page.locator("#owasp-crs-check")).toBeEnabled();

    const next = !Boolean(original.hourly_auto_update_enabled);
    await page.locator("#owasp-crs-hourly-auto").setChecked(next);
    await expect(page.locator("#owasp-crs-save-hourly")).toBeEnabled();
    await page.locator("#owasp-crs-save-hourly").click();
    await expect.poll(async () => Boolean((await status(page)).hourly_auto_update_enabled), { timeout: 30_000 }).toBe(next);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#owasp-crs-hourly-auto")).toBeChecked({ checked: next });

    await page.locator("#owasp-crs-update").click();
    await expect(page.locator("#owasp-crs-update")).toBeEnabled({ timeout: 120_000 });
    await expect(page.locator(".waf-crs-console")).not.toContainText(/failed|error|ошиб|Fehler/i);
    expect((await status(page)).active_version).toBeTruthy();
    const audit = await api(page, "/api/audit?limit=500");
    expect(audit.status, audit.body).toBe(200);
    expect(audit.body).toContain("owasp_crs.update");
    expect(audit.body).toContain("owasp_crs.hourly_auto_update");
  } finally {
    const restore = await api(page, "/api/owasp-crs/update", { method: "POST", body: JSON.stringify({ enable_hourly_auto_update: Boolean(original.hourly_auto_update_enabled) }) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("owasp-crs.api-validation-and-disabled-busy-state", async ({ authenticatedPage: page }) => {
  await page.goto("/owasp-crs", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const invalidDry = await api(page, "/api/owasp-crs/check-updates", { method: "POST", body: JSON.stringify({ dry_run: "yes" }) });
  expect(invalidDry.status, invalidDry.body).toBe(400);
  const invalidHourly = await api(page, "/api/owasp-crs/update", { method: "POST", body: JSON.stringify({ enable_hourly_auto_update: 1 }) });
  expect(invalidHourly.status, invalidHourly.body).toBe(400);

  const request = page.waitForResponse((response) => response.url().includes("/api/owasp-crs/check-updates"));
  await page.locator("#owasp-crs-check").click();
  await expect(page.locator("#owasp-crs-check")).toBeDisabled();
  await expect(page.locator("#owasp-crs-update")).toBeDisabled();
  expect((await request).status()).toBe(200);
  await expect(page.locator("#owasp-crs-check")).toBeEnabled({ timeout: 60_000 });
});
