import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

type Revision = { id: string; is_active?: boolean; status?: string; sites?: unknown[]; checksum?: string };

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 120_000);
    try {
      const response = await fetch(path, {
        ...init, credentials: "include", signal: controller.signal,
        headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
      });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
}

async function catalog(page: import("@playwright/test").Page): Promise<Revision[]> {
  const result = await api(page, "/api/revisions");
  expect(result.status, result.body).toBe(200);
  const payload = JSON.parse(result.body);
  return Array.isArray(payload?.revisions) ? payload.revisions : [];
}

test("revisions.compile-apply-rollback-artifact", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const before = await catalog(page);
  const original = before.find((revision) => revision.is_active);
  expect(original?.id).toBeTruthy();
  const sites = await api(page, "/api/sites");
  expect(sites.status, sites.body).toBe(200);
  const siteItems = Array.isArray(JSON.parse(sites.body)) ? JSON.parse(sites.body) : JSON.parse(sites.body)?.sites || [];
  const targetSiteID = String(siteItems[0]?.id || "");
  expect(targetSiteID).toBeTruthy();
  const compile = await api(page, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [targetSiteID] }) });
  expect(compile.status, compile.body).toBe(201);
  const compiled = JSON.parse(compile.body);
  const candidate: Revision = compiled.revision;
  try {
    expect(candidate.id).toBeTruthy();
    expect(candidate.id).not.toBe(original!.id);
    expect(candidate.checksum).toBeTruthy();
    await expect.poll(async () => (await catalog(page)).some((revision) => revision.id === candidate.id), { timeout: 60_000 }).toBe(true);
    const persistedCandidateBeforeApply = (await catalog(page)).find((revision) => revision.id === candidate.id);
    expect(Array.isArray(persistedCandidateBeforeApply?.sites)).toBe(true);
    expect((await catalog(page)).find((revision) => revision.is_active)?.id).toBe(original!.id);
    const applied = await api(page, `/api/revisions/${encodeURIComponent(candidate.id)}/apply`, { method: "POST", body: "{}" });
    expect(applied.status, applied.body).toBe(201);
    const applyJob = JSON.parse(applied.body);
    expect(applyJob.status).not.toBe("failed");
    await expect.poll(async () => (await catalog(page)).find((revision) => revision.is_active)?.id, { timeout: 120_000 }).toBe(candidate.id);

    const rollback = await api(page, `/api/revisions/${encodeURIComponent(original!.id)}/apply`, { method: "POST", body: "{}" });
    expect(rollback.status, rollback.body).toBe(201);
    await expect.poll(async () => (await catalog(page)).find((revision) => revision.is_active)?.id, { timeout: 120_000 }).toBe(original!.id);

    const afterRollback = (await catalog(page)).find((revision) => revision.id === original!.id);
    expect(afterRollback?.checksum).toBe(original!.checksum);
    const persistedCandidate = (await catalog(page)).find((revision) => revision.id === candidate.id);
    expect(persistedCandidate?.checksum).toBe(candidate.checksum);
    expect(persistedCandidate?.sites).toEqual(persistedCandidateBeforeApply?.sites);
    const audit = await api(page, "/api/audit?limit=500");
    expect(audit.status, audit.body).toBe(200);
    expect(audit.body).toContain("revision.compile_request");
    expect(audit.body).toContain("revision.apply_trigger");
    const rollbackAudit = await api(page, `/api/audit?action=revision.apply_trigger&resource_id=${encodeURIComponent(original!.id)}&status=succeeded&limit=10`);
    expect(rollbackAudit.status, rollbackAudit.body).toBe(200);
    expect(JSON.parse(rollbackAudit.body).items).toEqual(expect.arrayContaining([expect.objectContaining({ action: "revision.apply_trigger", resource_id: original!.id })]));
  } finally {
    if ((await catalog(page)).find((revision) => revision.is_active)?.id !== original!.id) {
      await api(page, `/api/revisions/${encodeURIComponent(original!.id)}/apply`, { method: "POST", body: "{}" });
    }
    if (candidate.id) {
      const deletion = await api(page, `/api/revisions/${encodeURIComponent(candidate.id)}`, { method: "DELETE" });
      expect([200, 404], deletion.body).toContain(deletion.status);
      await expect.poll(async () => (await catalog(page)).some((revision) => revision.id === candidate.id), { timeout: 60_000 }).toBe(false);
    }
  }
});

