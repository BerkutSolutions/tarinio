import { expect, test } from "../fixtures/auth";
import { openPage } from "../support/waits";

test("dashboard.widgets", async ({ authenticatedPage: page }) => {
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#dashboard-page")).toBeVisible({ timeout: 15000 });
  for (const widget of ["services", "traffic-summary", "containers-health", "top-ips", "top-countries", "requests-series", "top-urls", "memory", "cpu"]) {
    await expect(page.locator('[data-widget-id="' + widget + '"]')).toBeVisible({ timeout: 15000 });
  }
  await page.locator("#dashboard-widgets-toggle").click();
  await expect(page.locator("#dashboard-widgets-menu")).toBeVisible();
  await page.locator("#dashboard-widgets-save").click();
  await expect(page.locator("#dashboard-widgets-menu")).toBeHidden();
});

test("dashboard.series-consistency", async ({ authenticatedPage: page }) => {
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator('[data-widget-id="requests-series"]')).toBeVisible({ timeout: 30000 });
  const stats = await page.evaluate(async () => (await fetch("/api/dashboard/stats", { credentials: "include" })).json());
  const sum = (rows: Array<{ count?: number }>) => rows.reduce((total, row) => total + Number(row.count || 0), 0);
  for (const field of ["requests_series", "attacks_series", "blocked_series"]) {
    expect(Array.isArray(stats[field]), field).toBe(true);
    expect(stats[field], field).toHaveLength(24);
  }
  expect(sum(stats.requests_series)).toBe(Number(stats.requests_day || 0));
  expect(sum(stats.attacks_series)).toBe(Number(stats.attacks_day || 0));
  expect(sum(stats.blocked_series)).toBe(Number(stats.blocked_attacks_day || 0));
  const chart = page.locator('[data-widget-body="requests-series"] [data-chart-overlay="true"]');
  await expect(chart).toBeVisible();
  await chart.hover({ position: { x: 40, y: 20 } });
  await expect(page.locator('[data-chart-tooltip="true"]')).toBeVisible();
  await expect(page.locator('[data-chart-tooltip="true"]')).toContainText(/requests|запрос|атак/i);
});

test("dashboard.top-data", async ({ authenticatedPage: page }) => {
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  for (const id of ["top-ips", "top-countries", "top-urls"]) {
    await expect(page.locator(`[data-widget-id="${id}"] .dashboard-list-row`).first()).toBeVisible({ timeout: 30000 });
  }
  const stats = await page.evaluate(async () => (await fetch("/api/dashboard/stats", { credentials: "include" })).json());
  for (const field of ["top_attacker_ips", "top_attacker_countries", "most_attacked_urls", "popular_errors"]) {
    expect(Array.isArray(stats[field]), field).toBe(true);
    expect(stats[field].length, field).toBeGreaterThan(0);
  }
  await page.locator("#dashboard-widgets-toggle").click();
  await page.locator('[data-widget-visibility-id="popular-errors"]').check();
  await page.locator("#dashboard-widgets-save").click();
  await expect(page.locator('[data-widget-id="popular-errors"] .dashboard-list-row').first()).toBeVisible();
});

test("dashboard.details", async ({ authenticatedPage: page }) => {
  test.setTimeout(90_000);
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  const board = page.locator("#dashboard-board");
  await expect(board.locator('[data-widget-action="services"]')).toBeVisible({ timeout: 30000 });
  const actions = ["services", "requests-day", "attacks-day", "blocked-attacks", "top-ips", "top-countries", "top-urls", "containers-health"];
  for (const [index, action] of actions.entries()) {
    const target = board.locator(`[data-widget-action="${action}"]`).first();
    await expect(target, action).toBeVisible();
    await target.click();
    await expect(page.locator("#dashboard-detail-modal"), action).toBeVisible();
    if (index === 0) await page.locator("#dashboard-detail-modal [data-dashboard-detail-close=true]").first().click({ position: { x: 5, y: 5 } });
    else if (index === 1) await page.locator("#dashboard-detail-modal").press("Escape");
    else await page.locator("#dashboard-detail-modal [data-dashboard-detail-close=true]").last().click();
    await expect(page.locator("#dashboard-detail-modal")).toBeHidden();
  }
  await expect(page.locator("#dashboard-detail-modal")).toBeHidden();
});

