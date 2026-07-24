import { escapeHtml } from "../ui.js";

function errorMessage(response, body) {
  try {
    const payload = JSON.parse(body || "{}");
    return String(payload?.error || payload?.message || `HTTP ${response.status}`);
  } catch {
    return String(body || `HTTP ${response.status}`);
  }
}

export function createTLSExportStepUp(container, ctx, downloadBlob) {
  let approvalID = "";
  let certificateIDs = [];
  const modal = document.createElement("div");
  modal.className = "waf-modal waf-hidden";
  modal.id = "tls-export-step-up-modal";
  modal.tabIndex = -1;
  modal.innerHTML = `
    <button class="waf-modal-overlay" type="button" data-tls-step-up-close aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
    <div class="waf-modal-card">
      <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.export.stepUp.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.export.stepUp.subtitle"))}</div></div></div>
      <div class="waf-card-body waf-stack">
        <label class="waf-field" for="tls-export-totp"><span>${escapeHtml(ctx.t("tls.export.stepUp.code"))}</span><input id="tls-export-totp" inputmode="numeric" autocomplete="one-time-code" maxlength="6"></label>
        <div id="tls-export-approval-id" class="waf-note"></div><div id="tls-export-step-up-status"></div>
        <div class="waf-actions"><button class="btn" id="tls-export-approval-retry" type="button">${escapeHtml(ctx.t("tls.export.approvalRetry"))}</button><button class="btn primary" id="tls-export-step-up-submit" type="button">${escapeHtml(ctx.t("tls.export.stepUp.submit"))}</button><button class="btn ghost" data-tls-step-up-close type="button">${escapeHtml(ctx.t("common.cancel"))}</button></div>
      </div>
    </div>`;
  container.appendChild(modal);
  const close = () => modal.classList.add("waf-hidden");
  modal.querySelectorAll("[data-tls-step-up-close]").forEach((node) => node.addEventListener("click", close));
  modal.addEventListener("keydown", (event) => { if (event.key === "Escape") close(); });

  async function requestApproval(ids) {
    const response = await fetch("/api/certificate-materials/export-approvals", { method: "POST", credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify({ certificate_ids: ids }) });
    const body = await response.text();
    if (!response.ok) throw new Error(errorMessage(response, body));
    return JSON.parse(body);
  }

  async function exportArchive(code = "") {
    if (code) {
      const step = await fetch("/api/auth/step-up/totp", { method: "POST", credentials: "include", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify({ code }) });
      const body = await step.text();
      if (!step.ok) throw new Error(errorMessage(step, body));
    }
    const response = await fetch("/api/certificate-materials/export", { method: "POST", credentials: "include", headers: { Accept: "application/zip, application/json", "Content-Type": "application/json" }, body: JSON.stringify({ certificate_ids: certificateIDs, approval_id: approvalID }) });
    if (!response.ok) throw new Error(errorMessage(response, await response.text()));
    downloadBlob(certificateIDs.length === 1 ? `${certificateIDs[0]}-materials.zip` : "certificate-materials.zip", await response.blob());
    ctx.notify(ctx.t("tls.certificates.exportArchive"));
    close();
  }
  async function continueExport() {
    try {
      await exportArchive();
    } catch (error) {
      const message = String(error?.message || error);
      if (/step-up|totp/i.test(message)) {
        modal.classList.remove("waf-hidden"); modal.focus(); modal.querySelector("#tls-export-totp")?.focus(); return;
      }
      modal.classList.remove("waf-hidden");
      const status = modal.querySelector("#tls-export-step-up-status");
      if (status) status.innerHTML = `<div class="alert">${escapeHtml(message)}</div>`;
    }
  }
  modal.querySelector("#tls-export-approval-retry")?.addEventListener("click", continueExport);

  modal.querySelector("#tls-export-step-up-submit")?.addEventListener("click", async () => {
    const status = modal.querySelector("#tls-export-step-up-status");
    try {
      await exportArchive(String(modal.querySelector("#tls-export-totp")?.value || "").trim());
      if (status) status.innerHTML = "";
    } catch (error) {
      if (status) status.innerHTML = `<div class="alert">${escapeHtml(String(error?.message || error))}</div>`;
    }
  });

  return async (ids) => {
    certificateIDs = ids;
    const approval = await requestApproval(ids);
    approvalID = String(approval?.id || "");
    ctx.notify(ctx.t("tls.export.approvalRequested"));
    const approvalNode = modal.querySelector("#tls-export-approval-id");
    if (approvalNode) approvalNode.textContent = ctx.t("tls.export.approvalId").replace("{id}", approvalID);
    await continueExport();
  };
}
