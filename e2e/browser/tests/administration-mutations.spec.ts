import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("administration.user-create-edit-readback", async ({ authenticatedPage: page }, testInfo) => {
  const username = e2eID(testInfo, "e2e-user");
  const email = username + "@example.test";
  const updatedEmail = "updated-" + email;
  await openPage(page, "/administration", "#administration-user-create");
  await expect(page.locator("#administration-user-create")).toBeVisible({ timeout: 30000 });
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const readUser = async () => {
    const result = await api("/api/administration/users");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const users = Array.isArray(payload?.users) ? payload.users : [];
    return users.find((user: { username?: string }) => user.username === username) || null;
  };
  const cleanup = new CleanupLedger();
  cleanup.add("user " + username, async () => {
    const user = await readUser();
    if (user?.id) await api("/api/administration/users/" + encodeURIComponent(user.id), { method: "DELETE" });
  }, async () => (await readUser()) === null);
  try {
    await page.locator("#administration-user-create").click();
    await expect(page.locator("#administration-entity-modal")).toBeVisible();
    await page.locator("#administration-user-username").fill(username);
    await page.locator("#administration-user-email").fill(email);
    await page.locator("#administration-user-password").fill("E2e-password-1234!");
    await page.locator("#administration-user-password-confirm").fill("E2e-password-1234!");
    const role = page.locator("#administration-user-form input[name='role_ids']").first();
    await expect(role).toBeVisible();
    await role.check();
    await page.locator("#administration-user-form button[type=submit]").click();
    await expect(page.locator("#administration-entity-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(async () => (await readUser())?.email, { timeout: 30000 }).toBe(email);

    const user = await readUser();
    expect(user?.id).toBeTruthy();
    await page.locator("[data-user-edit]").evaluateAll((nodes, id) => (nodes.find((node) => node.getAttribute("data-user-edit") === id) as HTMLElement)?.click(), user.id);
    await expect(page.locator("#administration-user-form")).toBeVisible();
    await page.locator("#administration-user-email").fill(updatedEmail);
    await page.locator("#administration-user-password").fill("");
    await page.locator("#administration-user-password-confirm").fill("");
    await page.locator("#administration-user-form button[type=submit]").click();
    await expect(page.locator("#administration-entity-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(async () => (await readUser())?.email, { timeout: 30000 }).toBe(updatedEmail);
    const deleteButton = page.locator(`[data-user-delete="${username}"]`);
    page.once("dialog", (dialog) => dialog.dismiss());
    await deleteButton.click();
    await expect.poll(async () => (await readUser())?.email, { timeout: 30000 }).toBe(updatedEmail);
    page.once("dialog", (dialog) => dialog.accept());
    await deleteButton.click();
    await expect.poll(async () => (await readUser()) === null, { timeout: 30000 }).toBe(true);
    const audit = await api("/api/audit?limit=500");
    expect(audit.status).toBe(200);
    expect(audit.body).toContain("administration.user.delete");
  } finally {
    await cleanup.run();
  }
});

test("administration.role-create-edit-readback", async ({ authenticatedPage: page }, testInfo) => {
  const roleID = e2eID(testInfo, "e2e-role");
  const roleName = "E2E Role " + roleID;
  const updatedName = roleName + " Updated";
  await openPage(page, "/administration", "#administration-role-create");
  await expect(page.locator("#administration-role-create")).toBeVisible({ timeout: 30000 });
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const readRole = async () => {
    const result = await api("/api/administration/roles");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const roles = Array.isArray(payload?.roles) ? payload.roles : [];
    return roles.find((role: { id?: string }) => role.id === roleID) || null;
  };
  const cleanup = new CleanupLedger();
  cleanup.add("role " + roleID, () => api("/api/administration/roles/" + encodeURIComponent(roleID), { method: "DELETE" }), async () => (await readRole()) === null);
  try {
    await page.locator("#administration-role-create").click();
    await expect(page.locator("#administration-role-form")).toBeVisible();
    await page.locator("#administration-role-id").fill(roleID);
    await page.locator("#administration-role-name").fill(roleName);
    const permission = page.locator("#administration-role-form input[name='permissions']").first();
    await expect(permission).toBeVisible();
    await permission.check();
    await page.locator("#administration-role-form button[type=submit]").click();
    await expect(page.locator("#administration-entity-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(async () => (await readRole())?.name, { timeout: 30000 }).toBe(roleName);
    await page.locator("[data-role-edit]").evaluateAll((nodes, id) => (nodes.find((node) => node.getAttribute("data-role-edit") === id) as HTMLElement)?.click(), roleID);
    await expect(page.locator("#administration-role-form")).toBeVisible();
    await page.locator("#administration-role-name").fill(updatedName);
    await page.locator("#administration-role-form button[type=submit]").click();
    await expect(page.locator("#administration-entity-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(async () => (await readRole())?.name, { timeout: 30000 }).toBe(updatedName);
    const deleteButton = page.locator(`[data-role-delete="${roleID}"]`);
    page.once("dialog", (dialog) => dialog.dismiss());
    await deleteButton.click();
    await expect.poll(async () => (await readRole())?.name, { timeout: 30000 }).toBe(updatedName);
    page.once("dialog", (dialog) => dialog.accept());
    await deleteButton.click();
    await expect.poll(async () => (await readRole()) === null, { timeout: 30000 }).toBe(true);
    const audit = await api("/api/audit?limit=500");
    expect(audit.status).toBe(200);
    expect(audit.body).toContain("administration.role.delete");
  } finally {
    await cleanup.run();
  }
});
