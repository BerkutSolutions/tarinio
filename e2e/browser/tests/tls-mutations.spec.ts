import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";

test("tls.certificate-binding-create-delete", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-tls-site");
  const upstreamID = siteID + "-upstream";
  const certificateID = e2eID(testInfo, "e2e-cert");
  const commonName = siteID + ".example.test";
  await page.goto("/tls", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#certificate-form")).toBeVisible({ timeout: 30000 });
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const list = async (path: string, keys: string[]) => {
    const result = await api(path);
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    if (Array.isArray(payload)) return payload;
    for (const key of keys) if (Array.isArray(payload?.[key])) return payload[key];
    return [];
  };
  const cleanup = new CleanupLedger();
  cleanup.add("upstream " + upstreamID, () => api("/api/upstreams/" + encodeURIComponent(upstreamID) + "?auto_apply=false", { method: "DELETE" }), async () => !(await list("/api/upstreams", ["upstreams", "items"])).some((item: { id?: string }) => item.id === upstreamID));
  cleanup.add("site " + siteID, () => api("/api/sites/" + encodeURIComponent(siteID) + "?auto_apply=false", { method: "DELETE" }), async () => !(await list("/api/sites", ["sites", "items"])).some((item: { id?: string }) => item.id === siteID));
  cleanup.add("certificate " + certificateID, () => api("/api/certificates/" + encodeURIComponent(certificateID), { method: "DELETE" }), async () => !(await list("/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === certificateID));
  cleanup.add("TLS binding " + siteID, () => api("/api/tls-configs/" + encodeURIComponent(siteID), { method: "DELETE" }), async () => !(await list("/api/tls-configs", ["tls_configs", "items"])).some((item: { site_id?: string }) => item.site_id === siteID));
  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: commonName, enabled: true, default_upstream_id: upstreamID }) });
    expect([200, 201]).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/" }) });
    expect([200, 201]).toContain(result.status);

    await page.locator("#certificate-id").fill(certificateID);
    await page.locator("#certificate-common-name").fill(commonName);
    await page.locator("#certificate-san-list").fill(commonName + String.fromCharCode(10) + "www." + commonName);
    await page.locator("#certificate-status").selectOption("active");
    await page.locator("#certificate-form button[type=submit]").click();
    await expect.poll(async () => (await list("/api/certificates", ["certificates", "items"])).find((item: { id?: string }) => item.id === certificateID)?.common_name, { timeout: 30000 }).toBe(commonName);

    await page.locator("#tls-site-id").fill(siteID);
    await page.locator("#tls-certificate-id").fill(certificateID);
    await page.locator("#tls-config-form button[type=submit]").click();
    await expect.poll(async () => (await list("/api/tls-configs", ["tls_configs", "items"])).find((item: { site_id?: string }) => item.site_id === siteID)?.certificate_id, { timeout: 30000 }).toBe(certificateID);

    await page.locator("#tls-site-id").fill(siteID);
    await page.locator("#tls-config-delete").click();
    await expect.poll(async () => (await list("/api/tls-configs", ["tls_configs", "items"])).some((item: { site_id?: string }) => item.site_id === siteID), { timeout: 30000 }).toBe(false);

    await page.locator("#certificate-id").fill(certificateID);
    await page.locator("#certificate-delete").click();
    await expect.poll(async () => (await list("/api/certificates", ["certificates", "items"])).some((item: { id?: string }) => item.id === certificateID), { timeout: 30000 }).toBe(false);
  } finally {
    await cleanup.run();
  }
});
