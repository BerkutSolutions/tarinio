import { setLoading } from "../ui.js";
import {
  go,
  mergeByID,
  mergeBySiteID,
  mergeProfilesBySite,
  normalizeArray,
  normalizeSiteID,
  notifyExpiringCertificates,
  routeInfo,
  tryGetJSON,
  unwrapList,
} from "./sites.routing-merge.js";
import {
  normalizeHost,
  normalizeServiceProfile,
} from "./sites.normalize.js";
import {
  buildGeoCatalogFallback,
  normalizeGeoCatalogPayload,
} from "./sites.geo-lists.js";
import { bindList as bindListModule } from "./sites.list-view.js";
import {
  defaultSiteDraft,
  bindStableDetail,
} from "./sites.stable-detail-bind.js";
import {
  downloadJSON,
  draftToEnvText,
  ensureControlPlaneAccessManagementMethods,
  exportSelectedServicesEnv,
  hydrateSiteDraft,
  importServicesFiles,
  putWithPostFallback,
  toEnvKey,
} from "./sites.stable-resources.js";
import {
  renderDetailView,
  renderListView,
} from "./sites.stable-renderers.js";
import {
  installSitesGlobalErrorLogging,
  logSitesError,
  renderSitesErrorAlert,
} from "./sites.stable-errors.js";

let pendingImportedDraft = null;

