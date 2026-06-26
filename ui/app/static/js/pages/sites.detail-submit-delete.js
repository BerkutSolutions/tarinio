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
      await upsertAccessPolicy(draft, ctx, existingAccessPolicy, saveOptions);
      const easyProfilePath = `/api/easy-site-profiles/${encodeURIComponent(draft.id)}`;
      await putWithPostFallback(ctx, easyProfilePath, draftToEasyProfile(draft), saveOptions);
      await compileAndApplySiteRevision(ctx, draft?.id ? [draft.id] : []);
      ctx.notify(ctx.t("toast.siteSaved"));
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
      back();
    } catch (_error) {
      setError(feedback, ctx.t("sites.error.deleteSite"));
    }
  });
}
