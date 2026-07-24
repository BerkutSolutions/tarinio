import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

test("cross-security.readonly-rbac-and-session-expiry-return-login", async ({ authenticatedPage: adminPage }, testInfo) => {
  const roleID = e2eID(testInfo, "e2e-cross-reader-role");
  const username = e2eID(testInfo, "e2e-cross-reader");
  const password = "E2e-cross-reader-1234!";
  const permissions = ["auth.self", "healthcheck.read", "profile.read", "dashboard.read", "sites.read", "upstreams.read", "antiddos.read", "owaspcrs.read", "certificates.read", "tls.read", "policies.read", "access.read", "ratelimits.read", "requests.read", "reports.read", "events.read", "bans.read", "revisions.read", "activity.read", "administration.read", "administration.users.read", "administration.roles.read", "settings.general.read", "settings.storage.read", "settings.about.read"];
  const cleanup = new CleanupLedger();
  cleanup.add("role " + roleID, () => api(adminPage, `/api/administration/roles/${encodeURIComponent(roleID)}`, { method: "DELETE" }), async () => !JSON.parse((await api(adminPage, "/api/administration/roles")).body)?.roles?.some((item: { id?: string }) => item.id === roleID));
  cleanup.add("user " + username, () => api(adminPage, `/api/administration/users/${encodeURIComponent(username)}`, { method: "DELETE" }), async () => !JSON.parse((await api(adminPage, "/api/administration/users")).body)?.users?.some((item: { id?: string }) => item.id === username));
  let context;
  try {
    expect([200, 201]).toContain((await api(adminPage, "/api/administration/roles", { method: "POST", body: JSON.stringify({ id: roleID, name: "E2E Cross Reader", permissions }) })).status);
    expect([200, 201]).toContain((await api(adminPage, "/api/administration/users", { method: "POST", body: JSON.stringify({ id: username, username, email: `${username}@example.test`, password, role_ids: [roleID], is_active: true }) })).status);
    context = await adminPage.context().browser()!.newContext({ ignoreHTTPSErrors: true });
    await context.clearCookies();
    const page = await context.newPage();
    await page.goto("/login", { waitUntil: "domcontentloaded" });
    const login = await api(page, "/api/auth/login", { method: "POST", body: JSON.stringify({ username, password }) });
    expect(login.status, login.body).toBe(200);
    const session = JSON.parse((await api(page, "/api/auth/me")).body);
    expect(session.username).toBe(username);
    for (const permission of permissions) expect(session.permissions).toContain(permission);
    for (const forbiddenPermission of ["sites.write", "access.write", "revisions.write", "antiddos.write", "owaspcrs.write", "certificates.write", "tls.write", "settings.general.write", "administration.users.write"]) expect(session.permissions).not.toContain(forbiddenPermission);

    const reads = ["/api/dashboard/stats", "/api/sites", "/api/upstreams", "/api/requests?limit=1", "/api/access-policies", "/api/revisions", "/api/anti-ddos/settings", "/api/owasp-crs/status", "/api/certificates", "/api/tls-configs", "/api/events?limit=1", "/api/audit?limit=1", "/api/settings/runtime", "/api/administration/users", "/api/administration/roles"];
    for (const path of reads) expect((await api(page, path)).status, `read ${path}`).toBe(200);
    const writes: Array<[string, RequestInit]> = [
      ["/api/sites?auto_apply=false", { method: "POST", body: '{}' }],
      ["/api/upstreams?auto_apply=false", { method: "POST", body: '{}' }],
      ["/api/sites/missing/ban", { method: "POST", body: '{"ip":"203.0.113.9"}' }],
      ["/api/revisions/compile", { method: "POST", body: '{}' }],
      ["/api/anti-ddos/settings", { method: "PUT", body: '{}' }],
      ["/api/owasp-crs/update", { method: "POST", body: '{}' }],
      ["/api/certificates", { method: "POST", body: '{}' }],
      ["/api/tls-configs", { method: "POST", body: '{}' }],
      ["/api/settings/runtime", { method: "PUT", body: '{"update_checks_enabled":true}' }],
      ["/api/administration/users", { method: "POST", body: '{}' }],
    ];
    for (const [path, init] of writes) expect((await api(page, path, init)).status, `write ${path}`).toBe(403);

    const deletion = await api(adminPage, `/api/administration/users/${encodeURIComponent(username)}`, { method: "DELETE" });
    expect([200, 204], deletion.body).toContain(deletion.status);
    expect((await api(page, "/api/auth/me")).status).toBe(401);
    await page.goto("/events", { waitUntil: "domcontentloaded" });
    await expect(page).toHaveURL(/\/login\?reason=session_(expired|invalid|missing)/, { timeout: 30_000 });
  } finally {
    await context?.close();
    await cleanup.run();
  }
});