export async function renderSites(container, ctx) {
  installSitesGlobalErrorLogging();
  const route = routeInfo();
  const state = {
    route,
    sites: [],
    upstreams: [],
    tlsConfigs: [],
    certificates: [],
    accessPolicies: [],
    easyProfiles: [],
    geoCatalog: buildGeoCatalogFallback(),
    search: "",
    sort: "updated-desc",
    editorMode: "easy",
    activeTab: "front",
    settingsSearch: "",
    settingsMatches: [],
    highlightedSelector: "",
    rawEnvText: "",
    rawMissingFields: [],
    filteredSites: [],
    selectedSiteIDs: new Set(),
    upstreamsBySite: new Map(),
    tlsBySite: new Map(),
    certificateBySiteID: new Map(),
    certificateByHost: new Map(),
    accessBySite: new Map(),
    easyProfilesBySite: new Map(),
    pendingImportedDraftRef: {
      set(imported) {
        pendingImportedDraft = imported
          ? {
              draft: imported.draft,
              missingFields: imported.missingFields,
              rawEnvText: imported.rawEnvText,
            }
          : null;
      },
    },
    countryFilters: {
      blacklist_country: "",
      whitelist_country: ""
    },
    listTemplateSelection: {
      blacklist_user_agent: [],
      blacklist_uri: []
    },
    draft: defaultSiteDraft()
  };

  const rebuildIndexes = () => {
    state.upstreamsBySite = new Map();
    state.tlsBySite = new Map();
    state.certificateBySiteID = new Map();
    state.certificateByHost = new Map();
    state.accessBySite = new Map();
    state.easyProfilesBySite = new Map();
    for (const upstream of state.upstreams) {
      const items = state.upstreamsBySite.get(upstream.site_id) || [];
      items.push(upstream);
      state.upstreamsBySite.set(upstream.site_id, items);
    }
    for (const tlsConfig of state.tlsConfigs) {
      state.tlsBySite.set(tlsConfig.site_id, tlsConfig);
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
      if (!siteID || state.accessBySite.has(siteID)) {
        continue;
      }
      state.accessBySite.set(siteID, accessPolicy);
    }
    for (const easyProfile of state.easyProfiles) {
      const siteID = normalizeSiteID(easyProfile?.site_id);
      if (!siteID || state.easyProfilesBySite.has(siteID)) {
        continue;
      }
      state.easyProfilesBySite.set(siteID, easyProfile);
    }
  };

  const applyFilters = () => {
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
  };

  const syncDraftFromRoute = async () => {
    if (state.route.mode === "list") {
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    if (state.route.mode === "create") {
      if (pendingImportedDraft && pendingImportedDraft.draft) {
        state.draft = ensureControlPlaneAccessManagementMethods({ ...pendingImportedDraft.draft });
        state.rawEnvText = String(pendingImportedDraft.rawEnvText || draftToEnvText(state.draft));
        state.rawMissingFields = normalizeArray(pendingImportedDraft.missingFields);
        pendingImportedDraft = null;
      } else {
        state.draft = defaultSiteDraft();
        state.rawEnvText = draftToEnvText(state.draft);
        state.rawMissingFields = [];
      }
      state.editorMode = "easy";
      state.listTemplateSelection.blacklist_user_agent = [];
      state.listTemplateSelection.blacklist_uri = [];
      state.activeTab = "front";
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    const site = state.sites.find((item) => item.id === state.route.siteID);
    const upstream = state.upstreamsBySite.get(state.route.siteID)?.[0] || null;
    const tlsConfig = state.tlsBySite.get(state.route.siteID) || null;
    const accessPolicy = state.accessBySite.get(normalizeSiteID(state.route.siteID)) || null;
    state.draft = await hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy);
    state.listTemplateSelection.blacklist_user_agent = [];
    state.listTemplateSelection.blacklist_uri = [];
    state.rawEnvText = draftToEnvText(state.draft);
    state.rawMissingFields = [];
    state.editorMode = "easy";
    state.activeTab = "front";
    state.settingsSearch = "";
    state.settingsMatches = [];
    state.highlightedSelector = "";
  };

  const render = () => {
    try {
      applyFilters();
      container.innerHTML = state.route.mode === "list" ? renderListView(state, ctx) : renderDetailView(state, ctx);
      bind();
    } catch (error) {
      const payload = logSitesError("render", error, {
        routeMode: state.route?.mode || "",
        siteID: state.route?.siteID || "",
      });
      container.innerHTML = renderSitesErrorAlert(ctx, payload);
    }
  };

  const load = async () => {
    try {
      setLoading(container, ctx.t("sites.loading"));
      const [sitesResponse, upstreamsResponse, tlsConfigsResponse, certificatesResponse, accessPoliciesResponse, easyProfilesResponse, geoCatalogResponse] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/upstreams"),
        ctx.api.get("/api/tls-configs"),
        ctx.api.get("/api/certificates").catch(() => []),
        ctx.api.get("/api/access-policies").catch(() => []),
        ctx.api.get("/api/easy-site-profiles").catch(() => []),
        ctx.api.get("/api/easy-site-profiles/catalog/countries").catch(() => null)
      ]);
      const [secondarySites, secondaryUpstreams, secondaryTLSConfigs, secondaryCertificates, secondaryEasyProfiles] = await Promise.all([
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/upstreams"),
        tryGetJSON("/api-app/tls-configs"),
        tryGetJSON("/api-app/certificates"),
        tryGetJSON("/api-app/easy-site-profiles")
      ]);
      state.sites = mergeByID(sitesResponse, unwrapList(secondarySites, ["sites"]));
      state.upstreams = mergeByID(upstreamsResponse, unwrapList(secondaryUpstreams, ["upstreams"]));
      state.tlsConfigs = mergeBySiteID(tlsConfigsResponse, unwrapList(secondaryTLSConfigs, ["tls_configs", "tlsConfigs"]));
      state.certificates = mergeByID(certificatesResponse, unwrapList(secondaryCertificates, ["certificates"]));
      notifyExpiringCertificates(ctx, state.certificates);
      state.accessPolicies = normalizeArray(accessPoliciesResponse);
      state.easyProfiles = mergeProfilesBySite(easyProfilesResponse, secondaryEasyProfiles);
      state.selectedSiteIDs = new Set(Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => site.id === id)));
      state.geoCatalog = normalizeGeoCatalogPayload(geoCatalogResponse);
      rebuildIndexes();
      await syncDraftFromRoute();
      render();
    } catch (error) {
      const payload = logSitesError("load", error, {
        routeMode: state.route?.mode || "",
        siteID: state.route?.siteID || "",
      });
      container.innerHTML = renderSitesErrorAlert(ctx, payload);
    }
  };

  const bindList = () => {
    bindListModule(container, state, ctx, {
      go,
      load,
      downloadJSON,
      exportSelectedServicesEnv,
      importServicesFiles,
      toEnvKey,
      putWithPostFallback,
      normalizeSiteID,
      render,
    });
  };

  const bind = () => {
    if (state.route.mode === "list") {
      bindList();
      return;
    }
    bindStableDetail(container, state, ctx, { load, render });
  };

  await load();
}
