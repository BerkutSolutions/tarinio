export async function renderSitesRuntime(container, ctx, deps) {
  const {
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
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
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
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    bindDetailSearchAndListEvents,
    bindDetailRuleEvents,
    syncAuthPasswordToggle,
    parseBanDurationSeconds,
    renderListView,
    renderDetailView,
  } = deps;

  const pendingImportedDraftRef = { current: null };
  const route = routeInfo();
  const state = {
    route,
    sites: [],
    upstreams: [],
    tlsConfigs: [],
    certificates: [],
    accessPolicies: [],
    easyProfiles: [],
    geoCatalog: buildGeoCatalogFallback(),
    search: "",
    sort: "updated-desc",
    editorMode: "easy",
    activeTab: "front",
    settingsSearch: "",
    settingsMatches: [],
    highlightedSelector: "",
    rawEnvText: "",
    rawMissingFields: [],
    filteredSites: [],
    selectedSiteIDs: new Set(),
    upstreamsBySite: new Map(),
    tlsBySite: new Map(),
    certificateBySiteID: new Map(),
    certificateByHost: new Map(),
    accessBySite: new Map(),
    easyProfilesBySite: new Map(),
    countryFilters: {
      blacklist_country: "",
      whitelist_country: ""
    },
    listTemplateSelection: {
      blacklist_user_agent: [],
      blacklist_uri: []
    },
    draft: defaultSiteDraft()
  };

  const rebuildIndexes = () => rebuildIndexesRuntime(state, {
    normalizeHost,
    normalizeSiteID
  });

  const applyFilters = () => applyFiltersRuntime(state, {
    normalizeSiteID,
    normalizeServiceProfile
  });

  const syncDraftFromRoute = async () => syncDraftFromRouteRuntime(state, ctx, {
    pendingImportedDraftRef,
    ensureControlPlaneAccessManagementMethods,
    draftToEnvText,
    normalizeArray,
    defaultSiteDraft,
    normalizeSiteID,
    hydrateSiteDraft
  });

  const render = () => {
    applyFilters();
    container.innerHTML = state.route.mode === "list" ? renderListView(state, ctx) : renderDetailView(state, ctx);
    bind();
  };

  const load = async () => loadSitesRuntime(state, ctx, container, {
    setLoading,
    escapeHtml,
    mergeByID,
    unwrapList,
    notifyExpiringCertificates,
    normalizeArray,
    normalizeSiteID,
    normalizeGeoCatalogPayload,
    mergeProfilesBySite,
    tryGetJSON,
    rebuildIndexes,
    syncDraftFromRoute,
    render
  });

  const bindList = () => bindListRuntime(state, ctx, container, {
    go,
    routeBase,
    load,
    render,
    setLoading,
    setError,
    downloadJSON,
    downloadText,
    toEnvKey,
    escapeHtml,
    exportSelectedServicesEnv,
    importServicesFiles,
    pendingImportedDraftRef,
    normalizeArray,
    normalizeSiteID,
    confirmAction,
    deleteServiceWithResources,
    putWithPostFallback
  });

  const bindDetail = () => bindDetailRuntime({
    container,
    state,
    ctx,
    feedback: container.querySelector("#sites-feedback"),
    load,
    render,
    go,
    routeBase,
    computeUpstreamID,
    normalizeServiceProfile,
    normalizeEmail,
    normalizeStringArray,
    normalizeArray,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies,
    buildDetailDraftFromForm,
    syncStateDraftFromFormModule,
    BAN_SCOPE_VALUES,
    normalizeAutoSiteIDModule,
    syncDerivedFieldsFromIDModule,
    highlightSelectorModule,
    bindDetailActionEvents,
    normalizeSiteID,
    confirmAction,
    setError,
    setLoading,
    deleteServiceWithResources,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions,
    downloadBlob,
    ensureControlPlaneAccessManagementMethods,
    parseRawDraft,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    putWithPostFallback,
    draftToEasyProfile,
    draftToEnvText,
    getQuickListTemplates,
    LIST_FIELD_SET,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    bindDetailSearchAndListEvents,
    bindDetailRuleEvents,
    syncAuthPasswordToggle,
    parseBanDurationSeconds
  });

  const bind = () => {
    if (state.route.mode === "list") {
      bindList();
      return;
    }
    bindDetail();
  };

  await load();
}
