import { escapeHtml, formatDate, notify, setError, setLoading } from "../ui.js";

function downloadBlob(filename, blob) {
  const link = document.createElement("a");
  const href = URL.createObjectURL(blob);
  link.href = href;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(href);
}

function translateMaybe(ctx, key, fallback) {
  const normalizedKey = String(key || "").trim();
  if (normalizedKey) {
    const translated = ctx.t(normalizedKey);
    if (translated && translated !== normalizedKey) {
      return translated;
    }
  }
  return String(fallback || "");
}

function permissionGroups(ctx, permissions) {
  const groups = new Map();
  (Array.isArray(permissions) ? permissions : []).forEach((permission) => {
    const raw = String(permission || "").trim();
    if (!raw) {
      return;
    }
    const groupID = raw.split(".")[0] || "other";
    if (!groups.has(groupID)) {
      groups.set(groupID, []);
    }
    groups.get(groupID).push(raw);
  });
  return Array.from(groups.entries())
    .sort((left, right) => left[0].localeCompare(right[0]))
    .map(([id, items]) => ({
      id,
      title: translateMaybe(ctx, `administration.roles.group.${id}.title`, humanizePermissionToken(id)),
      hint: translateMaybe(ctx, `administration.roles.group.${id}.hint`, ""),
      permissions: items.sort((left, right) => left.localeCompare(right)),
    }));
}

function humanizePermissionToken(value) {
  return String(value || "")
    .split(/[._-]+/)
    .filter(Boolean)
    .map((item) => item.charAt(0).toUpperCase() + item.slice(1))
    .join(" ");
}

function formatPermissionLabel(ctx, permission) {
  return translateMaybe(
    ctx,
    `administration.roles.permission.${String(permission || "").trim()}`,
    humanizePermissionToken(String(permission || "").split(".").slice(-2).join(" "))
  );
}

function formatRoleLabel(ctx, role) {
  const roleID = String(role?.id || "").trim();
  return translateMaybe(
    ctx,
    `administration.roles.name.${roleID}`,
    String(role?.name || roleID || "")
  );
}

function renderPasswordToggleButton(ctx, targetID) {
  return `
    <button
      class="waf-password-toggle"
      type="button"
      data-password-toggle="${escapeHtml(targetID)}"
      data-visible="false"
      aria-pressed="false"
      title="${escapeHtml(ctx.t("administration.users.password.show"))}"
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

function syncPasswordToggle(button, visible, ctx) {
  button.dataset.visible = visible ? "true" : "false";
  button.setAttribute("aria-pressed", visible ? "true" : "false");
  button.title = ctx.t(visible ? "administration.users.password.hide" : "administration.users.password.show");
  button.querySelector(".waf-password-icon-eye")?.classList.toggle("waf-hidden", visible);
  button.querySelector(".waf-password-icon-off")?.classList.toggle("waf-hidden", !visible);
}

function bindPasswordToggles(root, ctx) {
  root.querySelectorAll("[data-password-toggle]").forEach((button) => {
    syncPasswordToggle(button, false, ctx);
    button.addEventListener("click", () => {
      const input = root.querySelector(`#${String(button.dataset.passwordToggle || "")}`);
      if (!input) {
        return;
      }
      const visible = input.type === "text";
      input.type = visible ? "password" : "text";
      syncPasswordToggle(button, !visible, ctx);
    });
  });
}

function renderPasswordField(ctx, { id, name, label, placeholder, disabled = false, required = false }) {
  return `
    <div class="waf-field">
      <label for="${escapeHtml(id)}">${escapeHtml(label)}</label>
      <div class="waf-password-field">
        <input
          id="${escapeHtml(id)}"
          name="${escapeHtml(name)}"
          type="password"
          ${placeholder ? `placeholder="${escapeHtml(placeholder)}"` : ""}
          ${disabled ? "disabled" : ""}
          ${required ? "required" : ""}
        >
        ${renderPasswordToggleButton(ctx, id)}
      </div>
    </div>
  `;
}

