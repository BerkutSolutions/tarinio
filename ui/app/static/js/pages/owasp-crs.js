import { escapeHtml, formatDate, setError } from "../ui.js";

const MAX_CONSOLE_LINES = 120;

function formatStatusDate(value, ctx) {
  if (!value) {
    return ctx.t("owaspCrs.status.na");
  }
  return formatDate(value);
}

function toConsoleLine(item, ctx) {
  const timestamp = escapeHtml(item?.at || "");
  const message = escapeHtml(item?.message || ctx.t("app.error"));
  const tone = String(item?.tone || "info").toLowerCase();
  return `<div class="waf-crs-console-line is-${escapeHtml(tone)}">[${timestamp}] ${message}</div>`;
}

function renderStatusCard(status, ctx, uiState) {
  const active = String(status?.active_version || ctx.t("owaspCrs.status.na"));
  const latest = String(status?.latest_version || ctx.t("owaspCrs.status.na"));
  const checkedAt = formatStatusDate(status?.last_checked_at, ctx);
  const releaseURL = String(status?.latest_release_url || "").trim();
  const hasUpdate = Boolean(status?.has_update);
  const activePath = String(status?.active_path || ctx.t("owaspCrs.status.na"));
  const firstStartPending = Boolean(status?.first_start_pending);
  const currentHourlyAuto = Boolean(status?.hourly_auto_update_enabled);
  const hourlyAuto = Boolean(uiState?.hourlyAutoDraft);
  const lastError = String(status?.last_error || "").trim();
  const consoleLines = Array.isArray(uiState?.consoleLines) ? uiState.consoleLines : [];
  const pendingAction = String(uiState?.pendingAction || "").trim();
  const busy = pendingAction !== "";
  const canSaveHourly = !busy && hourlyAuto !== currentHourlyAuto;
  return `
    <div class="waf-crs-status-head">
      <div class="waf-list-title">${escapeHtml(ctx.t("owaspCrs.status.title"))}</div>
      ${releaseURL ? `<a class="btn ghost btn-sm waf-crs-release-link" href="${escapeHtml(releaseURL)}" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("owaspCrs.status.releaseLink"))}</a>` : ""}
    </div>
    <div class="waf-crs-status-grid">
      <div class="waf-crs-status-tile">
        <span>${escapeHtml(ctx.t("owaspCrs.status.activeVersion"))}</span>
        <strong>${escapeHtml(active)}</strong>
      </div>
      <div class="waf-crs-status-tile">
        <span>${escapeHtml(ctx.t("owaspCrs.status.latestVersion"))}</span>
        <strong>${escapeHtml(latest)}</strong>
      </div>
      <div class="waf-crs-status-tile">
        <span>${escapeHtml(ctx.t("owaspCrs.status.lastChecked"))}</span>
        <strong>${escapeHtml(checkedAt)}</strong>
      </div>
      <div class="waf-crs-status-tile">
        <span>${escapeHtml(ctx.t("owaspCrs.status.hasUpdate"))}</span>
        <strong>${escapeHtml(ctx.t(hasUpdate ? "owaspCrs.status.yes" : "owaspCrs.status.no"))}</strong>
      </div>
      <div class="waf-crs-status-tile">
        <span>${escapeHtml(ctx.t("owaspCrs.status.firstStart"))}</span>
        <strong>${escapeHtml(ctx.t(firstStartPending ? "owaspCrs.status.pending" : "owaspCrs.status.done"))}</strong>
      </div>
      <div class="waf-crs-status-tile waf-crs-status-path">
        <span>${escapeHtml(ctx.t("owaspCrs.status.activePath"))}</span>
        <code>${escapeHtml(activePath)}</code>
      </div>
    </div>
    ${lastError ? `<div class="alert">${escapeHtml(lastError)}</div>` : ""}
    <div class="waf-stack">
      <label class="waf-checkbox">
        <input id="owasp-crs-hourly-auto" type="checkbox"${hourlyAuto ? " checked" : ""}${busy ? " disabled" : ""}>
        <span>${escapeHtml(ctx.t("owaspCrs.hourlyAuto"))}</span>
      </label>
      <div class="waf-actions">
        <button class="btn btn-sm" id="owasp-crs-save-hourly" type="button"${canSaveHourly ? "" : " disabled"}>${escapeHtml(ctx.t("owaspCrs.actions.save"))}</button>
      </div>
    </div>
    <div class="waf-actions">
      <button class="btn ghost btn-sm" id="owasp-crs-check" type="button"${busy ? " disabled" : ""}>${escapeHtml(ctx.t("owaspCrs.actions.check"))}</button>
      <button class="btn primary btn-sm" id="owasp-crs-update" type="button"${busy ? " disabled" : ""}>${escapeHtml(ctx.t("owaspCrs.actions.update"))}</button>
    </div>
    <div class="waf-crs-console-wrap">
      <div class="waf-list-title">${escapeHtml(ctx.t("owaspCrs.console.title"))}</div>
      <div class="waf-note">${escapeHtml(pendingAction || ctx.t("owaspCrs.console.idle"))}</div>
      <div class="waf-crs-console">${consoleLines.length > 0 ? consoleLines.map((item) => toConsoleLine(item, ctx)).join("") : `<div class="waf-crs-console-line is-muted">${escapeHtml(ctx.t("owaspCrs.console.empty"))}</div>`}</div>
    </div>
  `;
}

