import { confirmAction, escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";
import {
  applyImportPayload as applyImportPayloadModule,
  buildImportInventory as buildImportInventoryModule,
  buildImportPayloadFromDraft as buildImportPayloadFromDraftModule,
  compileAndApplyImportedRevision,
  diffObjects,
  downloadBlob,
  downloadJSON,
  downloadText,
  draftToEnvText as draftToEnvTextModule,
  envToDraft as envToDraftModule,
  importServicesJSON as importServicesJSONModule,
  loadImportInventory as loadImportInventoryModule,
  toEnvKey
} from "./sites.import-export.js";
import {
  applyEasyProfileToDraft as applyEasyProfileToDraftModule,
  draftToEasyProfile as draftToEasyProfileModule,
  hydrateSiteDraft as hydrateSiteDraftModule,
  siteDraftFromData as siteDraftFromDataModule,
  validateDraft as validateDraftModule
} from "./sites.draft-profile.js";
import {
  isAlreadyExistsError as isAlreadyExistsErrorModule,
  isAutoApplyFailureError as isAutoApplyFailureErrorModule,
  resolveACMEAccountEmail as resolveACMEAccountEmailModule,
  upsertSiteResources as upsertSiteResourcesModule
} from "./sites.resource-upsert.js";
import {
  LIST_FIELD_SET as LIST_FIELD_SET_MODULE,
  SETTINGS_SEARCH_INDEX as SETTINGS_SEARCH_INDEX_MODULE,
  buildGeoCatalogFallback as buildGeoCatalogFallbackModule,
  configureSitesGeoListEditorsRuntime,
  countryFlagEmoji as countryFlagEmojiModule,
  getQuickListTemplates as getQuickListTemplatesModule,
  isCountryCode as isCountryCodeModule,
  normalizeGeoCatalogPayload as normalizeGeoCatalogPayloadModule,
  regionDisplayLabel as regionDisplayLabelModule,
  regionDisplayName as regionDisplayNameModule,
  renderCountryEditor as renderCountryEditorModule,
  renderListEditor as renderListEditorModule,
  renderStatusCodesEditor as renderStatusCodesEditorModule
} from "./sites.geo-list-editors.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesModule,
  putWithPostFallback as putWithPostFallbackModule,
  upsertAccessPolicy as upsertAccessPolicyModule
} from "./sites.resource-actions.js";
import {
  ensureControlPlaneAccessManagementMethods as ensureControlPlaneAccessManagementMethodsModule,
  exportSelectedServicesEnv as exportSelectedServicesEnvModule,
  importServicesFiles as importServicesFilesModule,
  renderListView as renderListViewModule,
  renderModeTabs as renderModeTabsModule,
  renderRawEditor as renderRawEditorModule,
  renderWizardNav as renderWizardNavModule,
  shouldUpsertBaseResources as shouldUpsertBaseResourcesModule
} from "./sites.view-io.js";
import {
  formatAuthLastLogin as formatAuthLastLoginModule,
  normalizeAntibotChallengeRules as normalizeAntibotChallengeRulesModule,
  normalizeAuthBasicUsers as normalizeAuthBasicUsersModule,
  normalizeAuthSessionTTLMinutes as normalizeAuthSessionTTLMinutesModule,
  normalizeCustomLimitRules as normalizeCustomLimitRulesModule,
  normalizeStringArray as normalizeStringArrayModule,
  parseIntListInput as parseIntListInputModule,
  parseListInput as parseListInputModule,
  renderAntibotChallengeRulesEditor as renderAntibotChallengeRulesEditorModule,
  renderAuthPasswordToggleButton as renderAuthPasswordToggleButtonModule,
  renderAuthSessionTtlOptions as renderAuthSessionTtlOptionsModule,
  renderAuthUsersEditor as renderAuthUsersEditorModule,
  renderCustomLimitRulesEditor as renderCustomLimitRulesEditorModule,
  syncAuthPasswordToggle as syncAuthPasswordToggleModule
} from "./sites.auth-rules-editors.js";
import { renderAntibotHelpModal, renderAuthHelpModal } from "./sites.auth-help-modals.js";
import {
  BAN_SCOPE_VALUES as BAN_SCOPE_VALUES_MODULE,
  applyServiceProfilePresetForMissingFields as applyServiceProfilePresetForMissingFieldsModule,
  applyServiceProfilePresetToDraft as applyServiceProfilePresetToDraftModule,
  buildReverseProxyHostFromUpstream as buildReverseProxyHostFromUpstreamModule,
  computeUpstreamID as computeUpstreamIDModule,
  formatBanDurationSeconds as formatBanDurationSecondsModule,
  formatServiceProfile as formatServiceProfileModule,
  isValidEmail as isValidEmailModule,
  normalizeAPIPositiveEndpointPolicies as normalizeAPIPositiveEndpointPoliciesModule,
  normalizeBanEscalationStages as normalizeBanEscalationStagesModule,
  normalizeEmail as normalizeEmailModule,
  normalizeHost as normalizeHostModule,
  normalizeReverseProxyHost as normalizeReverseProxyHostModule,
  normalizeServiceProfile as normalizeServiceProfileModule,
  normalizeSiteID as normalizeSiteIDModule,
  parseBanDurationSeconds as parseBanDurationSecondsModule,
  resolvePublicServiceURL as resolvePublicServiceURLModule,
  resolveReverseProxyHost as resolveReverseProxyHostModule
} from "./sites.service-policy-helpers.js";
import { BAD_BEHAVIOR_STATUS_OPTIONS as BAD_BEHAVIOR_STATUS_OPTIONS_MODULE, CONTINENT_VALUES as CONTINENT_VALUES_MODULE, COUNTRY_GROUP_VALUES as COUNTRY_GROUP_VALUES_MODULE, GEO_SELECTOR_LABELS as GEO_SELECTOR_LABELS_MODULE, QUICK_LIST_TEMPLATES as QUICK_LIST_TEMPLATES_MODULE } from "./sites.geo-presets.js";
import { defaultSiteDraft as defaultSiteDraftModule, requirePermissions as requirePermissionsModule, userPermissionsSet as userPermissionsSetModule } from "./sites.defaults-permissions.js";
import { buildDetailDraftFromForm } from "./sites.detail-draft-builder.js";
import { highlightSelector as highlightSelectorModule, normalizeAutoSiteID as normalizeAutoSiteIDModule, syncDerivedFieldsFromID as syncDerivedFieldsFromIDModule, syncStateDraftFromForm as syncStateDraftFromFormModule, toggleCertificateImportActions as toggleCertificateImportActionsModule } from "./sites.detail-bind-helpers.js";
import { bindDetailSearchAndListEvents } from "./sites.detail-events-search-lists.js";
import { bindDetailActionEvents } from "./sites.detail-events-actions.js";
import { bindDetailRuleEvents } from "./sites.detail-events-rules.js";
import { bindDetailRuntime } from "./sites.detail-bind-runtime.js";
import { renderDetailViewRuntime } from "./sites.detail-render-view.js";
import { renderSitesRuntime } from "./sites.page-render-runtime.js";
import { applyFilters as applyFiltersRuntime, rebuildIndexes as rebuildIndexesRuntime, syncDraftFromRoute as syncDraftFromRouteRuntime } from "./sites.runtime-state.js";
import { bindListRuntime, loadSitesRuntime } from "./sites.runtime-load-list.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesFacade,
  ensureControlPlaneAccessManagementMethods as ensureControlPlaneAccessManagementMethodsFacade,
  isAlreadyExistsError as isAlreadyExistsErrorFacade,
  isAutoApplyFailureError as isAutoApplyFailureErrorFacade,
  putWithPostFallback as putWithPostFallbackFacade,
  resolveACMEAccountEmail as resolveACMEAccountEmailFacade,
  shouldUpsertBaseResources as shouldUpsertBaseResourcesFacade,
  upsertAccessPolicy as upsertAccessPolicyFacade,
  upsertSiteResources as upsertSiteResourcesFacade
} from "./sites.page-resource-facade.js";
import {
  certificateDaysLeft as certificateDaysLeftModule,
  findEasyProfile as findEasyProfileModule,
  formatCertificateExpiryByLanguage as formatCertificateExpiryByLanguageModule,
  go as goModule,
  mergeByID as mergeByIDModule,
  mergeProfilesBySite as mergeProfilesBySiteModule,
  normalizeArray as normalizeArrayModule,
  notifyExpiringCertificates as notifyExpiringCertificatesModule,
  routeBase as routeBaseModule,
  routeInfo as routeInfoModule,
  tryGetJSON as tryGetJSONModule,
  unwrapList as unwrapListModule
} from "./sites.page-utilities.js";
import {
  BAN_SCOPE_VALUES as BAN_SCOPE_VALUES_FACADE,
  applyServiceProfilePresetForMissingFields as applyServiceProfilePresetForMissingFieldsFacade,
  applyServiceProfilePresetToDraft as applyServiceProfilePresetToDraftFacade,
  buildReverseProxyHostFromUpstream as buildReverseProxyHostFromUpstreamFacade,
  computeUpstreamID as computeUpstreamIDFacade,
  formatAuthLastLogin as formatAuthLastLoginFacade,
  formatBanDurationSeconds as formatBanDurationSecondsFacade,
  formatServiceProfile as formatServiceProfileFacade,
  isValidEmail as isValidEmailFacade,
  normalizeAPIPositiveEndpointPolicies as normalizeAPIPositiveEndpointPoliciesFacade,
  normalizeAntibotExclusionRules as normalizeAntibotExclusionRulesFacade,
  normalizeAntibotChallengeRules as normalizeAntibotChallengeRulesFacade,
  normalizeAuthBasicUsers as normalizeAuthBasicUsersFacade,
  normalizeAuthExclusionRules as normalizeAuthExclusionRulesFacade,
  normalizeAuthMode as normalizeAuthModeFacade,
  normalizeAuthOrder as normalizeAuthOrderFacade,
  normalizeAuthSessionTTLMinutes as normalizeAuthSessionTTLMinutesFacade,
  normalizeAuthServiceTokens as normalizeAuthServiceTokensFacade,
  normalizeCustomLimitRules as normalizeCustomLimitRulesFacade,
  normalizeEmail as normalizeEmailFacade,
  normalizeHost as normalizeHostFacade,
  normalizeReverseProxyHost as normalizeReverseProxyHostFacade,
  normalizeServiceProfile as normalizeServiceProfileFacade,
  normalizeSiteID as normalizeSiteIDFacade,
  normalizeStringArray as normalizeStringArrayFacade,
  parseBanDurationSeconds as parseBanDurationSecondsFacade,
  parseIntListInput as parseIntListInputFacade,
  parseListInput as parseListInputFacade,
  readAuthExclusionDraftRows as readAuthExclusionDraftRowsFacade,
  renderAntibotExclusionRulesEditor as renderAntibotExclusionRulesEditorFacade,
  renderAntibotChallengeRulesEditor as renderAntibotChallengeRulesEditorFacade,
  renderAuthExclusionRulesEditor as renderAuthExclusionRulesEditorFacade,
  renderAuthPasswordToggleButton as renderAuthPasswordToggleButtonFacade,
  renderAuthSessionTtlOptions as renderAuthSessionTtlOptionsFacade,
  renderAuthServiceTokensEditor as renderAuthServiceTokensEditorFacade,
  renderAuthUsersEditor as renderAuthUsersEditorFacade,
  renderCustomLimitRulesEditor as renderCustomLimitRulesEditorFacade,
  resolvePublicServiceURL as resolvePublicServiceURLFacade,
  resolveReverseProxyHost as resolveReverseProxyHostFacade,
  syncAuthPasswordToggle as syncAuthPasswordToggleFacade
} from "./sites.page-policy-facade.js";
import {
  LIST_FIELD_SET as LIST_FIELD_SET_FACADE,
  SETTINGS_SEARCH_INDEX as SETTINGS_SEARCH_INDEX_FACADE,
  applyEasyProfileToDraft as applyEasyProfileToDraftFacade,
  buildGeoCatalogFallback as buildGeoCatalogFallbackFacade,
  buildImportInventory as buildImportInventoryFacade,
  buildImportPayloadFromDraft as buildImportPayloadFromDraftFacade,
  countryFlagEmoji as countryFlagEmojiFacade,
  defaultSiteDraft as defaultSiteDraftFacade,
  draftToEasyProfile as draftToEasyProfileFacade,
  draftToEnvText as draftToEnvTextFacade,
  envToDraft as envToDraftFacade,
  getQuickListTemplates as getQuickListTemplatesFacade,
  hydrateSiteDraft as hydrateSiteDraftFacade,
  isCountryCode as isCountryCodeFacade,
  normalizeGeoCatalogPayload as normalizeGeoCatalogPayloadFacade,
  regionDisplayLabel as regionDisplayLabelFacade,
  regionDisplayName as regionDisplayNameFacade,
  renderCountryEditor as renderCountryEditorFacade,
  renderListEditor as renderListEditorFacade,
  renderListView as renderListViewFacade,
  renderModeTabs as renderModeTabsFacade,
  renderRawEditor as renderRawEditorFacade,
  renderStatusCodesEditor as renderStatusCodesEditorFacade,
  renderWizardNav as renderWizardNavFacade,
  requirePermissions as requirePermissionsFacade,
  siteDraftFromData as siteDraftFromDataFacade,
  userPermissionsSet as userPermissionsSetFacade,
  validateDraft as validateDraftFacade
} from "./sites.page-view-facade.js";
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

