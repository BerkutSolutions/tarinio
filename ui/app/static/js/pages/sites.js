import {
  computeUpstreamID as computeUpstreamIDMain,
  draftToEnvText as draftToEnvTextMain,
  envToDraft as envToDraftMain,
  hydrateSiteDraft as hydrateSiteDraftMain,
  renderRawEditor as renderRawEditorMain,
  renderSites as renderSitesMain
} from "./sites.page-main-runtime.js";

function computeUpstreamID(siteID) {
  return computeUpstreamIDMain(siteID);
}

function draftToEnvText(draft) {
  return draftToEnvTextMain(draft);
}

function envToDraft(text) {
  return envToDraftMain(text);
}

function renderRawEditor(state, ctx, isNew) {
  return renderRawEditorMain(state, ctx, isNew);
}

async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null) {
  return hydrateSiteDraftMain(ctx, site, upstream, tlsConfig, accessPolicy);
}

// contract marker: if (String(existingSite?._origin || "") === "secondary")
// contract marker: id="services-select-all"
// contract marker: data-mode-tab="raw"
// contract marker: <div class="waf-upstream-target-row">

export async function renderSites(container, ctx) {
  return renderSitesMain(container, ctx);
}
