import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";

test("services.readonly-rbac", async ({ authenticatedPage: adminPage }, testInfo) => {
  const roleID = e2eID(testInfo, "e2e-sites-viewer");
  const username = e2eID(testInfo, "e2e-sites-reader");
  const password = "E2e-reader-password-1234!";
  const api = async (path: string, init: RequestInit = {}) => adminPage.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const readUsers = async () => JSON.parse((await api("/api/administration/users")).body)?.users || [];
  const cleanup = new CleanupLedger();
  cleanup.add("role " + roleID, () => api("/api/administration/roles/" + encodeURIComponent(roleID), { method: "DELETE" }), async () => {
    const roles = JSON.parse((await api("/api/administration/roles")).body)?.roles || [];
    return !roles.some((item: { id?: string }) => item.id === roleID);
  });
  cleanup.add("user " + username, async () => {
    const user = (await readUsers()).find((item: { username?: string }) => item.username === username);
    if (user?.id) await api("/api/administration/users/" + encodeURIComponent(user.id), { method: "DELETE" });
  }, async () => !(await readUsers()).some((item: { username?: string }) => item.username === username));
  let readerContext;
  try {
    let response = await api("/api/administration/roles", { method: "POST", body: JSON.stringify({ id: roleID, name: "E2E Services Reader", permissions: [
      "sites.read", "upstreams.read", "tls.read", "certificates.read", "access.read", "policies.read", "ratelimits.read", "auth.self", "profile.read", "healthcheck.read"
    ] }) });
    expect([200, 201]).toContain(response.status);
    response = await api("/api/administration/users", { method: "POST", body: JSON.stringify({ id: username, username, email: username + "@example.test", password, role_ids: [roleID], is_active: true }) });
    expect([200, 201]).toContain(response.status);

    readerContext = await adminPage.context().browser()!.newContext({ ignoreHTTPSErrors: true });
    const reader = await readerContext.newPage();
    await reader.goto("/login", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await reader.locator("#username").fill(username);
    await reader.locator("#password").fill(password);
    await reader.locator("#login-form").evaluate((form) => (form as HTMLFormElement).requestSubmit());
    await expect(reader).toHaveURL(/\/(dashboard|services|healthcheck)$/, { timeout: 30_000 });
    await expect.poll(() => reader.evaluate(async () => (await fetch("/api/auth/me", { credentials: "include" })).status), { timeout: 30_000 }).toBe(200);
    await reader.goto("/services", { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(reader.locator("#services-refresh")).toBeVisible({ timeout: 30_000 });
    await expect(reader.locator("#services-create, #services-import, #services-delete-selected, #services-select-all, [data-select-site], [data-toggle-site]")).toHaveCount(0);
    await expect(reader.locator("[data-open-site]").first()).toBeVisible();
    const forbidden = await reader.evaluate(async () => {
      const response = await fetch("/api/sites?auto_apply=false", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: "forbidden-rbac-site", primary_host: "forbidden-rbac.test" }) });
      return response.status;
    });
    expect(forbidden).toBe(403);
  } finally {
    await readerContext?.close();
    await cleanup.run();
  }
});
