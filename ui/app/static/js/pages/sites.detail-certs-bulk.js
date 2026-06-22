import { confirmAction, setError, setLoading } from "../ui.js";
import { normalizeArray, normalizeSiteID } from "./sites.routing-merge.js";

export function bindDetailBulkDelete(container, state, ctx, deps) {
  const { load, deleteServiceWithResources } = deps;
  const feedback = container.querySelector("#sites-feedback");
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
}

export function bindDetailCertificateActions(container, ctx, deps) {
  const { load, downloadBlob } = deps;
  const feedback = container.querySelector("#sites-feedback");
  const certificateArchiveInput = container.querySelector("#service-certificate-archive-file");

  container.querySelector("#service-import-certificate-search")?.addEventListener("change", (event) => {
    const selectedID = String(event?.target?.value || "").trim().toLowerCase();
    if (!selectedID) return;
    const certificateInput = container.querySelector("#service-certificate-id");
    certificateInput.value = selectedID;
    certificateInput.dataset.dirty = "true";
  });
  container.querySelector("#service-certificate-import")?.addEventListener("click", () => {
    certificateArchiveInput?.click();
  });
  certificateArchiveInput?.addEventListener("change", async () => {
    const archiveFile = certificateArchiveInput?.files?.[0] || null;
    if (!archiveFile) return;
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
      if (certificateArchiveInput) certificateArchiveInput.value = "";
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
        headers: { Accept: "application/zip, application/json", "Content-Type": "application/json" },
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
}
