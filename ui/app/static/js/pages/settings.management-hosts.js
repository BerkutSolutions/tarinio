import { escapeHtml } from "../ui.js";

export function renderManagementHostsPanel(ctx) {
  return `
    <div class="settings-panel" data-settings-panel="management-hosts" hidden>
      <section class="waf-card">
        <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("settings.managementHosts.title"))}</h3><div class="muted">${escapeHtml(ctx.t("settings.managementHosts.subtitle"))}</div></div></div>
        <div class="waf-card-body waf-stack">
          <div class="waf-note">${escapeHtml(ctx.t("settings.managementHosts.warning"))}</div>
          <div class="waf-field"><label for="settings-management-hosts">${escapeHtml(ctx.t("settings.managementHosts.label"))}</label><textarea id="settings-management-hosts" rows="5" spellcheck="false"></textarea></div>
          <div class="waf-note" id="settings-management-hosts-status"></div>
          <div class="waf-actions"><button id="settings-management-hosts-save" class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("common.save"))}</button></div>
        </div>
      </section>
    </div>`;
}

export async function bindManagementHostsPanel(container, ctx, setAlert) {
  const field = container.querySelector("#settings-management-hosts");
  const status = container.querySelector("#settings-management-hosts-status");
  const save = container.querySelector("#settings-management-hosts-save");
  if (!field || !status || !save) return;
  let version = 0;
  const load = async () => {
    try {
      const settings = await ctx.api.get("/api/settings/management-hosts");
      version = Number(settings?.version || 0);
      field.value = Array.isArray(settings?.management_hosts) ? settings.management_hosts.join("\n") : "";
      status.textContent = settings?.management_hosts_setup_required ? ctx.t("settings.managementHosts.setupRequired") : ctx.t("settings.managementHosts.ready");
	  if (!settings?.management_hosts_setup_required) {
		const runtime = await ctx.api.get("/api/settings/management-hosts/status").catch(() => null);
		if (runtime?.drift) status.textContent = ctx.t("settings.managementHosts.drift");
	  }
    } catch (error) { status.textContent = error?.message || ctx.t("common.error"); }
  };
  save.addEventListener("click", async () => {
    const management_hosts = field.value.split(/\r?\n|,/).map((value) => value.trim()).filter(Boolean);
    try {
      const settings = await ctx.api.put("/api/settings/management-hosts", { management_hosts, version });
      version = Number(settings?.version || 0);
      field.value = settings.management_hosts.join("\n");
      status.textContent = ctx.t("settings.managementHosts.saved");
      setAlert(ctx.t("settings.managementHosts.applyRequired"), true);
    } catch (error) { setAlert(error?.message || ctx.t("common.error")); }
  });
  await load();
}
