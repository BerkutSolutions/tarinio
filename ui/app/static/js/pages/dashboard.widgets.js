import { escapeHtml } from "../ui.js";

function countryFlag(code, deps) {
  const token = deps.normalizeCountryCode(code);
  if (!/^[A-Z]{2}$/.test(token)) {
    return "";
  }
  const cc = token.toLowerCase();
  return `<img class="country-flag-img" src="https://flagcdn.com/16x12/${cc}.png" srcset="https://flagcdn.com/32x24/${cc}.png 2x" width="16" height="12" alt="${token}" loading="lazy" onerror="this.style.display='none';this.nextSibling&&(this.nextSibling.style.display='')"><span class="country-flag-fallback" style="display:none">${token}</span>`;
}

function countryName(code, deps) {
  const token = deps.normalizeCountryCode(code);
  if (token === "UNK") {
    return "Unknown";
  }
  try {
    const names = new Intl.DisplayNames(["ru", "en"], { type: "region" });
    return names.of(token) || token;
  } catch (_error) {
    return token;
  }
}

function renderCountryBadge(code, deps) {
  const normalized = deps.normalizeCountryCode(code);
  if (normalized === "UNK") {
    return `<span class="dashboard-ip-country">${escapeHtml(countryName(normalized, deps))}</span>`;
  }
  const flag = countryFlag(normalized, deps);
  const name = countryName(normalized, deps);
  return `<span class="dashboard-ip-country">${escapeHtml(name)}${flag ? " " + flag : ""}</span>`;
}

// Форматирует строку даты/времени в локализованный вид браузера
function formatCheckedAt(raw) {
  const str = String(raw || "").trim();
  if (!str) return "";
  const ts = Date.parse(str);
  if (Number.isNaN(ts)) return str;
  try {
    return new Intl.DateTimeFormat(undefined, {
      year: "numeric", month: "2-digit", day: "2-digit",
      hour: "2-digit", minute: "2-digit", hour12: false
    }).format(new Date(ts));
  } catch (_e) {
    return str;
  }
}

function renderMetric(value, label, tone, widgetAction, deps) {
  const clickable = widgetAction ? "clickable" : "";
  const actionAttr = widgetAction ? ` data-widget-action="${escapeHtml(widgetAction)}"` : "";
  return `
    <button type="button" class="dashboard-widget-content dashboard-stat dashboard-metric tone-${escapeHtml(tone)} ${clickable}"${actionAttr}>
      <div class="dashboard-stat-value">${escapeHtml(deps.formatNumber(value))}</div>
      <div class="dashboard-stat-label">${escapeHtml(label)}</div>
    </button>
  `;
}

// Виджет «Трафик и атаки» — три метрики в столбец с цветными рамками
function renderTrafficSummaryWidget(stats, ctx, deps) {
  const requests = Number(stats?.requests_day || 0);
  const attacks  = Number(stats?.attacks_day || 0);
  const blocked  = Number(stats?.blocked_attacks_day || 0);
  return `
    <div class="dashboard-widget-content dashboard-traffic-summary">
      <button type="button" class="dashboard-traffic-row tone-success clickable" data-widget-action="requests-day">
        <div class="dashboard-traffic-value">${escapeHtml(deps.formatNumber(requests))}</div>
        <div class="dashboard-traffic-label">${escapeHtml(ctx.t("dashboard.value.requestsDay"))}</div>
      </button>
      <button type="button" class="dashboard-traffic-row tone-warning clickable" data-widget-action="attacks-day">
        <div class="dashboard-traffic-value">${escapeHtml(deps.formatNumber(attacks))}</div>
        <div class="dashboard-traffic-label">${escapeHtml(ctx.t("dashboard.value.attacksDay"))}</div>
      </button>
      <button type="button" class="dashboard-traffic-row tone-danger clickable" data-widget-action="blocked-attacks">
        <div class="dashboard-traffic-value">${escapeHtml(deps.formatNumber(blocked))}</div>
        <div class="dashboard-traffic-label">${escapeHtml(ctx.t("dashboard.value.blockedAttacksDay"))}</div>
      </button>
    </div>
  `;
}

