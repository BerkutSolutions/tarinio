import { expect, test } from "../fixtures/auth";

async function api(page: import("@playwright/test").Page, path: string) {
  return page.evaluate(async (path) => {
    const response = await fetch(path, { credentials: "include", headers: { Accept: "application/json" } });
    return { status: response.status, body: await response.text() };
  }, path);
}

async function ensureSecondEventsPage(page: import("@playwright/test").Page) {
  const result = await page.evaluate(async () => {
    for (let attempt = 0; attempt < 6; attempt += 1) {
      const existing = await fetch("/api/events?limit=500&offset=0", { credentials: "include" });
      const payload = await existing.json();
      if (existing.ok && Number(payload.total) >= 11) return { total: Number(payload.total) };

      const compiled = await fetch("/api/revisions/compile", {
        method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}"
      });
      const revision = await compiled.json();
      const id = String(revision.revision_id || revision.id || revision.revision?.id || "");
      if (!compiled.ok || !id) return { error: `compile failed (${compiled.status})` };

      const applied = await fetch(`/api/revisions/${encodeURIComponent(id)}/apply`, {
        method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}"
      });
      if (!applied.ok) return { error: `apply failed (${applied.status})` };
    }
    return { error: "fewer than 11 real runtime events after six compile/apply operations" };
  });
  expect(result.error || "", result.error).toBe("");
}

test("events.api-filter-pagination-detail-contract", async ({ authenticatedPage: page }) => {
  await page.goto("/events", { waitUntil: "domcontentloaded" });
  const result = await api(page, "/api/events?limit=500&offset=0");
  expect(result.status, result.body).toBe(200);
  const payload = JSON.parse(result.body);
  expect(Array.isArray(payload.events)).toBe(true);
  expect(payload.limit).toBe(500);
  expect(payload.offset).toBe(0);
  expect(payload.total).toBeGreaterThanOrEqual(payload.events.length);
  expect(payload.events.length).toBeGreaterThan(1);
  const item = payload.events[0];
  for (const key of ["id", "type", "severity", "source_component", "occurred_at", "summary"]) expect(item).toHaveProperty(key);
  const filtered = await api(page, `/api/events?type=${encodeURIComponent(item.type)}&severity=${encodeURIComponent(item.severity)}&site_id=${encodeURIComponent(item.site_id || "")}&limit=1&offset=0`);
  expect(filtered.status, filtered.body).toBe(200);
  const filteredPayload = JSON.parse(filtered.body);
  expect(filteredPayload.events).toHaveLength(1);
  expect(String(filteredPayload.events[0].type)).toBe(String(item.type));
  expect(String(filteredPayload.events[0].severity)).toBe(String(item.severity));
  const second = await api(page, "/api/events?limit=1&offset=1");
  expect(second.status, second.body).toBe(200);
  expect(JSON.parse(second.body).events).toHaveLength(1);
  const invalid = await api(page, "/api/events?from=not-rfc3339");
  expect(invalid.status, invalid.body).toBe(400);
});

test("events.browser-pagination-detail-keyboard", async ({ authenticatedPage: page }) => {
  await ensureSecondEventsPage(page);
  await page.goto("/events", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#events-filters")).toBeVisible();
  const row = page.locator("[data-event-row]").first();
  await expect(row).toBeVisible();
  await row.focus();
  await row.press("Enter");
  await expect(page.locator("#events-detail-modal")).toBeVisible();
  await expect(page.locator("#events-detail-content tr")).toHaveCount(11);
  await page.locator("#events-detail-modal").press("Escape");
  await expect(page.locator("#events-detail-modal")).toBeHidden();
  await row.press(" " );
  await expect(page.locator("#events-detail-modal")).toBeVisible();
  await page.locator("#events-detail-modal [data-events-detail-close]").last().click();
  const size = page.locator("#events-page-size");
  await expect(size).toBeVisible();
  await size.selectOption("10");
  expect(await page.locator("[data-event-row]").count()).toBeLessThanOrEqual(10);
  const page2 = page.locator("[data-events-page='2']");
  await expect(page2).toBeVisible();
  await page2.click();
  await expect(page2).toHaveAttribute("aria-current", "page");
});

test("events.loading-empty-error-malformed", async ({ authenticatedPage: source }) => {
  for (const state of ["loading", "empty", "error", "malformed"] as const) {
    const page = await source.context().newPage();
    try {
      if (state === "loading") await page.route("**/api/events**", async (route) => { await new Promise((resolve) => setTimeout(resolve, 600)); await route.fulfill({ status: 200, contentType: "application/json", body: '{"events":[]}' }); });
      if (state === "empty") await page.route("**/api/events**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: '{"events":[],"total":0}' }));
      if (state === "error") await page.route("**/api/events**", (route) => route.fulfill({ status: 503, contentType: "application/json", body: '{"error":"unavailable"}' }));
      if (state === "malformed") await page.route("**/api/events**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: '{"events":"bad"}' }));
      await page.goto("/events", { waitUntil: "domcontentloaded" });
      if (state === "loading") await expect(page.locator("#events-status")).toContainText(/loading|загруз|laden|učit|加载/i);
      else if (state === "error") await expect(page.locator("#events-status")).not.toBeEmpty();
      else await expect(page.locator("#events-list")).toContainText(/no events|нет событий|событий пока нет|keine|nema|没有/i);
      await expect(page.locator("nav")).toBeVisible();
    } finally { await page.close(); }
  }
});
