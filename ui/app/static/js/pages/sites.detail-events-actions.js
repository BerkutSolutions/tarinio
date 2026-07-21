import { compileAndApplySiteRevision, siteSaveNoAutoApplyOptions } from "./sites.save-apply.js";

export function bindDetailActionEvents(params) {
  const {
    container,
    state,
    ctx,
    feedback,
    load,
    render,
    back,
    hostInput,
    idInput,
    normalizeAutoSiteID,
    syncDerivedFieldsFromID,
    normalizeSiteID,
    confirmAction,
    setError,
    setLoading,
    normalizeArray,
    deleteServiceWithResources,
    syncStateDraftFromForm,
    normalizeServiceProfile,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions,
    downloadBlob,
    ensureControlPlaneAccessManagementMethods,
    parseRawDraft,
    getDraft,
    validateDraft,
    shouldUpsertBaseResources,
    upsertSiteResources,
    upsertAccessPolicy,
    putWithPostFallback,
    draftToEasyProfile,
    go,
    routeBase,
    highlightSelector
  } = params;

  container.querySelector("#services-delete-selected")?.addEventListener("click", async () => {
    feedback.innerHTML = "";
    const selectedIDs = Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => normalizeSiteID(site.id) === normalizeSiteID(id)));
    if (!selectedIDs.length) {
      setError(feedback, ctx.t("sites.error.noServicesSelected"));
      return;
    }
    if (!confirmAction(ctx.t("sites.confirm.deleteSelected", { count: selectedIDs.length }))) {
      return;
    }
    try {
      setLoading(feedback, ctx.t("sites.action.deleting"));
      const sharedSnapshot = {
        sites: state.sites,
        upstreams: state.upstreams,
        tlsConfigs: state.tlsConfigs,
        easyProfiles: await ctx.api.get("/api/easy-site-profiles").catch(() => []),
        wafPolicies: await ctx.api.get("/api/waf-policies").catch(() => []),
        ratePolicies: await ctx.api.get("/api/rate-limit-policies").catch(() => []),
        accessPolicies: await ctx.api.get("/api/access-policies").catch(() => [])
      };
      let deletedCount = 0;
      for (const siteID of selectedIDs) {
        await deleteServiceWithResources(siteID, ctx, sharedSnapshot);
        deletedCount += 1;
        sharedSnapshot.sites = normalizeArray(sharedSnapshot.sites).filter((item) => normalizeSiteID(item?.id) !== normalizeSiteID(siteID));
        sharedSnapshot.upstreams = normalizeArray(sharedSnapshot.upstreams).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        sharedSnapshot.tlsConfigs = normalizeArray(sharedSnapshot.tlsConfigs).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        sharedSnapshot.easyProfiles = normalizeArray(sharedSnapshot.easyProfiles).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        sharedSnapshot.wafPolicies = normalizeArray(sharedSnapshot.wafPolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        sharedSnapshot.ratePolicies = normalizeArray(sharedSnapshot.ratePolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        sharedSnapshot.accessPolicies = normalizeArray(sharedSnapshot.accessPolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
      }
      state.selectedSiteIDs = new Set();
      ctx.notify(ctx.t("sites.toast.servicesDeleted", { count: deletedCount }));
      await load();
    } catch (error) {
      setError(feedback, `${ctx.t("sites.error.deleteSite")}: ${String(error?.message || error)}`);
    }
  });
  container.querySelector("#service-id")?.addEventListener("input", (event) => {
    const id = event.target.value.trim().toLowerCase();
    const autoID = normalizeAutoSiteID(hostInput?.value || "");
    event.target.dataset.dirty = id && id !== autoID ? "true" : "";
    syncDerivedFieldsFromID();
  });
  container.querySelector("#service-profile")?.addEventListener("change", (event) => {
    syncStateDraftFromForm();
    const selectedProfile = normalizeServiceProfile(event?.target?.value || state.draft.service_profile);
    const currentProfile = normalizeServiceProfile(state.draft.service_profile);
    if (selectedProfile === currentProfile) {
      return;
    }
    state.draft = applyServiceProfilePresetToDraft(state.draft, selectedProfile);
    render();
  });
  container.querySelector("#service-certificate-id")?.addEventListener("input", (event) => {
    event.target.dataset.dirty = event.target.value.trim() ? "true" : "";
  });

  container.querySelector("#service-ca-server")?.addEventListener("change", toggleCertificateImportActions);
  const certificateArchiveInput = container.querySelector("#service-certificate-archive-file");
  container.querySelector("#service-import-certificate-search")?.addEventListener("change", (event) => {
    const selectedID = String(event?.target?.value || "").trim().toLowerCase();
    if (!selectedID) {
      return;
    }
    const certificateInput = container.querySelector("#service-certificate-id");
    certificateInput.value = selectedID;
    certificateInput.dataset.dirty = "true";
  });

  container.querySelector("#service-certificate-import")?.addEventListener("click", () => {
    certificateArchiveInput?.click();
  });

  certificateArchiveInput?.addEventListener("change", async () => {
    const archiveFile = certificateArchiveInput?.files?.[0] || null;
    if (!archiveFile) {
      return;
    }
    const formData = new FormData();
    formData.set("archive_file", archiveFile);
    try {
      const result = await ctx.api.post("/api/certificate-materials/import-archive", formData);
      const importedCount = Number(result?.imported_count || 0);
      const firstCertificateID = String(result?.items?.[0]?.certificate?.id || "").trim();
      if (firstCertificateID) {
        const certificateInput = container.querySelector("#service-certificate-id");
        if (!String(certificateInput.value || "").trim()) {
          certificateInput.value = firstCertificateID;
          certificateInput.dataset.dirty = "true";
        }
      }
      ctx.notify(importedCount > 0 ? ctx.t("tls.certificates.importedArchive", { count: importedCount }) : ctx.t("sites.tls.imported"));
      await load();
    } catch (error) {
      setError(feedback, `${ctx.t("sites.tls.importFailed")}: ${String(error?.message || error)}`);
    } finally {
      if (certificateArchiveInput) {
        certificateArchiveInput.value = "";
      }
    }
  });

  container.querySelector("#service-certificate-export")?.addEventListener("click", async () => {
    const certificateID = String(container.querySelector("#service-certificate-id")?.value || "").trim().toLowerCase();
    if (!certificateID) {
      ctx.notify(ctx.t("sites.tls.certificateIdRequired"), "error");
      return;
    }
    try {
      const response = await fetch("/api/certificate-materials/export", {
        method: "POST",
        credentials: "include",
        headers: {
          Accept: "application/zip, application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ certificate_ids: [certificateID] })
      });
      if (!response.ok) {
        const bodyText = await response.text();
        let message = `HTTP ${response.status}`;
        if (bodyText) {
          try {
            const payload = JSON.parse(bodyText);
            message = String(payload?.error || payload?.message || message);
          } catch {
            message = bodyText;
          }
        }
        throw new Error(message);
      }
      const blob = await response.blob();
      downloadBlob(`${certificateID}-materials.zip`, blob);
      ctx.notify(ctx.t("sites.tls.exported"));
    } catch (error) {
      setError(feedback, `${ctx.t("sites.tls.exportFailed")}: ${String(error?.message || error)}`);
    }
  });

  toggleCertificateImportActions();
  container.querySelector("#service-use-modsecurity-custom-configuration")?.addEventListener("change", () => {
    syncStateDraftFromForm();
    render();
  });
  container.querySelector("#service-pass-host-header")?.addEventListener("change", () => {
    syncStateDraftFromForm();
    render();
  });

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
      // Access policy is the canonical representation of allow/deny lists.
      // Persist it after the compatibility bridge used by the Easy Profile.
      await upsertAccessPolicy(draft, ctx, existingAccessPolicy, saveOptions);
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
    if (!confirmAction(ctx.t("sites.confirm.deleteSite", { id: siteID }))) {
      return;
    }
    try {
      await deleteServiceWithResources(siteID, ctx, {
        upstreams: state.upstreams
      });
      ctx.notify(ctx.t("toast.siteDeleted"));
      back();
    } catch (error) {
      setError(feedback, ctx.t("sites.error.deleteSite"));
    }
  });

  if (state.highlightedSelector) {
    window.setTimeout(() => highlightSelector(state.highlightedSelector), 30);
  }
}