test("dashboard.picker-persistence", async ({ authenticatedPage: page }) => {
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  await page.locator("#dashboard-widgets-toggle").click();
  await page.locator('[data-widget-visibility-id="unique-attackers"]').check();
  await page.locator("#dashboard-widgets-save").click();
  await expect(page.locator('[data-widget-id="unique-attackers"]')).toBeVisible();
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page.locator('[data-widget-id="unique-attackers"]')).toBeVisible({ timeout: 30000 });
});

test("dashboard.layout-reset-resize", async ({ authenticatedPage: page }) => {
  await openPage(page, "/dashboard", "#dashboard-page");
  await expect(page.locator("#dashboard-edit-toggle")).toBeVisible({ timeout: 30_000 });
  await page.locator("#dashboard-edit-toggle").click();
  const frame = page.locator('[data-widget-id="memory"]');
  const handle = frame.locator('[data-resize-dir="se"]');
  await expect(handle).toBeAttached();
  if ((page.viewportSize()?.width || 0) > 900) {
    await handle.scrollIntoViewIfNeeded();
    const before = await frame.boundingBox();
    const box = await handle.boundingBox();
    if (!before || !box) throw new Error("desktop resize geometry is unavailable");
    const startX = box.x + box.width / 2;
    const startY = box.y + box.height / 2;
    await page.mouse.move(startX, startY);
    await page.mouse.down();
    await page.mouse.move(startX + 80, startY + 60, { steps: 5 });
    await page.mouse.up();
    const after = await frame.boundingBox();
    expect(after?.width || 0).toBeGreaterThan(before.width);
    await expect.poll(() => page.evaluate(() => {
      const persisted = JSON.parse(localStorage.getItem("waf.dashboard.layout.v1") || "[]");
      return Number(persisted.find((item: { id?: string }) => item.id === "memory")?.width || 0);
    })).toBeGreaterThan(330);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator('[data-widget-id="memory"]')).toHaveCSS("width", `${Math.round(after?.width || 0)}px`);
  } else {
    await expect(frame).not.toHaveClass(/dragging/);
  }
  const reset = page.locator("#dashboard-layout-reset");
  await expect(reset).toBeVisible({ timeout: 30_000 });
  await reset.click();
  if ((page.viewportSize()?.width || 0) > 900) {
    await expect(frame).toHaveCSS("width", "340px");
    await expect.poll(() => page.evaluate(() => {
      const persisted = JSON.parse(localStorage.getItem("waf.dashboard.layout.v1") || "[]");
      return Number(persisted.find((item: { id?: string }) => item.id === "memory")?.width || 0);
    })).toBe(340);
  }
});

