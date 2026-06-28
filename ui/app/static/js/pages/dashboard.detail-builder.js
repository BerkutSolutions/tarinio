import { escapeHtml } from "../ui.js";
import { renderSummaryMetrics, renderDetailTable, renderDetailSection } from "./dashboard.detail-shared.js";
import { topCounts } from "./dashboard.detail-model-helpers.js";

function countryFlag(code, deps) {
  const token = deps.normalizeCountryCode(code);
  if (!/^[A-Z]{2}$/.test(token)) return "";
  const cc = token.toLowerCase();
  return `<img class="country-flag-img" src="https://flagcdn.com/16x12/${cc}.png" srcset="https://flagcdn.com/32x24/${cc}.png 2x" width="16" height="12" alt="${token}" loading="lazy" onerror="this.style.display='none';this.nextSibling&&(this.nextSibling.style.display='')"><span class="country-flag-fallback" style="display:none">${token}</span>`;
}

function countryName(code, deps) {
  const token = deps.normalizeCountryCode(code);
  if (token === "UNK") return "Unknown";
  try {
    const names = new Intl.DisplayNames(["ru", "en"], { type: "region" });
    return names.of(token) || token;
  } catch (_error) {
    return token;
  }
}

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

function formatAttackURLLabel(item) {
  const url = escapeHtml(String(item?.key || "-"));
  const sites = Array.isArray(item?.sites) ? item.sites : [];
  if (!sites.length) {
    return url;
  }
  return `
    <div><strong>${url}</strong></div>
    <div class="muted">${escapeHtml(sites.map((site) => String(site?.key || "-")).join(", "))}</div>
  `;
}

