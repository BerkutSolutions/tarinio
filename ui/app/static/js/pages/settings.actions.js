export function bindSettingsActions(params) {
  const {
    container,
    ctx,
    syncVaultTLSControls,
    setAlert,
    updateStatus,
    updatesEnabled,
    languageSelect,
    languageSave,
    runtimeSave,
    storageSave,
    securitySave,
    loggingSave,
    secretsSave,
    storageLogs,
    storageActivity,
    storageEvents,
    storageBans,
    storageHotIndexDays,
    storageColdIndexDays,
    securityAllowInsecureVaultTLS,
    securityRequireCertificateExportApproval,
    blockDirectIPAccess,
    securityLoginRateEnabled,
    securityLoginRateAttempts,
    securityLoginRateWindow,
    securityLoginRateBlock,
    renderRuntime,
    buildLoggingPayload,
    renderUpdateStatus
  } = params;

  securityAllowInsecureVaultTLS?.addEventListener("change", () => {
    syncVaultTLSControls();
  });

  container.querySelector("#settings-update-check")?.addEventListener("click", async () => {
    setAlert("");
    updateStatus.textContent = ctx.t("settings.updates.checking");
    try {
      const result = await ctx.api.post("/api/settings/runtime/check-updates", {
        update_checks_enabled: !!updatesEnabled?.checked,
        manual: true,
      });
      renderUpdateStatus(ctx, updateStatus, result || {});
      setAlert(ctx.t("settings.updates.checkCompleted"), true);
    } catch {
      updateStatus.textContent = ctx.t("settings.updates.notAvailable");
      setAlert(updateStatus.textContent);
    }
  });

  languageSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      const nextLanguage = String(languageSelect?.value || "en");
      const result = await ctx.api.put("/api/settings/runtime", { language: nextLanguage });
      if (typeof ctx.setLanguage === "function") {
        await ctx.setLanguage(String(result?.language || nextLanguage || "en"));
      }
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  runtimeSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      const payload = {
        update_checks_enabled: !!updatesEnabled?.checked,
      };
      const result = await ctx.api.put("/api/settings/runtime", payload);
      renderUpdateStatus(ctx, updateStatus, result || {});
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  storageSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      const payload = {
        storage: {
          logs_days: Number(storageLogs?.value || "14"),
          activity_days: Number(storageActivity?.value || "30"),
          events_days: Number(storageEvents?.value || "30"),
          bans_days: Number(storageBans?.value || "30"),
          hot_index_days: Number(storageHotIndexDays?.value || "30"),
          cold_index_days: Number(storageColdIndexDays?.value || "730"),
        },
      };
      await ctx.api.put("/api/settings/runtime", payload);
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  securitySave?.addEventListener("click", async () => {
    setAlert("");
    try {
      const payload = {
        security: {
          allow_insecure_vault_tls: !!securityAllowInsecureVaultTLS?.checked,
          require_certificate_export_approval: !!securityRequireCertificateExportApproval?.checked,
          login_rate_limit_enabled: !!securityLoginRateEnabled?.checked,
          login_rate_limit_max_attempts: Number(securityLoginRateAttempts?.value || "10"),
          login_rate_limit_window_seconds: Number(securityLoginRateWindow?.value || "300"),
          login_rate_limit_block_seconds: Number(securityLoginRateBlock?.value || "600"),
        },
      };
      await ctx.api.put("/api/settings/runtime", payload);
      await ctx.api.put("/api/settings/direct-ip-access", { block_direct_ip_access: !!blockDirectIPAccess?.checked });
      syncVaultTLSControls();
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  loggingSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      await ctx.api.put("/api/settings/runtime", buildLoggingPayload());
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  secretsSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      await ctx.api.put("/api/settings/runtime", buildLoggingPayload());
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });
}
