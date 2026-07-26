import { expect, test } from "../fixtures/auth";
import { CleanupLedger, e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

test("bans.create-extend-unban-mutation", async ({ authenticatedPage: page }, testInfo) => {
  const suffix = Number.parseInt(e2eID(testInfo, "e2e-ban").replace(/[^0-9]/g, "").slice(-3) || "20", 10);
  const ip = "203.0.113." + (20 + (suffix % 180));
  await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#bans-create")).toBeVisible({ timeout: 30000 });

  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const isDenied = async (siteID: string) => {
    const result = await api("/api/access-policies");
    expect(result.status).toBe(200);
    const payload = JSON.parse(result.body);
    const policies = Array.isArray(payload) ? payload : (payload?.access_policies || payload?.items || []);
    const policy = policies.find((item: { site_id?: string }) => item.site_id === siteID);
    return Array.isArray(policy?.denylist) && policy.denylist.includes(ip);
  };
  const cleanup = new CleanupLedger();

  await page.locator("#bans-create").click();
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  const siteSelect = page.locator("#bans-create-site");
  const siteID = await siteSelect.locator("option:not([value='__all__'])").first().getAttribute("value");
  expect(siteID).toBeTruthy();
  try {
    await siteSelect.selectOption(siteID!);
    await page.locator("#bans-create-ip").fill(ip);
    await page.locator("#bans-create-duration").selectOption("3600");
    await page.locator("#bans-create-submit").click();
    await expect(page.locator("#bans-create-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(() => isDenied(siteID!), { timeout: 30000 }).toBe(true);

    const row = page.locator("[data-ban-row]", { hasText: ip }).first();
    await expect(row).toBeVisible({ timeout: 30000 });
    await row.locator("[data-action='extend']").click();
    await expect(page.locator("#bans-extend-modal")).toBeVisible();
    await expect(page.locator("#bans-extend-ip")).toHaveValue(ip);
    await page.locator("#bans-extend-duration").selectOption("3600");
    await page.locator("#bans-extend-submit").click();
    await expect(page.locator("#bans-extend-modal")).toBeHidden({ timeout: 30000 });
    await expect.poll(() => isDenied(siteID!), { timeout: 30000 }).toBe(true);

    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("[data-ban-row]", { hasText: ip }).first().locator("[data-action='unban']").click();
    await expect.poll(() => isDenied(siteID!), { timeout: 30000 }).toBe(false);
    await expect(page.locator("[data-ban-row]", { hasText: ip })).toHaveCount(0);

    const audit = await api("/api/audit?action=accesspolicy.unban&status=succeeded&limit=500");
    expect(audit.status).toBe(200);
    expect(audit.body).toContain("accesspolicy.unban");
  } finally {
    if (siteID) {
      cleanup.add("ban " + ip, () => api("/api/sites/" + encodeURIComponent(siteID) + "/unban", { method: "POST", body: JSON.stringify({ ip }) }), async () => !(await isDenied(siteID)));
      await cleanup.run();
    }
  }
});

test("bans.detail-country-keyboard-pagination-cancel", async ({ authenticatedPage: page }, testInfo) => {
  const siteID = e2eID(testInfo, "e2e-bans-pages");
  const policyID = `${siteID}-access`;
  const host = `${siteID}.example.test`;
  const ips = Array.from({ length: 12 }, (_, index) => `198.51.100.${80 + index}`);
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
    });
    return { status: response.status, body: await response.text() };
  }, { path, init });
  const list = async (path: string, keys: string[] = []) => {
    const result = await api(path);
    expect(result.status, result.body).toBe(200);
    const payload = JSON.parse(result.body);
    if (Array.isArray(payload)) return payload;
    for (const key of keys) if (Array.isArray(payload?.[key])) return payload[key];
    throw new Error(`${path} returned an unsupported list payload: ${result.body}`);
  };
  const policy = async () => (await list("/api/access-policies", ["access_policies", "items"]))
    .find((item: { id?: string }) => item.id === policyID);
  const cleanup = new CleanupLedger();
  cleanup.add(`site ${siteID}`, () => api(`/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" }), async () =>
    !(await list("/api/sites", ["sites", "items"])).some((item: { id?: string }) => item.id === siteID));
  cleanup.add(`access policy ${policyID}`, () => api(`/api/access-policies/${encodeURIComponent(policyID)}?auto_apply=false`, { method: "DELETE" }), async () =>
    !(await list("/api/access-policies", ["access_policies", "items"])).some((item: { id?: string }) => item.id === policyID));

  try {
    const createdSite = await api("/api/sites?auto_apply=false", {
      method: "POST", body: JSON.stringify({ id: siteID, primary_host: host, enabled: true }),
    });
    expect(createdSite.status, createdSite.body).toBe(201);
    const createdPolicy = await api("/api/access-policies?auto_apply=false", {
      method: "POST", body: JSON.stringify({ id: policyID, site_id: siteID, enabled: true, denylist: ips }),
    });
    expect(createdPolicy.status, createdPolicy.body).toBe(201);
    await expect.poll(async () => (await policy())?.denylist, { timeout: 30_000 }).toEqual(ips);

    await openPage(page, "/bans", page.locator("#bans-status"));
    await page.locator("#bans-filter").fill(siteID);
    await expect(page.locator("#bans-status")).toContainText("12");
    await expect(page.locator("#bans-page-size")).toHaveValue("10");
    await expect(page.locator("[data-ban-row]")).toHaveCount(10);

    const targetRow = page.locator("[data-ban-row]", { hasText: ips[0] });
    await expect(targetRow).toBeVisible();
    await targetRow.focus();
    await targetRow.press("Enter");
    await expect(page.locator("#bans-detail-modal")).toBeVisible();
    const detail = page.locator("#bans-detail-content");
    await expect(detail.locator("tr")).toHaveCount(8);
    await expect(detail).toContainText(ips[0]);
    await expect(detail).toContainText(host);
    await expect(detail).toContainText(/IP|country|страна|国家|Land|zemlja/i);
    await page.locator("#bans-detail-modal").press("Escape");
    await expect(page.locator("#bans-detail-modal")).toBeHidden();

    await targetRow.focus();
    await targetRow.press(" ");
    await expect(page.locator("#bans-detail-modal")).toBeVisible();
    await page.locator("#bans-detail-modal [data-bans-detail-close]").last().click();
    await expect(page.locator("#bans-detail-modal")).toBeHidden();

    await targetRow.locator("[data-action='extend']").click();
    await expect(page.locator("#bans-extend-modal")).toBeVisible();
    await expect(page.locator("#bans-extend-ip")).toHaveValue(ips[0]);
    await page.locator("#bans-extend-modal [data-bans-extend-close]").last().click();
    await expect(page.locator("#bans-extend-modal")).toBeHidden();

    page.once("dialog", (dialog) => dialog.dismiss());
    await targetRow.locator("[data-action='unban']").click();
    await expect(targetRow).toBeVisible();
    expect((await policy())?.denylist).toContain(ips[0]);

    await page.locator("#bans-filter").fill(ips[0]);
    await expect(page.locator("[data-ban-row]")).toHaveCount(1);
    await expect(page.locator("[data-ban-row]")).toContainText(ips[0]);
    await page.locator("#bans-filter").fill("e2e-filter-no-match");
    await expect(page.locator("[data-ban-row]")).toHaveCount(0);

    await page.locator("#bans-filter").fill(siteID);
    await expect(page.locator("[data-ban-row]")).toHaveCount(10);
    const pageTwo = page.locator("[data-bans-page='2']");
    await expect(pageTwo).toBeVisible();
    await pageTwo.focus();
    await pageTwo.press("Enter");
    await expect(page.locator("[data-bans-page='2']")).toHaveClass(/active/);
    await expect(page.locator("[data-ban-row]")).toHaveCount(2);
    const secondPageText = (await page.locator("[data-ban-row]").allTextContents()).join("\n");
    expect(ips.filter((ip) => secondPageText.includes(ip))).toHaveLength(2);
  } finally {
    await cleanup.run();
  }
});

test("bans.duration-and-unauthorized-site-errors", async ({ authenticatedPage: page }) => {
  await page.goto("/bans", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const invoke = async (path: string, body: unknown) => page.evaluate(async ({ path, body }) => {
    const response = await fetch(path, { method: "POST", credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(body) });
    return { status: response.status, text: await response.text() };
  }, { path, body });
  const missing = await invoke("/api/sites/e2e-missing-site/ban", { ip: "203.0.113.199" });
  expect(missing.status, missing.text).toBe(404);
  expect(JSON.parse(missing.text)?.error).toMatch(/not found/i);

  const readPolicies = async () => page.evaluate(async () => {
    const response = await fetch("/api/access-policies", { credentials: "include", headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`access policies readback returned ${response.status}`);
    return response.json();
  });
  const policiesBefore = await readPolicies();
  let banRequests = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && /\/api\/sites\/[^/]+\/ban$/.test(new URL(request.url()).pathname)) banRequests += 1;
  });

  await page.locator("#bans-create").click();
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  await page.locator("#bans-create-ip").fill("203.0.113.199");
  await page.locator("#bans-create-duration").evaluate((node) => {
    const select = node as HTMLSelectElement;
    select.innerHTML += '<option value="-1">invalid</option>';
    select.value = "-1";
  });
  await page.locator("#bans-create-submit").click();
  await expect(page.locator("#bans-create-status")).toContainText(/valid time|корректное время|gültige Zeit|важеће време|有效时间/i);
  await expect(page.locator("#bans-create-modal")).toBeVisible();
  expect(banRequests).toBe(0);
  expect(await readPolicies()).toEqual(policiesBefore);
  await page.locator("#bans-create-modal [data-bans-create-close]").last().click();
});
