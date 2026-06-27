import { confirmAction, setError, setLoading } from "../ui.js";
import { normalizeArray, go } from "./sites.routing-merge.js";
import {
  applyServiceProfilePresetForMissingFields,
  applyServiceProfilePresetToDraft,
  normalizeAPIPositiveEndpointPolicies,
  normalizeAntibotChallengeRules,
  normalizeAntibotExclusionRules,
  normalizeCustomLimitRules,
  normalizeGeoTimeWindows,
  normalizeWSBlockPatterns,
  normalizeServiceProfile,
  normalizeStringArray,
} from "./sites.normalize.js";
import {
  normalizeAuthBasicUsers,
  normalizeAuthSessionTTLMinutes,
  syncAuthPasswordToggle,
} from "./sites.auth-geo.js";
import {
  normalizeAuthExclusionRules,
  normalizeAuthMode,
  normalizeAuthOrder,
  normalizeAuthServiceTokens,
} from "./sites.auth-extended-editors.js";
import {
  BAN_SCOPE_VALUES,
  computeUpstreamID,
  normalizeBanEscalationStages,
  normalizeEmail,
  parseBanDurationSeconds,
} from "./sites.traffic-helpers.js";
import { LIST_FIELD_SET, getQuickListTemplates } from "./sites.geo-lists.js";
import { defaultSiteDraft, draftToEasyProfile } from "./sites.draft-core.js";
import {
  highlightSelector as highlightSelectorModule,
  normalizeAutoSiteID,
  parseRawDraftFromContainer,
  syncDerivedFieldsFromID as syncDerivedFieldsFromIDModule,
  toggleCertificateImportActions as toggleCertificateImportActionsModule,
} from "./sites.detail-draft.js";
import { buildDetailDraftFromForm } from "./sites.detail-draft-builder.js";
import { syncStateDraftFromForm as syncStateDraftFromFormModule } from "./sites.detail-bind-helpers.js";
import { bindDetailListEditors } from "./sites.detail-list-bindings.js";
import { bindDetailSubmitDelete } from "./sites.detail-submit-delete.js";
import { bindDetailCore } from "./sites.detail-core-bindings.js";
import { bindDetailRuleEvents } from "./sites.detail-events-rules.js";
import {
  bindDetailBulkDelete,
  bindDetailCertificateActions,
} from "./sites.detail-certs-bulk.js";
import {
  deleteServiceWithResources,
  draftToEnvText,
  ensureControlPlaneAccessManagementMethods,
  envToDraft,
  putWithPostFallback,
  shouldUpsertBaseResources,
  upsertAccessPolicy,
  upsertSiteResources,
  validateDraft,
} from "./sites.stable-resources.js";
import { downloadBlob } from "./sites.import-pipeline.js";

export function bindStableDetail(container, state, ctx, deps) {
  const { load, render } = deps;
  const feedback = container.querySelector("#sites-feedback");
  const parseRawDraft = () => parseRawDraftFromContainer(container, state, {
    envToDraft,
    normalizeArray,
    applyServiceProfilePresetForMissingFields,
  });
  const getDraft = () => buildDetailDraftFromForm(container, state, {
    computeUpstreamID,
    normalizeServiceProfile,
    normalizeEmail,
    normalizeStringArray,
    normalizeArray,
    normalizeBanEscalationStages,
    normalizeAntibotExclusionRules,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies,
    normalizeGeoTimeWindows,
    normalizeWSBlockPatterns,
  });
  const syncStateDraftFromForm = () => syncStateDraftFromFormModule(state, getDraft, {
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
  });
  const syncDerivedFieldsFromID = (idInput, certificateInput, upstreamInput) => syncDerivedFieldsFromIDModule(
    idInput,
    certificateInput,
    upstreamInput,
    computeUpstreamID
  );

  bindDetailCore(container, state, ctx, {
    go,
    render,
    getDraft,
    parseRawDraft,
    syncStateDraftFromForm,
    draftToEnvText,
    ensureControlPlaneAccessManagementMethods,
    normalizeAutoSiteID,
    syncDerivedFieldsFromID,
    normalizeServiceProfile,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions: toggleCertificateImportActionsModule,
    highlightSelector: highlightSelectorModule,
  });

  bindDetailBulkDelete(container, state, ctx, { load, deleteServiceWithResources });
  bindDetailCertificateActions(container, ctx, { load, downloadBlob });
  bindDetailListEditors(container, state, {
    LIST_FIELD_SET,
    getQuickListTemplates,
    normalizeStringArray,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthServiceTokens,
    syncAuthPasswordToggle,
    normalizeArray,
    parseBanDurationSeconds,
    normalizeBanEscalationStages,
    normalizeGeoTimeWindows,
    normalizeWSBlockPatterns,
    setError,
    render,
    syncStateDraftFromForm,
    ctx,
    feedback,
  });

  bindDetailSubmitDelete(container, state, ctx, {
    parseRawDraft,
    getDraft,
    syncStateDraftFromForm,
    ensureControlPlaneAccessManagementMethods,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    putWithPostFallback,
    draftToEasyProfile,
    go,
    deleteServiceWithResources,
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
    setError,
  });

  if (state.highlightedSelector) {
    window.setTimeout(() => highlightSelectorModule(container, state.highlightedSelector), 30);
  }
}

export { defaultSiteDraft };
