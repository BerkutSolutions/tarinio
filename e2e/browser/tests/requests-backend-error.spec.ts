import { existsSync, writeFileSync } from "node:fs";
import { expect, test } from "../fixtures/auth";

test("requests.real-backend-failure-keeps-shell", async ({ authenticatedPage: page }, testInfo) => {
  await page.goto("/requests", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("#requests-refresh")).toBeVisible();
  const syncFile = process.env.WAF_BROWSER_FAULT_SYNC_CONTAINER_FILE;
  if (!syncFile) throw new Error("WAF_BROWSER_FAULT_SYNC_CONTAINER_FILE is required for the real runtime-fault workflow");
  const projectSignal = `${syncFile}.${testInfo.project.name}`;
  writeFileSync(`${projectSignal}.ready`, "ready", "utf8");
  await expect.poll(() => existsSync(`${syncFile}.paused`), { timeout: 60_000 }).toBe(true);
  await page.locator("#requests-refresh").click();
  await expect(page.locator(".waf-empty")).toContainText(/error|ошиб|fehler|греш|错误/i, { timeout: 30_000 });
  await expect(page.locator("nav")).toBeVisible();
  await expect(page.locator("#app-shell, main, #app").first()).toBeVisible();
});
