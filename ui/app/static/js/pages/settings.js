import { escapeHtml } from "../ui.js";
import {
  activeTabFromPath,
  availableTabs,
  permissionSet,
  renderLanguageOptions,
  renderUpdateStatus,
  showTab
} from "./settings.shared.js";
import {
  buildLoggingPayload as buildLoggingPayloadExternal,
  renderLoggingStatusText as renderLoggingStatusTextExternal,
  renderStorageIndexes as renderStorageIndexesExternal
} from "./settings.storage-logging.js";
import { clearRuntimeAutoCheckTimer, setRuntimeAutoCheckTimer } from "./settings.runtime-timer.js";
import { renderRuntimeData } from "./settings.runtime-render.js";
import { bindSettingsActions } from "./settings.actions.js";

const SETTINGS_TABS = [
  { id: "general", path: "/settings/general", labelKey: "settings.tabs.general", permissions: ["settings.general.read", "settings.general.write"] },
  { id: "storage", path: "/settings/storage", labelKey: "settings.tabs.storage", permissions: ["settings.storage.read", "settings.storage.write"] },
  { id: "security", path: "/settings/security", labelKey: "settings.tabs.security", permissions: ["settings.general.read", "settings.general.write"] },
  { id: "logging", path: "/settings/logging", labelKey: "settings.tabs.logging", permissions: ["settings.general.read", "settings.general.write", "settings.storage.read", "settings.storage.write"] },
  { id: "secrets", path: "/settings/secrets", labelKey: "settings.tabs.secrets", permissions: ["settings.general.read", "settings.general.write", "settings.storage.read", "settings.storage.write"] },
  { id: "about", path: "/settings/about", labelKey: "settings.tabs.about", permissions: ["settings.about.read"] },
];

