import { expect, test } from "../fixtures/auth";
import { openPage, waitForStableDOM } from "../support/waits";

const apiByTab: Array<[string, string[]]> = [
  ["dashboard", ["/api/dashboard/stats", "/api/dashboard/containers/overview"]],
  ["services", ["/api/sites", "/api/upstreams", "/api/easy-site-profiles", "/api/tls-configs"]],
  ["requests", ["/api/requests?limit=1&offset=0"]],
  ["bans", ["/api/sites", "/api/access-policies", "/api/events"]],
  ["revisions", ["/api/revisions"]],
  ["anti-ddos", ["/api/anti-ddos/settings"]],
  ["owasp-crs", ["/api/owasp-crs/status"]],
  ["tls", ["/api/certificates", "/api/tls-configs"]],
  ["administration", ["/api/administration/users", "/api/administration/roles", "/api/administration/scripts"]],
  ["events", ["/api/events?limit=1&offset=0"]],
  ["activity", ["/api/audit?limit=1&offset=0"]],
  ["settings", ["/api/settings/runtime", "/api/settings/management-hosts"]],
];

for (const [tab, endpoints] of apiByTab) {
  test("api." + tab + ".read-contract", async ({ authenticatedPage: page }) => {
    await page.goto("/login", { waitUntil: "domcontentloaded", timeout: 60000 });
    for (const endpoint of endpoints) {
      const result = await page.evaluate(async (path) => {
        const controller = new AbortController();
        const timer = window.setTimeout(() => controller.abort(), 15000);
        try {
          const response = await fetch(path, { credentials: "include", headers: { Accept: "application/json" }, signal: controller.signal });
          return { status: response.status, contentType: response.headers.get("content-type") || "", body: await response.text() };
        } finally {
          window.clearTimeout(timer);
        }
      }, endpoint);
      expect(result.status, `${endpoint}: ${result.body}`).toBe(200);
      expect(result.contentType, endpoint).toContain("json");
      const payload = JSON.parse(result.body);
      expect(payload, endpoint).not.toBeNull();
      expect(["object"], endpoint).toContain(typeof payload);
    }
  });
}

test("requests.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/requests", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#requests-refresh").click();
  await page.locator("#requests-search").fill("unlikely-request-filter");
  await page.locator("#requests-filter-method").selectOption({ index: 0 });
  await page.locator("#requests-filter-status").selectOption({ index: 0 });
  await expect(page.locator("#requests-detail-modal")).toBeHidden();
});

test("requests.pagination-sort-detail", async ({ authenticatedPage: page }) => {
  await page.goto("/requests", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#requests-page-size")).toBeVisible({ timeout: 30000 });
  await page.locator("#requests-page-size").selectOption("25");
  await expect(page.locator("#requests-page-size")).toHaveValue("25");
  const sort = page.locator("[data-sort-col]").first();
  await expect(sort).toBeVisible();
  const before = await sort.textContent();
  await sort.click();
  await expect(sort).not.toHaveText(before || "");
  const row = page.locator("[data-request-row]").first();
  await expect(row).toBeVisible();
  await row.press("Enter");
  await expect(page.locator("#requests-detail-modal")).toBeVisible();
  await page.locator("#requests-detail-modal").press("Escape");
  await expect(page.locator("#requests-detail-modal")).toBeHidden();
});

test("bans.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#bans-refresh").click();
  await expect(page.locator("#bans-status")).toBeVisible({ timeout: 30000 });
  const bansPageSize = page.locator("#bans-page-size");
  await expect(bansPageSize).toBeVisible();
  await bansPageSize.selectOption({ index: 0 });
  await page.locator("#bans-create").click();
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  await page.locator("#bans-create-modal [data-bans-create-close]").last().click();
  await expect(page.locator("#bans-create-modal")).toBeHidden();
});

