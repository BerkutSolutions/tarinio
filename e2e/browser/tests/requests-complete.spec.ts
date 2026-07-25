import { expect, test } from "../fixtures/auth";

type RequestRow = { entry?: { request_id?: string; status?: number }; security_reason?: string };

async function requests(page: import("@playwright/test").Page) {
  return page.evaluate(async () => {
    const response = await fetch("/api/requests?limit=500", { credentials: "include", headers: { Accept: "application/json" } });
    return { status: response.status, body: await response.json() };
  });
}

test("requests.real-runtime-filter-pagination-detail", async ({ authenticatedPage: page }) => {
  const response = await requests(page);
  expect(response.status).toBe(200);
  expect(Array.isArray(response.body)).toBe(true);
  const rows = response.body as RequestRow[];
  expect(rows.length).toBeGreaterThan(25);
  expect(rows.some((row) => String(row.entry?.request_id || "").startsWith("e2e-dashboard-request-"))).toBe(true);
  expect(rows.some((row) => String(row.entry?.request_id || "").startsWith("e2e-dashboard-attack-"))).toBe(true);

  await page.goto("/requests", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("#requests-status")).toContainText(/\d+/);
  await expect(page.locator("[data-request-row]").first()).toBeVisible({ timeout: 30_000 });

  const reason = page.locator("#requests-filter-security-reason");
  await expect(reason).toBeVisible();
  const availableReasons = await reason.locator("option").evaluateAll((options) => options.map((option) => (option as HTMLOptionElement).value).filter(Boolean));
  expect(availableReasons.length).toBeGreaterThan(0);
  await reason.selectOption(availableReasons[0]);
  await expect(page.locator("[data-request-row]").first()).toBeVisible();

  await page.locator("#requests-page-size").selectOption("10");
  await expect(page.locator("[data-request-row]")).toHaveCount(10);
  const secondPage = page.locator("[data-requests-page='2']");
  await expect(secondPage).toBeVisible();
  await secondPage.click();
  await expect(secondPage).toHaveClass(/active/);

  const row = page.locator("[data-request-row]").first();
  await row.focus();
  await row.press("Enter");
  await expect(page.locator("#requests-detail-modal")).toBeVisible();
  await expect(page.locator("#requests-detail-content")).toContainText(/request|запрос/i);
  await page.locator("#requests-detail-modal").press("Escape");
  await expect(page.locator("#requests-detail-modal")).toBeHidden();
});
