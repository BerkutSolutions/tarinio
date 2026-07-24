import { expect, test } from "../fixtures/auth";
import { requiredE2EEnv } from "../support/env";

test("auth.login-ui-real-submit", async ({ authenticatedPage: page }) => {
  await page.context().clearCookies();
  await page.goto("/login", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#login-form")).toBeVisible({ timeout: 30000 });
  await expect(page.locator("#username")).toBeVisible();
  await expect(page.locator("#password")).toBeVisible();
  await page.locator("#username").fill(requiredE2EEnv("WAF_E2E_USERNAME"));
  await page.locator("#password").fill(requiredE2EEnv("WAF_E2E_PASSWORD"));
  await page.locator("#login-form").evaluate((form) => form.requestSubmit());
  await expect(page).toHaveURL(/\/(dashboard|healthcheck)/, { timeout: 30000 });
});
