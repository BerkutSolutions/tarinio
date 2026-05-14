import {
  BAN_SCOPE_VALUES,
  LIST_FIELD_SET,
  SETTINGS_SEARCH_INDEX,
  applyEasyProfileToDraft,
  applyFiltersRuntime,
  applyImportPayload,
  applyServiceProfilePresetForMissingFields,
  applyServiceProfilePresetToDraft,
  bindDetailActionEvents,
  bindDetailRuleEvents,
  bindDetailRuntime,
  bindDetailSearchAndListEvents,
  bindListRuntime,
  buildDetailDraftFromForm,
  buildGeoCatalogFallback,
  computeUpstreamID,
  confirmAction,
  defaultSiteDraft,
  downloadBlob,
  downloadJSON,
  downloadText,
  draftToEasyProfile,
  draftToEnvText,
  ensureControlPlaneAccessManagementMethods,
  envToDraft,
  escapeHtml,
  getQuickListTemplates,
  go,
  highlightSelectorModule,
  hydrateSiteDraft,
  importServicesJSON,
  isValidEmail,
  loadSitesRuntime,
  mergeByID,
  mergeProfilesBySite,
  normalizeAPIPositiveEndpointPolicies,
  normalizeArray,
  normalizeAntibotChallengeRules,
  normalizeAuthBasicUsers,
  normalizeAuthSessionTTLMinutes,
  normalizeAutoSiteIDModule,
  normalizeBanEscalationStages,
  normalizeCustomLimitRules,
  normalizeEmail,
  normalizeGeoCatalogPayload,
  normalizeHost,
  normalizeServiceProfile,
  normalizeSiteID,
  normalizeStringArray,
  notifyExpiringCertificates,
  parseBanDurationSeconds,
  parseRawDraft,
  rebuildIndexesRuntime,
  renderAntibotChallengeRulesEditor,
  renderAuthSessionTtlOptions,
  renderAuthUsersEditor,
  renderCountryEditor,
  renderCustomLimitRulesEditor,
  renderListEditor,
  renderRawEditor,
  renderStatusCodesEditor,
  requirePermissions,
  resolvePublicServiceURL,
  routeBase,
  routeInfo,
  setError,
  setLoading,
  syncAuthPasswordToggle,
  syncDerivedFieldsFromIDModule,
  syncDraftFromRouteRuntime,
  syncStateDraftFromFormModule,
  toEnvKey,
  toggleCertificateImportActions,
  tryGetJSON,
  unwrapList,
  upsertAccessPolicy,
  validateDraft
} from "./sites.page-main-core.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesPart2,
  exportSelectedServicesEnv as exportSelectedServicesEnvPart2,
  importServicesFiles as importServicesFilesPart2,
  isAlreadyExistsError as isAlreadyExistsErrorPart2,
  isAutoApplyFailureError as isAutoApplyFailureErrorPart2,
  putWithPostFallback as putWithPostFallbackPart2,
  renderDetailView as renderDetailViewPart2,
  renderListView as renderListViewPart2,
  renderModeTabs as renderModeTabsPart2,
  renderWizardNav as renderWizardNavPart2,
  resolveACMEAccountEmail as resolveACMEAccountEmailPart2,
  shouldUpsertBaseResources as shouldUpsertBaseResourcesPart2,
  upsertSiteResources as upsertSiteResourcesPart2
} from "./sites.page-main-render-runtime.js";
import {
  deleteServiceWithResourcesRuntime,
  exportSelectedServicesEnvRuntime,
  importServicesFilesRuntime,
  isAlreadyExistsErrorRuntime,
  isAutoApplyFailureErrorRuntime,
  putWithPostFallbackRuntime,
  renderDetailViewRuntimeBridge,
  renderListViewRuntime,
  renderModeTabsRuntime,
  renderWizardNavRuntime,
  resolveACMEAccountEmailRuntime,
  shouldUpsertBaseResourcesRuntime,
  upsertSiteResourcesRuntime
} from "./sites.page-main-actions-runtime.js";

function renderListView(state, ctx) {
  return renderListViewRuntime(state, ctx, {
    renderListViewPart2,
    normalizeSiteID,
    normalizeServiceProfile,
    normalizeHost,
    formatCertificateExpiryByLanguage,
    certificateDaysLeft,
    resolvePublicServiceURL,
    formatServiceProfile
  });
}

