import { escapeHtml } from "../ui.js";

function renderSummaryMetrics(items, ctx, deps) {
  if (!Array.isArray(items) || !items.length) {
    return "";
  }
  const displayValue = (value) => {
    if (typeof value === "string") {
      return value;
    }
    return deps.formatNumber(value);
  };
  return `
    <div class="dashboard-mini-metrics">
      ${items.map((item) => `
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(displayValue(item.value))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t(item.labelKey))}</div>
        </div>
      `).join("")}
    </div>
  `;
}

function renderDetailTable(items, ctx, labelTitle, countTitle, options = {}, deps) {
  const list = Array.isArray(items) ? items : [];
  if (!list.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
  }
  const labelFormatter = typeof options.labelFormatter === "function"
    ? options.labelFormatter
    : (item) => escapeHtml(String(item?.key || "-"));
  const countFormatter = typeof options.countFormatter === "function"
    ? options.countFormatter
    : (item) => escapeHtml(deps.formatNumber(item?.count || 0));
  const attrs = typeof options.rowAttrs === "function" ? options.rowAttrs : () => "";
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr><th>${escapeHtml(labelTitle)}</th><th>${escapeHtml(countTitle)}</th></tr></thead>
        <tbody>
          ${list.map((item, index) => {
            const rowAttrs = attrs(item, index);
            const extraClass = rowAttrs ? "waf-table-row-clickable" : "";
            const roleAttrs = rowAttrs ? ' tabindex="0" role="button"' : "";
            return `<tr class="${extraClass}" ${rowAttrs}${roleAttrs}><td>${labelFormatter(item)}</td><td>${countFormatter(item)}</td></tr>`;
          }).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderDetailSection(title, content) {
  return `
    <section class="dashboard-detail-section">
      <h4 class="dashboard-detail-section-title">${escapeHtml(title)}</h4>
      ${content}
    </section>
  `;
}

export {
  renderSummaryMetrics,
  renderDetailTable,
  renderDetailSection
};
