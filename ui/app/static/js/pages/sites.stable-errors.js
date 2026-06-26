import { escapeHtml } from "../ui.js";

let sitesErrorLoggingInstalled = false;

function formatSitesError(error) {
  if (!error) {
    return "unknown error";
  }
  if (error instanceof Error) {
    const message = String(error.message || error.name || "error");
    const stack = String(error.stack || "").trim();
    return stack ? `${message}\n${stack}` : message;
  }
  return String(error);
}

export function logSitesError(stage, error, details = {}) {
  const payload = {
    stage,
    details,
    error: formatSitesError(error),
    href: window.location.href,
    at: new Date().toISOString(),
  };
  console.error("[sites-page]", payload);
  return payload;
}

export function renderSitesErrorAlert(ctx, payload) {
  const detailsText = `${payload.stage}\n${payload.error}`;
  return `
    <div class="alert">
      <div>${escapeHtml(ctx.t("sites.error.load"))}</div>
      <pre class="waf-code" style="margin-top:8px;white-space:pre-wrap;">${escapeHtml(detailsText)}</pre>
    </div>
  `;
}

export function installSitesGlobalErrorLogging() {
  if (sitesErrorLoggingInstalled) {
    return;
  }
  sitesErrorLoggingInstalled = true;
  window.addEventListener("error", (event) => {
    logSitesError("window.error", event?.error || event?.message || event, {
      filename: event?.filename || "",
      lineno: Number(event?.lineno || 0),
      colno: Number(event?.colno || 0),
    });
  });
  window.addEventListener("unhandledrejection", (event) => {
    logSitesError("window.unhandledrejection", event?.reason || event, {});
  });
}
