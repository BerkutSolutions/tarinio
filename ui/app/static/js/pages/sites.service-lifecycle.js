import { normalizeArray, normalizeSiteID } from "./sites.routing-merge.js";
import { normalizeStringArray } from "./sites.normalize.js";

export function ensureControlPlaneAccessManagementMethods(draft) {
  const siteID = String(draft?.id || "").trim().toLowerCase();
  if (siteID !== "control-plane-access") return draft;
  const required = ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"];
  const methods = normalizeStringArray(draft.allowed_methods).map((item) => item.toUpperCase());
  const merged = [...methods];
  for (const method of required) {
    if (!merged.includes(method)) merged.push(method);
  }
  return { ...draft, allowed_methods: merged };
}

export async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs, hydrateSiteDraft, downloadText, draftToEnvText) {
  const targets = sites.filter((site) => selectedSiteIDs.has(site.id));
  for (const site of targets) {
    const upstream = upstreamsBySite.get(site.id)?.[0] || null;
    const tlsConfig = tlsBySite.get(site.id) || null;
    const accessPolicy = accessBySite.get(normalizeSiteID(site.id)) || null;
    const draft = await hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy);
    downloadText(`${site.id}.env`, draftToEnvText(draft));
  }
  return targets.length;
}

export async function importServicesFiles(files, ctx, importServicesFilesModule, validateDraft, upsertSiteResources, putWithPostFallback) {
  return importServicesFilesModule(
    files,
    ctx,
    ensureControlPlaneAccessManagementMethods,
    validateDraft,
    upsertSiteResources,
    putWithPostFallback
  );
}

export async function deleteServiceWithResources(siteID, ctx, isAutoApplyFailureError, snapshot = null) {
  const normalizedSiteID = normalizeSiteID(siteID);
  if (!normalizedSiteID) return;
  const normalizeIDValue = (value) => String(value || "").trim().toLowerCase();
  const isNotFound = (error) => error?.status === 404;
  const includesByID = (items, id) => normalizeArray(items).some((item) => normalizeIDValue(item?.id) === normalizeIDValue(id));
  const includesBySiteID = (items, id) => normalizeArray(items).some((item) => normalizeIDValue(item?.site_id) === normalizeIDValue(id));
  const deleteIgnoreSafe = async (path, verifyDeleted = null) => {
    try { await ctx.api.delete(path); } catch (error) {
      if (isNotFound(error)) return;
      if (isAutoApplyFailureError(error) && typeof verifyDeleted === "function") {
        const deleted = await verifyDeleted();
        if (deleted) return;
      }
      if (!isAutoApplyFailureError(error)) throw error;
    }
  };
  const sites = Array.isArray(snapshot?.sites) ? snapshot.sites : await ctx.api.get("/api/sites").catch(() => []);
  const upstreams = Array.isArray(snapshot?.upstreams) ? snapshot.upstreams : await ctx.api.get("/api/upstreams").catch(() => []);
  const tlsConfigs = Array.isArray(snapshot?.tlsConfigs) ? snapshot.tlsConfigs : await ctx.api.get("/api/tls-configs").catch(() => []);
  const easyProfiles = Array.isArray(snapshot?.easyProfiles) ? snapshot.easyProfiles : await ctx.api.get("/api/easy-site-profiles").catch(() => []);
  const upstreamsForSite = normalizeArray(upstreams).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID);
  const hasSite = includesByID(sites, normalizedSiteID);
  const hasTLSConfig = includesBySiteID(tlsConfigs, normalizedSiteID);
  const hasEasyProfile = includesBySiteID(easyProfiles, normalizedSiteID);
  if (hasTLSConfig) {
    await deleteIgnoreSafe(`/api/tls-configs/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/tls-configs").catch(() => []);
      return !includesBySiteID(latest, normalizedSiteID);
    });
  }
  const easyAccessPolicyID = `easy-${normalizedSiteID}-access`;
  const hasEasyAccessPolicy = includesByID(snapshot?.accessPolicies, easyAccessPolicyID);
  if (hasEasyAccessPolicy) {
    await deleteIgnoreSafe(`/api/access-policies/${encodeURIComponent(easyAccessPolicyID)}`, async () => {
      const latest = await ctx.api.get("/api/access-policies").catch(() => []);
      return !includesByID(latest, easyAccessPolicyID);
    });
  }
  if (hasEasyProfile) {
    await deleteIgnoreSafe(`/api/easy-site-profiles/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/easy-site-profiles").catch(() => []);
      return !includesBySiteID(latest, normalizedSiteID);
    });
  }
  const [wafPolicies, ratePolicies, accessPolicies] = await Promise.all([
    Array.isArray(snapshot?.wafPolicies) ? snapshot.wafPolicies : ctx.api.get("/api/waf-policies").catch(() => []),
    Array.isArray(snapshot?.ratePolicies) ? snapshot.ratePolicies : ctx.api.get("/api/rate-limit-policies").catch(() => []),
    Array.isArray(snapshot?.accessPolicies) ? snapshot.accessPolicies : ctx.api.get("/api/access-policies").catch(() => [])
  ]);
  for (const policy of normalizeArray(wafPolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) continue;
    await deleteIgnoreSafe(`/api/waf-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/waf-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of normalizeArray(ratePolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) continue;
    await deleteIgnoreSafe(`/api/rate-limit-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/rate-limit-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of normalizeArray(accessPolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) continue;
    await deleteIgnoreSafe(`/api/access-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/access-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const upstream of upstreamsForSite) {
    const upstreamID = String(upstream?.id || "").trim();
    if (!upstreamID) continue;
    await deleteIgnoreSafe(`/api/upstreams/${encodeURIComponent(upstreamID)}`, async () => {
      const latest = await ctx.api.get("/api/upstreams").catch(() => []);
      return !includesByID(latest, upstreamID);
    });
  }
  if (hasSite) {
    await deleteIgnoreSafe(`/api/sites/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/sites").catch(() => []);
      return !includesByID(latest, normalizedSiteID);
    });
  }
}
