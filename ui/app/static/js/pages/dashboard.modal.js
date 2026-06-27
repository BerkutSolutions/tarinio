import { escapeHtml } from "../ui.js";

function createModalState(container, ctx) {
  const modalNode = container.querySelector("#dashboard-detail-modal");
  const modalTitleNode = container.querySelector("#dashboard-detail-title");
  const modalSubtitleNode = container.querySelector("#dashboard-detail-subtitle");
  const modalBodyNode = container.querySelector("#dashboard-detail-content");

  const close = () => {
    modalNode?.classList.add("waf-hidden");
  };

  modalNode?.querySelectorAll("[data-dashboard-detail-close='true']").forEach((node) => {
    node.addEventListener("click", close);
  });
  modalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      close();
    }
  });

  return {
    open(detail) {
      if (!modalNode || !modalTitleNode || !modalSubtitleNode || !modalBodyNode) {
        return;
      }
      modalTitleNode.innerHTML = escapeHtml(String(detail?.title || ctx.t("dashboard.detail.title")));
      if (detail?.titleHtml) {
        modalTitleNode.innerHTML = detail.titleHtml;
      }
      modalSubtitleNode.textContent = String(detail?.subtitle || "");
      modalBodyNode.innerHTML = detail?.body || `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
      modalNode.classList.remove("waf-hidden");
      modalNode.focus();
    },
    close
  };
}

export {
  createModalState
};