function renderServicesWidget(stats, ctx, deps) {
  const SYSTEM_SERVICES = new Set(["control-plane", "runtime"]);
  const services = (Array.isArray(stats?.services) ? stats.services : [])
    .filter((s) => !SYSTEM_SERVICES.has(String(s?.name || "").trim().toLowerCase()));
  if (!services.length) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.services.empty"))}</div>`;
  }
  const upCount   = services.filter((s) => Boolean(s?.up)).length;
  const downCount = services.length - upCount;
  const sorted = services.slice().sort((a, b) => {
    const aUp = Boolean(a?.up);
    const bUp = Boolean(b?.up);
    if (aUp !== bUp) return aUp ? 1 : -1;
    return String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" });
  });

  return `
    <div class="dashboard-widget-content dashboard-services-widget" data-widget-action="services">
      <div class="dashboard-containers-system-row">
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(String(upCount))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.services.up"))}</div>
        </div>
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(String(downCount))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.services.down"))}</div>
        </div>
        <div class="mini-metric mini-metric-running">
          <div class="mini-metric-value">${escapeHtml(String(services.length))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.services.total"))}</div>
        </div>
      </div>
      <div class="dashboard-containers-list-wrap">
        <div class="dashboard-containers-list-title">${escapeHtml(ctx.t("dashboard.services.service"))}</div>
        <div class="dashboard-list dashboard-containers-list">
          ${sorted.map((item) => {
            const isUp     = Boolean(item?.up);
            const hasErrors = Boolean(item?.has_errors);
            // Жёлтый если сервис работает но есть upstream ошибки, красный если down.
            const tone     = !isUp ? "warning" : hasErrors ? "caution" : "success";
            const statusLbl = isUp ? ctx.t("dashboard.services.statusUp") : ctx.t("dashboard.services.statusDown");
            const checkedAt = formatCheckedAt(item?.checked_at);
            const errorCount = Array.isArray(item?.upstream_errors) ? item.upstream_errors.length : 0;
            return `
              <button type="button" class="dashboard-list-row clickable container-status-${escapeHtml(tone)}" data-widget-action="service-detail" data-service-name="${escapeHtml(String(item?.name || ""))}">
                <div class="dashboard-list-label">
                  <strong>${escapeHtml(String(item?.name || "-"))}</strong>
                  <div class="muted">${escapeHtml(statusLbl)}${checkedAt ? ` · ${escapeHtml(checkedAt)}` : ""}${errorCount > 0 ? ` · <span style="color:var(--color-warning)">${escapeHtml(String(errorCount))} ${escapeHtml(ctx.t("dashboard.services.errorsCount"))}</span>` : ""}</div>
                </div>
                <div class="dashboard-list-meta">
                  <span class="badge badge-${escapeHtml(!isUp ? "warning" : hasErrors ? "caution" : "success")}">${escapeHtml(statusLbl)}</span>
                  ${hasErrors ? `<span class="badge badge-caution">${escapeHtml(ctx.t("dashboard.services.hasErrors"))}</span>` : ""}
                </div>
              </button>
            `;
          }).join("")}
        </div>
      </div>
    </div>
  `;
}