test("dashboard.resilience-states", async ({ authenticatedPage: page }) => {
  const realStats = await page.evaluate(async () => (await fetch("/api/dashboard/stats", { credentials: "include" })).json());
  let releaseLoading: (() => void) | undefined;
  const loadingGate = new Promise<void>((resolve) => { releaseLoading = resolve; });
  await page.route("**/api/dashboard/stats", async (route) => {
    await loadingGate;
    await route.continue();
  });
  const navigation = page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  try {
    await expect(page.locator("#dashboard-board .waf-empty").first()).toBeVisible({ timeout: 30000 });
    await expect(page.locator("#dashboard-board .waf-empty").first()).toContainText(/loading|загруз/i);
  } finally {
    releaseLoading?.();
  }
  await navigation;
  await page.unroute("**/api/dashboard/stats");
  await expect(page.locator('[data-widget-id="traffic-summary"]')).toBeVisible();
  await page.route("**/api/dashboard/stats", (route) => route.fulfill({ status: 503, contentType: "application/json", body: '{"error":"synthetic dashboard outage"}' }));
  await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
  await expect(page.locator("#dashboard-board .alert")).toContainText("synthetic dashboard outage", { timeout: 30000 });
  await page.unroute("**/api/dashboard/stats");
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page.locator("#dashboard-page")).toBeVisible({ timeout: 30000 });

  for (const state of ["empty", "partial"] as const) {
    const statePage = await page.context().newPage();
    try {
      await statePage.route("**/api/dashboard/stats", (route) => {
        const payload = state === "empty" ? {
          generated_at: new Date().toISOString(), services: [], services_up: 0, services_down: 0,
          requests_day: 0, attacks_day: 0, blocked_attacks_day: 0, requests_series: [],
          attacks_series: [], blocked_series: [], request_top_sites: [], request_top_urls: [],
          top_attacker_ips: [], top_attacker_countries: [], most_attacked_urls: [], popular_errors: [], system: {}
        } : {
          ...realStats, services: realStats.services?.slice?.(0, 1) || [], top_attacker_ips: [],
          top_attacker_countries: undefined, most_attacked_urls: [], popular_errors: undefined,
          system: { cpu_load_percent: 12.5, cpu_cores: 2 }
        };
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(payload) });
      });
      await statePage.route("**/api/dashboard/containers/overview", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(state === "partial" ? { containers: [{ name: "partial-container", cpu_percent: 12.5, memory_percent: 0 }], total_containers: 1, running_containers: 1, total_cpu_percent: 12.5, cpu_capacity_percent: 100, avg_memory_percent: 0 } : { containers: [], total_containers: 0, running_containers: 0 }) }));
      await statePage.route("**/api/requests**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: "[]" }));
      await statePage.route("**/api/events**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: '{"events":[]}' }));
      await openPage(statePage, `/dashboard?e2e-state=${state}`, "#dashboard-page");
      await expect(statePage.locator('[data-widget-id="traffic-summary"]')).toBeVisible();
      if (state === "empty") {
        await expect(statePage.locator('[data-widget-id="traffic-summary"] .dashboard-traffic-value')).toHaveText(["0", "0", "0"]);
        await expect(statePage.locator('[data-widget-id="requests-series"] .waf-empty')).toBeVisible();
        await expect(statePage.locator('[data-widget-id="top-ips"] .waf-empty')).toBeVisible();
      } else {
        await expect(statePage.locator('[data-widget-id="cpu"] .dashboard-system-main')).toHaveText("12.5%");
        await expect(statePage.locator('[data-widget-id="top-ips"] .waf-empty')).toBeVisible();
      }
      await expect(statePage.locator("#dashboard-board .alert")).toHaveCount(0);
    } finally {
      await statePage.close();
    }
  }
});

