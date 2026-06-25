export function renderListViewRuntime(state, ctx, deps) {
  const { renderListViewPart2, normalizeSiteID, normalizeServiceProfile, normalizeHost, formatCertificateExpiryByLanguage, certificateDaysLeft, resolvePublicServiceURL, formatServiceProfile } = deps;
  return renderListViewPart2(state, ctx, {
    normalizeSiteID,
    normalizeServiceProfile,
    normalizeHost,
    formatCertificateExpiryByLanguage,
    certificateDaysLeft,
    resolvePublicServiceURL,
    formatServiceProfile
  });
}

export function renderModeTabsRuntime(activeMode, ctx, deps) {
  const { renderModeTabsPart2 } = deps;
  return renderModeTabsPart2(activeMode, ctx);
}

export function renderWizardNavRuntime(activeTab, ctx, deps) {
  const { renderWizardNavPart2 } = deps;
  return renderWizardNavPart2(activeTab, ctx);
}

export function renderDetailViewRuntimeBridge(state, ctx, deps) {
  const {
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
    renderAntibotExclusionRulesEditor,
    normalizeBanEscalationStages,
    formatBanDurationSeconds,
    renderAntibotChallengeRulesEditor,
    renderAuthSessionTtlOptions,
    renderAuthUsersEditor,
    renderCountryEditor
  } = deps;
  return renderDetailViewPart2(state, ctx, {
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
    renderAntibotExclusionRulesEditor,
    normalizeBanEscalationStages,
    formatBanDurationSeconds,
    renderAntibotChallengeRulesEditor,
    renderAuthSessionTtlOptions,
    renderAuthUsersEditor,
    renderCountryEditor
  });
}

export async function resolveACMEAccountEmailRuntime(draft, ctx, deps) {
  const { resolveACMEAccountEmailPart2, normalizeEmail, isValidEmail, normalizeArray } = deps;
  return resolveACMEAccountEmailPart2(draft, ctx, {
    normalizeEmail,
    isValidEmail,
    normalizeArray
  });
}

export function isAutoApplyFailureErrorRuntime(error, deps) {
  const { isAutoApplyFailureErrorPart2 } = deps;
  return isAutoApplyFailureErrorPart2(error);
}

export function isAlreadyExistsErrorRuntime(error, deps) {
  const { isAlreadyExistsErrorPart2 } = deps;
  return isAlreadyExistsErrorPart2(error);
}

export async function upsertSiteResourcesRuntime(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}, deps) {
  const { upsertSiteResourcesPart2, normalizeSiteID, normalizeArray, normalizeEmail, isValidEmail } = deps;
  return upsertSiteResourcesPart2(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options, {
    normalizeSiteID,
    normalizeArray,
    normalizeEmail,
    isValidEmail
  });
}

export async function deleteServiceWithResourcesRuntime(siteID, ctx, snapshot = null, deps) {
  const { deleteServiceWithResourcesPart2, normalizeSiteID, normalizeArray, isAutoApplyFailureError } = deps;
  return deleteServiceWithResourcesPart2(siteID, ctx, snapshot, {
    normalizeSiteID,
    normalizeArray,
    isAutoApplyFailureError
  });
}

export async function putWithPostFallbackRuntime(ctx, path, payload, options = {}, deps) {
  const { putWithPostFallbackPart2, isAutoApplyFailureError } = deps;
  return putWithPostFallbackPart2(ctx, path, payload, options, {
    isAutoApplyFailureError
  });
}

export function shouldUpsertBaseResourcesRuntime(draft, existingSite, existingUpstream, existingTLSConfig, deps) {
  const { shouldUpsertBaseResourcesPart2, normalizeSiteID } = deps;
  return shouldUpsertBaseResourcesPart2(draft, existingSite, existingUpstream, existingTLSConfig, {
    normalizeSiteID
  });
}

export async function exportSelectedServicesEnvRuntime(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, deps) {
  const { exportSelectedServicesEnvPart2, normalizeSiteID, hydrateSiteDraft, downloadText, draftToEnvText } = deps;
  return exportSelectedServicesEnvPart2(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, {
    normalizeSiteID,
    hydrateSiteDraft,
    downloadText,
    draftToEnvText
  });
}

export async function importServicesFilesRuntime(files, ctx, deps) {
  const { importServicesFilesPart2, requirePermissions, envToDraft, applyServiceProfilePresetForMissingFields, validateDraft } = deps;
  return importServicesFilesPart2(files, ctx, {
    requirePermissions,
    envToDraft,
    applyServiceProfilePresetForMissingFields,
    validateDraft
  });
}
