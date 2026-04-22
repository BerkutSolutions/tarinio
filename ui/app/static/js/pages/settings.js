import { availableLanguages } from "../i18n.js";
import { escapeHtml } from "../ui.js";

const SETTINGS_TABS = [
  { id: "general", path: "/settings/general", labelKey: "settings.tabs.general", permissions: ["settings.general.read", "settings.general.write"] },
  { id: "storage", path: "/settings/storage", labelKey: "settings.tabs.storage", permissions: ["settings.storage.read", "settings.storage.write"] },
  { id: "about", path: "/settings/about", labelKey: "settings.tabs.about", permissions: ["settings.about.read"] },
];

let runtimeAutoCheckTimer = null;

function renderLanguageOptions() {
  return availableLanguages()
    .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.label)}</option>`)
    .join("");
}

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

function permissionSet(ctx) {
  return new Set(Array.isArray(ctx?.currentUser?.permissions) ? ctx.currentUser.permissions.map((item) => String(item || "").trim()) : []);
}

function availableTabs(ctx) {
  const permissions = permissionSet(ctx);
  return SETTINGS_TABS.filter((tab) => (tab.permissions || []).some((permission) => permissions.has(permission)));
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
  let storageIndexesOffset = 0;
  const storageIndexesLimit = 10;
  const tabs = availableTabs(ctx);
  const currentTab = tabs.find((tab) => tab.id === activeTabFromPath()) || tabs[0] || SETTINGS_TABS.find((tab) => tab.id === "about");
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
            </div>
            <div class="waf-note">${escapeHtml(ctx.t("settings.storage.note"))}</div>
            <div class="waf-actions">
              <button id="settings-storage-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
            <div id="settings-storage-indexes"></div>
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
  const storageSave = container.querySelector("#settings-storage-save");
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
          await ctx.api.delete(`/api/settings/runtime/storage-indexes?date=${encodeURIComponent(day)}`);
          setAlert(ctx.t("settings.saved"), true);
          await renderRuntime();
        } catch (error) {
          setAlert(error?.message || ctx.t("common.error"));
        }
      });
    });
  };

  const renderRuntime = async () => {
    const canReadGeneral = permissionSet(ctx).has("settings.general.read") || permissionSet(ctx).has("settings.general.write");
    const canReadStorage = permissionSet(ctx).has("settings.storage.read") || permissionSet(ctx).has("settings.storage.write");
    try {
      let runtime = null;
      if (canReadGeneral) {
        runtime = await ctx.api.get("/api/settings/runtime");
        const mode = String(runtime?.deployment_mode || "-");
        runtimeStatus.textContent = ctx.t("settings.runtime.loaded", { mode });
        if (languageSelect) {
          languageSelect.value = String(runtime?.language || ctx.getLanguage?.() || "en");
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
          runtimeAutoCheckTimer = window.setInterval(async () => {
            try {
              const result = await ctx.api.post("/api/settings/runtime/check-updates", { update_checks_enabled: true, manual: false });
              renderUpdateStatus(ctx, updateStatus, result || {});
            } catch {
              // keep last known status silently
            }
          }, 60 * 60 * 1000);
        } else {
          clearRuntimeAutoCheckTimer();
        }
      }
      if (canReadStorage) {
        const storage = runtime?.storage || {};
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
        const indexesPayload = await ctx.api.get(`/api/settings/runtime/storage-indexes?storage_indexes_limit=${storageIndexesLimit}&storage_indexes_offset=${storageIndexesOffset}`).catch(() => ({ items: [], total: 0, limit: storageIndexesLimit, offset: storageIndexesOffset }));
        storageIndexesOffset = Number(indexesPayload?.offset || 0);
        renderStorageIndexes(indexesPayload);
      }
    } catch {
      runtimeStatus.textContent = ctx.t("settings.runtime.shell");
      updateStatus.textContent = ctx.t("settings.updates.notAvailable");
      renderStorageIndexes({ items: [], total: 0, limit: storageIndexesLimit, offset: storageIndexesOffset });
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
        },
      };
      await ctx.api.put("/api/settings/runtime", payload);
      setAlert(ctx.t("settings.saved"), true);
      await renderRuntime();
    } catch (error) {
      setAlert(error?.message || ctx.t("common.error"));
    }
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
