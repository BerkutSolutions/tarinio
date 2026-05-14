import { openTableRow, collectValues } from "./administration.helpers-modals.js";

export function bindAdministrationUsersRolesRuntime(deps) {
  const {
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
  } = deps;

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
        role_ids: collectValues(userForm, "role_ids")
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
      permissions: collectValues(roleForm, "permissions")
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
}