function renderField(script, field, ctx) {
  const inputID = `administration-script-${script.id}-${field.name}`;
  const type = String(field?.type || "text").toLowerCase();
  const valueAttr = field?.default_value ? ` value="${escapeHtml(field.default_value)}"` : "";
  const placeholderText = translateMaybe(ctx, field?.placeholder_key, field?.placeholder);
  const placeholderAttr = placeholderText ? ` placeholder="${escapeHtml(placeholderText)}"` : "";
  const requiredAttr = field?.required ? " required" : "";
  const helpText = translateMaybe(ctx, field?.help_text_key, field?.help_text);
  const help = helpText ? `<div class="waf-note">${escapeHtml(helpText)}</div>` : "";
  const labelText = translateMaybe(ctx, field?.label_key, field?.label || field?.name);
  if (type === "textarea") {
    return `
      <div class="waf-field full">
        <label for="${escapeHtml(inputID)}">${escapeHtml(labelText)}</label>
        <textarea id="${escapeHtml(inputID)}" data-script-input="${escapeHtml(field.name)}"${placeholderAttr}${requiredAttr}>${escapeHtml(field.default_value || "")}</textarea>
        ${help}
      </div>
    `;
  }
  return `
    <div class="waf-field">
      <label for="${escapeHtml(inputID)}">${escapeHtml(labelText)}</label>
      <input id="${escapeHtml(inputID)}" type="${escapeHtml(type === "password" ? "password" : "text")}" data-script-input="${escapeHtml(field.name)}"${valueAttr}${placeholderAttr}${requiredAttr}>
      ${help}
    </div>
  `;
}

function renderRunResult(result, ctx) {
  if (!result) {
    return `<div class="waf-note">${escapeHtml(ctx.t("administration.scripts.result.empty"))}</div>`;
  }
  const tone = result.status === "succeeded" ? "badge-success" : "badge-danger";
  const downloadButton = result.archive_name
    ? `<button type="button" class="btn ghost btn-sm" data-script-download="${escapeHtml(result.run_id)}" data-script-download-name="${escapeHtml(result.archive_name)}">${escapeHtml(ctx.t("administration.scripts.download"))}</button>`
    : "";
  const errorLine = result.error ? `<div class="alert">${escapeHtml(result.error)}</div>` : "";
  return `
    <div class="waf-subcard waf-stack waf-antiddos-frame">
      <div class="waf-inline">
        <span class="waf-note">${escapeHtml(ctx.t("administration.scripts.result.status"))}:</span>
        <span class="badge ${tone}">${escapeHtml(result.status || "-")}</span>
      </div>
      <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.scripts.result.finished"))}:</span><span>${escapeHtml(formatDate(result.finished_at || result.started_at || ""))}</span></div>
      <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.scripts.result.exitCode"))}:</span><span>${escapeHtml(String(result.exit_code ?? "-"))}</span></div>
      <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.scripts.result.archive"))}:</span><span class="waf-code">${escapeHtml(result.archive_name || ctx.t("common.none"))}</span></div>
      <div class="waf-inline">${downloadButton}</div>
      ${errorLine}
    </div>
  `;
}

function renderScriptCard(script, ctx, result) {
  const titleText = translateMaybe(ctx, script?.title_key, script?.title || script?.id);
  const descriptionText = translateMaybe(ctx, script?.description_key, script?.description || "");
  return `
    <section class="waf-subcard waf-stack waf-antiddos-frame administration-script-frame" data-script-frame="${escapeHtml(script.id)}">
      <div>
        <div class="waf-list-title">${escapeHtml(titleText)}</div>
        <div class="waf-note">${escapeHtml(descriptionText)}</div>
        <div class="waf-note administration-script-file-note">${escapeHtml(ctx.t("administration.scripts.file"))}: <span class="waf-code">${escapeHtml(script.file_name || "-")}</span></div>
      </div>
      <form class="waf-form administration-script-form" data-script-form="${escapeHtml(script.id)}">
        <div class="waf-form-grid">
          ${(Array.isArray(script.fields) ? script.fields : []).map((field) => renderField(script, field, ctx)).join("")}
        </div>
        <div class="waf-inline">
          <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("administration.scripts.run"))}</button>
          <span class="waf-note" data-script-running-note="${escapeHtml(script.id)}"></span>
        </div>
      </form>
      <div data-script-result="${escapeHtml(script.id)}">
        ${renderRunResult(result, ctx)}
      </div>
    </section>
  `;
}

function renderRoleList(roleIDs, roles, ctx) {
  const items = Array.isArray(roleIDs) ? roleIDs : [];
  if (!items.length) {
    return '<span class="waf-note">-</span>';
  }
  const roleByID = new Map((Array.isArray(roles) ? roles : []).map((role) => [String(role?.id || ""), role]));
  return `<div class="administration-role-pill-list">${items.map((roleID) => {
    const normalizedID = String(roleID || "");
    const role = roleByID.get(normalizedID) || { id: normalizedID, name: normalizedID };
    return `<span class="badge badge-neutral">${escapeHtml(formatRoleLabel(ctx, role))}</span>`;
  }).join("")}</div>`;
}

