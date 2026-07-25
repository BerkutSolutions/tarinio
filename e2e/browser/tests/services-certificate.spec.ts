import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { createHmac } from "node:crypto";
import { gotoWithNetworkRetry, openPage } from "../support/waits";

function totpCode(secret: string) {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const char of secret.toUpperCase().replace(/=+$/, "")) bits += alphabet.indexOf(char).toString(2).padStart(5, "0");
  const key = Buffer.alloc(Math.floor(bits.length / 8));
  for (let index = 0; index < key.length; index += 1) key[index] = Number.parseInt(bits.slice(index * 8, index * 8 + 8), 2);
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000)));
  const digest = createHmac("sha1", key).update(counter).digest();
  const offset = digest[digest.length - 1] & 0x0f;
  return String((digest.readUInt32BE(offset) & 0x7fffffff) % 1_000_000).padStart(6, "0");
}

test("services.certificate-picker-errors-export-binding", async ({ authenticatedPage: page, browser }, testInfo) => {
  test.setTimeout(5 * 60_000);
  const siteID = e2eID(testInfo, "e2e-service-cert");
  const upstreamID = `${siteID}-upstream`;
  const certificateID = `${siteID}-cert`;
  const exporterID = e2eID(testInfo, "e2e-service-exporter");
  const exporterPassword = "e2e-service-exporter-1234";
  const apiFor = async (target: import("@playwright/test").Page, path: string, init: RequestInit = {}) => target.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const api = (path: string, init: RequestInit = {}) => apiFor(page, path, init);
  const list = async (path: string) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    return Array.isArray(payload) ? payload : (payload?.items || payload?.tls_configs || payload?.certificates || []);
  };
  const cleanup = new CleanupLedger();
  cleanup.add(`tls ${siteID}`, () => api(`/api/tls-configs/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/tls-configs")).some((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add(`profile ${siteID}`, () => api(`/api/easy-site-profiles/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/easy-site-profiles")).some((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add(`upstream ${upstreamID}`, () => api(`/api/upstreams/${upstreamID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add(`site ${siteID}`, () => api(`/api/sites/${siteID}?auto_apply=false`, { method: "DELETE" }), async () => !(await list("/api/sites")).some((item: { id?: string }) => item.id === siteID));
  cleanup.add(`certificate ${certificateID}`, () => api(`/api/certificates/${certificateID}`, { method: "DELETE" }), async () => !(await list("/api/certificates")).some((item: { id?: string }) => item.id === certificateID));
  cleanup.add(`user ${exporterID}`, () => api(`/api/administration/users/${exporterID}`, { method: "DELETE" }), async () => !(await list("/api/administration/users")).some((item: { id?: string }) => item.id === exporterID));
  try {
    let result = await api("/api/administration/users", { method: "POST", body: JSON.stringify({ id: exporterID, username: exporterID, email: `${exporterID}@example.test`, password: exporterPassword, role_ids: ["admin"] }) });
    expect(result.status, result.body).toBe(201);
    result = await api("/api/certificates/self-signed/issue", { method: "POST", body: JSON.stringify({ certificate_id: certificateID, common_name: `${siteID}.test` }) });
    expect(result.status, result.body).toBe(201);
    result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: `${siteID}.test`, enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, scheme: "http", host: "upstream-echo", port: 8888 }) });
    expect([200, 201], result.body).toContain(result.status);

    await openPage(page, `/services/${siteID}`, "#service-editor-form");
    await page.locator("#service-ca-server").selectOption("import");
    await expect(page.locator("#service-certificate-import-actions")).toBeVisible();
    await page.locator("#service-import-certificate-search").fill(certificateID);
    await page.locator("#service-import-certificate-search").dispatchEvent("change");
    await expect(page.locator("#service-certificate-id")).toHaveValue(certificateID);

    await page.locator("#service-certificate-archive-file").setInputFiles({ name: "invalid.zip", mimeType: "application/zip", buffer: Buffer.from("not-a-zip") });
    await expect(page.locator("#sites-feedback")).not.toBeEmpty();

    const exporter = await browser.newContext({ ignoreHTTPSErrors: true, acceptDownloads: true });
    const exporterPage = await exporter.newPage();
    try {
      await gotoWithNetworkRetry(exporterPage, `${process.env.WAF_BROWSER_BASE_URL || "https://e2e-management.test:10443"}/login`);
      let exporterResult = await apiFor(exporterPage, "/api/auth/login", { method: "POST", body: JSON.stringify({ username: exporterID, password: exporterPassword }) });
      expect(exporterResult.status, exporterResult.body).toBe(200);
      exporterResult = await apiFor(exporterPage, "/api/auth/2fa/setup", { method: "POST", body: "{}" });
      expect(exporterResult.status, exporterResult.body).toBe(200);
      const setup = JSON.parse(exporterResult.body);
      exporterResult = await apiFor(exporterPage, "/api/auth/2fa/enable", { method: "POST", body: JSON.stringify({ challenge_id: setup.challenge_id, code: totpCode(setup.secret) }) });
      expect(exporterResult.status, exporterResult.body).toBe(200);
      await openPage(exporterPage, `/services/${siteID}`, "#service-editor-form");
      await exporterPage.locator("#service-ca-server").selectOption("import");
      await exporterPage.locator("#service-import-certificate-search").fill(certificateID);
      await exporterPage.locator("#service-import-certificate-search").dispatchEvent("change");
      const approvalResponse = exporterPage.waitForResponse((response) => response.url().endsWith("/api/certificate-materials/export-approvals") && response.request().method() === "POST");
      await exporterPage.locator("#service-certificate-export").click();
      const approval = await (await approvalResponse).json();
      expect(String(approval?.id || "")).not.toBe("");
      const approve = await api(`/api/certificate-materials/export-approvals/${approval.id}/approve`, { method: "POST", body: "{}" });
      expect(approve.status, approve.body).toBe(200);
      await exporterPage.locator("#tls-export-approval-retry").click();
      await exporterPage.locator("#tls-export-totp").fill(totpCode(setup.secret));
      const downloadPromise = exporterPage.waitForEvent("download", { timeout: 30_000 });
      await exporterPage.locator("#tls-export-step-up-submit").click();
      const download = await downloadPromise;
      expect(download.suggestedFilename()).toBe(`${certificateID}-materials.zip`);
    } finally {
      await exporter.close();
    }

    await page.locator("#service-tls-enabled").check();
    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => (await list("/api/tls-configs")).find((item: { site_id?: string }) => item.site_id === siteID)?.certificate_id, { timeout: 120_000 }).toBe(certificateID);
  } finally {
    await cleanup.run();
  }
});
