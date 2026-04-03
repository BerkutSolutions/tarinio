import { escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";

export async function renderJobs(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("jobs.title"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("jobs.subtitle"))}</div>
        </div>
      </div>
      <div class="waf-card-body waf-stack">
        <div class="waf-note">${escapeHtml(ctx.t("jobs.note"))}</div>
        <div id="jobs-status"></div>
      </div>
    </section>
  `;

  const status = container.querySelector("#jobs-status");
  setLoading(status, ctx.t("jobs.loading"));
  try {
    const summary = await ctx.api.get("/api/reports/revisions");
    const apply = summary?.apply || {};
    const latest = Array.isArray(summary?.latest_revisions) ? summary.latest_revisions.slice(0, 6) : [];
    status.innerHTML = `
      <div class="waf-grid two">
        <div class="waf-subcard waf-stack waf-antiddos-frame">
          <div class="waf-list-title">${escapeHtml(ctx.t("jobs.card.summary"))}</div>
          <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("revisions.summary.totalJobs"))}:</span><strong>${escapeHtml(String(apply.total_apply_jobs ?? 0))}</strong></div>
          <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("revisions.summary.succeededJobs"))}:</span><strong>${escapeHtml(String(apply.succeeded_apply_jobs ?? 0))}</strong></div>
          <div class="waf-inline"><span class="waf-note">${escapeHtml(ctx.t("revisions.summary.failedJobs"))}:</span><strong>${escapeHtml(String(apply.failed_apply_jobs ?? 0))}</strong></div>
        </div>
        <div class="waf-subcard waf-stack waf-antiddos-frame">
          <div class="waf-list-title">${escapeHtml(ctx.t("jobs.card.backlog"))}</div>
          <div class="waf-note">${escapeHtml(ctx.t("jobs.shell.backlog"))}</div>
        </div>
      </div>
      <div class="waf-subcard waf-stack waf-antiddos-frame">
        <div class="waf-list-title">${escapeHtml(ctx.t("jobs.card.latest"))}</div>
        ${latest.length ? `
          <div class="waf-table-wrap">
            <table class="waf-table">
              <thead>
                <tr>
                  <th>${escapeHtml(ctx.t("revisions.field.revision"))}</th>
                  <th>${escapeHtml(ctx.t("revisions.field.created"))}</th>
                  <th>${escapeHtml(ctx.t("revisions.field.result"))}</th>
                </tr>
              </thead>
              <tbody>
                ${latest.map((item) => `
                  <tr>
                    <td class="waf-code">${escapeHtml(String(item?.revision_id || "-"))}</td>
                    <td>${escapeHtml(formatDate(String(item?.created_at || "")))}</td>
                    <td>${statusBadge(String(item?.last_apply_result || "unknown"))}</td>
                  </tr>
                `).join("")}
              </tbody>
            </table>
          </div>
        ` : `<div class="waf-empty">${escapeHtml(ctx.t("jobs.empty"))}</div>`}
      </div>
    `;
  } catch (error) {
    setError(status, ctx.t("jobs.error.load"));
  }
}
