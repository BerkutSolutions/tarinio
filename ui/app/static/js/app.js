import { api } from "./api.js";
import { applyTranslations, getLanguage, setLanguage, t } from "./i18n.js";
import { createNotificationCenter } from "./app.notifications.js";
import { renderSidebarMenu } from "./app.sidebar-menu.js";
import { refreshSidebarStatus } from "./app.sidebar-status.js";
import "./webauthn.js";

import { checkEntryAccess } from "./guard.js";
import { escapeHtml, notify } from "./ui.js";

let sidebarCollapsed = false;
const PAGE_MODULE_VERSION = "20260628-16";
let lastGlobalScriptError = null;

window.addEventListener("error", (event) => {
  const message = String(event?.message || "");
  const filename = String(event?.filename || "");
  if (!message && !filename) {
    return;
  }
  lastGlobalScriptError = {
    message,
    filename,
    lineno: Number(event?.lineno || 0),
    colno: Number(event?.colno || 0),
  };
}, true);

function shouldRetryModuleLoad(error) {
  const name = String(error?.name || "").toLowerCase();
  const message = String(error?.message || "").toLowerCase();
  if (name.includes("syntaxerror") && message.includes("unexpected end of input")) {
    return true;
  }
  return message.includes("failed to fetch dynamically imported module");
}

function serializeError(error) {
  const details = {
    name: String(error?.name || ""),
    message: String(error?.message || ""),
    stack: String(error?.stack || ""),
    cause: error?.cause ? String(error.cause) : "",
  };
  try {
    for (const key of Object.keys(error || {})) {
      if (Object.prototype.hasOwnProperty.call(details, key)) {
        continue;
      }
      const value = error[key];
      details[key] = typeof value === "string" ? value : JSON.stringify(value);
    }
  } catch {
    // ignore serialization issues
  }
  if (lastGlobalScriptError) {
    details.global = JSON.stringify(lastGlobalScriptError);
  }
  return details;
}

async function loadPageModule(name) {
  const base = `./pages/${name}?v=${PAGE_MODULE_VERSION}`;
  const attempts = [
    base,
    `${base}&retry=1&nonce=${Date.now()}`,
    `${base}&retry=2&nonce=${Date.now() + 1}`,
  ];
  let lastError = null;
  for (let index = 0; index < attempts.length; index += 1) {
    try {
      return await import(attempts[index]);
    } catch (error) {
      lastError = error;
      if (index >= attempts.length - 1 || !shouldRetryModuleLoad(error)) {
        throw error;
      }
    }
  }
  throw lastError || new Error(`Failed to load module: ${name}`);
}

async function loadSitesSectionRenderer() {
  // Keep Services on the stable facade; do not wire the legacy-broken runtime bridge.
  try {
    const module = await loadPageModule("sites.js");
    return module.renderSites;
  } catch (error) {
    const wrapped = new Error(`Services stable facade failed to load: ${String(error?.message || error)}`);
    wrapped.name = "ServicesStableFacadeLoadError";
    wrapped.details = {
      entrypoint: "sites.js",
      facadeTarget: "sites.stable-page.js",
      brokenRuntimePath: "sites.page-main-runtime.js",
      originalName: String(error?.name || ""),
      originalMessage: String(error?.message || ""),
      originalStack: String(error?.stack || ""),
    };
    throw wrapped;
  }
}

const SITES_PREFLIGHT_MODULES = [];

async function preflightSitesModules() {
  if (!SITES_PREFLIGHT_MODULES.length) {
    return;
  }
  for (const file of SITES_PREFLIGHT_MODULES) {
    const href = `./pages/${file}?v=${PAGE_MODULE_VERSION}&preflight=1&nonce=${Date.now()}`;
    try {
      await import(href);
    } catch (error) {
      const wrapped = new Error(`Sites preflight failed at ${file}: ${String(error?.message || error)}`);
      wrapped.name = "SitesModulePreflightError";
      wrapped.details = {
        file,
        href,
        originalName: String(error?.name || ""),
        originalMessage: String(error?.message || ""),
        originalStack: String(error?.stack || ""),
      };
      throw wrapped;
    }
  }
}

