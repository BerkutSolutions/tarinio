import { expect, test } from "../fixtures/auth";
import { requiredE2EEnv } from "../support/env";
import { openPage } from "../support/waits";

const readySettingsPage = (page: import("@playwright/test").Page) => page.locator("#settings-page[data-runtime-ready=\"true\"]");

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 4; attempt += 1) {
    try {
      return await page.evaluate(async ({ path, init }) => {
        const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
        return { status: response.status, body: await response.text() };
      }, { path, init });
    } catch (error) {
      lastError = error;
      if (!/Failed to fetch|Execution context was destroyed/i.test(String(error)) || attempt === 3) throw error;
      await page.waitForTimeout(250 * (attempt + 1));
    }
  }
  throw lastError;
}

test("settings.logging-backends-routing-migration-roundtrip", async ({ authenticatedPage: page }) => {
  await openPage(page, "/settings/logging", readySettingsPage(page));
  const original = JSON.parse((await api(page, "/api/settings/runtime")).body);
  try {
    await page.locator("#settings-logging-hot-backend").selectOption("file");
    await page.locator("#settings-logging-cold-backend").selectOption("file");
    const routes: Array<[string, boolean]> = [["#settings-logging-route-requests-hot", true], ["#settings-logging-route-requests-cold", false], ["#settings-logging-route-events-hot", true], ["#settings-logging-route-events-cold", false], ["#settings-logging-route-activity-hot", true], ["#settings-logging-route-activity-cold", false], ["#settings-logging-route-fallback", true]];
    for (const [selector, checked] of routes) await page.locator(selector).setChecked(checked);
    await page.locator("#settings-logging-migration-enabled").setChecked(!Boolean(original.logging?.clickhouse?.migration_enabled));
    await page.locator("#settings-logging-save").click();
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/runtime")).body).logging.hot.backend).toBe("file");
    const saved = JSON.parse((await api(page, "/api/settings/runtime")).body).logging;
    expect(saved.cold.backend).toBe("file");
    expect(saved.routing).toEqual(expect.objectContaining({ write_requests_to_hot: true, write_requests_to_cold: false, write_events_to_hot: true, write_events_to_cold: false, write_activity_to_hot: true, write_activity_to_cold: false, keep_local_fallback: true }));
    await openPage(page, "/settings/logging", readySettingsPage(page));
    await expect(page.locator("#settings-logging-hot-backend")).toHaveValue("file");
    await expect(page.locator("#settings-logging-route-events-hot")).toBeChecked();
  } finally {
    const restore = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ logging: original.logging }) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("settings.secrets-masked-save-and-toggle", async ({ authenticatedPage: page }) => {
  await openPage(page, "/settings/secrets", readySettingsPage(page));
  const original = JSON.parse((await api(page, "/api/settings/runtime")).body);
  const verifyToggles = async (secretFields: ReadonlyArray<readonly [string, string]>) => {
    for (const [field, toggle] of secretFields) {
      const input = page.locator(field);
      expect(await input.inputValue()).not.toContain("password");
      await page.locator(toggle).click();
      await expect(input).toHaveAttribute("type", "text");
      expect(await input.inputValue()).toBe("");
      await page.locator(toggle).click();
      await expect(input).toHaveAttribute("type", "password");
    }
  };
  await verifyToggles([["#settings-logging-vault-token", "#settings-logging-vault-token-toggle"]]);
  await page.locator("#settings-secrets-save").click();
  await expect(page.locator("#settings-alert")).not.toBeEmpty();
  const response = JSON.parse((await api(page, "/api/settings/runtime")).body);
  const serialized = JSON.stringify(response);
  expect(serialized).not.toContain(requiredE2EEnv("WAF_E2E_PASSWORD"));
  expect(response.logging).toEqual(original.logging);
  for (const key of [response.logging?.vault?.token, response.logging?.opensearch?.password, response.logging?.opensearch?.api_key, response.logging?.clickhouse?.password].filter(Boolean)) expect(key).toBe("********");
  await openPage(page, "/settings/secrets", readySettingsPage(page));
  await expect(page).toHaveURL(/\/settings\/secrets$/);
  expect(await page.locator("#settings-logging-vault-token").inputValue()).toBe("");
  await openPage(page, "/settings/logging", readySettingsPage(page));
  await verifyToggles([["#settings-logging-opensearch-password", "#settings-logging-opensearch-password-toggle"], ["#settings-logging-opensearch-apikey", "#settings-logging-opensearch-apikey-toggle"], ["#settings-logging-password", "#settings-logging-password-toggle"]]);
});
