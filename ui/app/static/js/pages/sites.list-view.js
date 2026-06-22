import { escapeHtml, setError, setLoading } from "../ui.js";
import { certificateDaysLeft, normalizeSiteID, routeBase } from "./sites.routing-merge.js";
import { normalizeHost, normalizeServiceProfile, formatServiceProfile } from "./sites.normalize.js";
import { resolvePublicServiceURL } from "./sites.traffic-helpers.js";

export function renderListView(state, ctx, formatCertificateExpiryByLanguage, statusBadge, formatDate) {
  return `
    <div class="waf-page-stack">
      <section class="waf-card waf-services-card">
        <div class="waf-card-head waf-services-toolbar">
          <div>
            <h3>${escapeHtml(ctx.t("sites.list.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("sites.list.subtitle"))}</div>
          </div>
          <div class="waf-actions">
            <button class="btn primary btn-sm" type="button" id="services-create">${escapeHtml(ctx.t("sites.action.createSite"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-import">${escapeHtml(ctx.t("sites.action.import"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-export">${escapeHtml(ctx.t("sites.action.export"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-delete-selected">${escapeHtml(ctx.t("sites.action.deleteSelected"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="sites-feedback"></div>
          <div class="waf-services-filters">
            <div class="waf-field">
              <label for="services-search">${escapeHtml(ctx.t("sites.filters.search"))}</label>
              <input id="services-search" value="${escapeHtml(state.search)}" placeholder="${escapeHtml(ctx.t("sites.filters.searchPlaceholder"))}">
            </div>
            <div class="waf-field">
              <label for="services-sort">${escapeHtml(ctx.t("sites.filters.sort"))}</label>
              <select id="services-sort">
                <option value="updated-desc"${state.sort === "updated-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.updatedDesc"))}</option>
                <option value="name-asc"${state.sort === "name-asc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.nameAsc"))}</option>
                <option value="name-desc"${state.sort === "name-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.nameDesc"))}</option>
                <option value="created-desc"${state.sort === "created-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.createdDesc"))}</option>
              </select>
            </div>
          </div>
          <div class="waf-table-wrap">
            <table class="waf-table waf-services-table">
              <thead><tr><th class="waf-check-col"><input type="checkbox" id="services-select-all"${state.filteredSites.length && state.filteredSites.every((site) => state.selectedSiteIDs.has(site.id)) ? " checked" : ""}></th><th>${escapeHtml(ctx.t("sites.table.name"))}</th><th>${escapeHtml(ctx.t("sites.table.profile"))}</th><th>${escapeHtml(ctx.t("sites.table.upstream"))}</th><th>${escapeHtml(ctx.t("sites.table.tls"))}</th><th>${escapeHtml(ctx.t("sites.table.updated"))}</th><th>${escapeHtml(ctx.t("sites.table.status"))}</th><th>${escapeHtml(ctx.t("sites.table.actions"))}</th></tr></thead>
              <tbody>
                ${state.filteredSites.length ? state.filteredSites.map((site) => {
                  const easyProfile = state.easyProfilesBySite.get(normalizeSiteID(site.id)) || null;
                  const serviceProfile = normalizeServiceProfile(easyProfile?.front_service?.profile);
                  const upstream = state.upstreamsBySite.get(site.id)?.[0] || null;
                  const tls = state.tlsBySite.get(site.id);
                  const language = String(ctx.getLanguage?.() || "en").trim().toLowerCase();
                  const certificateFromID = state.certificates.find((item) => String(item?.id || "").trim().toLowerCase() === String(tls?.certificate_id || "").trim().toLowerCase());
                  const certificateFallback = state.certificateBySiteID.get(site.id) || state.certificateByHost.get(normalizeHost(site.primary_host));
                  const certificate = certificateFromID || certificateFallback || null;
                  const tlsState = tls ? "managed" : (certificate ? "detected" : "missing");
                  const certificateTitle = String(certificate?.common_name || certificate?.id || "").trim();
                  const certificateExpiry = formatCertificateExpiryByLanguage(certificate?.not_after, language);
                  const certificateExpiryDays = certificateDaysLeft(certificate?.not_after);
                  const certificateIsExpiring = typeof certificateExpiryDays === "number" && certificateExpiryDays < 30;
                  const serviceURL = resolvePublicServiceURL(site, tlsState);
                  const tlsBadgeClass = tlsState === "missing" ? "badge-neutral" : (certificateIsExpiring ? "badge-danger" : "badge-success");
                  const tlsBadgeTitle = tlsState === "missing" ? ctx.t("sites.state.tlsMissing") : (certificateTitle || ctx.t("sites.state.tlsManaged"));
                  return `<tr class="waf-table-row-clickable" data-open-site-edit="${escapeHtml(site.id)}"><td class="waf-check-col"><input type="checkbox" data-select-site="${escapeHtml(site.id)}"${state.selectedSiteIDs.has(site.id) ? " checked" : ""}></td><td><button class="waf-link-button" type="button" data-open-service="${escapeHtml(serviceURL)}" title="${escapeHtml(ctx.t("sites.action.openService"))}">${escapeHtml(site.primary_host || site.id)}</button></td><td>${escapeHtml(formatServiceProfile(serviceProfile, ctx))}</td><td>${upstream ? `${escapeHtml(upstream.host)}:${escapeHtml(String(upstream.port))}` : escapeHtml(ctx.t("common.notSet"))}</td><td><div class="waf-services-tls-cell"><div class="badge ${tlsBadgeClass} waf-services-tls-badge"><div class="waf-services-tls-badge-title">${escapeHtml(tlsBadgeTitle)}</div>${tlsState !== "missing" && certificateTitle ? `<div class="waf-services-tls-badge-expire">${escapeHtml(ctx.t("sites.table.tlsValidTill"))}: ${escapeHtml(certificateExpiry)}</div>` : ""}</div></div></td><td>${escapeHtml(formatDate(site.updated_at || site.created_at))}</td><td>${statusBadge(site.enabled ? "active" : "failed")}</td><td><div class="waf-actions"><button class="btn ghost btn-sm" type="button" data-open-site="${escapeHtml(site.id)}">${escapeHtml(ctx.t("common.edit"))}</button><button class="btn ghost btn-sm" type="button" data-toggle-site="${escapeHtml(site.id)}" data-toggle-enabled="${site.enabled ? "1" : "0"}">${escapeHtml(ctx.t(site.enabled ? "common.disable" : "common.enable"))}</button></div></td></tr>`;
                }).join("") : `<tr><td colspan="8"><div class="waf-empty">${escapeHtml(ctx.t("sites.empty.sites"))}</div></td></tr>`}
              </tbody>
            </table>
          </div>
          <input id="services-import-file" type="file" accept=".env,text/plain" class="waf-hidden">
        </div>
      </section>
    </div>
  `;
}

