function formatRuntimeSitesError(error) {
  if (!error) {
    return "unknown error";
  }
  if (error instanceof Error) {
    const message = String(error.message || error.name || "error");
    const stack = String(error.stack || "").trim();
    return stack ? `${message}\n${stack}` : message;
  }
  return String(error);
}

function logRuntimeSitesError(stage, error, details = {}) {
  const payload = {
    stage,
    details,
    error: formatRuntimeSitesError(error),
    href: window.location.href,
    at: new Date().toISOString(),
  };
  console.error("[sites-runtime]", payload);
  return payload;
}

function renderRuntimeSitesErrorAlert(ctx, escapeHtml, payload) {
  const detailsText = `${payload.stage}\n${payload.error}`;
  return `
    <div class="alert">
      <div>${escapeHtml(ctx.t("sites.error.load"))}</div>
      <pre class="waf-code" style="margin-top:8px;white-space:pre-wrap;">${escapeHtml(detailsText)}</pre>
    </div>
  `;
}

export async function loadSitesRuntime(state, ctx, container, deps = {}) {
  const {
    setLoading,
    escapeHtml,
    mergeByID,
    unwrapList,
    notifyExpiringCertificates,
    normalizeArray,
    normalizeSiteID,
    normalizeGeoCatalogPayload,
    mergeProfilesBySite,
    tryGetJSON,
    rebuildIndexes,
    syncDraftFromRoute,
    render
  } = deps;
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
    state.tlsConfigs = mergeByID(tlsConfigsResponse, unwrapList(secondaryTLSConfigs, ["tls_configs", "tlsConfigs"]));
    state.certificates = mergeByID(certificatesResponse, unwrapList(secondaryCertificates, ["certificates"]));
    notifyExpiringCertificates(ctx, state.certificates);
    state.accessPolicies = normalizeArray(accessPoliciesResponse);
    state.easyProfiles = mergeProfilesBySite(easyProfilesResponse, secondaryEasyProfiles);
    state.selectedSiteIDs = new Set(Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => normalizeSiteID(site?.id) === normalizeSiteID(id))));
    state.geoCatalog = normalizeGeoCatalogPayload(geoCatalogResponse);
    rebuildIndexes();
    await syncDraftFromRoute();
    render();
  } catch (error) {
    const payload = logRuntimeSitesError("runtime-load-failed", error, {
      routeMode: state?.route?.mode || "",
      siteID: state?.route?.siteID || "",
    });
    container.innerHTML = renderRuntimeSitesErrorAlert(ctx, escapeHtml, payload);
  }
}