function renderUsersTable(users, roles, ctx) {
  const rows = Array.isArray(users) ? users : [];
  if (!rows.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("administration.users.empty"))}</div>`;
  }
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead>
          <tr>
            <th>${escapeHtml(ctx.t("administration.users.col.username"))}</th>
            <th>${escapeHtml(ctx.t("administration.users.col.email"))}</th>
            <th>${escapeHtml(ctx.t("administration.users.col.roles"))}</th>
            <th>${escapeHtml(ctx.t("administration.users.col.status"))}</th>
            <th>${escapeHtml(ctx.t("administration.users.col.lastLogin"))}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((item) => `
            <tr class="waf-table-row-clickable" data-user-open="${escapeHtml(item.id)}" tabindex="0" role="button">
              <td>
                <strong>${escapeHtml(item.username || item.id || "-")}</strong>
              </td>
              <td>${escapeHtml(item.email || "-")}</td>
              <td>${renderRoleList(item.role_ids, roles, ctx)}</td>
              <td>${escapeHtml(item.is_active ? ctx.t("administration.users.status.active") : ctx.t("administration.users.status.disabled"))}</td>
              <td>${escapeHtml(formatDate(item.last_login_at || ""))}</td>
              <td>
                <div class="table-actions administration-table-actions">
                  <button class="btn ghost btn-sm" type="button" data-user-edit="${escapeHtml(item.id)}">${escapeHtml(ctx.t("common.edit"))}</button>
                </div>
              </td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderRolesTable(roles, ctx) {
  const rows = Array.isArray(roles) ? roles : [];
  if (!rows.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("administration.roles.empty"))}</div>`;
  }
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead>
          <tr>
            <th>${escapeHtml(ctx.t("administration.roles.col.id"))}</th>
            <th>${escapeHtml(ctx.t("administration.roles.col.name"))}</th>
            <th>${escapeHtml(ctx.t("administration.roles.col.permissions"))}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${rows.map((item) => `
            <tr class="waf-table-row-clickable" data-role-open="${escapeHtml(item.id)}" tabindex="0" role="button">
              <td><span class="waf-code">${escapeHtml(item.id || "-")}</span></td>
              <td>${escapeHtml(item.name || item.id || "-")}</td>
              <td>${escapeHtml(ctx.t("administration.roles.permissionSummary", { count: String(Array.isArray(item.permissions) ? item.permissions.length : 0) }))}</td>
              <td>
                <div class="table-actions administration-table-actions">
                  <button class="btn ghost btn-sm" type="button" data-role-edit="${escapeHtml(item.id)}">${escapeHtml(ctx.t("common.edit"))}</button>
                </div>
              </td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function parseCSV(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseRoleMappings(value) {
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

function formatRoleMappings(items) {
  return (Array.isArray(items) ? items : [])
    .map((item) => `${String(item?.external_group || "").trim()}=${String((item?.role_ids || []).join(",")).trim()}`)
    .filter(Boolean)
    .join("\n");
}

function renderEnterprisePanel(state, ctx) {
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

function renderUserModal(item, mode, roles, ctx) {
  const editable = mode !== "view";
  const createMode = mode === "create";
  const roleIDs = new Set(Array.isArray(item?.role_ids) ? item.role_ids : []);
  return `
    <div class="modal-backdrop" data-close-modal="administration-entity-modal"></div>
    <div class="modal-body administration-modal-body">
      <div class="modal-header">
        <h3>${escapeHtml(mode === "create" ? ctx.t("administration.users.create") : mode === "edit" ? ctx.t("administration.users.edit") : ctx.t("administration.users.view"))}</h3>
        <button class="btn ghost" type="button" data-close-modal="administration-entity-modal">x</button>
      </div>
      <div class="modal-content">
        <div class="alert" id="administration-entity-alert" hidden></div>
        <form id="administration-user-form" class="waf-form" data-mode="${escapeHtml(mode)}" data-user-id="${escapeHtml(item?.id || "")}">
          <div class="waf-form-grid administration-modal-grid">
            <div class="waf-field">
              <label for="administration-user-username">${escapeHtml(ctx.t("administration.users.col.username"))}</label>
              <input id="administration-user-username" name="username" value="${escapeHtml(item?.username || "")}" ${editable && createMode ? "" : "disabled"}>
            </div>
            <div class="waf-field">
              <label for="administration-user-email">${escapeHtml(ctx.t("administration.users.col.email"))}</label>
              <input id="administration-user-email" name="email" value="${escapeHtml(item?.email || "")}" ${editable ? "" : "disabled"}>
            </div>
            <div class="waf-field">
              <label for="administration-user-department">${escapeHtml(ctx.t("administration.users.field.department"))}</label>
              <input id="administration-user-department" name="department" value="${escapeHtml(item?.department || "")}" ${editable ? "" : "disabled"}>
            </div>
            <div class="waf-field">
              <label for="administration-user-position">${escapeHtml(ctx.t("administration.users.field.position"))}</label>
              <input id="administration-user-position" name="position" value="${escapeHtml(item?.position || "")}" ${editable ? "" : "disabled"}>
            </div>
            ${editable ? renderPasswordField(ctx, {
              id: "administration-user-password",
              name: "password",
              label: ctx.t("administration.users.field.password"),
              placeholder: createMode ? ctx.t("administration.users.field.passwordCreateHint") : ctx.t("administration.users.field.passwordUpdateHint"),
              required: createMode,
            }) : ""}
            ${editable ? renderPasswordField(ctx, {
              id: "administration-user-password-confirm",
              name: "password_confirm",
              label: ctx.t("administration.users.field.passwordConfirm"),
              placeholder: createMode ? ctx.t("administration.users.field.passwordCreateHint") : ctx.t("administration.users.field.passwordUpdateHint"),
              required: createMode,
            }) : ""}
            <div class="waf-field">
              <label for="administration-user-status">${escapeHtml(ctx.t("administration.users.col.status"))}</label>
              <select id="administration-user-status" name="is_active" ${editable ? "" : "disabled"}>
                <option value="true"${item?.is_active !== false ? " selected" : ""}>${escapeHtml(ctx.t("administration.users.status.active"))}</option>
                <option value="false"${item?.is_active === false ? " selected" : ""}>${escapeHtml(ctx.t("administration.users.status.disabled"))}</option>
              </select>
            </div>
            <div class="waf-field full">
              <label>${escapeHtml(ctx.t("administration.users.col.roles"))}</label>
              <div class="administration-role-picker">
                ${(Array.isArray(roles) ? roles : []).map((role) => `
                  <label class="administration-role-option">
                    <input type="checkbox" name="role_ids" value="${escapeHtml(role.id)}"${roleIDs.has(role.id) ? " checked" : ""}${editable ? "" : " disabled"}>
                    <span class="administration-role-option-title">${escapeHtml(formatRoleLabel(ctx, role))}</span>
                  </label>
                `).join("")}
              </div>
            </div>
          </div>
          ${mode === "view" ? `
            <div class="administration-modal-meta">
              <div class="administration-modal-meta-item"><span>${escapeHtml(ctx.t("administration.users.meta.created"))}</span><strong>${escapeHtml(formatDate(item?.created_at || ""))}</strong></div>
              <div class="administration-modal-meta-item"><span>${escapeHtml(ctx.t("administration.users.meta.updated"))}</span><strong>${escapeHtml(formatDate(item?.updated_at || ""))}</strong></div>
              <div class="administration-modal-meta-item"><span>${escapeHtml(ctx.t("administration.users.meta.lastLogin"))}</span><strong>${escapeHtml(formatDate(item?.last_login_at || ""))}</strong></div>
              <div class="administration-modal-meta-item"><span>${escapeHtml(ctx.t("administration.users.meta.totp"))}</span><strong>${escapeHtml(item?.totp_enabled ? ctx.t("auth.2fa.status.enabled") : ctx.t("auth.2fa.status.disabled"))}</strong></div>
            </div>
          ` : ""}
          <div class="waf-actions">
            ${editable ? `<button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("common.save"))}</button>` : ""}
            <button class="btn ghost btn-sm" type="button" data-close-modal="administration-entity-modal">${escapeHtml(ctx.t("common.close"))}</button>
          </div>
        </form>
      </div>
    </div>
  `;
}

function renderRoleModal(item, mode, availablePermissions, ctx) {
  const editable = mode !== "view";
  const groups = permissionGroups(ctx, availablePermissions);
  const selected = new Set(Array.isArray(item?.permissions) ? item.permissions : []);
  return `
    <div class="modal-backdrop" data-close-modal="administration-entity-modal"></div>
    <div class="modal-body administration-modal-body administration-modal-body-wide">
      <div class="modal-header">
        <h3>${escapeHtml(mode === "create" ? ctx.t("administration.roles.create") : mode === "edit" ? ctx.t("administration.roles.edit") : ctx.t("administration.roles.view"))}</h3>
        <button class="btn ghost" type="button" data-close-modal="administration-entity-modal">x</button>
      </div>
      <div class="modal-content">
        <div class="alert" id="administration-entity-alert" hidden></div>
        <form id="administration-role-form" class="waf-form" data-mode="${escapeHtml(mode)}" data-role-id="${escapeHtml(item?.id || "")}">
          <div class="waf-form-grid administration-modal-grid">
            <div class="waf-field">
              <label for="administration-role-id">${escapeHtml(ctx.t("administration.roles.col.id"))}</label>
              <input id="administration-role-id" name="id" value="${escapeHtml(item?.id || "")}" ${editable && mode === "create" ? "" : "disabled"}>
            </div>
            <div class="waf-field">
              <label for="administration-role-name">${escapeHtml(ctx.t("administration.roles.col.name"))}</label>
              <input id="administration-role-name" name="name" value="${escapeHtml(item?.name || "")}" ${editable ? "" : "disabled"}>
            </div>
            <div class="waf-field full">
              <label>${escapeHtml(ctx.t("administration.roles.col.permissions"))}</label>
              <div class="waf-note administration-modal-note">${escapeHtml(ctx.t("administration.roles.permissionsHelp"))}</div>
              <div class="administration-role-groups">
                ${groups.map((group) => `
                  <section class="administration-role-group">
                    <div class="administration-role-group-head">
                      <div>
                        <div class="administration-role-group-title">${escapeHtml(group.title)}</div>
                        ${group.hint ? `<div class="administration-role-group-note">${escapeHtml(group.hint)}</div>` : ""}
                      </div>
                      <span class="badge badge-neutral">${escapeHtml(ctx.t("administration.roles.permissionSummary", { count: String(group.permissions.length) }))}</span>
                    </div>
                    <div class="administration-permission-grid">
                      ${group.permissions.map((permission) => `
                        <label class="administration-permission-item">
                          <input type="checkbox" name="permissions" value="${escapeHtml(permission)}"${selected.has(permission) ? " checked" : ""}${editable ? "" : " disabled"}>
                          <span class="administration-permission-text">
                            <span class="administration-permission-label">${escapeHtml(formatPermissionLabel(ctx, permission))}</span>
                            <span class="administration-permission-code">${escapeHtml(permission)}</span>
                          </span>
                        </label>
                      `).join("")}
                    </div>
                  </section>
                `).join("")}
              </div>
            </div>
          </div>
          <div class="waf-actions">
            ${editable ? `<button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("common.save"))}</button>` : ""}
            <button class="btn ghost btn-sm" type="button" data-close-modal="administration-entity-modal">${escapeHtml(ctx.t("common.close"))}</button>
          </div>
        </form>
      </div>
    </div>
  `;
}