const routeBase = routeBaseModule;
const routeInfo = routeInfoModule;
const go = goModule;
const formatCertificateExpiryByLanguage = formatCertificateExpiryByLanguageModule;
const certificateDaysLeft = certificateDaysLeftModule;
const normalizeArray = normalizeArrayModule;
const notifyExpiringCertificates = notifyExpiringCertificatesModule;
const tryGetJSON = tryGetJSONModule;
const mergeByID = mergeByIDModule;
const unwrapList = unwrapListModule;
const findEasyProfile = findEasyProfileModule;
const normalizeStringArray = (value) => normalizeStringArrayFacade(value, normalizeArray);
const parseListInput = parseListInputFacade;
const parseIntListInput = parseIntListInputFacade;
const normalizeCustomLimitRules = (value) => normalizeCustomLimitRulesFacade(value, normalizeArray);
const normalizeAntibotExclusionRules = (value) => normalizeAntibotExclusionRulesFacade(value, normalizeArray);
const normalizeAntibotChallengeRules = (value) => normalizeAntibotChallengeRulesFacade(value, normalizeArray);
const normalizeAuthBasicUsers = (value) => normalizeAuthBasicUsersFacade(value, normalizeArray);
const normalizeAuthExclusionRules = (value) => normalizeAuthExclusionRulesFacade(value, normalizeArray);
const normalizeAuthServiceTokens = (value) => normalizeAuthServiceTokensFacade(value, normalizeArray);
const normalizeAuthMode = normalizeAuthModeFacade;
const normalizeAuthOrder = normalizeAuthOrderFacade;
const readAuthExclusionDraftRows = readAuthExclusionDraftRowsFacade;
const normalizeAuthSessionTTLMinutes = normalizeAuthSessionTTLMinutesFacade;
const formatAuthLastLogin = formatAuthLastLoginFacade;
const renderAuthPasswordToggleButton = (index, ctx) => renderAuthPasswordToggleButtonFacade(index, ctx, escapeHtml);
const syncAuthPasswordToggle = syncAuthPasswordToggleFacade;
const renderAuthUsersEditor = (users, ctx) => renderAuthUsersEditorFacade(users, ctx, escapeHtml, normalizeArray);
const renderAuthExclusionRulesEditor = (rules, ctx) => renderAuthExclusionRulesEditorFacade(rules, ctx, escapeHtml, normalizeArray);
const renderAuthServiceTokensEditor = (tokens, ctx) => renderAuthServiceTokensEditorFacade(tokens, ctx, escapeHtml, normalizeArray);
const renderAuthSessionTtlOptions = (ttlMinutes, ctx) => renderAuthSessionTtlOptionsFacade(ttlMinutes, ctx, escapeHtml);
const renderCustomLimitRulesEditor = (rules, ctx) => renderCustomLimitRulesEditorFacade(rules, ctx, escapeHtml, normalizeArray);
const renderAntibotExclusionRulesEditor = (rules, ctx) => renderAntibotExclusionRulesEditorFacade(rules, ctx, escapeHtml, normalizeArray);
const renderAntibotChallengeRulesEditor = (rules, ctx) => renderAntibotChallengeRulesEditorFacade(rules, ctx, escapeHtml, normalizeArray);
const renderAuthHelpModalSafe = (ctx) => renderAuthHelpModal(ctx, { escapeHtml });
const renderAntibotHelpModalSafe = (ctx) => renderAntibotHelpModal(ctx, { escapeHtml });
const normalizeHost = normalizeHostFacade;
const normalizeSiteID = normalizeSiteIDFacade;
const BAN_SCOPE_VALUES = BAN_SCOPE_VALUES_FACADE;
const normalizeServiceProfile = normalizeServiceProfileFacade;
const formatServiceProfile = formatServiceProfileFacade;
const normalizeAPIPositiveEndpointPolicies = (value) => normalizeAPIPositiveEndpointPoliciesFacade(value, normalizeStringArray);
const applyServiceProfilePresetToDraft = applyServiceProfilePresetToDraftFacade;
const applyServiceProfilePresetForMissingFields = (draft, missingFields) => applyServiceProfilePresetForMissingFieldsFacade(draft, missingFields, normalizeArray);
const normalizeEmail = normalizeEmailFacade;
const parseBanDurationSeconds = parseBanDurationSecondsFacade;
const formatBanDurationSeconds = formatBanDurationSecondsFacade;
const normalizeBanEscalationStages = (values, fallbackBase = 300) => normalizeBanEscalationStagesFacade(values, fallbackBase, normalizeArray);
const normalizeReverseProxyHost = normalizeReverseProxyHostFacade;
const buildReverseProxyHostFromUpstream = buildReverseProxyHostFromUpstreamFacade;
const resolveReverseProxyHost = resolveReverseProxyHostFacade;
const isValidEmail = isValidEmailFacade;
const resolvePublicServiceURL = resolvePublicServiceURLFacade;
function computeUpstreamID(siteID) {
  return computeUpstreamIDFacade(siteID);
}
const mergeProfilesBySite = (primary, secondaryPayload) => mergeProfilesBySiteModule(primary, secondaryPayload, normalizeSiteID);

