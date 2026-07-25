import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.valid-import-cleanup", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-import");
  const upstreamID = siteID + "-upstream";
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const list = async (path: string) => {
    const response = await api(path);
    expect(response.status, path).toBe(200);
    const payload = JSON.parse(response.body);
    return Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
  };
  const siteExists = async () => (await list("/api/sites")).some((item: { id?: string }) => item.id === siteID);
  const upstreamExists = async () => (await list("/api/upstreams")).some((item: { id?: string }) => item.id === upstreamID);
  const cleanup = new CleanupLedger();
  cleanup.add("upstream " + upstreamID, () => api("/api/upstreams/" + encodeURIComponent(upstreamID) + "?auto_apply=false", { method: "DELETE" }), async () => !(await upstreamExists()));
  cleanup.add("site " + siteID, () => api("/api/sites/" + encodeURIComponent(siteID) + "?auto_apply=false", { method: "DELETE" }), async () => !(await siteExists()));

  try {
    let response = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({
      id: siteID, primary_host: siteID + ".example.test", enabled: true, listen_http: true, listen_https: false,
      use_easy_config: true, default_upstream_id: upstreamID,
    }) });
    expect([200, 201]).toContain(response.status);
    response = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({
      id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/",
    }) });
    expect([200, 201]).toContain(response.status);

    await openPage(page, "/services", "#services-search");
    await page.locator("#services-search").fill(siteID);
    const selected = page.locator(`[data-select-site="${siteID}"]`);
    await expect(selected).toBeVisible({ timeout: 30000 });
    await selected.check();
    const downloadPromise = page.waitForEvent("download");
    await page.locator("#services-export").click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.env$/);
    const stream = await download.createReadStream();
    if (!stream) throw new Error("selected service ENV download stream is unavailable");
    const chunks: Buffer[] = [];
    for await (const chunk of stream) chunks.push(Buffer.from(chunk));
    const env = Buffer.concat(chunks);
    expect(env.toString("utf8")).toContain(`WAF_SITE_ID=${siteID}`);

    response = await api("/api/sites/" + encodeURIComponent(siteID) + "?auto_apply=false", { method: "DELETE" });
    expect([200, 204]).toContain(response.status);
    response = await api("/api/upstreams/" + encodeURIComponent(upstreamID) + "?auto_apply=false", { method: "DELETE" });
    expect([200, 204, 404]).toContain(response.status);
    await expect.poll(siteExists).toBe(false);

    await openPage(page, "/services", "#services-import");
    const chooserPromise = page.waitForEvent("filechooser");
    await page.locator("#services-import").click();
    const chooser = await chooserPromise;
    await chooser.setFiles([]);
    await expect(page).toHaveURL(/\/services$/);
    await expect.poll(siteExists).toBe(false);

    await page.locator("#services-import-file").setInputFiles({ name: siteID + ".env", mimeType: "text/plain", buffer: env });
    await expect.poll(siteExists, { timeout: 120000 }).toBe(true);
    await expect.poll(upstreamExists, { timeout: 120000 }).toBe(true);
    await expect(page).toHaveURL(new RegExp(`/services/new$`));
    await expect(page.locator("#service-editor-form #service-id")).toBeVisible();
    await expect(page.locator("#service-id")).toHaveValue(siteID);
  } finally {
    await cleanup.run();
  }
});
