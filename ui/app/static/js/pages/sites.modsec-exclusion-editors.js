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

function normalizeRuleIDsValue(value, normalizeArray) {
  const source = Array.isArray(value)
    ? value
    : String(value || "")
      .split(/[\s,|]+/)
      .map((item) => Number.parseInt(item, 10))
      .filter((item) => Number.isInteger(item) && item > 0);
  return Array.from(new Set(normalizeArray(source)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isInteger(item) && item > 0))).sort((a, b) => a - b);
}

function normalizeTargetsValue(value, normalizeArray) {
  return Array.from(new Set(normalizeArray(
    Array.isArray(value)
      ? value
      : String(value || "").split(/[\n,|]+/)
  )
    .map((item) => String(item || "").trim())
    .filter(Boolean)));
}

function buildRuleKey(rule) {
  return [
    String(rule.path || "").toLowerCase(),
    String(rule.path_pattern || ""),
    String(rule.mode || "exact"),
    (rule.methods || []).join(","),
    (rule.rule_ids || []).join(","),
    (rule.targets || []).join(",")
  ].join("\u0000");
}

export function normalizeModSecurityExclusionRules(value, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      path_pattern: String(item?.path_pattern || item?.pathPattern || "").trim(),
      methods: normalizeMethodsValue(item?.methods, normalizeArray),
      mode: ["exact", "prefix", "regex"].includes(String(item?.mode || "").trim().toLowerCase())
        ? String(item?.mode || "").trim().toLowerCase()
        : "exact",
      rule_ids: normalizeRuleIDsValue(item?.rule_ids ?? item?.ruleIds, normalizeArray),
      targets: normalizeTargetsValue(item?.targets, normalizeArray),
      comment: String(item?.comment || "").trim()
    }))
    .filter((item) => item.path || item.path_pattern)
    .filter((item) => item.rule_ids.length > 0)
    .filter((item) => {
      const key = buildRuleKey(item);
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

export function normalizeModSecurityExclusionDraftRows(value, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  return normalizeArray(value).map((item) => ({
    path: String(item?.path || "").trim(),
    path_pattern: String(item?.path_pattern || item?.pathPattern || "").trim(),
    methods: normalizeMethodsValue(item?.methods, normalizeArray),
    mode: ["exact", "prefix", "regex"].includes(String(item?.mode || "").trim().toLowerCase())
      ? String(item?.mode || "").trim().toLowerCase()
      : "exact",
    rule_ids: normalizeRuleIDsValue(item?.rule_ids ?? item?.ruleIds, normalizeArray),
    targets: normalizeTargetsValue(item?.targets, normalizeArray),
    comment: String(item?.comment || "").trim()
  }));
}

export function readModSecurityExclusionDraftRows(container, deps = {}) {
  const normalizeArray = resolveNormalizeArray(deps);
  return Array.from(container.querySelectorAll("[data-modsec-exclusion-path]"))
    .map((input) => {
      const index = String(input.getAttribute("data-modsec-exclusion-path") || "");
      return {
        path: String(input.value || "").trim(),
        path_pattern: String(container.querySelector(`[data-modsec-exclusion-path-pattern="${index}"]`)?.value || "").trim(),
        methods: normalizeMethodsValue(container.querySelector(`[data-modsec-exclusion-methods="${index}"]`)?.value, normalizeArray),
        mode: String(container.querySelector(`[data-modsec-exclusion-mode="${index}"]`)?.value || "exact").trim().toLowerCase() || "exact",
        rule_ids: normalizeRuleIDsValue(container.querySelector(`[data-modsec-exclusion-rule-ids="${index}"]`)?.value, normalizeArray),
        targets: normalizeTargetsValue(container.querySelector(`[data-modsec-exclusion-targets="${index}"]`)?.value, normalizeArray),
        comment: String(container.querySelector(`[data-modsec-exclusion-comment="${index}"]`)?.value || "").trim()
      };
    });
}

export function renderModSecurityExclusionRulesEditor(rules, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeRules = normalizeModSecurityExclusionDraftRows(rules, deps);
  const modeOptions = ["exact", "prefix", "regex"];
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusions"))}</label>
      <div class="waf-note">${escapeHtml(ctx.t("sites.easy.modsec.exclusionsHint"))}</div>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-stack waf-subcard" style="padding:12px;gap:8px;">
            <div class="waf-form-grid">
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.path"))}</label>
                <input data-modsec-exclusion-path="${index}" placeholder="/api/" value="${escapeHtml(rule.path)}">
              </div>
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.pathPattern"))}</label>
                <input data-modsec-exclusion-path-pattern="${index}" placeholder="^/static/js/.*$" value="${escapeHtml(rule.path_pattern)}">
              </div>
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.mode"))}</label>
                <select data-modsec-exclusion-mode="${index}">
                  ${modeOptions.map((mode) => `<option value="${mode}"${rule.mode === mode ? " selected" : ""}>${escapeHtml(ctx.t(`sites.easy.modsec.exclusion.mode.${mode}`))}</option>`).join("")}
                </select>
              </div>
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.methods"))}</label>
                <input data-modsec-exclusion-methods="${index}" placeholder="GET,POST,*" value="${escapeHtml(rule.methods.join(","))}">
              </div>
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.ruleIds"))}</label>
                <input data-modsec-exclusion-rule-ids="${index}" placeholder="949110,942100" value="${escapeHtml(rule.rule_ids.join(","))}">
              </div>
              <div class="waf-field">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.targets"))}</label>
                <input data-modsec-exclusion-targets="${index}" placeholder="ARGS:*,REQUEST_HEADERS:User-Agent" value="${escapeHtml(rule.targets.join(","))}">
              </div>
              <div class="waf-field full">
                <label>${escapeHtml(ctx.t("sites.easy.modsec.exclusion.comment"))}</label>
                <input data-modsec-exclusion-comment="${index}" placeholder="${escapeHtml(ctx.t("sites.easy.modsec.exclusion.commentPlaceholder"))}" value="${escapeHtml(rule.comment)}">
              </div>
            </div>
            <div class="waf-inline" style="justify-content:space-between;align-items:center;">
              <div class="waf-note">${escapeHtml(ctx.t("sites.easy.modsec.exclusion.targetsHelp"))}</div>
              <button class="btn ghost btn-sm" type="button" data-modsec-exclusion-remove="${index}">x</button>
            </div>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <div class="waf-note">${escapeHtml(ctx.t("sites.easy.modsec.exclusion.methodsHelp"))}</div>
        <button class="btn ghost btn-sm" type="button" data-modsec-exclusion-add>${escapeHtml(ctx.t("sites.easy.modsec.addExclusionRule"))}</button>
      </div>
    </div>
  `;
}
