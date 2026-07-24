import { expect, test } from "../fixtures/auth";
import { e2eID } from "../support/isolation";

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}) {
  return page.evaluate(async ({ path, init }) => {
    const response = await fetch(path, { ...init, credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
    return { status: response.status, body: await response.text() };
  }, { path, init });
}

test("settings.management-hosts-validation-status-restore", async ({ authenticatedPage: page }, testInfo) => {
  await page.goto("/settings/management-hosts", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  const original = JSON.parse((await api(page, "/api/settings/management-hosts")).body);
  const siteID = e2eID(testInfo, "e2e-management-host");
  const host = `${siteID}.example.test`;
  let current = original;
  try {
    const site = await api(page, "/api/sites?auto_apply=false", { method: "POST", body: JSON.stringify({ id: siteID, primary_host: host, enabled: true, listen_http: true }) });
    expect([200, 201], site.body).toContain(site.status);
    await expect(page.locator("#settings-management-hosts-status")).not.toBeEmpty();
    await expect(page.locator("#settings-management-hosts")).toHaveValue(original.management_hosts.join("\n"));
    const invalid = await api(page, "/api/settings/management-hosts", { method: "PUT", body: JSON.stringify({ management_hosts: ["not a host"], version: original.version }) });
    expect(invalid.status, invalid.body).toBe(400);
    const unowned = await api(page, "/api/settings/management-hosts", { method: "PUT", body: JSON.stringify({ management_hosts: [...original.management_hosts, "unowned-e2e.example.test"], version: original.version }) });
    expect(unowned.status, unowned.body).toBe(400);
    const stale = await api(page, "/api/settings/management-hosts", { method: "PUT", body: JSON.stringify({ management_hosts: original.management_hosts, version: Math.max(0, Number(original.version) - 1) }) });
    if (Number(original.version) > 0) expect(stale.status, stale.body).toBe(409);
    const intendedHosts = [...original.management_hosts, host];
    await page.locator("#settings-management-hosts").fill(intendedHosts.join("\n"));
    await expect(page.locator("#settings-management-hosts")).toHaveValue(intendedHosts.join("\n"));
    const saveResponse = page.waitForResponse((response) => {
      if (!response.url().endsWith("/api/settings/management-hosts") || response.request().method() !== "PUT") return false;
      return response.request().postDataJSON()?.management_hosts?.includes(host) === true;
    });
    await page.locator("#settings-management-hosts-save").click();
    const savedResponse = await saveResponse;
    expect(savedResponse.request().postDataJSON()).toMatchObject({ management_hosts: intendedHosts });
    expect(savedResponse.status(), await savedResponse.text()).toBe(200);
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/management-hosts")).body).management_hosts).toContain(host);
    current = JSON.parse((await api(page, "/api/settings/management-hosts")).body);
    expect(Number(current.version)).toBeGreaterThan(Number(original.version));
    await expect(page.locator("#settings-alert")).not.toBeEmpty();
    const status = await api(page, "/api/settings/management-hosts/status");
    expect(status.status, status.body).toBe(200);
    const statusBody = JSON.parse(status.body);
    expect(statusBody).toHaveProperty("active_revision_id");
    expect(statusBody).toHaveProperty("drift");
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    await expect(page.locator("#settings-management-hosts")).toHaveValue(new RegExp(host.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  } finally {
    current = JSON.parse((await api(page, "/api/settings/management-hosts")).body);
    const restore = await api(page, "/api/settings/management-hosts", { method: "PUT", body: JSON.stringify({ management_hosts: original.management_hosts, version: current.version }) });
    expect(restore.status, restore.body).toBe(200);
    const deletion = await api(page, `/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`, { method: "DELETE" });
    expect([200, 204, 404], deletion.body).toContain(deletion.status);
  }
});

test("settings.appearance-preview-save-reload-restore", async ({ authenticatedPage: page }) => {
  await page.goto("/settings/general", { waitUntil: "domcontentloaded" });
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  const original = JSON.parse((await api(page, "/api/settings/runtime")).body);
  const login = original.login_appearance === "security-card" ? "incident-console" : "security-card";
  const health = original.healthcheck_appearance === "variant-3" ? "variant-4" : "variant-3";
  try {
    await expect(page.locator("#settings-login-appearance")).toHaveValue(original.login_appearance);
    await page.locator("#settings-login-appearance").selectOption(login);
    await expect(page.locator("#settings-login-appearance")).toHaveValue(login);
    const loginPopup = page.waitForEvent("popup");
    await page.locator("#settings-login-appearance-preview").click();
    const loginPage = await loginPopup;
    expect(loginPage.url()).toContain(`/api/login-appearance/preview/${login}?screen=login`);
    await loginPage.close();
    const twoFactorPopup = page.waitForEvent("popup");
    await page.locator("#settings-login-appearance-preview-2fa").click();
    const twoFactorPage = await twoFactorPopup;
    expect(twoFactorPage.url()).toContain(`screen=2fa`);
    await twoFactorPage.close();

    // Opening/closing a preview can overlap the runtime re-render on slower
    // mobile projects. Re-apply the intended value immediately before save
    // and synchronize on the PUT carrying that exact value.
    await page.locator("#settings-login-appearance").selectOption(login);
    await expect(page.locator("#settings-login-appearance")).toHaveValue(login);
    const loginSaveResponsePromise = page.waitForResponse((response) => {
      if (!response.url().endsWith("/api/settings/runtime") || response.request().method() !== "PUT") return false;
      return response.request().postDataJSON()?.login_appearance === login;
    });
    await page.locator("#settings-login-appearance-save").click();
    const loginSaveResponse = await loginSaveResponsePromise;
    expect(loginSaveResponse.request().postDataJSON()).toMatchObject({ login_appearance: login });
    expect(loginSaveResponse.status(), await loginSaveResponse.text()).toBe(200);
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/runtime")).body).login_appearance).toBe(login);

    await expect(page.locator("#settings-healthcheck-appearance")).toHaveValue(original.healthcheck_appearance);
    await page.locator("#settings-healthcheck-appearance").selectOption(health);
    const healthPopup = page.waitForEvent("popup");
    await page.locator("#settings-healthcheck-appearance-preview").click();
    const healthPage = await healthPopup;
    expect(healthPage.url()).toContain(`/healthcheck?appearance=${health}`);
    await healthPage.close();

    await page.locator("#settings-healthcheck-appearance").selectOption(health);
    await expect(page.locator("#settings-healthcheck-appearance")).toHaveValue(health);
    const healthSaveResponsePromise = page.waitForResponse((response) => {
      if (!response.url().endsWith("/api/settings/runtime") || response.request().method() !== "PUT") return false;
      return response.request().postDataJSON()?.healthcheck_appearance === health;
    });
    await page.locator("#settings-healthcheck-appearance-save").click();
    const healthSaveResponse = await healthSaveResponsePromise;
    expect(healthSaveResponse.request().postDataJSON()).toMatchObject({ healthcheck_appearance: health });
    expect(healthSaveResponse.status(), await healthSaveResponse.text()).toBe(200);
    await expect.poll(async () => JSON.parse((await api(page, "/api/settings/runtime")).body).healthcheck_appearance).toBe(health);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    await expect(page.locator("#settings-login-appearance")).toHaveValue(login);
    await expect(page.locator("#settings-healthcheck-appearance")).toHaveValue(health);
  } finally {
    const restore = await api(page, "/api/settings/runtime", { method: "PUT", body: JSON.stringify({ login_appearance: original.login_appearance, healthcheck_appearance: original.healthcheck_appearance }) });
    expect(restore.status, restore.body).toBe(200);
  }
});

test("settings.update-check-success-disabled-offline", async ({ authenticatedPage: source }) => {
  await source.goto("/settings/general", { waitUntil: "domcontentloaded" });
  await expect(source.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  await source.locator("#settings-update-check").click();
  await expect(source.locator("#settings-update-status")).not.toBeEmpty();
  for (const status of [200, 503]) {
      const page = await source.context().newPage();
    try {
      await page.route("**/api/settings/runtime/check-updates", (route) => route.fulfill({ status, contentType: "application/json", body: status === 200 ? '{"update_checks_enabled":false,"update":{"has_update":false,"latest_version":"1.0.0"}}' : '{"error":"offline"}' }));
      await page.goto("/settings/general", { waitUntil: "domcontentloaded" });
      await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
      await page.locator("#settings-updates-enabled").setChecked(false);
      await page.locator("#settings-update-check").click();
      await expect(page.locator("#settings-update-status")).not.toBeEmpty();
      await expect(page.locator("nav")).toBeVisible();
    } finally { await page.close(); }
  }
});