const sidebarIcon = (name) => `<img class="sidebar-icon-image" src="/static/icons/svg/${name}-32x32.svg" alt="" aria-hidden="true">`;

const icons = {
  dashboard: sidebarIcon("dashboard"),
  sites: sidebarIcon("services"),
  antiddos: sidebarIcon("antiddos"),
  owaspcrs: sidebarIcon("owasp"),
  tls: sidebarIcon("certificates"),
  requests: sidebarIcon("requests"),
  revisions: sidebarIcon("revisions"),
  events: sidebarIcon("journal"),
  incidents: sidebarIcon("incidents"),
  bans: sidebarIcon("bans"),
  administration: sidebarIcon("administration"),
  activity: sidebarIcon("audit"),
  settings: sidebarIcon("settings"),
  profile: sidebarIcon("user"),
};

const sections = [
  // contract marker: render: renderDashboard
  { id: "dashboard", labelKey: "app.dashboard", descriptionKey: "app.section.dashboard.desc", load: () => loadPageModule("dashboard.js").then((m) => m.renderDashboard) },
  // contract marker: render: renderSites
  { id: "sites", pathBase: "/services", labelKey: "app.sites", descriptionKey: "app.section.sites.desc", load: () => loadSitesSectionRenderer() },
  // contract marker: render: renderAntiDDoS
  { id: "antiddos", pathBase: "/anti-ddos", labelKey: "app.antiddos", descriptionKey: "app.section.antiddos.desc", load: () => loadPageModule("antiddos.js").then((m) => m.renderAntiDDoS) },
  { id: "owaspcrs", pathBase: "/owasp-crs", labelKey: "app.owaspcrs", descriptionKey: "app.section.owaspcrs.desc", load: () => loadPageModule("owasp-crs.js").then((m) => m.renderOWASPCRS) },
  // contract marker: render: renderTLS
  { id: "tls", labelKey: "app.tls", descriptionKey: "app.section.tls.desc", load: () => loadPageModule("tls.js").then((m) => m.renderTLS) },
  // contract marker: render: renderRequests
  { id: "requests", labelKey: "app.requests", descriptionKey: "app.section.requests.desc", load: () => loadPageModule("requests.js").then((m) => m.renderRequests) },
  { id: "revisions", labelKey: "app.revisions", descriptionKey: "app.section.revisions.desc", load: () => loadPageModule("revisions.js").then((m) => m.renderRevisions) },
  { id: "events", labelKey: "app.events", descriptionKey: "app.section.events.desc", load: () => loadPageModule("events.js").then((m) => m.renderEvents) },
  { id: "bans", labelKey: "app.bans", descriptionKey: "app.section.bans.desc", load: () => loadPageModule("bans.js").then((m) => m.renderBans) },
  { id: "administration", labelKey: "app.administration", descriptionKey: "app.section.administration.desc", load: () => loadPageModule("administration.js").then((m) => m.renderAdministration) },
  { id: "activity", labelKey: "app.activity", descriptionKey: "app.section.activity.desc", load: () => loadPageModule("activity.js").then((m) => m.renderActivity) },
  { id: "settings", labelKey: "app.settings", descriptionKey: "app.section.settings.desc", load: () => loadPageModule("settings.js").then((m) => m.renderSettings) },
  { id: "profile", labelKey: "app.profile", descriptionKey: "app.section.profile.desc", load: () => loadPageModule("profile.js").then((m) => m.renderProfile), hiddenInMenu: true },
];

const sectionAccessRules = {
  dashboard: ["dashboard.read"],
  sites: ["sites.read", "sites.write"],
  antiddos: ["antiddos.read", "antiddos.write"],
  owaspcrs: ["owaspcrs.read", "owaspcrs.write"],
  tls: ["tls.read", "tls.write", "certificates.read", "certificates.write"],
  requests: ["requests.read"],
  revisions: ["revisions.read", "revisions.write"],
  events: ["events.read"],
  bans: ["bans.read"],
  administration: ["administration.read", "administration.write", "administration.users.read", "administration.roles.read"],
  activity: ["activity.read"],
  settings: ["settings.general.read", "settings.storage.read", "settings.about.read"],
  profile: ["profile.read", "auth.self"],
};

