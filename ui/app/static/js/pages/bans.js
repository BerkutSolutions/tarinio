import { confirmAction, escapeHtml, formatDate, setError, setLoading } from "../ui.js";

const ICON_PLUS = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v14h-2zM5 11h14v2H5z"/></svg>';
const ICON_UNLOCK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v2h-2V7a3 3 0 1 0-6 0v2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-9a2 2 0 0 1 2-2h2V7a5 5 0 0 1 5-5Z"/></svg>';

function normalizeIP(value) {
  return String(value || "").trim();
}

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function buildPageButtons(totalPages, currentPage, dataAttr) {
  const pages = [];
  for (let page = 1; page <= Math.min(10, totalPages); page += 1) {
    pages.push(`<button type="button" class="btn ghost btn-sm${page === currentPage ? " active" : ""}" ${dataAttr}="${page}">${page}</button>`);
  }
  if (totalPages > 10) {
    pages.push(`<span class="muted">...</span>`);
    pages.push(`<button type="button" class="btn ghost btn-sm${totalPages === currentPage ? " active" : ""}" ${dataAttr}="${totalPages}">${totalPages}</button>`);
  }
  return pages.join("");
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
  } catch (_error) {
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

function normalizeSiteToken(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+/, "")
    .replace(/-+$/, "");
}

function buildSiteAliasMap(sites) {
  const map = new Map();
  for (const site of normalizeList(sites)) {
    const canonical = String(site?.id || "").trim();
    if (!canonical) {
      continue;
    }
    const lower = canonical.toLowerCase();
    const token = normalizeSiteToken(canonical);
    map.set(lower, canonical);
    if (token) {
      map.set(token, canonical);
      map.set(token.replace(/-/g, "_"), canonical);
    }
  }
  return map;
}

function buildHostSiteMap(sites) {
  const map = new Map();
  for (const site of normalizeList(sites)) {
    const id = String(site?.id || "").trim();
    const host = String(site?.primary_host || "").trim().toLowerCase();
    if (!id) {
      continue;
    }
    map.set(id.toLowerCase(), id);
    if (host) {
      map.set(host, id);
    }
  }
  return map;
}

function resolveCanonicalSiteID(siteID, aliasMap, hostMap, hostHint = "") {
  const raw = String(siteID || "").trim();
  if (raw) {
    const lower = raw.toLowerCase();
    const direct = aliasMap.get(lower) || aliasMap.get(normalizeSiteToken(raw)) || hostMap.get(lower);
    if (direct) {
      return direct;
    }
    if (raw !== "-" && raw !== "unknown") {
      return raw;
    }
  }
  const host = String(hostHint || "").trim().toLowerCase();
  if (host) {
    return hostMap.get(host) || "";
  }
  return "";
}

function siteCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_${slug || "site"}`;
}

function siteEscalationCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_escalation_${slug || "site"}`;
}

