import { normalizeArray, normalizeSiteID } from "./sites.routing-merge.js";
import { normalizeStringArray } from "./sites.normalize.js";
import { isValidEmail, normalizeEmail } from "./sites.traffic-helpers.js";
import { isAutoApplyFailureError } from "./sites.access-upsert.js";

export function isAlreadyExistsError(error) {
  const message = String(error?.message || "").toLowerCase();
  return (error?.status === 400 || error?.status === 409) && message.includes("already exists");
}

export async function resolveACMEAccountEmail(draft, ctx) {
  const fromDraft = normalizeEmail(draft?.acme_account_email);
  if (isValidEmail(fromDraft)) return fromDraft;
  const siteID = String(draft?.id || "").trim().toLowerCase();
  if (siteID) {
    try {
      const ownProfile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`);
      const ownEmail = normalizeEmail(ownProfile?.front_service?.acme_account_email);
      if (isValidEmail(ownEmail)) return ownEmail;
    } catch (error) {
      if (error?.status !== 404) console.warn("failed to read own easy profile for acme email", error);
    }
  }
  try {
    const sites = await ctx.api.get("/api/sites");
    for (const site of normalizeArray(sites)) {
      const candidateSiteID = String(site?.id || "").trim().toLowerCase();
      if (!candidateSiteID || candidateSiteID === siteID) continue;
      try {
        const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(candidateSiteID)}`);
        const email = normalizeEmail(profile?.front_service?.acme_account_email);
        if (isValidEmail(email)) return email;
      } catch (error) {
        if (error?.status !== 404) console.warn("failed to read easy profile for acme email", error);
      }
    }
  } catch (error) {
    console.warn("failed to list sites for acme email fallback", error);
  }
  try {
    const me = await ctx.api.get("/api/auth/me");
    const sessionEmail = normalizeEmail(me?.email || me?.user?.email);
    if (isValidEmail(sessionEmail)) return sessionEmail;
  } catch (error) {
    if (error?.status !== 401 && error?.status !== 403 && error?.status !== 404) console.warn("failed to resolve auth/me email fallback", error);
  }
  return "";
}