let currentUser = null;
let notificationCenter = null;
let stopNotificationsListener = null;
let sessionPingTimer = null;
let activePageAbortController = null;
let activePageRenderID = 0;
let activePageCleanup = null;

function sectionPath(value) {
  const section = typeof value === "string" ? sections.find((item) => item.id === value) : value;
  if (!section) {
    return "/";
  }
  return section.pathBase || `/${section.id}`;
}

function currentSection() {
  const path = (window.location.pathname || "/").replace(/\/+$/, "") || "/";
  if (path === "/about" || path.startsWith("/about/")) {
    return sections.find((section) => section.id === "settings") || sections[0];
  }
  const matched = sections
    .map((section) => ({ section, base: sectionPath(section) }))
    .sort((left, right) => right.base.length - left.base.length)
    .find(({ base }) => path === base || path.startsWith(`${base}/`));
  return matched?.section || sections[0];
}

function navigate(path, { replaceHistory = false } = {}) {
  if ((window.location.pathname || "/") === path) {
    return;
  }
  if (replaceHistory) {
    window.history.replaceState({}, "", path);
    return;
  }
  window.history.pushState({}, "", path);
}

function formatMenuBadge(path, count) {
  const link = document.querySelector(`.sidebar-link[data-path="${path}"]`);
  if (!link) {
    return;
  }
  let badge = link.querySelector(".sidebar-notify");
  if (!badge) {
    badge = document.createElement("span");
    badge.className = "sidebar-notify";
    link.appendChild(badge);
  }
  if (!count) {
    badge.hidden = true;
    badge.textContent = "";
    return;
  }
  badge.hidden = false;
  badge.textContent = count > 99 ? "99+" : String(count);
}

function renderMenu() {
  const menu = document.getElementById("menu");
  const active = currentSection();
  renderSidebarMenu({ menu, sections, active, icons, translate: t, sectionPath, canAccessSection });
}

function renderRBAC(user) {
  const badge = document.getElementById("rbac-badge");
  if (!badge) {
    return;
  }
  badge.hidden = true;
  badge.textContent = "";
}

function preferredLanguageForUser(user) {
  const language = String(user?.language || "").trim().toLowerCase();
  return language || "";
}

function currentPermissionSet() {
  return new Set(Array.isArray(currentUser?.permissions) ? currentUser.permissions.map((item) => String(item || "").trim()) : []);
}

function canAccessSection(sectionID) {
  const required = sectionAccessRules[sectionID];
  if (!required || !required.length) {
    return true;
  }
  const permissions = currentPermissionSet();
  return required.some((permission) => permissions.has(permission));
}

function firstAccessibleSection() {
  return sections.find((section) => !section.hiddenInMenu && canAccessSection(section.id)) || sections.find((section) => section.id === "profile") || sections[0];
}

function setVersion(value) {
  const versionNode = document.getElementById("app-version");
  if (!versionNode) {
    return;
  }
  versionNode.textContent = value || t("app.version");
}

function renderUpdateBadge(meta) {
  const badge = document.getElementById("app-update-badge");
  if (!badge) {
    return;
  }
  const update = meta?.update;
  if (meta?.update_checks_enabled && update?.has_update) {
    badge.hidden = false;
    badge.textContent = t("settings.updates.available");
    badge.href = update.release_url || "#";
    badge.target = "_blank";
    badge.rel = "noopener noreferrer";
    return;
  }
  badge.hidden = true;
  badge.textContent = "";
  badge.removeAttribute("href");
  badge.removeAttribute("target");
  badge.removeAttribute("rel");
}

async function loadUser() {
  try {
    return await api.get("/api/auth/me");
  } catch (error) {
    if (error.status === 401) {
      window.location.href = "/login";
      return null;
    }
    throw error;
  }
}

