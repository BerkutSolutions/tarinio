import { normalizeArray, normalizeSiteID } from "./sites.routing-merge.js";
import { normalizeStringArray } from "./sites.normalize.js";

export function isAutoApplyFailureError(error) {
  const message = String(error?.message || "").toLowerCase();
  return message.includes("apply failed") ||
    message.includes("reload failed") ||
    message.includes("health-check failed") ||
    message.includes("default upstream is required") ||
    message.includes("revision apply") ||
    message.includes("limit_req");
}

export async function upsertAccessPolicy(draft, ctx, existingAccessPolicy, options = {}) {
  const siteID = normalizeSiteID(draft.id);
  if (!siteID) return;
  const requestOptions = options?.requestOptions || {};
  const allowlistSources = [];
  if (draft.use_allowlist || normalizeStringArray(draft.access_allowlist).length) {
    allowlistSources.push(...normalizeStringArray(draft.access_allowlist));
  }
  const allowlist = Array.from(new Set(allowlistSources));
  const denylist = normalizeStringArray(draft.access_denylist);
  if (!allowlist.length && !denylist.length && !existingAccessPolicy) return;
  const payload = { id: String(existingAccessPolicy?.id || `${siteID}-access`), site_id: siteID, enabled: true, allowlist, denylist };
  const resolvePolicyForSite = async () => {
    const accessPolicies = normalizeArray(await ctx.api.get("/api/access-policies", requestOptions));
    return accessPolicies.find((item) => normalizeSiteID(item?.site_id) === siteID) || null;
  };
  const normalizeListForCompare = (values) => normalizeStringArray(values).slice().sort();
  const matchesPayload = (policy) => {
    if (!policy || normalizeSiteID(policy.site_id) !== siteID) return false;
    const policyAllow = normalizeListForCompare(policy.allowlist);
    const policyDeny = normalizeListForCompare(policy.denylist);
    const expectedAllow = normalizeListForCompare(payload.allowlist);
    const expectedDeny = normalizeListForCompare(payload.denylist);
    return JSON.stringify(policyAllow) === JSON.stringify(expectedAllow) && JSON.stringify(policyDeny) === JSON.stringify(expectedDeny);
  };
  if (!allowlist.length && !denylist.length && existingAccessPolicy) {
    const deleteByID = async (policyID) => {
      const normalizedID = String(policyID || "").trim();
      if (!normalizedID) return false;
      try { await ctx.api.delete(`/api/access-policies/${encodeURIComponent(normalizedID)}`, requestOptions); }
      catch (error) {
        const policyForSite = await resolvePolicyForSite();
        if (!policyForSite) return true;
        if (error?.status === 404 || isAutoApplyFailureError(error) || error?.status === 400 || error?.status === 409 || error?.status === 500) return false;
        throw error;
      }
      return true;
    };
    const deletedByPayloadID = await deleteByID(payload.id);
    if (!deletedByPayloadID) {
      const policyForSite = await resolvePolicyForSite();
      if (policyForSite?.id && String(policyForSite.id) !== String(payload.id)) {
        const deletedBySiteID = await deleteByID(policyForSite.id);
        if (!deletedBySiteID) {
          const persisted = await resolvePolicyForSite();
          if (persisted) throw new Error(`access policy delete failed for site ${siteID}`);
        }
      } else {
        const persisted = await resolvePolicyForSite();
        if (persisted) throw new Error(`access policy delete failed for site ${siteID}`);
      }
    }
    return;
  }
  try {
    await ctx.api.post("/api/access-policies/upsert", payload, requestOptions);
  } catch (error) {
    if (isAutoApplyFailureError(error)) {
      const policyForSite = await resolvePolicyForSite();
      if (matchesPayload(policyForSite)) return;
    }
    if (error?.status === 404 || error?.status === 405) {
      if (existingAccessPolicy) {
        try { await ctx.api.put(`/api/access-policies/${encodeURIComponent(payload.id)}`, payload, requestOptions); return; }
        catch (putError) {
          if (isAutoApplyFailureError(putError)) {
            const policyForSite = await resolvePolicyForSite();
            if (matchesPayload(policyForSite)) return;
          }
          if (putError?.status !== 404) throw putError;
        }
      }
      try { await ctx.api.post("/api/access-policies", payload, requestOptions); }
      catch (postError) {
        if (isAutoApplyFailureError(postError)) {
          const policyForSite = await resolvePolicyForSite();
          if (matchesPayload(policyForSite)) return;
        }
        throw postError;
      }
      return;
    }
    const message = String(error?.message || "").toLowerCase();
    if (error?.status === 400 && message.includes("already exists")) {
      const policyForSite = await resolvePolicyForSite();
      if (policyForSite?.id) {
        const upsertPayload = { ...payload, id: String(policyForSite.id) };
        try { await ctx.api.put(`/api/access-policies/${encodeURIComponent(String(policyForSite.id))}`, upsertPayload, requestOptions); }
        catch (putError) {
          if (isAutoApplyFailureError(putError)) {
            const persistedPolicy = await resolvePolicyForSite();
            if (matchesPayload(persistedPolicy)) return;
          }
          throw putError;
        }
        return;
      }
    }
    throw error;
  }
}

export async function putWithPostFallback(ctx, path, payload, options = {}) {
  const tolerateAutoApplyError = Boolean(options?.tolerateAutoApplyError);
  const verifyPersisted = typeof options?.verifyPersisted === "function" ? options.verifyPersisted : null;
  const requestOptions = options?.requestOptions || {};
  try {
    await ctx.api.post(path, payload, requestOptions);
    return;
  } catch (postError) {
    if (tolerateAutoApplyError && isAutoApplyFailureError(postError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) return;
    }
    if (postError?.status !== 404 && postError?.status !== 405) throw postError;
  }
  try {
    await ctx.api.put(path, payload, requestOptions);
  } catch (putError) {
    if (tolerateAutoApplyError && isAutoApplyFailureError(putError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) return;
    }
    throw putError;
  }
}

export function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig) {
  if (!existingSite) return true;
  if (String(existingSite?._origin || "") === "secondary") return true;
  if (existingUpstream && String(existingUpstream?._origin || "") === "secondary") return true;
  if (existingTLSConfig && String(existingTLSConfig?._origin || "") === "secondary") return true;
  const siteID = draft.id.trim().toLowerCase();
  const siteHost = draft.primary_host.trim().toLowerCase();
  if (String(existingSite.id || "").toLowerCase() !== siteID) return true;
  if (String(existingSite.primary_host || "").toLowerCase() !== siteHost) return true;
  if (Boolean(existingSite.enabled) !== Boolean(draft.enabled)) return true;
  const upstreamID = draft.upstream_id.trim().toLowerCase();
  if (!existingUpstream || String(existingUpstream.id || "").toLowerCase() !== upstreamID) return true;
  if (String(existingUpstream.site_id || "").toLowerCase() !== siteID) return true;
  if (String(existingUpstream.host || "") !== String(draft.upstream_host || "").trim()) return true;
  if (Number(existingUpstream.port || 0) !== Number(draft.upstream_port || 0)) return true;
  if (String(existingUpstream.scheme || "").toLowerCase() !== String(draft.upstream_scheme || "").toLowerCase()) return true;
  if (draft.tls_enabled) {
    if (!existingTLSConfig) return true;
    const certificateID = (draft.certificate_id.trim() || `${siteID}-tls`).toLowerCase();
    if (String(existingTLSConfig.site_id || "").toLowerCase() !== siteID) return true;
    if (String(existingTLSConfig.certificate_id || "").toLowerCase() !== certificateID) return true;
  }
  return false;
}
