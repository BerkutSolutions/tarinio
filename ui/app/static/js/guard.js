function originForProtocol(protocol) {
  const host = window.location.hostname || "localhost";
  const port = window.location.port ? `:${window.location.port}` : "";
  return `${protocol}//${host}${port}`;
}

function httpsUrl(path = "/") {
  return `${originForProtocol("https:")}${path}`;
}

function httpUrl(path = "/") {
  return `${originForProtocol("http:")}${path}`;
}

function replace(path) {
  window.location.replace(path);
}

const onboardingRedirectMarkerKey = "waf_onboarding_redirecting";

export function secureAppUrl(path = "/") {
  return window.location.protocol === "https:" ? path : httpsUrl(path);
}

export function onboardingUrl(path = "/") {
  return window.location.protocol === "http:" ? path : httpUrl(path);
}

export function markOnboardingRedirecting() {
  window.sessionStorage.setItem(onboardingRedirectMarkerKey, String(Date.now()));
}

export function clearOnboardingRedirecting() {
  window.sessionStorage.removeItem(onboardingRedirectMarkerKey);
}

function normalizeReturnPath(path = "/") {
  const value = String(path || "/").trim() || "/";
  return value.startsWith("/") ? value : `/${value}`;
}

function normalizeReturnArgs(args = "") {
  if (args instanceof URLSearchParams) {
    return args.toString();
  }
  const value = String(args || "").trim();
  return value.startsWith("?") ? value.slice(1) : value;
}

export function buildChallengeUrl(returnPath = "/", returnArgs = "") {
  const params = new URLSearchParams();
  params.set("return_uri", normalizeReturnPath(returnPath));
  const normalizedArgs = normalizeReturnArgs(returnArgs);
  if (normalizedArgs) {
    params.set("return_args", normalizedArgs);
  }
  return secureAppUrl(`/challenge?${params.toString()}`);
}

export function buildLoginChallengeUrl(reason = "session_invalid") {
  const params = new URLSearchParams();
  if (String(reason || "").trim()) {
    params.set("reason", String(reason).trim());
  }
  return buildChallengeUrl("/login", params);
}

function clearSessionCookieClient() {
  const variants = [
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Strict`,
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Lax`,
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Strict; Secure`
  ];
  variants.forEach((value) => {
    document.cookie = value;
  });
}

export async function forceRelogin(reason = "session_invalid") {
  clearSessionCookieClient();
  clearOnboardingRedirecting();
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
  } catch {
    // best-effort only
  }
  const target = buildLoginChallengeUrl(reason);
  if (window.location.pathname.startsWith("/login")) {
    return;
  }
  replace(target);
}

function isOnboardingRedirecting() {
  const raw = window.sessionStorage.getItem(onboardingRedirectMarkerKey);
  if (!raw) {
    return false;
  }
  const startedAt = Number(raw);
  if (!Number.isFinite(startedAt)) {
    window.sessionStorage.removeItem(onboardingRedirectMarkerKey);
    return false;
  }
  if (Date.now() - startedAt > 30000) {
    window.sessionStorage.removeItem(onboardingRedirectMarkerKey);
    return false;
  }
  return true;
}

async function readJSON(response) {
  const text = await response.text();
  if (!text) {
    return null;
  }
  try {
    return JSON.parse(text);
  } catch {
    return { raw: text };
  }
}

async function getSetupStatus() {
  const response = await fetch("/api/setup/status", {
    credentials: "include",
    headers: { Accept: "application/json" }
  });
  if (!response.ok) {
    const payload = await readJSON(response);
    const error = new Error(payload?.error || `request failed with ${response.status}`);
    error.status = response.status;
    throw error;
  }
  return readJSON(response);
}

async function getCurrentUserQuiet() {
  const response = await fetch("/api/auth/me", {
    credentials: "include",
    headers: { Accept: "application/json" }
  });
  if (response.status === 401) {
    clearSessionCookieClient();
    return null;
  }
  if (!response.ok) {
    const payload = await readJSON(response);
    const error = new Error(payload?.error || `request failed with ${response.status}`);
    error.status = response.status;
    throw error;
  }
  return readJSON(response);
}

export async function checkEntryAccess(mode) {
  let setup;
  try {
    setup = await getSetupStatus();
  } catch (error) {
    if (mode === "app") {
      await forceRelogin("session_check_failed");
      return { setup: null, user: null, allowed: false };
    }
    throw error;
  }
  const onboardingRequired = Boolean(setup.needs_bootstrap);
  const initializationIncomplete = Boolean(setup && !setup.has_active_revision);
  const onboardingRedirecting = isOnboardingRedirecting();
  let user = null;

  if (mode === "onboarding") {
    if ((onboardingRequired || initializationIncomplete) && window.location.protocol !== "http:") {
      replace(httpUrl("/onboarding/user-creation"));
      return { setup, user, allowed: false };
    }
    if (onboardingRequired || initializationIncomplete) {
      return { setup, user, allowed: true };
    }
    if (onboardingRedirecting) {
      replace(buildLoginChallengeUrl("onboarding_complete"));
      return { setup, user, allowed: false };
    }
    user = await getCurrentUserQuiet();
    replace(user ? secureAppUrl("/healthcheck") : buildLoginChallengeUrl("session_missing"));
    return { setup, user, allowed: false };
  }

  if (mode === "login") {
    if (onboardingRequired || initializationIncomplete) {
      replace(httpUrl("/onboarding/user-creation"));
      return { setup, user, allowed: false };
    }
    if (window.location.protocol !== "https:") {
      replace(httpsUrl(window.location.pathname || "/login"));
      return { setup, user, allowed: false };
    }
    if (onboardingRedirecting) {
      clearOnboardingRedirecting();
      return { setup, user, allowed: true };
    }
    try {
      user = await getCurrentUserQuiet();
    } catch {
      return { setup, user: null, allowed: true };
    }
    if (user) {
      replace(secureAppUrl("/healthcheck"));
      return { setup, user, allowed: false };
    }
    return { setup, user, allowed: true };
  }

  if (mode === "login-2fa") {
    if (onboardingRequired || initializationIncomplete) {
      replace(httpUrl("/onboarding/user-creation"));
      return { setup, user, allowed: false };
    }
    if (window.location.protocol !== "https:") {
      replace(httpsUrl(window.location.pathname || "/login/2fa"));
      return { setup, user, allowed: false };
    }
    if (onboardingRedirecting) {
      clearOnboardingRedirecting();
      return { setup, user, allowed: true };
    }
    try {
      user = await getCurrentUserQuiet();
    } catch {
      return { setup, user: null, allowed: true };
    }
    if (user) {
      replace(secureAppUrl("/healthcheck"));
      return { setup, user, allowed: false };
    }
    return { setup, user, allowed: true };
  }

  if (mode === "app") {
    if (onboardingRequired || initializationIncomplete) {
      replace(httpUrl("/onboarding/user-creation"));
      return { setup, user, allowed: false };
    }
    if (window.location.protocol !== "https:") {
      replace(httpsUrl(window.location.pathname || "/healthcheck"));
      return { setup, user, allowed: false };
    }
    try {
      user = await getCurrentUserQuiet();
    } catch {
      await forceRelogin("session_check_failed");
      return { setup, user: null, allowed: false };
    }
    if (!user) {
      await forceRelogin("session_missing");
      return { setup, user, allowed: false };
    }
    clearOnboardingRedirecting();
    return { setup, user, allowed: true };
  }

  return { setup, user, allowed: true };
}