export async function renderOWASPCRS(container, ctx) {
  container.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("app.loading"))}</div>`;
  let state = null;
  const uiState = {
    pendingAction: "",
    hourlyAutoDraft: false,
    consoleLines: [],
  };

  function pushConsole(message, tone = "info") {
    const line = {
      at: new Date().toLocaleTimeString(),
      message: String(message || ctx.t("app.error")),
      tone: String(tone || "info"),
    };
    uiState.consoleLines = [line, ...uiState.consoleLines].slice(0, MAX_CONSOLE_LINES);
  }

  async function loadStatus() {
    state = await ctx.api.get("/api/owasp-crs/status");
    if (!uiState.pendingAction) {
      uiState.hourlyAutoDraft = Boolean(state?.hourly_auto_update_enabled);
    }
  }

  function render() {
    container.innerHTML = `
      <section class="waf-stack">
        <div class="waf-card">
          <div class="waf-card-head">
            <h3>${escapeHtml(ctx.t("owaspCrs.title"))}</h3>
          </div>
          <div class="waf-card-body waf-stack">
            <p class="waf-note">${escapeHtml(ctx.t("owaspCrs.description"))}</p>
            ${renderStatusCard(state, ctx, uiState)}
          </div>
        </div>
      </section>
    `;
    bind();
  }

  async function run(actionMessage, fn) {
    uiState.pendingAction = actionMessage;
    pushConsole(actionMessage, "info");
    render();
    try {
      const result = await fn();
      await loadStatus();
      pushConsole(ctx.t("owaspCrs.console.requestDone"), "success");
      if (state?.latest_version) {
        pushConsole(ctx.t("owaspCrs.console.latestVersion", { version: state.latest_version }), "info");
      }
      pushConsole(
        ctx.t("owaspCrs.console.updateAvailability", {
          value: ctx.t(state?.has_update ? "owaspCrs.status.yes" : "owaspCrs.status.no"),
        }),
        "info"
      );
      if (state?.active_version) {
        pushConsole(ctx.t("owaspCrs.console.activeVersion", { version: state.active_version }), "info");
      }
      render();
      ctx.notify(ctx.t("owaspCrs.actions.done"), "success");
      return result;
    } catch (error) {
      const message = error?.message || ctx.t("app.error");
      pushConsole(`${ctx.t("owaspCrs.console.failed")}: ${message}`, "error");
      render();
      ctx.notify(message, "error");
      return null;
    } finally {
      uiState.pendingAction = "";
      render();
    }
  }

  function bind() {
    container.querySelector("#owasp-crs-check")?.addEventListener("click", async () => {
      await run(ctx.t("owaspCrs.actions.checking"), async () => {
        return ctx.api.post("/api/owasp-crs/check-updates", { dry_run: true });
      });
    });

    container.querySelector("#owasp-crs-update")?.addEventListener("click", async () => {
      await run(ctx.t("owaspCrs.actions.updating"), async () => {
        return ctx.api.post("/api/owasp-crs/update", {});
      });
    });

    container.querySelector("#owasp-crs-hourly-auto")?.addEventListener("change", (event) => {
      uiState.hourlyAutoDraft = Boolean(event?.target?.checked);
      render();
    });

    container.querySelector("#owasp-crs-save-hourly")?.addEventListener("click", async () => {
      const enabled = Boolean(uiState.hourlyAutoDraft);
      await run(ctx.t("owaspCrs.actions.saving"), async () => {
        return ctx.api.post("/api/owasp-crs/update", { enable_hourly_auto_update: enabled });
      });
    });
  }

  try {
    await loadStatus();
    pushConsole(ctx.t("owaspCrs.console.initialized"), "success");
    render();
  } catch (error) {
    setError(container, error?.message || ctx.t("app.error"));
  }
}
