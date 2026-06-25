import {
  isAlreadyExistsError as isAlreadyExistsErrorModule,
  isAutoApplyFailureError as isAutoApplyFailureErrorModule,
  resolveACMEAccountEmail as resolveACMEAccountEmailModule,
  upsertSiteResources as upsertSiteResourcesModule
} from "./sites.resource-upsert.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesModule,
  putWithPostFallback as putWithPostFallbackModule,
  upsertAccessPolicy as upsertAccessPolicyModule
} from "./sites.resource-actions.js";
import {
  ensureControlPlaneAccessManagementMethods as ensureControlPlaneAccessManagementMethodsModule,
  shouldUpsertBaseResources as shouldUpsertBaseResourcesModule
} from "./sites.view-io.js";

export function isAutoApplyFailureError(error) {
  return isAutoApplyFailureErrorModule(error);
}

export function isAlreadyExistsError(error) {
  return isAlreadyExistsErrorModule(error);
}

export function upsertAccessPolicy(draft, ctx, existingAccessPolicy, options = {}, deps) {
  return upsertAccessPolicyModule(draft, ctx, existingAccessPolicy, options, deps);
}

export function resolveACMEAccountEmail(draft, ctx) {
  return resolveACMEAccountEmailModule(draft, ctx);
}

export function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}, deps) {
  return upsertSiteResourcesModule(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options, deps);
}

export function deleteServiceWithResources(siteID, ctx, snapshot = null, deps) {
  return deleteServiceWithResourcesModule(siteID, ctx, snapshot, deps);
}

export function putWithPostFallback(ctx, path, payload, options = {}, deps) {
  return putWithPostFallbackModule(ctx, path, payload, options, deps);
}

export function ensureControlPlaneAccessManagementMethods(draft) {
  return ensureControlPlaneAccessManagementMethodsModule(draft);
}

export function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig) {
  return shouldUpsertBaseResourcesModule(draft, existingSite, existingUpstream, existingTLSConfig);
}