for (const metric of ["memory", "cpu"]) {
  test("dashboard." + metric + "-progress", async ({ authenticatedPage: page }) => {
    const statsResponsePromise = page.waitForResponse((response) =>
      new URL(response.url()).pathname === "/api/dashboard/stats" && response.request().method() === "GET",
    );
    const containerOverview = await page.evaluate(async () => {
      const response = await fetch("/api/dashboard/containers/overview", { credentials: "include", headers: { Accept: "application/json" } });
      const body = await response.text();
      if (!response.ok) throw new Error(`container overview returned ${response.status}: ${body}`);
      return JSON.parse(body);
    });
    await page.route("**/api/dashboard/containers/overview", (route) => route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(containerOverview),
    }));
    await page.goto("/dashboard", { waitUntil: "domcontentloaded", timeout: 60000 });
    const statsResponse = await statsResponsePromise;
    const statsBody = await statsResponse.text();
    expect(statsResponse.status(), statsBody).toBe(200);
    const realStats = JSON.parse(statsBody);
    const system = realStats?.system || {};
    const widget = page.locator('[data-widget-id="' + metric + '"]');
    await expect(widget).toBeVisible({ timeout: 15000 });
    const language = await page.evaluate(async () => {
      const response = await fetch("/api/settings/runtime", { credentials: "include", headers: { Accept: "application/json" } });
      if (!response.ok) throw new Error(`runtime settings returned ${response.status}`);
      return String((await response.json())?.language || "en");
    });
    const titles: Record<string, Record<string, string>> = {
      cpu: { en: "CPU load", ru: "Нагрузка CPU", de: "CPU-Last", sr: "Оптерећење процесора", zh: "CPU负载" },
      memory: { en: "Memory utilization", ru: "Использование памяти", de: "Speicherauslastung", sr: "Коришћење меморије", zh: "内存利用率" },
    };
    expect(titles[metric][language], `unsupported dashboard language ${language}`).toBeTruthy();
    await expect(widget.locator(".dashboard-frame-header")).toContainText(titles[metric][language]);
    await expect(widget.locator(".dashboard-system-main")).toHaveText(/%/);
    await expect(widget.locator(".dashboard-progress span")).toHaveAttribute("style", /width:/);
    const expectedAggregate = metric === "cpu" ? Number(containerOverview?.total_cpu_percent || 0) : Number(containerOverview?.avg_memory_percent || 0);
    await expect(widget.locator(".dashboard-system-main")).toHaveText(`${expectedAggregate.toFixed(1)}%`);
    const systemRows = await widget.locator(".dashboard-system-row").count();
    expect(systemRows).toBe(Math.min(8, containerOverview?.containers?.length || 0));
    for (const container of (containerOverview?.containers || []).slice(0, 8)) {
      const row = widget.locator(".dashboard-system-container-row").filter({ hasText: String(container.name) });
      await expect(row).toContainText(metric === "cpu" ? `${Number(container.cpu_percent || 0).toFixed(1)}%` : `${Number(container.memory_percent || 0).toFixed(1)}%`);
    }
    await widget.locator('[data-widget-action="' + metric + '"]').click();
    const modal = page.locator("#dashboard-detail-modal");
    await expect(modal).toBeVisible();
    const formatPercent = (value: unknown) => `${Number(value || 0).toFixed(1)}%`;
    const formatBytes = (raw: unknown) => {
      let value = Number(raw || 0);
      if (!value) return "0 B";
      const units = ["B", "KB", "MB", "GB", "TB"];
      let unit = 0;
      while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; }
      return `${value.toFixed(unit > 1 ? 1 : 0)} ${units[unit]}`;
    };
    const expectedSummary = [formatPercent(metric === "cpu" ? containerOverview.total_cpu_percent : containerOverview.avg_memory_percent)];
    await expect(modal.locator(".mini-metric-value")).toHaveText(expectedSummary);
    const expectedProcesses = (containerOverview?.containers || [])
      .map((item: { name?: string; image?: string; pids?: number }) => ({ ...item, command: item.image || "-", threads: item.pids || 0 }))
      .sort((left, right) => String(left.name || "").localeCompare(String(right.name || ""), undefined, { sensitivity: "base" }));
    const processRows = await modal.locator("tbody tr").allTextContents();
    expect(processRows).toHaveLength(expectedProcesses.length);
    for (const [index, process] of expectedProcesses.entries()) {
      expect(processRows[index]).toContain(String(process.name || process.command || `pid-${process.pid || 0}`));
      expect(processRows[index]).toContain(`Threads: ${process.threads || 0}`);
    }
    await modal.press("Escape");
    await expect(modal).toBeHidden();

    for (const value of [0, 37.5, 100, 150]) {
      const edgePage = await page.context().newPage();
      try {
        await edgePage.route("**/api/dashboard/stats", async (route) => {
          const payload = structuredClone(realStats);
          payload.system = {
            ...(payload.system || {}),
            cpu_load_percent: metric === "cpu" ? value : 37.5,
            cpu_cores: 8,
            goroutines: 12,
            memory_used_percent: metric === "memory" ? value : 37.5,
            memory_used_bytes: 3 * 1024 * 1024,
            memory_free_bytes: 5 * 1024 * 1024,
            memory_total_bytes: 8 * 1024 * 1024
          };
          await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(payload) });
        });
        await edgePage.route("**/api/dashboard/containers/overview", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ containers: [{ name: "edge-container", image: "edge", state: "running", status: "Up", cpu_percent: metric === "cpu" ? value : 0, memory_percent: metric === "memory" ? value : 0, pids: 1, memory_usage_bytes: 1024 }], total_containers: 1, running_containers: 1, total_cpu_percent: metric === "cpu" ? value : 0, cpu_capacity_percent: 100, avg_memory_percent: metric === "memory" ? value : 0 }) }));
        await openPage(edgePage, `/dashboard?e2e-system-edge=${encodeURIComponent(String(value))}`, `[data-widget-id="${metric}"]`);
        const edgeWidget = edgePage.locator(`[data-widget-id="${metric}"]`);
        const expected = Math.min(100, value);
        await expect(edgeWidget.locator(".dashboard-system-main")).toHaveText(`${expected.toFixed(1)}%`);
        await expect(edgeWidget.locator(".dashboard-progress span")).toHaveAttribute("style", `width:${expected}%`);
        await expect(edgeWidget.locator(".dashboard-system-container-row")).toHaveCount(1);
      } finally {
        await edgePage.close();
      }
    }
  });
}
