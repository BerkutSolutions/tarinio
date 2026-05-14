export function syncStateDraftFromForm(state, getDraft, deps = {}) {
  const {
    normalizeArray,
    BAN_SCOPE_VALUES,
    normalizeBanEscalationStages,
    normalizeAuthBasicUsers,
    normalizeAuthSessionTTLMinutes
  } = deps;
  state.draft = getDraft();
  state.draft.bad_behavior_status_codes = normalizeArray(state.draft.bad_behavior_status_codes)
    .map((item) => Number(item))
    .filter((item) => Number.isInteger(item))
    .sort((a, b) => a - b);
  state.draft.ban_escalation_scope = BAN_SCOPE_VALUES.includes(String(state.draft.ban_escalation_scope || "").trim().toLowerCase())
    ? String(state.draft.ban_escalation_scope || "").trim().toLowerCase()
    : "all_sites";
  state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(
    state.draft.ban_escalation_stages_seconds,
    state.draft.bad_behavior_ban_time_seconds
  );
  state.draft.auth_basic_users = normalizeAuthBasicUsers(state.draft.auth_basic_users);
  state.draft.auth_basic_session_inactivity_minutes = normalizeAuthSessionTTLMinutes(
    state.draft.auth_basic_session_inactivity_minutes
  );
  const firstAuthUser = state.draft.auth_basic_users[0] || { username: "", password: "" };
  state.draft.auth_basic_user = String(firstAuthUser.username || "").trim();
  state.draft.auth_basic_password = String(firstAuthUser.password || "").trim();
}

export function normalizeAutoSiteID(value) {
  return String(value || "").trim().toLowerCase().replace(/[^a-z0-9.-]+/g, "-").replace(/^-+|-+$/g, "");
}

export function syncDerivedFieldsFromID(idInput, upstreamInput, certificateInput, computeUpstreamID) {
  const id = String(idInput?.value || "").trim().toLowerCase();
  if (upstreamInput) {
    upstreamInput.value = computeUpstreamID(id);
  }
  if (certificateInput && (!certificateInput.dataset.dirty || !certificateInput.value.trim())) {
    certificateInput.value = id ? `${id}-tls` : "";
  }
}

export function toggleCertificateImportActions(container) {
  const caServer = String(container.querySelector("#service-ca-server")?.value || "").trim().toLowerCase();
  const row = container.querySelector("#service-certificate-import-actions");
  const picker = container.querySelector("#service-certificate-picker");
  if (!row) {
    return;
  }
  row.style.display = caServer === "import" ? "" : "none";
  if (picker) {
    picker.style.display = caServer === "import" ? "" : "none";
  }
}

export function highlightSelector(container, selector) {
  if (!selector) {
    return;
  }
  const target = container.querySelector(selector);
  if (!target) {
    return;
  }
  target.classList.add("waf-search-highlight");
  window.setTimeout(() => target.classList.remove("waf-search-highlight"), 2200);
  if (typeof target.scrollIntoView === "function") {
    target.scrollIntoView({ behavior: "smooth", block: "center" });
  }
  if (typeof target.focus === "function") {
    target.focus({ preventScroll: true });
  }
}