function renderContainersHealthWidget(overview, ctx, deps) {
  if (!overview || !Array.isArray(overview?.containers)) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.containers.empty"))}</div>`;
  }
  const containers = overview.containers
    .slice()
    .sort((l, r) => String(l?.name || "").localeCompare(String(r?.name || ""), undefined, { sensitivity: "base" }))
    .slice(0, 12);
  const uptimeText = deps.formatUptimeLocalized(overview?.host_uptime_seconds || 0, ctx);
  return `
    <div class="dashboard-widget-content dashboard-containers-widget" data-widget-action="containers-health">
      <div class="dashboard-containers-uptime-inline">${escapeHtml(ctx.t("dashboard.containers.uptime"))}: ${escapeHtml(uptimeText)}</div>
      <div class="dashboard-containers-system-row">
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(deps.formatPercent(overview?.total_cpu_percent || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.cpu"))}</div>
        </div>
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(deps.formatPercent(overview?.avg_memory_percent || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.memory"))}</div>
        </div>
        <div class="mini-metric mini-metric-running">
          <div class="mini-metric-value">${escapeHtml(String(overview?.running_containers || 0))} / ${escapeHtml(String(overview?.total_containers || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.running"))}</div>
        </div>
      </div>
      <div class="dashboard-containers-list-wrap">
        <div class="dashboard-containers-list-title">${escapeHtml(ctx.t("dashboard.containers.container"))}</div>
        <div class="dashboard-list dashboard-containers-list">
          ${containers.map((item) => `
            <button type="button" class="dashboard-list-row clickable container-status-${escapeHtml(deps.getContainerStatusTone(item))}" data-widget-action="container-logs" data-container-name="${escapeHtml(String(item?.name || ""))}">
              <div class="dashboard-list-label">
                <strong>${escapeHtml(String(item?.name || "-"))}</strong>
                <div class="muted">${escapeHtml(deps.formatContainerStatusLabel(item?.status || "-", ctx))}</div>
              </div>
              <div class="dashboard-list-meta">
                <span class="badge badge-neutral">CPU ${escapeHtml(deps.formatPercent(item?.cpu_percent || 0))}</span>
                <span class="badge badge-neutral">MEM ${escapeHtml(deps.formatPercent(item?.memory_percent || 0))}</span>
                <span class="badge badge-neutral">${escapeHtml(String(item?.network_in_text || "0 B"))} / ${escapeHtml(String(item?.network_out_text || "0 B"))}</span>
              </div>
            </button>
          `).join("")}
        </div>
      </div>
    </div>
  `;
}

function renderTopList(items, emptyText, options = {}, deps) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(emptyText)}</div>`;
  }
  const renderLabel    = typeof options.renderLabel    === "function" ? options.renderLabel    : (item) => escapeHtml(String(item?.key || "-"));
  const renderMetaRight = typeof options.renderMetaRight === "function" ? options.renderMetaRight : () => "";
  const rowAttrs       = typeof options.rowAttrs       === "function" ? options.rowAttrs       : () => "";
  const containerAttr  = options.containerAction ? ` data-widget-action="${escapeHtml(options.containerAction)}"` : "";
  return `
    <div class="dashboard-widget-content dashboard-scroll-area"${containerAttr}>
      <div class="dashboard-list">
        ${items.map((item, index) => {
          const attrs = rowAttrs(item, index);
          return `
            <button type="button" class="dashboard-list-row ${attrs ? "clickable" : ""}" ${attrs}>
              <div class="dashboard-list-label">${renderLabel(item)}</div>
              <div class="dashboard-list-meta">
                <span class="badge badge-neutral">${escapeHtml(deps.formatNumber(item?.count || 0))}</span>
                ${renderMetaRight(item)}
              </div>
            </button>
          `;
        }).join("")}
      </div>
    </div>
  `;
}

function renderIPTopList(items, emptyText, widgetAction = "", deps) {
  return renderTopList(items, emptyText, {
    containerAction: widgetAction,
    renderLabel: (item) => {
      const ip = String(item?.key || "-").trim() || "-";
      return `<span class="dashboard-ip-main">${escapeHtml(ip)}</span><div class="dashboard-ip-country-block">${renderCountryBadge(item?.countryCode || item?.key, deps)}</div>`;
    },
    rowAttrs: (item) => {
      const ip = String(item?.key || "").trim();
      if (!ip) return widgetAction ? `data-widget-action="${escapeHtml(widgetAction)}"` : "";
      return `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"`;
    }
  }, deps);
}

