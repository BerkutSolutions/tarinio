import { api } from "./api.js";
import { applyTranslations, getLanguage, setLanguage, t } from "./i18n.js";
import { createNotificationCenter } from "./app.notifications.js";
import "./webauthn.js";

import { checkEntryAccess } from "./guard.js";
import { escapeHtml, notify } from "./ui.js";

let sidebarCollapsed = false;

const icons = {
  dashboard: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 13h6V4H4v9Zm0 7h6v-5H4v5Zm8 0h8v-9h-8v9Zm0-18v7h8V2h-8Z"/></svg>',
  sites: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 10.5 12 3l9 7.5V21h-7v-6H10v6H3v-10.5Z"/></svg>',
  antiddos: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 1 4 5v6c0 5 3.4 9.7 8 11 4.6-1.3 8-6 8-11V5l-8-4Zm0 3.2L17 6.7v4.2c0 3.8-2.2 7.2-5 8.5-2.8-1.3-5-4.7-5-8.5V6.7l5-2.5Zm-1.2 3.3v3H8.8v2h2V15h2.4v-2.5h2v-2h-2v-3h-2.4Z"/></svg>',
  owaspcrs: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 1 4 5v6c0 5 3.4 9.7 8 11 4.6-1.3 8-6 8-11V5l-8-4Zm0 3.2L17 6.7v4.2c0 3.8-2.2 7.2-5 8.5-2.8-1.3-5-4.7-5-8.5V6.7l5-2.5Zm-3 4.8h6v2H9v-2Zm0 4h6v2H9v-2Z"/></svg>',
  tls: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 1 4 5v6c0 5 3.4 9.7 8 11 4.6-1.3 8-6 8-11V5l-8-4Zm0 10.2a2.3 2.3 0 1 1 0 4.6 2.3 2.3 0 0 1 0-4.6Zm4 6.8H8v-1.2c0-1.8 1.8-2.8 4-2.8s4 1 4 2.8V18Z"/></svg>',
  requests: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v2H3V5Zm0 4h18v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9Zm4 3v2h4v-2H7Zm6 0v2h4v-2h-4Z"/></svg>',
  revisions: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 3h14a2 2 0 0 1 2 2v4H3V5a2 2 0 0 1 2-2Zm-2 8h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-8Zm4 2v2h4v-2H7Zm6 0h4v2h-4v-2Zm-6 4v2h10v-2H7Z"/></svg>',
  events: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M13 2 4 14h6l-1 8 9-12h-6l1-8Z"/></svg>',
  bans: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 1 4 5v6c0 5 3.4 9.7 8 11 4.6-1.3 8-6 8-11V5l-8-4Zm0 3.2L17 6.7v4.2c0 3.8-2.2 7.2-5 8.5-2.8-1.3-5-4.7-5-8.5V6.7l5-2.5Zm-3 4.8h6v2H9V9Zm0 4h6v2H9v-2Z"/></svg>',
  administration: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 12a5 5 0 1 0-5-5 5 5 0 0 0 5 5Zm0 2c-4.4 0-8 2.2-8 5v1h16v-1c0-2.8-3.6-5-8-5Z"/></svg>',
  activity: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 4h16v2H4V4Zm0 7h10v2H4v-2Zm0 7h16v2H4v-2Zm12-8h4v4h-4v-4Z"/></svg>',
  settings: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M19.14 12.94a7.96 7.96 0 0 0 .06-.94 7.96 7.96 0 0 0-.06-.94l2.03-1.58a.5.5 0 0 0 .12-.64l-1.92-3.32a.5.5 0 0 0-.6-.22l-2.39.96a7.48 7.48 0 0 0-1.63-.94l-.36-2.54a.5.5 0 0 0-.5-.42h-3.84a.5.5 0 0 0-.5.42l-.36 2.54c-.58.22-1.12.53-1.63.94l-2.39-.96a.5.5 0 0 0-.6.22L2.7 8.84a.5.5 0 0 0 .12.64l2.03 1.58a7.96 7.96 0 0 0-.06.94c0 .32.02.63.06.94l-2.03 1.58a.5.5 0 0 0-.12.64l1.92 3.32c.13.22.39.31.6.22l2.39-.96c.5.41 1.05.73 1.63.94l.36 2.54c.04.24.25.42.5.42h3.84c.25 0 .46-.18.5-.42l.36-2.54c.58-.21 1.13-.53 1.63-.94l2.39.96c.22.09.47 0 .6-.22l1.92-3.32a.5.5 0 0 0-.12-.64l-2.03-1.58ZM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2Z"/></svg>',
  about: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 7h2V5h-2v2Zm0 12h2V9h-2v10ZM12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2Z"/></svg>',
  profile: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 12a5 5 0 1 0-5-5 5 5 0 0 0 5 5Zm0 2c-4.4 0-8 2.2-8 5v1h16v-1c0-2.8-3.6-5-8-5Z"/></svg>',
};