export async function upsertSiteResources(draft, ctx, resolveACMEAccountEmailFn, existingSite, existingUpstream, existingTLSConfig, options = {}) {
  const requestOptions = options?.requestOptions || {};
  const sitePayload = { id: draft.id.trim().toLowerCase(), primary_host: draft.primary_host.trim().toLowerCase(), enabled: draft.enabled };
  const upstreamPayload = { id: draft.upstream_id.trim().toLowerCase(), site_id: sitePayload.id, host: draft.upstream_host.trim(), port: draft.upstream_port, scheme: draft.upstream_scheme };
  const cleanupActions = [];
  const runCleanup = async () => {
    for (let index = cleanupActions.length - 1; index >= 0; index -= 1) {
      try { await cleanupActions[index](); } catch (_error) {}
    }
  };
  const rollbackable = (fn) => cleanupActions.push(fn);
  const shouldKeepStateOnApplyError = (error) => isAutoApplyFailureError(error);
  const isNotFound = (error) => error?.status === 404;
  const deleteIgnoreNotFound = async (path) => {
    try { await ctx.api.delete(path, requestOptions); } catch (error) { if (!isNotFound(error)) throw error; }
  };
  const siteExists = async (siteID) => {
    const sites = await ctx.api.get("/api/sites", requestOptions);
    return Array.isArray(sites) && sites.some((site) => normalizeSiteID(site?.id) === normalizeSiteID(siteID));
  };
  const upstreamExists = async (upstreamID) => {
    const upstreams = await ctx.api.get("/api/upstreams", requestOptions);
    return Array.isArray(upstreams) && upstreams.some((upstream) => normalizeSiteID(upstream?.id) === normalizeSiteID(upstreamID));
  };
  const tlsConfigMatches = async (siteID, certificateID) => {
    const tlsConfigs = await ctx.api.get("/api/tls-configs", requestOptions);
    if (!Array.isArray(tlsConfigs)) return false;
    return tlsConfigs.some((item) => normalizeSiteID(item?.site_id) === normalizeSiteID(siteID) && String(item?.certificate_id || "").trim().toLowerCase() === String(certificateID || "").trim().toLowerCase());
  };
  const certificateExists = async (certificateID) => {
    const certificates = await ctx.api.get("/api/certificates", requestOptions);
    return Array.isArray(certificates) && certificates.some((certificate) => String(certificate?.id || "").toLowerCase() === String(certificateID || "").toLowerCase());
  };
  try {
    if (existingSite) {
      try { await ctx.api.put(`/api/sites/${encodeURIComponent(sitePayload.id)}`, sitePayload, requestOptions); }
      catch (error) {
        if (error?.status === 404) {
          try { await ctx.api.post("/api/sites", sitePayload, requestOptions); }
          catch (postError) {
            if (!isAutoApplyFailureError(postError)) throw postError;
            const persisted = await siteExists(sitePayload.id);
            if (!persisted) throw postError;
          }
        } else if (isAutoApplyFailureError(error)) {
          const persisted = await siteExists(sitePayload.id);
          if (!persisted) throw error;
        } else throw error;
      }
    } else {
      let createdSite = false;
      try { await ctx.api.post("/api/sites", sitePayload, requestOptions); createdSite = true; }
      catch (error) {
        if (isAlreadyExistsError(error)) { await ctx.api.put(`/api/sites/${encodeURIComponent(sitePayload.id)}`, sitePayload, requestOptions); createdSite = false; }
        else if (!isAutoApplyFailureError(error)) throw error;
        else { createdSite = await siteExists(sitePayload.id); if (!createdSite) throw error; }
      }
      if (createdSite) rollbackable(async () => deleteIgnoreNotFound(`/api/sites/${encodeURIComponent(sitePayload.id)}`));
    }
    if (existingUpstream) {
      try { await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`, upstreamPayload, requestOptions); }
      catch (error) {
        if (error?.status === 404) {
          try { await ctx.api.post("/api/upstreams", upstreamPayload, requestOptions); }
          catch (postError) {
            if (!isAutoApplyFailureError(postError)) throw postError;
            const persisted = await upstreamExists(upstreamPayload.id);
            if (!persisted) throw postError;
          }
        } else if (isAutoApplyFailureError(error)) {
          const persisted = await upstreamExists(upstreamPayload.id);
          if (!persisted) throw error;
        } else throw error;
      }
    } else {
      let createdUpstream = false;
      try { await ctx.api.post("/api/upstreams", upstreamPayload, requestOptions); createdUpstream = true; }
      catch (error) {
        if (isAlreadyExistsError(error)) { await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`, upstreamPayload, requestOptions); createdUpstream = false; }
        else if (isAutoApplyFailureError(error)) { createdUpstream = await upstreamExists(upstreamPayload.id); if (!createdUpstream) throw error; }
        else throw error;
      }
      if (createdUpstream) rollbackable(async () => deleteIgnoreNotFound(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`));
    }
    if (draft.tls_enabled) {
      const certificateID = (draft.certificate_id.trim() || `${sitePayload.id}-tls`).toLowerCase();
      const hadCertificateBefore = await certificateExists(certificateID);
      const importMode = String(draft.certificate_authority_server || "").trim().toLowerCase() === "import";
      const isCurrentTLSCertificate = normalizeSiteID(existingTLSConfig?.site_id) === normalizeSiteID(sitePayload.id) && String(existingTLSConfig?.certificate_id || "").trim().toLowerCase() === certificateID;
      if (importMode && !hadCertificateBefore) throw new Error(ctx.t("sites.tls.importRequired"));
      if (!importMode && !hadCertificateBefore && !isCurrentTLSCertificate) {
        const issueEndpoint = draft.tls_self_signed ? "/api/certificates/self-signed/issue" : "/api/certificates/acme/issue";
        const acmeAccountEmail = draft.tls_self_signed ? "" : await resolveACMEAccountEmailFn(draft, ctx);
        if (!draft.tls_self_signed && acmeAccountEmail) draft.acme_account_email = acmeAccountEmail;
        try {
          await ctx.api.post(issueEndpoint, { certificate_id: certificateID, common_name: sitePayload.primary_host, san_list: [], certificate_authority_server: draft.certificate_authority_server, use_lets_encrypt_staging: Boolean(draft.use_lets_encrypt_staging), account_email: acmeAccountEmail }, requestOptions);
        } catch (error) { if (!isAlreadyExistsError(error)) throw error; }
      }
      if (!hadCertificateBefore && !importMode) rollbackable(async () => deleteIgnoreNotFound(`/api/certificates/${encodeURIComponent(certificateID)}`));
      const tlsPayload = { site_id: sitePayload.id, certificate_id: certificateID };
      if (existingTLSConfig) {
        try { await ctx.api.put(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`, tlsPayload, requestOptions); }
        catch (error) {
          if (error?.status === 404) {
            try { await ctx.api.post("/api/tls-configs", tlsPayload, requestOptions); }
            catch (postError) {
              if (!isAutoApplyFailureError(postError)) throw postError;
              const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
              if (!persisted) throw postError;
            }
          } else if (isAutoApplyFailureError(error)) {
            const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
            if (!persisted) throw error;
          } else throw error;
        }
      } else {
        try { await ctx.api.post("/api/tls-configs", tlsPayload, requestOptions); }
        catch (error) {
          const message = String(error?.message || "").toLowerCase();
          const hasConflict = error?.status === 409 || message.includes("already exists");
          if (hasConflict) await ctx.api.put(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`, tlsPayload, requestOptions);
          else if (isAutoApplyFailureError(error)) {
            const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
            if (!persisted) throw error;
          } else throw error;
        }
        rollbackable(async () => deleteIgnoreNotFound(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`));
      }
    }
  } catch (error) {
    if (!shouldKeepStateOnApplyError(error)) await runCleanup();
    throw error;
  }
}