test("revisions.validation-delete-active-and-missing-errors", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const revisions = await catalog(page);
  const active = revisions.find((revision) => revision.is_active);
  expect(active?.id).toBeTruthy();
  const deleteActive = await api(page, `/api/revisions/${encodeURIComponent(active!.id)}`, { method: "DELETE" });
  expect(deleteActive.status, deleteActive.body).toBe(409);
  expect(JSON.parse(deleteActive.body).error).toBeTruthy();
  const applyMissing = await api(page, "/api/revisions/e2e-missing-revision/apply", { method: "POST", body: "{}" });
  expect(applyMissing.status, applyMissing.body).toBe(404);
  expect((await catalog(page)).find((revision) => revision.is_active)?.id).toBe(active!.id);
});

test("revisions.status-and-delete-confirm-cancel-ui", async ({ authenticatedPage: page }) => {
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const sitesResponse = await api(page, "/api/sites");
  expect(sitesResponse.status, sitesResponse.body).toBe(200);
  const sitesPayload = JSON.parse(sitesResponse.body);
  const siteItems = Array.isArray(sitesPayload) ? sitesPayload : sitesPayload?.sites || [];
  const targetSiteID = String(siteItems[0]?.id || "");
  expect(targetSiteID).toBeTruthy();
  const compile = await api(page, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [targetSiteID] }) });
  expect(compile.status, compile.body).toBe(201);
  const candidateID = String(JSON.parse(compile.body)?.revision?.id || "");
  expect(candidateID).toBeTruthy();
  try {
    const timeline = page.locator("[data-revision-status-index]").first();
    await openPage(page, "/revisions", timeline);
    await timeline.click();
    await expect(page.locator("#revisions-status-modal")).toBeVisible();
    await page.locator("#revisions-status-modal").press("Escape");
    await expect(page.locator("#revisions-status-modal")).toBeHidden();
    const tile = page.locator(`[data-revision-site="${targetSiteID}"]`);
    await expect(tile).toBeVisible();
    await tile.click();
    await expect(page.locator("#revisions-detail-modal")).toBeVisible();
    const deleteOthers = page.locator("#revisions-delete-others");
    await expect(deleteOthers).toBeEnabled();
    page.once("dialog", (dialog) => dialog.dismiss());
    await deleteOthers.click();
    await expect(page.locator("#revisions-detail-modal")).toBeVisible();
    expect((await catalog(page)).some((revision) => revision.id === candidateID)).toBe(true);
    await page.locator("#revisions-detail-modal [data-revisions-modal-close]").last().click();
  } finally {
    const deletion = await api(page, `/api/revisions/${encodeURIComponent(candidateID)}`, { method: "DELETE" });
    expect([200, 404], deletion.body).toContain(deletion.status);
    await expect.poll(async () => (await catalog(page)).some((revision) => revision.id === candidateID)).toBe(false);
  }
});

