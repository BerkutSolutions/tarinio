import { confirmAction, escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function normalizeToken(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function normalizeSites(value) {
  return normalizeList(value).map((item) => ({
    site_id: String(item?.site_id || ""),
    primary_host: String(item?.primary_host || ""),
    enabled: Boolean(item?.enabled),
  }));
}

function translateKey(ctx, key) {
  const value = ctx.t(key);
  return value === key ? "" : value;
}

function translateTimelineLabel(entry, ctx) {
  const summaryToken = normalizeToken(entry?.summary);
  const typeToken = normalizeToken(entry?.type);
  return (
    translateKey(ctx, `activity.summary.${summaryToken}`) ||
    translateKey(ctx, `events.summary.${summaryToken}`) ||
    translateKey(ctx, `activity.summary.${typeToken}`) ||
    translateKey(ctx, `events.type.${typeToken}`) ||
    String(entry?.summary || entry?.type || "-")
  );
}

function translateApplyResult(value, ctx) {
  const token = normalizeToken(value);
  if (token === "revision_applied") {
    return ctx.t("revisions.result.revisionApplied");
  }
  return String(value || "");
}

function formatServiceName(site, ctx) {
  const siteID = String(site?.site_id || "");
  if (siteID === "control-plane-access") {
    return ctx.t("revisions.site.controlPlaneAccess");
  }
  if (String(site?.primary_host || "").trim()) {
    return `${siteID} (${site.primary_host})`;
  }
  return siteID || "-";
}

function timelineBadge(entry) {
  const severity = String(entry?.severity || "").toLowerCase();
  if (severity === "error") {
    return statusBadge("failed");
  }
  if (severity === "warning") {
    return statusBadge("pending");
  }
  return statusBadge("info");
}

function serviceTileStatus(service) {
  if (Number(service?.active_revision_count || 0) > 0) {
    return "active";
  }
  if (Number(service?.failed_revision_count || 0) > 0) {
    return "failed";
  }
  if (Number(service?.pending_revision_count || 0) > 0) {
    return "pending";
  }
  if (Number(service?.revision_count || 0) > 0) {
    return "inactive";
  }
  return "unknown";
}

function lastStatusBadge(service) {
  const lastStatus = serviceTileStatus(service);
  return statusBadge(lastStatus || "unknown");
}

function revisionVisualStatus(item) {
  if (item?.is_active) {
    return "active";
  }
  const lastApplyStatus = String(item?.last_apply_status || "").trim().toLowerCase();
  const status = String(item?.status || "").trim().toLowerCase();
  if (lastApplyStatus === "succeeded") {
    return "inactive";
  }
  if (lastApplyStatus === "failed" || status === "failed") {
    return "failed";
  }
  if (lastApplyStatus === "running") {
    return "running";
  }
  return status || "unknown";
}

function stringifyDetailValue(value) {
  if (value == null) {
    return "-";
  }
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export async function renderRevisions(container, ctx) {
  container.innerHTML = `
    <section class="waf-card revisions-page" id="revisions-page">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.revisions"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("revisions.page.subtitle"))}</div>
        </div>
        <button class="btn ghost btn-sm" type="button" id="revisions-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
      </div>
      <div class="waf-card-body revisions-shell">
        <div class="revisions-layout">
          <section class="revisions-frame revisions-main-frame">
            <div class="revisions-frame-head">
              <div class="revisions-frame-title">${escapeHtml(ctx.t("revisions.frame.services"))}</div>
            </div>
            <div id="revisions-services" class="revisions-services"></div>
          </section>
          <aside class="revisions-frame revisions-side-frame">
            <div class="revisions-frame-head">
              <div class="revisions-frame-title">${escapeHtml(ctx.t("revisions.frame.statuses"))}</div>
            </div>
            <div id="revisions-sidebar" class="revisions-sidebar"></div>
          </aside>
        </div>
      </div>
    </section>
    <div class="waf-modal waf-hidden" id="revisions-detail-modal" role="dialog" aria-modal="true" aria-labelledby="revisions-detail-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-revisions-modal-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card revisions-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="revisions-detail-title">${escapeHtml(ctx.t("revisions.modal.title"))}</h3>
            <div class="muted" id="revisions-detail-subtitle">${escapeHtml(ctx.t("revisions.modal.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-revisions-modal-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body revisions-modal-body" id="revisions-detail-content"></div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="revisions-status-modal" role="dialog" aria-modal="true" aria-labelledby="revisions-status-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-revisions-status-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card revisions-status-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="revisions-status-title">${escapeHtml(ctx.t("revisions.timeline.detailTitle"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("revisions.timeline.detailSubtitle"))}</div>
          </div>
          <div class="revisions-status-modal-actions">
            <button class="btn ghost btn-sm" type="button" id="revisions-clear-statuses">${escapeHtml(ctx.t("revisions.timeline.clear"))}</button>
            <button class="btn ghost btn-sm" type="button" data-revisions-status-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
          </div>
        </div>
        <div class="waf-card-body revisions-status-detail" id="revisions-status-content"></div>
      </div>
    </div>
  `;

  const servicesNode = container.querySelector("#revisions-services");
  const sidebarNode = container.querySelector("#revisions-sidebar");
  const modalNode = container.querySelector("#revisions-detail-modal");
  const modalSubtitleNode = container.querySelector("#revisions-detail-subtitle");
  const modalContentNode = container.querySelector("#revisions-detail-content");
  const statusModalNode = container.querySelector("#revisions-status-modal");
  const statusContentNode = container.querySelector("#revisions-status-content");
  const state = {
    payload: null,
    selectedSiteID: "",
    pendingRevisionID: "",
    expandedRevisionIDs: new Set(),
    selectedTimelineIndex: -1,
  };

  const closeModal = () => {
    state.selectedSiteID = "";
    state.expandedRevisionIDs = new Set();
    modalNode?.classList.add("waf-hidden");
  };

  const closeStatusModal = () => {
    state.selectedTimelineIndex = -1;
    statusModalNode?.classList.add("waf-hidden");
  };

  const renderSidebar = () => {
    const summary = state.payload?.summary || {};
    const timeline = normalizeList(state.payload?.timeline);
    sidebarNode.innerHTML = `
      <div class="revisions-status-stack">
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.sidebar.totalServices"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(normalizeList(state.payload?.services).length))}</strong>
        </div>
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.sidebar.totalRevisions"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(summary?.total_revisions ?? 0))}</strong>
        </div>
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.sidebar.pendingRevisions"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(summary?.pending_revisions ?? 0))}</strong>
        </div>
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.sidebar.failedRevisions"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(summary?.failed_revisions ?? 0))}</strong>
        </div>
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.summary.succeededJobs"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(summary?.succeeded_apply_jobs ?? 0))}</strong>
        </div>
        <div class="revisions-status-card">
          <div class="revisions-status-label">${escapeHtml(ctx.t("revisions.summary.failedJobs"))}</div>
          <strong class="revisions-status-value">${escapeHtml(String(summary?.failed_apply_jobs ?? 0))}</strong>
        </div>
      </div>
      <div class="revisions-timeline-frame">
        <div class="revisions-frame-title">${escapeHtml(ctx.t("revisions.timeline.title"))}</div>
        ${timeline.length ? `
          <div class="revisions-timeline-list">
            ${timeline.map((item, index) => `
              <button type="button" class="revisions-timeline-item" data-revision-status-index="${index}">
                <div class="revisions-timeline-head">
                  <strong>${escapeHtml(String(item?.revision_id || "-"))}</strong>
                  ${timelineBadge(item)}
                </div>
                <div class="revisions-timeline-meta">${escapeHtml(formatDate(String(item?.occurred_at || "")))}</div>
                <div class="revisions-timeline-text">${escapeHtml(translateTimelineLabel(item, ctx))}</div>
              </button>
            `).join("")}
          </div>
        ` : `<div class="waf-empty">${escapeHtml(ctx.t("revisions.timeline.empty"))}</div>`}
      </div>
    `;
  };

  const renderStatusModal = () => {
    const timeline = normalizeList(state.payload?.timeline);
    if (state.selectedTimelineIndex < 0 || state.selectedTimelineIndex >= timeline.length) {
      return;
    }
    const entry = timeline[state.selectedTimelineIndex];
    const details = entry?.details && typeof entry.details === "object" ? entry.details : {};
    const detailRows = [
      [ctx.t("revisions.timeline.field.revision"), String(entry?.revision_id || "-")],
      [ctx.t("revisions.timeline.field.job"), String(entry?.job_id || "-")],
      [ctx.t("revisions.timeline.field.time"), formatDate(String(entry?.occurred_at || ""))],
      [ctx.t("revisions.timeline.field.type"), translateTimelineLabel(entry, ctx)],
      [ctx.t("revisions.timeline.field.severity"), String(entry?.severity || "-")],
      [ctx.t("revisions.timeline.field.source"), String(entry?.source_component || "-")],
      [ctx.t("revisions.timeline.field.site"), String(entry?.site_id || "-")],
    ];
    statusContentNode.innerHTML = `
      <div class="revisions-status-summary">
        ${detailRows.map(([label, value]) => `
          <div class="revisions-status-line">
            <span>${escapeHtml(label)}</span>
            <strong>${escapeHtml(value)}</strong>
          </div>
        `).join("")}
      </div>
      <div class="revisions-status-details-frame">
        <div class="revisions-frame-title">${escapeHtml(ctx.t("revisions.timeline.field.details"))}</div>
        ${Object.keys(details).length ? `
          <div class="revisions-status-details-list">
            ${Object.entries(details).map(([key, value]) => `
              <div class="revisions-status-detail-tile">
                <span>${escapeHtml(key)}</span>
                <pre>${escapeHtml(stringifyDetailValue(value))}</pre>
              </div>
            `).join("")}
          </div>
        ` : `<div class="waf-empty">${escapeHtml(ctx.t("revisions.timeline.noDetails"))}</div>`}
      </div>
    `;
    statusModalNode.classList.remove("waf-hidden");
    statusModalNode.focus();
  };

  const renderServices = () => {
    const services = normalizeList(state.payload?.services);
    if (!services.length) {
      servicesNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("revisions.services.empty"))}</div>`;
      return;
    }
    servicesNode.innerHTML = `
      <div class="revisions-service-grid">
        ${services.map((service) => `
          <button type="button" class="revisions-service-tile" data-revision-site="${escapeHtml(String(service?.site_id || ""))}">
            <div class="revisions-service-head">
              <div>
                <div class="revisions-service-title">${escapeHtml(String(service?.site_id || "-"))}</div>
                <div class="revisions-service-host">${escapeHtml(String(service?.primary_host || "-"))}</div>
              </div>
              ${statusBadge(service?.enabled ? "active" : "inactive")}
            </div>
            <div class="revisions-service-stats">
              <div class="revisions-service-stat">
                <span>${escapeHtml(ctx.t("revisions.services.count"))}</span>
                <strong>${escapeHtml(String(service?.revision_count ?? 0))}</strong>
              </div>
              <div class="revisions-service-stat">
                <span>${escapeHtml(ctx.t("revisions.services.activeCount"))}</span>
                <strong>${escapeHtml(String(service?.active_revision_count ?? 0))}</strong>
              </div>
              <div class="revisions-service-stat">
                <span>${escapeHtml(ctx.t("revisions.services.pendingCount"))}</span>
                <strong>${escapeHtml(String(service?.pending_revision_count ?? 0))}</strong>
              </div>
              <div class="revisions-service-stat">
                <span>${escapeHtml(ctx.t("revisions.services.failedCount"))}</span>
                <strong>${escapeHtml(String(service?.failed_revision_count ?? 0))}</strong>
              </div>
            </div>
            <div class="revisions-service-footer">
              <div class="revisions-service-meta">
                <span>${escapeHtml(ctx.t("revisions.services.lastRevision"))}</span>
                <strong>${escapeHtml(String(service?.last_revision_id || "-"))}</strong>
              </div>
              <div class="revisions-service-badge">${lastStatusBadge(service)}</div>
            </div>
          </button>
        `).join("")}
      </div>
    `;
  };

  const renderModal = () => {
    if (!state.selectedSiteID) {
      return;
    }
    const services = normalizeList(state.payload?.services);
    const revisions = normalizeList(state.payload?.revisions);
    const service = services.find((item) => String(item?.site_id || "") === state.selectedSiteID);
    if (!service) {
      closeModal();
      return;
    }
    const filtered = revisions.filter((revision) => normalizeSites(revision?.sites).some((site) => site.site_id === state.selectedSiteID));
    if (!state.expandedRevisionIDs.size && filtered.length) {
      state.expandedRevisionIDs = new Set([String(filtered[0].id || "")]);
    }
    filtered.forEach((item) => {
      if (item?.is_active) {
        state.expandedRevisionIDs.add(String(item?.id || ""));
      }
    });
    modalSubtitleNode.textContent = ctx.t("revisions.modal.subtitle", { service: service.site_id, host: service.primary_host || "-" });
    modalContentNode.innerHTML = `
      <div class="revisions-modal-summary">
        <div class="revisions-modal-chip"><span>${escapeHtml(ctx.t("revisions.modal.service"))}</span><strong>${escapeHtml(service.site_id)}</strong></div>
        <div class="revisions-modal-chip"><span>${escapeHtml(ctx.t("revisions.modal.host"))}</span><strong>${escapeHtml(service.primary_host || "-")}</strong></div>
        <div class="revisions-modal-chip"><span>${escapeHtml(ctx.t("revisions.modal.revisions"))}</span><strong>${escapeHtml(String(filtered.length))}</strong></div>
      </div>
      ${filtered.length ? `
        <div class="revisions-modal-list">
          ${filtered.map((item) => {
            const sites = normalizeSites(item?.sites);
            const isActive = Boolean(item?.is_active);
            const isExpanded = isActive || state.expandedRevisionIDs.has(String(item?.id || ""));
            const applyDisabled = state.pendingRevisionID === item.id || Boolean(item?.is_active);
            const deleteDisabled = state.pendingRevisionID === item.id || Boolean(item?.is_active);
            const visualStatus = revisionVisualStatus(item);
            return `
              <article class="revisions-modal-item ${isExpanded ? "is-expanded" : ""}">
                <div class="revisions-modal-item-head">
                  <div>
                    <div class="revisions-modal-item-title">${escapeHtml(String(item?.id || "-"))}</div>
                    <div class="revisions-modal-item-meta">${escapeHtml(ctx.t("revisions.field.version"))}: ${escapeHtml(String(item?.version ?? "-"))} - ${escapeHtml(formatDate(String(item?.created_at || "")))}</div>
                  </div>
                  <div class="revisions-modal-badges">
                    ${statusBadge(visualStatus)}
                    <button type="button" class="btn btn-sm" data-revision-apply="${escapeHtml(String(item?.id || ""))}" ${applyDisabled ? "disabled" : ""}>
                      ${escapeHtml(state.pendingRevisionID === item.id ? ctx.t("revisions.action.applying") : item?.is_active ? ctx.t("revisions.revision.applyBlocked") : ctx.t("revisions.action.apply"))}
                    </button>
                    <button type="button" class="btn ghost btn-sm" data-revision-delete="${escapeHtml(String(item?.id || ""))}" ${deleteDisabled ? "disabled" : ""}>
                      ${escapeHtml(state.pendingRevisionID === item.id ? ctx.t("revisions.action.deleting") : ctx.t("revisions.action.delete"))}
                    </button>
                    ${isActive ? "" : `
                      <button type="button" class="revisions-accordion-indicator" data-revision-toggle="${escapeHtml(String(item?.id || ""))}" aria-label="${escapeHtml(ctx.t("revisions.action.toggleDetails"))}">
                        ${isExpanded ? "-" : "+"}
                      </button>
                    `}
                  </div>
                </div>
                <div class="revisions-modal-item-preview">
                  <div class="revisions-modal-preview-tile">
                    <span>${escapeHtml(ctx.t("revisions.revision.lastApply"))}</span>
                    <strong>${escapeHtml(translateApplyResult(item?.last_apply_result || item?.last_apply_status || ctx.t("revisions.revision.noApply"), ctx))}</strong>
                  </div>
                  <div class="revisions-modal-preview-tile">
                    <span>${escapeHtml(ctx.t("revisions.revision.lastEvent"))}</span>
                    <strong>${escapeHtml(translateTimelineLabel({ summary: item?.last_event_summary, type: item?.last_event_type }, ctx))}</strong>
                  </div>
                </div>
                <div class="revisions-modal-accordion ${isExpanded ? "" : "waf-hidden"}">
                  <div class="revisions-modal-item-grid">
                    <div class="revisions-modal-line revisions-modal-line-wide">
                      <span>${escapeHtml(ctx.t("revisions.revision.sites"))}</span>
                      <strong>${escapeHtml(sites.map((site) => formatServiceName(site, ctx)).join(", ") || "-")}</strong>
                    </div>
                    <div class="revisions-modal-line">
                      <span>${escapeHtml(ctx.t("revisions.revision.checksum"))}</span>
                      <strong>${escapeHtml(String(item?.checksum || "-"))}</strong>
                    </div>
                    ${item?.last_apply_at ? `
                      <div class="revisions-modal-line">
                        <span>${escapeHtml(ctx.t("revisions.revision.lastApplyAt"))}</span>
                        <strong>${escapeHtml(formatDate(String(item?.last_apply_at || "")))}</strong>
                      </div>
                    ` : ""}
                    ${item?.snapshot_error ? `
                      <div class="revisions-modal-line revisions-modal-line-wide">
                        <span>${escapeHtml(ctx.t("revisions.revision.snapshotError"))}</span>
                        <strong>${escapeHtml(String(item.snapshot_error))}</strong>
                      </div>
                    ` : ""}
                  </div>
                </div>
              </article>
            `;
          }).join("")}
        </div>
      ` : `<div class="waf-empty">${escapeHtml(ctx.t("revisions.modal.empty"))}</div>`}
    `;
    modalNode.classList.remove("waf-hidden");
    modalNode.focus();
  };

  const renderAll = () => {
    renderServices();
    renderSidebar();
    if (state.selectedSiteID) {
      renderModal();
    }
  };

  const loadCatalog = async () => {
    setLoading(servicesNode, ctx.t("revisions.summary.loading"));
    setLoading(sidebarNode, ctx.t("revisions.summary.loading"));
    try {
      const payload = await ctx.api.get("/api/revisions");
      if (ctx.signal?.aborted) {
        return;
      }
      state.payload = payload || {};
      renderAll();
    } catch (error) {
      setError(servicesNode, error?.message || ctx.t("revisions.error.loadCatalog"));
      setError(sidebarNode, error?.message || ctx.t("revisions.error.loadCatalog"));
    }
  };

  const runApply = async (revisionID) => {
    if (!revisionID || !confirmAction(ctx.t("revisions.confirm.apply", { revisionId: revisionID }))) {
      return;
    }
    state.pendingRevisionID = revisionID;
    renderModal();
    try {
      const job = await ctx.api.post(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, {});
      if (String(job?.status || "").toLowerCase() === "failed") {
        ctx.notify(job?.result || ctx.t("revisions.error.apply"), "error");
      } else {
        ctx.notify(ctx.t("revisions.toast.applyStarted", { revisionId: revisionID }), "success");
      }
      await loadCatalog();
    } catch (error) {
      ctx.notify(error?.message || ctx.t("revisions.error.apply"), "error");
    } finally {
      state.pendingRevisionID = "";
      renderModal();
    }
  };

  const runDelete = async (revisionID) => {
    if (!revisionID || !confirmAction(ctx.t("revisions.confirm.delete", { revisionId: revisionID }))) {
      return;
    }
    state.pendingRevisionID = revisionID;
    renderModal();
    try {
      await ctx.api.delete(`/api/revisions/${encodeURIComponent(revisionID)}`);
      ctx.notify(ctx.t("revisions.toast.deleteSucceeded", { revisionId: revisionID }), "success");
      await loadCatalog();
    } catch (error) {
      ctx.notify(error?.message || ctx.t("revisions.error.delete"), "error");
    } finally {
      state.pendingRevisionID = "";
      renderModal();
    }
  };

  const clearStatuses = async () => {
    if (!confirmAction(ctx.t("revisions.timeline.confirmClear"))) {
      return;
    }
    try {
      await ctx.api.delete("/api/revisions/statuses");
      ctx.notify(ctx.t("revisions.timeline.cleared"), "success");
      closeStatusModal();
      await loadCatalog();
    } catch (error) {
      ctx.notify(error?.message || ctx.t("revisions.timeline.clearFailed"), "error");
    }
  };

  container.querySelector("#revisions-refresh")?.addEventListener("click", loadCatalog);
  container.querySelector("#revisions-clear-statuses")?.addEventListener("click", clearStatuses);

  servicesNode.addEventListener("click", (event) => {
    const tile = event.target.closest("[data-revision-site]");
    if (!tile) {
      return;
    }
    state.selectedSiteID = String(tile.getAttribute("data-revision-site") || "");
    state.expandedRevisionIDs = new Set();
    renderModal();
  });

  sidebarNode.addEventListener("click", (event) => {
    const item = event.target.closest("[data-revision-status-index]");
    if (!item) {
      return;
    }
    state.selectedTimelineIndex = Number.parseInt(String(item.getAttribute("data-revision-status-index") || "-1"), 10);
    renderStatusModal();
  });

  modalNode?.querySelectorAll("[data-revisions-modal-close='true']").forEach((node) => {
    node.addEventListener("click", closeModal);
  });
  modalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeModal();
    }
  });
  modalContentNode?.addEventListener("click", async (event) => {
    const toggle = event.target.closest("[data-revision-toggle]");
    if (toggle) {
      const revisionID = String(toggle.getAttribute("data-revision-toggle") || "");
      const current = normalizeList(state.payload?.revisions).find((item) => String(item?.id || "") === revisionID);
      if (current?.is_active) {
        return;
      }
      if (state.expandedRevisionIDs.has(revisionID)) {
        state.expandedRevisionIDs.delete(revisionID);
      } else {
        state.expandedRevisionIDs.add(revisionID);
      }
      renderModal();
      return;
    }
    const applyButton = event.target.closest("[data-revision-apply]");
    if (applyButton) {
      await runApply(String(applyButton.getAttribute("data-revision-apply") || ""));
      return;
    }
    const deleteButton = event.target.closest("[data-revision-delete]");
    if (deleteButton) {
      await runDelete(String(deleteButton.getAttribute("data-revision-delete") || ""));
    }
  });
  statusModalNode?.querySelectorAll("[data-revisions-status-close='true']").forEach((node) => {
    node.addEventListener("click", closeStatusModal);
  });
  statusModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeStatusModal();
    }
  });

  await loadCatalog();
}
