export async function upsertAccessPolicy(draft, ctx, existingAccessPolicy, deps) {
  const siteID = deps.normalizeSiteID(draft.id);
  if (!siteID) {
    return;
  }
  const allowlistSources = [];
  if (draft.use_allowlist || deps.normalizeStringArray(draft.access_allowlist).length) {
    allowlistSources.push(...deps.normalizeStringArray(draft.access_allowlist));
  }
  const allowlist = Array.from(new Set(allowlistSources));
  const denylist = deps.normalizeStringArray(draft.access_denylist);
  if (!allowlist.length && !denylist.length && !existingAccessPolicy) {
    return;
  }
  const payload = {
    id: String(existingAccessPolicy?.id || `${siteID}-access`),
    site_id: siteID,
    enabled: true,
    allowlist,
    denylist
  };
  const resolvePolicyForSite = async () => {
    const accessPolicies = deps.normalizeArray(await ctx.api.get("/api/access-policies"));
    return accessPolicies.find((item) => deps.normalizeSiteID(item?.site_id) === siteID) || null;
  };
  const normalizeListForCompare = (values) => deps.normalizeStringArray(values).slice().sort();
  const matchesPayload = (policy) => {
    if (!policy) {
      return false;
    }
    if (deps.normalizeSiteID(policy.site_id) !== siteID) {
      return false;
    }
    const policyAllow = normalizeListForCompare(policy.allowlist);
    const policyDeny = normalizeListForCompare(policy.denylist);
    const expectedAllow = normalizeListForCompare(payload.allowlist);
    const expectedDeny = normalizeListForCompare(payload.denylist);
    return JSON.stringify(policyAllow) === JSON.stringify(expectedAllow) &&
      JSON.stringify(policyDeny) === JSON.stringify(expectedDeny);
  };
  if (!allowlist.length && !denylist.length && existingAccessPolicy) {
    const deleteByID = async (policyID) => {
      const normalizedID = String(policyID || "").trim();
      if (!normalizedID) {
        return false;
      }
      try {
        await ctx.api.delete(`/api/access-policies/${encodeURIComponent(normalizedID)}`);
      } catch (error) {
        const policyForSite = await resolvePolicyForSite();
        if (!policyForSite) {
          return true;
        }
        if (error?.status === 404) {
          return false;
        }
        if (deps.isAutoApplyFailureError(error)) {
          return false;
        }
        if (error?.status === 400 || error?.status === 409 || error?.status === 500) {
          return false;
        }
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
          if (persisted) {
            throw new Error(`access policy delete failed for site ${siteID}`);
          }
        }
      } else {
        const persisted = await resolvePolicyForSite();
        if (persisted) {
          throw new Error(`access policy delete failed for site ${siteID}`);
        }
      }
    }
    return;
  }
  try {
    await ctx.api.post("/api/access-policies/upsert", payload);
  } catch (error) {
    if (deps.isAutoApplyFailureError(error)) {
      const policyForSite = await resolvePolicyForSite();
      if (matchesPayload(policyForSite)) {
        return;
      }
    }
    if (error?.status === 404 || error?.status === 405) {
      // Backward compatibility with older backend versions without upsert endpoint.
      if (existingAccessPolicy) {
        try {
          await ctx.api.put(`/api/access-policies/${encodeURIComponent(payload.id)}`, payload);
          return;
        } catch (putError) {
          if (deps.isAutoApplyFailureError(putError)) {
            const policyForSite = await resolvePolicyForSite();
            if (matchesPayload(policyForSite)) {
              return;
            }
          }
          if (putError?.status !== 404) {
            throw putError;
          }
        }
      }
      try {
        await ctx.api.post("/api/access-policies", payload);
      } catch (postError) {
        if (deps.isAutoApplyFailureError(postError)) {
          const policyForSite = await resolvePolicyForSite();
          if (matchesPayload(policyForSite)) {
            return;
          }
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
        try {
          await ctx.api.put(`/api/access-policies/${encodeURIComponent(String(policyForSite.id))}`, upsertPayload);
        } catch (putError) {
          if (deps.isAutoApplyFailureError(putError)) {
            const persistedPolicy = await resolvePolicyForSite();
            if (matchesPayload(persistedPolicy)) {
              return;
            }
          }
          throw putError;
        }
        return;
      }
    }
    throw error;
  }
}