export async function renderSettings(container, ctx) {
  clearRuntimeAutoCheckTimer();
  let storageIndexesOffset = 0;
  const storageIndexesLimit = 10;
  let storageIndexesStream = "requests";
  const tabs = availableTabs(ctx, SETTINGS_TABS);
  const currentTab = tabs.find((tab) => tab.id === activeTabFromPath(SETTINGS_TABS)) || tabs[0] || SETTINGS_TABS.find((tab) => tab.id === "about");
  if (!currentTab) {
    container.innerHTML = "";
    return;
  }

  container.innerHTML = `
    <div class="waf-page-stack" id="settings-page">
      <div id="settings-alert" class="alert" hidden></div>

      <section class="waf-card settings-tabs-card">
        <div class="waf-card-body">
          <div class="tabs browser-tabs settings-tabs" id="settings-tabs" role="tablist" aria-label="${escapeHtml(ctx.t("settings.title"))}">
            ${tabs.map((tab) => `
              <a
                class="tab-btn"
                href="${escapeHtml(tab.path)}"
                role="tab"
                data-settings-tab-link="${escapeHtml(tab.id)}"
              >${escapeHtml(ctx.t(tab.labelKey))}</a>
            `).join("")}
          </div>
        </div>
      </section>

      <div class="settings-panel" data-settings-panel="general" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("settings.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("settings.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="waf-grid two">
              <div class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.language"))}</div>
                </div>
                <div class="waf-field">
                  <label for="settings-language-select">${escapeHtml(ctx.t("settings.languageHint"))}</label>
                  <select id="settings-language-select">
                    ${renderLanguageOptions()}
                  </select>
                </div>
                <div class="waf-actions" style="margin-top:10px;">
                  <button id="settings-language-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
                </div>
              </div>

              <div class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.runtime.title"))}</div>
                </div>
                <div class="waf-note" id="settings-runtime-status">${escapeHtml(ctx.t("settings.runtime.shell"))}</div>
                <div class="waf-note settings-version-note" id="settings-about-version-inline">${escapeHtml(ctx.t("about.version"))}: -</div>
              </div>

              <div class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.updates.title"))}</div>
                  <button id="settings-update-check" class="btn ghost btn-sm" type="button">${escapeHtml(ctx.t("settings.updates.checkNow"))}</button>
                </div>
                <label class="waf-checkbox" for="settings-updates-enabled">
                  <input type="checkbox" id="settings-updates-enabled">
                  <span>${escapeHtml(ctx.t("settings.updates.enabled"))}</span>
                </label>
                <div class="waf-actions" style="margin-top:10px;">
                  <button id="settings-runtime-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
                </div>
                <div class="waf-note settings-update-status-note" id="settings-update-status">${escapeHtml(ctx.t("settings.updates.notChecked"))}</div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <div class="settings-panel" data-settings-panel="storage" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("settings.storage.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("settings.storage.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="waf-form-grid two">
              <div class="waf-field">
                <label for="settings-storage-logs">${escapeHtml(ctx.t("settings.storage.logs"))}</label>
                <input id="settings-storage-logs" type="number" min="1" step="1" value="14">
              </div>
              <div class="waf-field">
                <label for="settings-storage-activity">${escapeHtml(ctx.t("settings.storage.activity"))}</label>
                <input id="settings-storage-activity" type="number" min="1" step="1" value="30">
              </div>
              <div class="waf-field">
                <label for="settings-storage-events">${escapeHtml(ctx.t("settings.storage.events"))}</label>
                <input id="settings-storage-events" type="number" min="1" step="1" value="30">
              </div>
              <div class="waf-field">
                <label for="settings-storage-bans">${escapeHtml(ctx.t("settings.storage.bans"))}</label>
                <input id="settings-storage-bans" type="number" min="1" step="1" value="30">
              </div>
              <div class="waf-field">
                <label for="settings-storage-hot-index-days">${escapeHtml(ctx.t("settings.storage.hotIndexDays"))}</label>
                <input id="settings-storage-hot-index-days" type="number" min="1" max="30" step="1" value="30">
              </div>
              <div class="waf-field">
                <label for="settings-storage-cold-index-days">${escapeHtml(ctx.t("settings.storage.coldIndexDays"))}</label>
                <input id="settings-storage-cold-index-days" type="number" min="1" max="730" step="1" value="730">
              </div>
            </div>
            <div class="waf-actions">
              <button id="settings-storage-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
          </div>
        </section>
      </div>

      <div class="settings-panel" data-settings-panel="security" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("settings.security.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("settings.security.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <section class="waf-list-item">
              <div class="waf-list-head">
                <div class="waf-list-title">${escapeHtml(ctx.t("settings.security.loginRateLimit.title"))}</div>
              </div>
              <label class="waf-checkbox" for="settings-security-login-rate-enabled">
                <input type="checkbox" id="settings-security-login-rate-enabled" checked>
                <span>${escapeHtml(ctx.t("settings.security.loginRateLimit.enabled"))}</span>
              </label>
              <div class="waf-form-grid three">
                <div class="waf-field">
                  <label for="settings-security-login-rate-attempts">${escapeHtml(ctx.t("settings.security.loginRateLimit.maxAttempts"))}</label>
                  <input id="settings-security-login-rate-attempts" type="number" min="3" max="100" step="1" value="10">
                </div>
                <div class="waf-field">
                  <label for="settings-security-login-rate-window">${escapeHtml(ctx.t("settings.security.loginRateLimit.windowSeconds"))}</label>
                  <input id="settings-security-login-rate-window" type="number" min="60" max="86400" step="1" value="300">
                </div>
                <div class="waf-field">
                  <label for="settings-security-login-rate-block">${escapeHtml(ctx.t("settings.security.loginRateLimit.blockSeconds"))}</label>
                  <input id="settings-security-login-rate-block" type="number" min="60" max="86400" step="1" value="600">
                </div>
              </div>
            </section>

            <section class="waf-list-item">
              <div class="waf-list-head">
                <div class="waf-list-title">${escapeHtml(ctx.t("settings.security.vaultTls.title"))}</div>
              </div>
              <label class="waf-checkbox" for="settings-security-allow-insecure-vault-tls">
                <input type="checkbox" id="settings-security-allow-insecure-vault-tls">
                <span>${escapeHtml(ctx.t("settings.security.vaultTls.allowInsecure"))}</span>
              </label>
            </section>

            <div class="waf-actions">
              <button id="settings-security-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
          </div>
        </section>
      </div>

      <div class="settings-panel" data-settings-panel="logging" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("settings.logging.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("settings.logging.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="waf-grid two">
              <section class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.logging.frame.hot"))}</div>
                </div>
                <div class="waf-note">${escapeHtml(ctx.t("settings.logging.frame.hotHint"))}</div>
                <div class="waf-form-grid two">
                  <div class="waf-field">
                    <label for="settings-logging-hot-backend">${escapeHtml(ctx.t("settings.logging.hot.backend"))}</label>
                    <select id="settings-logging-hot-backend">
                      <option value="file">${escapeHtml(ctx.t("settings.logging.backend.file"))}</option>
                      <option value="opensearch" selected>${escapeHtml(ctx.t("settings.logging.backend.opensearch"))}</option>
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-opensearch-endpoint">${escapeHtml(ctx.t("settings.logging.opensearch.endpoint"))}</label>
                    <input id="settings-logging-opensearch-endpoint" type="text" placeholder="http://opensearch:9200">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-opensearch-prefix">${escapeHtml(ctx.t("settings.logging.opensearch.indexPrefix"))}</label>
                    <input id="settings-logging-opensearch-prefix" type="text" placeholder="waf-hot">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-opensearch-username">${escapeHtml(ctx.t("settings.logging.opensearch.username"))}</label>
                    <input id="settings-logging-opensearch-username" type="text" placeholder="admin">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-opensearch-password">${escapeHtml(ctx.t("settings.logging.opensearch.password"))}</label>
                    <input id="settings-logging-opensearch-password" type="password" placeholder="${escapeHtml(ctx.t("settings.logging.passwordPlaceholder"))}">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-opensearch-apikey">${escapeHtml(ctx.t("settings.logging.opensearch.apiKey"))}</label>
                    <input id="settings-logging-opensearch-apikey" type="password" placeholder="${escapeHtml(ctx.t("settings.logging.passwordPlaceholder"))}">
                  </div>
                </div>
              </section>

              <section class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.logging.frame.cold"))}</div>
                </div>
                <div class="waf-note">${escapeHtml(ctx.t("settings.logging.frame.coldHint"))}</div>
                <div class="waf-form-grid two">
                  <div class="waf-field">
                    <label for="settings-logging-cold-backend">${escapeHtml(ctx.t("settings.logging.cold.backend"))}</label>
                    <select id="settings-logging-cold-backend">
                      <option value="file">${escapeHtml(ctx.t("settings.logging.backend.file"))}</option>
                      <option value="opensearch" selected>${escapeHtml(ctx.t("settings.logging.backend.opensearch"))}</option>
                      <option value="clickhouse">${escapeHtml(ctx.t("settings.logging.backend.clickhouse"))}</option>
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-endpoint">${escapeHtml(ctx.t("settings.logging.endpoint"))}</label>
                    <input id="settings-logging-endpoint" type="text" placeholder="http://clickhouse:8123">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-database">${escapeHtml(ctx.t("settings.logging.database"))}</label>
                    <input id="settings-logging-database" type="text" placeholder="waf_logs">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-table">${escapeHtml(ctx.t("settings.logging.table"))}</label>
                    <input id="settings-logging-table" type="text" placeholder="request_logs">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-username">${escapeHtml(ctx.t("settings.logging.username"))}</label>
                    <input id="settings-logging-username" type="text" placeholder="waf">
                  </div>
                  <div class="waf-field">
                    <label for="settings-logging-password">${escapeHtml(ctx.t("settings.logging.password"))}</label>
                    <input id="settings-logging-password" type="password" placeholder="${escapeHtml(ctx.t("settings.logging.passwordPlaceholder"))}">
                  </div>
                </div>
                <label class="waf-checkbox" for="settings-logging-migration-enabled">
                  <input type="checkbox" id="settings-logging-migration-enabled">
                  <span>${escapeHtml(ctx.t("settings.logging.migrationEnabled"))}</span>
                </label>
              </section>
            </div>

            <div class="waf-grid two">
              <section class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.logging.frame.routing"))}</div>
                </div>
                <div class="waf-note">${escapeHtml(ctx.t("settings.logging.frame.routingHint"))}</div>
                <div class="waf-stack">
                  <label class="waf-checkbox" for="settings-logging-route-requests-hot"><input type="checkbox" id="settings-logging-route-requests-hot"><span>${escapeHtml(ctx.t("settings.logging.routing.requestsHot"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-requests-cold"><input type="checkbox" id="settings-logging-route-requests-cold"><span>${escapeHtml(ctx.t("settings.logging.routing.requestsCold"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-events-hot"><input type="checkbox" id="settings-logging-route-events-hot"><span>${escapeHtml(ctx.t("settings.logging.routing.eventsHot"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-events-cold"><input type="checkbox" id="settings-logging-route-events-cold"><span>${escapeHtml(ctx.t("settings.logging.routing.eventsCold"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-activity-hot"><input type="checkbox" id="settings-logging-route-activity-hot"><span>${escapeHtml(ctx.t("settings.logging.routing.activityHot"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-activity-cold"><input type="checkbox" id="settings-logging-route-activity-cold"><span>${escapeHtml(ctx.t("settings.logging.routing.activityCold"))}</span></label>
                  <label class="waf-checkbox" for="settings-logging-route-fallback"><input type="checkbox" id="settings-logging-route-fallback"><span>${escapeHtml(ctx.t("settings.logging.routing.keepFallback"))}</span></label>
                </div>
              </section>

            </div>
            <div class="waf-note" id="settings-logging-status">${escapeHtml(ctx.t("settings.logging.status.notConfigured"))}</div>
            <div class="waf-actions">
              <button id="settings-logging-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
            <div id="settings-storage-indexes"></div>
          </div>
        </section>
      </div>

      <div class="settings-panel" data-settings-panel="secrets" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("settings.secrets.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("settings.secrets.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <section class="waf-list-item">
              <div class="waf-list-head">
                <div class="waf-list-title">${escapeHtml(ctx.t("settings.logging.frame.vault"))}</div>
              </div>
              <div class="waf-note">${escapeHtml(ctx.t("settings.logging.frame.vaultHint"))}</div>
              <div class="waf-form-grid two">
                <div class="waf-field">
                  <label for="settings-logging-secret-provider">${escapeHtml(ctx.t("settings.logging.vault.provider"))}</label>
                  <select id="settings-logging-secret-provider">
                    <option value="encrypted_file">${escapeHtml(ctx.t("settings.logging.vault.provider.file"))}</option>
                    <option value="vault" selected>${escapeHtml(ctx.t("settings.logging.vault.provider.vault"))}</option>
                  </select>
                </div>
                <div class="waf-field">
                  <label for="settings-logging-vault-address">${escapeHtml(ctx.t("settings.logging.vault.address"))}</label>
                  <input id="settings-logging-vault-address" type="text" placeholder="http://vault:8200">
                </div>
                <div class="waf-field">
                  <label for="settings-logging-vault-mount">${escapeHtml(ctx.t("settings.logging.vault.mount"))}</label>
                  <input id="settings-logging-vault-mount" type="text" placeholder="secret">
                </div>
                <div class="waf-field">
                  <label for="settings-logging-vault-path-prefix">${escapeHtml(ctx.t("settings.logging.vault.pathPrefix"))}</label>
                  <input id="settings-logging-vault-path-prefix" type="text" placeholder="tarinio">
                </div>
                <div class="waf-field">
                  <label for="settings-logging-vault-token">${escapeHtml(ctx.t("settings.logging.vault.token"))}</label>
                  <input id="settings-logging-vault-token" type="password" placeholder="${escapeHtml(ctx.t("settings.logging.passwordPlaceholder"))}">
                </div>
              </div>
              <label class="waf-checkbox" for="settings-logging-vault-enabled">
                <input type="checkbox" id="settings-logging-vault-enabled" checked>
                <span>${escapeHtml(ctx.t("settings.logging.vault.enabled"))}</span>
              </label>
              <label class="waf-checkbox" for="settings-logging-vault-tls-skip-verify">
                <input type="checkbox" id="settings-logging-vault-tls-skip-verify">
                <span>${escapeHtml(ctx.t("settings.logging.vault.tlsSkipVerify"))}</span>
              </label>
              <div class="waf-actions" style="margin-top:12px;">
                <button id="settings-secrets-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
              </div>
            </section>
          </div>
        </section>
      </div>

      <div class="settings-panel" data-settings-panel="about" hidden>
        <section class="waf-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("about.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("about.subtitle"))}</div>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="about-grid">
              <div class="about-logo-wrap">
                <img class="about-logo" src="/static/logo500x300.png" alt="Berkut Solutions - TARINIO">
              </div>
              <div class="about-content">
                <h4 class="about-name">${escapeHtml(ctx.t("about.projectName"))}</h4>
                <p class="about-description">${escapeHtml(ctx.t("about.projectDescription"))}</p>
                <p class="muted">${escapeHtml(ctx.t("about.version"))}: <strong id="settings-about-version">${escapeHtml(ctx.t("about.versionFallback"))}</strong></p>
                <div class="about-links">
                  <a class="btn primary btn-sm" id="settings-about-project-link" href="https://github.com/BerkutSolutions/tarinio" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.project"))}</a>
                  <a class="btn ghost btn-sm" href="https://github.com/BerkutSolutions" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.profile"))}</a>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  `;

  const alert = container.querySelector("#settings-alert");
  const updateStatus = container.querySelector("#settings-update-status");
  const runtimeStatus = container.querySelector("#settings-runtime-status");
  const aboutVersion = container.querySelector("#settings-about-version");
  const aboutVersionInline = container.querySelector("#settings-about-version-inline");
  const aboutProjectLink = container.querySelector("#settings-about-project-link");
  const updatesEnabled = container.querySelector("#settings-updates-enabled");
  const storageLogs = container.querySelector("#settings-storage-logs");
  const storageActivity = container.querySelector("#settings-storage-activity");
  const storageEvents = container.querySelector("#settings-storage-events");
  const storageBans = container.querySelector("#settings-storage-bans");
  const storageHotIndexDays = container.querySelector("#settings-storage-hot-index-days");
  const storageColdIndexDays = container.querySelector("#settings-storage-cold-index-days");
  const storageSave = container.querySelector("#settings-storage-save");
  const securityLoginRateEnabled = container.querySelector("#settings-security-login-rate-enabled");
  const securityLoginRateAttempts = container.querySelector("#settings-security-login-rate-attempts");
  const securityLoginRateWindow = container.querySelector("#settings-security-login-rate-window");
  const securityLoginRateBlock = container.querySelector("#settings-security-login-rate-block");
  const securityAllowInsecureVaultTLS = container.querySelector("#settings-security-allow-insecure-vault-tls");
  const securitySave = container.querySelector("#settings-security-save");
  const secretsSave = container.querySelector("#settings-secrets-save");
  const loggingHotBackend = container.querySelector("#settings-logging-hot-backend");
  const loggingColdBackend = container.querySelector("#settings-logging-cold-backend");
  const loggingHotRetention = container.querySelector("#settings-logging-hot-retention");
  const loggingColdRetention = container.querySelector("#settings-logging-cold-retention");
  const loggingEndpoint = container.querySelector("#settings-logging-endpoint");
  const loggingDatabase = container.querySelector("#settings-logging-database");
  const loggingTable = container.querySelector("#settings-logging-table");
  const loggingUsername = container.querySelector("#settings-logging-username");
  const loggingPassword = container.querySelector("#settings-logging-password");
  const loggingOpenSearchEndpoint = container.querySelector("#settings-logging-opensearch-endpoint");
  const loggingOpenSearchPrefix = container.querySelector("#settings-logging-opensearch-prefix");
  const loggingOpenSearchUsername = container.querySelector("#settings-logging-opensearch-username");
  const loggingOpenSearchPassword = container.querySelector("#settings-logging-opensearch-password");
  const loggingOpenSearchAPIKey = container.querySelector("#settings-logging-opensearch-apikey");
  const loggingMigrationEnabled = container.querySelector("#settings-logging-migration-enabled");
  const loggingRouteRequestsHot = container.querySelector("#settings-logging-route-requests-hot");
  const loggingRouteRequestsCold = container.querySelector("#settings-logging-route-requests-cold");
  const loggingRouteEventsHot = container.querySelector("#settings-logging-route-events-hot");
  const loggingRouteEventsCold = container.querySelector("#settings-logging-route-events-cold");
  const loggingRouteActivityHot = container.querySelector("#settings-logging-route-activity-hot");
  const loggingRouteActivityCold = container.querySelector("#settings-logging-route-activity-cold");
  const loggingRouteFallback = container.querySelector("#settings-logging-route-fallback");
  const loggingSecretProvider = container.querySelector("#settings-logging-secret-provider");
  const loggingVaultEnabled = container.querySelector("#settings-logging-vault-enabled");
  const loggingVaultAddress = container.querySelector("#settings-logging-vault-address");
  const loggingVaultMount = container.querySelector("#settings-logging-vault-mount");
  const loggingVaultPathPrefix = container.querySelector("#settings-logging-vault-path-prefix");
  const loggingVaultToken = container.querySelector("#settings-logging-vault-token");
  const loggingVaultTLSSkipVerify = container.querySelector("#settings-logging-vault-tls-skip-verify");
  const loggingStatus = container.querySelector("#settings-logging-status");
  const loggingSave = container.querySelector("#settings-logging-save");
  const storageIndexesNode = container.querySelector("#settings-storage-indexes");
  const languageSelect = container.querySelector("#settings-language-select");
  const languageSave = container.querySelector("#settings-language-save");
  const runtimeSave = container.querySelector("#settings-runtime-save");
  if (languageSelect) {
    languageSelect.value = String(ctx.getLanguage?.() || "en");
  }

  const setAlert = (message, success = false) => {
    const text = String(message || "").trim();
    if (!text) {
      alert.hidden = true;
      alert.textContent = "";
      alert.classList.remove("success");
      return;
    }
    alert.hidden = false;
    alert.textContent = text;
    alert.classList.toggle("success", !!success);
  };

  const syncVaultTLSControls = () => {
    const allowInsecure = !!securityAllowInsecureVaultTLS?.checked;
    if (loggingVaultTLSSkipVerify) {
      if (!allowInsecure) {
        loggingVaultTLSSkipVerify.checked = false;
      }
      loggingVaultTLSSkipVerify.disabled = !allowInsecure;
    }
  };

  const renderStorageIndexes = (indexes) => {
    if (!storageIndexesNode) {
      return;
    }
    const items = Array.isArray(indexes?.items) ? indexes.items : [];
    const total = Number(indexes?.total || 0);
    const limit = Number(indexes?.limit || storageIndexesLimit);
    const offset = Number(indexes?.offset || 0);
    const currentPage = Math.floor(offset / Math.max(1, limit)) + 1;
    const totalPages = Math.max(1, Math.ceil(total / Math.max(1, limit)));
    const pages = [];
    for (let page = 1; page <= Math.min(10, totalPages); page += 1) {
      pages.push(`<button type="button" class="btn ghost btn-sm${page === currentPage ? " active" : ""}" data-storage-index-page="${page}">${page}</button>`);
    }
    if (totalPages > 10) {
      pages.push(`<span class="muted">...</span>`);
      pages.push(`<button type="button" class="btn ghost btn-sm${currentPage === totalPages ? " active" : ""}" data-storage-index-page="${totalPages}">${totalPages}</button>`);
    }
    storageIndexesNode.innerHTML = `
      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("settings.storage.indexes.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("settings.storage.indexes.subtitle"))}</div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="tabs browser-tabs" role="tablist" aria-label="${escapeHtml(ctx.t("settings.storage.indexes.title"))}">
            <button type="button" class="tab-btn${storageIndexesStream === "requests" ? " active" : ""}" data-storage-index-stream="requests">${escapeHtml(ctx.t("app.requests"))}</button>
            <button type="button" class="tab-btn${storageIndexesStream === "events" ? " active" : ""}" data-storage-index-stream="events">${escapeHtml(ctx.t("app.events"))}</button>
            <button type="button" class="tab-btn${storageIndexesStream === "activity" ? " active" : ""}" data-storage-index-stream="activity">${escapeHtml(ctx.t("app.activity"))}</button>
          </div>
          <div class="waf-empty">${escapeHtml(ctx.t("settings.storage.indexes.total"))}: ${total}</div>
          <div class="waf-table-wrap">
            <table class="waf-table">
              <thead>
                <tr>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.date"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.file"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.lines"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.size"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.updated"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.actions"))}</th>
                </tr>
              </thead>
              <tbody>
                ${items.length
                  ? items.map((item) => `
                    <tr>
                      <td>${escapeHtml(String(item?.date || "-"))}</td>
                      <td>${escapeHtml(String(item?.file_name || "-"))}</td>
                      <td>${escapeHtml(String(item?.lines ?? 0))}</td>
                      <td>${escapeHtml(String(item?.size_bytes ?? 0))}</td>
                      <td>${escapeHtml(String(item?.updated_at || "-"))}</td>
                      <td>
                        <button
                          type="button"
                          class="btn ghost btn-sm"
                          data-storage-index-delete="${escapeHtml(String(item?.date || ""))}"
                        >${escapeHtml(ctx.t("common.delete"))}</button>
                      </td>
                    </tr>
                  `).join("")
                  : `<tr><td colspan="6"><div class="waf-empty">${escapeHtml(ctx.t("settings.storage.indexes.empty"))}</div></td></tr>`}
              </tbody>
            </table>
          </div>
          <div class="waf-pager">
            <div class="muted">${escapeHtml(ctx.t("settings.storage.indexes.page"))}: ${currentPage}/${totalPages}</div>
            <div class="waf-actions">${pages.join("")}</div>
          </div>
        </div>
      </section>
    `;
    storageIndexesNode.querySelectorAll("[data-storage-index-page]").forEach((button) => {
      button.addEventListener("click", async () => {
        const page = Number.parseInt(String(button.dataset.storageIndexPage || "1"), 10);
        if (!Number.isFinite(page) || page < 1) {
          return;
        }
        storageIndexesOffset = (page - 1) * storageIndexesLimit;
        await renderRuntime();
      });
    });
    storageIndexesNode.querySelectorAll("[data-storage-index-stream]").forEach((button) => {
      button.addEventListener("click", async () => {
        const nextStream = String(button.dataset.storageIndexStream || "requests");
        if (!nextStream || nextStream === storageIndexesStream) {
          return;
        }
        storageIndexesStream = nextStream;
        storageIndexesOffset = 0;
        await renderRuntime();
      });
    });
    storageIndexesNode.querySelectorAll("[data-storage-index-delete]").forEach((button) => {
      button.addEventListener("click", async () => {
        const day = String(button.dataset.storageIndexDelete || "").trim();
        if (!day) {
          return;
        }
        if (!window.confirm(ctx.t("settings.storage.indexes.deleteConfirm", { date: day }))) {
          return;
        }
        setAlert("");
        try {
          await ctx.api.delete(`/api/settings/runtime/storage-indexes?stream=${encodeURIComponent(storageIndexesStream)}&date=${encodeURIComponent(day)}`);
          setAlert(ctx.t("settings.saved"), true);
          await renderRuntime();
        } catch (error) {
          setAlert(error?.message || ctx.t("common.error"));
        }
      });
    });
  };

  const renderLoggingStatusText = (logging, summary) => {
    const hotBackend = String(summary?.hot_backend || logging?.hot?.backend || "file");
    const coldBackend = String(summary?.cold_backend || logging?.cold?.backend || "file");
    const secretProvider = String(summary?.secret_provider || logging?.secret_provider || "vault");
    const hotRetention = Number(logging?.retention?.hot_days || 30);
    const coldRetention = Number(logging?.retention?.cold_days || 730);
    if (hotBackend === "opensearch" && coldBackend === "clickhouse") {
      return ctx.t("settings.logging.status.dual", {
        hotDays: hotRetention,
        coldDays: coldRetention,
        secretProvider,
      });
    }
    if (hotBackend === "opensearch" && coldBackend === "opensearch") {
      return ctx.t("settings.logging.status.opensearch", {
        endpoint: String(logging?.opensearch?.endpoint || "-"),
        retention: coldRetention,
      });
    }
    if (hotBackend === "opensearch" || coldBackend === "opensearch") {
      return ctx.t("settings.logging.status.opensearch", {
        endpoint: String(logging?.opensearch?.endpoint || "-"),
        retention: hotBackend === "opensearch" ? hotRetention : coldRetention,
      });
    }
    if (coldBackend === "clickhouse") {
      return ctx.t("settings.logging.status.clickhouse", {
        endpoint: String(logging?.clickhouse?.endpoint || "-"),
        database: String(logging?.clickhouse?.database || "waf_logs"),
        table: String(logging?.clickhouse?.table || "request_logs"),
      });
    }
    return ctx.t("settings.logging.status.file");
  };

  const buildLoggingPayload = () => ({
    logging: {
      backend: String(loggingHotBackend?.value || "opensearch") === "opensearch"
        ? "opensearch"
        : (String(loggingColdBackend?.value || "opensearch") === "clickhouse" ? "clickhouse" : (String(loggingColdBackend?.value || "opensearch") === "opensearch" ? "opensearch" : "file")),
      hot: {
        backend: String(loggingHotBackend?.value || "opensearch"),
      },
      cold: {
        backend: String(loggingColdBackend?.value || "opensearch"),
      },
      retention: {
        hot_days: Number(storageHotIndexDays?.value || loggingHotRetention?.value || "30"),
        cold_days: Number(storageColdIndexDays?.value || loggingColdRetention?.value || "730"),
      },
      routing: {
        write_requests_to_hot: !!loggingRouteRequestsHot?.checked,
        write_requests_to_cold: !!loggingRouteRequestsCold?.checked,
        write_events_to_hot: !!loggingRouteEventsHot?.checked,
        write_events_to_cold: !!loggingRouteEventsCold?.checked,
        write_activity_to_hot: !!loggingRouteActivityHot?.checked,
        write_activity_to_cold: !!loggingRouteActivityCold?.checked,
        keep_local_fallback: !!loggingRouteFallback?.checked,
      },
      secret_provider: String(loggingSecretProvider?.value || "vault"),
      vault: {
        enabled: !!loggingVaultEnabled?.checked,
        address: String(loggingVaultAddress?.value || "").trim(),
        mount: String(loggingVaultMount?.value || "secret").trim(),
        path_prefix: String(loggingVaultPathPrefix?.value || "tarinio").trim(),
        token: String(loggingVaultToken?.value || "").trim(),
        tls_skip_verify: !!loggingVaultTLSSkipVerify?.checked,
      },
      opensearch: {
        endpoint: String(loggingOpenSearchEndpoint?.value || "").trim(),
        index_prefix: String(loggingOpenSearchPrefix?.value || "waf-hot").trim(),
        username: String(loggingOpenSearchUsername?.value || "").trim(),
        password: String(loggingOpenSearchPassword?.value || "").trim(),
        api_key: String(loggingOpenSearchAPIKey?.value || "").trim(),
      },
      clickhouse: {
        endpoint: String(loggingEndpoint?.value || "").trim(),
        database: String(loggingDatabase?.value || "waf_logs").trim(),
        table: String(loggingTable?.value || "request_logs").trim(),
        username: String(loggingUsername?.value || "").trim(),
        password: String(loggingPassword?.value || "").trim(),
        migration_enabled: !!loggingMigrationEnabled?.checked,
      },
    },
  });

  const renderRuntime = async () => {
    await renderRuntimeData({
      ctx,
      permissionSet,
      runtimeStatus,
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
      syncVaultTLSControls,
      loggingStatus,
      settingsRenderLoggingStatusText: renderLoggingStatusText,
      storageLogs,
      storageActivity,
      storageEvents,
      storageBans,
      storageHotIndexDays,
      storageColdIndexDays,
      storageIndexesLimit,
      storageIndexesOffset,
      storageIndexesStream,
      setStorageIndexesOffset: (value) => { storageIndexesOffset = value; },
      settingsStorageRenderIndexes: renderStorageIndexes
    });
  };

  bindSettingsActions({
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
    securityLoginRateEnabled,
    securityLoginRateAttempts,
    securityLoginRateWindow,
    securityLoginRateBlock,
    renderRuntime,
    buildLoggingPayload,
    renderUpdateStatus
  });

  const selectedTab = currentTab.id;
  showTab(container, selectedTab);

  try {
    const meta = await ctx.api.get("/api/app/meta");
    const version = String(meta?.app_version || ctx.t("about.versionFallback"));
    aboutVersion.textContent = version;
    aboutVersionInline.textContent = `${ctx.t("about.version")}: ${version}`;
    if (aboutProjectLink) {
      const repo = String(meta?.repository_url || "").trim();
      aboutProjectLink.href = repo || "https://github.com/BerkutSolutions/tarinio";
    }
  } catch {
    aboutVersion.textContent = ctx.t("about.versionFallback");
    if (aboutProjectLink) {
      aboutProjectLink.href = "https://github.com/BerkutSolutions/tarinio";
    }
  }

  await renderRuntime();
}
