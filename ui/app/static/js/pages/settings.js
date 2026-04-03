import { loadPreferences, savePreferences } from "../preferences.js";
import { escapeHtml } from "../ui.js";

const SETTINGS_TABS = [
  { id: "general", path: "/settings/general", labelKey: "settings.tabs.general" },
  { id: "about", path: "/settings/about", labelKey: "settings.tabs.about" },
];

let runtimeAutoCheckTimer = null;

function clearRuntimeAutoCheckTimer() {
  if (runtimeAutoCheckTimer) {
    window.clearInterval(runtimeAutoCheckTimer);
    runtimeAutoCheckTimer = null;
  }
}

function activeTabFromPath() {
  const path = (window.location.pathname || "").replace(/\/+$/, "") || "/";
  const matched = SETTINGS_TABS.find((tab) => path === tab.path || path.startsWith(`${tab.path}/`));
  return matched?.id || "general";
}

function showTab(root, tabID) {
  root.querySelectorAll("[data-settings-tab-link]").forEach((link) => {
    const active = String(link.getAttribute("data-settings-tab-link") || "") === tabID;
    link.classList.toggle("active", active);
    link.setAttribute("aria-current", active ? "page" : "false");
  });
  root.querySelectorAll("[data-settings-panel]").forEach((panel) => {
    const active = String(panel.getAttribute("data-settings-panel") || "") === tabID;
    panel.hidden = !active;
  });
}

function renderUpdateStatus(ctx, element, payload) {
  if (!element) {
    return;
  }
  const result = payload?.update || payload?.result || payload || {};
  const checkedAt = String(result?.checked_at || result?.checkedAt || "").trim();
  const hasUpdate = !!(result?.has_update || result?.hasUpdate);
  const latest = String(result?.latest_version || result?.latestVersion || "-").trim();
  const releaseURL = String(result?.release_url || result?.releaseURL || "").trim();

  element.classList.remove("banner", "settings-updates-banner", "muted");
  element.textContent = "";
  element.innerHTML = "";

  if (!checkedAt) {
    element.classList.add("muted");
    element.textContent = ctx.t("settings.updates.notChecked");
    return;
  }

  if (!hasUpdate) {
    element.classList.add("muted");
    element.textContent = `${ctx.t("settings.updates.noUpdates")} (${checkedAt})`;
    return;
  }

  element.classList.add("banner", "settings-updates-banner");
  const text = document.createElement("span");
  text.textContent = `${ctx.t("settings.updates.availableVersion", { version: latest })} (${checkedAt})`;
  element.appendChild(text);

  if (releaseURL) {
    const link = document.createElement("a");
    link.className = "btn btn-sm primary";
    link.href = releaseURL;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.textContent = ctx.t("settings.updates.openRelease");
    element.appendChild(link);
  }
}

export async function renderSettings(container, ctx) {
  clearRuntimeAutoCheckTimer();

  container.innerHTML = `
    <div class="waf-page-stack" id="settings-page">
      <div id="settings-alert" class="alert" hidden></div>

      <section class="waf-card settings-tabs-card">
        <div class="waf-card-body">
          <div class="tabs browser-tabs settings-tabs" id="settings-tabs" role="tablist" aria-label="${escapeHtml(ctx.t("settings.title"))}">
            ${SETTINGS_TABS.map((tab) => `
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
                    <option value="ru">RU</option>
                    <option value="en">EN</option>
                  </select>
                </div>
              </div>

              <div class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.runtime.title"))}</div>
                </div>
                <div class="waf-note" id="settings-runtime-status">${escapeHtml(ctx.t("settings.runtime.shell"))}</div>
                <div class="waf-note" id="settings-about-version-inline">${escapeHtml(ctx.t("about.version"))}: -</div>
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
                <div class="waf-note" id="settings-update-status">${escapeHtml(ctx.t("settings.updates.notChecked"))}</div>
              </div>
            </div>
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
                  <a class="btn primary btn-sm" id="settings-about-project-link" href="https://github.com/BerkutSolutions/" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.project"))}</a>
                  <a class="btn ghost btn-sm" href="https://github.com/BerkutSolutions/" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.profile"))}</a>
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
  const languageSelect = container.querySelector("#settings-language-select");
  const runtimeSave = container.querySelector("#settings-runtime-save");
  const prefs = loadPreferences();
  if (languageSelect) {
    languageSelect.value = String(prefs?.language || ctx.getLanguage?.() || "ru");
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

  const renderRuntime = async () => {
    try {
      const runtime = await ctx.api.get("/api/settings/runtime");
      const mode = String(runtime?.deployment_mode || "-");
      runtimeStatus.textContent = ctx.t("settings.runtime.loaded", { mode });
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
        runtimeAutoCheckTimer = window.setInterval(async () => {
          try {
            const result = await ctx.api.post("/api/settings/runtime/check-updates", { update_checks_enabled: true, manual: false });
            renderUpdateStatus(ctx, updateStatus, result || {});
          } catch {
            // keep last known status silently
          }
        }, 5 * 60 * 1000);
      } else {
        clearRuntimeAutoCheckTimer();
      }
    } catch {
      runtimeStatus.textContent = ctx.t("settings.runtime.shell");
      updateStatus.textContent = ctx.t("settings.updates.notAvailable");
      clearRuntimeAutoCheckTimer();
    }
  };

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

  runtimeSave?.addEventListener("click", async () => {
    setAlert("");
    try {
      const nextLanguage = String(languageSelect?.value || "ru");
      savePreferences({ language: nextLanguage });
      const currentLanguage = String(ctx.getLanguage?.() || "ru");
      const languageChanged = nextLanguage !== currentLanguage;
      if (languageChanged && typeof ctx.setLanguage === "function") {
        await ctx.setLanguage(nextLanguage);
      }
      const payload = { update_checks_enabled: !!updatesEnabled?.checked };
      const result = await ctx.api.put("/api/settings/runtime", payload);
      renderUpdateStatus(ctx, updateStatus, result || {});
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
  });

  const selectedTab = activeTabFromPath();
  showTab(container, selectedTab);

  try {
    const meta = await ctx.api.get("/api/app/meta");
    const version = String(meta?.app_version || ctx.t("about.versionFallback"));
    aboutVersion.textContent = version;
    aboutVersionInline.textContent = `${ctx.t("about.version")}: ${version}`;
    if (aboutProjectLink) {
      const repo = String(meta?.repository_url || "").trim();
      aboutProjectLink.href = repo || "https://github.com/BerkutSolutions/";
    }
  } catch {
    aboutVersion.textContent = ctx.t("about.versionFallback");
    if (aboutProjectLink) {
      aboutProjectLink.href = "https://github.com/BerkutSolutions/";
    }
  }

  await renderRuntime();
}
