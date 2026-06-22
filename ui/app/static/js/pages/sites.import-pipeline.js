import { normalizeArray, normalizeSiteID } from "./sites.routing-merge.js";
import { applyServiceProfilePresetForMissingFields } from "./sites.normalize.js";
import { computeUpstreamID } from "./sites.traffic-helpers.js";
import { defaultSiteDraft, draftToEasyProfile } from "./sites.draft-core.js";

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
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "0";
  return String(value ?? "");
}

export function draftToEnvText(draft) {
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
  if (["1", "true", "yes", "on"].includes(normalized)) return true;
  if (["0", "false", "no", "off"].includes(normalized)) return false;
  throw new Error(`invalid boolean value: ${value}`);
}

function parseNumberEnv(value) {
  const num = Number(String(value || "").trim());
  if (!Number.isFinite(num)) throw new Error(`invalid number value: ${value}`);
  return num;
}

function parseArrayEnv(value) {
  const trimmed = String(value || "").trim();
  if (!trimmed) return [];
  if (trimmed.startsWith("[")) {
    const parsed = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) throw new Error(`invalid array value: ${value}`);
    return parsed;
  }
  return trimmed.split(",").map((item) => item.trim()).filter(Boolean);
}

function userPermissionsSet(ctx) {
  const items = Array.isArray(ctx?.currentUser?.permissions) ? ctx.currentUser.permissions : [];
  return new Set(items.map((item) => String(item || "").trim().toLowerCase()).filter(Boolean));
}

export function requirePermissions(ctx, requiredPermissions, errorKey) {
  const required = Array.isArray(requiredPermissions) ? requiredPermissions : [];
  if (!required.length) return;
  const granted = userPermissionsSet(ctx);
  const missing = required.map((item) => String(item || "").trim().toLowerCase()).filter((item) => item && !granted.has(item));
  if (!missing.length) return;
  const permissions = missing.join(", ");
  throw new Error(ctx.t(errorKey, { permissions }));
}

export function envToDraft(text) {
  const baseDraft = defaultSiteDraft();
  const knownFields = new Set(Object.keys(baseDraft));
  const envToField = new Map(Object.keys(baseDraft).map((field) => [toEnvKey(field), field]));
  const parsed = {};
  const presentFields = new Set();
  const lines = String(text || "").split(/\r?\n/);
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const eq = line.indexOf("=");
    if (eq <= 0) throw new Error(`invalid env line: ${line}`);
    const key = line.slice(0, eq).trim();
    const rawValue = line.slice(eq + 1);
    const field = envToField.get(key);
    if (!field || !knownFields.has(field)) throw new Error(`unknown env parameter: ${key}`);
    const template = baseDraft[field];
    if (Array.isArray(template)) parsed[field] = parseArrayEnv(rawValue);
    else if (typeof template === "boolean") parsed[field] = parseBooleanEnv(rawValue);
    else if (typeof template === "number") parsed[field] = parseNumberEnv(rawValue);
    else parsed[field] = String(rawValue ?? "");
    presentFields.add(field);
  }
  const missingFields = Object.keys(baseDraft).filter((field) => !presentFields.has(field));
  return { draft: { ...baseDraft, ...parsed }, missingFields };
}

function diffObjects(previous, next, path = "") {
  const left = previous && typeof previous === "object" ? previous : {};
  const right = next && typeof next === "object" ? next : {};
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);
  const lines = [];
  for (const key of keys) {
    const currentPath = path ? `${path}.${key}` : key;
    const l = left[key];
    const r = right[key];
    const bothObjects = l && r && typeof l === "object" && typeof r === "object" && !Array.isArray(l) && !Array.isArray(r);
    if (bothObjects) lines.push(...diffObjects(l, r, currentPath));
    else if (JSON.stringify(l) !== JSON.stringify(r)) lines.push(`${currentPath}: ${JSON.stringify(l)} -> ${JSON.stringify(r)}`);
  }
  return lines;
}

const importNoAutoApplyOptions = { headers: { "X-WAF-Auto-Apply-Disabled": "1" } };

function buildImportInventory(resources = {}) {
  const sitesByID = new Map();
  const upstreamsByID = new Map();
  const tlsConfigsBySiteID = new Map();
  for (const item of normalizeArray(resources.sites)) {
    const id = normalizeSiteID(item?.id);
    if (id) sitesByID.set(id, item);
  }
  for (const item of normalizeArray(resources.upstreams)) {
    const id = normalizeSiteID(item?.id);
    if (id) upstreamsByID.set(id, item);
  }
  for (const item of normalizeArray(resources.tlsConfigs)) {
    const siteID = normalizeSiteID(item?.site_id);
    if (siteID) tlsConfigsBySiteID.set(siteID, item);
  }
  return { sitesByID, upstreamsByID, tlsConfigsBySiteID };
}

async function loadImportInventory(ctx) {
  const [sites, upstreams, tlsConfigs] = await Promise.all([
    ctx.api.get("/api/sites").catch(() => []),
    ctx.api.get("/api/upstreams").catch(() => []),
    ctx.api.get("/api/tls-configs").catch(() => [])
  ]);
  return buildImportInventory({ sites, upstreams, tlsConfigs });
}