function collectValues(form, name) {
  return Array.from(form.querySelectorAll(`[name="${name}"]:checked`))
    .map((node) => String(node.value || "").trim())
    .filter(Boolean);
}

function openTableRow(event, attrName, items, open) {
  const rowNode = event.target.closest(`[${attrName}]`);
  if (!rowNode || event.target.closest("button")) {
    return;
  }
  const item = items.find((candidate) => candidate.id === String(rowNode.getAttribute(attrName) || ""));
  if (item) {
    open(item);
  }
}

export async function renderAdministration(container, ctx) {
  container.innerHTML = `
    <div class="waf-stack">
      <div class="waf-grid two">
        <section class="waf-card">
          <div class="waf-card-head">
            <div><h3>${escapeHtml(ctx.t("administration.users.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.users.subtitle"))}</div></div>
            <div class="waf-actions"><button class="btn primary btn-sm" type="button" id="administration-user-create">${escapeHtml(ctx.t("administration.users.create"))}</button></div>
          </div>
          <div class="waf-card-body">
            <div id="users-status"></div>
          </div>
        </section>
        <section class="waf-card">
          <div class="waf-card-head">
            <div><h3>${escapeHtml(ctx.t("administration.roles.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.roles.subtitle"))}</div></div>
            <div class="waf-actions"><button class="btn primary btn-sm" type="button" id="administration-role-create">${escapeHtml(ctx.t("administration.roles.create"))}</button></div>
          </div>
          <div class="waf-card-body">
            <div id="roles-status"></div>
          </div>
        </section>
      </div>
      <section class="waf-card">
        <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("administration.scripts.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.scripts.subtitle"))}</div></div></div>
        <div class="waf-card-body waf-stack">
          <div class="waf-note">${escapeHtml(ctx.t("administration.scripts.note"))}</div>
          <div id="administration-scripts-status"></div>
        </div>
      </section>
      <section class="waf-card">
        <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("administration.enterprise.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.enterprise.subtitle"))}</div></div></div>
        <div class="waf-card-body waf-stack">
          <div id="administration-enterprise-status"></div>
        </div>
      </section>
      <div class="modal" id="administration-entity-modal" hidden></div>
    </div>
  `;

  const usersStatus = container.querySelector("#users-status");
  const rolesStatus = container.querySelector("#roles-status");
  const scriptsStatus = container.querySelector("#administration-scripts-status");
  const enterpriseStatus = container.querySelector("#administration-enterprise-status");
  const modal = container.querySelector("#administration-entity-modal");
  const latestRuns = new Map();
  const state = { users: [], roles: [], availablePermissions: [], enterprise: null };

  const closeModal = () => {
    modal.hidden = true;
    modal.innerHTML = "";
  };

  const showModalAlert = (message) => {
    const node = modal.querySelector("#administration-entity-alert");
    if (!node) {
      return;
    }
    const text = String(message || "").trim();
    node.hidden = !text;
    node.textContent = text;
  };

  const renderUsers = () => {
    usersStatus.innerHTML = renderUsersTable(state.users, state.roles, ctx);
  };

  const renderRoles = () => {
    rolesStatus.innerHTML = renderRolesTable(state.roles, ctx);
  };

  const renderEnterprise = () => {
    if (!state.enterprise) {
      setLoading(enterpriseStatus, ctx.t("administration.enterprise.loading"));
      return;
    }
    enterpriseStatus.innerHTML = renderEnterprisePanel(state, ctx);
  };

  const loadUsers = async () => {
    setLoading(usersStatus, ctx.t("administration.users.loading"));
    const payload = await ctx.api.get("/api/administration/users");
    state.users = Array.isArray(payload?.users) ? payload.users : [];
    renderUsers();
  };

  const loadRoles = async () => {
    setLoading(rolesStatus, ctx.t("administration.roles.loading"));
    const payload = await ctx.api.get("/api/administration/roles");
    state.roles = Array.isArray(payload?.roles) ? payload.roles : [];
    state.availablePermissions = Array.isArray(payload?.available_permissions) ? payload.available_permissions : [];
    renderRoles();
  };

  const loadEnterprise = async () => {
    setLoading(enterpriseStatus, ctx.t("administration.enterprise.loading"));
    state.enterprise = await ctx.api.get("/api/administration/enterprise");
    renderEnterprise();
  };

  const openUserModal = (item, mode) => {
    modal.hidden = false;
    modal.innerHTML = renderUserModal(item, mode, state.roles, ctx);
    bindPasswordToggles(modal, ctx);
  };

  const openRoleModal = (item, mode) => {
    modal.hidden = false;
    modal.innerHTML = renderRoleModal(item, mode, state.availablePermissions, ctx);
  };

  const renderScripts = (catalog) => {
    const scripts = Array.isArray(catalog?.scripts) ? catalog.scripts : [];
    if (!scripts.length) {
      scriptsStatus.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("administration.scripts.empty"))}</div>`;
      return;
    }
    scriptsStatus.innerHTML = `<div class="waf-stack">${scripts.map((script) => renderScriptCard(script, ctx, latestRuns.get(script.id) || null)).join("")}</div>`;

    scriptsStatus.querySelectorAll("[data-script-form]").forEach((formNode) => {
      formNode.addEventListener("submit", async (event) => {
        event.preventDefault();
        const scriptID = String(formNode.getAttribute("data-script-form") || "").trim();
        if (!scriptID) {
          return;
        }
        const runningNote = scriptsStatus.querySelector(`[data-script-running-note="${scriptID}"]`);
        if (runningNote) {
          runningNote.textContent = ctx.t("administration.scripts.running");
        }
        const input = {};
        formNode.querySelectorAll("[data-script-input]").forEach((inputNode) => {
          const key = String(inputNode.getAttribute("data-script-input") || "").trim();
          if (key) {
            input[key] = String(inputNode.value || "");
          }
        });
        try {
          const result = await ctx.api.post(`/api/administration/scripts/${encodeURIComponent(scriptID)}/run`, { input });
          latestRuns.set(scriptID, result);
          renderScripts(catalog);
          notify(ctx.t("administration.scripts.runSuccess"));
        } catch (error) {
          const resultNode = scriptsStatus.querySelector(`[data-script-result="${scriptID}"]`);
          if (resultNode) {
            resultNode.innerHTML = `<div class="alert">${escapeHtml(error?.message || ctx.t("administration.scripts.runError"))}</div>`;
          }
        } finally {
          if (runningNote) {
            runningNote.textContent = "";
          }
        }
      });
    });

    scriptsStatus.querySelectorAll("[data-script-download]").forEach((buttonNode) => {
      buttonNode.addEventListener("click", async () => {
        const runID = String(buttonNode.getAttribute("data-script-download") || "").trim();
        const fileName = String(buttonNode.getAttribute("data-script-download-name") || "script-output.tar.gz").trim();
        if (!runID) {
          return;
        }
        try {
          const response = await fetch(`/api/administration/scripts/runs/${encodeURIComponent(runID)}/download`, {
            method: "GET",
            credentials: "include",
            headers: { Accept: "application/gzip" }
          });
          if (!response.ok) {
            let message = `HTTP ${response.status}`;
            try {
              const payload = await response.json();
              if (payload?.error) {
                message = String(payload.error);
              }
            } catch {
              // ignore parse errors
            }
            throw new Error(message);
          }
          downloadBlob(fileName, await response.blob());
        } catch (error) {
          notify(error?.message || ctx.t("administration.scripts.downloadError"), "error");
        }
      });
    });
  };

  container.querySelector("#administration-user-create")?.addEventListener("click", () => openUserModal({}, "create"));
  container.querySelector("#administration-role-create")?.addEventListener("click", () => openRoleModal({}, "create"));

  usersStatus.addEventListener("click", (event) => {
    const editNode = event.target.closest("[data-user-edit]");
    if (editNode) {
      event.preventDefault();
      const item = state.users.find((candidate) => candidate.id === String(editNode.getAttribute("data-user-edit") || ""));
      if (item) {
        openUserModal(item, "edit");
      }
      return;
    }
    openTableRow(event, "data-user-open", state.users, (item) => openUserModal(item, "view"));
  });

  usersStatus.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    openTableRow(event, "data-user-open", state.users, (item) => openUserModal(item, "view"));
  });

  rolesStatus.addEventListener("click", (event) => {
    const editNode = event.target.closest("[data-role-edit]");
    if (editNode) {
      event.preventDefault();
      const item = state.roles.find((candidate) => candidate.id === String(editNode.getAttribute("data-role-edit") || ""));
      if (item) {
        openRoleModal(item, "edit");
      }
      return;
    }
    openTableRow(event, "data-role-open", state.roles, (item) => openRoleModal(item, "view"));
  });

  rolesStatus.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    openTableRow(event, "data-role-open", state.roles, (item) => openRoleModal(item, "view"));
  });

  modal.addEventListener("click", (event) => {
    if (event.target.closest("[data-close-modal]")) {
      closeModal();
    }
  });

  modal.addEventListener("submit", async (event) => {
    event.preventDefault();

    const userForm = event.target.closest("#administration-user-form");
    if (userForm) {
      showModalAlert("");
      const password = String(userForm.elements.password?.value || "");
      const passwordConfirm = String(userForm.elements.password_confirm?.value || "");
      const createMode = userForm.dataset.mode === "create";
      if (createMode && !password) {
        showModalAlert(ctx.t("administration.users.error.passwordRequired"));
        return;
      }
      if (password !== passwordConfirm) {
        showModalAlert(ctx.t("administration.users.error.passwordMismatch"));
        return;
      }
      const payload = {
        username: String(userForm.elements.username?.value || "").trim(),
        email: String(userForm.elements.email?.value || "").trim(),
        department: String(userForm.elements.department?.value || "").trim(),
        position: String(userForm.elements.position?.value || "").trim(),
        password,
        is_active: String(userForm.elements.is_active?.value || "true") === "true",
        role_ids: collectValues(userForm, "role_ids"),
      };
      try {
        if (createMode) {
          await ctx.api.post("/api/administration/users", payload);
        } else {
          await ctx.api.put(`/api/administration/users/${encodeURIComponent(String(userForm.dataset.userId || ""))}`, payload);
        }
        closeModal();
        await loadUsers();
      } catch (error) {
        showModalAlert(error?.message || ctx.t("common.error"));
      }
      return;
    }

    const roleForm = event.target.closest("#administration-role-form");
    if (!roleForm) {
      return;
    }
    showModalAlert("");
    const payload = {
      id: String(roleForm.elements.id?.value || "").trim(),
      name: String(roleForm.elements.name?.value || "").trim(),
      permissions: collectValues(roleForm, "permissions"),
    };
    try {
      if (roleForm.dataset.mode === "create") {
        await ctx.api.post("/api/administration/roles", payload);
      } else {
        await ctx.api.put(`/api/administration/roles/${encodeURIComponent(String(roleForm.dataset.roleId || ""))}`, payload);
      }
      closeModal();
      await loadRoles();
      await loadUsers();
    } catch (error) {
      showModalAlert(error?.message || ctx.t("common.error"));
    }
  });

  setLoading(scriptsStatus, ctx.t("administration.scripts.loading"));

  try {
    await Promise.all([loadUsers(), loadRoles(), loadEnterprise()]);
  } catch (error) {
    if (!state.users.length) {
      setError(usersStatus, error?.message || ctx.t("administration.users.error.load"));
    }
    if (!state.roles.length) {
      setError(rolesStatus, error?.message || ctx.t("administration.roles.error.load"));
    }
    if (!state.enterprise) {
      setError(enterpriseStatus, error?.message || ctx.t("administration.enterprise.error.load"));
    }
  }

  try {
    const catalog = await ctx.api.get("/api/administration/scripts");
    renderScripts(catalog);
  } catch (_error) {
    setError(scriptsStatus, ctx.t("administration.scripts.error.load"));
  }

  enterpriseStatus.addEventListener("submit", async (event) => {
    const form = event.target.closest("#administration-enterprise-form");
    if (!form) {
      return;
    }
    event.preventDefault();
    try {
      await ctx.api.put("/api/administration/enterprise", {
        oidc: {
          enabled: Boolean(enterpriseStatus.querySelector("#enterprise-oidc-enabled")?.checked),
          display_name: enterpriseStatus.querySelector("#enterprise-oidc-name")?.value || "",
          issuer_url: enterpriseStatus.querySelector("#enterprise-oidc-issuer")?.value || "",
          client_id: enterpriseStatus.querySelector("#enterprise-oidc-client-id")?.value || "",
          client_secret: enterpriseStatus.querySelector("#enterprise-oidc-client-secret")?.value || "",
          redirect_url: enterpriseStatus.querySelector("#enterprise-oidc-redirect")?.value || "",
          default_role_ids: parseCSV(enterpriseStatus.querySelector("#enterprise-oidc-default-roles")?.value || ""),
          allowed_email_domains: parseCSV(enterpriseStatus.querySelector("#enterprise-oidc-domains")?.value || ""),
          group_role_mappings: parseRoleMappings(enterpriseStatus.querySelector("#enterprise-oidc-mappings")?.value || ""),
          auto_provision: true,
          require_verified_email: true,
        },
        approvals: {
          enabled: Boolean(enterpriseStatus.querySelector("#enterprise-approvals-enabled")?.checked),
          required_approvals: Number.parseInt(String(enterpriseStatus.querySelector("#enterprise-approvals-count")?.value || "1"), 10) || 1,
          allow_self_approval: Boolean(enterpriseStatus.querySelector("#enterprise-approvals-self")?.checked),
          reviewer_role_ids: parseCSV(enterpriseStatus.querySelector("#enterprise-approvals-reviewers")?.value || ""),
        },
        scim: {
          enabled: Boolean(enterpriseStatus.querySelector("#enterprise-scim-enabled")?.checked),
          default_role_ids: parseCSV(enterpriseStatus.querySelector("#enterprise-scim-default-roles")?.value || ""),
          group_role_mappings: parseRoleMappings(enterpriseStatus.querySelector("#enterprise-scim-mappings")?.value || ""),
        },
      });
      notify(ctx.t("administration.enterprise.saved"));
      await loadEnterprise();
    } catch (error) {
      notify(error?.message || ctx.t("common.error"), "error");
    }
  });

  enterpriseStatus.addEventListener("click", async (event) => {
    const createToken = event.target.closest("#enterprise-scim-token-create");
    if (createToken) {
      const displayName = window.prompt(ctx.t("administration.enterprise.scim.prompt")) || "";
      if (!String(displayName).trim()) {
        return;
      }
      try {
        const result = await ctx.api.post("/api/administration/enterprise/scim-tokens", { display_name: displayName });
        const output = enterpriseStatus.querySelector("#enterprise-scim-token-output");
        if (output) {
          output.textContent = `${ctx.t("administration.enterprise.scim.tokenCreated")}: ${result.token}`;
        }
        await loadEnterprise();
      } catch (error) {
        notify(error?.message || ctx.t("common.error"), "error");
      }
      return;
    }
    const deleteToken = event.target.closest("[data-enterprise-token-delete]");
    if (deleteToken) {
      try {
        await ctx.api.delete(`/api/administration/enterprise/scim-tokens/${encodeURIComponent(String(deleteToken.getAttribute("data-enterprise-token-delete") || ""))}`);
        await loadEnterprise();
      } catch (error) {
        notify(error?.message || ctx.t("common.error"), "error");
      }
      return;
    }
    const supportBundle = event.target.closest("#enterprise-support-bundle");
    if (supportBundle) {
      try {
        const response = await fetch("/api/administration/support-bundle", { credentials: "include" });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const disposition = response.headers.get("content-disposition") || "";
        const match = disposition.match(/filename=\"?([^"]+)\"?/i);
        downloadBlob(match?.[1] || "support-bundle.tar.gz", await response.blob());
      } catch (error) {
        notify(error?.message || ctx.t("common.error"), "error");
      }
    }
  });
}