export async function deleteServiceWithResources(siteID, ctx, snapshot = null, deps) {
  const normalizedSiteID = deps.normalizeSiteID(siteID);
  if (!normalizedSiteID) {
    return;
  }
  const normalizeIDValue = (value) => String(value || "").trim().toLowerCase();
  const isNotFound = (error) => error?.status === 404;
  const includesByID = (items, id) => deps.normalizeArray(items).some((item) => normalizeIDValue(item?.id) === normalizeIDValue(id));
  const includesBySiteID = (items, id) => deps.normalizeArray(items).some((item) => normalizeIDValue(item?.site_id) === normalizeIDValue(id));
  const deleteIgnoreSafe = async (path, verifyDeleted = null) => {
    try {
      await ctx.api.delete(path);
    } catch (error) {
      if (isNotFound(error)) {
        return;
      }
      if (deps.isAutoApplyFailureError(error) && typeof verifyDeleted === "function") {
        const deleted = await verifyDeleted();
        if (deleted) {
          return;
        }
      }
      if (!deps.isAutoApplyFailureError(error)) {
        throw error;
      }
    }
  };
  const sites = Array.isArray(snapshot?.sites) ? snapshot.sites : await ctx.api.get("/api/sites").catch(() => []);
  const upstreams = Array.isArray(snapshot?.upstreams)
    ? snapshot.upstreams
    : await ctx.api.get("/api/upstreams").catch(() => []);
  const tlsConfigs = Array.isArray(snapshot?.tlsConfigs)
    ? snapshot.tlsConfigs
    : await ctx.api.get("/api/tls-configs").catch(() => []);
  const easyProfiles = Array.isArray(snapshot?.easyProfiles)
    ? snapshot.easyProfiles
    : await ctx.api.get("/api/easy-site-profiles").catch(() => []);
  const upstreamsForSite = deps.normalizeArray(upstreams).filter((item) => deps.normalizeSiteID(item?.site_id) === normalizedSiteID);
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
  for (const policy of deps.normalizeArray(wafPolicies).filter((item) => deps.normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/waf-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/waf-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of deps.normalizeArray(ratePolicies).filter((item) => deps.normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/rate-limit-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/rate-limit-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of deps.normalizeArray(accessPolicies).filter((item) => deps.normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/access-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/access-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  if (hasSite) {
    await deleteIgnoreSafe(`/api/sites/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/sites").catch(() => []);
      return !includesByID(latest, normalizedSiteID);
    });
  }
  for (const upstream of upstreamsForSite) {
    const upstreamID = String(upstream?.id || "").trim();
    if (!upstreamID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/upstreams/${encodeURIComponent(upstreamID)}`, async () => {
      const latest = await ctx.api.get("/api/upstreams").catch(() => []);
      return !includesByID(latest, upstreamID);
    });
  }
}

export async function putWithPostFallback(ctx, path, payload, options = {}, deps) {
  const tolerateAutoApplyError = Boolean(options?.tolerateAutoApplyError);
  const verifyPersisted = typeof options?.verifyPersisted === "function" ? options.verifyPersisted : null;
  const requestOptions = options?.requestOptions || {};
  try {
    await ctx.api.post(path, payload, requestOptions);
    return;
  } catch (postError) {
    if (tolerateAutoApplyError && deps.isAutoApplyFailureError(postError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) {
        return;
      }
    }
    if (postError?.status !== 404 && postError?.status !== 405) {
      throw postError;
    }
  }
  try {
    await ctx.api.put(path, payload, requestOptions);
  } catch (putError) {
    if (tolerateAutoApplyError && deps.isAutoApplyFailureError(putError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) {
        return;
      }
    }
    throw putError;
  }
}
