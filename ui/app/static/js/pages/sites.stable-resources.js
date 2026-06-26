import {
  downloadJSON,
  downloadText,
  draftToEnvText as draftToEnvTextModule,
  envToDraft,
  importServicesFiles as importServicesFilesModule,
  requirePermissions,
  toEnvKey,
} from "./sites.import-pipeline.js";
import {
  isAutoApplyFailureError,
  putWithPostFallback,
  shouldUpsertBaseResources,
  upsertAccessPolicy,
} from "./sites.access-upsert.js";
import {
  resolveACMEAccountEmail,
  upsertSiteResources as upsertSiteResourcesModule,
} from "./sites.resource-pipeline.js";
import { hydrateSiteDraft as hydrateSiteDraftModule } from "./sites.profile-hydration.js";
import { validateDraft as validateDraftModule } from "./sites.validation.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesModule,
  ensureControlPlaneAccessManagementMethods,
  exportSelectedServicesEnv as exportSelectedServicesEnvModule,
  importServicesFiles as importServicesFilesLifecycle,
} from "./sites.service-lifecycle.js";

export {
  downloadJSON,
  ensureControlPlaneAccessManagementMethods,
  envToDraft,
  isAutoApplyFailureError,
  putWithPostFallback,
  requirePermissions,
  shouldUpsertBaseResources,
  toEnvKey,
  upsertAccessPolicy,
};

export async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null) {
  return hydrateSiteDraftModule(ctx, site, upstream, tlsConfig, accessPolicy);
}

export function draftToEnvText(draft) {
  return draftToEnvTextModule(draft);
}

export function validateDraft(draft, ctx) {
  return validateDraftModule(draft, ctx);
}

export async function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}) {
  return upsertSiteResourcesModule(
    draft,
    ctx,
    resolveACMEAccountEmail,
    existingSite,
    existingUpstream,
    existingTLSConfig,
    options
  );
}

export async function deleteServiceWithResources(siteID, ctx, snapshot = null) {
  return deleteServiceWithResourcesModule(siteID, ctx, isAutoApplyFailureError, snapshot);
}

export async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs) {
  return exportSelectedServicesEnvModule(
    ctx,
    sites,
    upstreamsBySite,
    tlsBySite,
    accessBySite,
    selectedSiteIDs,
    hydrateSiteDraft,
  downloadText,
  downloadJSON,
    draftToEnvText
  );
}

export async function importServicesFiles(files, ctx) {
  return importServicesFilesLifecycle(
    files,
    ctx,
    importServicesFilesModule,
    validateDraft,
    upsertSiteResources,
    putWithPostFallback
  );
}
