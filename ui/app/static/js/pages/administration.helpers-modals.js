import { escapeHtml, formatDate } from "../ui.js";
import {
  formatPermissionLabel,
  formatRoleLabel,
  permissionGroups,
  renderPasswordField
} from "./administration.helpers-base.js";

export function renderUserModal(item, mode, roles, ctx) {
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
            ${editable ? renderPasswordField(ctx, { id: "administration-user-password", name: "password", label: ctx.t("administration.users.field.password"), placeholder: createMode ? ctx.t("administration.users.field.passwordCreateHint") : ctx.t("administration.users.field.passwordUpdateHint"), required: createMode }) : ""}
            ${editable ? renderPasswordField(ctx, { id: "administration-user-password-confirm", name: "password_confirm", label: ctx.t("administration.users.field.passwordConfirm"), placeholder: createMode ? ctx.t("administration.users.field.passwordCreateHint") : ctx.t("administration.users.field.passwordUpdateHint"), required: createMode }) : ""}
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

export function renderRoleModal(item, mode, availablePermissions, ctx) {
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

export function collectValues(form, name) {
  return Array.from(form.querySelectorAll(`[name="${name}"]:checked`))
    .map((node) => String(node.value || "").trim())
    .filter(Boolean);
}

export function openTableRow(event, attrName, items, open) {
  const rowNode = event.target.closest(`[${attrName}]`);
  if (!rowNode || event.target.closest("button")) {
    return;
  }
  const item = items.find((candidate) => candidate.id === String(rowNode.getAttribute(attrName) || ""));
  if (item) {
    open(item);
  }
}
