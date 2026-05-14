import { escapeHtml, formatDate } from "../ui.js";

export function downloadBlob(filename, blob) {
  const link = document.createElement("a");
  const href = URL.createObjectURL(blob);
  link.href = href;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(href);
}

export function translateMaybe(ctx, key, fallback) {
  const normalizedKey = String(key || "").trim();
  if (normalizedKey) {
    const translated = ctx.t(normalizedKey);
    if (translated && translated !== normalizedKey) {
      return translated;
    }
  }
  return String(fallback || "");
}

export function permissionGroups(ctx, permissions) {
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

export function humanizePermissionToken(value) {
  return String(value || "")
    .split(/[._-]+/)
    .filter(Boolean)
    .map((item) => item.charAt(0).toUpperCase() + item.slice(1))
    .join(" ");
}

export function formatPermissionLabel(ctx, permission) {
  return translateMaybe(
    ctx,
    `administration.roles.permission.${String(permission || "").trim()}`,
    humanizePermissionToken(String(permission || "").split(".").slice(-2).join(" "))
  );
}

export function formatRoleLabel(ctx, role) {
  const roleID = String(role?.id || "").trim();
  return translateMaybe(
    ctx,
    `administration.roles.name.${roleID}`,
    String(role?.name || roleID || "")
  );
}

export function renderPasswordToggleButton(ctx, targetID) {
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

export function syncPasswordToggle(button, visible, ctx) {
  button.dataset.visible = visible ? "true" : "false";
  button.setAttribute("aria-pressed", visible ? "true" : "false");
  button.title = ctx.t(visible ? "administration.users.password.hide" : "administration.users.password.show");
  button.querySelector(".waf-password-icon-eye")?.classList.toggle("waf-hidden", visible);
  button.querySelector(".waf-password-icon-off")?.classList.toggle("waf-hidden", !visible);
}

export function bindPasswordToggles(root, ctx) {
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

export function renderPasswordField(ctx, { id, name, label, placeholder, disabled = false, required = false }) {
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

export function renderField(script, field, ctx) {
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

export function renderRunResult(result, ctx) {
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

export function renderScriptCard(script, ctx, latestRun = null) {
  const scriptID = String(script?.id || "").trim();
  const title = translateMaybe(ctx, script?.title_key, script?.title || scriptID);
  const description = translateMaybe(ctx, script?.description_key, script?.description || "");
  const fields = Array.isArray(script?.inputs) ? script.inputs : [];
  const fieldsHTML = fields.map((field) => renderField(script, field, ctx)).join("");
  const runResultHTML = renderRunResult(latestRun, ctx);
  const submitLabel = escapeHtml(ctx.t("administration.scripts.run"));

  return `
    <section class="waf-subcard waf-stack">
      <div>
        <h4>${escapeHtml(title)}</h4>
        ${description ? `<div class="waf-note">${escapeHtml(description)}</div>` : ""}
        <div class="waf-code">${escapeHtml(scriptID)}</div>
      </div>
      <form class="waf-form waf-stack" data-script-form="${escapeHtml(scriptID)}">
        <div class="waf-form-grid">${fieldsHTML}</div>
        <div class="waf-inline">
          <button type="submit" class="btn primary btn-sm">${submitLabel}</button>
          <span class="waf-note" data-script-running-note="${escapeHtml(scriptID)}"></span>
        </div>
      </form>
      <div data-script-result="${escapeHtml(scriptID)}">${runResultHTML}</div>
    </section>
  `;
}
