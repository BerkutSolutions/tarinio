import { availableLanguages } from "../i18n.js";
import { escapeHtml } from "../ui.js";

export function renderLanguageOptions() {
  return availableLanguages()
    .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.label)}</option>`)
    .join("");
}

export function activeTabFromPath(settingsTabs) {
  const path = (window.location.pathname || "").replace(/\/+$/, "") || "/";
  const matched = settingsTabs.find((tab) => path === tab.path || path.startsWith(`${tab.path}/`));
  return matched?.id || "general";
}

export function permissionSet(ctx) {
  return new Set(Array.isArray(ctx?.currentUser?.permissions) ? ctx.currentUser.permissions.map((item) => String(item || "").trim()) : []);
}

export function availableTabs(ctx, settingsTabs) {
  const permissions = permissionSet(ctx);
  return settingsTabs.filter((tab) => (tab.permissions || []).some((permission) => permissions.has(permission)));
}

export function showTab(root, tabID) {
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

export function renderUpdateStatus(ctx, element, payload) {
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
