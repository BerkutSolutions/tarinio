export function rebuildIndexes(state, deps = {}) {
  const { normalizeHost, normalizeSiteID } = deps;
  state.upstreamsBySite = new Map();
  state.tlsBySite = new Map();
  state.certificateBySiteID = new Map();
  state.certificateByHost = new Map();
  state.accessBySite = new Map();
  state.easyProfilesBySite = new Map();
  for (const upstream of state.upstreams) {
    const siteID = normalizeSiteID(upstream?.site_id);
    if (!siteID) {
      continue;
    }
    const items = state.upstreamsBySite.get(siteID) || [];
    items.push(upstream);
    state.upstreamsBySite.set(siteID, items);
  }
  for (const tlsConfig of state.tlsConfigs) {
    const siteID = normalizeSiteID(tlsConfig?.site_id);
    if (!siteID || state.tlsBySite.has(siteID)) {
      continue;
    }
    state.tlsBySite.set(siteID, tlsConfig);
  }
  for (const certificate of state.certificates) {
    const certificateID = String(certificate?.id || "").trim().toLowerCase();
    if (certificateID.endsWith("-tls")) {
      const relatedSiteID = certificateID.slice(0, -4);
      if (relatedSiteID && !state.certificateBySiteID.has(relatedSiteID)) {
        state.certificateBySiteID.set(relatedSiteID, certificate);
      }
    }
    const host = normalizeHost(certificate?.common_name);
    if (host && !state.certificateByHost.has(host)) {
      state.certificateByHost.set(host, certificate);
    }
  }
  for (const accessPolicy of state.accessPolicies) {
    const siteID = normalizeSiteID(accessPolicy?.site_id);
    if (!siteID) {
      continue;
    }
    const existing = state.accessBySite.get(siteID);
    const isCompatibilityPolicy = String(accessPolicy?.id || "").startsWith(`easy-${siteID}-`);
    const existingIsCompatibility = String(existing?.id || "").startsWith(`easy-${siteID}-`);
    if (!existing || (existingIsCompatibility && !isCompatibilityPolicy)) {
      state.accessBySite.set(siteID, accessPolicy);
    }
  }
  for (const easyProfile of state.easyProfiles) {
    const siteID = normalizeSiteID(easyProfile?.site_id);
    if (!siteID || state.easyProfilesBySite.has(siteID)) {
      continue;
    }
    state.easyProfilesBySite.set(siteID, easyProfile);
  }
}

export function applyFilters(state, deps = {}) {
  const { normalizeSiteID, normalizeServiceProfile } = deps;
  const search = state.search.trim().toLowerCase();
  const sites = state.sites.filter((site) => {
    if (!search) {
      return true;
    }
    const profile = state.easyProfilesBySite.get(normalizeSiteID(site.id));
    const profileValue = normalizeServiceProfile(profile?.front_service?.profile);
    return `${site.id} ${site.primary_host} ${profileValue}`.toLowerCase().includes(search);
  });
  sites.sort((left, right) => {
    if (state.sort === "name-asc") {
      return String(left.primary_host || left.id).localeCompare(String(right.primary_host || right.id));
    }
    if (state.sort === "name-desc") {
      return String(right.primary_host || right.id).localeCompare(String(left.primary_host || left.id));
    }
    if (state.sort === "created-desc") {
      return String(right.created_at || "").localeCompare(String(left.created_at || ""));
    }
    return String(right.updated_at || right.created_at || "").localeCompare(String(left.updated_at || left.created_at || ""));
  });
  state.filteredSites = sites;
}

export async function syncDraftFromRoute(state, ctx, deps = {}) {
  const {
    pendingImportedDraftRef,
    ensureControlPlaneAccessManagementMethods,
    draftToEnvText,
    normalizeArray,
    defaultSiteDraft,
    normalizeSiteID,
    hydrateSiteDraft
  } = deps;
  if (state.route.mode === "list") {
    state.settingsSearch = "";
    state.settingsMatches = [];
    state.highlightedSelector = "";
    return;
  }
  if (state.route.mode === "create") {
    if (pendingImportedDraftRef.current && pendingImportedDraftRef.current.draft) {
      state.draft = ensureControlPlaneAccessManagementMethods({ ...pendingImportedDraftRef.current.draft });
      state.rawEnvText = String(pendingImportedDraftRef.current.rawEnvText || draftToEnvText(state.draft));
      state.rawMissingFields = normalizeArray(pendingImportedDraftRef.current.missingFields);
      pendingImportedDraftRef.current = null;
    } else {
      state.draft = defaultSiteDraft();
      state.rawEnvText = draftToEnvText(state.draft);
      state.rawMissingFields = [];
    }
    state.editorMode = "easy";
    state.listTemplateSelection.blacklist_user_agent = [];
    state.listTemplateSelection.blacklist_uri = [];
    state.listTemplateSelection.blacklist_ja3 = [];
    state.activeTab = "front";
    state.settingsSearch = "";
    state.settingsMatches = [];
    state.highlightedSelector = "";
    return;
  }
  const normalizedRouteSiteID = normalizeSiteID(state.route.siteID);
  const site = state.sites.find((item) => normalizeSiteID(item?.id) === normalizedRouteSiteID) || null;
  const upstream = state.upstreamsBySite.get(normalizedRouteSiteID)?.[0] || null;
  const tlsConfig = state.tlsBySite.get(normalizedRouteSiteID) || null;
  const accessPolicy = state.accessBySite.get(normalizedRouteSiteID) || null;
  state.draft = await hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy);
  state.listTemplateSelection.blacklist_user_agent = [];
  state.listTemplateSelection.blacklist_uri = [];
  state.listTemplateSelection.blacklist_ja3 = [];
  state.rawEnvText = draftToEnvText(state.draft);
  state.rawMissingFields = [];
  state.editorMode = "easy";
  state.activeTab = "front";
  state.settingsSearch = "";
  state.settingsMatches = [];
  state.highlightedSelector = "";
}