function renderModeTabs(activeMode, ctx) {
  return renderModeTabsRuntime(activeMode, ctx, {
    renderModeTabsPart2
  });
}

function renderWizardNav(activeTab, ctx) {
  return renderWizardNavRuntime(activeTab, ctx, {
    renderWizardNavPart2
  });
}

function renderDetailView(state, ctx) {
  return renderDetailViewRuntimeBridge(state, ctx, {
    renderDetailViewPart2,
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

async function resolveACMEAccountEmail(draft, ctx) {
  return resolveACMEAccountEmailRuntime(draft, ctx, {
    resolveACMEAccountEmailPart2,
    normalizeEmail,
    isValidEmail,
    normalizeArray
  });
}

function isAutoApplyFailureError(error) {
  return isAutoApplyFailureErrorRuntime(error, {
    isAutoApplyFailureErrorPart2
  });
}

function isAlreadyExistsError(error) {
  return isAlreadyExistsErrorRuntime(error, {
    isAlreadyExistsErrorPart2
  });
}

async function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}) {
  return upsertSiteResourcesRuntime(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options, {
    upsertSiteResourcesPart2,
    normalizeSiteID,
    normalizeArray,
    normalizeEmail,
    isValidEmail
  });
}

async function deleteServiceWithResources(siteID, ctx, snapshot = null) {
  return deleteServiceWithResourcesRuntime(siteID, ctx, snapshot, {
    deleteServiceWithResourcesPart2,
    normalizeSiteID,
    normalizeArray,
    isAutoApplyFailureError
  });
}

async function putWithPostFallback(ctx, path, payload, options = {}) {
  return putWithPostFallbackRuntime(ctx, path, payload, options, {
    putWithPostFallbackPart2,
    isAutoApplyFailureError
  });
}

function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig) {
  return shouldUpsertBaseResourcesRuntime(draft, existingSite, existingUpstream, existingTLSConfig, {
    shouldUpsertBaseResourcesPart2,
    normalizeSiteID
  });
}

async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs) {
  return exportSelectedServicesEnvRuntime(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, {
    exportSelectedServicesEnvPart2,
    normalizeSiteID,
    hydrateSiteDraft,
    downloadText,
    draftToEnvText
  });
}

async function importServicesFiles(files, ctx) {
  return importServicesFilesRuntime(files, ctx, {
    importServicesFilesPart2,
    requirePermissions,
    envToDraft,
    applyServiceProfilePresetForMissingFields,
    validateDraft
  });
}

export function buildRenderSitesDeps() {
  return {
    routeInfo,
    buildGeoCatalogFallback,
    defaultSiteDraft,
    rebuildIndexesRuntime,
    applyFiltersRuntime,
    syncDraftFromRouteRuntime,
    loadSitesRuntime,
    bindListRuntime,
    bindDetailRuntime,
    routeBase,
    go,
    normalizeHost,
    normalizeSiteID,
    normalizeServiceProfile,
    ensureControlPlaneAccessManagementMethods,
    draftToEnvText,
    normalizeArray,
    hydrateSiteDraft,
    setLoading,
    escapeHtml,
    mergeByID,
    unwrapList,
    notifyExpiringCertificates,
    normalizeGeoCatalogPayload,
    mergeProfilesBySite,
    tryGetJSON,
    setError,
    downloadJSON,
    downloadText,
    toEnvKey,
    exportSelectedServicesEnv,
    importServicesFiles,
    confirmAction,
    deleteServiceWithResources,
    putWithPostFallback,
    computeUpstreamID,
    normalizeEmail,
    normalizeStringArray,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies,
    buildDetailDraftFromForm,
    syncStateDraftFromFormModule,
    BAN_SCOPE_VALUES,
    normalizeAutoSiteIDModule,
    syncDerivedFieldsFromIDModule,
    highlightSelectorModule,
    bindDetailActionEvents,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions,
    downloadBlob,
    parseRawDraft,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    draftToEasyProfile,
    getQuickListTemplates,
    LIST_FIELD_SET,
    normalizeCustomLimitRules,
    normalizeAntibotChallengeRules,
    bindDetailSearchAndListEvents,
    bindDetailRuleEvents,
    syncAuthPasswordToggle,
    parseBanDurationSeconds,
    renderListView,
    renderDetailView
  };
}

export {
  computeUpstreamID,
  draftToEnvText,
  envToDraft,
  hydrateSiteDraft,
  renderRawEditor
};
