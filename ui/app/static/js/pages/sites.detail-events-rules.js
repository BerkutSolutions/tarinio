export function bindDetailRuleEvents(params) {
  const {
    container,
    state,
    ctx,
    feedback,
    render,
    syncStateDraftFromForm,
    normalizeAuthBasicUsers,
    syncAuthPasswordToggle,
    normalizeArray,
    parseBanDurationSeconds,
    normalizeBanEscalationStages,
    setError
  } = params;

  container.querySelectorAll("[data-auth-user-toggle]").forEach((button) => {
    syncAuthPasswordToggle(button, false, ctx);
    button.addEventListener("click", () => {
      const index = String(button.dataset.authUserToggle || "");
      const input = container.querySelector(`[data-auth-user-password="${index}"]`);
      if (!input) {
        return;
      }
      const nextVisible = input.type !== "text";
      input.type = nextVisible ? "text" : "password";
      syncAuthPasswordToggle(button, nextVisible, ctx);
    });
  });

  container.querySelectorAll("[data-auth-token-toggle]").forEach((button) => {
    button.dataset.visible = "false";
    button.setAttribute("title", ctx.t("common.show"));
    button.setAttribute("aria-label", ctx.t("common.show"));
    button.textContent = ctx.t("common.show");
    button.addEventListener("click", () => {
      const index = String(button.dataset.authTokenToggle || "");
      const input = container.querySelector(`[data-auth-token-secret="${index}"]`);
      if (!input) {
        return;
      }
      const nextVisible = input.type !== "text";
      input.type = nextVisible ? "text" : "password";
      button.dataset.visible = nextVisible ? "true" : "false";
      button.setAttribute("aria-pressed", nextVisible ? "true" : "false");
      button.setAttribute("title", ctx.t(nextVisible ? "common.hide" : "common.show"));
      button.setAttribute("aria-label", ctx.t(nextVisible ? "common.hide" : "common.show"));
      button.textContent = ctx.t(nextVisible ? "common.hide" : "common.show");
    });
  });

  container.querySelector("#service-auth-mode")?.addEventListener("change", () => {
    syncStateDraftFromForm();
    render();
  });

  function toggleHelpModal(modalID, open) {
    const modal = container.querySelector(`#${modalID}`);
    if (!modal) {
      return;
    }
    modal.classList.toggle("waf-hidden", !open);
    if (open) {
      modal.focus();
    }
  }

  container.querySelector("#service-auth-help-btn")?.addEventListener("click", () => toggleHelpModal("service-auth-help-modal", true));
  container.querySelector("#service-antibot-help-btn")?.addEventListener("click", () => toggleHelpModal("service-antibot-help-modal", true));
  container.querySelector("#service-traffic-badbehavior-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-badbehavior-help-modal", true));
  container.querySelector("#service-traffic-limits-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-limits-help-modal", true));
  container.querySelector("#service-traffic-blacklist-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-blacklist-help-modal", true));
  container.querySelector("#service-traffic-allowlist-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-allowlist-help-modal", true));
  container.querySelector("#service-traffic-dnsbl-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-dnsbl-help-modal", true));
  container.querySelector("#service-upstream-headers-help-btn")?.addEventListener("click", () => toggleHelpModal("service-upstream-headers-help-modal", true));
  container.querySelector("#service-front-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-front-chapter-help-modal", true));
  container.querySelector("#service-upstream-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-upstream-chapter-help-modal", true));
  container.querySelector("#service-http-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-http-chapter-help-modal", true));
  container.querySelector("#service-headers-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-headers-chapter-help-modal", true));
  container.querySelector("#service-blocking-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-blocking-chapter-help-modal", true));
  container.querySelector("#service-antibot-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-antibot-chapter-help-modal", true));
  container.querySelector("#service-geo-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-geo-chapter-help-modal", true));
  container.querySelector("#service-modsec-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-modsec-chapter-help-modal", true));
  container.querySelector("#service-websocket-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-websocket-chapter-help-modal", true));
  container.querySelector("#service-virtualpatches-chapter-help-btn")?.addEventListener("click", () => toggleHelpModal("service-virtualpatches-chapter-help-modal", true));
  container.querySelector("#service-upstream-mtls-help-btn")?.addEventListener("click", () => toggleHelpModal("service-upstreamMtls-chapter-help-modal", true));
  container.querySelector("#service-front-main-help-btn")?.addEventListener("click", () => toggleHelpModal("service-front-main-help-modal", true));
  container.querySelector("#service-front-mtls-help-btn")?.addEventListener("click", () => toggleHelpModal("service-front-mtls-help-modal", true));
  container.querySelector("#service-traffic-allowlist-help-btn")?.addEventListener("click", () => toggleHelpModal("service-traffic-allowlist-help-modal", true));
  container.querySelectorAll("[data-help-close]").forEach((button) => {
    button.addEventListener("click", () => toggleHelpModal(String(button.dataset.helpClose || ""), false));
  });

  container.querySelectorAll("[data-bad-code]").forEach((checkbox) => {
    checkbox.addEventListener("change", () => {
      const code = Number(checkbox.dataset.badCode || "0");
      if (!Number.isInteger(code) || code <= 0) {
        return;
      }
      syncStateDraftFromForm();
      const selected = new Set(
        normalizeArray(state.draft.bad_behavior_status_codes)
          .map((item) => Number(item))
          .filter((item) => Number.isInteger(item))
      );
      if (checkbox.checked) {
        selected.add(code);
      } else {
        selected.delete(code);
      }
      state.draft.bad_behavior_status_codes = Array.from(selected).sort((a, b) => a - b);
    });
  });

  container.querySelector("[data-ban-stage-add]")?.addEventListener("click", () => {
    const input = container.querySelector("#service-ban-stage-input");
    if (!input) {
      return;
    }
    const parsed = parseBanDurationSeconds(input.value);
    if (parsed === null) {
      setError(feedback, ctx.t("sites.validation.banStageFormat"));
      return;
    }
    syncStateDraftFromForm();
    const current = normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, state.draft.bad_behavior_ban_time_seconds);
    current.push(parsed);
    state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(current, state.draft.bad_behavior_ban_time_seconds);
    render();
  });

  container.querySelectorAll("[data-ban-stage-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.banStageRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, state.draft.bad_behavior_ban_time_seconds);
      if (index >= current.length) {
        return;
      }
      current.splice(index, 1);
      state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(current, state.draft.bad_behavior_ban_time_seconds);
      render();
    });
  });
}
