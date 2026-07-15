import { escapeHtml } from "../ui.js";
import { normalizeList } from "./bans.helpers.base.js";
import { countryFlagEmoji, parsePositiveNumber, renderModules, renderReasonList } from "./bans.helpers.events.js";

export function createPaginationMetaGetter(pagingState) {
  return (total) => {
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
}

export function createModalControllers({
  detailModalNode,
  createModalNode,
  createStatusNode,
  createSiteNode,
  createDurationNode,
  createIPNode,
  extendModalNode,
  extendStatusNode,
  extendSiteNode,
  extendIPNode,
  extendDurationNode,
  unbanModalNode,
  unbanStatusNode,
  unbanSiteNode,
  unbanIPNode,
  unbanSubmitNode,
  latestSiteIDsRef,
  extendDurations
}) {
  let extendDraft = null;
  let unbanDraft = null;
  const closeDetail = () => {
    detailModalNode?.classList.add("waf-hidden");
  };
  const openCreateModal = () => {
    createStatusNode.innerHTML = "";
    if (createSiteNode && latestSiteIDsRef.current.length && !createSiteNode.value) {
      createSiteNode.value = latestSiteIDsRef.current[0];
    }
    if (createDurationNode && !createDurationNode.value) {
      createDurationNode.value = String(extendDurations[0]?.seconds || 0);
    }
    createModalNode?.classList.remove("waf-hidden");
    createIPNode?.focus();
  };
  const closeCreateModal = () => {
    createModalNode?.classList.add("waf-hidden");
  };
  const closeExtendModal = () => {
    extendDraft = null;
    if (extendStatusNode) {
      extendStatusNode.innerHTML = "";
    }
    extendModalNode?.classList.add("waf-hidden");
  };
  const openExtendModal = (row, siteLabel) => {
    if (!row || !extendModalNode) {
      return;
    }
    extendDraft = { row };
    if (extendSiteNode) {
      extendSiteNode.value = String(siteLabel || row.siteID || "").trim();
    }
    if (extendIPNode) {
      extendIPNode.value = String(row.ip || "").trim();
    }
    if (extendDurationNode) {
      const current = row.expiresAt ? Math.max(0, Math.round((row.expiresAt.getTime() - Date.now()) / 1000)) : 0;
      const closest = extendDurations.find((item) => item.seconds === 0 ? current <= 0 : current <= item.seconds) || extendDurations[0];
      extendDurationNode.value = String(closest.seconds);
    }
    extendStatusNode.innerHTML = "";
    extendModalNode.classList.remove("waf-hidden");
    extendDurationNode?.focus();
  };
  const closeUnbanModal = () => {
    unbanDraft = null;
    if (unbanStatusNode) {
      unbanStatusNode.innerHTML = "";
    }
    unbanModalNode?.classList.add("waf-hidden");
  };
  const openUnbanModal = (row, siteLabel) => {
    if (!row || !unbanModalNode) {
      return;
    }
    unbanDraft = { row };
    if (unbanSiteNode) {
      unbanSiteNode.value = String(siteLabel || row.siteID || "").trim();
    }
    if (unbanIPNode) {
      unbanIPNode.value = String(row.ip || "").trim();
    }
    if (unbanStatusNode) {
      unbanStatusNode.innerHTML = "";
    }
    unbanModalNode.classList.remove("waf-hidden");
    unbanSubmitNode?.focus();
  };

  detailModalNode?.querySelectorAll("[data-bans-detail-close]").forEach((node) => {
    node.addEventListener("click", closeDetail);
  });
  detailModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeDetail();
    }
  });
  createModalNode?.querySelectorAll("[data-bans-create-close]").forEach((node) => {
    node.addEventListener("click", closeCreateModal);
  });
  createModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeCreateModal();
    }
  });
  extendModalNode?.querySelectorAll("[data-bans-extend-close]").forEach((node) => {
    node.addEventListener("click", closeExtendModal);
  });
  extendModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeExtendModal();
    }
  });
  unbanModalNode?.querySelectorAll("[data-bans-unban-close]").forEach((node) => {
    node.addEventListener("click", closeUnbanModal);
  });
  unbanModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeUnbanModal();
    }
  });

  return {
    closeDetail,
    openCreateModal,
    closeCreateModal,
    closeExtendModal,
    openExtendModal,
    closeUnbanModal,
    openUnbanModal,
    getExtendDraft: () => extendDraft,
    getUnbanDraft: () => unbanDraft
  };
}

