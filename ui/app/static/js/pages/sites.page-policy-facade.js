import {
  normalizeAntibotExclusionRules as normalizeAntibotExclusionRulesModule,
  renderAntibotExclusionRulesEditor as renderAntibotExclusionRulesEditorModule
} from "./sites.antibot-exclusion-editors.js";
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
import {
  normalizeAuthExclusionRules as normalizeAuthExclusionRulesModule,
  normalizeAuthMode as normalizeAuthModeModule,
  normalizeAuthOrder as normalizeAuthOrderModule,
  normalizeAuthServiceTokens as normalizeAuthServiceTokensModule,
  readAuthExclusionDraftRows as readAuthExclusionDraftRowsModule,
  renderAuthExclusionRulesEditor as renderAuthExclusionRulesEditorModule,
  renderAuthServiceTokensEditor as renderAuthServiceTokensEditorModule
} from "./sites.auth-extended-editors.js";
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

export const BAN_SCOPE_VALUES = BAN_SCOPE_VALUES_MODULE;

export function normalizeStringArray(value, normalizeArray) {
  return normalizeStringArrayModule(value, { normalizeArray });
}

export function parseListInput(value) {
  return parseListInputModule(value);
}

export function parseIntListInput(value) {
  return parseIntListInputModule(value);
}

export function normalizeCustomLimitRules(value, normalizeArray) {
  return normalizeCustomLimitRulesModule(value, { normalizeArray });
}

export function normalizeAntibotExclusionRules(value, normalizeArray) {
  return normalizeAntibotExclusionRulesModule(value, { normalizeArray });
}

export function normalizeAntibotChallengeRules(value, normalizeArray) {
  return normalizeAntibotChallengeRulesModule(value, { normalizeArray });
}

export function normalizeAuthBasicUsers(value, normalizeArray) {
  return normalizeAuthBasicUsersModule(value, { normalizeArray });
}

export function normalizeAuthSessionTTLMinutes(value) {
  return normalizeAuthSessionTTLMinutesModule(value);
}

export function normalizeAuthMode(value) {
  return normalizeAuthModeModule(value);
}

export function normalizeAuthOrder(value) {
  return normalizeAuthOrderModule(value);
}

export function normalizeAuthExclusionRules(value, normalizeArray) {
  return normalizeAuthExclusionRulesModule(value, { normalizeArray });
}

export function readAuthExclusionDraftRows(container) {
  return readAuthExclusionDraftRowsModule(container);
}

export function normalizeAuthServiceTokens(value, normalizeArray) {
  return normalizeAuthServiceTokensModule(value, { normalizeArray });
}

export function formatAuthLastLogin(value, ctx) {
  return formatAuthLastLoginModule(value, ctx);
}

export function renderAuthPasswordToggleButton(index, ctx, escapeHtml) {
  return renderAuthPasswordToggleButtonModule(index, ctx, { escapeHtml });
}

export function syncAuthPasswordToggle(button, visible, ctx) {
  return syncAuthPasswordToggleModule(button, visible, ctx);
}

export function renderAuthUsersEditor(users, ctx, escapeHtml, normalizeArray) {
  return renderAuthUsersEditorModule(users, ctx, { escapeHtml, normalizeArray });
}

export function renderAuthExclusionRulesEditor(rules, ctx, escapeHtml, normalizeArray) {
  return renderAuthExclusionRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

export function renderAuthServiceTokensEditor(tokens, ctx, escapeHtml, normalizeArray) {
  return renderAuthServiceTokensEditorModule(tokens, ctx, { escapeHtml, normalizeArray });
}

export function renderAuthSessionTtlOptions(ttlMinutes, ctx, escapeHtml) {
  return renderAuthSessionTtlOptionsModule(ttlMinutes, ctx, { escapeHtml });
}

export function renderCustomLimitRulesEditor(rules, ctx, escapeHtml, normalizeArray) {
  return renderCustomLimitRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

export function renderAntibotExclusionRulesEditor(rules, ctx, escapeHtml, normalizeArray) {
  return renderAntibotExclusionRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

export function renderAntibotChallengeRulesEditor(rules, ctx, escapeHtml, normalizeArray) {
  return renderAntibotChallengeRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

export function normalizeHost(value) {
  return normalizeHostModule(value);
}

export function normalizeSiteID(value) {
  return normalizeSiteIDModule(value);
}

export function normalizeServiceProfile(value) {
  return normalizeServiceProfileModule(value);
}

export function formatServiceProfile(value, ctx) {
  return formatServiceProfileModule(value, ctx);
}

export function normalizeAPIPositiveEndpointPolicies(value, normalizeStringArrayFn) {
  return normalizeAPIPositiveEndpointPoliciesModule(value, { normalizeStringArray: normalizeStringArrayFn });
}

export function applyServiceProfilePresetToDraft(draft, profile) {
  return applyServiceProfilePresetToDraftModule(draft, profile);
}

export function applyServiceProfilePresetForMissingFields(draft, missingFields, normalizeArray) {
  return applyServiceProfilePresetForMissingFieldsModule(draft, missingFields, { normalizeArray });
}

export function normalizeEmail(value) {
  return normalizeEmailModule(value);
}

export function parseBanDurationSeconds(value) {
  return parseBanDurationSecondsModule(value);
}

export function formatBanDurationSeconds(seconds) {
  return formatBanDurationSecondsModule(seconds);
}

export function normalizeBanEscalationStages(values, fallbackBase = 300, normalizeArray) {
  return normalizeBanEscalationStagesModule(values, fallbackBase, { normalizeArray });
}

export function normalizeReverseProxyHost(value) {
  return normalizeReverseProxyHostModule(value);
}

export function buildReverseProxyHostFromUpstream(upstreamScheme, upstreamHost, upstreamPort) {
  return buildReverseProxyHostFromUpstreamModule(upstreamScheme, upstreamHost, upstreamPort);
}

export function resolveReverseProxyHost(draft, explicitValue = "") {
  return resolveReverseProxyHostModule(draft, explicitValue);
}

export function isValidEmail(value) {
  return isValidEmailModule(value);
}

export function resolvePublicServiceURL(site, tlsState) {
  return resolvePublicServiceURLModule(site, tlsState);
}

export function computeUpstreamID(siteID) {
  return computeUpstreamIDModule(siteID);
}