const BAD_BEHAVIOR_STATUS_OPTIONS = BAD_BEHAVIOR_STATUS_OPTIONS_MODULE;
const CONTINENT_VALUES = CONTINENT_VALUES_MODULE;
const COUNTRY_GROUP_VALUES = COUNTRY_GROUP_VALUES_MODULE;
const GEO_SELECTOR_LABELS = GEO_SELECTOR_LABELS_MODULE;
const QUICK_LIST_TEMPLATES = QUICK_LIST_TEMPLATES_MODULE;

configureSitesGeoListEditorsRuntime({
  GEO_SELECTOR_LABELS,
  CONTINENT_VALUES,
  COUNTRY_GROUP_VALUES,
  QUICK_LIST_TEMPLATES,
  BAD_BEHAVIOR_STATUS_OPTIONS
});

const regionDisplayName = regionDisplayNameFacade;
const isCountryCode = isCountryCodeFacade;
const countryFlagEmoji = countryFlagEmojiFacade;
const regionDisplayLabel = regionDisplayLabelFacade;
const buildGeoCatalogFallback = buildGeoCatalogFallbackFacade;
const normalizeGeoCatalogPayload = normalizeGeoCatalogPayloadFacade;
const getQuickListTemplates = getQuickListTemplatesFacade;
const LIST_FIELD_SET = LIST_FIELD_SET_FACADE;
const SETTINGS_SEARCH_INDEX = SETTINGS_SEARCH_INDEX_FACADE;
const renderListEditor = renderListEditorFacade;
const renderCountryEditor = renderCountryEditorFacade;
const renderStatusCodesEditor = renderStatusCodesEditorFacade;
const renderRawEditor = renderRawEditorFacade;
function defaultSiteDraft() {
  return defaultSiteDraftFacade();
}

