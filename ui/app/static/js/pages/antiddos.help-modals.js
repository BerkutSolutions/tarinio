export function renderProtectionHelpModals(ctx, escapeHtml) {
  const modal = (id, titleKey, bodyKey) => `
    <div class="waf-modal waf-hidden" id="${id}" role="dialog" aria-modal="true" aria-labelledby="${id}-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-antiddos-protection-help-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <h3 id="${id}-title">${escapeHtml(ctx.t(titleKey))}</h3>
          <button class="btn ghost btn-sm" type="button" data-antiddos-protection-help-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body"><p class="waf-note">${escapeHtml(ctx.t(bodyKey))}</p></div>
      </div>
    </div>`;
  return modal("antiddos-l4-help-modal", "antiddos.help.l4.title", "antiddos.help.l4.body") +
    modal("antiddos-l7-help-modal", "antiddos.help.l7.title", "antiddos.help.l7.body");
}

export function bindProtectionHelpModals(container) {
  const bind = (buttonID, modalID) => {
    const modal = container.querySelector(`#${modalID}`);
    const close = () => modal?.classList.add("waf-hidden");
    container.querySelector(`#${buttonID}`)?.addEventListener("click", () => {
      modal?.classList.remove("waf-hidden");
      modal?.focus();
    });
    modal?.querySelectorAll("[data-antiddos-protection-help-close]").forEach((node) => node.addEventListener("click", close));
    modal?.addEventListener("keydown", (event) => {
      if (event.key === "Escape") close();
    });
  };
  bind("antiddos-l4-help-btn", "antiddos-l4-help-modal");
  bind("antiddos-l7-help-btn", "antiddos-l7-help-modal");
}
