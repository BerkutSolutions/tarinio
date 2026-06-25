export function bindDetailListEditors(container, state, deps) {
  const {
    LIST_FIELD_SET,
    getQuickListTemplates,
    normalizeStringArray,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers,
    syncAuthPasswordToggle,
    normalizeArray,
    parseBanDurationSeconds,
    normalizeBanEscalationStages,
    setError,
    render,
    syncStateDraftFromForm,
    ctx,
    feedback,
  } = deps;

  container.querySelectorAll("[data-list-add]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listAdd || "";
      if (!LIST_FIELD_SET.has(field)) return;
      const input = container.querySelector(`#list-input-${field}`);
      if (!input) return;
      const value = String(input.value || "").trim();
      if (!value) return;
      syncStateDraftFromForm();
      const current = normalizeStringArray(state.draft[field]);
      if (!current.includes(value)) current.push(value);
      state.draft[field] = current;
      render();
    });
  });

  container.querySelectorAll("[data-list-template-apply]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listTemplateApply || "";
      const presetID = String(button.dataset.listTemplateId || "").trim();
      if (!LIST_FIELD_SET.has(field) || !presetID) return;
      syncStateDraftFromForm();
      const preset = getQuickListTemplates(field).find((item) => item.id === presetID);
      if (!preset) return;
      const current = new Set(normalizeStringArray(state.draft[field]));
      for (const item of normalizeStringArray(preset.items)) current.add(item);
      const selected = new Set(normalizeStringArray(state.listTemplateSelection[field]));
      selected.add(presetID);
      state.listTemplateSelection[field] = Array.from(selected);
      state.draft[field] = Array.from(current);
      render();
    });
  });

  container.querySelectorAll("[data-list-template-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listTemplateRemove || "";
      const presetID = String(button.dataset.listTemplateId || "").trim();
      if (!LIST_FIELD_SET.has(field) || !presetID) return;
      syncStateDraftFromForm();
      const selected = new Set(normalizeStringArray(state.listTemplateSelection[field]));
      if (!selected.has(presetID)) return;
      selected.delete(presetID);
      state.listTemplateSelection[field] = Array.from(selected);
      const removedPreset = getQuickListTemplates(field).find((item) => item.id === presetID);
      if (!removedPreset) {
        render();
        return;
      }
      const remainingTemplateItems = new Set();
      for (const id of selected) {
        const preset = getQuickListTemplates(field).find((item) => item.id === id);
        if (!preset) continue;
        for (const value of normalizeStringArray(preset.items)) remainingTemplateItems.add(value);
      }
      const removedItems = new Set(normalizeStringArray(removedPreset.items));
      const current = normalizeStringArray(state.draft[field]);
      state.draft[field] = current.filter((value) => !removedItems.has(value) || remainingTemplateItems.has(value));
      render();
    });
  });

  container.querySelectorAll("[data-country-toggle]").forEach((checkbox) => {
    checkbox.addEventListener("change", () => {
      const field = checkbox.dataset.countryToggle || "";
      const value = String(checkbox.dataset.countryValue || "").trim().toUpperCase();
      if (!LIST_FIELD_SET.has(field) || !value) return;
      syncStateDraftFromForm();
      const current = new Set(normalizeStringArray(state.draft[field]));
      if (checkbox.checked) current.add(value);
      else current.delete(value);
      state.draft[field] = Array.from(current);
      render();
    });
  });

  container.querySelectorAll("[id^='country-search-']").forEach((input) => {
    input.addEventListener("input", (event) => {
      const field = String(input.id || "").replace("country-search-", "");
      if (!field) return;
      state.countryFilters[field] = String(event.target.value || "");
      const cursorStart = Number(event.target.selectionStart || state.countryFilters[field].length);
      const cursorEnd = Number(event.target.selectionEnd || cursorStart);
      render();
      const nextInput = container.querySelector(`#country-search-${field}`);
      if (nextInput) {
        nextInput.focus();
        nextInput.setSelectionRange(cursorStart, cursorEnd);
      }
    });
  });

  container.querySelectorAll("[data-list-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listRemove || "";
      if (!LIST_FIELD_SET.has(field)) return;
      const index = Number(button.dataset.listIndex || "-1");
      syncStateDraftFromForm();
      const current = normalizeStringArray(state.draft[field]);
      if (index < 0 || index >= current.length) return;
      current.splice(index, 1);
      state.draft[field] = current;
      render();
    });
  });

  container.querySelector("[data-custom-limit-add]")?.addEventListener("click", () => {
    syncStateDraftFromForm();
    state.draft.custom_limit_rules = [...normalizeCustomLimitRules(state.draft.custom_limit_rules), { path: "/", rate: "10r/s" }];
    render();
  });
  container.querySelectorAll("[data-custom-limit-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.customLimitRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) return;
      syncStateDraftFromForm();
      const current = normalizeCustomLimitRules(state.draft.custom_limit_rules);
      if (index >= current.length) return;
      current.splice(index, 1);
      state.draft.custom_limit_rules = current;
      render();
    });
  });

  container.querySelector("[data-antibot-rule-add]")?.addEventListener("click", () => {
    syncStateDraftFromForm();
    state.draft.antibot_challenge_rules = [...normalizeAntibotChallengeRules(state.draft.antibot_challenge_rules), { path: "/", challenge: "javascript" }];
    render();
  });
  container.querySelector("[data-antibot-exclusion-add]")?.addEventListener("click", () => {
    syncStateDraftFromForm();
    state.draft.antibot_exclusion_rules = [...normalizeAntibotExclusionRules(state.draft.antibot_exclusion_rules), { path: "/api/", methods: ["*"] }];
    render();
  });
  container.querySelectorAll("[data-antibot-rule-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.antibotRuleRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) return;
      syncStateDraftFromForm();
      const current = normalizeAntibotChallengeRules(state.draft.antibot_challenge_rules);
      if (index >= current.length) return;
      current.splice(index, 1);
      state.draft.antibot_challenge_rules = current;
      render();
    });
  });
  container.querySelectorAll("[data-antibot-exclusion-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.antibotExclusionRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) return;
      syncStateDraftFromForm();
      const current = normalizeAntibotExclusionRules(state.draft.antibot_exclusion_rules);
      if (index >= current.length) return;
      current.splice(index, 1);
      state.draft.antibot_exclusion_rules = current;
      render();
    });
  });

  container.querySelector("[data-auth-user-add]")?.addEventListener("click", () => {
    syncStateDraftFromForm();
    const current = normalizeAuthBasicUsers(state.draft.auth_basic_users);
    current.push({ username: `user${current.length + 1}`, password: "", enabled: true, last_login_at: "" });
    state.draft.auth_basic_users = current;
    render();
  });
  container.querySelectorAll("[data-auth-user-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.authUserRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) return;
      syncStateDraftFromForm();
      const current = normalizeAuthBasicUsers(state.draft.auth_basic_users);
      if (index >= current.length) return;
      current.splice(index, 1);
      state.draft.auth_basic_users = current.length ? current : [{ username: "changeme", password: "", enabled: true, last_login_at: "" }];
      render();
    });
  });
  container.querySelectorAll("[data-auth-user-toggle]").forEach((button) => {
    syncAuthPasswordToggle(button, false, ctx);
    button.addEventListener("click", () => {
      const index = String(button.dataset.authUserToggle || "");
      const input = container.querySelector(`[data-auth-user-password="${index}"]`);
      if (!input) return;
      const nextVisible = input.type !== "text";
      input.type = nextVisible ? "text" : "password";
      syncAuthPasswordToggle(button, nextVisible, ctx);
    });
  });

  container.querySelectorAll("[data-bad-code]").forEach((checkbox) => {
    checkbox.addEventListener("change", () => {
      const code = Number(checkbox.dataset.badCode || "0");
      if (!Number.isInteger(code) || code <= 0) return;
      syncStateDraftFromForm();
      const selected = new Set(normalizeArray(state.draft.bad_behavior_status_codes).map((item) => Number(item)).filter((item) => Number.isInteger(item)));
      if (checkbox.checked) selected.add(code);
      else selected.delete(code);
      state.draft.bad_behavior_status_codes = Array.from(selected).sort((a, b) => a - b);
    });
  });
  container.querySelector("[data-ban-stage-add]")?.addEventListener("click", () => {
    const input = container.querySelector("#service-ban-stage-input");
    if (!input) return;
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
      if (!Number.isInteger(index) || index < 0) return;
      syncStateDraftFromForm();
      const current = normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, state.draft.bad_behavior_ban_time_seconds);
      if (index >= current.length) return;
      current.splice(index, 1);
      state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(current, state.draft.bad_behavior_ban_time_seconds);
      render();
    });
  });
}