export function bindListRuntime(state, ctx, container, deps = {}) {
  const {
    go,
    routeBase,
    load,
    setLoading,
    setError,
    downloadJSON,
    downloadText,
    toEnvKey,
    escapeHtml,
    exportSelectedServicesEnv,
    importServicesFiles,
    pendingImportedDraftRef,
    normalizeArray,
    normalizeSiteID,
    confirmAction,
    deleteServiceWithResources,
    putWithPostFallback
  } = deps;
  const feedback = container.querySelector("#sites-feedback");
  container.querySelector("#services-create")?.addEventListener("click", () => go(`${routeBase()}/new`));
  container.querySelector("#services-refresh")?.addEventListener("click", load);
  container.querySelector("#services-export")?.addEventListener("click", async () => {
    feedback.innerHTML = "";
    if (!state.selectedSiteIDs.size) {
      downloadJSON("waf-services-export.json", { sites: state.sites, upstreams: state.upstreams, tls_configs: state.tlsConfigs });
      ctx.notify(ctx.t("sites.toast.exported"));
      return;
    }
    try {
      const exportedCount = await exportSelectedServicesEnv(ctx, state.sites, state.upstreamsBySite, state.tlsBySite, state.accessBySite, state.selectedSiteIDs);
      ctx.notify(ctx.t("sites.toast.exportedEnv", { count: exportedCount }));
    } catch (error) {
      setError(feedback, `${ctx.t("sites.error.exportEnv")}: ${String(error?.message || error)}`);
    }
  });
  container.querySelector("#services-import")?.addEventListener("click", () => {
    container.querySelector("#services-import-file")?.click();
  });
  container.querySelector("#services-import-file")?.addEventListener("change", async (event) => {
    const files = Array.from(event.target.files || []);
    if (!files.length) {
      return;
    }
    try {
      setLoading(feedback, ctx.t("sites.import.loading"));
      const imported = await importServicesFiles(files, ctx);
      pendingImportedDraftRef.current = {
        draft: imported.draft,
        missingFields: imported.missingFields,
        rawEnvText: imported.rawEnvText,
      };
      if (Array.isArray(imported.missingFields) && imported.missingFields.length) {
        feedback.innerHTML = `
          <div class="waf-empty">
            <div><strong>${escapeHtml(ctx.t("sites.import.warnings"))}</strong><pre class="waf-code">${escapeHtml(`${imported.file}: ${ctx.t("sites.import.missingFields")}: ${imported.missingFields.map((field) => toEnvKey(field)).join(", ")}`)}</pre></div>
          </div>
        `;
      } else {
        feedback.innerHTML = "";
      }
      go(`${routeBase()}/new`);
    } catch (error) {
      setError(feedback, `${ctx.t("sites.error.import")}: ${String(error?.message || error)}`);
    } finally {
      event.target.value = "";
    }
  });
  container.querySelector("#services-search")?.addEventListener("input", (event) => {
    state.search = event.target.value;
    const cursorStart = Number(event.target.selectionStart || state.search.length);
    const cursorEnd = Number(event.target.selectionEnd || cursorStart);
    deps.render();
    const nextInput = container.querySelector("#services-search");
    if (nextInput) {
      nextInput.focus();
      nextInput.setSelectionRange(cursorStart, cursorEnd);
    }
  });
  container.querySelector("#services-sort")?.addEventListener("change", (event) => {
    state.sort = event.target.value;
    deps.render();
  });
  container.querySelector("#services-select-all")?.addEventListener("change", (event) => {
    const checked = Boolean(event.target.checked);
    for (const site of state.filteredSites) {
      if (checked) {
        state.selectedSiteIDs.add(site.id);
      } else {
        state.selectedSiteIDs.delete(site.id);
      }
    }
    deps.render();
  });
  container.querySelectorAll("[data-select-site]").forEach((checkbox) => {
    checkbox.addEventListener("change", (event) => {
      event.stopPropagation();
      const siteID = String(event.target.dataset.selectSite || "");
      if (!siteID) {
        return;
      }
      if (event.target.checked) {
        state.selectedSiteIDs.add(siteID);
      } else {
        state.selectedSiteIDs.delete(siteID);
      }
    });
  });
  container.querySelectorAll("[data-open-site]").forEach((button) => {
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      go(`${routeBase()}/${encodeURIComponent(button.dataset.openSite)}`);
    });
  });
  container.querySelectorAll("[data-toggle-site]").forEach((button) => {
    button.addEventListener("click", async (event) => {
      event.stopPropagation();
      const siteID = String(button.dataset.toggleSite || "").trim();
      if (!siteID) {
        return;
      }
      const site = state.sites.find((item) => normalizeSiteID(item?.id) === normalizeSiteID(siteID));
      if (!site) {
        return;
      }
      const nextEnabled = !(String(button.dataset.toggleEnabled || "") === "1");
      try {
        setLoading(feedback, ctx.t("sites.editor.saving"));
        await ctx.api.put(`/api/sites/${encodeURIComponent(siteID)}`, {
          ...site,
          enabled: nextEnabled,
        });
        const easyProfilePath = `/api/easy-site-profiles/${encodeURIComponent(siteID)}`;
        const profile = await ctx.api.get(easyProfilePath).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
        if (profile && typeof profile === "object") {
          const nextProfile = {
            ...profile,
            front_service: {
              ...(profile.front_service || {}),
              enabled: nextEnabled,
            },
          };
          await putWithPostFallback(ctx, easyProfilePath, nextProfile, { tolerateAutoApplyError: true });
        }
        feedback.innerHTML = "";
        ctx.notify(ctx.t(nextEnabled ? "toast.siteEnabled" : "toast.siteDisabled"));
        await load();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.error.saveSite")}: ${String(error?.message || error)}`);
      }
    });
  });
  container.querySelectorAll("[data-open-service]").forEach((button) => {
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      const url = String(button.dataset.openService || "").trim();
      if (!url) {
        return;
      }
      window.open(url, "_blank", "noopener,noreferrer");
    });
  });
  container.querySelectorAll("[data-open-site-edit]").forEach((row) => {
    row.addEventListener("click", (event) => {
      if (event.target.closest("button, input, select, textarea, a, label")) {
        return;
      }
      const siteID = String(row.dataset.openSiteEdit || "").trim();
      if (!siteID) {
        return;
      }
      go(`${routeBase()}/${encodeURIComponent(siteID)}`);
    });
  });
}
