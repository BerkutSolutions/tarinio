import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("services.list-load services.search services.sort services.select-all", async ({ authenticatedPage: page }) => {
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#services-refresh")).toBeVisible({ timeout: 15000 });
  const readSites = async () => page.evaluate(async () => {
    const response = await fetch("/api/sites", { credentials: "include", headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`sites readback returned ${response.status}`);
    const payload = await response.json();
    return Array.isArray(payload) ? payload : (payload?.items || []);
  });
  const beforeSites = await readSites();
  const beforeIDs = beforeSites.map((site: { id?: string }) => String(site.id || "")).sort();
  const rows = page.locator("[data-open-site]");
  const before = await rows.count();
  expect(before, "services list requires at least one real seeded service").toBeGreaterThan(0);
  expect(before).toBe(beforeSites.length);
  await page.locator("#services-search").fill("e2e-nonexistent-service");
  await expect(page.locator(".waf-empty")).toBeVisible();
  await page.locator("#services-search").fill("");
  await page.locator("#services-sort").selectOption("name-asc");
  const visibleNames = await page.locator("[data-open-service]").allTextContents();
  const expectedNames = beforeSites
    .map((site: { id?: string; primary_host?: string }) => String(site.primary_host || site.id || ""))
    .sort((left: string, right: string) => left.localeCompare(right));
  expect(visibleNames).toEqual(expectedNames);
  const afterIDs = (await readSites()).map((site: { id?: string }) => String(site.id || "")).sort();
  expect(afterIDs).toEqual(beforeIDs);
  await page.locator("#services-select-all").setChecked(true);
  await expect(page.locator("[data-select-site]:checked")).toHaveCount(before);
  await page.locator("#services-select-all").setChecked(false);
});

test("services.create-route", async ({ authenticatedPage: page }) => {
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#services-create")).toBeVisible({ timeout: 15000 });
  await page.locator("#services-create").click();
  await expect(page).toHaveURL(/\/services\/new$/);
  await expect(page.locator("#service-editor-form")).toBeVisible();
});

test("services.selection-navigation", async ({ authenticatedPage: page }) => {
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  const checkboxes = page.locator("[data-select-site]");
  await expect(checkboxes.first()).toBeVisible({ timeout: 30000 });
  await checkboxes.first().check();
  await expect(checkboxes.first()).toBeChecked();
  await page.locator("#services-select-all").check();
  await expect(page.locator("[data-select-site]:checked")).toHaveCount(await checkboxes.count());
  await page.locator("#services-select-all").uncheck();
  await expect(page.locator("[data-select-site]:checked")).toHaveCount(0);

  const row = page.locator("[data-open-site-edit]").first();
  const siteID = await row.getAttribute("data-open-site-edit");
  await row.click({ position: { x: 300, y: 20 } });
  await expect(page).toHaveURL(new RegExp(`/services/${siteID}$`));
  await expect(page.locator("#service-editor-form")).toBeVisible();
  await page.locator("#service-back").click();
  await expect(page).toHaveURL(/\/services$/);

  const external = page.locator("[data-open-service]").first();
  const expectedURL = await external.getAttribute("data-open-service");
  if (!expectedURL) throw new Error("service external URL is empty");
  await page.context().route(expectedURL, (route) => route.fulfill({ status: 200, contentType: "text/html", body: "<main id='external-service-ready'>external service</main>" }));
  const popupPromise = page.waitForEvent("popup");
  await external.click();
  const popup = await popupPromise;
  await expect.poll(() => popup.url()).toBe(new URL(expectedURL).href);
  await expect(popup.locator("#external-service-ready")).toBeVisible();
  await popup.close();
});

test("services.editor-modes-search-validation-back", async ({ authenticatedPage: page }) => {
  await openPage(page, "/services/new", "#service-editor-form");
  await expect(page.locator("#service-id")).toBeVisible();
  await expect(page.locator("#service-host")).toBeVisible();
  await page.locator('[data-mode-tab="raw"]').click();
  await expect(page.locator("#service-raw-env")).toBeVisible();
  await page.locator('[data-mode-tab="easy"]').click();
  await expect(page.locator("#service-settings-search")).toBeVisible();
  await page.locator("#service-settings-search").fill("rate");
  await expect(page.locator(".waf-service-settings-search-dropdown")).toBeVisible();
  await page.locator("#service-id").fill("");
  await page.locator("#service-editor-form").evaluate((form) => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  await expect(page.locator("#sites-feedback")).toContainText(/required|обяз|нужн/i);
  page.once("dialog", (dialog) => dialog.accept());
  await page.locator("#service-back").click();
  await expect(page).toHaveURL(/\/services$/);
});

test("services.export-invalid-import", async ({ authenticatedPage: page }) => {
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#services-export")).toBeVisible({ timeout: 30000 });
  const downloadPromise = page.waitForEvent("download");
  await page.locator("#services-export").click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("waf-services-export.json");
  const exported = await download.createReadStream();
  expect(exported).toBeTruthy();
  const chunks: Buffer[] = [];
  for await (const chunk of exported!) chunks.push(Buffer.from(chunk));
  const payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  expect(Array.isArray(payload.sites)).toBeTruthy();
  expect(Array.isArray(payload.upstreams)).toBeTruthy();
  expect(Array.isArray(payload.tls_configs)).toBeTruthy();

  await page.locator("#services-import-file").setInputFiles({
    name: "invalid.env",
    mimeType: "text/plain",
    buffer: Buffer.from("THIS_IS_NOT_A_VALID_WAF_PROFILE=true\n"),
  });
  await expect(page.locator("#sites-feedback")).toContainText(/import|импорт|invalid|недопуст/i, { timeout: 30000 });
});

test("services.save-delete-mutation", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-browser");
  const upstreamID = siteID + "-upstream";
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#services-refresh")).toBeVisible({ timeout: 30000 });
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const readSite = async () => {
    const result = await api("/api/sites");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const sites = Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
    return sites.find((site: { id?: string }) => site.id === siteID) || null;
  };
  const readUpstream = async () => {
    const result = await api("/api/upstreams");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const items = Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
    return items.find((item: { id?: string }) => item.id === upstreamID) || null;
  };
  const cleanup = new CleanupLedger();
  cleanup.add("upstream " + upstreamID, () => api("/api/upstreams/" + encodeURIComponent(upstreamID) + "?auto_apply=false", { method: "DELETE" }), async () => (await readUpstream()) === null);
  cleanup.add("site " + siteID, () => api("/api/sites/" + encodeURIComponent(siteID) + "?auto_apply=false", { method: "DELETE" }), async () => (await readSite()) === null);
  try {
    let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: siteID + ".example.test", enabled: true, default_upstream_id: upstreamID }) });
    expect([200, 201]).toContain(result.status);
    result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/" }) });
    expect([200, 201]).toContain(result.status);

    await page.goto("/services/" + encodeURIComponent(siteID), { waitUntil: "domcontentloaded", timeout: 60000 });
    await expect(page.locator("#service-editor-form")).toBeVisible({ timeout: 30000 });
    await expect(page.locator("#service-id")).toHaveValue(siteID);
    await page.locator("#service-host").fill(siteID + ".updated.example.test");
    await page.locator("#service-editor-form button[type=submit]").click();
    await expect.poll(async () => (await readSite())?.primary_host, { timeout: 120000 }).toBe(siteID + ".updated.example.test");
    expect((await readSite())?.id).toBe(siteID);
    await expect.poll(() => new URL(page.url()).pathname, { timeout: 120000 }).toBe("/services/" + siteID);

    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#service-delete").click();
    await expect.poll(() => new URL(page.url()).pathname, { timeout: 120000 }).toBe("/services");
    await expect.poll(async () => await readSite(), { timeout: 120000 }).toBeNull();
  } finally {
    await cleanup.run();
  }
});

