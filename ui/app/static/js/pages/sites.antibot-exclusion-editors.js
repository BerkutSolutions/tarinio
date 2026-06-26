function fallbackNormalizeArray(value) {
  return Array.isArray(value) ? value : [];
}

function resolveNormalizeArray(deps) {
  return typeof deps.normalizeArray === "function" ? deps.normalizeArray : fallbackNormalizeArray;
}

function normalizeMethodsValue(value, normalizeArray) {
  const source = Array.isArray(value)
    ? value
    : String(value || "")
      .split(/[\s,|]+/)
      .map((item) => item.trim())
      .filter(Boolean);
  const methods = normalizeArray(source)
    .map((item) => String(item || "").trim().toUpperCase())
    .filter(Boolean);
  if (!methods.length || methods.includes("*")) {
    return ["*"];
  }
  return Array.from(new Set(methods)).sort();
}

export function normalizeAntibotExclusionRules(value, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      methods: normalizeMethodsValue(item?.methods, normalizeArray)
    }))
    .filter((item) => item.path)
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.methods.join(",");
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

export function normalizeAntibotExclusionDraftRows(value, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  return normalizeArray(value).map((item) => ({
    path: String(item?.path || "").trim(),
    methods: normalizeMethodsValue(item?.methods, normalizeArray)
  }));
}

export function readAntibotExclusionDraftRows(container, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  return Array.from(container.querySelectorAll("[data-antibot-exclusion-path]")).map((input) => {
    const index = String(input.getAttribute("data-antibot-exclusion-path") || "");
    const methodsInput = container.querySelector(`[data-antibot-exclusion-methods="${index}"]`);
    return {
      path: String(input.value || "").trim(),
      methods: normalizeMethodsValue(methodsInput?.value, normalizeArray)
    };
  });
}

export function renderAntibotExclusionRulesEditor(rules, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeRules = normalizeAntibotExclusionDraftRows(rules, deps);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.antibot.exclusionRulesByUrl"))}</label>
      <div class="waf-note">${escapeHtml(ctx.t("sites.easy.antibot.exclusionRulesHint"))}</div>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input
              data-antibot-exclusion-path="${index}"
              placeholder="/api/"
              value="${escapeHtml(rule.path)}"
            >
            <input
              data-antibot-exclusion-methods="${index}"
              placeholder="${escapeHtml(ctx.t("sites.easy.antibot.exclusionMethodsPlaceholder"))}"
              value="${escapeHtml(rule.methods.join(","))}"
            >
            <button class="btn ghost btn-sm" type="button" data-antibot-exclusion-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <div class="waf-note">${escapeHtml(ctx.t("sites.easy.antibot.exclusionMethodsHelp"))}</div>
        <button class="btn ghost btn-sm" type="button" data-antibot-exclusion-add>${escapeHtml(ctx.t("sites.easy.antibot.addExclusionRule"))}</button>
      </div>
    </div>
  `;
}
