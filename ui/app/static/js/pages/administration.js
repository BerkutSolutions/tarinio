import { escapeHtml, setError, setLoading } from "../ui.js";

export async function renderAdministration(container, ctx) {
  container.innerHTML = `
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
  `;

  const usersStatus = container.querySelector("#users-status");
  const rolesStatus = container.querySelector("#roles-status");

  setLoading(usersStatus, ctx.t("administration.users.loading"));
  setLoading(rolesStatus, ctx.t("administration.roles.loading"));

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
  } catch (error) {
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
  } catch (error) {
    setError(rolesStatus, ctx.t("administration.roles.error.load"));
  }
}
