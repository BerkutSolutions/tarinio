// Legacy-broken compatibility path.
// Do not wire this module into app navigation until its runtime bridge reaches
// feature parity with the stable Services renderer in `sites.js`.
import { renderSitesRuntime } from "./sites.page-render-runtime.js";
import {
  buildRenderSitesDeps,
  computeUpstreamID,
  draftToEnvText,
  envToDraft,
  hydrateSiteDraft,
  renderRawEditor
} from "./sites.page-main-helpers.js";

export async function renderSites(container, ctx) {
  return renderSitesRuntime(container, ctx, buildRenderSitesDeps());
}

export {
  computeUpstreamID,
  draftToEnvText,
  envToDraft,
  hydrateSiteDraft,
  renderRawEditor
};
