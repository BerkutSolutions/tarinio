import { expect, test as setup } from "@playwright/test";
import path from "node:path";
import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { requiredE2EEnv } from "../support/env";

const testDirectory = path.dirname(fileURLToPath(import.meta.url));
const storageStatePath = path.resolve(testDirectory, "../.auth/storage-state.json");

setup("authenticate", async ({ page }) => {
  mkdirSync(path.dirname(storageStatePath), { recursive: true });
  const username = requiredE2EEnv("WAF_E2E_USERNAME");
  const password = requiredE2EEnv("WAF_E2E_PASSWORD");
  await page.goto("/login", { waitUntil: "domcontentloaded", timeout: 60000 });
  const status = await page.evaluate(async ({ username: user, password: pass }) => {
    const response = await fetch("/api/auth/login", {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ username: user, password: pass }),
    });
    return response.status;
  }, { username, password });
  expect(status).toBe(200);
  await page.context().storageState({ path: storageStatePath });
});
