import { expect, test as base, type Page } from "@playwright/test";
import { requiredE2EEnv } from "../support/env";
import { gotoWithNetworkRetry } from "../support/waits";

type Fixtures = { authenticatedPage: Page };

async function activeRevisionID(page: Page) {
  return page.evaluate(async () => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30_000);
    try {
      const response = await fetch("/api/revisions", { credentials: "include", signal: controller.signal, headers: { Accept: "application/json" } });
      if (!response.ok) throw new Error(`revision catalog returned ${response.status}`);
      const payload = await response.json();
      const revisions = Array.isArray(payload?.revisions) ? payload.revisions : [];
      return String(revisions.find((item: { is_active?: boolean; id?: string }) => item.is_active)?.id || "");
    } finally {
      window.clearTimeout(timer);
    }
  });
}

async function ensureAuthenticated(page: Page) {
  let status = 0;
  await expect.poll(async () => {
    try {
      status = await page.evaluate(async () => fetch("/api/auth/me", { credentials: "include" }).then((response) => response.status));
      return status === 200 || status === 401;
    } catch {
      return false;
    }
  }, { timeout: 30_000 }).toBe(true);
  if (status === 200) return;
  expect(status).toBe(401);
  const username = requiredE2EEnv("WAF_E2E_USERNAME");
  const password = requiredE2EEnv("WAF_E2E_PASSWORD");
  let login = { status: 0, body: "login request did not complete" };
  await expect.poll(async () => {
    try {
      login = await page.evaluate(async ({ username, password }) => {
        const response = await fetch("/api/auth/login", { method: "POST", credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify({ username, password }) });
        return { status: response.status, body: await response.text() };
      }, { username, password });
      return login.status;
    } catch {
      return 0;
    }
  }, { timeout: 30_000 }).toBe(200);
  expect(login.status, login.body).toBe(200);
}

async function restoreActiveRevision(page: Page, revisionID: string) {
  const response = await page.evaluate(async (id) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 120_000);
    try {
      const result = await fetch(`/api/revisions/${encodeURIComponent(id)}/apply`, { method: "POST", credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json" }, body: "{}" });
      return { status: result.status, body: await result.text() };
    } finally {
      window.clearTimeout(timer);
    }
  }, revisionID);
  expect([200, 201], response.body).toContain(response.status);
  await expect.poll(() => activeRevisionID(page).catch(() => ""), { timeout: 120_000 }).toBe(revisionID);
}

export const test = base.extend<Fixtures>({
  authenticatedPage: async ({ page }, use) => {
    await gotoWithNetworkRetry(page, "/login");
    await page.context().clearCookies();
    await ensureAuthenticated(page);
    const originalRevisionID = await activeRevisionID(page);
    try {
      await use(page);
    } finally {
      const currentRevisionID = await activeRevisionID(page).catch(() => "");
      if (originalRevisionID && currentRevisionID !== originalRevisionID) {
        await ensureAuthenticated(page);
        await restoreActiveRevision(page, originalRevisionID);
      } else {
        expect(currentRevisionID).toBe(originalRevisionID);
      }
    }
  },
});

export { expect };
