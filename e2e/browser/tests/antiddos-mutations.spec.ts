import { expect, test } from "../fixtures/auth";
import { e2eID } from "../support/isolation";
import { runtimeBaseURL } from "../support/env";
import { openPage } from "../support/waits";

type APIResult = { status: number; body: string };

async function api(page: import("@playwright/test").Page, path: string, init: RequestInit = {}): Promise<APIResult> {
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    try {
      return await page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30_000);
    try {
      const response = await fetch(path, {
        ...init, credentials: "include", signal: controller.signal,
        headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) },
      });
      return { status: response.status, body: await response.text() };
    } finally {
      window.clearTimeout(timer);
    }
      }, { path, init });
    } catch (error) {
      lastError = error;
      if (!/Failed to fetch|Execution context was destroyed|ERR_NETWORK_CHANGED|ERR_CONNECTION_(CLOSED|RESET)/i.test(String(error)) || attempt === 2) throw error;
      await page.waitForTimeout(250 * (attempt + 1));
    }
  }
  throw lastError;
}

async function readSettings(page: import("@playwright/test").Page) {
  const result = await api(page, "/api/anti-ddos/settings");
  expect(result.status, result.body).toBe(200);
  return JSON.parse(result.body);
}

async function writeSettings(page: import("@playwright/test").Page, value: unknown) {
  const result = await api(page, "/api/anti-ddos/settings", { method: "PUT", body: JSON.stringify(value) });
  expect(result.status, result.body).toBe(200);
  return JSON.parse(result.body);
}