test("services.bulk-delete-confirm-cancel", async ({ authenticatedPage: page }, testInfo) => {
  const prefix = e2eID(testInfo, "e2e-bulk");
  const siteIDs = [prefix + "-a", prefix + "-b"];
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const listIDs = async () => {
    const result = await api("/api/sites");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const sites = Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
    return sites.map((site: { id?: string }) => site.id);
  };
  const cleanup = new CleanupLedger();
  for (const siteID of siteIDs) {
    const upstreamID = siteID + "-upstream";
    cleanup.add("upstream " + upstreamID, () => api("/api/upstreams/" + encodeURIComponent(upstreamID) + "?auto_apply=false", { method: "DELETE" }), async () => {
      const result = await api("/api/upstreams");
      const payload = JSON.parse(result.body);
      const items = Array.isArray(payload) ? payload : (Array.isArray(payload?.items) ? payload.items : []);
      return !items.some((item: { id?: string }) => item.id === upstreamID);
    });
    cleanup.add("site " + siteID, () => api("/api/sites/" + encodeURIComponent(siteID) + "?auto_apply=false", { method: "DELETE" }), async () => !(await listIDs()).includes(siteID));
  }
  await page.goto("/services", { waitUntil: "domcontentloaded", timeout: 60000 });
  try {
    for (const siteID of siteIDs) {
      const upstreamID = siteID + "-upstream";
      let result = await api("/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: siteID + ".example.test", enabled: true, default_upstream_id: upstreamID }) });
      expect([200, 201]).toContain(result.status);
      result = await api("/api/upstreams?auto_apply=false", { method: "POST", body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/" }) });
      expect([200, 201]).toContain(result.status);
    }
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.locator("#services-search").fill(prefix);
    await expect(page.locator("[data-select-site]")).toHaveCount(2);
    await page.locator("#services-select-all").setChecked(true);

    page.once("dialog", (dialog) => dialog.dismiss());
    await page.locator("#services-delete-selected").click();
    await expect.poll(async () => (await listIDs()).filter((id: string) => siteIDs.includes(id)).length).toBe(2);

    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#services-delete-selected").click();
    await expect.poll(async () => (await listIDs()).filter((id: string) => siteIDs.includes(id)).length, { timeout: 120000 }).toBe(0);
  } finally {
    await cleanup.run();
  }
});
