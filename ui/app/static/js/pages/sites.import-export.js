const importNoAutoApplyOptions = {
  headers: {
    "X-WAF-Auto-Apply-Disabled": "1"
  }
};

export function downloadJSON(filename, payload) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function downloadText(filename, content, type = "text/plain;charset=utf-8") {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function downloadBlob(filename, blob) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function toEnvKey(field) {
  return `WAF_SITE_${String(field || "").toUpperCase()}`;
}

function normalizeEnvPrimitive(value) {
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? String(value) : "0";
  }
  return String(value ?? "");
}

export function draftToEnvText(draft, defaultSiteDraft) {
  const baseDraft = defaultSiteDraft();
  const lines = [];
  for (const field of Object.keys(baseDraft)) {
    const value = draft?.[field];
    const rendered = Array.isArray(value) ? JSON.stringify(value) : normalizeEnvPrimitive(value);
    lines.push(`${toEnvKey(field)}=${rendered}`);
  }
  return `${lines.join("\n")}\n`;
}

function parseBooleanEnv(value) {
  const normalized = String(value || "").trim().toLowerCase();
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  throw new Error(`invalid boolean value: ${value}`);
}

function parseNumberEnv(value) {
  const num = Number(String(value || "").trim());
  if (!Number.isFinite(num)) {
    throw new Error(`invalid number value: ${value}`);
  }
  return num;
}

function parseArrayEnv(value) {
  const trimmed = String(value || "").trim();
  if (!trimmed) {
    return [];
  }
  if (trimmed.startsWith("[")) {
    const parsed = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) {
      throw new Error(`invalid array value: ${value}`);
    }
    return parsed;
  }
  return trimmed
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function envToDraft(text, defaultSiteDraft) {
  const baseDraft = defaultSiteDraft();
  const knownFields = new Set(Object.keys(baseDraft));
  const envToField = new Map(Object.keys(baseDraft).map((field) => [toEnvKey(field), field]));
  const parsed = {};
  const presentFields = new Set();

  const lines = String(text || "").split(/\r?\n/);
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) {
      continue;
    }
    const eq = line.indexOf("=");
    if (eq <= 0) {
      throw new Error(`invalid env line: ${line}`);
    }
    const key = line.slice(0, eq).trim();
    const rawValue = line.slice(eq + 1);
    const field = envToField.get(key);
    if (!field || !knownFields.has(field)) {
      throw new Error(`unknown env parameter: ${key}`);
    }
    const template = baseDraft[field];
    if (Array.isArray(template)) {
      parsed[field] = parseArrayEnv(rawValue);
    } else if (typeof template === "boolean") {
      parsed[field] = parseBooleanEnv(rawValue);
    } else if (typeof template === "number") {
      parsed[field] = parseNumberEnv(rawValue);
    } else {
      parsed[field] = String(rawValue ?? "");
    }
    presentFields.add(field);
  }

  const missingFields = Object.keys(baseDraft).filter((field) => !presentFields.has(field));
  return {
    draft: { ...baseDraft, ...parsed },
    missingFields
  };
}

export function buildImportPayloadFromDraft(draft, deps) {
  const normalized = deps.ensureControlPlaneAccessManagementMethods({ ...draft });
  const site = {
    id: normalized.id.trim().toLowerCase(),
    primary_host: normalized.primary_host.trim().toLowerCase(),
    enabled: Boolean(normalized.enabled)
  };
  const upstream = {
    id: deps.computeUpstreamID(site.id),
    site_id: site.id,
    host: normalized.upstream_host.trim(),
    port: Number(normalized.upstream_port || 80),
    scheme: String(normalized.upstream_scheme || "http").toLowerCase()
  };
  const tls = normalized.tls_enabled
    ? {
        site_id: site.id,
        certificate_id: (normalized.certificate_id.trim() || `${site.id}-tls`).toLowerCase()
      }
    : null;
  return {
    draft: { ...normalized, id: site.id, upstream_id: upstream.id },
    site,
    upstream,
    tls,
    easyProfile: deps.draftToEasyProfile({ ...normalized, id: site.id, upstream_id: upstream.id })
  };
}

export function diffObjects(previous, next, path = "") {
  const left = previous && typeof previous === "object" ? previous : {};
  const right = next && typeof next === "object" ? next : {};
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);
  const lines = [];
  for (const key of keys) {
    const currentPath = path ? `${path}.${key}` : key;
    const l = left[key];
    const r = right[key];
    const bothObjects = l && r && typeof l === "object" && typeof r === "object" && !Array.isArray(l) && !Array.isArray(r);
    if (bothObjects) {
      lines.push(...diffObjects(l, r, currentPath));
      continue;
    }
    if (JSON.stringify(l) !== JSON.stringify(r)) {
      lines.push(`${currentPath}: ${JSON.stringify(l)} -> ${JSON.stringify(r)}`);
    }
  }
  return lines;
}

