import { confirmAction, escapeHtml, formatDate, setError, setLoading } from "../ui.js";

const AUTO_BAN_EVENT_TYPES = new Set(["security_rate_limit", "security_access", "security_waf"]);
const AUTO_BAN_FALLBACK_SECONDS = 10;

const ICON_PLUS = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v14h-2zM5 11h14v2H5z"/></svg>';
const ICON_UNLOCK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v2h-2V7a3 3 0 1 0-6 0v2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-9a2 2 0 0 1 2-2h2V7a5 5 0 0 1 5-5Z"/></svg>';

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

async function tryGetJSON(path) {
  try {
    const response = await fetch(path, {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return null;
    }
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  } catch (error) {
    return null;
  }
}

function mergeByID(primary, secondary, idField = "id") {
  const map = new Map();
  for (const item of normalizeList(primary)) {
    const id = String(item?.[idField] || "").trim();
    if (!id) {
      continue;
    }
    map.set(id, { ...item, _origin: "primary" });
  }
  for (const item of normalizeList(secondary)) {
    const id = String(item?.[idField] || "").trim();
    if (!id || map.has(id)) {
      continue;
    }
    map.set(id, { ...item, _origin: "secondary" });
  }
  return Array.from(map.values());
}

function unwrapList(payload, keys = []) {
  if (Array.isArray(payload)) {
    return payload;
  }
  for (const key of keys) {
    if (Array.isArray(payload?.[key])) {
      return payload[key];
    }
  }
  return [];
}

function asDate(value) {
  const stamp = Date.parse(String(value || ""));
  return Number.isNaN(stamp) ? null : new Date(stamp);
}

function siteCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_${slug || "site"}`;
}

function clearRateLimitCookie(siteID) {
  const cookie = siteCookieName(siteID);
  document.cookie = `${cookie}=; Path=/; Max-Age=0; SameSite=Lax`;
}

function parseIP(details) {
  if (!details || typeof details !== "object") {
    return "";
  }
  return String(details.client_ip || details.ip || "").trim();
}

function parseCountry(details) {
  if (!details || typeof details !== "object") {
    return "-";
  }
  return String(details.country || details.geo_country || details.country_code || "").trim() || "-";
}

function moduleByEvent(item) {
  const type = String(item?.type || "").trim().toLowerCase();
  if (type === "security_rate_limit") {
    return "limits";
  }
  if (type === "security_waf") {
    return "ddos";
  }
  if (type === "security_access") {
    const status = Number(item?.details?.status || 0);
    if (status === 429) {
      return "bad_behavior";
    }
    if (status === 403 || status === 444) {
      return "headers_access";
    }
    return "access";
  }
  return "unknown";
}

function formatRemaining(expiresAt, now, t) {
  if (expiresAt === null) {
    return t("bans.time.permanent");
  }
  const diff = expiresAt.getTime() - now.getTime();
  if (diff <= 0) {
    return t("bans.time.expired");
  }
  const totalSec = Math.floor(diff / 1000);
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

function buildAutoBanRows(events, banSecondsBySite) {
  const now = new Date();
  const byKey = new Map();
  for (const item of normalizeList(events)) {
    const type = String(item?.type || "").trim().toLowerCase();
    if (!AUTO_BAN_EVENT_TYPES.has(type)) {
      continue;
    }
    const siteID = String(item?.site_id || "").trim();
    if (!siteID) {
      continue;
    }
    const ip = parseIP(item?.details);
    if (!ip) {
      continue;
    }
    const occurredAt = asDate(item?.occurred_at);
    if (!occurredAt) {
      continue;
    }
    const key = `${siteID}|${ip}`;
    const moduleID = moduleByEvent(item);
    const banSecondsRaw = Number.parseInt(String(banSecondsBySite.get(siteID) ?? AUTO_BAN_FALLBACK_SECONDS), 10);
    const banSeconds = Number.isFinite(banSecondsRaw) ? Math.max(0, banSecondsRaw) : AUTO_BAN_FALLBACK_SECONDS;
    const expiresAt = banSeconds === 0 ? null : new Date(occurredAt.getTime() + banSeconds * 1000);

    if (expiresAt && expiresAt.getTime() <= now.getTime()) {
      continue;
    }

    const existing = byKey.get(key);
    if (!existing || occurredAt.getTime() > existing.occurredAt.getTime()) {
      byKey.set(key, {
        siteID,
        ip,
        country: parseCountry(item?.details),
        source: "auto",
        occurredAt,
        expiresAt,
        modules: new Set([moduleID]),
        origin: item?._origin || "primary"
      });
      continue;
    }
    existing.modules.add(moduleID);
  }
  return Array.from(byKey.values());
}

function buildManualBanRows(accessPolicies) {
  const rows = [];
  for (const policy of normalizeList(accessPolicies)) {
    const siteID = String(policy?.site_id || "").trim();
    if (!siteID) {
      continue;
    }
    const updatedAt = asDate(policy?.updated_at);
    for (const ip of normalizeList(policy?.denylist)) {
      const value = String(ip || "").trim();
      if (!value) {
        continue;
      }
      rows.push({
        siteID,
        ip: value,
        country: "-",
        source: "manual",
        occurredAt: updatedAt,
        expiresAt: null,
        modules: new Set(["manual"]),
        origin: policy?._origin || "primary"
      });
    }
  }
  return rows;
}

function mergeRows(autoRows, manualRows) {
  const merged = new Map();
  for (const item of [...autoRows, ...manualRows]) {
    const key = `${item.origin}|${item.siteID}|${item.ip}`;
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, item);
      continue;
    }
    if (item.source === "manual") {
      merged.set(key, { ...existing, ...item, source: "manual", expiresAt: null, modules: new Set([...existing.modules, ...item.modules]) });
      continue;
    }
    if (!existing.occurredAt || (item.occurredAt && item.occurredAt.getTime() > existing.occurredAt.getTime())) {
      merged.set(key, { ...existing, ...item, modules: new Set([...existing.modules, ...item.modules]) });
      continue;
    }
    existing.modules = new Set([...existing.modules, ...item.modules]);
  }
  return Array.from(merged.values()).sort((a, b) => {
    const left = a.occurredAt ? a.occurredAt.getTime() : 0;
    const right = b.occurredAt ? b.occurredAt.getTime() : 0;
    return right - left;
  });
}

function renderModules(modules, t) {
  return Array.from(modules)
    .map((item) => t(`bans.module.${item}`))
    .join(", ");
}

async function callBanAction(row, action, payload) {
  const base = row.origin === "secondary" ? "/api-app/sites/" : "/api/sites/";
  const path = `${base}${encodeURIComponent(row.siteID)}/${action}`;
  if (row.origin === "secondary") {
    const response = await fetch(path, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `HTTP ${response.status}`);
    }
    return;
  }
  await payload.ctx.api.post(path, { ip: payload.ip });
}

export async function renderBans(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.bans"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("bans.subtitle"))}</div>
        </div>
        <button class="btn ghost btn-sm" id="bans-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
      </div>
      <div class="waf-card-body waf-stack">
        <div id="bans-status"></div>
        <div id="bans-list"></div>
      </div>
    </section>
  `;

  const statusNode = container.querySelector("#bans-status");
  const listNode = container.querySelector("#bans-list");

  const loadBanSecondsBySite = async (siteIDs) => {
    const out = new Map();
    const secondaryProfileDump = await tryGetJSON("/api-app/easy-site-profiles");
    const secondaryProfiles = unwrapList(secondaryProfileDump, ["easy_site_profiles"]);
    const secondaryBySite = new Map(secondaryProfiles.map((item) => [String(item?.site_id || "").trim(), item]));
    await Promise.all(siteIDs.map(async (siteID) => {
      try {
        const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`);
        const value = Number.parseInt(String(profile?.security_behavior_and_limits?.bad_behavior_ban_time_seconds ?? AUTO_BAN_FALLBACK_SECONDS), 10);
        out.set(siteID, Number.isFinite(value) ? Math.max(0, value) : AUTO_BAN_FALLBACK_SECONDS);
      } catch (error) {
        const secondary = secondaryBySite.get(siteID);
        const value = Number.parseInt(String(secondary?.security_behavior_and_limits?.bad_behavior_ban_time_seconds ?? AUTO_BAN_FALLBACK_SECONDS), 10);
        out.set(siteID, Number.isFinite(value) ? Math.max(0, value) : AUTO_BAN_FALLBACK_SECONDS);
      }
    }));
    return out;
  };

  const renderRows = async () => {
    setLoading(statusNode, ctx.t("bans.loading"));
    try {
      const [sitesResponse, accessResponse, eventsResponse, sitesSecondary, accessSecondary, eventsSecondary] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/access-policies"),
        ctx.api.get("/api/events"),
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/access-policies"),
        tryGetJSON("/api-app/events")
      ]);

      const sites = mergeByID(sitesResponse, unwrapList(sitesSecondary, ["sites"]), "id");
      const accessPolicies = mergeByID(accessResponse?.access_policies, unwrapList(accessSecondary, ["access_policies"]), "id");
      const events = [
        ...normalizeList(eventsResponse?.events).map((item) => ({ ...item, _origin: "primary" })),
        ...unwrapList(eventsSecondary, ["events"]).map((item) => ({ ...item, _origin: "secondary" }))
      ];

      const siteByID = new Map(sites.map((site) => [String(site?.id || ""), site]));
      const banSecondsBySite = await loadBanSecondsBySite(
        Array.from(new Set([
          ...sites.map((site) => String(site?.id || "").trim()).filter(Boolean),
          ...events.map((item) => String(item?.site_id || "").trim()).filter(Boolean),
          ...accessPolicies.map((item) => String(item?.site_id || "").trim()).filter(Boolean)
        ]))
      );

      const rows = mergeRows(buildAutoBanRows(events, banSecondsBySite), buildManualBanRows(accessPolicies));
      if (rows.length === 0) {
        statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.empty"))}</div>`;
        listNode.innerHTML = "";
        return;
      }

      statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.total"))}: ${rows.length}</div>`;

      const now = new Date();
      listNode.innerHTML = `
        <div class="waf-table-wrap">
          <table class="waf-table">
            <thead>
              <tr>
                <th>${escapeHtml(ctx.t("bans.col.ip"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.country"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.date"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.left"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.site"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.module"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.actions"))}</th>
              </tr>
            </thead>
            <tbody>
              ${rows.map((row, index) => {
                const site = siteByID.get(row.siteID);
                const siteLabel = site?.primary_host || row.siteID;
                return `
                  <tr>
                    <td>${escapeHtml(row.ip)}</td>
                    <td>${escapeHtml(row.country)}</td>
                    <td>${escapeHtml(row.occurredAt ? formatDate(row.occurredAt.toISOString()) : "-")}</td>
                    <td>${escapeHtml(formatRemaining(row.expiresAt, now, ctx.t))}</td>
                    <td>${escapeHtml(siteLabel)}</td>
                    <td>${escapeHtml(renderModules(row.modules, ctx.t))}</td>
                    <td>
                      <div class="waf-ban-actions">
                        <button type="button" class="btn success btn-sm waf-ban-btn" data-action="extend" data-row="${index}" title="${escapeHtml(ctx.t("bans.action.extend"))}">
                          <span class="waf-ban-btn-icon">${ICON_PLUS}</span>
                          <span>${escapeHtml(ctx.t("bans.action.extend"))}</span>
                        </button>
                        <button type="button" class="btn danger btn-sm waf-ban-btn" data-action="unban" data-row="${index}" title="${escapeHtml(ctx.t("bans.action.unban"))}">
                          <span class="waf-ban-btn-icon">${ICON_UNLOCK}</span>
                          <span>${escapeHtml(ctx.t("bans.action.unban"))}</span>
                        </button>
                      </div>
                    </td>
                  </tr>
                `;
              }).join("")}
            </tbody>
          </table>
        </div>
      `;

      listNode.querySelectorAll(".waf-ban-btn").forEach((button) => {
        button.addEventListener("click", async () => {
          const action = String(button.dataset.action || "");
          const rowIndex = Number.parseInt(String(button.dataset.row || "-1"), 10);
          const row = rows[rowIndex];
          if (!row || !row.siteID || !row.ip) {
            return;
          }

          if (action === "unban") {
            if (!confirmAction(ctx.t("bans.confirm.unban", { ip: row.ip, site: row.siteID }))) {
              return;
            }
            try {
              button.disabled = true;
              if (row.origin === "secondary") {
                const response = await fetch(`/api-app/sites/${encodeURIComponent(row.siteID)}/unban`, {
                  method: "POST",
                  credentials: "include",
                  headers: { "Content-Type": "application/json", Accept: "application/json" },
                  body: JSON.stringify({ ip: row.ip })
                });
                if (!response.ok) {
                  throw new Error(`HTTP ${response.status}`);
                }
              } else {
                await ctx.api.post(`/api/sites/${encodeURIComponent(row.siteID)}/unban`, { ip: row.ip });
              }
              clearRateLimitCookie(row.siteID);
              ctx.notify(ctx.t("toast.ipUnbanned"));
              await renderRows();
            } catch (error) {
              setError(statusNode, error?.message || ctx.t("bans.error.action"));
            } finally {
              button.disabled = false;
            }
            return;
          }

          if (!confirmAction(ctx.t("bans.confirm.extend", { ip: row.ip, site: row.siteID }))) {
            return;
          }
          try {
            button.disabled = true;
            if (row.origin === "secondary") {
              const response = await fetch(`/api-app/sites/${encodeURIComponent(row.siteID)}/ban`, {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json", Accept: "application/json" },
                body: JSON.stringify({ ip: row.ip })
              });
              if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
              }
            } else {
              await ctx.api.post(`/api/sites/${encodeURIComponent(row.siteID)}/ban`, { ip: row.ip });
            }
            ctx.notify(ctx.t("toast.ipBanned"));
            await renderRows();
          } catch (error) {
            setError(statusNode, error?.message || ctx.t("bans.error.action"));
          } finally {
            button.disabled = false;
          }
        });
      });
    } catch (error) {
      setError(statusNode, ctx.t("bans.error.load"));
      listNode.innerHTML = "";
    }
  };

  container.querySelector("#bans-refresh")?.addEventListener("click", () => {
    renderRows();
  });

  await renderRows();
}
