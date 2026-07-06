export function normalizeStringArray(value, deps = {}) {
  const normalizeArray = deps.normalizeArray;
  return normalizeArray(value)
    .map((item) => String(item || "").trim())
    .filter(Boolean);
}
export function parseListInput(value) {
  return String(value || "")
    .split(/[\n,| ]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}
export function parseIntListInput(value) {
  return parseListInput(value)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isInteger(item));
}
export function normalizeCustomLimitRules(value, deps = {}) {
  const normalizeArray = deps.normalizeArray;
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      rate: String(item?.rate || "").trim().toLowerCase().replace(/\s+/g, "")
    }))
    .filter((item) => item.path && item.rate)
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.rate;
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}
export function normalizeAntibotChallengeRules(value, deps = {}) {
  const normalizeArray = deps.normalizeArray;
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      challenge: String(item?.challenge || "").trim().toLowerCase()
    }))
    .filter((item) => item.path && item.challenge)
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.challenge;
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}
export function normalizeAuthBasicUsers(value, deps = {}) {
  const normalizeArray = deps.normalizeArray;
  const seen = new Set();
  const out = normalizeArray(value)
    .map((item) => ({
      username: String(item?.username || "").trim(),
      password: String(item?.password || "").trim(),
      enabled: Boolean(item?.enabled ?? true),
      last_login_at: String(item?.last_login_at || "").trim()
    }))
    .filter((item) => item.username)
    .filter((item) => {
      const key = item.username.toLowerCase();
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
  if (!out.length) {
    return [{ username: "changeme", password: "", enabled: true, last_login_at: "" }];
  }
  return out;
}
export function normalizeAuthSessionTTLMinutes(value) {
  const parsed = Number.parseInt(String(value ?? "").trim(), 10);
  if (!Number.isInteger(parsed)) {
    return 60;
  }
  if (parsed === -1) {
    return -1;
  }
  if (parsed < 5) {
    return 5;
  }
  if (parsed > 1440) {
    return 1440;
  }
  return parsed;
}
export function formatAuthLastLogin(value, ctx) {
  const raw = String(value || "").trim();
  if (!raw) {
    return ctx.t("sites.easy.antibot.authUsersNever");
  }
  const ts = Date.parse(raw);
  if (Number.isNaN(ts)) {
    return raw;
  }
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return raw;
  }
}
export function renderAuthPasswordToggleButton(index, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  return `
    <button
      class="waf-password-toggle"
      type="button"
      data-auth-user-toggle="${index}"
      data-visible="false"
      aria-pressed="false"
      title="${escapeHtml(ctx.t("common.show"))}"
    >
      <svg class="waf-password-icon-eye" viewBox="0 0 24 24" aria-hidden="true">
        <path fill="currentColor" d="M12 5c5.2 0 9.4 4.7 10.9 6.8a1 1 0 0 1 0 1.2C21.4 15.1 17.2 19.8 12 19.8S2.6 15.1 1.1 13a1 1 0 0 1 0-1.2C2.6 9.7 6.8 5 12 5Zm0 2c-3.9 0-7.2 3.2-8.8 5.4 1.6 2.2 4.9 5.4 8.8 5.4s7.2-3.2 8.8-5.4C19.2 10.2 15.9 7 12 7Zm0 1.8a3.6 3.6 0 1 1 0 7.2 3.6 3.6 0 0 1 0-7.2Zm0 2a1.6 1.6 0 1 0 0 3.2 1.6 1.6 0 0 0 0-3.2Z"/>
      </svg>
      <svg class="waf-password-icon-off waf-hidden" viewBox="0 0 24 24" aria-hidden="true">
        <path fill="currentColor" d="m3.3 2 18.7 18.7-1.4 1.4-3.2-3.2c-1.6.7-3.4 1.1-5.4 1.1-5.2 0-9.4-4.7-10.9-6.8a1 1 0 0 1 0-1.2c1.1-1.6 3.7-4.6 7.1-6l-3.3-3.3L3.3 2Zm6.2 6.2a3.6 3.6 0 0 1 4.3 4.3l-4.3-4.3Zm5.9 5.9A3.6 3.6 0 0 1 10 8.7l-1.5-1.5A9.5 9.5 0 0 0 3.2 12c1.6 2.2 4.9 5.4 8.8 5.4 1.4 0 2.7-.4 3.8-.9l-1.4-1.4Zm-3.4-9.1c5.2 0 9.4 4.7 10.9 6.8a1 1 0 0 1 0 1.2 18 18 0 0 1-3.6 3.8l-1.4-1.4c1.2-.9 2.2-2 2.9-3-1.6-2.2-4.9-5.4-8.8-5.4-.8 0-1.5.1-2.2.3L8.2 5.8c1.2-.5 2.5-.8 3.8-.8Z"/>
      </svg>
    </button>
  `;
}
export function syncAuthPasswordToggle(button, visible, ctx) {
  button.dataset.visible = visible ? "true" : "false";
  button.setAttribute("aria-pressed", visible ? "true" : "false");
  button.title = ctx.t(visible ? "common.hide" : "common.show");
  button.querySelector(".waf-password-icon-eye")?.classList.toggle("waf-hidden", visible);
  button.querySelector(".waf-password-icon-off")?.classList.toggle("waf-hidden", !visible);
}
export function renderAuthUsersEditor(users, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeUsers = normalizeAuthBasicUsers(users, deps);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.antibot.authUsers"))}</label>
      <div class="waf-table-wrap">
        <table class="waf-table waf-services-table waf-auth-users-table">
          <thead>
            <tr>
              <th>${escapeHtml(ctx.t("sites.easy.antibot.authUsersUsername"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.antibot.authUsersPassword"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.antibot.authUsersEnabled"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.antibot.authUsersLastLogin"))}</th>
              <th>${escapeHtml(ctx.t("sites.easy.antibot.authUsersActions"))}</th>
            </tr>
          </thead>
          <tbody>
            ${safeUsers.length ? safeUsers.map((user, index) => `
              <tr>
                <td>
                  <input data-auth-user-username="${index}" value="${escapeHtml(user.username)}" placeholder="user${index + 1}">
                </td>
                <td>
                  <div class="waf-auth-password-cell">
                    <input data-auth-user-password="${index}" type="password" value="${escapeHtml(user.password)}">
                    ${renderAuthPasswordToggleButton(index, ctx, deps)}
                  </div>
                </td>
                <td class="waf-auth-users-enabled-cell">
                  <label class="waf-checkbox">
                    <input data-auth-user-enabled="${index}" type="checkbox"${user.enabled ? " checked" : ""}>
                  </label>
                </td>
                <td>
                  ${escapeHtml(formatAuthLastLogin(user.last_login_at, ctx))}
                </td>
                <td>
                  <button class="btn ghost btn-sm" type="button" data-auth-user-remove="${index}">x</button>
                </td>
              </tr>
            `).join("") : `
              <tr>
                <td colspan="5">
                  <div class="waf-empty">${escapeHtml(ctx.t("sites.easy.noValues"))}</div>
                </td>
              </tr>
            `}
          </tbody>
        </table>
      </div>
      <div class="waf-actions">
        <button class="btn ghost btn-sm" type="button" data-auth-user-add>${escapeHtml(ctx.t("sites.easy.antibot.authUsersAdd"))}</button>
      </div>
    </div>
  `;
}

export function renderAuthSessionTtlOptions(ttlMinutes, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeTTL = normalizeAuthSessionTTLMinutes(ttlMinutes);
  const ttlOptions = [
    { value: 5, label: "5m" },
    { value: 10, label: "10m" },
    { value: 15, label: "15m" },
    { value: 30, label: "30m" },
    ...Array.from({ length: 24 }, (_unused, idx) => {
      const hours = idx + 1;
      return { value: hours * 60, label: `${hours}h` };
    }),
    { value: -1, label: ctx.t("sites.easy.antibot.authSessionUnlimited") }
  ];
  return ttlOptions
    .map((option) => `<option value="${option.value}"${option.value === safeTTL ? " selected" : ""}>${escapeHtml(option.label)}</option>`)
    .join("");
}

export function renderCustomLimitRulesEditor(rules, ctx, deps = {}, disabled = false) {
  const escapeHtml = deps.escapeHtml;
  const safeRules = normalizeCustomLimitRules(rules, deps);
  const dis = disabled ? " disabled" : "";
  return `
    <div class="waf-field full${disabled ? " waf-disabled" : ""}">
      <label>${escapeHtml(ctx.t("sites.easy.traffic.customLimitRules"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-custom-limit-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}"${dis}>
            <div class="waf-custom-limit-rate-wrap">
              <input data-custom-limit-rate="${index}" type="number" min="1" inputmode="numeric" placeholder="20" value="${escapeHtml(String(rule.rate || "").replace(/r\/s$|r\/m$/i, "").trim())}"${dis}>
              <select data-custom-limit-rate-unit="${index}"${dis}>
                <option value="r/s"${!String(rule.rate || "").endsWith("r/m") ? " selected" : ""}>r/s</option>
                <option value="r/m"${String(rule.rate || "").endsWith("r/m") ? " selected" : ""}>r/m</option>
              </select>
            </div>
            <button class="btn ghost btn-sm" type="button" data-custom-limit-remove="${index}"${dis}>x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-custom-limit-add${dis}>${escapeHtml(ctx.t("sites.easy.traffic.addCustomLimit"))}</button>
      </div>
    </div>`;
}

export function renderAntibotChallengeRulesEditor(rules, ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const safeRules = normalizeAntibotChallengeRules(rules, deps);
  const modes = ["no", "cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"];
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.antibot.challengeRulesByUrl"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-antibot-rule-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}">
            <select data-antibot-rule-challenge="${index}">
              ${modes.map((mode) => `<option value="${mode}"${rule.challenge === mode ? " selected" : ""}>${mode}</option>`).join("")}
            </select>
            <button class="btn ghost btn-sm" type="button" data-antibot-rule-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-antibot-rule-add>${escapeHtml(ctx.t("sites.easy.antibot.addChallengeRule"))}</button>
      </div>
    </div>`;
}