test("revisions.apply-and-delete-success-ui", async ({ authenticatedPage: page }) => {
  test.setTimeout(5 * 60_000);
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const original = (await catalog(page)).find((revision) => revision.is_active);
  expect(original?.id).toBeTruthy();
  const sitesResponse = await api(page, "/api/sites");
  expect(sitesResponse.status, sitesResponse.body).toBe(200);
  const sitesPayload = JSON.parse(sitesResponse.body);
  const siteItems = Array.isArray(sitesPayload) ? sitesPayload : sitesPayload?.sites || [];
  const targetSiteID = String(siteItems.find((item: { id?: string }) => item.id !== "control-plane-access")?.id || siteItems[0]?.id || "");
  expect(targetSiteID).toBeTruthy();
  const compile = await api(page, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [targetSiteID] }) });
  expect(compile.status, compile.body).toBe(201);
  const candidateID = String(JSON.parse(compile.body)?.revision?.id || "");
  expect(candidateID).toBeTruthy();
  try {
    await page.locator("#revisions-refresh").click();
    const candidate = (await catalog(page)).find((revision) => revision.id === candidateID) as Revision & { sites?: Array<{ site_id?: string }> };
    const siteID = String(candidate?.sites?.[0]?.site_id || targetSiteID);
    expect(siteID).toBeTruthy();
    await page.locator(`[data-revision-site="${siteID}"]`).click();
    await expect(page.locator("#revisions-detail-modal")).toBeVisible();
    const applyButton = page.locator(`[data-revision-apply="${candidateID}"]`);
    await expect(applyButton).toBeEnabled();
    page.once("dialog", (dialog) => dialog.accept());
    await applyButton.click({ timeout: 15_000 });
    await expect.poll(async () => (await catalog(page)).find((revision) => revision.is_active)?.id, { timeout: 120_000 }).toBe(candidateID);
    await expect(page.locator("#revisions-detail-modal")).toContainText(candidateID);

    const rollback = await api(page, `/api/revisions/${encodeURIComponent(original!.id)}/apply`, { method: "POST", body: "{}" });
    expect(rollback.status, rollback.body).toBe(201);
    await expect.poll(async () => (await catalog(page)).find((revision) => revision.is_active)?.id, { timeout: 120_000 }).toBe(original!.id);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#revisions-page")).toBeVisible({ timeout: 30_000 });
    await page.locator(`[data-revision-site="${siteID}"]`).click();
    const deleteButton = page.locator(`[data-revision-delete="${candidateID}"]`);
    await expect(deleteButton).toBeEnabled();
    page.once("dialog", (dialog) => dialog.accept());
    await deleteButton.click({ timeout: 15_000 });
    await expect.poll(async () => (await catalog(page)).some((revision) => revision.id === candidateID), { timeout: 60_000 }).toBe(false);
    await expect(page.locator("#revisions-detail-modal")).not.toContainText(candidateID);
  } finally {
    if ((await catalog(page)).find((revision) => revision.is_active)?.id !== original!.id) {
      await api(page, `/api/revisions/${encodeURIComponent(original!.id)}/apply`, { method: "POST", body: "{}" });
    }
    if ((await catalog(page)).some((revision) => revision.id === candidateID)) {
      await api(page, `/api/revisions/${encodeURIComponent(candidateID)}`, { method: "DELETE" });
    }
  }
});

test("revisions.bulk-delete-others-success", async ({ authenticatedPage: page }) => {
  test.setTimeout(3 * 60_000);
  await page.goto("/revisions", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const activeID = (await catalog(page)).find((revision) => revision.is_active)?.id;
  const sitesResult = await api(page, "/api/sites");
  const sitesPayload = JSON.parse(sitesResult.body);
  const siteID = String((Array.isArray(sitesPayload) ? sitesPayload : sitesPayload?.sites || [])[0]?.id || "");
  expect(siteID).toBeTruthy();
  const ids: string[] = [];
  try {
    for (let index = 0; index < 2; index++) {
      const result = await api(page, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [siteID] }) });
      expect(result.status, result.body).toBe(201);
      ids.push(String(JSON.parse(result.body)?.revision?.id || ""));
    }
    await page.locator("#revisions-refresh").click();
    await page.locator(`[data-revision-site="${siteID}"]`).click();
    await expect(page.locator("#revisions-delete-others")).toBeEnabled();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#revisions-delete-others").click();
    await expect.poll(async () => {
      const items = await catalog(page);
      return ids.every((id) => !items.some((item) => item.id === id));
    }, { timeout: 60_000 }).toBe(true);
    expect((await catalog(page)).find((revision) => revision.is_active)?.id).toBe(activeID);
  } finally {
    for (const id of ids) if ((await catalog(page)).some((item) => item.id === id)) await api(page, `/api/revisions/${encodeURIComponent(id)}`, { method: "DELETE" });
  }
});