export function buildImportInventory(resources = {}, deps) {
  const sitesByID = new Map();
  const upstreamsByID = new Map();
  const tlsConfigsBySiteID = new Map();
  for (const item of deps.normalizeArray(resources.sites)) {
    const id = deps.normalizeSiteID(item?.id);
    if (id) {
      sitesByID.set(id, item);
    }
  }
  for (const item of deps.normalizeArray(resources.upstreams)) {
    const id = deps.normalizeSiteID(item?.id);
    if (id) {
      upstreamsByID.set(id, item);
    }
  }
  for (const item of deps.normalizeArray(resources.tlsConfigs)) {
    const siteID = deps.normalizeSiteID(item?.site_id);
    if (siteID) {
      tlsConfigsBySiteID.set(siteID, item);
    }
  }
  return { sitesByID, upstreamsByID, tlsConfigsBySiteID };
}

export async function loadImportInventory(ctx, buildImportInventoryFn) {
  const [sites, upstreams, tlsConfigs] = await Promise.all([
    ctx.api.get("/api/sites").catch(() => []),
    ctx.api.get("/api/upstreams").catch(() => []),
    ctx.api.get("/api/tls-configs").catch(() => [])
  ]);
  return buildImportInventoryFn({ sites, upstreams, tlsConfigs });
}

export async function compileAndApplyImportedRevision(ctx) {
  const compileResponse = await ctx.api.post("/api/revisions/compile", {}, importNoAutoApplyOptions);
  const revisionID = String(compileResponse?.revision?.id || "").trim();
  if (!revisionID) {
    throw new Error("Import compile failed");
  }
  const applyResponse = await ctx.api.post(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, {});
  if (String(applyResponse?.status || "").trim().toLowerCase() === "failed") {
    throw new Error(String(applyResponse?.result || "").trim() || "Import apply failed");
  }
  return revisionID;
}

export async function applyImportPayload(ctx, payload, inventory = null, deps) {
  const { draft, site, upstream, tls, easyProfile } = payload;
  const existingSite = inventory?.sitesByID?.get(deps.normalizeSiteID(site.id)) || null;
  const existingUpstream = inventory?.upstreamsByID?.get(deps.normalizeSiteID(upstream.id)) || null;
  const existingTLSConfig = inventory?.tlsConfigsBySiteID?.get(deps.normalizeSiteID(site.id)) || null;
  const existingEasy = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));

  await deps.upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, { requestOptions: importNoAutoApplyOptions });
  await deps.putWithPostFallback(ctx, `/api/easy-site-profiles/${encodeURIComponent(site.id)}`, easyProfile, { requestOptions: importNoAutoApplyOptions });
  if (inventory) {
    inventory.sitesByID.set(deps.normalizeSiteID(site.id), site);
    inventory.upstreamsByID.set(deps.normalizeSiteID(upstream.id), upstream);
    if (tls) {
      inventory.tlsConfigsBySiteID.set(deps.normalizeSiteID(site.id), tls);
    } else {
      inventory.tlsConfigsBySiteID.delete(deps.normalizeSiteID(site.id));
    }
  }

  const diffLines = [];
  if (existingSite) {
    diffLines.push(...diffObjects(existingSite, site, "site"));
  }
  if (existingUpstream) {
    diffLines.push(...diffObjects(existingUpstream, upstream, "upstream"));
  }
  if (tls && existingTLSConfig) {
    diffLines.push(...diffObjects(existingTLSConfig, tls, "tls"));
  }
  if (existingEasy) {
    diffLines.push(...diffObjects(existingEasy, easyProfile, "easy"));
  }
  return {
    updatedExisting: Boolean(existingSite),
    diffLines
  };
}

export async function importServicesJSON(file, ctx, deps) {
  deps.requirePermissions(ctx, ["sites.write", "upstreams.write"], "sites.error.importJsonPermissions");
  const payload = JSON.parse(await file.text());
  const sites = deps.normalizeArray(payload.sites);
  const upstreams = deps.normalizeArray(payload.upstreams);
  for (const site of sites) {
    try {
      await ctx.api.post("/api/sites", site, importNoAutoApplyOptions);
    } catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) {
        await ctx.api.put(`/api/sites/${encodeURIComponent(site.id)}`, site, importNoAutoApplyOptions);
      } else {
        throw error;
      }
    }
  }
  for (const upstream of upstreams) {
    try {
      await ctx.api.post("/api/upstreams", upstream, importNoAutoApplyOptions);
    } catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) {
        await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstream.id)}`, upstream, importNoAutoApplyOptions);
      } else {
        throw error;
      }
    }
  }
  return { imported: sites.length };
}