test("anti-ddos.save-validation-restore", async ({ authenticatedPage: page }) => {
  await page.goto("/anti-ddos", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#antiddos-form")).toBeVisible({ timeout: 30000 });
  const api = async (path: string, init: RequestInit = {}) => page.evaluate(async ({ path, init }) => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch(path, { ...init, credentials: "include", signal: controller.signal, headers: { Accept: "application/json", "Content-Type": "application/json", ...(init.headers || {}) } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  }, { path, init });
  const readSettings = async () => {
    const result = await api("/api/anti-ddos/settings");
    expect(result.status).toBe(200);
    return JSON.parse(result.body);
  };
  const original = await readSettings();
  const nextEnabled = original.model_enabled === false;
  try {
    await page.locator("#model-enabled").setChecked(nextEnabled);
    await page.locator("#antiddos-form button[type=submit]").click();
    await expect.poll(async () => Boolean((await readSettings()).model_enabled), { timeout: 30000 }).toBe(nextEnabled);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#model-enabled")).toBeChecked({ checked: nextEnabled });

    const statusCode = page.locator("#antiddos-l7-status");
    const persistedStatus = Number((await readSettings()).l7_status_code);
    await statusCode.fill("99");
    await expect.poll(() => statusCode.evaluate((node) => (node as HTMLInputElement).validity.valid)).toBe(false);
    await page.locator("#antiddos-form button[type=submit]").click();
    await expect.poll(async () => Number((await readSettings()).l7_status_code)).toBe(persistedStatus);
  } finally {
    let restore = await api("/api/anti-ddos/settings", { method: "POST", body: JSON.stringify(original) });
    if (restore.status === 404 || restore.status === 405) restore = await api("/api/anti-ddos/settings", { method: "PUT", body: JSON.stringify(original) });
    expect(restore.status).toBeLessThan(300);
    await expect.poll(async () => Boolean((await readSettings()).model_enabled), { timeout: 30000 }).toBe(Boolean(original.model_enabled));
  }
});

test("anti-ddos.all-groups-roundtrip-restore", async ({ authenticatedPage: page }) => {
  await openPage(page, "/anti-ddos", "#antiddos-model-logs");
  await expect(page.locator("#antiddos-form")).toBeVisible({ timeout: 30_000 });
  const original = await readSettings(page);
  const expected = {
    ...original,
    use_l4_guard: !original.use_l4_guard, chain_mode: "input", target: "REJECT",
    conn_limit: 321, rate_per_second: 41, rate_burst: 82, ports: [8080, 8443], destination_ip: "192.0.2.44",
    enforce_l7_rate_limit: true, l7_requests_per_second: 37, l7_burst: 74, l7_status_code: 444,
    model_enabled: true, model_poll_interval_seconds: 3, model_decay_lambda: 0.12,
    model_throttle_threshold: 3.5, model_drop_threshold: 7.5, model_hold_seconds: 75,
    model_throttle_rate_per_second: 4, model_throttle_burst: 9, model_throttle_target: "DROP",
    model_weight_429: 1.1, model_weight_403: 1.9, model_weight_444: 2.4,
    model_emergency_rps: 190, model_emergency_unique_ips: 45, model_emergency_per_ip_rps: 65,
    model_weight_emergency_botnet: 6.5, model_weight_emergency_single: 4.5,
  };
  const fields: Array<[string, string]> = [
    ["#antiddos-chain-mode", expected.chain_mode], ["#antiddos-target", expected.target],
    ["#antiddos-conn-limit", String(expected.conn_limit)], ["#antiddos-rate-ps", String(expected.rate_per_second)],
    ["#antiddos-rate-burst", String(expected.rate_burst)], ["#antiddos-ports", "8080, 8443"],
    ["#antiddos-destination-ip", expected.destination_ip], ["#antiddos-l7-rps", String(expected.l7_requests_per_second)],
    ["#antiddos-l7-burst", String(expected.l7_burst)], ["#antiddos-l7-status", String(expected.l7_status_code)],
    ["#model-poll", "3"], ["#model-decay", "0.12"], ["#model-threshold-throttle", "3.5"],
    ["#model-threshold-drop", "7.5"], ["#model-hold", "75"], ["#model-throttle-rps", "4"],
    ["#model-throttle-burst", "9"], ["#model-throttle-target", "DROP"], ["#model-weight-429", "1.1"],
    ["#model-weight-403", "1.9"], ["#model-weight-444", "2.4"], ["#model-emergency-rps", "190"],
    ["#model-emergency-unique", "45"], ["#model-emergency-per-ip", "65"],
    ["#model-weight-emergency-botnet", "6.5"], ["#model-weight-emergency-single", "4.5"],
  ];
  try {
    await page.locator("#antiddos-use-l4").setChecked(expected.use_l4_guard);
    await page.locator("#antiddos-enforce-l7").setChecked(true);
    await page.locator("#model-enabled").setChecked(true);
    for (const [selector, value] of fields) {
      const node = page.locator(selector);
      await expect(node, selector).toBeVisible();
      if (await node.evaluate((element) => element.tagName === "SELECT")) await node.selectOption(value);
      else await node.fill(value);
    }
    await page.locator("#antiddos-form button[type=submit]").click();
    await expect.poll(async () => (await readSettings(page)).destination_ip, { timeout: 30_000 }).toBe(expected.destination_ip);
    const saved = await readSettings(page);
    for (const key of Object.keys(expected)) {
      if (key === "created_at" || key === "updated_at") continue;
      expect(saved[key], key).toEqual(expected[key]);
    }
    const audit = await api(page, "/api/audit?action=antiddos.settings.upsert&status=succeeded&limit=100");
    expect(audit.status, audit.body).toBe(200);
    expect(audit.body).toContain("antiddos.settings.upsert");
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#antiddos-ports")).toHaveValue("8080, 8443");
    await expect(page.locator("#model-throttle-target")).toHaveValue("DROP");
  } finally {
    await writeSettings(page, original);
    await expect.poll(async () => (await readSettings(page)).updated_at, { timeout: 30_000 }).not.toBe("");
  }
});

test("anti-ddos.server-validation-no-partial-save", async ({ authenticatedPage: page }) => {
  await page.goto("/anti-ddos", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const original = await readSettings(page);
  const invalidCases = [
    { ...original, chain_mode: "forward" },
    { ...original, target: "ACCEPT" },
    { ...original, rate_per_second: 50, rate_burst: 49 },
    { ...original, destination_ip: "invalid-ip" },
    { ...original, enforce_l7_rate_limit: true, l7_status_code: 600 },
    { ...original, use_l4_guard: true, ports: [0, 70000] },
  ];
  try {
    for (const value of invalidCases) {
      const result = await api(page, "/api/anti-ddos/settings", { method: "PUT", body: JSON.stringify(value) });
      expect(result.status, result.body).toBe(400);
      expect(JSON.parse(result.body).error).toBeTruthy();
      expect(await readSettings(page)).toEqual(original);
    }
    const normalized = await writeSettings(page, {
      ...original, model_enabled: true, model_throttle_threshold: 5, model_drop_threshold: 5, model_throttle_target: "ACCEPT",
    });
    expect(normalized.model_drop_threshold).toBe(5.5);
    expect(normalized.model_throttle_target).toBe("REJECT");
  } finally {
    await writeSettings(page, original);
  }
});

test("anti-ddos.rule-suggestions-create-list-detail-status", async ({ authenticatedPage: page }, testInfo) => {
  await page.goto("/anti-ddos", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const id = e2eID(testInfo, "path-suggestion");
  const pathPrefix = `/e2e/${id}`;
  const created = await api(page, "/api/anti-ddos/rule-suggestions", {
    method: "POST",
    body: JSON.stringify({ id, path_prefix: pathPrefix, status: "suggested", hits: 17, unique_ips: 5, source: "e2e", reason: "coverage" }),
  });
  expect(created.status, created.body).toBe(200);
  expect(JSON.parse(created.body)).toMatchObject({ id, path_prefix: pathPrefix, status: "suggested", hits: 17, unique_ips: 5 });
  const listed = await api(page, "/api/anti-ddos/rule-suggestions");
  expect(listed.status, listed.body).toBe(200);
  expect(JSON.parse(listed.body).items).toContainEqual(expect.objectContaining({ id, path_prefix: pathPrefix, reason: "coverage" }));
  const status = await api(page, `/api/anti-ddos/rule-suggestions/${encodeURIComponent(id)}/status`, {
    method: "PUT", body: JSON.stringify({ status: "shadow" }),
  });
  expect(status.status, status.body).toBe(200);
  expect(JSON.parse(status.body)).toMatchObject({ id, status: "shadow" });
});

test("anti-ddos.logs-refresh-detail-or-empty", async ({ authenticatedPage: page }) => {
  const id = e2eID(test.info(), "telemetry");
  const siteID = `${id}-site`;
  const upstreamID = `${id}-upstream`;
  const host = `${id}.test`;
  const originalSettings = await readSettings(page);
  const runtimeStatus = async () => (await page.context().request.get(`${runtimeBaseURL()}/`, {
    headers: { Host: host }, failOnStatusCode: false, timeout: 30_000,
  })).status();
  const compileAndApply = async () => {
    const compiled = await api(page, "/api/revisions/compile", { method: "POST", body: "{}" });
    expect(compiled.status, compiled.body).toBe(201);
    const revisionID = String(JSON.parse(compiled.body)?.revision?.id || "");
    expect(revisionID).not.toBe("");
    const applied = await api(page, `/api/revisions/${encodeURIComponent(revisionID)}/apply`, { method: "POST", body: "{}" });
    expect(applied.status, applied.body).toBe(201);
  };

  await page.goto("/anti-ddos", { waitUntil: "domcontentloaded", timeout: 60_000 });
  const logs = page.locator("#antiddos-model-logs");
  await expect(logs).toBeVisible({ timeout: 30_000 });
  try {
    let created = await api(page, "/api/sites?auto_apply=false", {
      method: "POST",
      body: JSON.stringify({ id: siteID, primary_host: host, enabled: true, listen_http: true, listen_https: false, use_easy_config: true, default_upstream_id: upstreamID }),
    });
    expect([200, 201], created.body).toContain(created.status);
    created = await api(page, "/api/upstreams?auto_apply=false", {
      method: "POST",
      body: JSON.stringify({ id: upstreamID, site_id: siteID, name: upstreamID, scheme: "http", host: "upstream-echo", port: 8888, base_path: "/", pass_host_header: false }),
    });
    expect([200, 201], created.body).toContain(created.status);
    await writeSettings(page, {
      ...originalSettings, use_l4_guard: false, enforce_l7_rate_limit: true,
      l7_requests_per_second: 1, l7_burst: 1, l7_status_code: 429,
    });
    await compileAndApply();
    await expect.poll(runtimeStatus, { timeout: 120_000 }).toBe(200);
    const statuses: number[] = [];
    for (let index = 0; index < 30 && !statuses.includes(429); index += 1) statuses.push(await runtimeStatus());
    expect(statuses, `runtime did not emit a real L7 429: ${statuses.join(",")}`).toContain(429);

    await expect.poll(async () => {
      const events = await api(page, "/api/events");
      if (events.status !== 200) return false;
      return (JSON.parse(events.body)?.events || []).some((item: { type?: string; site_id?: string; details?: { host?: string; status?: number } }) =>
        item.type === "security_rate_limit" && item.site_id === siteID && item.details?.host === host && item.details?.status === 429,
      );
    }, { timeout: 30_000 }).toBe(true);
    await page.locator("#antiddos-model-logs-refresh").click();
    const row = logs.locator("[data-log-index]").filter({ hasText: siteID }).first();
    await expect(row, "security telemetry must contain the L7 429 emitted by the temporary public service").toBeVisible({ timeout: 30_000 });
    const cells = (await row.locator("td").allTextContents()).map((value) => value.trim());
    expect(cells).toHaveLength(5);
    expect(cells[0]).not.toBe("");
    expect(cells[1]).not.toBe("-");
    expect(cells[2]).not.toBe("-");
    expect(cells[3]).not.toBe("-");
    expect(cells[4]).toMatch(/^(?:\d{1,3}\.){3}\d{1,3}$|^[0-9a-f:]+$/i);
    await row.press("Enter");
    await expect(page.locator("#antiddos-model-log-detail-modal")).toBeVisible();
    const detail = page.locator("#antiddos-model-log-detail-content");
    await expect(detail.locator("tr")).toHaveCount(11);
    for (const value of cells) await expect(detail).toContainText(value);
    await page.locator("#antiddos-model-log-detail-modal").press("Escape");
    await expect(page.locator("#antiddos-model-log-detail-modal")).toBeHidden();
  } finally {
    await writeSettings(page, originalSettings);
    for (const path of [
      `/api/sites/${encodeURIComponent(siteID)}?auto_apply=false`,
      `/api/upstreams/${encodeURIComponent(upstreamID)}?auto_apply=false`,
      `/api/easy-site-profiles/${encodeURIComponent(siteID)}?auto_apply=false`,
    ]) {
      const deleted = await api(page, path, { method: "DELETE" });
      expect([200, 204, 404], deleted.body).toContain(deleted.status);
    }
    await compileAndApply();
  }
});
