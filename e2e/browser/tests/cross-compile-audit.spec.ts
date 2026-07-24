import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";

type Revision = { id: string; is_active?: boolean };
async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 120_000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
}
async function revisions(page: import("@playwright/test").Page): Promise<Revision[]> {
  const result = await api(page, "/api/revisions");
  expect(result.status, result.body).toBe(200);
  return JSON.parse(result.body)?.revisions || [];
}

test("cross-module.settings-services-tls-antiddos-compile-apply-audit", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(6 * 60_000);
  await page.goto("/revisions", { waitUntil: "domcontentloaded" });
  const siteID = e2eID(testInfo, "e2e-cross-site");
  const upstreamID = siteID + "-upstream";
  const certificateID = siteID + "-cert";
  const host = `${siteID}.example.test`;
  const originalActive = (await revisions(page)).find((item) => item.is_active)?.id || "";
  expect(originalActive).toBeTruthy();
  const originalSettings = JSON.parse((await api(page, "/api/settings/runtime")).body);
  const originalAntiDDoS = JSON.parse((await api(page, "/api/anti-ddos/settings")).body);
  const cleanup = new CleanupLedger();
  let candidateID = "";
  cleanup.add("tls binding " + siteID, () => api(page, `/api/tls-configs/${encodeURIComponent(siteID)}`, { method: "DELETE" }), async () => !JSON.parse((await api(page, "/api/tls-configs")).body)?.some?.((item: { site_id?: string }) => item.site_id === siteID));
  cleanup.add("certificate " + certificateID, () => api(page, `/api/certificates/${encodeURIComponent(certificateID)}`, { method: "DELETE" }), async () => !JSON.parse((await api(page, "/api/certificates")).body)?.some?.((item: { id?: string }) => item.id === certificateID));
  cleanup.add("upstream " + upstreamID, () => api(page, `/api/upstreams/${encodeURIComponent(upstreamID)}?auto_apply=false`, { method: "DELETE" }), async () => !JSON.parse((await api(page, "/api/upstreams")).body)?.some?.((item: { id?: string }) => item.id === upstreamID));
  cleanup.add("site " + siteID, () => api(page, `/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" }), async () => !JSON.parse((await api(page, "/api/sites")).body)?.some?.((item: { id?: string }) => item.id === siteID));
  try {
    let result = await api(page, "/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: host, enabled: true, listen_http: true, listen_https: true, use_easy_config: true, default_upstream_id: upstreamID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api(page, "/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/" }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api(page, "/api/certificates/self-signed/issue", { method: "POST", body: JSON.stringify({ certificate_id: certificateID, common_name: host, san_list: [host] }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api(page, "/api/tls-configs?auto_apply=false", { method: "POST", body: JSON.stringify({ site_id: siteID, certificate_id: certificateID }) });
    expect([200, 201], result.body).toContain(result.status);
    result = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ update_checks_enabled: !originalSettings.update_checks_enabled }) });
    expect(result.status, result.body).toBe(200);
    result = await api(page, "/api/anti-ddos/settings", { method: "PUT", body: JSON.stringify({ ...originalAntiDDoS, model_enabled: !Boolean(originalAntiDDoS.model_enabled) }) });
    expect(result.status, result.body).toBe(200);

    const compile = await api(page, "/api/revisions/compile", { method: "POST", body: JSON.stringify({ target_site_ids: [siteID] }) });
    expect(compile.status, compile.body).toBe(201);
    candidateID = String(JSON.parse(compile.body)?.revision?.id || "");
    expect(candidateID).toBeTruthy();
    const apply = await api(page, `/api/revisions/${encodeURIComponent(candidateID)}/apply`, { method: "POST", body: "{}" });
    expect(apply.status, apply.body).toBe(201);
    await expect.poll(async () => (await revisions(page)).find((item) => item.is_active)?.id, { timeout: 120_000 }).toBe(candidateID);
    const rollback = await api(page, `/api/revisions/${encodeURIComponent(originalActive)}/apply`, { method: "POST", body: "{}" });
    expect(rollback.status, rollback.body).toBe(201);
    await expect.poll(async () => (await revisions(page)).find((item) => item.is_active)?.id, { timeout: 120_000 }).toBe(originalActive);

    for (const action of ["site.create", "upstream.create", "antiddos.settings.upsert", "revision.compile_request", "revision.apply_trigger"]) {
      const audit = JSON.parse((await api(page, `/api/audit?action=${encodeURIComponent(action)}&limit=100`)).body);
      expect(audit.items, action).toEqual(expect.arrayContaining([expect.objectContaining({ action, status: "succeeded" })]));
    }
  } finally {
    if ((await revisions(page)).find((item) => item.is_active)?.id !== originalActive) await api(page, `/api/revisions/${encodeURIComponent(originalActive)}/apply`, { method: "POST", body: "{}" });
    if (candidateID && (await revisions(page)).some((item) => item.id === candidateID)) await api(page, `/api/revisions/${encodeURIComponent(candidateID)}`, { method: "DELETE" });
    await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ update_checks_enabled: originalSettings.update_checks_enabled }) });
    await api(page, "/api/anti-ddos/settings", { method: "PUT", body: JSON.stringify(originalAntiDDoS) });
    await cleanup.run();
  }
});
