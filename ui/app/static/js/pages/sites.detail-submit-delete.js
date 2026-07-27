import { setError, setLoading } from "../ui.js";
import { normalizeSiteID, routeBase } from "./sites.routing-merge.js";
import { compileAndApplySiteRevision, siteSaveNoAutoApplyOptions } from "./sites.save-apply.js";

export function bindDetailSubmitDelete(container, state, ctx, deps) {
  const {
    parseRawDraft,
    getDraft,
    syncStateDraftFromForm,
    ensureControlPlaneAccessManagementMethods,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    putWithPostFallback,
    draftToEasyProfile,
    go,
    deleteServiceWithResources,
    clearUnsavedChanges,
  } = deps;
  const feedback = container.querySelector("#sites-feedback");
  const back = () => go(routeBase());

  container.querySelector("#service-editor-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    let draft;
    try {
      if (state.editorMode === "raw") {
        draft = ensureControlPlaneAccessManagementMethods(parseRawDraft());
      } else {
        syncStateDraftFromForm();
        draft = ensureControlPlaneAccessManagementMethods(getDraft());
      }
    } catch (error) {
      setError(feedback, `${ctx.t("sites.raw.parseError")}: ${String(error?.message || error)}`);
      return;
    }
    const validationError = validateDraft(draft, ctx);
    if (validationError) {
      setError(feedback, validationError);
      return;
    }
    try {
      setLoading(feedback, ctx.t("sites.editor.saving"));
      const saveOptions = { requestOptions: siteSaveNoAutoApplyOptions };
      const existingSite = state.sites.find((item) => normalizeSiteID(item?.id) === normalizeSiteID(state.route.siteID) || normalizeSiteID(item?.id) === normalizeSiteID(draft.id));
      const existingUpstream = state.upstreams.find((item) => item.id === draft.upstream_id);
      const existingTLSConfig = state.tlsBySite.get(draft.id) || null;
      const existingAccessPolicy = state.accessBySite.get(normalizeSiteID(draft.id)) || null;
      if (shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig)) {
        await upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, saveOptions);
      }
      const easyProfilePath = `/api/easy-site-profiles/${encodeURIComponent(draft.id)}`;
      await putWithPostFallback(ctx, easyProfilePath, draftToEasyProfile(draft), saveOptions);
      await upsertAccessPolicy(draft, ctx, existingAccessPolicy, saveOptions);
      const accessPolicyID = normalizeSiteID(draft.id);
      const accessAllowlist = Array.isArray(draft.access_allowlist) ? [...draft.access_allowlist] : [];
      const accessDenylist = Array.isArray(draft.access_denylist) ? [...draft.access_denylist] : [];
      if (accessAllowlist.length || accessDenylist.length) {
        state.accessBySite.set(accessPolicyID, {
          id: existingAccessPolicy?.id || `${accessPolicyID}-access`,
          site_id: accessPolicyID,
          enabled: true,
          allowlist: accessAllowlist,
          denylist: accessDenylist,
        });
      } else {
        state.accessBySite.delete(accessPolicyID);
      }
      await compileAndApplySiteRevision(ctx, draft?.id ? [draft.id] : []);
      ctx.notify(ctx.t("toast.siteSaved"));
      clearUnsavedChanges();
      go(`${routeBase()}/${encodeURIComponent(draft.id)}`);
    } catch (error) {
      console.warn("save site failed", error);
      const backendMessage = String(error?.message || "").trim();
      setError(feedback, backendMessage ? `${ctx.t("sites.error.saveSite")}: ${backendMessage}` : ctx.t("sites.error.saveSite"));
    }
  });

  container.querySelector("#service-delete")?.addEventListener("click", async () => {
    const siteID = state.route.siteID;
    if (!siteID) {
      return;
    }
    if (!window.confirm(ctx.t("sites.confirm.deleteSite", { id: siteID }))) {
      return;
    }
    try {
      await deleteServiceWithResources(siteID, ctx, { upstreams: state.upstreams });
      ctx.notify(ctx.t("toast.siteDeleted"));
      clearUnsavedChanges();
      back();
    } catch (_error) {
      setError(feedback, ctx.t("sites.error.deleteSite"));
    }
  });
}
