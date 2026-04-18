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
        <div class="waf-note">${escapeHtml(ctx.t("administration.scripts.file"))}: <span class="waf-code">${escapeHtml(script.file_name || "-")}</span></div>
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

export async function renderAdministration(container, ctx) {
  container.innerHTML = `
    <div class="waf-stack">
      <div class="waf-grid two">
        <section class="waf-card">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("administration.users.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.users.subtitle"))}</div></div></div>
          <div class="waf-card-body">
            <div id="users-status"></div>
          </div>
        </section>
        <section class="waf-card">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("administration.roles.title"))}</h3><div class="muted">${escapeHtml(ctx.t("administration.roles.subtitle"))}</div></div></div>
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
    </div>
  `;

  const usersStatus = container.querySelector("#users-status");
  const rolesStatus = container.querySelector("#roles-status");
  const scriptsStatus = container.querySelector("#administration-scripts-status");
  const latestRuns = new Map();

  setLoading(usersStatus, ctx.t("administration.users.loading"));
  setLoading(rolesStatus, ctx.t("administration.roles.loading"));
  setLoading(scriptsStatus, ctx.t("administration.scripts.loading"));

  try {
    const me = await ctx.api.get("/api/auth/me");
    usersStatus.innerHTML = `
      <div class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("administration.users.shellTitle"))}</div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.users.shellNote"))}</div>
        <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.users.currentUser"))}:</span><span class="waf-code">${escapeHtml(String(me?.username || me?.id || "-"))}</span></div>
        <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.users.currentRoles"))}:</span><span>${escapeHtml((Array.isArray(me?.role_ids) && me.role_ids.length ? me.role_ids.join(", ") : ctx.t("common.none")))}</span></div>
      </div>
    `;
  } catch (_error) {
    setError(usersStatus, ctx.t("administration.users.error.load"));
  }

  try {
    const me = await ctx.api.get("/api/auth/me");
    const permissions = Array.isArray(me?.permissions) ? me.permissions : [];
    rolesStatus.innerHTML = `
      <div class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("administration.roles.shellTitle"))}</div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.roles.shellNote"))}</div>
        <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("administration.roles.permissionCount"))}:</span><strong>${escapeHtml(String(permissions.length))}</strong></div>
        <div class="waf-note">${escapeHtml(ctx.t("administration.roles.permissionHint"))}</div>
      </div>
    `;
  } catch (_error) {
    setError(rolesStatus, ctx.t("administration.roles.error.load"));
  }

  const renderScripts = (catalog) => {
    const scripts = Array.isArray(catalog?.scripts) ? catalog.scripts : [];
    if (!scripts.length) {
      scriptsStatus.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("administration.scripts.empty"))}</div>`;
      return;
    }
    scriptsStatus.innerHTML = `
      <div class="waf-stack">
        ${scripts.map((script) => renderScriptCard(script, ctx, latestRuns.get(script.id) || null)).join("")}
      </div>
    `;

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
          if (!key) {
            return;
          }
          input[key] = String(inputNode.value || "");
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
            } catch (_error) {
              // ignore
            }
            throw new Error(message);
          }
          const blob = await response.blob();
          downloadBlob(fileName, blob);
        } catch (error) {
          notify(error?.message || ctx.t("administration.scripts.downloadError"), "error");
        }
      });
    });
  };

  try {
    const catalog = await ctx.api.get("/api/administration/scripts");
    renderScripts(catalog);
  } catch (_error) {
    setError(scriptsStatus, ctx.t("administration.scripts.error.load"));
  }
}
