import { renderUpdateStatus } from "./settings.shared.js";
import { setSecretFieldValue } from "./settings.logging-form.js";

export async function renderRuntimeData(params) {
  const {
    ctx,
    permissionSet,
    runtimeStatus,
    loginAppearanceSelect,
    healthcheckAppearanceSelect,
    languageSelect,
    updatesEnabled,
    updateStatus,
    aboutVersion,
    aboutVersionInline,
    clearRuntimeAutoCheckTimer,
    setRuntimeAutoCheckTimer,
    loggingHotBackend,
    loggingColdBackend,
    loggingHotRetention,
    loggingColdRetention,
    loggingEndpoint,
    loggingDatabase,
    loggingTable,
    loggingUsername,
    loggingPassword,
    loggingOpenSearchEndpoint,
    loggingOpenSearchPrefix,
    loggingOpenSearchUsername,
    loggingOpenSearchPassword,
    loggingOpenSearchAPIKey,
    loggingMigrationEnabled,
    loggingRouteRequestsHot,
    loggingRouteRequestsCold,
    loggingRouteEventsHot,
    loggingRouteEventsCold,
    loggingRouteActivityHot,
    loggingRouteActivityCold,
    loggingRouteFallback,
    loggingSecretProvider,
    loggingVaultEnabled,
    loggingVaultAddress,
    loggingVaultMount,
    loggingVaultPathPrefix,
    loggingVaultToken,
    loggingVaultTLSSkipVerify,
    securityLoginRateEnabled,
    securityLoginRateAttempts,
    securityLoginRateWindow,
    securityLoginRateBlock,
    securityAllowInsecureVaultTLS,
    securityRequireCertificateExportApproval,
    syncVaultTLSControls,
    loggingStatus,
    settingsRenderLoggingStatusText,
    storageLogs,
    storageActivity,
    storageEvents,
    storageBans,
    storageHotIndexDays,
    storageColdIndexDays,
    storageIndexesLimit,
    storageIndexesOffset,
    storageIndexesStream,
    setStorageIndexesOffset,
    settingsStorageRenderIndexes
  } = params;
  const canReadGeneral = permissionSet(ctx).has("settings.general.read") || permissionSet(ctx).has("settings.general.write");
  const canReadStorage = permissionSet(ctx).has("settings.storage.read") || permissionSet(ctx).has("settings.storage.write");
  try {
    let runtime = null;
    if (canReadGeneral) {
      runtime = await ctx.api.get("/api/settings/runtime");
      const mode = String(runtime?.deployment_mode || "-");
      const logging = runtime?.logging || {};
      const loggingSummary = runtime?.logging_summary || {};
      const security = runtime?.security || {};
      const clickhouse = logging?.clickhouse || {};
      const opensearch = logging?.opensearch || {};
      const routing = logging?.routing || {};
      const retention = logging?.retention || {};
      const vault = logging?.vault || {};
      runtimeStatus.textContent = ctx.t("settings.runtime.loaded", { mode });
      if (languageSelect) {
        languageSelect.value = String(runtime?.language || ctx.getLanguage?.() || "en");
      }
      if (loginAppearanceSelect) {
        loginAppearanceSelect.value = String(runtime?.login_appearance || "command-center");
      }
      if (healthcheckAppearanceSelect) {
        healthcheckAppearanceSelect.value = String(runtime?.healthcheck_appearance || "variant-1");
      }
      if (updatesEnabled) {
        updatesEnabled.checked = !!runtime?.update_checks_enabled;
      }
      renderUpdateStatus(ctx, updateStatus, runtime || {});
      const currentVersion = String(runtime?.app_version || runtime?.version || "").trim();
      if (currentVersion) {
        const text = `${ctx.t("about.version")}: ${currentVersion}`;
        aboutVersion.textContent = currentVersion;
        aboutVersionInline.textContent = text;
      }
      if (runtime?.update_checks_enabled) {
        clearRuntimeAutoCheckTimer();
        setRuntimeAutoCheckTimer(window.setInterval(async () => {
          try {
            const result = await ctx.api.post("/api/settings/runtime/check-updates", { update_checks_enabled: true, manual: false });
            renderUpdateStatus(ctx, updateStatus, result || {});
          } catch {
            // keep last known status silently
          }
        }, 60 * 60 * 1000));
      } else {
        clearRuntimeAutoCheckTimer();
      }
      if (loggingHotBackend) {
        loggingHotBackend.value = String(logging?.hot?.backend || "opensearch");
      }
      if (loggingColdBackend) {
        loggingColdBackend.value = String(logging?.cold?.backend || "opensearch");
      }
      if (loggingHotRetention) {
        loggingHotRetention.value = String(Number(retention?.hot_days || 30));
      }
      if (loggingColdRetention) {
        loggingColdRetention.value = String(Number(retention?.cold_days || 730));
      }
      if (loggingEndpoint) {
        loggingEndpoint.value = String(clickhouse?.endpoint || "");
      }
      if (loggingDatabase) {
        loggingDatabase.value = String(clickhouse?.database || "waf_logs");
      }
      if (loggingTable) {
        loggingTable.value = String(clickhouse?.table || "request_logs");
      }
      if (loggingUsername) {
        loggingUsername.value = String(clickhouse?.username || "");
      }
      if (loggingPassword) {
        setSecretFieldValue(loggingPassword, clickhouse?.password);
      }
      if (loggingOpenSearchEndpoint) {
        loggingOpenSearchEndpoint.value = String(opensearch?.endpoint || "");
      }
      if (loggingOpenSearchPrefix) {
        loggingOpenSearchPrefix.value = String(opensearch?.index_prefix || "waf-hot");
      }
      if (loggingOpenSearchUsername) {
        loggingOpenSearchUsername.value = String(opensearch?.username || "");
      }
      if (loggingOpenSearchPassword) {
        setSecretFieldValue(loggingOpenSearchPassword, opensearch?.password);
      }
      if (loggingOpenSearchAPIKey) {
        setSecretFieldValue(loggingOpenSearchAPIKey, opensearch?.api_key);
      }
      if (loggingMigrationEnabled) {
        loggingMigrationEnabled.checked = !!clickhouse?.migration_enabled;
      }
      if (loggingRouteRequestsHot) {
        loggingRouteRequestsHot.checked = !!routing?.write_requests_to_hot;
      }
      if (loggingRouteRequestsCold) {
        loggingRouteRequestsCold.checked = !!routing?.write_requests_to_cold;
      }
      if (loggingRouteEventsHot) {
        loggingRouteEventsHot.checked = !!routing?.write_events_to_hot;
      }
      if (loggingRouteEventsCold) {
        loggingRouteEventsCold.checked = !!routing?.write_events_to_cold;
      }
      if (loggingRouteActivityHot) {
        loggingRouteActivityHot.checked = !!routing?.write_activity_to_hot;
      }
      if (loggingRouteActivityCold) {
        loggingRouteActivityCold.checked = !!routing?.write_activity_to_cold;
      }
      if (loggingRouteFallback) {
        loggingRouteFallback.checked = routing?.keep_local_fallback !== false;
      }
      if (loggingSecretProvider) {
        loggingSecretProvider.value = String(logging?.secret_provider || "vault");
      }
      if (loggingVaultEnabled) {
        loggingVaultEnabled.checked = vault?.enabled !== false;
      }
      if (loggingVaultAddress) {
        loggingVaultAddress.value = String(vault?.address || "");
      }
      if (loggingVaultMount) {
        loggingVaultMount.value = String(vault?.mount || "secret");
      }
      if (loggingVaultPathPrefix) {
        loggingVaultPathPrefix.value = String(vault?.path_prefix || "tarinio");
      }
      if (loggingVaultToken) {
        setSecretFieldValue(loggingVaultToken, vault?.token);
      }
      if (loggingVaultTLSSkipVerify) {
        loggingVaultTLSSkipVerify.checked = !!vault?.tls_skip_verify;
      }
      if (securityLoginRateEnabled) {
        securityLoginRateEnabled.checked = security?.login_rate_limit_enabled !== false;
      }
      if (securityLoginRateAttempts) {
        securityLoginRateAttempts.value = String(Number(security?.login_rate_limit_max_attempts || 10));
      }
      if (securityLoginRateWindow) {
        securityLoginRateWindow.value = String(Number(security?.login_rate_limit_window_seconds || 300));
      }
      if (securityLoginRateBlock) {
        securityLoginRateBlock.value = String(Number(security?.login_rate_limit_block_seconds || 600));
      }
      if (securityAllowInsecureVaultTLS) {
        securityAllowInsecureVaultTLS.checked = !!security?.allow_insecure_vault_tls;
      }
      if (securityRequireCertificateExportApproval) {
        securityRequireCertificateExportApproval.checked = security?.require_certificate_export_approval !== false;
      }
      syncVaultTLSControls();
      if (loggingStatus) {
        loggingStatus.textContent = settingsRenderLoggingStatusText(logging, loggingSummary);
      }
    }
    if (canReadStorage) {
      const storage = runtime?.storage || {};
      const storageRetention = runtime?.logging?.retention || {};
      if (storageLogs) {
        storageLogs.value = String(Number(storage?.logs_days || 14));
      }
      if (storageActivity) {
        storageActivity.value = String(Number(storage?.activity_days || 30));
      }
      if (storageEvents) {
        storageEvents.value = String(Number(storage?.events_days || 30));
      }
      if (storageBans) {
        storageBans.value = String(Number(storage?.bans_days || 30));
      }
      if (storageHotIndexDays) {
        storageHotIndexDays.value = String(Number(storage?.hot_index_days || storageRetention?.hot_days || 30));
      }
      if (storageColdIndexDays) {
        storageColdIndexDays.value = String(Number(storage?.cold_index_days || storageRetention?.cold_days || 730));
      }
      const indexesPayload = await ctx.api.get(`/api/settings/runtime/storage-indexes?stream=${encodeURIComponent(storageIndexesStream)}&storage_indexes_limit=${storageIndexesLimit}&storage_indexes_offset=${storageIndexesOffset}`).catch(() => ({ items: [], total: 0, limit: storageIndexesLimit, offset: storageIndexesOffset, stream: storageIndexesStream }));
      setStorageIndexesOffset(Number(indexesPayload?.offset || 0));
      settingsStorageRenderIndexes(indexesPayload);
    }
  } catch {
    runtimeStatus.textContent = ctx.t("settings.runtime.shell");
    updateStatus.textContent = ctx.t("settings.updates.notAvailable");
    settingsStorageRenderIndexes({ items: [], total: 0, limit: storageIndexesLimit, offset: storageIndexesOffset });
    clearRuntimeAutoCheckTimer();
  }
}