function renderServiceMiniDashboard(service, stats, detailModel, ctx, deps) {
  const name           = String(service?.name || "");
  const isUp           = Boolean(service?.up);
  const attacksBySite  = Array.isArray(detailModel?.attacksBySite)  ? detailModel.attacksBySite  : [];
  const blockedBySite  = Array.isArray(detailModel?.blockedBySite)  ? detailModel.blockedBySite  : [];
  const requestsBySite = Array.isArray(detailModel?.requestsBySite) ? detailModel.requestsBySite : [];
  const attackCount    = Number(attacksBySite.find((i)  => i.key === name)?.count || 0);
  const blockedCount   = Number(blockedBySite.find((i)  => i.key === name)?.count || 0);
  const requestCount   = Number(requestsBySite.find((i) => i.key === name)?.count || 0);
  const ipSummary      = Array.isArray(detailModel?.ipDetailsSummary) ? detailModel.ipDetailsSummary : [];
  const topAttackers   = ipSummary
    .filter((item) => (Array.isArray(item.sites) ? item.sites : []).some((s) => s.key === name))
    .slice(0, 10)
    .map((item) => ({ key: item.ip, count: item.attacks || item.requests || 0, countryCode: item.countryCode }));
  const attacksByURL  = Array.isArray(detailModel?.attacksByURL) ? detailModel.attacksByURL.slice(0, 10) : [];
  const tone          = isUp ? "success" : "danger";
  const statusLabel   = isUp ? ctx.t("dashboard.services.statusUp") : ctx.t("dashboard.services.statusDown");
  const checkedAt     = formatCheckedAt(service?.checked_at);
  return `
    <div class="dashboard-service-detail">
      <div class="dashboard-service-detail-header">
        <span class="badge badge-${escapeHtml(tone)}">${escapeHtml(statusLabel)}</span>
        ${checkedAt ? `<span class="muted" style="font-size:12px">${escapeHtml(ctx.t("dashboard.services.checkedAt"))}: ${escapeHtml(checkedAt)}</span>` : ""}
      </div>
      ${!isUp ? `<div class="alert warning" style="margin:8px 0 4px">${escapeHtml(ctx.t("dashboard.services.hostDown"))}</div>` : ""}
      ${renderSummaryMetrics([
        { labelKey: "dashboard.detail.requests", value: requestCount },
        { labelKey: "dashboard.detail.attacks",  value: attackCount  },
        { labelKey: "dashboard.detail.blocked",  value: blockedCount }
      ], ctx, deps)}
      ${renderDetailSection(
        ctx.t("dashboard.services.topAttackers"),
        topAttackers.length
          ? renderDetailTable(topAttackers, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.attacks"), {
              labelFormatter: (item) => `${escapeHtml(String(item?.key || "-"))} ${deps.renderCountryBadge(item?.countryCode)}`,
              rowAttrs: (item) => {
                const ip = String(item?.key || "").trim();
                return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
              }
            }, deps)
          : `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.empty.topIPs"))}</div>`
      )}
      ${renderDetailSection(
        ctx.t("dashboard.services.topURLs"),
        attacksByURL.length
          ? renderDetailTable(attacksByURL, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.attacks"), {
              labelFormatter: (item) => formatAttackURLLabel(item)
            }, deps)
          : `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.empty.topURLs"))}</div>`
      )}
    </div>
  `;
}

function buildWidgetDetail(action, payload, stats, detailModel, containersOverview, ctx, deps) {
  const SYSTEM_SERVICES = new Set(["control-plane", "runtime"]);
  const services        = (Array.isArray(stats?.services) ? stats.services : [])
    .filter((s) => !SYSTEM_SERVICES.has(String(s?.name || "").trim().toLowerCase()));
  const attackBySiteMap = new Map((detailModel?.attacksBySite || []).map((item) => [item.key, item.count]));

  if (action === "traffic-summary") {
    return buildWidgetDetail("requests-day", payload, stats, detailModel, containersOverview, ctx, deps);
  }

  if (action === "services") {
    const upCount   = services.filter((s) => Boolean(s?.up)).length;
    const downCount = services.length - upCount;
    return {
      title:    ctx.t("dashboard.widget.services"),
      subtitle: ctx.t("dashboard.services.subtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.services.up",    value: upCount         },
        { labelKey: "dashboard.services.down",  value: downCount       },
        { labelKey: "dashboard.services.total", value: services.length }
      ], ctx, deps) +
      renderDetailTable(
        services.map((item) => ({ key: item?.name || "-", count: 1, up: Boolean(item?.up), checked_at: item?.checked_at || "" })),
        ctx,
        ctx.t("dashboard.services.service"),
        ctx.t("dashboard.detail.state"),
        {
          labelFormatter: (item) => {
            const tone = item.up ? "success" : "warning";
            const lbl  = item.up ? ctx.t("dashboard.services.statusUp") : ctx.t("dashboard.services.statusDown");
            return `<strong>${escapeHtml(String(item?.key || "-"))}</strong> <span class="badge badge-${escapeHtml(tone)}">${escapeHtml(lbl)}</span>`;
          },
          countFormatter: (item) => escapeHtml(formatCheckedAt(item?.checked_at) || "-"),
          rowAttrs: (item) => {
            const nm = String(item?.key || "").trim();
            return nm ? `data-widget-action="service-detail" data-service-name="${escapeHtml(nm)}"` : "";
          }
        },
        deps
      )
    };
  }

  if (action === "service-detail") {
    const serviceName = String(payload?.serviceName || "").trim();
    const service     = services.find((s) => String(s?.name || "") === serviceName);
    if (!service) {
      return {
        title: ctx.t("dashboard.widget.services"), subtitle: serviceName || "-",
        body: `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`
      };
    }
    return {
      title:    String(service?.name || "-"),
      subtitle: ctx.t("dashboard.services.detailSubtitle"),
      body:     renderServiceMiniDashboard(service, stats, detailModel, ctx, deps)
    };
  }

  if (action === "requests-day") {
    const requestTopSites = Array.isArray(stats?.request_top_sites) && stats.request_top_sites.length
      ? stats.request_top_sites
      : (detailModel?.requestsBySite || []);
    const requestTopURLs = Array.isArray(stats?.request_top_urls) && stats.request_top_urls.length
      ? stats.request_top_urls
      : (detailModel?.requestsByURL || []);
    const uniqueRequestIPs = Number(stats?.request_unique_ips_day || 0) || Number(detailModel?.requestsUniqueIPs || 0);
    return {
      title:    ctx.t("dashboard.widget.requestsDay"),
      subtitle: ctx.t("dashboard.detail.requestsSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.value.requestsDay", value: stats?.requests_day || 0              },
        { labelKey: "dashboard.detail.uniqueIPs",  value: uniqueRequestIPs }
      ], ctx, deps) +
      renderDetailTable(requestTopSites, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests"), {
        labelFormatter: (item) => {
          const raw = String(item?.key || "-");
          const label = raw === "_global" ? escapeHtml(ctx.t("dashboard.global.siteLabel")) : escapeHtml(raw);
          return `<strong>${label}</strong>`;
        },
        rowAttrs: (item) => {
          const siteId = String(item?.key || "").trim();
          return siteId ? `data-widget-action="site-detail" data-site-id="${escapeHtml(siteId)}"` : "";
        }
      }, deps) +
      renderDetailTable(requestTopURLs,  ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.requests"), {
        labelFormatter: (item) => {
          const url = escapeHtml(String(item?.key || "-"));
          const sites = Array.isArray(item?.sites) ? item.sites : [];
          if (!sites.length) return `<strong>${url}</strong>`;
          return `<div><strong>${url}</strong></div><div class="muted">${escapeHtml(sites.map((s) => String(s?.key || "-")).join(", "))}</div>`;
        },
        rowAttrs: (item) => {
          const url = String(item?.key || "").trim();
          return url ? `data-widget-action="url-detail" data-url="${escapeHtml(url)}"` : "";
        }
      }, deps)
    };
  }

  if (action === "site-detail") {
    const targetSite = String(payload?.siteId || "").trim();
    const siteLabel = targetSite === "_global"
      ? ctx.t("dashboard.global.siteLabel")
      : (targetSite || "-");
    const urlMap = detailModel?.requestsURLsBySite?.get?.(targetSite) || new Map();
    const urlRows = topCounts(urlMap, 20);
    const urlTable = renderDetailTable(urlRows, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.requests"), {
      labelFormatter: (item) => `<strong>${escapeHtml(String(item?.key || "-"))}</strong>`
    }, deps);
    return {
      title:    siteLabel,
      subtitle: ctx.t("dashboard.detail.siteRequestsSubtitle"),
      body:     urlTable || `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.detail.noData"))}</div>`
    };
  }

  if (action === "attacks-day") {
    return {
      title:    ctx.t("dashboard.widget.attacksDay"),
      subtitle: ctx.t("dashboard.detail.attacksSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.value.attacksDay",        value: stats?.attacks_day         || 0 },
        { labelKey: "dashboard.value.blockedAttacksDay",  value: stats?.blocked_attacks_day || 0 }
      ], ctx, deps) +
      renderDetailTable(detailModel?.attacksBySite || [], ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.attacks"), {}, deps) +
      renderDetailTable(detailModel?.attacksByURL  || [], ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.attacks"), {
        labelFormatter: (item) => formatAttackURLLabel(item)
      }, deps)
    };
  }

  if (action === "blocked-attacks") {
    const rows = (detailModel?.blockedBySite || []).map((item) => ({
      key: item.key, count: item.count, total: attackBySiteMap.get(item.key) || 0
    }));
    const blockedIPs = (detailModel?.ipDetailsSummary || [])
      .filter((item) => Number(item?.blocked || 0) > 0)
      .slice(0, 20)
      .map((item) => ({ key: item.ip, count: item.blocked, countryCode: item.countryCode }));
    return {
      title:    ctx.t("dashboard.widget.blockedAttacks"),
      subtitle: ctx.t("dashboard.detail.blockedSubtitle"),
      body: renderDetailTable(rows, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.blocked"), {
        labelFormatter: (item) => {
          const pct = item.total > 0 ? `${((item.count * 100) / item.total).toFixed(1)}%` : "0%";
          return `${escapeHtml(item.key)} <span class="muted">(${escapeHtml(pct)})</span>`;
        }
      }, deps) +
      renderDetailTable(blockedIPs, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.blocked"), {
        labelFormatter: (item) => `${escapeHtml(String(item?.key || "-"))} ${deps.renderCountryBadge(item?.countryCode)}`,
        rowAttrs: (item) => {
          const ip = String(item?.key || "").trim();
          return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
        }
      }, deps)
    };
  }

  if (action === "popular-errors") {
    return {
      title:    ctx.t("dashboard.widget.popularErrors"),
      subtitle: ctx.t("dashboard.detail.errorsSubtitle"),
      body: renderDetailTable(detailModel?.errorsByCode || stats?.popular_errors || [], ctx, ctx.t("dashboard.detail.errorCode"), ctx.t("dashboard.detail.requests"), {
        rowAttrs: (item) => {
          const code = String(item?.key || "").trim();
          return code ? `data-widget-action="error-detail" data-error-code="${escapeHtml(code)}"` : "";
        }
      }, deps)
    };
  }

  if (action === "error-detail") {
    const code = String(payload?.errorCode || "").trim();
    return {
      title:    `${ctx.t("dashboard.detail.errorCode")} ${code || "-"}`,
      subtitle: ctx.t("dashboard.detail.errorsBySiteSubtitle"),
      body: renderDetailTable(detailModel?.errorsByCodeSites?.get?.(code) || [], ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests"), {}, deps)
    };
  }

  if (action === "top-ips" || action === "unique-attackers") {
    const items = (detailModel?.ipDetailsSummary || []).slice(0, 20).map((item) => ({
      key: item.ip, count: Math.max(item.attacks, item.requests), countryCode: item.countryCode
    }));
    return {
      title:    ctx.t("dashboard.widget.topIPs"),
      subtitle: ctx.t("dashboard.detail.topIPsSubtitle"),
      body: renderDetailTable(items, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.requests"), {
        labelFormatter: (item) => `${escapeHtml(String(item?.key || "-"))} ${deps.renderCountryBadge(item?.countryCode)}`,
        rowAttrs: (item) => {
          const ip = String(item?.key || "").trim();
          return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
        }
      }, deps)
    };
  }

  if (action === "ip-detail") {
    const ip     = String(payload?.ip || "").trim();
    const detail = detailModel?.ipDetailsByIP?.get?.(ip);
    if (!detail) {
      return { title: `${ctx.t("dashboard.detail.ip")} ${ip || "-"}`, subtitle: ctx.t("dashboard.detail.ipSubtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>` };
    }
    return {
      title:    `${ctx.t("dashboard.detail.ip")} ${detail.ip}`,
      subtitle: ctx.t("dashboard.detail.ipSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.detail.requests", value: detail.requests },
        { labelKey: "dashboard.detail.attacks",  value: detail.attacks  },
        { labelKey: "dashboard.detail.blocked",  value: detail.blocked  }
      ], ctx, deps) +
      `<div class="dashboard-ip-country-block">${deps.renderCountryBadge(detail.countryCode)}</div>` +
      renderDetailTable(detail.sites,   ctx, ctx.t("dashboard.detail.site"),   ctx.t("dashboard.detail.requests"), {}, deps) +
      renderDetailTable(detail.pages,   ctx, ctx.t("dashboard.detail.page"),   ctx.t("dashboard.detail.requests"), {}, deps) +
      renderDetailTable(detail.methods, ctx, ctx.t("dashboard.detail.method"), ctx.t("dashboard.detail.requests"), {}, deps)
    };
  }

  if (action === "top-countries" || action === "country-detail") {
    const code = deps.normalizeCountryCode(payload?.countryCode || "");
    const countryRow = code && code !== "UNK"
      ? (detailModel?.attacksByCountry || []).find((item) => deps.normalizeCountryCode(item.key) === code)
      : null;
    const totalAttacks = countryRow?.count || 0;

    // IP из этой страны
    const ipRows = code && code !== "UNK"
      ? (detailModel?.ipDetailsSummary || [])
          .filter((item) => deps.normalizeCountryCode(item.countryCode || "") === code)
          .sort((a, b) => (b.attacks || 0) - (a.attacks || 0))
          .slice(0, 20)
          .map((item) => ({ key: item.ip, count: item.attacks || item.requests || 0 }))
      : (detailModel?.attacksByCountry || []).map((item) => ({ key: item.key, count: item.count }));

    const flag = code && code !== "UNK" ? countryFlag(code, deps) : "";
    const name = code && code !== "UNK" ? countryName(code, deps) : ctx.t("dashboard.widget.topCountries");
    const titleHtml = flag
      ? `${escapeHtml(ctx.t("dashboard.detail.country"))} ${escapeHtml(name)} ${flag}`
      : escapeHtml(code && code !== "UNK" ? `${ctx.t("dashboard.detail.country")} ${name}` : ctx.t("dashboard.widget.topCountries"));

    const totalHtml = code && code !== "UNK" && totalAttacks
      ? `<div class="dashboard-detail-total">${escapeHtml(ctx.t("dashboard.detail.attacks"))}: <strong>${escapeHtml(String(deps.formatNumber(totalAttacks)))}</strong></div>`
      : "";

    const ipTable = renderDetailTable(ipRows, ctx, "IP", ctx.t("dashboard.detail.attacks"), {
      labelFormatter: (item) => escapeHtml(String(item?.key || "-"))
    }, deps);

    return {
      title:    `${ctx.t("dashboard.detail.country")} ${name}`,
      titleHtml,
      subtitle: ctx.t("dashboard.detail.countrySubtitle"),
      body:     totalHtml + ipTable
    };
  }

  if (action === "top-urls" || action === "url-detail") {
    const targetURL = String(payload?.url || "").trim();
    const urlRow = targetURL
      ? (detailModel?.attacksByURL || []).find((item) => item.key === targetURL)
      : null;

    // IP которые ломились на этот URL
    const ipRows = (urlRow?.ips || []).map((item) => {
      const ipDetail = detailModel?.ipDetailsByIP?.get?.(item.key);
      const countryCode = ipDetail?.countryCode || "";
      const sites = (ipDetail?.sites || [])
        .filter((s) => !targetURL || true)
        .slice(0, 3)
        .map((s) => s.key)
        .join(", ");
      return { key: item.key, count: item.count, countryCode, sites };
    });

    const rows = targetURL
      ? (detailModel?.attacksByURL || []).filter((item) => item.key === targetURL)
      : (detailModel?.attacksByURL || []);

    const urlTable = renderDetailTable(rows, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.attacks"), {
      labelFormatter: (item) => formatAttackURLLabel(item)
    }, deps);

    const ipTable = ipRows.length
      ? renderDetailSection(
          ctx.t("dashboard.widget.topIPs"),
          renderDetailTable(ipRows, ctx, "IP", ctx.t("dashboard.detail.attacks"), {
            labelFormatter: (item) => {
              const flag = item.countryCode ? countryFlag(item.countryCode, deps) : "";
              const sitesHtml = item.sites ? `<br><span class="dashboard-detail-sub">${escapeHtml(item.sites)}</span>` : "";
              return `${escapeHtml(String(item?.key || "-"))}${flag ? " " + flag : ""}${sitesHtml}`;
            }
          }, deps)
        )
      : "";

    return {
      title:    targetURL ? `${ctx.t("dashboard.detail.page")}: ${targetURL}` : ctx.t("dashboard.widget.topURLs"),
      subtitle: ctx.t("dashboard.detail.urlSubtitle"),
      body:     (targetURL ? "" : urlTable) + ipTable
    };
  }

  if (action === "memory" || action === "cpu") {
    const system  = stats?.system || {};
    const metrics = action === "cpu"
      ? [
          { labelKey: "dashboard.detail.cpuLoad",       value: deps.formatPercent(system.cpu_load_percent || 0) },
          { labelKey: "dashboard.system.cpuCores",       value: system.cpu_cores  || 0 },
          { labelKey: "dashboard.system.goroutines",     value: system.goroutines || 0 }
        ]
      : [
          { labelKey: "dashboard.detail.memoryUsedBytes",  value: deps.formatBytes(system.memory_used_bytes  || 0) },
          { labelKey: "dashboard.detail.memoryFreeBytes",  value: deps.formatBytes(system.memory_free_bytes  || 0) },
          { labelKey: "dashboard.detail.memoryTotalBytes", value: deps.formatBytes(system.memory_total_bytes || 0) }
        ];
    const processList  = action === "cpu"
      ? (Array.isArray(system?.top_cpu_processes)    ? system.top_cpu_processes    : [])
      : (Array.isArray(system?.top_memory_processes) ? system.top_memory_processes : []);
    const processRows  = processList.map((item) => ({ ...item, key: item?.name || item?.command || `pid-${item?.pid || 0}` }));
    const sectionTitle = action === "cpu" ? "dashboard.detail.processesByCPU"  : "dashboard.detail.processesByMemory";
    const countTitle   = action === "cpu" ? "dashboard.detail.cpuPercent"       : "dashboard.detail.memoryUsedBytes";
    return {
      title:    ctx.t(action === "cpu" ? "dashboard.widget.cpu" : "dashboard.widget.memory"),
      subtitle: ctx.t("dashboard.detail.loadSubtitle"),
      body: renderSummaryMetrics(metrics, ctx, deps) +
        renderDetailSection(
          ctx.t(sectionTitle),
          renderDetailTable(processRows, ctx, ctx.t("dashboard.detail.process"), ctx.t(countTitle), {
            labelFormatter: (item) => `
              <div><strong>${escapeHtml(String(item?.name || item?.command || "-"))}</strong></div>
              <div class="muted">PID ${escapeHtml(String(item?.pid || 0))} | ${escapeHtml(ctx.t("dashboard.detail.threads"))}: ${escapeHtml(deps.formatNumber(item?.threads || 0))} | ${escapeHtml(ctx.t("dashboard.detail.state"))}: ${escapeHtml(String(item?.state || "-"))}</div>
              <div class="muted">${escapeHtml(String(item?.command || item?.name || "-"))}</div>
            `,
            countFormatter: (item) => action === "cpu"
              ? escapeHtml(deps.formatPercent(item?.cpu_percent || 0))
              : `${escapeHtml(deps.formatBytes(item?.memory_rss_bytes || 0))} <span class="muted">(${escapeHtml(deps.formatPercent(item?.memory_percent || 0))})</span>`
          }, deps)
        )
    };
  }

  if (action === "containers-health") {
    const overview = containersOverview;
    if (!overview || !Array.isArray(overview?.containers)) {
      return {
        title:    ctx.t("dashboard.widget.containersHealth"),
        subtitle: ctx.t("dashboard.containers.subtitle"),
        body: `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.containers.empty"))}</div>`
      };
    }
    const rows = overview.containers
      .slice()
      .sort((l, r) => String(l?.name || "").localeCompare(String(r?.name || ""), undefined, { sensitivity: "base" }))
      .map((item) => ({ ...item, key: item?.name || "-", count: item?.cpu_percent || 0 }));
    return {
      title:    ctx.t("dashboard.widget.containersHealth"),
      subtitle: ctx.t("dashboard.containers.subtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.containers.uptime",  value: deps.formatUptimeLocalized(overview?.host_uptime_seconds || 0, ctx) },
        { labelKey: "dashboard.containers.cpu",     value: deps.formatPercent(overview?.total_cpu_percent  || 0) },
        { labelKey: "dashboard.containers.memory",  value: deps.formatPercent(overview?.avg_memory_percent || 0) },
        { labelKey: "dashboard.containers.network", value: `${overview?.total_network_in_text || "0 B"} / ${overview?.total_network_out_text || "0 B"}` }
      ], ctx, deps) +
      renderDetailTable(rows, ctx, ctx.t("dashboard.containers.container"), ctx.t("dashboard.containers.cpu"), {
        labelFormatter: (item) => `
          <div><strong>${escapeHtml(String(item?.name || "-"))}</strong></div>
          <div class="muted">${escapeHtml(deps.formatContainerStatusLabel(item?.status || "-", ctx))}</div>
          <div class="muted">MEM ${escapeHtml(deps.formatPercent(item?.memory_percent || 0))} | NET ${escapeHtml(String(item?.network_in_text || "0 B"))} / ${escapeHtml(String(item?.network_out_text || "0 B"))}</div>
        `,
        rowAttrs: (item) => {
          const name = String(item?.name || "").trim();
          if (!name) return "";
          return `data-status-tone="${escapeHtml(deps.getContainerStatusTone(item))}" data-widget-action="container-logs" data-container-name="${escapeHtml(name)}"`;
        },
        countFormatter: (item) => `CPU ${deps.formatPercent(item?.cpu_percent || 0)}`
      }, deps)
    };
  }

  return { title: ctx.t("dashboard.detail.title"), subtitle: ctx.t("dashboard.detail.subtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>` };
}

export { buildWidgetDetail };
