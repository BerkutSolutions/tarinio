import {
  LIST_FIELD_SET as LIST_FIELD_SET_MODULE,
  SETTINGS_SEARCH_INDEX as SETTINGS_SEARCH_INDEX_MODULE,
  buildGeoCatalogFallback as buildGeoCatalogFallbackModule,
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
  applyEasyProfileToDraft as applyEasyProfileToDraftModule,
  draftToEasyProfile as draftToEasyProfileModule,
  hydrateSiteDraft as hydrateSiteDraftModule,
  siteDraftFromData as siteDraftFromDataModule,
  validateDraft as validateDraftModule
} from "./sites.draft-profile.js";
import {
  draftToEnvText as draftToEnvTextModule,
  envToDraft as envToDraftModule,
  buildImportPayloadFromDraft as buildImportPayloadFromDraftModule,
  buildImportInventory as buildImportInventoryModule
} from "./sites.import-export.js";
import { defaultSiteDraft as defaultSiteDraftModule, requirePermissions as requirePermissionsModule, userPermissionsSet as userPermissionsSetModule } from "./sites.defaults-permissions.js";
import { renderListView as renderListViewModule, renderModeTabs as renderModeTabsModule, renderRawEditor as renderRawEditorModule, renderWizardNav as renderWizardNavModule } from "./sites.view-io.js";

export const LIST_FIELD_SET = LIST_FIELD_SET_MODULE;
export const SETTINGS_SEARCH_INDEX = SETTINGS_SEARCH_INDEX_MODULE;

export function buildGeoCatalogFallback() {
  return buildGeoCatalogFallbackModule();
}

export function normalizeGeoCatalogPayload(payload) {
  return normalizeGeoCatalogPayloadModule(payload);
}

export function getQuickListTemplates(field) {
  return getQuickListTemplatesModule(field);
}

export function regionDisplayName(code) {
  return regionDisplayNameModule(code);
}

export function isCountryCode(value) {
  return isCountryCodeModule(value);
}

export function countryFlagEmoji(code) {
  return countryFlagEmojiModule(code);
}

export function regionDisplayLabel(code) {
  return regionDisplayLabelModule(code);
}

export function renderListEditor(field, label, values, placeholder = "", options = {}) {
  return renderListEditorModule(field, label, values, placeholder, options);
}

export function renderCountryEditor(field, label, values, catalog, options = {}) {
  return renderCountryEditorModule(field, label, values, catalog, options);
}

export function renderStatusCodesEditor(selectedCodes, ctx) {
  return renderStatusCodesEditorModule(selectedCodes, ctx);
}

export function defaultSiteDraft() {
  return defaultSiteDraftModule();
}

export function siteDraftFromData(site, upstream, tlsConfig) {
  return siteDraftFromDataModule(site, upstream, tlsConfig, defaultSiteDraft);
}

export function applyEasyProfileToDraft(draft, profile, deps) {
  return applyEasyProfileToDraftModule(draft, profile, deps);
}

export function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null, deps) {
  return hydrateSiteDraftModule(ctx, site, upstream, tlsConfig, accessPolicy, deps);
}

export function draftToEasyProfile(draft, deps) {
  return draftToEasyProfileModule(draft, deps);
}

export function validateDraft(draft, ctx, deps) {
  return validateDraftModule(draft, ctx, deps);
}

export function userPermissionsSet(ctx) {
  return userPermissionsSetModule(ctx);
}

export function requirePermissions(ctx, requiredPermissions, errorKey) {
  return requirePermissionsModule(ctx, requiredPermissions, errorKey);
}

export function draftToEnvText(draft) {
  return draftToEnvTextModule(draft, defaultSiteDraft);
}

export function envToDraft(text) {
  return envToDraftModule(text, defaultSiteDraft);
}

export function buildImportPayloadFromDraft(draft, deps) {
  return buildImportPayloadFromDraftModule(draft, deps);
}

export function buildImportInventory(resources = {}, deps) {
  return buildImportInventoryModule(resources, deps);
}

export function renderListView(state, ctx, deps) {
  return renderListViewModule(state, ctx, deps);
}

export function renderModeTabs(activeMode, ctx) {
  return renderModeTabsModule(activeMode, ctx);
}

export function renderWizardNav(activeTab, ctx) {
  return renderWizardNavModule(activeTab, ctx);
}

export function renderRawEditor(state, ctx, isNew, deps) {
  return renderRawEditorModule(state, ctx, isNew, deps);
}