function buildImportPayloadFromDraft(draft, ensureControlPlaneAccessManagementMethods) {
  const normalized = ensureControlPlaneAccessManagementMethods({ ...draft });
  const site = { id: normalized.id.trim().toLowerCase(), primary_host: normalized.primary_host.trim().toLowerCase(), enabled: Boolean(normalized.enabled) };
  const upstream = { id: computeUpstreamID(site.id), site_id: site.id, host: normalized.upstream_host.trim(), port: Number(normalized.upstream_port || 80), scheme: String(normalized.upstream_scheme || "http").toLowerCase() };
  const tls = normalized.tls_enabled ? { site_id: site.id, certificate_id: (normalized.certificate_id.trim() || `${site.id}-tls`).toLowerCase() } : null;
  return { draft: { ...normalized, id: site.id, upstream_id: upstream.id }, site, upstream, tls, easyProfile: draftToEasyProfile({ ...normalized, id: site.id, upstream_id: upstream.id }) };
}

async function compileAndApplyImportedRevision(ctx) {
  const compileResponse = await ctx.api.post("/api/revisions/compile", {}, importNoAutoApplyOptions);
  const revisionID = String(compileResponse?.revision?.id || "").trim();
  if (!revisionID) throw new Error("Import compile failed");
  const applyResponse = await ctx.api.post(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, {});
  if (String(applyResponse?.status || "").trim().toLowerCase() === "failed") throw new Error(String(applyResponse?.result || "").trim() || "Import apply failed");
  return revisionID;
}

async function applyImportPayload(ctx, payload, upsertSiteResources, putWithPostFallback, inventory = null) {
  const { draft, site, upstream, tls, easyProfile } = payload;
  const existingSite = inventory?.sitesByID?.get(normalizeSiteID(site.id)) || null;
  const existingUpstream = inventory?.upstreamsByID?.get(normalizeSiteID(upstream.id)) || null;
  const existingTLSConfig = inventory?.tlsConfigsBySiteID?.get(normalizeSiteID(site.id)) || null;
  const existingEasy = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
  await upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, { requestOptions: importNoAutoApplyOptions });
  await putWithPostFallback(ctx, `/api/easy-site-profiles/${encodeURIComponent(site.id)}`, easyProfile, { requestOptions: importNoAutoApplyOptions });
  if (inventory) {
    inventory.sitesByID.set(normalizeSiteID(site.id), site);
    inventory.upstreamsByID.set(normalizeSiteID(upstream.id), upstream);
    if (tls) inventory.tlsConfigsBySiteID.set(normalizeSiteID(site.id), tls); else inventory.tlsConfigsBySiteID.delete(normalizeSiteID(site.id));
  }
  const diffLines = [];
  if (existingSite) diffLines.push(...diffObjects(existingSite, site, "site"));
  if (existingUpstream) diffLines.push(...diffObjects(existingUpstream, upstream, "upstream"));
  if (tls && existingTLSConfig) diffLines.push(...diffObjects(existingTLSConfig, tls, "tls"));
  if (existingEasy) diffLines.push(...diffObjects(existingEasy, easyProfile, "easy"));
  return { updatedExisting: Boolean(existingSite), diffLines };
}

async function importServicesJSON(file, ctx) {
  requirePermissions(ctx, ["sites.write", "upstreams.write"], "sites.error.importJsonPermissions");
  const payload = JSON.parse(await file.text());
  const sites = normalizeArray(payload.sites);
  const upstreams = normalizeArray(payload.upstreams);
  for (const site of sites) {
    try { await ctx.api.post("/api/sites", site, importNoAutoApplyOptions); }
    catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) await ctx.api.put(`/api/sites/${encodeURIComponent(site.id)}`, site, importNoAutoApplyOptions); else throw error;
    }
  }
  for (const upstream of upstreams) {
    try { await ctx.api.post("/api/upstreams", upstream, importNoAutoApplyOptions); }
    catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstream.id)}`, upstream, importNoAutoApplyOptions); else throw error;
    }
  }
  return { imported: sites.length };
}

export async function importServicesFiles(files, ctx, ensureControlPlaneAccessManagementMethods, validateDraft, upsertSiteResources, putWithPostFallback) {
  if (!Array.isArray(files) || files.length !== 1) throw new Error(ctx.t("sites.error.importSingleEnv"));
  const file = files[0];
  const name = String(file?.name || "").toLowerCase();
  if (!name.endsWith(".env")) throw new Error(`${file?.name || "file"}: ${ctx.t("sites.error.importEnvOnly")}`);
  requirePermissions(ctx, ["sites.write", "upstreams.write", "tls.write", "certificates.write"], "sites.error.importEnvPermissions");
  const text = await file.text();
  const { draft, missingFields } = envToDraft(text);
  const draftWithPreset = applyServiceProfilePresetForMissingFields(draft, missingFields);
  const validationError = validateDraft(draftWithPreset, ctx);
  if (validationError) throw new Error(`${file.name}: ${validationError}`);
  const inventory = await loadImportInventory(ctx);
  const payload = buildImportPayloadFromDraft(draftWithPreset, ensureControlPlaneAccessManagementMethods);
  await applyImportPayload(ctx, payload, upsertSiteResources, putWithPostFallback, inventory);
  await compileAndApplyImportedRevision(ctx);
  return { file: file.name, draft: payload.draft, missingFields, rawEnvText: text };
}

// marker for UI contract search compatibility
async function importServicesJSONMarker(file, ctx) { return importServicesJSON(file, ctx); }
