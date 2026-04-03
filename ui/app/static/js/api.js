const apiBase = window.WAF_API_BASE || "";
let reloginInFlight = false;

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
      headers: { Accept: "application/json" }
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

async function request(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...(options.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(options.headers || {}),
    },
    ...options,
  });

  if (response.status === 204) {
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
    const message = payload?.error || payload?.message || payload?.raw || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    error.payload = payload;
    const contentType = String(response.headers.get("content-type") || "").toLowerCase();
    const htmlBody = payload?.raw && looksLikeHTML(payload.raw) ? String(payload.raw) : "";
    if (response.status !== 401 && htmlBody && contentType.includes("text/html")) {
      renderStandaloneErrorPage(htmlBody);
    }
    if (response.status === 401 && !path.startsWith("/api/auth/login") && !path.startsWith("/api/auth/bootstrap")) {
      await hardRelogin("session_expired");
    }
    throw error;
  }

  return payload;
}

export const api = {
  get: (path) => request(path, { method: "GET" }),
  post: (path, body) => request(path, { method: "POST", body: body instanceof FormData ? body : JSON.stringify(body || {}) }),
  put: (path, body) => request(path, { method: "PUT", body: JSON.stringify(body || {}) }),
  delete: (path) => request(path, { method: "DELETE" }),
};