export function renderCreateSiteOptions({ createSiteNode, sites, isStartupSelfTestSite, latestSiteIDsRef, t, allServicesSiteID }) {
  if (!createSiteNode) {
    return;
  }
  const selected = String(createSiteNode.value || "").trim();
  const options = normalizeList(sites)
    .map((site) => {
      const id = String(site?.id || "").trim();
      if (!id) {
        return null;
      }
      if (isStartupSelfTestSite(id)) {
        return null;
      }
      const label = String(site?.primary_host || id).trim() || id;
      return { id, label };
    })
    .filter(Boolean)
    .sort((left, right) => left.label.localeCompare(right.label, undefined, { sensitivity: "base" }));
  latestSiteIDsRef.current = options.map((item) => item.id);
  createSiteNode.innerHTML = [
    `<option value="${allServicesSiteID}">${escapeHtml(t("bans.site.allServices"))}</option>`,
    ...options.map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.label)}</option>`)
  ].join("");
  if (selected && latestSiteIDsRef.current.includes(selected)) {
    createSiteNode.value = selected;
    return;
  }
  createSiteNode.value = allServicesSiteID;
}

export function renderDetail({ detailContentNode, detailModalNode, row, siteLabel, t, formatRemaining }) {
  if (!detailContentNode || !row) {
    return;
  }
  const detailRows = [
    ["events.detail.field.time", row.occurredAt ? row.occurredAt.toISOString() : "-"],
    ["events.detail.field.site", siteLabel || row.siteID || "-"],
    ["bans.col.ip", row.ip || "-"],
    ["bans.col.country", countryFlagEmoji(row.country), true],
    ["bans.col.module", renderModules(row.modules || new Set(), t) || "-"],
    ["events.detail.field.summary", renderReasonList(row.reasons || new Set())],
    ["bans.col.left", row.expiresAt ? formatRemaining(row.expiresAt, new Date(), t) : t("bans.time.permanent")]
  ];
  if (row.blockedCount > 0) {
    detailRows.push(["dashboard.detail.blocked", String(row.blockedCount)]);
  }
  if (row.statuses?.size) {
    detailRows.push(["events.col.status", renderReasonList(row.statuses)]);
  }
  const latestRaw = row.latestEvent?.details && typeof row.latestEvent.details === "object"
    ? row.latestEvent.details
    : {};
  const eventMeta = {
    event_ids: Array.from(row.eventIDs || []),
    paths: Array.from(row.paths || []),
    hosts: Array.from(row.hosts || []),
    referers: Array.from(row.referers || []),
    user_agents: Array.from(row.userAgents || []),
    latest_event: latestRaw
  };
  detailContentNode.innerHTML = `
      <div class="waf-table-wrap">
        <table class="waf-table waf-detail-table">
          <tbody>
            ${detailRows.map(([labelKey, value, trustedHTML]) => `
              <tr>
                <th>${escapeHtml(t(labelKey) === labelKey ? labelKey : t(labelKey))}</th>
                <td><pre class="waf-code">${trustedHTML ? String(value || "-") : escapeHtml(String(value || "-"))}</pre></td>
              </tr>
            `).join("")}
            <tr>
              <th>${escapeHtml(t("events.detail.field.details"))}</th>
              <td><pre class="waf-code">${escapeHtml(JSON.stringify(eventMeta, null, 2))}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    `;
  detailModalNode?.classList.remove("waf-hidden");
}

export async function loadSiteBanDurations({ sites, resolveSiteID, api }) {
  const entries = await Promise.all(
    normalizeList(sites).map(async (site) => {
      const candidate = String(site?.id || "").trim();
      if (!candidate) {
        return null;
      }
      try {
        const profile = await api.get(`/api/easy-site-profiles/${encodeURIComponent(candidate)}`);
        const whitelist = normalizeList(profile?.security_country_policy?.whitelist_country)
          .map((item) => String(item || "").trim())
          .filter(Boolean);
        if (whitelist.length > 0) {
          return [resolveSiteID(candidate), 0];
        }
        const duration = parsePositiveNumber(profile?.security_behavior_and_limits?.bad_behavior_ban_time_seconds, 300);
        return [resolveSiteID(candidate), duration];
      } catch (_error) {
        return [resolveSiteID(candidate), 300];
      }
    })
  );
  const out = new Map();
  for (const item of entries) {
    if (!item || !item[0]) {
      continue;
    }
    out.set(item[0], item[1]);
  }
  return out;
}