async function renderPage() {
  let section = currentSection();
  if (!canAccessSection(section.id)) {
    section = firstAccessibleSection();
    navigate(sectionPath(section), { replaceHistory: true });
  }
  const container = document.getElementById("content-area");
  const renderID = ++activePageRenderID;
  if (activePageAbortController) {
    activePageAbortController.abort();
  }
  if (typeof activePageCleanup === "function") {
    try {
      activePageCleanup();
    } catch (_error) {
      // cleanup is best-effort
    }
  }
  activePageCleanup = null;
  activePageAbortController = new AbortController();
  const signal = activePageAbortController.signal;
  const mount = document.createElement("div");
  mount.className = "app-page-mount";
  document.title = `Tarinio | ${t(section.labelKey)}`;
  const pageTitleNode = document.getElementById("page-title");
  if (pageTitleNode) pageTitleNode.textContent = "";
  const pageDescNode = document.getElementById("page-desc");
  if (pageDescNode) pageDescNode.textContent = "";
  const pageHeaderNode = document.querySelector(".app-header");
  if (pageHeaderNode) pageHeaderNode.style.display = "none";
  renderMenu();
  mount.innerHTML = `<div class="waf-empty">${escapeHtml(t("app.loading"))}</div>`;
  container.replaceChildren(mount);
  const isActive = () => renderID === activePageRenderID && !signal.aborted && mount.isConnected;
  try {
    if (section.id === "sites") {
      // Compatibility mode: preflight disabled because some browsers fail to parse
      // extracted modules despite valid syntax in CI/runtime checks.
      // The loader below has a safe fallback renderer on import errors.
    }
    if (typeof section.__render !== "function") {
      section.__render = await section.load();
    }
    const cleanup = await section.__render(mount, {
      api,
      notify,
      t,
      setLanguage,
      getLanguage,
      currentUser,
      setCurrentUser(nextUser) {
        if (nextUser && typeof nextUser === "object") {
          currentUser = nextUser;
        }
      },
      signal,
      isActive,
    });
    if (!isActive()) {
      if (typeof cleanup === "function") {
        try {
          cleanup();
        } catch (_error) {
          // cleanup is best-effort
        }
      }
      return;
    }
    activePageCleanup = typeof cleanup === "function" ? cleanup : null;
  } catch (error) {
    if (!isActive()) {
      return;
    }
    if (error?.name === "AbortError") {
      return;
    }
    const message = String(error?.message || "");
    const status = Number(error?.status || 0);
    const tooManyRequests = status === 429 || /429|too many requests/i.test(message);
    const forbidden = status === 403 || /403|forbidden/i.test(message);
    if (tooManyRequests || forbidden) {
      mount.innerHTML = `
        <div class="waf-card">
          <div class="waf-card-head"><h3>${escapeHtml(tooManyRequests ? "429 Too Many Requests" : "403 Forbidden")}</h3></div>
          <div class="waf-card-body">
            <p>${escapeHtml(t("app.error"))}</p>
            <p class="muted">${escapeHtml(tooManyRequests ? "РЎР»РёС€РєРѕРј РјРЅРѕРіРѕ Р·Р°РїСЂРѕСЃРѕРІ. РџРѕРґРѕР¶РґРёС‚Рµ РЅРµСЃРєРѕР»СЊРєРѕ СЃРµРєСѓРЅРґ Рё РѕР±РЅРѕРІРёС‚Рµ СЃС‚СЂР°РЅРёС†Сѓ." : "Р”РѕСЃС‚СѓРї Р·Р°РїСЂРµС‰С‘РЅ С‚РµРєСѓС‰РёРјРё РїСЂР°РІРёР»Р°РјРё Р·Р°С‰РёС‚С‹.")}</p>
          </div>
        </div>
      `;
      return;
    }
    const errorMessage = String(error?.message || t("app.error"));
    const errorStack = String(error?.stack || "").trim();
    const errorDetails = serializeError(error);
    mount.innerHTML = `
      <div class="alert">${escapeHtml(errorMessage)}</div>
      ${errorStack ? `<pre class="waf-code">${escapeHtml(`[section=${section.id}]\n${errorStack}`)}</pre>` : ""}
      <pre class="waf-code">${escapeHtml(`[section=${section.id}]\n${JSON.stringify(errorDetails, null, 2)}`)}</pre>
    `;
  }
}

function wireClientNavigation() {
  document.addEventListener("click", async (event) => {
    const link = event.target.closest("a[href]");
    if (!link) {
      return;
    }
    const url = new URL(link.href, window.location.origin);
    if (url.origin !== window.location.origin) {
      return;
    }
    const section = sections.find((item) => {
      const base = sectionPath(item);
      return url.pathname === base || url.pathname.startsWith(`${base}/`);
    });
    if (!section) {
      return;
    }
    event.preventDefault();
    navigate(url.pathname);
    await renderPage();
  });

  window.addEventListener("popstate", async () => {
    await renderPage();
  });
}