test("bans.create-validation", async ({ authenticatedPage: page }) => {
  await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#bans-create").click();
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  await page.locator("#bans-create-ip").fill("not-an-ip");
  await page.locator("#bans-create-submit").click();
  await expect(page.locator("#bans-create-status")).toContainText(/invalid|недопуст|IP/i);
  await page.locator("#bans-create-modal [data-bans-create-close]").last().click();
  await expect(page.locator("#bans-create-modal")).toBeHidden();
});

test("bans.create-cancel", async ({ authenticatedPage: page }) => {
  await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#bans-create").click();
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  await page.locator("#bans-create-modal [data-bans-create-close]").last().click();
  await expect(page.locator("#bans-create-modal")).toBeHidden();
});

test("revisions.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#revisions-refresh").click();
  await expect(page.locator("#revisions-detail-modal")).toBeHidden();
});

test("anti-ddos.help-modal", async ({ authenticatedPage: page }) => {
  await page.goto("/anti-ddos", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#antiddos-form")).toBeVisible({ timeout: 10000 });
  await page.locator("#antiddos-model-help-btn").click();
  await expect(page.locator("#antiddos-model-help-modal")).toBeVisible();
  await page.locator("#antiddos-model-help-modal").press("Escape");
  await expect(page.locator("#antiddos-model-help-modal")).toBeHidden();
});

test("owasp-crs.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/owasp-crs", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#owasp-crs-check")).toBeVisible();
  await expect(page.locator("#owasp-crs-update")).toBeVisible();
  await expect(page.locator("#owasp-crs-hourly-auto")).toBeVisible();
});

test("tls.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#certificate-refresh").click();
  await page.locator("#tls-config-refresh").click();
  await expect(page.locator("#certificate-form")).toBeVisible();
  await expect(page.locator("#tls-config-form")).toBeVisible();
});

test("administration.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/administration", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#administration-user-create")).toBeVisible();
  await expect(page.locator("#administration-role-create")).toBeVisible();
});

test("events.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/events", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#events-filters")).toBeVisible({ timeout: 30000 });
  const eventsPageSize = page.locator("#events-page-size");
  await expect(eventsPageSize).toBeVisible();
  await eventsPageSize.selectOption({ index: 0 });
  await page.locator("#events-type").fill("unlikely-event-filter");
  await expect(page.locator("#events-detail-modal")).toBeHidden();
});

