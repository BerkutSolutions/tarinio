import { expect, test } from "../fixtures/auth";
import { openPage } from "../support/waits";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

test("settings.storage-roundtrip-indexes-and-no-partial-save", async ({ authenticatedPage: page }) => {
  await page.goto("/settings/storage", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  const originalResult = await api(page, "/api/settings/runtime");
  expect(originalResult.status, originalResult.body).toBe(200);
  const original = JSON.parse(originalResult.body);
  const next = {
    logs_days: Number(original.storage.logs_days) === 13 ? 14 : 13,
    activity_days: Number(original.storage.activity_days) === 29 ? 30 : 29,
    events_days: Number(original.storage.events_days) === 28 ? 30 : 28,
    bans_days: Number(original.storage.bans_days) === 27 ? 30 : 27,
    hot_index_days: Number(original.storage.hot_index_days) === 20 ? 21 : 20,
    cold_index_days: Number(original.storage.cold_index_days) === 365 ? 366 : 365,
  };
  try {
    for (const [selector, value] of [["#settings-storage-logs", next.logs_days], ["#settings-storage-activity", next.activity_days], ["#settings-storage-events", next.events_days], ["#settings-storage-bans", next.bans_days], ["#settings-storage-hot-index-days", next.hot_index_days], ["#settings-storage-cold-index-days", next.cold_index_days]] as const) await page.locator(selector).fill(String(value));
    await page.locator("#settings-storage-save").click();
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/runtime")).body).storage).toEqual(expect.objectContaining(next));
    await openPage(page, "/settings/storage", page.locator("#settings-storage-logs"));
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    await expect(page.locator("#settings-storage-logs")).toHaveValue(String(next.logs_days));

    const beforeInvalid = JSON.parse((await api(page, "/api/settings/runtime")).body);
    const invalid = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ update_checks_enabled: !beforeInvalid.update_checks_enabled, storage: { logs_days: 0 } }) });
    expect(invalid.status, invalid.body).toBe(400);
    const afterInvalid = JSON.parse((await api(page, "/api/settings/runtime")).body);
    expect(afterInvalid.update_checks_enabled).toBe(beforeInvalid.update_checks_enabled);
    expect(afterInvalid.storage).toEqual(beforeInvalid.storage);

    for (const stream of ["requests", "events", "activity"]) {
      const indexes = await api(page, `/api/settings/runtime/storage-indexes?stream=${stream}&limit=10&offset=0`);
      expect(indexes.status, indexes.body).toBe(200);
      const payload = JSON.parse(indexes.body);
      expect(payload.stream, indexes.body).toBe(stream);
      expect(Array.isArray(payload.items)).toBe(true);
      expect(payload.limit).toBe(10);
    }
    const invalidDelete = await api(page, "/api/settings/runtime/storage-indexes?stream=requests&date=invalid", { method: "DELETE" });
    expect(invalidDelete.status, invalidDelete.body).toBe(400);
    await page.goto("/settings/logging", { waitUntil: "domcontentloaded" });
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    for (const stream of ["requests", "events", "activity"]) { await page.locator(`[data-storage-index-stream="${stream}"]`).click(); await expect(page.locator(`[data-storage-index-stream="${stream}"]`)).toHaveClass(/active/); }
  } finally {
    const restore = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ update_checks_enabled: original.update_checks_enabled, storage: original.storage }) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("settings.security-roundtrip-direct-ip-and-validation", async ({ authenticatedPage: page }) => {
  await page.goto("/settings/security", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  const original = JSON.parse((await api(page, "/api/settings/runtime")).body);
  const directOriginal = JSON.parse((await api(page, "/api/settings/direct-ip-access")).body);
  const next = {
    allow_insecure_vault_tls: !original.security.allow_insecure_vault_tls,
    require_certificate_export_approval: !original.security.require_certificate_export_approval,
    login_rate_limit_enabled: !original.security.login_rate_limit_enabled,
    login_rate_limit_max_attempts: Number(original.security.login_rate_limit_max_attempts) === 17 ? 18 : 17,
    login_rate_limit_window_seconds: 420,
    login_rate_limit_block_seconds: 840,
  };
  try {
    await page.locator("#settings-security-allow-insecure-vault-tls").setChecked(next.allow_insecure_vault_tls);
    await page.locator("#settings-security-require-certificate-export-approval").setChecked(next.require_certificate_export_approval);
    await page.locator("#settings-security-login-rate-enabled").setChecked(next.login_rate_limit_enabled);
    await page.locator("#settings-security-login-rate-attempts").fill(String(next.login_rate_limit_max_attempts));
    await page.locator("#settings-security-login-rate-window").fill(String(next.login_rate_limit_window_seconds));
    await page.locator("#settings-security-login-rate-block").fill(String(next.login_rate_limit_block_seconds));
    await page.locator("#settings-security-block-direct-ip-access").setChecked(Boolean(directOriginal.block_direct_ip_access));
    await page.locator("#settings-security-save").click();
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/runtime")).body).security).toEqual(expect.objectContaining(next));
    await openPage(page, "/settings/security", page.locator("#settings-security-login-rate-attempts"));
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    await expect(page.locator("#settings-security-login-rate-attempts")).toHaveValue(String(next.login_rate_limit_max_attempts));
    const invalid = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ security: { login_rate_limit_max_attempts: 1 } }) });
    expect(invalid.status, invalid.body).toBe(400);
    expect(JSON.parse((await api(page, "/api/settings/direct-ip-access")).body).block_direct_ip_access).toBe(directOriginal.block_direct_ip_access);
  } finally {
    expect((await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ security: original.security }) })).status).toBe(200);
    expect((await api(page, "/api/settings/direct-ip-access", { method: "PUT", body: JSON.stringify(directOriginal) })).status).toBe(200);
  }
});
