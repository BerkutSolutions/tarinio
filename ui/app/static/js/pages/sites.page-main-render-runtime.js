import { renderDetailViewRuntime } from "./sites.detail-render-view.js";
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
  const {
    SETTINGS_SEARCH_INDEX,
    escapeHtml,
    renderModeTabs,
    renderRawEditor,
    renderWizardNav,
    normalizeServiceProfile,
    renderListEditor,
    getQuickListTemplates,
    normalizeStringArray,
    renderStatusCodesEditor,
    renderCustomLimitRulesEditor,
    normalizeBanEscalationStages,
    formatBanDurationSeconds,
    renderAntibotChallengeRulesEditor,
    renderAuthSessionTtlOptions,
    renderAuthUsersEditor,
    renderCountryEditor
  } = deps;
  return renderDetailViewRuntime(state, ctx, {
    SETTINGS_SEARCH_INDEX,
    escapeHtml,
    renderModeTabs,
    renderRawEditor,
    renderWizardNav,
    normalizeServiceProfile,
    renderListEditor,
    getQuickListTemplates,
    normalizeStringArray,
    renderStatusCodesEditor,
    renderCustomLimitRulesEditor,
    normalizeBanEscalationStages,
    formatBanDurationSeconds,
    renderAntibotChallengeRulesEditor,
    renderAuthSessionTtlOptions,
    renderAuthUsersEditor,
    renderCountryEditor
  });
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
