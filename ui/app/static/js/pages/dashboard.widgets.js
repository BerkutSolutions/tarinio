import { escapeHtml } from "../ui.js";

function renderMetric(value, label, tone = "success", widgetAction = "", deps) {
  const clickable = widgetAction ? "clickable" : "";
  const actionAttr = widgetAction ? ` data-widget-action="${escapeHtml(widgetAction)}"` : "";
  return `
    <button type="button" class="dashboard-widget-content dashboard-stat dashboard-metric tone-${escapeHtml(tone)} ${clickable}"${actionAttr}>
      <div class="dashboard-stat-value">${escapeHtml(deps.formatNumber(value))}</div>
      <div class="dashboard-stat-label">${escapeHtml(label)}</div>
    </button>
  `;
}

function renderCountryBadge(code, deps) {
  const normalized = deps.normalizeCountryCode(code);
  if (normalized === "UNK") {
    return `<span class="dashboard-ip-country">${escapeHtml(countryName(normalized, deps))}</span>`;
  }
  const flag = countryFlag(normalized, deps);
  const name = countryName(normalized, deps);
  return `<span class="dashboard-ip-country">${escapeHtml(name)}${flag ? ` (${escapeHtml(flag)})` : ""}</span>`;
}

function countryFlag(code, deps) {
  const token = deps.normalizeCountryCode(code);
  if (!/^[A-Z]{2}$/.test(token)) {
    return "";
  }
  const first = 127397 + token.charCodeAt(0);
  const second = 127397 + token.charCodeAt(1);
  return String.fromCodePoint(first, second);
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

function renderContainersHealthWidget(overview, ctx, deps) {
  if (!overview || !Array.isArray(overview?.containers)) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.containers.empty"))}</div>`;
  }
  const containers = overview.containers
    .slice()
    .sort((left, right) => String(left?.name || "").localeCompare(String(right?.name || ""), undefined, { sensitivity: "base" }))
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
  const renderLabel = typeof options.renderLabel === "function" ? options.renderLabel : (item) => escapeHtml(String(item?.key || "-"));
  const renderMetaRight = typeof options.renderMetaRight === "function" ? options.renderMetaRight : () => "";
  const rowAttrs = typeof options.rowAttrs === "function" ? options.rowAttrs : () => "";
  const containerAttr = options.containerAction ? ` data-widget-action="${escapeHtml(options.containerAction)}"` : "";
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
      if (!ip) {
        return widgetAction ? `data-widget-action="${escapeHtml(widgetAction)}"` : "";
      }
      return `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"`;
    }
  }, deps);
}

function mergeWidgetData(stats, detailModel, containersOverview, ctx, deps) {
  const statsTopIPs = Array.isArray(stats?.top_attacker_ips) ? stats.top_attacker_ips : [];
  const statsTopCountries = Array.isArray(stats?.top_attacker_countries) ? stats.top_attacker_countries : [];
  const fallbackTopCountries = Array.isArray(detailModel?.attacksByCountry) ? detailModel.attacksByCountry : [];
  const topCountryItems = (statsTopCountries.length ? statsTopCountries : fallbackTopCountries).map((item) => ({
    key: item?.key,
    count: item?.count,
    countryCode: item?.key
  }));
  const topIPItems = statsTopIPs.map((item) => {
    const key = String(item?.key || "").trim();
    return {
      key,
      count: Number(item?.count || 0),
      countryCode: detailModel?.ipCountryByIP?.get?.(key) || "UNK"
    };
  });

  return {
    "services-up": renderMetric(stats?.services_up, ctx.t("dashboard.value.servicesUp"), "success", "services-up", deps),
    "services-down": renderMetric(stats?.services_down, ctx.t("dashboard.value.servicesDown"), "danger", "services-down", deps),
    "requests-day": renderMetric(stats?.requests_day, ctx.t("dashboard.value.requestsDay"), "success", "requests-day", deps),
    "attacks-day": renderMetric(stats?.attacks_day, ctx.t("dashboard.value.attacksDay"), "warning", "attacks-day", deps),
    "blocked-attacks": renderMetric(stats?.blocked_attacks_day, ctx.t("dashboard.value.blockedAttacksDay"), "danger", "blocked-attacks", deps),
    "unique-attackers": renderIPTopList(topIPItems.length ? topIPItems : (detailModel?.ipDetailsSummary || []).slice(0, 10).map((item) => ({
      key: item.ip,
      count: Math.max(item.attacks, item.requests),
      countryCode: item.countryCode
    })), ctx.t("dashboard.empty.topIPs"), "unique-attackers", deps),
    "popular-errors": renderTopList(stats?.popular_errors || [], ctx.t("dashboard.empty.popularErrors"), {
      containerAction: "popular-errors",
      rowAttrs: (item) => {
        const code = String(item?.key || "").trim();
        return code ? `data-widget-action="error-detail" data-error-code="${escapeHtml(code)}"` : "";
      }
    }, deps),
    "top-ips": renderIPTopList(topIPItems, ctx.t("dashboard.empty.topIPs"), "top-ips", deps),
    "top-countries": renderTopList(topCountryItems, ctx.t("dashboard.empty.topCountries"), {
      containerAction: "top-countries",
      renderLabel: (item) => renderCountryBadge(item?.countryCode || item?.key, deps),
      rowAttrs: (item) => `data-widget-action="country-detail" data-country-code="${escapeHtml(deps.normalizeCountryCode(item?.countryCode || item?.key))}"`
    }, deps),
    "top-urls": renderTopList((Array.isArray(stats?.most_attacked_urls) && stats.most_attacked_urls.length ? stats.most_attacked_urls : (detailModel?.attacksByURL || [])), ctx.t("dashboard.empty.topURLs"), {
      containerAction: "top-urls",
      rowAttrs: (item) => {
        const url = String(item?.key || "").trim();
        return url ? `data-widget-action="url-detail" data-url="${escapeHtml(url)}"` : "";
      }
    }, deps),
    memory: deps.renderSystemMemory(stats, ctx),
    cpu: deps.renderSystemCPU(stats, ctx),
    "containers-health": renderContainersHealthWidget(containersOverview, ctx, deps)
  };
}

export {
  renderCountryBadge,
  mergeWidgetData
};
