import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { execFileSync } from "node:child_process";
import { createHmac } from "node:crypto";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

async function list(page: import("@playwright/test").Page, path: string, keys: string[]) {
  const result = await api(page, path);
  expect(result.status, result.body).toBe(200);
  const payload = JSON.parse(result.body);
  if (Array.isArray(payload)) return payload;
  for (const key of keys) if (Array.isArray(payload?.[key])) return payload[key];
  return [];
}

function totpCode(secret: string) {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const char of secret.toUpperCase().replace(/=+$/, "")) bits += alphabet.indexOf(char).toString(2).padStart(5, "0");
  const key = Buffer.alloc(Math.floor(bits.length / 8));
  for (let i = 0; i < key.length; i++) key[i] = Number.parseInt(bits.slice(i * 8, i * 8 + 8), 2);
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000)));
  const digest = createHmac("sha1", key).update(counter).digest();
  const offset = digest[digest.length - 1] & 0x0f;
  return String((digest.readUInt32BE(offset) & 0x7fffffff) % 1_000_000).padStart(6, "0");
}

test("tls.self-signed-selection-detail-validation", async ({ authenticatedPage: page }, testInfo) => {
  const id = e2eID(testInfo, "e2e-self-cert");
  const commonName = `${id}.example.test`;
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const cleanup = new CleanupLedger();
  cleanup.add("certificate " + id, () => api(page, `/api/certificates/${encodeURIComponent(id)}`, { method: "DELETE" }), async () => !(await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === id));
  try {
    const invalid = await api(page, "/api/certificates", { method: "POST", body: JSON.stringify({ id: "bad id", common_name: "not a host", status: "unknown" }) });
    expect(invalid.status).toBe(400);
    const issue = await api(page, "/api/certificates/self-signed/issue", { method: "POST", body: JSON.stringify({ certificate_id: id, common_name: commonName, san_list: [commonName, `www.${commonName}`] }) });
    expect([200, 201], issue.body).toContain(issue.status);
    await expect.poll(async () => (await list(page, "/api/certificates", ["certificates", "items"])).find((item: { id?: string }) => item.id === id)?.common_name, { timeout: 30_000 }).toBe(commonName);
    await page.locator("#certificate-refresh").click();
    const checkbox = page.locator(`[data-cert-select-id="${id}"]`);
    await expect(checkbox).toBeVisible();
    await checkbox.check();
    await expect(checkbox).toBeChecked();
    await page.locator("#tls-cert-select-all").check();
    for (const selected of await page.locator("[data-cert-select-id]").all()) await expect(selected).toBeChecked();
    await page.locator("#tls-cert-select-all").uncheck();
    await expect(checkbox).not.toBeChecked();
  } finally { await cleanup.run(); }
});

