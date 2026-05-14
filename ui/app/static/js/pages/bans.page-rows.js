import { escapeHtml, formatDate, setError, setLoading } from "../ui.js";
import {
  buildHostSiteMap,
  buildIPSetBySite,
  buildPageButtons,
  buildSiteAliasMap,
  clearRateLimitCookies,
  formatRemaining,
  mergeByID,
  normalizeIP,
  unwrapList
} from "./bans.helpers.base.js";
import {
  buildManualBanRows,
  dismissBanRow,
  isDismissedBanRow,
  loadManualBanTimers,
  mergeBanRows,
  processExpiredManualBanTimers,
  removeManualBanTimer,
  saveManualBanTimers
} from "./bans.helpers.timers.js";
import { buildAutoBanRows, buildLatestUnbanMarkers, countryFlagEmoji, postBanAction, renderModules } from "./bans.helpers.events.js";

export function createRenderRows({
  ctx,
  statusNode,
  listNode,
  pagingState,
  getPaginationMeta,
  renderCreateSiteOptions,
  loadSiteBanDurations,
  renderDetail,
  openExtendModal,
  resolveCanonicalSiteID,
  tryGetJSON,
  confirmAction,
  iconPlus,
  iconUnlock
}) {
  const renderRows = async () => {
    setLoading(statusNode, ctx.t("bans.loading"));
    try {
      const [sitesResponse, accessResponse, eventsResponse, auditResponse, sitesSecondary, accessSecondary, eventsSecondary] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/access-policies"),
        ctx.api.get("/api/events"),
        tryGetJSON("/api/audit?action=accesspolicy.unban&status=succeeded&limit=500"),
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/access-policies"),
        tryGetJSON("/api-app/events")
      ]);

      const sites = mergeByID(sitesResponse, unwrapList(sitesSecondary, ["sites"]), "id");
      const accessPolicies = mergeByID(
        unwrapList(accessResponse, ["access_policies"]),
        unwrapList(accessSecondary, ["access_policies"]),
        "id"
      );
      const events = mergeByID(unwrapList(eventsResponse, ["events", "items"]), unwrapList(eventsSecondary, ["events", "items"]), "id");

      const siteByID = new Map(sites.map((site) => [String(site?.id || ""), site]));
      renderCreateSiteOptions(sites);
      const siteAliasMap = buildSiteAliasMap(sites);
      const hostSiteMap = buildHostSiteMap(sites);
      const canonicalSiteID = (value, hostHint = "") => resolveCanonicalSiteID(value, siteAliasMap, hostSiteMap, hostHint);
      const allowlistBySite = buildIPSetBySite(accessPolicies, "allowlist", canonicalSiteID);
      const denylistBySite = buildIPSetBySite(accessPolicies, "denylist", canonicalSiteID);
      const siteBanDurationByID = await loadSiteBanDurations(sites, canonicalSiteID);
      const unbanMarkersBySite = buildLatestUnbanMarkers(unwrapList(auditResponse, ["items"]), canonicalSiteID);
      const manualBanTimers = loadManualBanTimers();
      await processExpiredManualBanTimers(ctx, manualBanTimers);
      const manualRows = buildManualBanRows(accessPolicies, canonicalSiteID, manualBanTimers);
      const autoRows = buildAutoBanRows(events, canonicalSiteID, siteBanDurationByID, unbanMarkersBySite);
      const rows = mergeBanRows(manualRows, autoRows)
        .filter((row) => !isDismissedBanRow(row.siteID, row.ip))
        .sort((a, b) => {
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
                const allowlisted = (allowlistBySite.get(row.siteID) || new Set()).has(row.ip);
                const currentlyDenied = (denylistBySite.get(row.siteID) || new Set()).has(row.ip);
                return `
                  <tr class="waf-table-row-clickable" data-ban-row="${meta.start + index}" tabindex="0" role="button">
                    <td>${escapeHtml(row.ip)}</td>
                    <td>${escapeHtml(countryFlagEmoji(row.country))}</td>
                    <td>${escapeHtml(row.occurredAt ? formatDate(row.occurredAt.toISOString()) : "-")}</td>
                    <td>${escapeHtml(formatRemaining(row.expiresAt, now, ctx.t))}</td>
                    <td>${escapeHtml(siteLabel)}</td>
                    <td>${escapeHtml(renderModules(row.modules, ctx.t))}${allowlisted ? " (allowlist)" : ""}</td>
                    <td>
                      <div class="waf-ban-actions">
                        <button type="button" class="btn success btn-sm waf-ban-btn" data-action="extend" data-row="${meta.start + index}" title="${escapeHtml(ctx.t("bans.action.extend"))}">
                          <span class="waf-ban-btn-icon">${iconPlus}</span>
                          <span>${escapeHtml(ctx.t("bans.action.extend"))}</span>
                        </button>
                        <button type="button" class="btn danger btn-sm waf-ban-btn" data-action="unban" data-row="${meta.start + index}" data-currently-denied="${currentlyDenied ? "1" : "0"}" title="${escapeHtml(ctx.t("bans.action.unban"))}">
                          <span class="waf-ban-btn-icon">${iconUnlock}</span>
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
          button.blur();
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
            if (!confirmAction(ctx.t("bans.confirm.unban", { ip, site: row.siteID }))) {
              return;
            }
            try {
              await postBanAction(ctx, row.siteID, "primary", "unban", ip);
              const timers = loadManualBanTimers();
              removeManualBanTimer(timers, row.siteID, ip);
              saveManualBanTimers(timers);
              dismissBanRow(row.siteID, ip);
              clearRateLimitCookies(row.siteID);
              ctx.notify(ctx.t("toast.ipUnbanned"));
              await renderRows();
            } catch (error) {
              setError(statusNode, error?.message || ctx.t("bans.error.action"));
            }
            return;
          }
          const site = siteByID.get(row.siteID);
          openExtendModal(row, site?.primary_host || row.siteID);
        });
      });

      listNode.querySelectorAll("[data-ban-row]").forEach((rowNode) => {
        const open = () => {
          const rowIndex = Number.parseInt(String(rowNode.getAttribute("data-ban-row") || "-1"), 10);
          const row = rows[rowIndex];
          if (!row) {
            return;
          }
          const site = siteByID.get(row.siteID);
          const siteLabel = site?.primary_host || row.siteID;
          renderDetail(row, siteLabel);
        };
        rowNode.addEventListener("click", (event) => {
          if (event.target instanceof HTMLElement && event.target.closest(".waf-ban-btn")) {
            return;
          }
          open();
        });
        rowNode.addEventListener("keydown", (event) => {
          if (event.key !== "Enter" && event.key !== " ") {
            return;
          }
          event.preventDefault();
          open();
        });
      });
    } catch (_error) {
      setError(statusNode, ctx.t("bans.error.load"));
      listNode.innerHTML = "";
    }
  };
  return renderRows;
}
