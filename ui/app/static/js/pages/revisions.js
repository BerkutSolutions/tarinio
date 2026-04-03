import { escapeHtml, setError, setLoading, statusBadge } from "../ui.js";

export async function renderRevisions(container, ctx) {
  container.innerHTML = `
    <div class="waf-page-stack">
      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("revisions.summary.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("revisions.summary.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" id="revision-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
        <div class="waf-card-body">
          <div class="waf-note">${escapeHtml(ctx.t("revisions.autoApply.note"))}</div>
          <div id="revision-summary-list" class="waf-list"></div>
        </div>
      </section>
    </div>
  `;

  const summaryList = container.querySelector("#revision-summary-list");

  const renderSummary = async () => {
    setLoading(summaryList, ctx.t("revisions.summary.loading"));
    try {
      const summary = await ctx.api.get("/api/reports/revisions");
      const apply = summary?.revision_apply || {};
      const activeRevisionID = apply.active_revision_id || "-";
      summaryList.innerHTML = `
        <div class="waf-list-item">
          <div class="waf-list-head">
            <div class="waf-list-title">${escapeHtml(ctx.t("revisions.summary.title"))}</div>
            ${statusBadge("info")}
          </div>
          <div>${escapeHtml(ctx.t("revisions.field.revision"))}: ${escapeHtml(activeRevisionID)}</div>
          <div>${escapeHtml(ctx.t("revisions.field.version"))}: ${escapeHtml(String(apply.total_revisions ?? 0))}</div>
          <div>${escapeHtml(ctx.t("revisions.summary.pending"))}: ${escapeHtml(String(apply.pending_revisions ?? 0))}</div>
          <div>${escapeHtml(ctx.t("revisions.summary.failed"))}: ${escapeHtml(String(apply.failed_revisions ?? 0))}</div>
          <div>${escapeHtml(ctx.t("revisions.summary.totalJobs"))}: ${escapeHtml(String(apply.total_apply_jobs ?? 0))}</div>
          <div>${escapeHtml(ctx.t("revisions.summary.succeededJobs"))}: ${escapeHtml(String(apply.succeeded_apply_jobs ?? 0))}</div>
          <div>${escapeHtml(ctx.t("revisions.summary.failedJobs"))}: ${escapeHtml(String(apply.failed_apply_jobs ?? 0))}</div>
        </div>
      `;
    } catch (error) {
      setError(summaryList, error.message || ctx.t("revisions.summary.error"));
    }
  };

  container.querySelector("#revision-refresh")?.addEventListener("click", renderSummary);
  await renderSummary();
}
