import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";

test("bans.readonly-rbac-and-server-errors", async ({ authenticatedPage: adminPage }, testInfo) => {
  const roleID = e2eID(testInfo, "e2e-bans-viewer");
  const username = e2eID(testInfo, "e2e-bans-reader");
  const password = "E2e-bans-reader-1234!";
  const api = async (path: string, init: RequestInit = {}) => adminPage.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const readUsers = async () => JSON.parse((await api("/api/administration/users")).body)?.users || [];
  const sitesPayload = JSON.parse((await api("/api/sites")).body);
  const siteID = (Array.isArray(sitesPayload) ? sitesPayload : sitesPayload?.sites || [])[0]?.id || "control-plane-access";
  const testIP = "203.0.113.222";
  const cleanup = new CleanupLedger();
  cleanup.add("forbidden ban " + testIP, () => api(`/api/sites/${encodeURIComponent(siteID)}/unban`, { method: "POST", body: JSON.stringify({ ip: testIP }) }), async () => {
    const policies = JSON.parse((await api("/api/access-policies")).body);
    const list = Array.isArray(policies) ? policies : policies?.access_policies || [];
    return !list.find((item: { site_id?: string }) => item.site_id === siteID)?.denylist?.includes(testIP);
  });
  cleanup.add("role " + roleID, () => api("/api/administration/roles/" + encodeURIComponent(roleID), { method: "DELETE" }), async () => {
    const roles = JSON.parse((await api("/api/administration/roles")).body)?.roles || [];
    return !roles.some((item: { id?: string }) => item.id === roleID);
  });
  cleanup.add("user " + username, async () => {
    const user = (await readUsers()).find((item: { username?: string }) => item.username === username);
    if (user?.id) await api("/api/administration/users/" + encodeURIComponent(user.id), { method: "DELETE" });
  }, async () => !(await readUsers()).some((item: { username?: string }) => item.username === username));
  let context;
  try {
    expect([200, 201]).toContain((await api("/api/administration/roles", { method: "POST", body: JSON.stringify({ id: roleID, name: "E2E Bans Reader", permissions: ["bans.read", "sites.read", "access.read", "events.read", "reports.read", "auth.self", "profile.read"] }) })).status);
    expect([200, 201]).toContain((await api("/api/administration/users", { method: "POST", body: JSON.stringify({ id: username, username, email: username + "@example.test", password, role_ids: [roleID], is_active: true }) })).status);
    const seeded = await api(`/api/sites/${encodeURIComponent(siteID)}/ban`, { method: "POST", body: JSON.stringify({ ip: testIP }) });
    expect([200, 201], seeded.body).toContain(seeded.status);
    context = await adminPage.context().browser()!.newContext({ ignoreHTTPSErrors: true });
    await context.clearCookies();
    const page = await context.newPage();
    await page.goto("/login", { waitUntil: "domcontentloaded", timeout: 60_000 });
    const login = await page.evaluate(async ({ username, password }) => {
      const response = await fetch("/api/auth/login", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ username, password }) });
      return { status: response.status, body: await response.text() };
    }, { username, password });
    expect(login.status, login.body).toBe(200);
    await expect.poll(async () => page.evaluate(async () => fetch("/api/auth/me", { credentials: "include" }).then((response) => response.json()).then((value) => value.username))).toBe(username);
    await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(page.locator("#bans-refresh")).toBeVisible({ timeout: 30_000 });
    await expect(page.locator("#bans-create")).toBeVisible();
    const session = await page.evaluate(async () => fetch("/api/auth/me", { credentials: "include" }).then((response) => response.json()));
    expect(session.username).toBe(username);
    expect(session.permissions).toContain("bans.read");
    expect(session.permissions).not.toContain("access.write");
    const row = page.locator("[data-ban-row]", { hasText: testIP }).first();
    await expect(row).toBeVisible();
    await row.locator("[data-action='extend']").click();
    await page.locator("#bans-extend-duration").selectOption("3600");
    await page.locator("#bans-extend-submit").click();
    await expect(page.locator("#bans-extend-status")).not.toBeEmpty();
    await expect(page.locator("#bans-extend-modal")).toBeVisible();
    await page.locator("#bans-extend-modal [data-bans-extend-close]").last().click();
    page.once("dialog", (dialog) => dialog.accept());
    await row.locator("[data-action='unban']").click();
    await expect(row).toBeVisible();
    await expect.poll(async () => {
      const policies = JSON.parse((await api("/api/access-policies")).body);
      const list = Array.isArray(policies) ? policies : policies?.access_policies || [];
      return Boolean(list.find((item: { site_id?: string }) => item.site_id === siteID)?.denylist?.includes(testIP));
    }).toBe(true);
    const forbidden = await page.evaluate(async ({ siteID, testIP }) => {
      const response = await fetch(`/api/sites/${encodeURIComponent(siteID)}/ban`, { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ip: testIP }) });
      return response.status;
    }, { siteID, testIP });
    expect(forbidden).toBe(403);
  } finally {
    await context?.close();
    await cleanup.run();
  }
});