function siteDraftFromData(site, upstream, tlsConfig) {
  return siteDraftFromDataFacade(site, upstream, tlsConfig);
}

function applyEasyProfileToDraft(draft, profile) {
  return applyEasyProfileToDraftFacade(draft, profile, {
    normalizeServiceProfile,
    normalizeEmail,
    resolveReverseProxyHost,
    normalizeStringArray,
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies
  });
}

async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null) {
  return hydrateSiteDraftFacade(ctx, site, upstream, tlsConfig, accessPolicy, {
    defaultSiteDraft,
    tryGetJSON,
    findEasyProfile,
    normalizeStringArray,
    applyEasyProfileToDraft: (draft, profile) => applyEasyProfileToDraft(draft, profile),
    normalizeServiceProfile,
    normalizeEmail,
    resolveReverseProxyHost,
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies
  });
}

function draftToEasyProfile(draft) {
  return draftToEasyProfileFacade(draft, {
    resolveReverseProxyHost,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeServiceProfile,
    normalizeEmail,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAPIPositiveEndpointPolicies
  });
}

function validateDraft(draft, ctx) {
  return validateDraftFacade(draft, ctx, {
    normalizeStringArray,
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
    normalizeAuthExclusionRules,
    normalizeAuthServiceTokens,
    normalizeAuthMode,
    normalizeAuthOrder
  });
}

