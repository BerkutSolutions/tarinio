import { availableTimeZones, formatDateTimeInZone, loadPreferences, savePreferences } from "../preferences.js";
import { escapeHtml } from "../ui.js";

function showAlert(el, msg, success = false) {
  if (!el) return;
  el.textContent = String(msg || "");
  el.hidden = !msg;
  el.classList.toggle("success", !!success);
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function modalOpen(id) {
  const node = document.getElementById(id);
  if (node) node.hidden = false;
}

function modalClose(id) {
  const node = document.getElementById(id);
  if (node) node.hidden = true;
}

function bindModalClose(root) {
  root.querySelectorAll("[data-close]").forEach((node) => {
    node.addEventListener("click", (event) => {
      event.preventDefault();
      const target = String(node.getAttribute("data-close") || "").trim();
      if (target) {
        modalClose(target.replace("#", ""));
      }
    });
  });
}

function webAuthnSupported() {
  return !!(window.BerkutWebAuthn && window.BerkutWebAuthn.supported && window.BerkutWebAuthn.supported());
}

function formatDateTime(value, prefs) {
  if (!value) return "-";
  return formatDateTimeInZone(value, prefs.timeZone || "Europe/Moscow");
}

async function refresh2FA(ctx) {
  return ctx.api.get("/api/auth/2fa/status");
}

async function refreshPasskeys(ctx) {
  return ctx.api.get("/api/auth/passkeys");
}

export async function renderProfile(container, ctx) {
  container.innerHTML = `
    <div class="waf-page-stack" id="profile-page">
      <div class="alert" id="profile-alert" hidden></div>

      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("profile.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("profile.subtitle"))}</div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="waf-grid two profile-overview-grid">
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.field.username"))}:</span>
                <strong id="profile-username">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.fullName"))}:</span>
                <strong id="profile-fullname">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.department"))}:</span>
                <strong id="profile-department">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.position"))}:</span>
                <strong id="profile-position">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.sessionStarted"))}:</span>
                <strong id="profile-session-start">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.sessionExpires"))}:</span>
                <strong id="profile-session-expire">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.lastLoginIp"))}:</span>
                <strong id="profile-last-login-ip">-</strong>
              </div>
            </div>
            <div class="waf-list-item profile-overview-item">
              <div class="profile-overview-inline-row">
                <span class="profile-overview-label">${escapeHtml(ctx.t("profile.trustedIp"))}:</span>
                <strong id="profile-trusted-ip">-</strong>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="waf-card" id="password-change-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("accounts.passwordChange"))}</h3>
            <div class="muted" id="password-last-changed"></div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <form id="password-change-form" class="waf-form">
            <div class="waf-form-grid two profile-password-grid">
              <div class="waf-field required">
                <label>${escapeHtml(ctx.t("accounts.currentPassword"))}</label>
                <input type="password" name="current_password" autocomplete="current-password" required>
              </div>
              <div class="waf-field required">
                <label>${escapeHtml(ctx.t("accounts.newPassword"))}</label>
                <input type="password" name="password" autocomplete="new-password" required>
              </div>
              <div class="waf-field required">
                <label>${escapeHtml(ctx.t("accounts.passwordConfirm"))}</label>
                <input type="password" name="password_confirm" autocomplete="new-password" required>
              </div>
            </div>
            <div class="waf-actions profile-actions-row">
              <button type="submit" class="btn primary btn-sm">${escapeHtml(ctx.t("accounts.passwordChange"))}</button>
            </div>
          </form>
        </div>
      </section>

      <div class="waf-grid two profile-security-row">
        <section class="waf-card" id="twofa-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("auth.2fa.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("auth.2fa.subtitle"))}</div>
            </div>
            <div class="waf-actions">
              <button class="btn primary btn-sm" id="twofa-enable-btn">${escapeHtml(ctx.t("auth.2fa.enable"))}</button>
              <button class="btn ghost danger btn-sm" id="twofa-disable-btn" hidden>${escapeHtml(ctx.t("auth.2fa.disable"))}</button>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="waf-note" id="twofa-status-text">${escapeHtml(ctx.t("auth.2fa.status.unknown"))}</div>
            <div class="waf-note" id="twofa-recovery-remaining" hidden></div>
          </div>
        </section>

        <section class="waf-card" id="passkeys-card">
          <div class="waf-card-head">
            <div>
              <h3>${escapeHtml(ctx.t("auth.passkeys.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("auth.passkeys.subtitle"))}</div>
            </div>
            <div class="waf-actions">
              <button class="btn primary btn-sm" id="passkeys-add-btn">${escapeHtml(ctx.t("auth.passkeys.add"))}</button>
            </div>
          </div>
          <div class="waf-card-body waf-stack">
            <div class="waf-note" id="passkeys-status-text">${escapeHtml(ctx.t("auth.passkeys.status.loading"))}</div>
            <div class="waf-note" id="passkeys-unsupported" hidden>${escapeHtml(ctx.t("auth.passkeys.unsupported"))}</div>
            <div class="waf-table-wrap" id="passkeys-table-wrap" hidden>
              <table class="waf-table">
                <thead>
                  <tr>
                    <th>${escapeHtml(ctx.t("auth.passkeys.table.name"))}</th>
                    <th>${escapeHtml(ctx.t("auth.passkeys.table.created"))}</th>
                    <th>${escapeHtml(ctx.t("auth.passkeys.table.lastUsed"))}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody id="passkeys-table-body"></tbody>
              </table>
            </div>
          </div>
        </section>
      </div>

      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("settings.general"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("profile.preferences.saved"))}</div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <form id="settings-form" class="waf-form">
            <div class="waf-grid two">
              <div class="waf-list-item">
                <div class="waf-list-head">
                  <div class="waf-list-title">${escapeHtml(ctx.t("settings.timezone.label"))}</div>
                </div>
                <div class="waf-field">
                  <label for="settings-timezone">${escapeHtml(ctx.t("settings.timezone.label"))}</label>
                  <select id="settings-timezone" class="select"></select>
                </div>
              </div>
            </div>
            <div class="waf-list-item">
              <div class="waf-list-head">
                <div class="waf-list-title">${escapeHtml(ctx.t("settings.general"))}</div>
              </div>
              <label class="waf-checkbox">
                <input type="checkbox" id="settings-auto-logout">
                <span>${escapeHtml(ctx.t("settings.autoLogout"))}</span>
              </label>
            </div>
            <div class="waf-actions profile-actions-row">
              <button type="submit" class="btn primary btn-sm">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
          </form>
        </div>
      </section>

      <div class="modal" id="twofa-setup-modal" hidden>
        <div class="modal-backdrop" data-close="#twofa-setup-modal"></div>
        <div class="modal-body">
          <div class="modal-header">
            <h3>${escapeHtml(ctx.t("auth.2fa.setup.title"))}</h3>
            <button class="btn ghost" data-close="#twofa-setup-modal" aria-label="Close">x</button>
          </div>
          <div class="modal-content">
            <div class="alert" id="twofa-setup-alert" hidden></div>
            <div id="twofa-setup-step" class="form-grid two-column">
              <div class="form-field wide"><div class="muted">${escapeHtml(ctx.t("auth.2fa.setup.scanHint"))}</div></div>
              <div class="form-field"><img id="twofa-setup-qr" class="qr-image" alt="2FA QR"></div>
              <div class="form-field"><label>${escapeHtml(ctx.t("auth.2fa.setup.manualSecret"))}</label><input id="twofa-setup-secret" readonly></div>
              <div class="form-field wide required"><label>${escapeHtml(ctx.t("auth.2fa.code"))}</label><input id="twofa-setup-code" autocomplete="one-time-code" inputmode="numeric" required></div>
            </div>
            <div id="twofa-recovery-step" hidden>
              <div class="modal-section-header"><h4>${escapeHtml(ctx.t("auth.2fa.recovery.title"))}</h4></div>
              <div class="muted">${escapeHtml(ctx.t("auth.2fa.recovery.hint"))}</div>
              <ul id="twofa-recovery-codes"></ul>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="button" id="twofa-setup-confirm">${escapeHtml(ctx.t("auth.2fa.enable"))}</button>
              <button class="btn ghost" id="twofa-setup-close" data-close="#twofa-setup-modal" hidden>${escapeHtml(ctx.t("common.close"))}</button>
              <button class="btn ghost" type="button" id="twofa-setup-cancel" data-close="#twofa-setup-modal">${escapeHtml(ctx.t("common.cancel"))}</button>
            </div>
          </div>
        </div>
      </div>

      <div class="modal" id="twofa-disable-modal" hidden>
        <div class="modal-backdrop" data-close="#twofa-disable-modal"></div>
        <div class="modal-body">
          <div class="modal-header">
            <h3>${escapeHtml(ctx.t("auth.2fa.disable.title"))}</h3>
            <button class="btn ghost" data-close="#twofa-disable-modal" aria-label="Close">x</button>
          </div>
          <div class="modal-content">
            <div class="alert" id="twofa-disable-alert" hidden></div>
            <div class="form-grid two-column">
              <div class="form-field wide required"><label>${escapeHtml(ctx.t("accounts.currentPassword"))}</label><input id="twofa-disable-password" type="password" autocomplete="current-password" required></div>
              <div class="form-field wide required"><label>${escapeHtml(ctx.t("auth.2fa.recovery.code"))}</label><input id="twofa-disable-recovery" autocomplete="one-time-code" required></div>
            </div>
            <div class="form-actions">
              <button class="btn danger" type="button" id="twofa-disable-confirm">${escapeHtml(ctx.t("auth.2fa.disable"))}</button>
              <button class="btn ghost" type="button" data-close="#twofa-disable-modal">${escapeHtml(ctx.t("common.cancel"))}</button>
            </div>
          </div>
        </div>
      </div>

      <div class="modal" id="passkeys-register-modal" hidden>
        <div class="modal-backdrop" data-close="#passkeys-register-modal"></div>
        <div class="modal-body">
          <div class="modal-header">
            <h3>${escapeHtml(ctx.t("auth.passkeys.register.title"))}</h3>
            <button class="btn ghost" data-close="#passkeys-register-modal" aria-label="Close">x</button>
          </div>
          <div class="modal-content">
            <div class="alert" id="passkeys-register-alert" hidden></div>
            <div class="form-grid two-column">
              <div class="form-field wide required">
                <label for="passkeys-register-name">${escapeHtml(ctx.t("auth.passkeys.name"))}</label>
                <input id="passkeys-register-name" autocomplete="off" required>
                <div class="muted small">${escapeHtml(ctx.t("auth.passkeys.register.hint"))}</div>
              </div>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="button" id="passkeys-register-confirm">${escapeHtml(ctx.t("auth.passkeys.register.confirm"))}</button>
              <button class="btn ghost" type="button" data-close="#passkeys-register-modal">${escapeHtml(ctx.t("common.cancel"))}</button>
            </div>
          </div>
        </div>
      </div>

      <div class="modal" id="passkeys-rename-modal" hidden>
        <div class="modal-backdrop" data-close="#passkeys-rename-modal"></div>
        <div class="modal-body">
          <div class="modal-header">
            <h3>${escapeHtml(ctx.t("auth.passkeys.rename.title"))}</h3>
            <button class="btn ghost" data-close="#passkeys-rename-modal" aria-label="Close">x</button>
          </div>
          <div class="modal-content">
            <div class="alert" id="passkeys-rename-alert" hidden></div>
            <div class="form-grid two-column">
              <div class="form-field wide required">
                <label for="passkeys-rename-name">${escapeHtml(ctx.t("auth.passkeys.name"))}</label>
                <input id="passkeys-rename-name" autocomplete="off" required>
              </div>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="button" id="passkeys-rename-confirm">${escapeHtml(ctx.t("common.save"))}</button>
              <button class="btn ghost" type="button" data-close="#passkeys-rename-modal">${escapeHtml(ctx.t("common.cancel"))}</button>
            </div>
          </div>
        </div>
      </div>

      <div class="modal confirm-modal" id="passkeys-delete-modal" hidden>
        <div class="modal-backdrop" data-close="#passkeys-delete-modal"></div>
        <div class="modal-body">
          <div class="modal-header">
            <h3>${escapeHtml(ctx.t("auth.passkeys.delete.title"))}</h3>
            <button class="btn ghost" data-close="#passkeys-delete-modal" aria-label="Close">x</button>
          </div>
          <div class="modal-content">
            <div class="alert" id="passkeys-delete-alert" hidden></div>
            <div id="passkeys-delete-message" class="confirm-message"></div>
            <div class="form-actions">
              <button type="button" class="btn danger" id="passkeys-delete-confirm">${escapeHtml(ctx.t("common.delete"))}</button>
              <button type="button" class="btn ghost" data-close="#passkeys-delete-modal">${escapeHtml(ctx.t("common.cancel"))}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;

  bindModalClose(container);
  const alertEl = container.querySelector("#profile-alert");
  const prefs = loadPreferences();

  let me = null;
  try {
    me = await ctx.api.get("/api/auth/me");
  } catch (err) {
    showAlert(alertEl, err?.message || ctx.t("profile.error.load"));
  }

  setText("profile-username", me?.username || "-");
  setText("profile-fullname", me?.full_name || me?.username || "-");
  setText("profile-department", me?.department || "-");
  setText("profile-position", me?.position || "-");
  setText("profile-session-start", formatDateTime(me?.session_created_at || new Date().toISOString(), prefs));
  setText("profile-session-expire", formatDateTime(me?.session_expires_at || "-", prefs));
  setText("profile-last-login-ip", me?.last_login_ip || "-");
  setText("profile-trusted-ip", me?.frequent_login_ip || "-");
  setText("password-last-changed", me?.password_changed_at ? `${ctx.t("accounts.passwordLastChanged")}: ${formatDateTime(me.password_changed_at, prefs)}` : "");

  const tz = container.querySelector("#settings-timezone");
  const autoLogout = container.querySelector("#settings-auto-logout");
  autoLogout.checked = !!prefs.autoLogout;
  tz.innerHTML = availableTimeZones().map((zone) => `<option value="${escapeHtml(zone)}">${escapeHtml(zone)}</option>`).join("");
  tz.value = prefs.timeZone || "Europe/Moscow";

  container.querySelector("#settings-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const next = savePreferences({
      timeZone: tz.value || "Europe/Moscow",
      autoLogout: !!autoLogout.checked,
    });
    showAlert(alertEl, ctx.t("settings.saved"), true);
    ctx.notify(ctx.t("settings.saved"));
  });

  container.querySelector("#password-change-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    showAlert(alertEl, "");
    const data = Object.fromEntries(new FormData(event.currentTarget).entries());
    if (String(data.password || "") !== String(data.password_confirm || "")) {
      showAlert(alertEl, ctx.t("accounts.passwordMismatch"));
      return;
    }
    try {
      await ctx.api.post("/api/auth/change-password", {
        current_password: data.current_password,
        password: data.password,
      });
      showAlert(alertEl, ctx.t("accounts.passwordChangeDone"), true);
      ctx.notify(ctx.t("accounts.passwordChangeDone"));
      event.currentTarget.reset();
    } catch (err) {
      showAlert(alertEl, err?.message || ctx.t("common.error"));
    }
  });

  const render2FAStatus = async () => {
    try {
      const status = await refresh2FA(ctx);
      const enabled = !!status?.enabled;
      setText("twofa-status-text", enabled ? ctx.t("auth.2fa.status.enabled") : ctx.t("auth.2fa.status.disabled"));
      const remEl = container.querySelector("#twofa-recovery-remaining");
      if (enabled) {
        remEl.hidden = false;
        remEl.textContent = `${ctx.t("auth.2fa.recovery.remaining")}: ${Number(status?.recovery_codes_remaining || 0)}`;
      } else {
        remEl.hidden = true;
        remEl.textContent = "";
      }
      container.querySelector("#twofa-enable-btn").hidden = enabled;
      container.querySelector("#twofa-disable-btn").hidden = !enabled;
    } catch {
      setText("twofa-status-text", ctx.t("profile.error.load2fa"));
    }
  };

  container.querySelector("#twofa-enable-btn")?.addEventListener("click", async () => {
    showAlert(container.querySelector("#twofa-setup-alert"), "");
    try {
      const setup = await ctx.api.post("/api/auth/2fa/setup", {});
      container.querySelector("#twofa-setup-qr").src = setup?.qr_png_base64 || "";
      container.querySelector("#twofa-setup-secret").value = setup?.manual_secret || setup?.secret || "";
      container.querySelector("#twofa-setup-code").value = "";
      container.querySelector("#twofa-setup-confirm").dataset.challengeId = String(setup?.challenge_id || "");
      container.querySelector("#twofa-setup-step").hidden = false;
      container.querySelector("#twofa-recovery-step").hidden = true;
      container.querySelector("#twofa-setup-close").hidden = true;
      container.querySelector("#twofa-setup-confirm").hidden = false;
      container.querySelector("#twofa-setup-cancel").hidden = false;
      modalOpen("twofa-setup-modal");
    } catch (err) {
      showAlert(alertEl, err?.message || ctx.t("profile.error.setup2fa"));
    }
  });

  container.querySelector("#twofa-setup-confirm")?.addEventListener("click", async () => {
    const setupAlert = container.querySelector("#twofa-setup-alert");
    showAlert(setupAlert, "");
    const code = String(container.querySelector("#twofa-setup-code").value || "").trim();
    const challengeID = String(container.querySelector("#twofa-setup-confirm").dataset.challengeId || "").trim();
    if (!code || !challengeID) {
      showAlert(setupAlert, ctx.t("auth.2fa.codeRequired"));
      return;
    }
    try {
      const resp = await ctx.api.post("/api/auth/2fa/enable", { challenge_id: challengeID, code });
      const codes = Array.isArray(resp?.recovery_codes) ? resp.recovery_codes : [];
      const list = container.querySelector("#twofa-recovery-codes");
      list.innerHTML = codes.map((item) => `<li>${escapeHtml(String(item || "").trim())}</li>`).join("");
      container.querySelector("#twofa-setup-step").hidden = true;
      container.querySelector("#twofa-recovery-step").hidden = false;
      container.querySelector("#twofa-setup-confirm").hidden = true;
      container.querySelector("#twofa-setup-close").hidden = false;
      container.querySelector("#twofa-setup-cancel").hidden = true;
      await render2FAStatus();
    } catch (err) {
      showAlert(setupAlert, err?.message || ctx.t("profile.error.enable2fa"));
    }
  });

  container.querySelector("#twofa-disable-btn")?.addEventListener("click", () => {
    showAlert(container.querySelector("#twofa-disable-alert"), "");
    container.querySelector("#twofa-disable-password").value = "";
    container.querySelector("#twofa-disable-recovery").value = "";
    modalOpen("twofa-disable-modal");
  });

  container.querySelector("#twofa-disable-confirm")?.addEventListener("click", async () => {
    const disableAlert = container.querySelector("#twofa-disable-alert");
    showAlert(disableAlert, "");
    const password = String(container.querySelector("#twofa-disable-password").value || "");
    const recoveryCode = String(container.querySelector("#twofa-disable-recovery").value || "").trim();
    if (!password || !recoveryCode) {
      showAlert(disableAlert, ctx.t("auth.2fa.disableRequiresRecovery"));
      return;
    }
    try {
      await ctx.api.post("/api/auth/2fa/disable", { password, recovery_code: recoveryCode });
      modalClose("twofa-disable-modal");
      await render2FAStatus();
    } catch (err) {
      showAlert(disableAlert, err?.message || ctx.t("profile.error.disable2fa"));
    }
  });

  let renameID = "";
  let deleteID = "";

  const renderPasskeys = async () => {
    const status = container.querySelector("#passkeys-status-text");
    const wrap = container.querySelector("#passkeys-table-wrap");
    const body = container.querySelector("#passkeys-table-body");
    status.textContent = ctx.t("auth.passkeys.status.loading");
    status.hidden = false;
    wrap.hidden = true;
    try {
      const result = await refreshPasskeys(ctx);
      const items = Array.isArray(result?.items) ? result.items : [];
      if (!items.length) {
        status.textContent = ctx.t("auth.passkeys.status.empty");
        body.innerHTML = "";
        wrap.hidden = true;
        return;
      }
      status.hidden = true;
      wrap.hidden = false;
      body.innerHTML = items.map((it) => `
        <tr>
          <td>${escapeHtml(String(it?.name || ctx.t("auth.passkeys.unnamed")))}</td>
          <td class="muted">${escapeHtml(formatDateTime(it?.created_at, prefs))}</td>
          <td class="muted">${escapeHtml(it?.last_used_at ? formatDateTime(it?.last_used_at, prefs) : ctx.t("auth.passkeys.neverUsed"))}</td>
          <td class="table-actions">
            <button class="btn ghost btn-sm" data-action="rename" data-id="${escapeHtml(String(it?.id || ""))}" data-name="${escapeHtml(String(it?.name || ""))}">${escapeHtml(ctx.t("common.rename"))}</button>
            <button class="btn ghost danger btn-sm" data-action="delete" data-id="${escapeHtml(String(it?.id || ""))}" data-name="${escapeHtml(String(it?.name || ""))}">${escapeHtml(ctx.t("common.delete"))}</button>
          </td>
        </tr>
      `).join("");
    } catch (err) {
      status.textContent = err?.message || ctx.t("common.error");
      wrap.hidden = true;
    }
  };

  container.querySelector("#passkeys-table-body")?.addEventListener("click", (event) => {
    const btn = event.target.closest("button[data-action]");
    if (!btn) return;
    const id = String(btn.getAttribute("data-id") || "");
    const name = String(btn.getAttribute("data-name") || "");
    if (btn.getAttribute("data-action") === "rename") {
      renameID = id;
      container.querySelector("#passkeys-rename-name").value = name;
      showAlert(container.querySelector("#passkeys-rename-alert"), "");
      modalOpen("passkeys-rename-modal");
      return;
    }
    deleteID = id;
    container.querySelector("#passkeys-delete-message").textContent = ctx.t("auth.passkeys.delete.confirm", { name });
    showAlert(container.querySelector("#passkeys-delete-alert"), "");
    modalOpen("passkeys-delete-modal");
  });

  container.querySelector("#passkeys-add-btn")?.addEventListener("click", () => {
    if (!webAuthnSupported()) {
      showAlert(alertEl, ctx.t("auth.passkeys.unsupported"));
      return;
    }
    showAlert(container.querySelector("#passkeys-register-alert"), "");
    container.querySelector("#passkeys-register-name").value = window.BerkutWebAuthn?.defaultDeviceName?.() || "";
    modalOpen("passkeys-register-modal");
  });

  container.querySelector("#passkeys-register-confirm")?.addEventListener("click", async () => {
    const registerAlert = container.querySelector("#passkeys-register-alert");
    showAlert(registerAlert, "");
    const name = String(container.querySelector("#passkeys-register-name").value || "").trim();
    if (!name) {
      showAlert(registerAlert, ctx.t("auth.passkeys.nameRequired"));
      return;
    }
    try {
      const begin = await ctx.api.post("/api/auth/passkeys/register/begin", { name });
      const challengeID = String(begin?.challenge_id || "").trim();
      const options = begin?.options;
      if (!challengeID || !options) {
        throw new Error(ctx.t("auth.passkeys.failed"));
      }
      const publicKey = window.BerkutWebAuthn.toPublicKeyCreationOptions(options);
      const credential = await navigator.credentials.create({ publicKey });
      await ctx.api.post("/api/auth/passkeys/register/finish", {
        challenge_id: challengeID,
        name,
        credential: window.BerkutWebAuthn.credentialToJSON(credential),
      });
      modalClose("passkeys-register-modal");
      await renderPasskeys();
    } catch (err) {
      const key = window.BerkutWebAuthn?.errorKey?.(err) || "";
      showAlert(registerAlert, key ? ctx.t(key) : (err?.message || ctx.t("auth.passkeys.failed")));
    }
  });

  container.querySelector("#passkeys-rename-confirm")?.addEventListener("click", async () => {
    const renameAlert = container.querySelector("#passkeys-rename-alert");
    showAlert(renameAlert, "");
    const name = String(container.querySelector("#passkeys-rename-name").value || "").trim();
    if (!renameID || !name) {
      showAlert(renameAlert, ctx.t("common.badRequest"));
      return;
    }
    try {
      await ctx.api.put(`/api/auth/passkeys/${encodeURIComponent(renameID)}/rename`, { name });
      modalClose("passkeys-rename-modal");
      await renderPasskeys();
    } catch (err) {
      showAlert(renameAlert, err?.message || ctx.t("common.error"));
    }
  });

  container.querySelector("#passkeys-delete-confirm")?.addEventListener("click", async () => {
    const deleteAlert = container.querySelector("#passkeys-delete-alert");
    showAlert(deleteAlert, "");
    if (!deleteID) {
      showAlert(deleteAlert, ctx.t("common.badRequest"));
      return;
    }
    try {
      await ctx.api.delete(`/api/auth/passkeys/${encodeURIComponent(deleteID)}`);
      modalClose("passkeys-delete-modal");
      await renderPasskeys();
    } catch (err) {
      showAlert(deleteAlert, err?.message || ctx.t("common.error"));
    }
  });

  container.querySelector("#passkeys-unsupported").hidden = webAuthnSupported();
  container.querySelector("#passkeys-add-btn").disabled = !webAuthnSupported();

  await render2FAStatus();
  await renderPasskeys();
  await applyTranslations();
}