test("activity.read-controls", async ({ authenticatedPage: page }) => {
  await page.goto("/activity", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#audit-reset").click();
  await expect(page.locator("#audit-results")).toBeVisible();
  await expect(page.locator("#audit-page-info")).toBeVisible();
});

test("settings.read-controls", async ({ authenticatedPage: page }) => {
  await openPage(page, "/settings/general", "#settings-language-save");
  await expect(page.locator("#settings-runtime-save")).toBeVisible();
  await openPage(page, "/settings/security", "#settings-security-save");
  await openPage(page, "/settings/management-hosts", "#settings-management-hosts");
});

test("settings.validation-no-partial-save", async ({ authenticatedPage: page }) => {
  await page.goto("/settings/general", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#settings-page")).toBeVisible({ timeout: 30000 });
  await page.locator("[data-settings-tab-link='security']").click();
  await expect.poll(() => new URL(page.url()).pathname).toBe("/settings/security");
  await expect(page.locator("#settings-security-save")).toBeVisible({ timeout: 30000 });
  await waitForStableDOM(page.locator("#settings-security-save"));
  const invalidField = page.locator("#settings-security-login-rate-attempts");
  await expect(invalidField).toHaveValue(/\d+/);
  const before = await invalidField.inputValue();
  const beforeRuntime = await page.evaluate(async () => fetch("/api/settings/runtime", { credentials: "include" }).then((response) => response.json()));
  await invalidField.evaluate((node) => {
    const input = node as HTMLInputElement;
    const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
    setter?.call(input, "1");
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await expect(invalidField).toHaveValue("1");
  await expect.poll(() => invalidField.evaluate((node) => (node as HTMLInputElement).validity.valid)).toBe(false);
  await page.locator("#settings-security-save").evaluate((button) => (button as HTMLButtonElement).click());
  await expect(invalidField).toHaveValue("1");
  const afterRuntime = await page.evaluate(async () => fetch("/api/settings/runtime", { credentials: "include" }).then((response) => response.json()));
  expect(afterRuntime.security).toEqual(beforeRuntime.security);
  expect(before).not.toBe("1");
});

test("revisions.detail-modal-close", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#revisions-page")).toBeVisible({ timeout: 30000 });
  const tile = page.locator("[data-revision-site]").first();
  await expect(tile).toBeVisible({ timeout: 30000 });
  await tile.click();
  await expect(page.locator("#revisions-detail-modal")).toBeVisible();
  await page.locator("#revisions-detail-modal [data-revisions-modal-close]").last().click();
  await expect(page.locator("#revisions-detail-modal")).toBeHidden();
});

test("revisions.clear-statuses-control", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60000 });
  const before = await page.evaluate(async () => fetch("/api/revisions", { credentials: "include" }).then((response) => response.json()));
  const originalID = String((before?.revisions || []).find((item: { is_active?: boolean }) => item.is_active)?.id || "");
  expect(originalID).toBeTruthy();
  const sites = await page.evaluate(async () => fetch("/api/sites", { credentials: "include" }).then((response) => response.json()));
  const siteID = String((Array.isArray(sites) ? sites : sites?.sites || [])[0]?.id || "");
  expect(siteID).toBeTruthy();
  const compile = await page.evaluate(async (targetSiteID) => {
    const response = await fetch("/api/revisions/compile", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ target_site_ids: [targetSiteID] }) });
    return { status: response.status, body: await response.text() };
  }, siteID);
  expect(compile.status, compile.body).toBe(201);
  const candidateID = String(JSON.parse(compile.body)?.revision?.id || "");
  expect(candidateID).toBeTruthy();
  const clear = page.locator("#revisions-clear-statuses");
  try {
    const apply = await page.evaluate(async (id) => {
      const response = await fetch(`/api/revisions/${encodeURIComponent(id)}/apply`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" });
      return { status: response.status, body: await response.text() };
    }, candidateID);
    expect(apply.status, apply.body).toBe(201);
    await expect.poll(async () => page.evaluate(async () => {
      const payload = await fetch("/api/revisions", { credentials: "include" }).then((response) => response.json());
      return String((payload?.revisions || []).find((item: { is_active?: boolean }) => item.is_active)?.id || "");
    }), { timeout: 120_000 }).toBe(candidateID);
    const rollback = await page.evaluate(async (id) => {
      const response = await fetch(`/api/revisions/${encodeURIComponent(id)}/apply`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: "{}" });
      return { status: response.status, body: await response.text() };
    }, originalID);
    expect(rollback.status, rollback.body).toBe(201);
    await expect.poll(async () => page.evaluate(async () => {
      const payload = await fetch("/api/revisions", { credentials: "include" }).then((response) => response.json());
      return String((payload?.revisions || []).find((item: { is_active?: boolean }) => item.is_active)?.id || "");
    }), { timeout: 120_000 }).toBe(originalID);
    await page.locator("#revisions-refresh").click();
    const timelineItem = page.locator("[data-revision-status-index]").first();
    await expect(timelineItem).toBeVisible();
    await timelineItem.click();
    await expect(clear).toBeVisible();
    page.once("dialog", (dialog) => dialog.accept());
    await clear.click();
    await expect(clear).toBeHidden();
    await expect.poll(async () => page.evaluate(async () => {
      const payload = await fetch("/api/revisions", { credentials: "include" }).then((response) => response.json());
      return Array.isArray(payload?.timeline) ? payload.timeline.length : -1;
    })).toBe(0);
  } finally {
    const deletion = await page.evaluate(async (id) => {
      const response = await fetch(`/api/revisions/${encodeURIComponent(id)}`, { method: "DELETE", credentials: "include" });
      return response.status;
    }, candidateID);
    expect([200, 404]).toContain(deletion);
  }
});