function userPermissionsSet(ctx) {
  return userPermissionsSetFacade(ctx);
}

function requirePermissions(ctx, requiredPermissions, errorKey) {
  return requirePermissionsFacade(ctx, requiredPermissions, errorKey);
}
function draftToEnvText(draft) {
  return draftToEnvTextFacade(draft);
}

function envToDraft(text) {
  return envToDraftFacade(text);
}

function buildImportPayloadFromDraft(draft) {
  return buildImportPayloadFromDraftFacade(draft, {
    ensureControlPlaneAccessManagementMethods,
    computeUpstreamID,
    draftToEasyProfile
  });
}

function buildImportInventory(resources = {}) {
  return buildImportInventoryFacade(resources, {
    normalizeArray,
    normalizeSiteID
  });
}

async function loadImportInventory(ctx) {
  return loadImportInventoryModule(ctx, (inventoryResources) => buildImportInventory(inventoryResources));
}

async function applyImportPayload(ctx, payload, inventory = null) {
  return applyImportPayloadModule(ctx, payload, inventory, {
    normalizeSiteID,
    upsertSiteResources,
    putWithPostFallback
  });
}

async function importServicesJSON(file, ctx) {
  return importServicesJSONModule(file, ctx, {
    requirePermissions,
    normalizeArray
  });
}