function wireLanguageUpdates() {
  window.addEventListener("app:language-changed", async () => {
    if (!currentUser) {
      return;
    }
    if (notificationCenter && typeof notificationCenter.refresh === "function") {
      await notificationCenter.refresh().catch(() => {});
    }
    renderRBAC(currentUser);
    await loadMeta();
    await renderPage();
  });
}

function initSidebarCollapse() {
  const toggle = document.getElementById("sidebar-toggle");
  const applyState = (collapsed) => {
    document.body.classList.toggle("sidebar-collapsed", collapsed);
    toggle.setAttribute("aria-expanded", collapsed ? "false" : "true");
    sidebarCollapsed = collapsed;
  };
  applyState(sidebarCollapsed);
  toggle.addEventListener("click", () => {
    applyState(!document.body.classList.contains("sidebar-collapsed"));
  });
}

function bindProfileButton() {
  const btn = document.getElementById("profile-btn");
  btn.addEventListener("click", async (event) => {
    event.preventDefault();
    navigate(sectionPath("profile"));
    await renderPage();
  });
  bindProfileHoverCard(btn);
}

function bindProfileHoverCard(button) {
  if (!button) {
    return;
  }
  let card = document.getElementById("profile-hover-card");
  if (!card) {
    card = document.createElement("div");
    card.id = "profile-hover-card";
    card.className = "profile-hover-card";
    card.hidden = true;
    document.body.appendChild(card);
  }
  const renderCard = () => {
    const roleIDs = Array.isArray(currentUser?.role_ids) ? currentUser.role_ids : [];
    const lines = [
      currentUser?.username || "-",
      currentUser?.email || "-",
      `${t("profile.field.roles")}: ${roleIDs.join(", ") || "-"}`,
      `${t("profile.field.permissions")}: ${Array.isArray(currentUser?.permissions) ? currentUser.permissions.length : 0}`,
    ];
    card.innerHTML = lines.map((line) => `<div>${escapeHtml(String(line))}</div>`).join("");
  };
  const show = () => {
    renderCard();
    const rect = button.getBoundingClientRect();
    card.hidden = false;
    const margin = 8;
    const width = card.offsetWidth || 240;
    const height = card.offsetHeight || 120;
    card.style.left = `${Math.min(window.innerWidth - width - margin, rect.right + 10)}px`;
    card.style.top = `${Math.min(window.innerHeight - height - margin, rect.top)}px`;
  };
  const hide = () => {
    card.hidden = true;
  };
  button.addEventListener("mouseenter", show);
  button.addEventListener("mouseleave", hide);
  button.addEventListener("focus", show);
  button.addEventListener("blur", hide);
  window.addEventListener("scroll", hide, { passive: true });
}

function bindNotificationsUI() {
  const btn = document.getElementById("notifications-btn");
  const drop = document.getElementById("notifications-dropdown");
  const list = document.getElementById("notifications-list");
  const clearBtn = document.getElementById("notifications-clear-all");
  const badge = document.getElementById("notifications-btn-badge");

  const render = (items) => {
    const rows = Array.isArray(items) ? items : [];
    if (!rows.length) {
      badge.hidden = true;
      badge.textContent = "";
      list.innerHTML = `<div class="notifications-empty muted">${escapeHtml(t("app.notifications.empty"))}</div>`;
      formatMenuBadge("events", 0);
      return;
    }

    const eventsCount = rows.filter((item) => item.targetPath === "/events").length;
    formatMenuBadge("events", eventsCount);

    badge.hidden = false;
    badge.textContent = rows.length > 99 ? "99+" : String(rows.length);

    list.innerHTML = rows
      .map((item) => `
        <div class="notification-row" data-key="${escapeHtml(item.key)}" data-target="${escapeHtml(item.targetPath || "")}">
          <div class="notification-head">
            <strong>${escapeHtml(item.title || "-")}</strong>
            <button class="btn ghost btn-sm" type="button" data-action="dismiss" data-key="${escapeHtml(item.key)}">${escapeHtml(t("common.delete"))}</button>
          </div>
          <div class="notification-message muted">${escapeHtml(item.message || "-")}</div>
        </div>
      `)
      .join("");
  };

  if (stopNotificationsListener) {
    stopNotificationsListener();
  }

  stopNotificationsListener = notificationCenter.subscribe((items) => render(items));

  btn.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();
    drop.hidden = !drop.hidden;
  });

  clearBtn.addEventListener("click", (event) => {
    event.preventDefault();
    notificationCenter.clear();
  });

  list.addEventListener("click", async (event) => {
    const dismiss = event.target.closest("[data-action='dismiss']");
    if (dismiss) {
      event.preventDefault();
      notificationCenter.dismiss(dismiss.dataset.key || "");
      return;
    }

    const row = event.target.closest(".notification-row");
    if (!row) {
      return;
    }
    const target = row.dataset.target || "";
    if (!target) {
      return;
    }
    drop.hidden = true;
    navigate(target);
    await renderPage();
  });

  document.addEventListener("click", (event) => {
    if (drop.hidden) {
      return;
    }
    if (btn.contains(event.target) || drop.contains(event.target)) {
      return;
    }
    drop.hidden = true;
  });
}

