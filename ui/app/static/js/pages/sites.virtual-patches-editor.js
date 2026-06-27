import { escapeHtml } from "../ui.js";

/**
 * Renders the virtual patches editor for a site.
 * state.virtualPatches is loaded from the API on tab activation.
 * New patches are created via POST /api/virtual-patches/{siteID}.
 * Existing patches are deleted via DELETE /api/virtual-patches/{siteID}/{patchID}.
 */
export function renderVirtualPatchesEditor(state, ctx) {
  const patches = Array.isArray(state.virtualPatches) ? state.virtualPatches : [];
  const targets = ["uri", "body", "header"];
  const actions = ["block", "monitor"];

  return `
    <div class="waf-field full" id="virtual-patches-editor">
      <label>${escapeHtml(ctx.t("sites.easy.virtualpatches.list"))}</label>
      <div class="waf-stack" id="virtual-patches-list">
        ${patches.length === 0
    ? `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`
    : patches.map((p) => `
          <div class="waf-inline waf-virtual-patch-row" data-patch-id="${escapeHtml(p.id)}">
            <span class="waf-note" style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="${escapeHtml(p.pattern)}">${escapeHtml(p.pattern)}</span>
            <span class="waf-badge">${escapeHtml(p.target)}</span>
            <span class="waf-badge${p.action === "block" ? " waf-badge-danger" : ""}">${escapeHtml(p.action)}</span>
            ${p.expires_at ? `<span class="waf-note" style="white-space:nowrap;">${escapeHtml(ctx.t("sites.easy.virtualpatches.expires"))}: ${escapeHtml(p.expires_at.slice(0, 10))}</span>` : ""}
            <button class="btn ghost btn-sm" type="button" data-vp-delete="${escapeHtml(p.id)}" aria-label="${escapeHtml(ctx.t("common.delete"))}">&times;</button>
          </div>`).join("")}
      </div>
      <div class="waf-inline waf-virtual-patch-add-row" style="margin-top:8px;flex-wrap:wrap;gap:6px;">
        <input id="vp-pattern" placeholder="/vuln-path" style="flex:2;min-width:140px;" value="">
        <select id="vp-target">
          ${targets.map((t) => `<option value="${t}">${t}</option>`).join("")}
        </select>
        <select id="vp-action">
          ${actions.map((a) => `<option value="${a}">${a}</option>`).join("")}
        </select>
        <input id="vp-expires" type="date" title="${escapeHtml(ctx.t("sites.easy.virtualpatches.expiresHint"))}" style="min-width:130px;">
        <button class="btn ghost btn-sm" type="button" id="vp-add-btn">${escapeHtml(ctx.t("sites.easy.virtualpatches.add"))}</button>
      </div>
    </div>`;
}