test("revisions.readonly-rbac-and-ui-apply-error", async ({ authenticatedPage: adminPage }, testInfo) => {
  const roleID = e2eID(testInfo, "e2e-revisions-viewer");
  const username = e2eID(testInfo, "e2e-revisions-reader");
  const password = "E2e-revisions-reader-1234!";
  const cleanup = new CleanupLedger();
  const activeID = (await catalog(adminPage)).find((revision) => revision.is_active)?.id;
  const sitesPayload = JSON.parse((await api(adminPage, "/api/sites")).body);
  const targetSiteID = String((Array.isArray(sitesPayload) ? sitesPayload : sitesPayload?.sites || [])[0]?.id || "");
  expect(targetSiteID).toBeTruthy();
  const compiled = await api(adminPage, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [targetSiteID] }) });
  expect(compiled.status, compiled.body).toBe(201);
  const candidateID = String(JSON.parse(compiled.body)?.revision?.id || "");
  cleanup.add("candidate " + candidateID, () => api(adminPage, `/api/revisions/${encodeURIComponent(candidateID)}`, { method: "DELETE" }), async () => !(await catalog(adminPage)).some((item) => item.id === candidateID));
  cleanup.add("role " + roleID, () => api(adminPage, `/api/administration/roles/${encodeURIComponent(roleID)}`, { method: "DELETE" }), async () => !JSON.parse((await api(adminPage, "/api/administration/roles")).body)?.roles?.some((item: { id?: string }) => item.id === roleID));
  cleanup.add("user " + username, () => api(adminPage, `/api/administration/users/${encodeURIComponent(username)}`, { method: "DELETE" }), async () => !JSON.parse((await api(adminPage, "/api/administration/users")).body)?.users?.some((item: { id?: string }) => item.id === username));
  let context;
  try {
    expect([200, 201]).toContain((await api(adminPage, "/api/administration/roles", { method: "POST", body: JSON.stringify({ id: roleID, name: "E2E Revisions Reader", permissions: ["revisions.read", "sites.read", "auth.self", "profile.read"] }) })).status);
    expect([200, 201]).toContain((await api(adminPage, "/api/administration/users", { method: "POST", body: JSON.stringify({ id: username, username, email: `${username}@example.test`, password, role_ids: [roleID], is_active: true }) })).status);
    context = await adminPage.context().browser()!.newContext({ ignoreHTTPSErrors: true });
    await context.clearCookies();
    const page = await context.newPage();
    await page.goto("/login", { waitUntil: "domcontentloaded" });
    const login = await api(page, "/api/auth/login", { method: "POST", body: JSON.stringify({ username, password }) });
    expect(login.status, login.body).toBe(200);
    await expect.poll(async () => JSON.parse((await api(page, "/api/auth/me")).body).username).toBe(username);
    await page.goto("/revisions", { waitUntil: "domcontentloaded" });
    const session = await page.evaluate(async () => fetch("/api/auth/me", { credentials: "include" }).then((response) => response.json()));
    expect(session.username).toBe(username);
    expect(session.permissions).toContain("revisions.read");
    expect(session.permissions).not.toContain("revisions.write");
    const candidate = (await catalog(page)).find((item) => item.id === candidateID) as Revision & { sites?: Array<{ site_id?: string }> };
    const siteID = String(candidate?.sites?.[0]?.site_id || targetSiteID);
    expect(siteID).toBeTruthy();
    await page.locator(`[data-revision-site="${siteID}"]`).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator(`[data-revision-apply="${candidateID}"]`).click();
    await expect(page.locator("body")).toContainText(/forbidden|permission|access denied|403/i);
    expect((await catalog(page)).find((revision) => revision.is_active)?.id).toBe(activeID);
    const forbidden = await api(page, `/api/revisions/${encodeURIComponent(candidateID)}/apply`, { method: "POST", body: "{}" });
    expect(forbidden.status, forbidden.body).toBe(403);
  } finally {
    await context?.close();
    await cleanup.run();
  }
});
