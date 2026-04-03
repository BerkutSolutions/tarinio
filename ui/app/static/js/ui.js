import { t } from "./i18n.js";

export function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function formatDate(value) {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;
}

export function statusBadge(status) {
  const normalized = String(status || "").toLowerCase();
  let css = "badge-neutral";
  if (normalized === "succeeded" || normalized === "active") {
    css = "badge-success";
  } else if (normalized === "failed" || normalized === "revoked" || normalized === "expired") {
    css = "badge-danger";
  } else if (normalized === "pending" || normalized === "running") {
    css = "badge-warning";
  }
  return `<span class="badge ${css}">${escapeHtml(t(`status.${normalized || "unknown"}`))}</span>`;
}

export function csvToList(value) {
  return String(value || "")
    .split("\n")
    .map((item) => item.split(","))
    .flat()
    .map((item) => item.trim())
    .filter(Boolean);
}

export function listToMultiline(values) {
  return Array.isArray(values) ? values.join("\n") : "";
}

export function notify(message, tone = "success") {
  const container = document.getElementById("app-toast-container");
  if (!container) {
    return;
  }
  const node = document.createElement("div");
  node.className = `app-toast ${tone}`;
  node.innerHTML = `
    <div class="app-toast-text">${escapeHtml(message)}</div>
    <button type="button" class="app-toast-close" aria-label="${escapeHtml(t("ui.close"))}">×</button>
  `;
  container.appendChild(node);
  const close = () => {
    node.classList.remove("show");
    window.setTimeout(() => node.remove(), 220);
  };
  node.querySelector(".app-toast-close").addEventListener("click", close);
  window.setTimeout(() => node.classList.add("show"), 10);
  window.setTimeout(close, 3200);
}

export function parseJSONLines(value) {
  return String(value || "")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [ruleID, enabledRaw] = line.split("=");
      return { rule_id: ruleID.trim(), enabled: enabledRaw?.trim().toLowerCase() !== "false" };
    })
    .filter((item) => item.rule_id);
}

export function setLoading(element, message = t("common.loading")) {
  if (!element) {
    return;
  }
  element.innerHTML = `<div class="waf-empty waf-loading">${escapeHtml(message)}</div>`;
}

export function setError(element, message = t("app.error")) {
  if (!element) {
    return;
  }
  element.innerHTML = `<div class="alert">${escapeHtml(message)}</div>`;
}

export function confirmAction(message) {
  return window.confirm(message);
}
