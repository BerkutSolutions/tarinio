export function bindDetailSearchAndListEvents(params) {
  const {
    container,
    state,
    ctx,
    feedback,
    render,
    syncStateDraftFromForm,
    highlightSelector,
    getQuickListTemplates,
    LIST_FIELD_SET,
    normalizeStringArray,
    normalizeCustomLimitRules,
    normalizeAntibotExclusionRules,
    normalizeAntibotChallengeRules,
    normalizeAuthBasicUsers
  } = params;

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
      window.setTimeout(() => highlightSelector(state.highlightedSelector), 30);
    });
  });

  container.querySelectorAll("[data-list-add]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listAdd || "";
      if (!LIST_FIELD_SET.has(field)) {
        return;
      }
      const input = container.querySelector(`#list-input-${field}`);
      if (!input) {
        return;
      }
      const value = String(input.value || "").trim();
      if (!value) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeStringArray(state.draft[field]);
      if (!current.includes(value)) {
        current.push(value);
        state.draft[field] = current;
      }
      render();
    });
  });

  container.querySelectorAll("[data-list-template-apply]").forEach((button) => {
    button.addEventListener("click", () => {
      const field = button.dataset.listTemplateApply || "";
      const presetID = String(button.dataset.listTemplateId || "").trim();
      if (!LIST_FIELD_SET.has(field)) {
        return;
      }
      if (!presetID) {
        return;
      }
      syncStateDraftFromForm();
      const preset = getQuickListTemplates(field).find((item) => item.id === presetID);
      if (!preset) {
        return;
      }
      const current = new Set(normalizeStringArray(state.draft[field]));
      for (const item of normalizeStringArray(preset.items)) {
        current.add(item);
      }
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
      if (!LIST_FIELD_SET.has(field) || !presetID) {
        return;
      }
      syncStateDraftFromForm();
      const selected = new Set(normalizeStringArray(state.listTemplateSelection[field]));
      if (!selected.has(presetID)) {
        return;
      }
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
        if (!preset) {
          continue;
        }
        for (const value of normalizeStringArray(preset.items)) {
          remainingTemplateItems.add(value);
        }
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
      if (!LIST_FIELD_SET.has(field) || !value) {
        return;
      }
      syncStateDraftFromForm();
      const current = new Set(normalizeStringArray(state.draft[field]));
      if (checkbox.checked) {
        current.add(value);
      } else {
        current.delete(value);
      }
      state.draft[field] = Array.from(current);
      render();
    });
  });

  container.querySelectorAll("[id^='country-search-']").forEach((input) => {
    input.addEventListener("input", (event) => {
      const field = String(input.id || "").replace("country-search-", "");
      if (!field) {
        return;
      }
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
      if (!LIST_FIELD_SET.has(field)) {
        return;
      }
      const index = Number(button.dataset.listIndex || "-1");
      syncStateDraftFromForm();
      const current = normalizeStringArray(state.draft[field]);
      if (index < 0 || index >= current.length) {
        return;
      }
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
      if (!Number.isInteger(index) || index < 0) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeCustomLimitRules(state.draft.custom_limit_rules);
      if (index >= current.length) {
        return;
      }
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
      if (!Number.isInteger(index) || index < 0) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeAntibotChallengeRules(state.draft.antibot_challenge_rules);
      if (index >= current.length) {
        return;
      }
      current.splice(index, 1);
      state.draft.antibot_challenge_rules = current;
      render();
    });
  });

  container.querySelectorAll("[data-antibot-exclusion-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.antibotExclusionRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeAntibotExclusionRules(state.draft.antibot_exclusion_rules);
      if (index >= current.length) {
        return;
      }
      current.splice(index, 1);
      state.draft.antibot_exclusion_rules = current;
      render();
    });
  });

  container.querySelector("[data-auth-user-add]")?.addEventListener("click", () => {
    syncStateDraftFromForm();
    const current = normalizeAuthBasicUsers(state.draft.auth_basic_users);
    current.push({
      username: `user${current.length + 1}`,
      password: "",
      enabled: true,
      last_login_at: ""
    });
    state.draft.auth_basic_users = current;
    render();
  });

  container.querySelectorAll("[data-auth-user-remove]").forEach((button) => {
    button.addEventListener("click", () => {
      const index = Number.parseInt(String(button.dataset.authUserRemove || "-1"), 10);
      if (!Number.isInteger(index) || index < 0) {
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeAuthBasicUsers(state.draft.auth_basic_users);
      if (index >= current.length) {
        return;
      }
      current.splice(index, 1);
      state.draft.auth_basic_users = current.length ? current : [{ username: "changeme", password: "", enabled: true, last_login_at: "" }];
      render();
    });
  });

}