function toggleCertificateImportActions(containerNode) {
  return toggleCertificateImportActionsModule(containerNode);
}

function parseRawDraft(rawEnvText) {
  return envToDraft(String(rawEnvText || ""));
}

function ensureControlPlaneAccessManagementMethods(draft) {
  return ensureControlPlaneAccessManagementMethodsFacade(draft);
}

function upsertAccessPolicy(draft, ctx, existingAccessPolicy, options = {}) {
  return upsertAccessPolicyFacade(draft, ctx, existingAccessPolicy, options, {
    normalizeSiteID,
    normalizeArray
  });
}

export {
  routeBase,
  routeInfo,
  go,
  formatCertificateExpiryByLanguage,
  certificateDaysLeft,
  normalizeArray,
  notifyExpiringCertificates,
  tryGetJSON,
  mergeByID,
  unwrapList,
  findEasyProfile,
  normalizeStringArray,
  parseListInput,
  parseIntListInput,
  normalizeCustomLimitRules,
  normalizeAntibotExclusionRules,
  normalizeAntibotChallengeRules,
  normalizeAuthBasicUsers,
  normalizeAuthExclusionRules,
  normalizeAuthServiceTokens,
  normalizeAuthMode,
  normalizeAuthOrder,
  readAuthExclusionDraftRows,
  normalizeAuthSessionTTLMinutes,
  formatAuthLastLogin,
  renderAuthPasswordToggleButton,
  syncAuthPasswordToggle,
  renderAuthUsersEditor,
  renderAuthExclusionRulesEditor,
  renderAuthServiceTokensEditor,
  renderAuthSessionTtlOptions,
  renderCustomLimitRulesEditor,
  renderAntibotExclusionRulesEditor,
  renderAntibotChallengeRulesEditor,
  renderAuthHelpModalSafe as renderAuthHelpModal,
  renderAntibotHelpModalSafe as renderAntibotHelpModal,
  normalizeHost,
  normalizeSiteID,
  BAN_SCOPE_VALUES,
  normalizeServiceProfile,
  formatServiceProfile,
  normalizeAPIPositiveEndpointPolicies,
  applyServiceProfilePresetToDraft,
  applyServiceProfilePresetForMissingFields,
  normalizeEmail,
  parseBanDurationSeconds,
  formatBanDurationSeconds,
  normalizeBanEscalationStages,
  normalizeReverseProxyHost,
  buildReverseProxyHostFromUpstream,
  resolveReverseProxyHost,
  isValidEmail,
  resolvePublicServiceURL,
  computeUpstreamID,
  mergeProfilesBySite,
  regionDisplayName,
  isCountryCode,
  countryFlagEmoji,
  regionDisplayLabel,
  buildGeoCatalogFallback,
  normalizeGeoCatalogPayload,
  getQuickListTemplates,
  LIST_FIELD_SET,
  SETTINGS_SEARCH_INDEX,
  renderListEditor,
  renderCountryEditor,
  renderStatusCodesEditor,
  defaultSiteDraft,
  siteDraftFromData,
  applyEasyProfileToDraft,
  hydrateSiteDraft,
  draftToEasyProfile,
  validateDraft,
  userPermissionsSet,
  requirePermissions,
  draftToEnvText,
  envToDraft,
  renderRawEditor,
  buildImportPayloadFromDraft,
  buildImportInventory,
  loadImportInventory,
  applyImportPayload,
  importServicesJSON,
  ensureControlPlaneAccessManagementMethods,
  upsertAccessPolicy,
  bindDetailRuntime,
  loadSitesRuntime,
  bindListRuntime,
  rebuildIndexesRuntime,
  applyFiltersRuntime,
  syncDraftFromRouteRuntime,
  setLoading,
  setError,
  escapeHtml,
  downloadJSON,
  downloadText,
  toEnvKey,
  confirmAction,
  buildDetailDraftFromForm,
  syncStateDraftFromFormModule,
  normalizeAutoSiteIDModule,
  syncDerivedFieldsFromIDModule,
  highlightSelectorModule,
  bindDetailActionEvents,
  toggleCertificateImportActions,
  downloadBlob,
  parseRawDraft,
  bindDetailSearchAndListEvents,
  bindDetailRuleEvents
};
