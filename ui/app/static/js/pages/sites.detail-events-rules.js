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