const sections = [
  // contract marker: render: renderDashboard
  { id: "dashboard", labelKey: "app.dashboard", descriptionKey: "app.section.dashboard.desc", load: () => import("./pages/dashboard.js").then((m) => m.renderDashboard) },
  // contract marker: render: renderSites
  { id: "sites", pathBase: "/services", labelKey: "app.sites", descriptionKey: "app.section.sites.desc", load: () => import("./pages/sites.js").then((m) => m.renderSites) },
  // contract marker: render: renderAntiDDoS
  { id: "antiddos", pathBase: "/anti-ddos", labelKey: "app.antiddos", descriptionKey: "app.section.antiddos.desc", load: () => import("./pages/antiddos.js").then((m) => m.renderAntiDDoS) },
  { id: "owaspcrs", pathBase: "/owasp-crs", labelKey: "app.owaspcrs", descriptionKey: "app.section.owaspcrs.desc", load: () => import("./pages/owasp-crs.js").then((m) => m.renderOWASPCRS) },
  // contract marker: render: renderTLS
  { id: "tls", labelKey: "app.tls", descriptionKey: "app.section.tls.desc", load: () => import("./pages/tls.js").then((m) => m.renderTLS) },
  // contract marker: render: renderRequests
  { id: "requests", labelKey: "app.requests", descriptionKey: "app.section.requests.desc", load: () => import("./pages/requests.js").then((m) => m.renderRequests) },
  { id: "revisions", labelKey: "app.revisions", descriptionKey: "app.section.revisions.desc", load: () => import("./pages/revisions.js").then((m) => m.renderRevisions) },
  { id: "events", labelKey: "app.events", descriptionKey: "app.section.events.desc", load: () => import("./pages/events.js").then((m) => m.renderEvents) },
  { id: "bans", labelKey: "app.bans", descriptionKey: "app.section.bans.desc", load: () => import("./pages/bans.js").then((m) => m.renderBans) },
  { id: "administration", labelKey: "app.administration", descriptionKey: "app.section.administration.desc", load: () => import("./pages/administration.js").then((m) => m.renderAdministration) },
  { id: "activity", labelKey: "app.activity", descriptionKey: "app.section.activity.desc", load: () => import("./pages/activity.js").then((m) => m.renderActivity) },
  { id: "settings", labelKey: "app.settings", descriptionKey: "app.section.settings.desc", load: () => import("./pages/settings.js").then((m) => m.renderSettings) },
  { id: "profile", labelKey: "app.profile", descriptionKey: "app.section.profile.desc", load: () => import("./pages/profile.js").then((m) => m.renderProfile), hiddenInMenu: true },
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
  menu.innerHTML = sections
    .filter((section) => !section.hiddenInMenu && canAccessSection(section.id))
    .map((section) => `
      <a class="sidebar-link ${section.id === active.id ? "active" : ""}" href="${sectionPath(section)}" data-path="${section.id}" title="${t(section.labelKey)}">
        <span class="sidebar-link-icon">${icons[section.id] || ""}</span>
        <span class="sidebar-link-label">${t(section.labelKey)}</span>
      </a>
    `)
    .join("");
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
            <p class="muted">${escapeHtml(tooManyRequests ? "Слишком много запросов. Подождите несколько секунд и обновите страницу." : "Доступ запрещён текущими правилами защиты.")}</p>
          </div>
        </div>
      `;
      return;
    }
    mount.innerHTML = `<div class="alert">${escapeHtml(error.message || t("app.error"))}</div>`;
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
  } catch {
    setVersion("v2.0.12");
    renderUpdateBadge(null);
  }
}

function startSessionPing() {
  if (sessionPingTimer) {
    window.clearInterval(sessionPingTimer);
    sessionPingTimer = null;
  }
  const run = async () => {
    try {
      await api.post("/api/app/ping", {}, { headers: { "X-Berkut-Background": "1" } });
    } catch (error) {
      const status = Number(error?.status || 0);
      const message = String(error?.message || "").toLowerCase();
      const networkLike = message.includes("network") || message.includes("failed to fetch") || message.includes("unavailable");
      if (status === 401 || networkLike) {
        if (sessionPingTimer) {
          window.clearInterval(sessionPingTimer);
          sessionPingTimer = null;
        }
      }
    }
  };
  sessionPingTimer = window.setInterval(run, 45000);
  run().catch(() => {});
}

async function bootstrap() {
  await applyTranslations(getLanguage());
  setVersion("v2.0.12");

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
  notificationCenter.start();
  startSessionPing();

  await loadMeta();
  await renderPage();
}

bootstrap();



