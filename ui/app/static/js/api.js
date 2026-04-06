const apiBase = window.WAF_API_BASE || "";
let reloginInFlight = false;
let bgBackoffUntil = 0;
let bgBackoffMessage = "service unavailable";
let bgBackoffStatus = 0;

function clearSessionCookieClient() {
  const variants = [
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Strict`,
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Lax`,
    `${encodeURIComponent("waf_session")}=; Path=/; Max-Age=0; SameSite=Strict; Secure`,
    `${encodeURIComponent("waf_session_boot")}=; Path=/; Max-Age=0; SameSite=Strict`,
    `${encodeURIComponent("waf_session_boot")}=; Path=/; Max-Age=0; SameSite=Lax`,
    `${encodeURIComponent("waf_session_boot")}=; Path=/; Max-Age=0; SameSite=Strict; Secure`,
  ];
  variants.forEach((value) => {
    document.cookie = value;
  });
}

async function hardRelogin(reason = "session_invalid") {
  if (reloginInFlight) {
    return;
  }
  reloginInFlight = true;
  clearSessionCookieClient();
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json" },
    });
  } catch {
    // best-effort only
  }
  const currentPath = window.location.pathname || "/";
  if (!currentPath.startsWith("/login") && !currentPath.startsWith("/onboarding")) {
    window.location.replace(`/login?reason=${encodeURIComponent(reason)}`);
  }
}

function looksLikeHTML(value) {
  const text = String(value || "").trim().toLowerCase();
  return text.startsWith("<!doctype html") || text.startsWith("<html");
}

function renderStandaloneErrorPage(html) {
  if (typeof document === "undefined") {
    return;
  }
  try {
    document.open();
    document.write(String(html || ""));
    document.close();
  } catch {
    // best-effort only
  }
}

function buildError(response, payload) {
  const message = payload?.error || payload?.message || payload?.raw || `HTTP ${response.status}`;
  const error = new Error(String(message || `HTTP ${response.status}`));
  error.status = response.status;
  error.payload = payload;
  return error;
}

function isBackgroundRequest(options = {}) {
  const headerValue = options?.headers?.["X-Berkut-Background"];
  return String(headerValue || "").trim() === "1";
}

async function request(path, options = {}) {
  const background = isBackgroundRequest(options);
  if (background && Date.now() < bgBackoffUntil) {
    const error = new Error(String(bgBackoffMessage || "service unavailable"));
    error.status = Number(bgBackoffStatus || 0);
    error.path = path;
    throw error;
  }

  let response;
  try {
    response = await fetch(`${apiBase}${path}`, {
      credentials: "include",
      headers: {
        Accept: "application/json",
        ...(options.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
        ...(options.headers || {}),
      },
      ...options,
    });
  } catch (error) {
    if (background) {
      bgBackoffUntil = Date.now() + 15000;
      bgBackoffMessage = String(error?.message || "network error");
      bgBackoffStatus = 0;
    }
    throw error;
  }

  if (response.status === 204) {
    if (background) {
      bgBackoffUntil = 0;
      bgBackoffMessage = "service unavailable";
      bgBackoffStatus = 0;
    }
    return null;
  }

  let payload = null;
  const text = await response.text();
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = { raw: text };
    }
  }

  if (!response.ok) {
    if (background && (response.status === 401 || response.status === 502 || response.status === 503 || response.status === 504)) {
      bgBackoffUntil = Date.now() + (response.status === 401 ? 45000 : 20000);
      bgBackoffMessage = String(payload?.error || payload?.message || payload?.raw || "service unavailable");
      bgBackoffStatus = response.status;
    }

    const contentType = String(response.headers.get("content-type") || "").toLowerCase();
    const htmlBody = payload?.raw && looksLikeHTML(payload.raw) ? String(payload.raw) : "";
    if (response.status !== 401 && htmlBody && contentType.includes("text/html")) {
      renderStandaloneErrorPage(htmlBody);
    }

    if (response.status === 401 && !path.startsWith("/api/auth/login") && !path.startsWith("/api/auth/bootstrap")) {
      await hardRelogin("session_expired");
    }
    throw buildError(response, payload);
  }

  if (background) {
    bgBackoffUntil = 0;
    bgBackoffMessage = "service unavailable";
    bgBackoffStatus = 0;
  }
  return payload;
}

export const api = {
  get: (path, options = {}) => request(path, { method: "GET", ...options }),
  post: (path, body, options = {}) =>
    request(path, {
      method: "POST",
      body: body instanceof FormData ? body : JSON.stringify(body || {}),
      ...options,
    }),
  put: (path, body, options = {}) => request(path, { method: "PUT", body: JSON.stringify(body || {}), ...options }),
  delete: (path, options = {}) => request(path, { method: "DELETE", ...options }),
};
