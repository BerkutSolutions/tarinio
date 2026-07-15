function firstRevisionID(payload) {
  const activeID = String(payload?.revision_apply?.active_revision_id || "").trim();
  if (activeID) {
    return activeID;
  }
  const items = Array.isArray(payload?.latest_revisions) ? payload.latest_revisions : [];
  const latest = items.find((item) => String(item?.revision_id || "").trim());
  return String(latest?.revision_id || "").trim();
}

function setProtectionIndicator(id, enabled) {
  const node = document.getElementById(id);
  if (!node) {
    return;
  }
  node.classList.toggle("is-on", enabled);
  node.classList.toggle("is-off", !enabled);
}

function setCRSVersion(value) {
  const node = document.getElementById("sidebar-mode-crs");
  if (!node) {
    return;
  }
  const version = String(value || "").trim();
  node.textContent = version ? `CRS ${version}` : "CRS —";
}

function setRuntimeStatus(node, healthy, translate) {
  node.classList.toggle("is-healthy", healthy);
  node.classList.toggle("is-unhealthy", !healthy);
  node.textContent = translate(healthy ? "app.runtimeHealthy" : "app.runtimeUnavailable");
}

export async function refreshSidebarStatus(api, translate) {
  const runtime = document.getElementById("sidebar-runtime-status");
  const revision = document.getElementById("sidebar-revision");
  if (!runtime || !revision) {
    return;
  }

  try {
    const response = await fetch("/healthz", { credentials: "include", cache: "no-store" });
    setRuntimeStatus(runtime, response.ok, translate);
  } catch {
    setRuntimeStatus(runtime, false, translate);
  }

  try {
    const report = await api.get("/api/reports/revisions");
    revision.textContent = firstRevisionID(report) || "rev —";
  } catch {
    revision.textContent = "rev —";
  }

  try {
    const settings = await api.get("/api/anti-ddos/settings");
    setProtectionIndicator("sidebar-mode-l4", settings?.use_l4_guard === true);
    setProtectionIndicator("sidebar-mode-l7", settings?.enforce_l7_rate_limit === true);
    setProtectionIndicator("sidebar-mode-model", settings?.model_enabled === true);
  } catch {
    setProtectionIndicator("sidebar-mode-l4", false);
    setProtectionIndicator("sidebar-mode-l7", false);
    setProtectionIndicator("sidebar-mode-model", false);
  }

  try {
    const status = await api.get("/api/owasp-crs/status");
    setCRSVersion(status?.active_version);
  } catch {
    setCRSVersion("");
  }
}
