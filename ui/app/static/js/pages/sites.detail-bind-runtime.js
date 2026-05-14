export function bindDetailRuntime(deps) {
  const {
    container,
    state,
    ctx,
    feedback,
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
    toggleCertificateImportActions: toggleCertificateImportActionsDep,
    downloadBlob,
    ensureControlPlaneAccessManagementMethods,
    parseRawDraft: parseRawDraftDep,
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
    normalizeAntibotChallengeRules,
    bindDetailSearchAndListEvents,
    bindDetailRuleEvents,
    syncAuthPasswordToggle,
    parseBanDurationSeconds
  } = deps;

  container.querySelectorAll("[data-mode-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      const nextMode = String(button.dataset.modeTab || "easy").trim().toLowerCase() === "raw" ? "raw" : "easy";
      if (nextMode === state.editorMode) {
        return;
      }
      if (nextMode === "raw") {
        if (state.editorMode === "easy") {
          syncStateDraftFromForm();
        }
        state.rawEnvText = draftToEnvText(ensureControlPlaneAccessManagementMethods({ ...state.draft }));
        state.rawMissingFields = [];
        state.editorMode = "raw";
        render();
        return;
      }
      try {
        parseRawDraft();
        state.editorMode = "easy";
        render();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.raw.parseError")}: ${String(error?.message || error)}`);
      }
    });
  });

  const getDraft = () => buildDetailDraftFromForm(container, state, {
    computeUpstreamID,
    normalizeServiceProfile,
    normalizeEmail,
    normalizeStringArray,
    normalizeArray,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies
  });

  const syncStateDraftFromForm = () => syncStateDraftFromFormModule(state, getDraft, {
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthSessionTTLMinutes
  });

  const back = () => go(routeBase());
  const toggleCertificateImportActions = () => toggleCertificateImportActionsDep(container);
  const parseRawDraft = () => {
    const rawEnvText = String(container.querySelector("#service-raw-env")?.value || state.rawEnvText || "").trim();
    const parsed = parseRawDraftDep(rawEnvText);
    state.rawEnvText = rawEnvText ? `${rawEnvText}\n` : "";
    state.rawMissingFields = normalizeArray(parsed.missingFields);
    state.draft = ensureControlPlaneAccessManagementMethods(parsed.draft);
    return state.draft;
  };
  const idInput = container.querySelector("#service-id");
  const hostInput = container.querySelector("#service-host");
  const certificateInput = container.querySelector("#service-certificate-id");
  const upstreamInput = container.querySelector("#service-upstream-id");
  const normalizeAutoSiteID = (value) => normalizeAutoSiteIDModule(value);
  const syncDerivedFieldsFromID = () => syncDerivedFieldsFromIDModule(idInput, upstreamInput, certificateInput, computeUpstreamID);
  if (state.route.mode !== "create") {
    if (idInput?.value?.trim()) {
      idInput.dataset.dirty = "true";
    }
    if (certificateInput?.value?.trim()) {
      certificateInput.dataset.dirty = "true";
    }
  }
  container.querySelector("#service-back")?.addEventListener("click", back);
  container.querySelector("#service-back-bottom")?.addEventListener("click", back);
  container.querySelector("#service-raw-env")?.addEventListener("input", (event) => {
    state.rawEnvText = String(event.target.value || "");
    state.rawMissingFields = [];
  });
  container.querySelector("#service-host")?.addEventListener("input", (event) => {
    const host = event.target.value.trim().toLowerCase();
    if (idInput && !idInput.dataset.dirty) {
      idInput.value = normalizeAutoSiteID(host);
      syncDerivedFieldsFromID();
    }
  });

  const highlightSelector = (selector) => highlightSelectorModule(container, selector);

  bindDetailActionEvents({
    container,
    state,
    ctx,
    feedback,
    load,
    render,
    back,
    hostInput,
    idInput,
    normalizeAutoSiteID,
    syncDerivedFieldsFromID,
    normalizeSiteID,
    confirmAction,
    setError,
    setLoading,
    normalizeArray,
    deleteServiceWithResources,
    syncStateDraftFromForm,
    normalizeServiceProfile,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions,
    downloadBlob,
    ensureControlPlaneAccessManagementMethods,
    parseRawDraft,
    getDraft,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    putWithPostFallback,
    draftToEasyProfile,
    go,
    routeBase,
    highlightSelector
  });

  bindDetailSearchAndListEvents({
    container,
    state,
    ctx,
    feedback,
    render,
    syncStateDraftFromForm,
    highlightSelector,
    getQuickListTemplates,
    LIST_FIELD_SET,
    normalizeStringArray,
    normalizeCustomLimitRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
  });

  bindDetailRuleEvents({
    container,
    state,
    ctx,
    feedback,
    render,
    syncStateDraftFromForm,
    normalizeAuthBasicUsers,
    syncAuthPasswordToggle,
    normalizeArray,
    parseBanDurationSeconds,
    normalizeBanEscalationStages,
    setError
  });
}
