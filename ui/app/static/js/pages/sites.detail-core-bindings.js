import { setError } from "../ui.js";
import { routeBase } from "./sites.routing-merge.js";

export function bindDetailCore(container, state, ctx, deps) {
  const {
    go,
    render,
    getDraft,
    parseRawDraft,
    syncStateDraftFromForm,
    draftToEnvText,
    ensureControlPlaneAccessManagementMethods,
    normalizeAutoSiteID,
    syncDerivedFieldsFromID,
    normalizeServiceProfile,
    applyServiceProfilePresetToDraft,
    toggleCertificateImportActions,
    highlightSelector,
  } = deps;
  const feedback = container.querySelector("#sites-feedback");
  const back = () => go(routeBase());
  const idInput = container.querySelector("#service-id");
  const hostInput = container.querySelector("#service-host");
  const certificateInput = container.querySelector("#service-certificate-id");
  const upstreamInput = container.querySelector("#service-upstream-id");

  container.querySelectorAll("[data-wizard-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      state.draft = getDraft();
      state.activeTab = button.dataset.wizardTab || "front";
      render();
    });
  });
  container.querySelectorAll("[data-mode-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      const nextMode = String(button.dataset.modeTab || "easy").trim().toLowerCase() === "raw" ? "raw" : "easy";
      if (nextMode === state.editorMode) {
        return;
      }
      if (nextMode === "raw") {
        if (state.editorMode === "easy") {
          syncStateDraftFromForm();
        }
        state.rawEnvText = draftToEnvText(ensureControlPlaneAccessManagementMethods({ ...state.draft }));
        state.rawMissingFields = [];
        state.editorMode = "raw";
        render();
        return;
      }
      try {
        parseRawDraft();
        state.editorMode = "easy";
        render();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.raw.parseError")}: ${String(error?.message || error)}`);
      }
    });
  });

  if (state.route.mode !== "create") {
    if (idInput?.value?.trim()) idInput.dataset.dirty = "true";
    if (certificateInput?.value?.trim()) certificateInput.dataset.dirty = "true";
  }

  container.querySelector("#service-back")?.addEventListener("click", back);
  container.querySelector("#service-back-bottom")?.addEventListener("click", back);
  container.querySelector("#service-raw-env")?.addEventListener("input", (event) => {
    state.rawEnvText = String(event.target.value || "");
    state.rawMissingFields = [];
  });
  container.querySelector("#service-host")?.addEventListener("input", (event) => {
    const host = event.target.value.trim().toLowerCase();
    if (idInput && !idInput.dataset.dirty) {
      idInput.value = normalizeAutoSiteID(host);
      syncDerivedFieldsFromID(idInput, certificateInput, upstreamInput);
    }
  });
  container.querySelector("#service-id")?.addEventListener("input", (event) => {
    const id = event.target.value.trim().toLowerCase();
    const autoID = normalizeAutoSiteID(hostInput?.value || "");
    event.target.dataset.dirty = id && id !== autoID ? "true" : "";
    syncDerivedFieldsFromID(idInput, certificateInput, upstreamInput);
  });
  container.querySelector("#service-profile")?.addEventListener("change", (event) => {
    syncStateDraftFromForm();
    const selectedProfile = normalizeServiceProfile(event?.target?.value || state.draft.service_profile);
    const currentProfile = normalizeServiceProfile(state.draft.service_profile);
    if (selectedProfile === currentProfile) {
      return;
    }
    state.draft = applyServiceProfilePresetToDraft(state.draft, selectedProfile);
    render();
  });
  container.querySelector("#service-certificate-id")?.addEventListener("input", (event) => {
    event.target.dataset.dirty = event.target.value.trim() ? "true" : "";
  });

  container.querySelector("#service-ca-server")?.addEventListener("change", () => toggleCertificateImportActions(container));
  toggleCertificateImportActions(container);

  container.querySelector("#service-use-modsecurity-custom-configuration")?.addEventListener("change", () => {
    syncStateDraftFromForm();
    render();
  });
  container.querySelector("#service-pass-host-header")?.addEventListener("change", () => {
    syncStateDraftFromForm();
    render();
  });

  // Checkboxes that enable/disable dependent fields — need re-render
  [
    "#service-health-check-enabled",
    "#service-upstream-mtls-enabled",
    "#service-mtls-enabled",
    "#service-use-limit-conn",
    "#service-use-limit-req",
    "#service-use-exceptions",
    "#service-use-allowlist",
    "#service-use-dnsbl",
    "#service-use-bad-behavior",
    "#service-use-blacklist",
    "#service-use-reverse-proxy",
    "#service-ban-escalation-enabled",
    "#service-sni-enabled",
    "#service-reverse-proxy-ssl-sni",
  ].forEach((id) => {
    container.querySelector(id)?.addEventListener("change", () => {
      syncStateDraftFromForm();
      render();
    });
  });

  container.querySelector("#service-settings-search")?.addEventListener("input", (event) => {
    state.settingsSearch = String(event.target.value || "");
    state.highlightedSelector = "";
    const cursorStart = Number(event.target.selectionStart || state.settingsSearch.length);
    const cursorEnd = Number(event.target.selectionEnd || cursorStart);
    render();
    const nextInput = container.querySelector("#service-settings-search");
    if (nextInput) {
      nextInput.focus();
      nextInput.setSelectionRange(cursorStart, cursorEnd);
    }
  });
  container.querySelectorAll("[data-settings-result]").forEach((button) => {
    button.addEventListener("click", () => {
      syncStateDraftFromForm();
      state.activeTab = String(button.dataset.settingsTab || "front");
      state.settingsSearch = "";
      state.highlightedSelector = String(button.dataset.settingsSelector || "");
      render();
      window.setTimeout(() => highlightSelector(container, state.highlightedSelector), 30);
    });
  });
  if (state.highlightedSelector) {
    window.setTimeout(() => highlightSelector(container, state.highlightedSelector), 30);
  }
}
