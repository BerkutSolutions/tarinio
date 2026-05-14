import { escapeHtml, formatDate } from "../ui.js";

export function parseCSV(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseRoleMappings(value) {
  return String(value || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [externalGroup, roleList] = line.split("=");
      return {
        external_group: String(externalGroup || "").trim(),
        role_ids: parseCSV(roleList || ""),
      };
    })
    .filter((item) => item.external_group && item.role_ids.length);
}

export function formatRoleMappings(items) {
  return (Array.isArray(items) ? items : [])
    .map((item) => `${String(item?.external_group || "").trim()}=${String((item?.role_ids || []).join(",")).trim()}`)
    .filter(Boolean)
    .join("\n");
}

export function renderEnterprisePanel(state, ctx) {
  const enterprise = state.enterprise || {};
  const oidc = enterprise?.oidc || {};
  const approvals = enterprise?.approvals || {};
  const scim = enterprise?.scim || {};
  return `
    <div class="waf-stack">
      <section class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("administration.enterprise.oidc.title"))}</div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.enterprise.oidc.subtitle"))}</div>
        <form class="waf-form" id="administration-enterprise-form">
          <div class="waf-form-grid">
            <label class="administration-role-option">
              <input type="checkbox" id="enterprise-oidc-enabled" ${oidc.enabled ? "checked" : ""}>
              <span class="administration-role-option-title">${escapeHtml(ctx.t("administration.enterprise.oidc.enabled"))}</span>
            </label>
            <div class="waf-field">
              <label for="enterprise-oidc-name">${escapeHtml(ctx.t("administration.enterprise.oidc.name"))}</label>
              <input id="enterprise-oidc-name" value="${escapeHtml(oidc.display_name || "")}">
            </div>
            <div class="waf-field">
              <label for="enterprise-oidc-issuer">${escapeHtml(ctx.t("administration.enterprise.oidc.issuer"))}</label>
              <input id="enterprise-oidc-issuer" value="${escapeHtml(oidc.issuer_url || "")}">
            </div>
            <div class="waf-field">
              <label for="enterprise-oidc-client-id">${escapeHtml(ctx.t("administration.enterprise.oidc.clientId"))}</label>
              <input id="enterprise-oidc-client-id" value="${escapeHtml(oidc.client_id || "")}">
            </div>
            <div class="waf-field">
              <label for="enterprise-oidc-client-secret">${escapeHtml(ctx.t("administration.enterprise.oidc.clientSecret"))}</label>
              <input id="enterprise-oidc-client-secret" type="password" placeholder="${escapeHtml(oidc.has_client_secret ? ctx.t("administration.enterprise.secretStored") : "")}">
            </div>
            <div class="waf-field full">
              <label for="enterprise-oidc-redirect">${escapeHtml(ctx.t("administration.enterprise.oidc.redirect"))}</label>
              <input id="enterprise-oidc-redirect" value="${escapeHtml(oidc.redirect_url || "")}">
            </div>
            <div class="waf-field">
              <label for="enterprise-oidc-default-roles">${escapeHtml(ctx.t("administration.enterprise.defaultRoles"))}</label>
              <input id="enterprise-oidc-default-roles" value="${escapeHtml((oidc.default_role_ids || []).join(", "))}">
            </div>
            <div class="waf-field">
              <label for="enterprise-oidc-domains">${escapeHtml(ctx.t("administration.enterprise.oidc.domains"))}</label>
              <input id="enterprise-oidc-domains" value="${escapeHtml((oidc.allowed_email_domains || []).join(", "))}">
            </div>
            <div class="waf-field full">
              <label for="enterprise-oidc-mappings">${escapeHtml(ctx.t("administration.enterprise.roleMappings"))}</label>
              <textarea id="enterprise-oidc-mappings" rows="4" placeholder="security-admins=admin,manager&#10;soc-team=soc">${escapeHtml(formatRoleMappings(oidc.group_role_mappings || []))}</textarea>
            </div>
          </div>
          <div class="waf-inline">
            <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("common.save"))}</button>
          </div>
        </form>
      </section>
      <section class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("administration.enterprise.approvals.title"))}</div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.enterprise.approvals.subtitle"))}</div>
        <div class="waf-form-grid">
          <label class="administration-role-option">
            <input type="checkbox" id="enterprise-approvals-enabled" ${approvals.enabled ? "checked" : ""}>
            <span class="administration-role-option-title">${escapeHtml(ctx.t("administration.enterprise.approvals.enabled"))}</span>
          </label>
          <div class="waf-field">
            <label for="enterprise-approvals-count">${escapeHtml(ctx.t("administration.enterprise.approvals.count"))}</label>
            <input id="enterprise-approvals-count" type="number" min="1" value="${escapeHtml(String(approvals.required_approvals || 1))}">
          </div>
          <label class="administration-role-option">
            <input type="checkbox" id="enterprise-approvals-self" ${approvals.allow_self_approval ? "checked" : ""}>
            <span class="administration-role-option-title">${escapeHtml(ctx.t("administration.enterprise.approvals.self"))}</span>
          </label>
          <div class="waf-field">
            <label for="enterprise-approvals-reviewers">${escapeHtml(ctx.t("administration.enterprise.approvals.reviewers"))}</label>
            <input id="enterprise-approvals-reviewers" value="${escapeHtml((approvals.reviewer_role_ids || []).join(", "))}">
          </div>
        </div>
      </section>
      <section class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("administration.enterprise.scim.title"))}</div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.enterprise.scim.subtitle"))}</div>
        <div class="waf-form-grid">
          <label class="administration-role-option">
            <input type="checkbox" id="enterprise-scim-enabled" ${scim.enabled ? "checked" : ""}>
            <span class="administration-role-option-title">${escapeHtml(ctx.t("administration.enterprise.scim.enabled"))}</span>
          </label>
          <div class="waf-field">
            <label for="enterprise-scim-default-roles">${escapeHtml(ctx.t("administration.enterprise.defaultRoles"))}</label>
            <input id="enterprise-scim-default-roles" value="${escapeHtml((scim.default_role_ids || []).join(", "))}">
          </div>
          <div class="waf-field full">
            <label for="enterprise-scim-mappings">${escapeHtml(ctx.t("administration.enterprise.roleMappings"))}</label>
            <textarea id="enterprise-scim-mappings" rows="4" placeholder="security-admins=admin,manager&#10;soc-team=soc">${escapeHtml(formatRoleMappings(scim.group_role_mappings || []))}</textarea>
          </div>
        </div>
        <div class="waf-inline">
          <button class="btn ghost btn-sm" type="button" id="enterprise-scim-token-create">${escapeHtml(ctx.t("administration.enterprise.scim.createToken"))}</button>
          <button class="btn ghost btn-sm" type="button" id="enterprise-support-bundle">${escapeHtml(ctx.t("administration.enterprise.supportBundle"))}</button>
        </div>
        <div class="waf-note" id="enterprise-scim-token-output"></div>
        <div class="waf-table-wrap">
          <table class="waf-table">
            <thead>
              <tr>
                <th>${escapeHtml(ctx.t("administration.enterprise.scim.tokenName"))}</th>
                <th>${escapeHtml(ctx.t("administration.enterprise.scim.tokenPrefix"))}</th>
                <th>${escapeHtml(ctx.t("administration.enterprise.scim.tokenUsed"))}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              ${(Array.isArray(scim.tokens) ? scim.tokens : []).map((item) => `
                <tr>
                  <td>${escapeHtml(item.display_name || "-")}</td>
                  <td><span class="waf-code">${escapeHtml(item.prefix || "-")}</span></td>
                  <td>${escapeHtml(formatDate(item.last_used_at || item.created_at || ""))}</td>
                  <td><button class="btn ghost btn-sm" type="button" data-enterprise-token-delete="${escapeHtml(item.id)}">${escapeHtml(ctx.t("common.delete"))}</button></td>
                </tr>
              `).join("")}
            </tbody>
          </table>
        </div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.enterprise.evidenceKey"))}: <span class="waf-code">${escapeHtml(enterprise?.evidence?.key_id || "-")}</span></div>
      </section>
    </div>
  `;
}