export function bindList(container, state, ctx, deps) {
  const { go, load, downloadJSON, exportSelectedServicesEnv, importServicesFiles, toEnvKey, putWithPostFallback, normalizeSiteID } = deps;
  const feedback = container.querySelector("#sites-feedback");
  container.querySelector("#services-create")?.addEventListener("click", () => go(`${routeBase()}/new`));
  container.querySelector("#services-refresh")?.addEventListener("click", load);
  container.querySelector("#services-export")?.addEventListener("click", async () => {
    feedback.innerHTML = "";
    if (!state.selectedSiteIDs.size) {
      downloadJSON("waf-services-export.json", { sites: state.sites, upstreams: state.upstreams, tls_configs: state.tlsConfigs });
      ctx.notify(ctx.t("sites.toast.exported"));
      return;
    }
    try {
      const exportedCount = await exportSelectedServicesEnv(ctx, state.sites, state.upstreamsBySite, state.tlsBySite, state.accessBySite, state.selectedSiteIDs);
      ctx.notify(ctx.t("sites.toast.exportedEnv", { count: exportedCount }));
    } catch (error) {
      setError(feedback, `${ctx.t("sites.error.exportEnv")}: ${String(error?.message || error)}`);
    }
  });
  container.querySelector("#services-import")?.addEventListener("click", () => { container.querySelector("#services-import-file")?.click(); });
  container.querySelector("#services-import-file")?.addEventListener("change", async (event) => {
    const files = Array.from(event.target.files || []);
    if (!files.length) return;
    try {
      setLoading(feedback, ctx.t("sites.import.loading"));
      const imported = await importServicesFiles(files, ctx);
      state.pendingImportedDraftRef.set(imported);
      if (Array.isArray(imported.missingFields) && imported.missingFields.length) {
        feedback.innerHTML = `<div class="waf-empty"><div><strong>${escapeHtml(ctx.t("sites.import.warnings"))}</strong><pre class="waf-code">${escapeHtml(`${imported.file}: ${ctx.t("sites.import.missingFields")}: ${imported.missingFields.map((field) => toEnvKey(field)).join(", ")}`)}</pre></div></div>`;
      } else feedback.innerHTML = "";
      ctx.notify(ctx.t("sites.toast.imported"));
      go(`${routeBase()}/new`);
    } catch (error) {
      setError(feedback, `${ctx.t("sites.error.import")}: ${String(error?.message || error)}`);
    } finally { event.target.value = ""; }
  });
  container.querySelector("#services-search")?.addEventListener("input", (event) => { state.search = event.target.value; const s = Number(event.target.selectionStart || state.search.length); const e = Number(event.target.selectionEnd || s); deps.render(); const nextInput = container.querySelector("#services-search"); if (nextInput) { nextInput.focus(); nextInput.setSelectionRange(s, e); } });
  container.querySelector("#services-sort")?.addEventListener("change", (event) => { state.sort = event.target.value; deps.render(); });
  container.querySelector("#services-select-all")?.addEventListener("change", (event) => { const checked = Boolean(event.target.checked); for (const site of state.filteredSites) { if (checked) state.selectedSiteIDs.add(site.id); else state.selectedSiteIDs.delete(site.id); } deps.render(); });
  container.querySelectorAll("[data-select-site]").forEach((checkbox) => {
    checkbox.addEventListener("change", (event) => { event.stopPropagation(); const siteID = String(event.target.dataset.selectSite || ""); if (!siteID) return; if (event.target.checked) state.selectedSiteIDs.add(siteID); else state.selectedSiteIDs.delete(siteID); });
  });
  container.querySelectorAll("[data-open-site]").forEach((button) => { button.addEventListener("click", (event) => { event.stopPropagation(); go(`${routeBase()}/${encodeURIComponent(button.dataset.openSite)}`); }); });
  container.querySelectorAll("[data-toggle-site]").forEach((button) => {
    button.addEventListener("click", async (event) => {
      event.stopPropagation();
      const siteID = String(button.dataset.toggleSite || "").trim();
      if (!siteID) return;
      const site = state.sites.find((item) => normalizeSiteID(item?.id) === normalizeSiteID(siteID));
      if (!site) return;
      const nextEnabled = !(String(button.dataset.toggleEnabled || "") === "1");
      try {
        setLoading(feedback, ctx.t("sites.editor.saving"));
        await ctx.api.put(`/api/sites/${encodeURIComponent(siteID)}`, { ...site, enabled: nextEnabled });
        const easyProfilePath = `/api/easy-site-profiles/${encodeURIComponent(siteID)}`;
        const profile = await ctx.api.get(easyProfilePath).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
        if (profile && typeof profile === "object") {
          const nextProfile = { ...profile, front_service: { ...(profile.front_service || {}), enabled: nextEnabled } };
          await putWithPostFallback(ctx, easyProfilePath, nextProfile, { tolerateAutoApplyError: true });
        }
        feedback.innerHTML = "";
        ctx.notify(ctx.t(nextEnabled ? "toast.siteEnabled" : "toast.siteDisabled"));
        await load();
      } catch (error) { setError(feedback, `${ctx.t("sites.error.saveSite")}: ${String(error?.message || error)}`); }
    });
  });
  container.querySelectorAll("[data-open-service]").forEach((button) => { button.addEventListener("click", (event) => { event.stopPropagation(); const url = String(button.dataset.openService || "").trim(); if (!url) return; window.open(url, "_blank", "noopener,noreferrer"); }); });
  container.querySelectorAll("[data-open-site-edit]").forEach((row) => { row.addEventListener("click", (event) => { if (event.target.closest("button, input, select, textarea, a, label")) return; const siteID = String(row.dataset.openSiteEdit || "").trim(); if (!siteID) return; go(`${routeBase()}/${encodeURIComponent(siteID)}`); }); });
}