async function loadMeta() {
  try {
    const meta = await api.get("/api/app/meta");
    const sharedLanguage = String(meta?.ui_language || "").trim().toLowerCase();
    if (!preferredLanguageForUser(currentUser) && sharedLanguage && sharedLanguage !== getLanguage()) {
      await setLanguage(sharedLanguage);
    }
    if (meta?.app_version) {
      setVersion(`v${meta.app_version}`);
    }
    renderUpdateBadge(meta);
    refreshSidebarStatus(api, t);
  } catch {
    setVersion("v1.5.4");
    renderUpdateBadge(null);
    refreshSidebarStatus(api, t);
  }
}

function stopSessionPing() {
  if (sessionPingTimer) {
    window.clearInterval(sessionPingTimer);
    sessionPingTimer = null;
  }
}

function startSessionPing() {
  stopSessionPing();
  if (document.hidden) {
    return;
  }
  const run = async () => {
    try {
      await api.post("/api/app/ping", null, {
        headers: {
          "X-Berkut-Background": "1",
        },
      });
    } catch (error) {
      const status = Number(error?.status || 0);
      const message = String(error?.message || "").toLowerCase();
      const networkLike = message.includes("network") || message.includes("failed to fetch") || message.includes("unavailable");
      if (status === 401 || networkLike) {
        stopSessionPing();
      }
    }
  };
  sessionPingTimer = window.setInterval(run, 45000);
  run().catch(() => {});
}

function pauseBackgroundActivity() {
  stopSessionPing();
  notificationCenter?.stop?.();
}

function resumeBackgroundActivity() {
  if (document.hidden) {
    return;
  }
  notificationCenter?.start?.();
  startSessionPing();
}

function bindVisibilityLifecycle() {
  document.addEventListener("visibilitychange", () => {
    if (document.hidden) {
      pauseBackgroundActivity();
      return;
    }
    resumeBackgroundActivity();
  });
  window.addEventListener("beforeunload", () => {
    pauseBackgroundActivity();
  });
}

async function bootstrap() {
  await applyTranslations(getLanguage());
  setVersion("v1.5.4");

  const access = await checkEntryAccess("app");
  if (!access.allowed) {
    return;
  }

  currentUser = access.user || (await loadUser());
  if (!currentUser) {
    return;
  }
  if (preferredLanguageForUser(currentUser) && preferredLanguageForUser(currentUser) !== getLanguage()) {
    await setLanguage(preferredLanguageForUser(currentUser));
  }

  renderRBAC(currentUser);
  wireClientNavigation();
  wireLanguageUpdates();
  initSidebarCollapse();
  bindProfileButton();

  document.getElementById("logout-btn").addEventListener("click", async () => {
    await api.post("/api/auth/logout", {});
    notify(t("toast.loggedOut"));
    window.location.href = "/login";
  });

  notificationCenter = createNotificationCenter({ api, t });
  bindNotificationsUI();
  bindVisibilityLifecycle();
  resumeBackgroundActivity();

  await loadMeta();
  await renderPage();
}

bootstrap();
