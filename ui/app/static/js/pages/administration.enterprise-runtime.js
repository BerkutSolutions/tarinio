import { notify } from "../ui.js";
import { downloadBlob } from "./administration.helpers-base.js";
import { parseCSV, parseRoleMappings } from "./administration.helpers-enterprise.js";

export function bindAdministrationEnterpriseRuntime(deps) {
  const { enterpriseStatus, ctx, loadEnterprise } = deps;

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
          require_verified_email: true
        },
        approvals: {
          enabled: Boolean(enterpriseStatus.querySelector("#enterprise-approvals-enabled")?.checked),
          required_approvals: Number.parseInt(String(enterpriseStatus.querySelector("#enterprise-approvals-count")?.value || "1"), 10) || 1,
          allow_self_approval: Boolean(enterpriseStatus.querySelector("#enterprise-approvals-self")?.checked),
          reviewer_role_ids: parseCSV(enterpriseStatus.querySelector("#enterprise-approvals-reviewers")?.value || "")
        },
        scim: {
          enabled: Boolean(enterpriseStatus.querySelector("#enterprise-scim-enabled")?.checked),
          default_role_ids: parseCSV(enterpriseStatus.querySelector("#enterprise-scim-default-roles")?.value || ""),
          group_role_mappings: parseRoleMappings(enterpriseStatus.querySelector("#enterprise-scim-mappings")?.value || "")
        }
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
