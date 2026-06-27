import { escapeHtml, formatDate, statusBadge } from "../ui.js";

export function renderListView(state, ctx, deps) {
  const {
    normalizeSiteID,
    normalizeServiceProfile,
    normalizeHost,
    formatCertificateExpiryByLanguage,
    certificateDaysLeft,
    resolvePublicServiceURL,
    formatServiceProfile
  } = deps;
  const hostCounts = new Map();
  for (const site of state.filteredSites) {
    const hostKey = String(site?.primary_host || site?.id || "").trim().toLowerCase();
    if (!hostKey) {
      continue;
    }
    hostCounts.set(hostKey, Number(hostCounts.get(hostKey) || 0) + 1);
  }
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
              <thead>
                <tr>
                  <th class="waf-check-col">
                    <input type="checkbox" id="services-select-all"${state.filteredSites.length && state.filteredSites.every((site) => state.selectedSiteIDs.has(site.id)) ? " checked" : ""}>
                  </th>
                  <th>${escapeHtml(ctx.t("sites.table.name"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.profile"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.upstream"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.tls"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.updated"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.status"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.actions"))}</th>
                </tr>
              </thead>
              <tbody>
                ${state.filteredSites.length ? state.filteredSites.map((site) => {
                  const siteID = normalizeSiteID(site.id);
                  const easyProfile = state.easyProfilesBySite.get(siteID) || null;
                  const serviceProfile = normalizeServiceProfile(easyProfile?.front_service?.profile);
                  const upstream = state.upstreamsBySite.get(siteID)?.[0] || null;
                  const tls = state.tlsBySite.get(siteID);
                  const language = String(ctx.getLanguage?.() || "en").trim().toLowerCase();
                  const certificateFromID = state.certificates.find((item) => String(item?.id || "").trim().toLowerCase() === String(tls?.certificate_id || "").trim().toLowerCase());
                  const certificateFallback = state.certificateBySiteID.get(siteID) || state.certificateByHost.get(normalizeHost(site.primary_host));
                  const certificate = certificateFromID || certificateFallback || null;
                  const tlsState = tls
                    ? "managed"
                    : (certificate ? "detected" : "missing");
                  const certificateTitle = String(certificate?.common_name || certificate?.id || "").trim();
                  const certificateExpiry = formatCertificateExpiryByLanguage(certificate?.not_after, language);
                  const certificateExpiryDays = certificateDaysLeft(certificate?.not_after);
                  const certificateIsExpiring = typeof certificateExpiryDays === "number" && certificateExpiryDays < 30;
                  const serviceURL = resolvePublicServiceURL(site, tlsState);
                  const displayHost = String(site.primary_host || site.id || "").trim();
                  const hostKey = displayHost.toLowerCase();
                  const showSiteIDMeta = Number(hostCounts.get(hostKey) || 0) > 1 || String(site?._origin || "") === "secondary";
                  const siteIDMetaHTML = showSiteIDMeta
                    ? `<div class="muted waf-services-site-id">${escapeHtml(site.id)}</div>`
                    : "";
                  const tlsBadgeClass = tlsState === "missing"
                    ? "badge-neutral"
                    : (certificateIsExpiring ? "badge-danger" : "badge-success");
                  const tlsBadgeTitle = tlsState === "missing"
                    ? ctx.t("sites.state.tlsMissing")
                    : (certificateTitle || ctx.t("sites.state.tlsManaged"));
                  return `
                    <tr class="waf-table-row-clickable" data-open-site-edit="${escapeHtml(site.id)}">
                      <td class="waf-check-col">
                        <input type="checkbox" data-select-site="${escapeHtml(site.id)}"${state.selectedSiteIDs.has(site.id) ? " checked" : ""}>
                      </td>
                      <td>
                        <button class="waf-link-button" type="button" data-open-service="${escapeHtml(serviceURL)}" title="${escapeHtml(ctx.t("sites.action.openService"))}">${escapeHtml(displayHost)}</button>
                        ${siteIDMetaHTML}
                      </td>
                      <td>${escapeHtml(formatServiceProfile(serviceProfile, ctx))}</td>
                      <td>${upstream ? `${escapeHtml(upstream.host)}:${escapeHtml(String(upstream.port))}` : escapeHtml(ctx.t("common.notSet"))}</td>
                      <td>
                        <div class="waf-services-tls-cell">
                          <div class="badge ${tlsBadgeClass} waf-services-tls-badge">
                            <div class="waf-services-tls-badge-title">${escapeHtml(tlsBadgeTitle)}</div>
                            ${tlsState !== "missing" && certificateTitle
                              ? `<div class="waf-services-tls-badge-expire">${escapeHtml(ctx.t("sites.table.tlsValidTill"))}: ${escapeHtml(certificateExpiry)}</div>`
                              : ""}
                          </div>
                        </div>
                      </td>
                      <td>${escapeHtml(formatDate(site.updated_at || site.created_at))}</td>
                      <td>${statusBadge(site.enabled ? "active" : "failed")}</td>
                      <td>
                        <div class="waf-actions">
                          <button class="btn ghost btn-sm" type="button" data-open-site="${escapeHtml(site.id)}">${escapeHtml(ctx.t("common.edit"))}</button>
                          <button class="btn ghost btn-sm" type="button" data-toggle-site="${escapeHtml(site.id)}" data-toggle-enabled="${site.enabled ? "1" : "0"}">${escapeHtml(ctx.t(site.enabled ? "common.disable" : "common.enable"))}</button>
                        </div>
                      </td>
                    </tr>
                  `;
                }).join("") : `
                  <tr>
                    <td colspan="8">
                      <div class="waf-empty">${escapeHtml(ctx.t("sites.empty.sites"))}</div>
                    </td>
                  </tr>
                `}
              </tbody>
            </table>
          </div>
          <input id="services-import-file" type="file" accept=".env,text/plain" class="waf-hidden">
        </div>
      </section>
    </div>
  `;
}

export function renderModeTabs(activeMode, ctx) {
  const mode = String(activeMode || "easy").trim().toLowerCase() === "raw" ? "raw" : "easy";
  return `
    <div class="waf-mode-switch">
      <button class="btn ${mode === "easy" ? "primary" : "ghost"} btn-sm" type="button" data-mode-tab="easy">${escapeHtml(ctx.t("sites.mode.easy"))}</button>
      <button class="btn ghost btn-sm" type="button" disabled>${escapeHtml(ctx.t("sites.mode.advanced"))}</button>
      <button class="btn ${mode === "raw" ? "primary" : "ghost"} btn-sm" type="button" data-mode-tab="raw">${escapeHtml(ctx.t("sites.mode.raw"))}</button>
    </div>
    ${mode === "raw" ? "" : `<div class="waf-note">${escapeHtml(ctx.t("sites.mode.note"))}</div>`}
  `;
}

export function renderWizardNav(activeTab, ctx) {
  const items = [
    { id: "front", title: ctx.t("sites.wizard.front.title"), subtitle: ctx.t("sites.wizard.front.subtitle") },
    { id: "upstream", title: ctx.t("sites.wizard.upstream.title"), subtitle: ctx.t("sites.wizard.upstream.subtitle") },
    { id: "http", title: ctx.t("sites.easy.tab.http.title"), subtitle: ctx.t("sites.easy.tab.http.subtitle") },
    { id: "headers", title: ctx.t("sites.easy.tab.headers.title"), subtitle: ctx.t("sites.easy.tab.headers.subtitle") },
    { id: "traffic", title: ctx.t("sites.easy.tab.traffic.title"), subtitle: ctx.t("sites.easy.tab.traffic.subtitle") },
    { id: "blocking", title: ctx.t("sites.easy.tab.blocking.title"), subtitle: ctx.t("sites.easy.tab.blocking.subtitle") },
    { id: "antibot", title: ctx.t("sites.easy.tab.antibot.title"), subtitle: ctx.t("sites.easy.tab.antibot.subtitle") },
    { id: "geo", title: ctx.t("sites.easy.tab.geo.title"), subtitle: ctx.t("sites.easy.tab.geo.subtitle") },
    { id: "modsec", title: ctx.t("sites.easy.tab.modsec.title"), subtitle: ctx.t("sites.easy.tab.modsec.subtitle") },
    { id: "websocket", title: ctx.t("sites.easy.tab.websocket.title"), subtitle: ctx.t("sites.easy.tab.websocket.subtitle") },
    { id: "virtualpatches", title: ctx.t("sites.easy.tab.virtualpatches.title"), subtitle: ctx.t("sites.easy.tab.virtualpatches.subtitle") }
  ];
  return `
    <aside class="waf-service-wizard-nav" role="tablist" aria-label="${escapeHtml(ctx.t("sites.wizard.aria"))}">
      ${items.map((item, index) => `
        <button
          class="waf-service-wizard-item${activeTab === item.id ? " is-active" : ""}"
          type="button"
          role="tab"
          aria-selected="${activeTab === item.id ? "true" : "false"}"
          data-wizard-tab="${item.id}">
          <div class="waf-service-step-index">${index + 1}</div>
          <div class="waf-service-wizard-copy">
            <div class="waf-service-wizard-title">${escapeHtml(item.title)}</div>
            <div class="waf-service-wizard-subtitle">${escapeHtml(item.subtitle)}</div>
          </div>
        </button>
      `).join("")}
    </aside>
  `;
}

export function renderRawEditor(state, ctx, isNew, deps) {
  const missingFields = deps.normalizeArray(state.rawMissingFields);
  return `
    <section class="waf-card waf-service-editor-card">
      <div class="waf-card-body waf-stack">
        <div id="sites-feedback"></div>
        <form id="service-editor-form" class="waf-form waf-stack">
          <div class="waf-list-title">${escapeHtml(ctx.t("sites.raw.title"))}</div>
          <div class="waf-note">${escapeHtml(ctx.t("sites.raw.description"))}</div>
          ${missingFields.length ? `
            <div class="waf-empty">
              <strong>${escapeHtml(ctx.t("sites.raw.missingFieldsTitle"))}</strong>
              <pre class="waf-code">${escapeHtml(missingFields.map((field) => deps.toEnvKey(field)).join("\n"))}</pre>
            </div>
          ` : ""}
          <div class="waf-field full">
            <label for="service-raw-env">${escapeHtml(ctx.t("sites.raw.label"))}</label>
            <textarea id="service-raw-env" rows="32" class="waf-code">${escapeHtml(state.rawEnvText || deps.draftToEnvText(state.draft))}</textarea>
          </div>
          <div class="waf-actions waf-actions-between">
            <button class="btn ghost btn-sm" type="button" id="service-back-bottom">${escapeHtml(ctx.t("common.back"))}</button>
            <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t(isNew ? "sites.action.createSite" : "sites.action.saveSite"))}</button>
          </div>
        </form>
      </div>
    </section>
  `;
}

export function ensureControlPlaneAccessManagementMethods(draft, normalizeStringArray) {
  const siteID = String(draft?.id || "").trim().toLowerCase();
  if (siteID !== "control-plane-access") {
    return draft;
  }
  const required = ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"];
  const methods = normalizeStringArray(draft.allowed_methods).map((item) => item.toUpperCase());
  const merged = [...methods];
  for (const method of required) {
    if (!merged.includes(method)) {
      merged.push(method);
    }
  }
  return { ...draft, allowed_methods: merged };
}

export function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig, normalizeSiteID) {
  if (!existingSite) {
    return true;
  }
  if (String(existingSite?._origin || "") === "secondary") {
    return true;
  }
  if (existingUpstream && String(existingUpstream?._origin || "") === "secondary") {
    return true;
  }
  if (existingTLSConfig && String(existingTLSConfig?._origin || "") === "secondary") {
    return true;
  }
  const siteID = draft.id.trim().toLowerCase();
  const siteHost = draft.primary_host.trim().toLowerCase();
  if (String(existingSite.id || "").toLowerCase() !== siteID) {
    return true;
  }
  if (String(existingSite.primary_host || "").toLowerCase() !== siteHost) {
    return true;
  }
  if (Boolean(existingSite.enabled) !== Boolean(draft.enabled)) {
    return true;
  }

  const upstreamID = draft.upstream_id.trim().toLowerCase();
  if (!existingUpstream || String(existingUpstream.id || "").toLowerCase() !== upstreamID) {
    return true;
  }
  if (String(existingUpstream.site_id || "").toLowerCase() !== siteID) {
    return true;
  }
  if (String(existingUpstream.host || "") !== String(draft.upstream_host || "").trim()) {
    return true;
  }
  if (Number(existingUpstream.port || 0) !== Number(draft.upstream_port || 0)) {
    return true;
  }
  if (String(existingUpstream.scheme || "").toLowerCase() !== String(draft.upstream_scheme || "").toLowerCase()) {
    return true;
  }

  if (draft.tls_enabled) {
    if (!existingTLSConfig) {
      return true;
    }
    const certificateID = (draft.certificate_id.trim() || `${siteID}-tls`).toLowerCase();
    if (String(existingTLSConfig.site_id || "").toLowerCase() !== siteID) {
      return true;
    }
    if (String(existingTLSConfig.certificate_id || "").toLowerCase() !== certificateID) {
      return true;
    }
  } else if (existingTLSConfig) {
    return true;
  }
  return false;
}

export async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, deps) {
  const targets = sites.filter((site) => selectedSiteIDs.has(site.id));
  for (const site of targets) {
    const siteID = deps.normalizeSiteID(site.id);
    const upstream = upstreamsBySite.get(siteID)?.[0] || null;
    const tlsConfig = tlsBySite.get(siteID) || null;
    const accessPolicy = accessBySite.get(siteID) || null;
    const draft = await deps.hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy);
    deps.downloadText(`${site.id}.env`, deps.draftToEnvText(draft));
  }
  return targets.length;
}

export async function importServicesFiles(files, ctx, deps) {
  if (!Array.isArray(files) || files.length !== 1) {
    throw new Error(ctx.t("sites.error.importSingleEnv"));
  }
  const file = files[0];
  const name = String(file?.name || "").toLowerCase();
  if (!name.endsWith(".env")) {
    throw new Error(`${file?.name || "file"}: ${ctx.t("sites.error.importEnvOnly")}`);
  }
  deps.requirePermissions(ctx, ["sites.write", "upstreams.write", "tls.write", "certificates.write"], "sites.error.importEnvPermissions");
  const text = await file.text();
  const { draft, missingFields } = deps.envToDraft(text);
  const draftWithPreset = deps.applyServiceProfilePresetForMissingFields(draft, missingFields);
  const validationError = deps.validateDraft(draftWithPreset, ctx);
  if (validationError) {
    throw new Error(`${file.name}: ${validationError}`);
  }
  return {
    file: file.name,
    draft: draftWithPreset,
    missingFields,
    rawEnvText: text,
  };
}
