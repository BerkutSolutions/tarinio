export const AUTH_MODE_VALUES = ["basic", "service_token", "basic_or_token"];
export const AUTH_ORDER_VALUES = ["auth_first", "antibot_first"];

function normalizeArray(value, deps = {}) {
  return Array.isArray(value) ? value : deps.normalizeArray?.(value) || [];
}

function normalizeMethods(value, deps = {}) {
  const methods = normalizeArray(value, deps)
    .map((method) => String(method || "").trim().toUpperCase())
    .filter(Boolean);
  return methods.includes("*") || !methods.length ? ["*"] : Array.from(new Set(methods));
}

export function normalizeAuthMode(value) {
  const mode = String(value || "").trim().toLowerCase();
  return AUTH_MODE_VALUES.includes(mode) ? mode : "basic";
}

export function normalizeAuthOrder(value) {
  const order = String(value || "").trim().toLowerCase();
  return AUTH_ORDER_VALUES.includes(order) ? order : "auth_first";
}

export function normalizeAuthExclusionRules(value, deps = {}) {
  const seen = new Set();
  return normalizeArray(value, deps)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      methods: normalizeMethods(item?.methods, deps)
    }))
    .filter((item) => item.path)
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.methods.join(",");
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

function normalizeAuthExclusionDraftRows(value, deps = {}) {
  return normalizeArray(value, deps).map((item) => ({
    path: String(item?.path || "").trim(),
    methods: normalizeMethods(item?.methods, deps)
  }));
}

export function readAuthExclusionDraftRows(container) {
  return Array.from(container.querySelectorAll("[data-auth-exclusion-path]")).map((input) => {
    const index = String(input.dataset.authExclusionPath || "");
    const methodsInput = container.querySelector(`[data-auth-exclusion-methods="${index}"]`);
    return {
      path: String(input.value || "").trim(),
      methods: String(methodsInput?.value || "").split(/[\s,|]+/).map((item) => item.trim()).filter(Boolean)
    };
  });
}

export function normalizeAuthServiceTokens(value, deps = {}) {
  const seen = new Set();
  return normalizeArray(value, deps)
    .map((item) => ({
      service_name: String(item?.service_name || "").trim(),
      token: String(item?.token || "").trim(),
      enabled: Boolean(item?.enabled ?? true),
      last_used_at: String(item?.last_used_at || "").trim()
    }))
    .filter((item) => item.service_name)
    .filter((item) => {
      const key = item.service_name.toLowerCase();
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

export function renderAuthExclusionRulesEditor(rules, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeRules = normalizeAuthExclusionDraftRows(rules, deps);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.auth.exclusionRules"))}</label>
      <div class="waf-note">${escapeHtml(ctx.t("sites.easy.auth.exclusionRulesHint"))}</div>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-auth-exclusion-path="${index}" placeholder="/api/" value="${escapeHtml(rule.path)}">
            <input data-auth-exclusion-methods="${index}" placeholder="${escapeHtml(ctx.t("sites.easy.auth.exclusionMethodsPlaceholder"))}" value="${escapeHtml(rule.methods.join(","))}">
            <button class="btn ghost btn-sm" type="button" data-auth-exclusion-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <div class="waf-note">${escapeHtml(ctx.t("sites.easy.auth.exclusionMethodsHelp"))}</div>
        <button class="btn ghost btn-sm" type="button" data-auth-exclusion-add>${escapeHtml(ctx.t("sites.easy.auth.addExclusionRule"))}</button>
      </div>
    </div>
  `;
}

export function renderAuthServiceTokensEditor(tokens, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeTokens = normalizeAuthServiceTokens(tokens, deps);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.auth.serviceTokens"))}</label>
      <div class="waf-note">${escapeHtml(ctx.t("sites.easy.auth.serviceTokensHint"))}</div>
      <div class="waf-table-wrap">
        <table class="waf-table waf-services-table waf-auth-users-table">
          <thead>
            <tr>
              <th>${escapeHtml(ctx.t("sites.easy.auth.serviceName"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.auth.serviceToken"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.auth.serviceTokenEnabled"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.auth.serviceTokenLastUsed"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.auth.serviceTokenActions"))}</th>
            </tr>
          </thead>
          <tbody>
            ${safeTokens.length ? safeTokens.map((token, index) => `
              <tr>
                <td><input data-auth-token-service-name="${index}" value="${escapeHtml(token.service_name)}" placeholder="sentry-ingest"></td>
                <td>
                  <div class="waf-auth-password-cell">
                    <input data-auth-token-secret="${index}" type="password" value="${escapeHtml(token.token)}">
                    <button class="waf-password-toggle" type="button" data-auth-token-toggle="${index}" data-visible="false" aria-pressed="false" title="${escapeHtml(ctx.t("common.show"))}" aria-label="${escapeHtml(ctx.t("common.show"))}">${escapeHtml(ctx.t("common.show"))}</button>
                  </div>
                </td>
                <td class="waf-auth-users-enabled-cell"><label class="waf-checkbox"><input data-auth-token-enabled="${index}" type="checkbox"${token.enabled ? " checked" : ""}></label></td>
                <td>${escapeHtml(token.last_used_at || ctx.t("sites.easy.antibot.authUsersNever"))}</td>
                <td><button class="btn ghost btn-sm" type="button" data-auth-token-remove="${index}">x</button></td>
              </tr>
            `).join("") : `
              <tr><td colspan="5"><div class="waf-empty">${escapeHtml(ctx.t("sites.easy.noValues"))}</div></td></tr>
            `}
          </tbody>
        </table>
      </div>
      <div class="waf-actions">
        <button class="btn ghost btn-sm" type="button" data-auth-token-add>${escapeHtml(ctx.t("sites.easy.auth.addServiceToken"))}</button>
      </div>
    </div>
  `;
}