function clearRateLimitCookies(siteID) {
  const rateCookie = siteCookieName(siteID);
  const escalationCookie = siteEscalationCookieName(siteID);
  document.cookie = `${rateCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
  document.cookie = `${escalationCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
}

function buildIPSetBySite(accessPolicies, fieldName, resolveSiteID) {
  const out = new Map();
  for (const policy of normalizeList(accessPolicies)) {
    const siteID = resolveSiteID(String(policy?.site_id || "").trim());
    if (!siteID) {
      continue;
    }
    const set = out.get(siteID) || new Set();
    for (const value of normalizeList(policy?.[fieldName])) {
      const ip = normalizeIP(value);
      if (ip) {
        set.add(ip);
      }
    }
    out.set(siteID, set);
  }
  return out;
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

function buildManualBanRows(accessPolicies, resolveSiteID) {
  const rows = [];
  for (const policy of normalizeList(accessPolicies)) {
    const siteID = resolveSiteID(String(policy?.site_id || "").trim());
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

function renderModules(modules, t) {
  return Array.from(modules)
    .map((item) => t(`bans.module.${item}`))
    .join(", ");
}

async function postBanAction(ctx, siteID, origin, action, ip) {
  const base = origin === "secondary" ? "/api-app/sites/" : "/api/sites/";
  const path = `${base}${encodeURIComponent(siteID)}/${action}`;
  if (origin === "secondary") {
    const response = await fetch(path, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ ip })
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `HTTP ${response.status}`);
    }
    return;
  }
  await ctx.api.post(path, { ip });
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
  const pagingState = {
    pageSize: 10,
    page: 1
  };

  const getPaginationMeta = (total) => {
    const totalPages = Math.max(1, Math.ceil(total / pagingState.pageSize));
    if (pagingState.page > totalPages) {
      pagingState.page = totalPages;
    }
    if (pagingState.page < 1) {
      pagingState.page = 1;
    }
    const start = total === 0 ? 0 : (pagingState.page - 1) * pagingState.pageSize;
    const end = Math.min(start + pagingState.pageSize, total);
    return { totalPages, start, end };
  };

  const renderRows = async () => {
    setLoading(statusNode, ctx.t("bans.loading"));
    try {
      const [sitesResponse, accessResponse, sitesSecondary, accessSecondary] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/access-policies"),
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/access-policies")
      ]);

      const sites = mergeByID(sitesResponse, unwrapList(sitesSecondary, ["sites"]), "id");
      const accessPolicies = mergeByID(accessResponse?.access_policies, unwrapList(accessSecondary, ["access_policies"]), "id");

      const siteByID = new Map(sites.map((site) => [String(site?.id || ""), site]));
      const siteAliasMap = buildSiteAliasMap(sites);
      const hostSiteMap = buildHostSiteMap(sites);
      const canonicalSiteID = (value, hostHint = "") => resolveCanonicalSiteID(value, siteAliasMap, hostSiteMap, hostHint);
      const allowlistBySite = buildIPSetBySite(accessPolicies, "allowlist", canonicalSiteID);
      const rows = buildManualBanRows(accessPolicies, canonicalSiteID).sort((a, b) => {
        const left = a.occurredAt ? a.occurredAt.getTime() : 0;
        const right = b.occurredAt ? b.occurredAt.getTime() : 0;
        return right - left;
      });

      if (rows.length === 0) {
        statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.empty"))}</div>`;
        listNode.innerHTML = "";
        return;
      }

      statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.total"))}: ${rows.length}</div>`;

      const now = new Date();
      const meta = getPaginationMeta(rows.length);
      const pageRows = rows.slice(meta.start, meta.end);

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
              ${pageRows.map((row, index) => {
                const site = siteByID.get(row.siteID);
                const siteLabel = site?.primary_host || row.siteID;
                return `
                  <tr>
                    <td>${escapeHtml(row.ip)}</td>
                    <td>${escapeHtml(row.country)}</td>
                    <td>${escapeHtml(row.occurredAt ? formatDate(row.occurredAt.toISOString()) : "-")}</td>
                    <td>${escapeHtml(formatRemaining(row.expiresAt, now, ctx.t))}</td>
                    <td>${escapeHtml(siteLabel)}</td>
                    <td>${escapeHtml(renderModules(row.modules, ctx.t))}${(allowlistBySite.get(row.siteID) || new Set()).has(row.ip) ? " (allowlist)" : ""}</td>
                    <td>
                      <div class="waf-ban-actions">
                        <button type="button" class="btn success btn-sm waf-ban-btn" data-action="extend" data-row="${meta.start + index}" title="${escapeHtml(ctx.t("bans.action.extend"))}">
                          <span class="waf-ban-btn-icon">${ICON_PLUS}</span>
                          <span>${escapeHtml(ctx.t("bans.action.extend"))}</span>
                        </button>
                        <button type="button" class="btn danger btn-sm waf-ban-btn" data-action="unban" data-row="${meta.start + index}" title="${escapeHtml(ctx.t("bans.action.unban"))}">
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
        <div class="waf-pager">
          <div class="waf-inline">
            <label for="bans-page-size">${escapeHtml(ctx.t("activity.filter.pageSize"))}</label>
            <select id="bans-page-size">
              <option value="10"${pagingState.pageSize === 10 ? " selected" : ""}>10</option>
              <option value="25"${pagingState.pageSize === 25 ? " selected" : ""}>25</option>
              <option value="50"${pagingState.pageSize === 50 ? " selected" : ""}>50</option>
              <option value="100"${pagingState.pageSize === 100 ? " selected" : ""}>100</option>
            </select>
          </div>
          <div class="waf-actions">
            ${buildPageButtons(meta.totalPages, pagingState.page, "data-bans-page")}
          </div>
        </div>
      `;

      listNode.querySelector("#bans-page-size")?.addEventListener("change", async (event) => {
        const nextSize = Number.parseInt(String(event.target?.value || "10"), 10);
        if (!Number.isFinite(nextSize) || nextSize <= 0) {
          return;
        }
        pagingState.pageSize = nextSize;
        pagingState.page = 1;
        await renderRows();
      });

      listNode.querySelectorAll("[data-bans-page]").forEach((button) => {
        button.addEventListener("click", async () => {
          const nextPage = Number.parseInt(String(button.dataset.bansPage || "1"), 10);
          if (!Number.isFinite(nextPage) || nextPage < 1) {
            return;
          }
          pagingState.page = nextPage;
          await renderRows();
        });
      });

      listNode.querySelectorAll(".waf-ban-btn").forEach((button) => {
        button.addEventListener("click", async () => {
          const action = String(button.dataset.action || "");
          const rowIndex = Number.parseInt(String(button.dataset.row || "-1"), 10);
          const row = rows[rowIndex];
          if (!row || !row.siteID || !row.ip) {
            return;
          }
          const ip = normalizeIP(row.ip);
          const isAllowlisted = (allowlistBySite.get(row.siteID) || new Set()).has(ip);
          if (isAllowlisted) {
            ctx.notify("IP is in allowlist for this site; no ban actions applied.");
            return;
          }

          if (action === "unban") {
            if (!confirmAction(ctx.t("bans.confirm.unban", { ip: row.ip, site: row.siteID }))) {
              return;
            }
            try {
              button.disabled = true;
              await postBanAction(ctx, row.siteID, row.origin, "unban", ip);
              clearRateLimitCookies(row.siteID);
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
            await postBanAction(ctx, row.siteID, row.origin, "ban", ip);
            ctx.notify(ctx.t("toast.ipBanned"));
            await renderRows();
          } catch (error) {
            setError(statusNode, error?.message || ctx.t("bans.error.action"));
          } finally {
            button.disabled = false;
          }
        });
      });
    } catch (_error) {
      setError(statusNode, ctx.t("bans.error.load"));
      listNode.innerHTML = "";
    }
  };

  container.querySelector("#bans-refresh")?.addEventListener("click", () => {
    renderRows();
  });

  await renderRows();
}
