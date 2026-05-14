import { escapeHtml, setError, setLoading } from "../ui.js";
import {
  bindPasswordToggles,
  renderRolesTable,
  renderUsersTable
} from "./administration.helpers-base.js";
import {
  renderEnterprisePanel
} from "./administration.helpers-enterprise.js";
import {
  renderRoleModal,
  renderUserModal
} from "./administration.helpers-modals.js";
import { setupAdministrationScriptsRuntime } from "./administration.scripts-runtime.js";
import { bindAdministrationUsersRolesRuntime } from "./administration.users-roles-runtime.js";
import { bindAdministrationEnterpriseRuntime } from "./administration.enterprise-runtime.js";



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

  const { renderScripts } = setupAdministrationScriptsRuntime({ ctx, scriptsStatus, latestRuns });

  container.querySelector("#administration-user-create")?.addEventListener("click", () => openUserModal({}, "create"));
  container.querySelector("#administration-role-create")?.addEventListener("click", () => openRoleModal({}, "create"));


  modal.addEventListener("click", (event) => {
    if (event.target.closest("[data-close-modal]")) {
      closeModal();
    }
  });

  bindAdministrationUsersRolesRuntime({
    usersStatus,
    rolesStatus,
    modal,
    state,
    ctx,
    openUserModal,
    openRoleModal,
    closeModal,
    showModalAlert,
    loadUsers,
    loadRoles
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

  bindAdministrationEnterpriseRuntime({ enterpriseStatus, ctx, loadEnterprise });
}
