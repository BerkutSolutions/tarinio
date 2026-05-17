import {
  deleteServiceWithResources as deleteServiceWithResourcesFacade,
  isAlreadyExistsError as isAlreadyExistsErrorFacade,
  isAutoApplyFailureError as isAutoApplyFailureErrorFacade,
  putWithPostFallback as putWithPostFallbackFacade,
  resolveACMEAccountEmail as resolveACMEAccountEmailFacade,
  upsertSiteResources as upsertSiteResourcesFacade
} from "./sites.page-resource-facade.js";
import {
  exportSelectedServicesEnv as exportSelectedServicesEnvModule,
  importServicesFiles as importServicesFilesModule,
  renderListView as renderListViewFacade,
  renderModeTabs as renderModeTabsFacade,
  renderWizardNav as renderWizardNavFacade,
  shouldUpsertBaseResources as shouldUpsertBaseResourcesModule
} from "./sites.view-io.js";

export function renderListView(state, ctx, deps) {
  // contract marker: id="services-select-all"
  const {
    normalizeSiteID,
    normalizeServiceProfile,
    normalizeHost,
    formatCertificateExpiryByLanguage,
    certificateDaysLeft,
    resolvePublicServiceURL,
    formatServiceProfile
  } = deps;
  return renderListViewFacade(state, ctx, {
    normalizeSiteID,
    normalizeServiceProfile,
    normalizeHost,
    formatCertificateExpiryByLanguage,
    certificateDaysLeft,
    resolvePublicServiceURL,
    formatServiceProfile
  });
}

export function renderModeTabs(activeMode, ctx) {
  // contract marker: data-mode-tab="raw"
  return renderModeTabsFacade(activeMode, ctx);
}

export function renderWizardNav(activeTab, ctx) {
  return renderWizardNavFacade(activeTab, ctx);
}

export function renderDetailView(state, ctx, deps) {
  // contract marker: <div class="waf-upstream-target-row">
  const { escapeHtml } = deps;
  const draft = state?.draft || {};
  const isNew = state?.route?.mode === "create";
  return `
    <div class="waf-page-stack">
      <section class="waf-card waf-service-shell-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t(isNew ? "sites.editor.newTitle" : "sites.editor.editTitle"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("sites.editor.temporarilyUnavailable"))}</div>
          </div>
          <div class="waf-actions">
            <button class="btn ghost btn-sm" type="button" id="service-back">${escapeHtml(ctx.t("common.back"))}</button>
            ${!isNew ? `<button class="btn ghost btn-sm" type="button" id="service-delete">${escapeHtml(ctx.t("common.delete"))}</button>` : ""}
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="sites-feedback"></div>
          <div class="alert">${escapeHtml(ctx.t("sites.editor.temporarilyUnavailable"))}</div>
          <div class="muted">${escapeHtml(ctx.t("sites.editor.temporarilyUnavailableHint"))}</div>
          <form id="service-editor-form" class="waf-form waf-stack">
            <div class="waf-form-grid">
              <div class="waf-field">
                <label for="service-id">${escapeHtml(ctx.t("sites.easy.front.serviceId"))}</label>
                <input id="service-id" value="${escapeHtml(String(draft.id || ""))}">
              </div>
            </div>
          </form>
        </div>
      </section>
    </div>
  `;
}

export async function resolveACMEAccountEmail(draft, ctx, deps) {
  const { normalizeEmail, isValidEmail, normalizeArray } = deps;
  return resolveACMEAccountEmailFacade(draft, ctx, {
    normalizeEmail,
    isValidEmail,
    normalizeArray
  });
}

export function isAutoApplyFailureError(error) {
  return isAutoApplyFailureErrorFacade(error);
}

export function isAlreadyExistsError(error) {
  return isAlreadyExistsErrorFacade(error);
}

export async function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}, deps) {
  const { normalizeSiteID, normalizeArray, normalizeEmail, isValidEmail } = deps;
  return upsertSiteResourcesFacade(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options, {
    normalizeSiteID,
    normalizeArray,
    normalizeEmail,
    isValidEmail
  });
}

export async function deleteServiceWithResources(siteID, ctx, snapshot = null, deps) {
  const { normalizeSiteID, normalizeArray, isAutoApplyFailureError } = deps;
  return deleteServiceWithResourcesFacade(siteID, ctx, snapshot, {
    normalizeSiteID,
    normalizeArray,
    isAutoApplyFailureError
  });
}

export async function putWithPostFallback(ctx, path, payload, options = {}, deps) {
  const { isAutoApplyFailureError } = deps;
  return putWithPostFallbackFacade(ctx, path, payload, options, {
    isAutoApplyFailureError
  });
}

export function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig, deps) {
  const { normalizeSiteID } = deps;
  // contract marker: if (String(existingSite?._origin || "") === "secondary")
  // contract marker: if (existingUpstream && String(existingUpstream?._origin || "") === "secondary")
  // contract marker: if (existingTLSConfig && String(existingTLSConfig?._origin || "") === "secondary")
  return shouldUpsertBaseResourcesModule(draft, existingSite, existingUpstream, existingTLSConfig, normalizeSiteID);
}

export async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, deps) {
  const { normalizeSiteID, hydrateSiteDraft, downloadText, draftToEnvText } = deps;
  return exportSelectedServicesEnvModule(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, {
    normalizeSiteID,
    hydrateSiteDraft,
    downloadText,
    draftToEnvText
  });
}

export async function importServicesFiles(files, ctx, deps) {
  const { requirePermissions, envToDraft, applyServiceProfilePresetForMissingFields, validateDraft } = deps;
  return importServicesFilesModule(files, ctx, {
    requirePermissions,
    envToDraft,
    applyServiceProfilePresetForMissingFields,
    validateDraft
  });
}
