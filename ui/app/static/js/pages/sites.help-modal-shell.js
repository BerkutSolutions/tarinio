export function renderHelpModalRows(rows, ctx, escapeHtml) {
  return rows.map((row) => `
    <tr>
      <td>${escapeHtml(ctx.t(row.labelKey))}</td>
      <td>${escapeHtml(ctx.t(row.helpKey))}</td>
    </tr>
  `).join("");
}

export function renderHelpModalShell({ modalID, titleID, titleKey, subtitleKey, rows, ctx, escapeHtml }) {
  return `
    <div class="waf-modal waf-hidden" id="${modalID}" role="dialog" aria-modal="true" aria-labelledby="${titleID}" tabindex="-1">
      <div class="waf-modal-backdrop" data-help-close="${modalID}"></div>
      <div class="waf-modal-dialog waf-modal-lg">
        <div class="waf-card-head">
          <div>
            <h3 id="${titleID}">${escapeHtml(ctx.t(titleKey))}</h3>
            <div class="muted">${escapeHtml(ctx.t(subtitleKey))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-help-close="${modalID}">${escapeHtml(ctx.t("common.close"))}</button>
        </div>
        <div class="waf-table-wrap">
          <table class="waf-table">
            <thead><tr><th>${escapeHtml(ctx.t("sites.easy.auth.help.field"))}</th><th>${escapeHtml(ctx.t("sites.easy.auth.help.usage"))}</th></tr></thead>
            <tbody>${renderHelpModalRows(rows, ctx, escapeHtml)}</tbody>
          </table>
        </div>
      </div>
    </div>
  `;
}

export function buildHelpRows(prefix, keys) {
  return keys.map((key) => ({
    labelKey: `${prefix}.${key}.label`,
    helpKey: `${prefix}.${key}.usage`,
  }));
}
