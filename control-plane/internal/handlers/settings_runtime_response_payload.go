package handlers

import (
	"strings"

	"waf/control-plane/internal/appmeta"
	"waf/internal/loggingconfig"
)

func (h *SettingsRuntimeHandler) responsePayload() map[string]any {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return responsePayloadLocked(nil)
}

func currentSecretProviderLocked() string {
	provider := strings.TrimSpace(runtimeSettingsState.logging.SecretProvider)
	if provider == "" {
		return loggingconfig.SecretProviderFile
	}
	return provider
}

func currentLoggingSummaryLocked() map[string]any {
	logging := loggingconfig.MaskSecrets(runtimeSettingsState.logging)
	return map[string]any{
		"hot_backend":        logging.Hot.Backend,
		"cold_backend":       logging.Cold.Backend,
		"secret_provider":    currentSecretProviderLocked(),
		"retention":          logging.Retention,
		"routing":            logging.Routing,
		"opensearch_enabled": loggingconfig.EnabledOpenSearch(runtimeSettingsState.logging),
		"clickhouse_enabled": loggingconfig.EnabledClickHouse(runtimeSettingsState.logging),
		"vault_enabled":      loggingconfig.EnabledVault(runtimeSettingsState.logging),
	}
}

func responsePayloadLocked(indexes map[string]any) map[string]any {
	payload := map[string]any{
		"deployment_mode":        "standalone",
		"app_version":            appmeta.AppVersion,
		"update_checks_enabled":  runtimeSettingsState.updateChecksEnabled,
		"language":               normalizeRuntimeLanguage(runtimeSettingsState.language),
		"storage":                runtimeSettingsState.storage,
		"security":               normalizeRuntimeSecuritySettings(runtimeSettingsState.security),
		"login_appearance":       normalizeLoginAppearance(runtimeSettingsState.loginAppearance),
		"healthcheck_appearance": normalizeHealthcheckAppearance(runtimeSettingsState.healthcheckAppearance),
		"logging":                loggingconfig.MaskSecrets(runtimeSettingsState.logging),
		"logging_summary":        currentLoggingSummaryLocked(),
		"update": map[string]any{
			"has_update":     runtimeSettingsState.hasUpdate,
			"latest_version": runtimeSettingsState.latestVersion,
			"checked_at":     runtimeSettingsState.lastCheckedAt,
			"release_url":    runtimeSettingsState.releaseURL,
		},
	}
	if indexes != nil {
		payload["storage_indexes"] = indexes
	}
	return payload
}

func responsePayloadWithoutIndexesLocked() map[string]any {
	return map[string]any{
		"deployment_mode":        "standalone",
		"app_version":            appmeta.AppVersion,
		"update_checks_enabled":  runtimeSettingsState.updateChecksEnabled,
		"language":               normalizeRuntimeLanguage(runtimeSettingsState.language),
		"storage":                runtimeSettingsState.storage,
		"login_appearance":       normalizeLoginAppearance(runtimeSettingsState.loginAppearance),
		"healthcheck_appearance": normalizeHealthcheckAppearance(runtimeSettingsState.healthcheckAppearance),
		"logging":                loggingconfig.MaskSecrets(runtimeSettingsState.logging),
		"logging_summary":        currentLoggingSummaryLocked(),
		"update": map[string]any{
			"has_update":     runtimeSettingsState.hasUpdate,
			"latest_version": runtimeSettingsState.latestVersion,
			"checked_at":     runtimeSettingsState.lastCheckedAt,
			"release_url":    runtimeSettingsState.releaseURL,
		},
	}
}