test("tls.auto-renew-persistence-and-validation", async ({ authenticatedPage: page }) => {
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const originalResult = await api(page, "/api/tls/auto-renew");
  expect(originalResult.status, originalResult.body).toBe(200);
  const original = JSON.parse(originalResult.body);
  try {
    const nextEnabled = !Boolean(original.enabled);
    const nextDays = Number(original.renew_before_days) === 17 ? 18 : 17;
    await page.locator("#tls-auto-renew-enabled").setChecked(nextEnabled);
    await page.locator("#tls-auto-renew-days").fill(String(nextDays));
    await page.locator("#tls-auto-renew-form button[type=submit]").click();
    await expect.poll(async () => JSON.parse((await api(page, "/api/tls/auto-renew")).body).renew_before_days).toBe(nextDays);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#tls-auto-renew-enabled")).toBeChecked({ checked: nextEnabled });
    await expect(page.locator("#tls-auto-renew-days")).toHaveValue(String(nextDays));
    const invalid = await api(page, "/api/tls/auto-renew", { method: "PUT", body: JSON.stringify({ enabled: true, renew_before_days: 0 }) });
    expect(invalid.status).toBe(400);
  } finally {
    const restore = await api(page, "/api/tls/auto-renew", { method: "PUT", body: JSON.stringify(original) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("tls.acme-conditional-fields-and-disabled-errors", async ({ authenticatedPage: page }, testInfo) => {
  const acmeID = e2eID(testInfo, "e2e-acme");
  const cleanup = new CleanupLedger();
  cleanup.add("ACME certificate " + acmeID, () => api(page, `/api/certificates/${encodeURIComponent(acmeID)}`, { method: "DELETE" }), async () => !(await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === acmeID));
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await page.locator("#le-ca-server").selectOption("custom");
  await expect(page.locator("[data-acme-visible-for=custom]")).toBeVisible();
  await expect(page.locator("#le-use-staging")).toBeDisabled();
  await page.locator("#le-ca-server").selectOption("zerossl");
  await expect(page.locator("#le-zerossl-eab-kid")).toBeVisible();
  await page.locator("#le-challenge-type").selectOption("dns-01");
  await expect(page.locator("#le-dns-provider-env")).toBeVisible();
  await page.locator("#le-certificate-id").fill(acmeID);
  await page.locator("#le-common-name").fill("acme-e2e.example.test");
  await page.locator("#letsencrypt-form button[type=submit]").click();
  await expect(page.locator("body")).toContainText(/EAB|external account|спешн|外部|extern/i);
  try {
    const issued = await api(page, "/api/certificates/acme/issue", { method: "POST", body: JSON.stringify({ certificate_id: acmeID, common_name: `${acmeID}.example.test`, use_lets_encrypt_staging: true }) });
    expect(issued.status, issued.body).toBe(201);
    const renewed = await api(page, `/api/certificates/acme/renew/${encodeURIComponent(acmeID)}`, { method: "POST", body: "{}" });
    expect(renewed.status, renewed.body).toBe(201);
    const missing = await api(page, "/api/certificates/acme/renew/e2e-missing-cert", { method: "POST", body: "{}" });
    expect(missing.status, missing.body).toBe(404);
  } finally { await cleanup.run(); }
});

test("tls.import-export-step-up-errors-and-mobile-layout", async ({ authenticatedPage: page, browser }, testInfo) => {
  const id = e2eID(testInfo, "e2e-export-cert");
  const exporterID = e2eID(testInfo, "e2e-export-user");
  const exporterPassword = "e2e-export-password-1234";
  const cleanup = new CleanupLedger();
  cleanup.add("export certificate " + id, () => api(page, `/api/certificates/${encodeURIComponent(id)}`, { method: "DELETE" }), async () => !(await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === id));
  cleanup.add("export user " + exporterID, () => api(page, `/api/administration/users/${encodeURIComponent(exporterID)}`, { method: "DELETE" }), async () => !(await list(page, "/api/administration/users", ["users", "items"])).some((item: { id?: string }) => item.id === exporterID));
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60_000 });
  try {
    const createExporter = await api(page, "/api/administration/users", { method: "POST", body: JSON.stringify({ id: exporterID, username: exporterID, email: `${exporterID}@example.test`, password: exporterPassword, role_ids: ["admin"] }) });
    expect(createExporter.status, createExporter.body).toBe(201);
    const issue = await api(page, "/api/certificates/self-signed/issue", { method: "POST", body: JSON.stringify({ certificate_id: id, common_name: `${id}.example.test` }) });
    expect(issue.status, issue.body).toBe(201);

    const exporter = await browser.newContext({ ignoreHTTPSErrors: true, acceptDownloads: true });
    const exporterPage = await exporter.newPage();
    await exporterPage.goto("/login", { waitUntil: "domcontentloaded" });
    try {
      const login = await api(exporterPage, "/api/auth/login", { method: "POST", body: JSON.stringify({ username: exporterID, password: exporterPassword }) });
      expect(login.status, login.body).toBe(200);
      const setup = await api(exporterPage, "/api/auth/2fa/setup", { method: "POST", body: "{}" });
      expect(setup.status, setup.body).toBe(200);
      const setupBody = JSON.parse(setup.body);
      const enable = await api(exporterPage, "/api/auth/2fa/enable", { method: "POST", body: JSON.stringify({ challenge_id: setupBody.challenge_id, code: totpCode(setupBody.secret) }) });
      expect(enable.status, enable.body).toBe(200);
      await exporterPage.goto("/tls", { waitUntil: "domcontentloaded" });
      await exporterPage.locator(`[data-cert-select-id="${id}"]`).check();
      const approvalResponse = exporterPage.waitForResponse((response) => response.url().endsWith("/api/certificate-materials/export-approvals") && response.request().method() === "POST");
      await exporterPage.locator("#tls-certificate-export").click();
      await expect(exporterPage.locator("#tls-export-step-up-modal")).toBeVisible();
      const approval = await (await approvalResponse).json();
      expect(String(approval?.id || "")).not.toBe("");
      await expect(exporterPage.locator("#tls-export-approval-id")).toContainText(String(approval.id));
      const approve = await api(page, `/api/certificate-materials/export-approvals/${encodeURIComponent(approval.id)}/approve`, { method: "POST", body: "{}" });
      expect(approve.status, approve.body).toBe(200);
      await exporterPage.locator("#tls-export-approval-retry").click();
      await exporterPage.locator("#tls-export-totp").fill("000000");
      let downloads = 0;
      let archiveRequests = 0;
      exporterPage.on("download", () => { downloads += 1; });
      exporterPage.on("request", (request) => {
        if (request.method() === "POST" && new URL(request.url()).pathname === "/api/certificate-materials/export") archiveRequests += 1;
      });
      const rejectedStepUp = exporterPage.waitForResponse((response) => response.url().endsWith("/api/auth/step-up/totp") && response.request().method() === "POST");
      await exporterPage.locator("#tls-export-step-up-submit").click();
      const rejectedResponse = await rejectedStepUp;
      expect(rejectedResponse.status()).toBe(401);
      expect((await rejectedResponse.json())?.error).toMatch(/invalid TOTP code/i);
      await expect(exporterPage.locator("#tls-export-step-up-status")).toContainText(/invalid TOTP code/i);
      expect(downloads).toBe(0);
      expect(archiveRequests).toBe(0);
      await exporterPage.locator("#tls-export-totp").fill(totpCode(setupBody.secret));
      const downloadPromise = exporterPage.waitForEvent("download");
      await exporterPage.locator("#tls-export-step-up-submit").click();
      const download = await downloadPromise;
      expect(download.suggestedFilename()).toBe(`${id}-materials.zip`);
      expect(archiveRequests).toBe(1);
      await expect(exporterPage.locator("#tls-export-step-up-modal")).toBeHidden();
    } finally {
      await exporter.close();
    }

    await page.locator("#tls-certificate-import").click();
    await page.locator("#tls-certificate-import-archive").setInputFiles({ name: "invalid.zip", mimeType: "application/zip", buffer: Buffer.from("not-a-zip") });
    await expect(page.locator("body")).toContainText(/invalid|zip|archive|архив|存档|Archiv/i);
    await expect(page.locator("#certificate-upload-form")).toBeVisible();
    await expect(page.locator("#letsencrypt-form")).toBeVisible();
  } finally { await cleanup.run(); }
});

test("tls.upload-and-valid-archive-import", async ({ authenticatedPage: page }, testInfo) => {
  const uploadID = e2eID(testInfo, "e2e-upload-cert");
  const archiveID = e2eID(testInfo, "e2e-archive-cert");
  const fixtureDir = resolve("test-results", `certfixture-${testInfo.project.name}-${Date.now()}`);
  mkdirSync(fixtureDir, { recursive: true });
  const generated = execFileSync("go", ["run", "./support/certfixture", fixtureDir, archiveID], { cwd: resolve("."), encoding: "utf8", windowsHide: true }).trim().split(/\r?\n/);
  const [certPath, keyPath, zipPath] = generated;
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const cleanup = new CleanupLedger();
  for (const id of [uploadID, archiveID]) cleanup.add("material certificate " + id, () => api(page, `/api/certificates/${encodeURIComponent(id)}`, { method: "DELETE" }), async () => !(await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === id));
  try {
    await page.locator("#upload-certificate-id").fill(uploadID);
    await page.locator("#upload-common-name").fill(`${uploadID}.example.test`);
    await page.locator("#certificate-file").setInputFiles(certPath);
    await page.locator("#private-key-file").setInputFiles(keyPath);
    await page.locator("#certificate-upload-form button[type=submit]").click();
    await expect.poll(async () => (await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === uploadID), { timeout: 30_000 }).toBe(true);
    await page.locator("#tls-certificate-import-archive").setInputFiles(zipPath);
    await expect.poll(async () => (await list(page, "/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === archiveID), { timeout: 30_000 }).toBe(true);
    const audit = await api(page, "/api/audit?action=certificate.upload&status=succeeded&limit=100");
    expect(audit.status, audit.body).toBe(200);
    expect(audit.body).toContain("certificate.upload");
  } finally { await cleanup.run(); }
});