function mergeWidgetData(stats, detailModel, containersOverview, ctx, deps) {
  const statsTopIPs       = Array.isArray(stats?.top_attacker_ips)       ? stats.top_attacker_ips       : [];
  const statsTopCountries = Array.isArray(stats?.top_attacker_countries)  ? stats.top_attacker_countries  : [];
  const fallbackTopCountries = Array.isArray(detailModel?.attacksByCountry) ? detailModel.attacksByCountry : [];

  const topCountryItems = (statsTopCountries.length ? statsTopCountries : fallbackTopCountries).map((item) => ({
    key: item?.key, count: item?.count, countryCode: item?.key
  }));

  const topIPItems = statsTopIPs.map((item) => {
    const key        = String(item?.key || "").trim();
    const rawCountry = item?.country_code || item?.countryCode || "";
    // Приоритет: поле из stats → detailModel по IP → UNK
    const countryCode = rawCountry
      ? deps.normalizeCountryCode(rawCountry)
      : (detailModel?.ipCountryByIP?.get?.(key) || detailModel?.ipDetailsByIP?.get?.(key)?.countryCode || "UNK");
    return { key, count: Number(item?.count || 0), countryCode };
  });

  // Если stats не содержат top_attacker_ips — берём из detailModel
  const effectiveTopIPItems = topIPItems.length
    ? topIPItems
    : (detailModel?.ipDetailsSummary || []).slice(0, 10).map((item) => ({
        key: item.ip, count: Math.max(item.attacks, item.requests), countryCode: item.countryCode
      }));

  // Топ стран: stats → detailModel.attacksByCountry (тот же источник что и остальные виджеты)
  const effectiveTopCountryItems = (statsTopCountries.length ? statsTopCountries : fallbackTopCountries).map((item) => ({
    key: item?.key, count: item?.count, countryCode: item?.key
  }));

  return {
    "services":         renderServicesWidget(stats, ctx, deps),
    "traffic-summary":  renderTrafficSummaryWidget(stats, ctx, deps),
    // legacy single-metric widgets (hidden by default but kept for detail actions)
    "requests-day":     renderMetric(stats?.requests_day,          ctx.t("dashboard.value.requestsDay"),       "success", "requests-day",    deps),
    "attacks-day":      renderMetric(stats?.attacks_day,           ctx.t("dashboard.value.attacksDay"),        "warning", "attacks-day",     deps),
    "blocked-attacks":  renderMetric(stats?.blocked_attacks_day,   ctx.t("dashboard.value.blockedAttacksDay"), "danger",  "blocked-attacks", deps),
    "unique-attackers": renderIPTopList(
      effectiveTopIPItems.length ? effectiveTopIPItems : (detailModel?.ipDetailsSummary || []).slice(0, 10).map((item) => ({
        key: item.ip, count: Math.max(item.attacks, item.requests), countryCode: item.countryCode
      })),
      ctx.t("dashboard.empty.topIPs"), "unique-attackers", deps
    ),
    "popular-errors": renderTopList(stats?.popular_errors || [], ctx.t("dashboard.empty.popularErrors"), {
      containerAction: "popular-errors",
      rowAttrs: (item) => {
        const code = String(item?.key || "").trim();
        return code ? `data-widget-action="error-detail" data-error-code="${escapeHtml(code)}"` : "";
      }
    }, deps),
    "top-ips": renderIPTopList(effectiveTopIPItems, ctx.t("dashboard.empty.topIPs"), "top-ips", deps),
    "top-countries": renderTopList(effectiveTopCountryItems, ctx.t("dashboard.empty.topCountries"), {
      containerAction: "top-countries",
      renderLabel: (item) => renderCountryBadge(item?.countryCode || item?.key, deps),
      rowAttrs: (item) => `data-widget-action="country-detail" data-country-code="${escapeHtml(deps.normalizeCountryCode(item?.countryCode || item?.key))}"`
    }, deps),
    "top-urls": renderTopList(
      (Array.isArray(stats?.most_attacked_urls) && stats.most_attacked_urls.length ? stats.most_attacked_urls : (detailModel?.attacksByURL || [])),
      ctx.t("dashboard.empty.topURLs"), {
        containerAction: "top-urls",
        rowAttrs: (item) => {
          const url = String(item?.key || "").trim();
          return url ? `data-widget-action="url-detail" data-url="${escapeHtml(url)}"` : "";
        }
      }, deps
    ),
    memory: deps.renderSystemMemory(stats, ctx),
    cpu:    deps.renderSystemCPU(stats, ctx),
    "containers-health": renderContainersHealthWidget(containersOverview, ctx, deps)
  };
}

export {
  renderCountryBadge,
  renderServicesWidget,
  renderTrafficSummaryWidget,
  mergeWidgetData
};
