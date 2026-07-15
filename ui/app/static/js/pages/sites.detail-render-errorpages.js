import { escapeHtml } from "../ui.js";

// Full list of WAF error page entries: slug → label + default enabled.
const ERROR_PAGE_ENTRIES = [
  ...["400", "401", "402", "403", "404", "405", "406", "407", "408", "409", "410", "411", "412", "413", "414", "415", "416", "417", "418", "421", "422", "423", "424", "425", "426", "428", "429", "431", "444"].map((slug) => ({ slug, labelKey: `sites.easy.errorpages.status.${slug}` })),
  { slug: "451", labelKey: "sites.easy.errorpages.extended.451" },
  { slug: "geo_block", labelKey: "sites.easy.errorpages.geoBlock" },
  ...["500", "501", "502", "503", "504", "505", "507", "508", "510", "511"].map((slug) => ({ slug, labelKey: `sites.easy.errorpages.status.${slug}` })),
  { slug: "494", labelKey: "sites.easy.errorpages.extended.494", vendor: true },
  { slug: "495", labelKey: "sites.easy.errorpages.extended.495", vendor: true },
  { slug: "496", labelKey: "sites.easy.errorpages.extended.496", vendor: true },
  { slug: "497", labelKey: "sites.easy.errorpages.extended.497", vendor: true },
  { slug: "499", labelKey: "sites.easy.errorpages.extended.499", vendor: true, diagnostic: true },
  { slug: "506", labelKey: "sites.easy.errorpages.extended.506" },
  { slug: "520", labelKey: "sites.easy.errorpages.extended.520", vendor: true },
  { slug: "521", labelKey: "sites.easy.errorpages.extended.521", vendor: true },
  { slug: "522", labelKey: "sites.easy.errorpages.extended.522", vendor: true },
  { slug: "523", labelKey: "sites.easy.errorpages.extended.523", vendor: true },
  { slug: "524", labelKey: "sites.easy.errorpages.extended.524", vendor: true },
  { slug: "525", labelKey: "sites.easy.errorpages.extended.525", vendor: true },
  { slug: "526", labelKey: "sites.easy.errorpages.extended.526", vendor: true },
];

function errorPageCategory(slug) {
  if (slug === "geo_block" || slug === "451") return "waf";
  const code = Number(slug);
  if ([444, 494, 495, 496, 497, 499].includes(code)) return "proxy";
  if (code >= 520 && code <= 526) return "upstream";
  return code >= 500 ? "server" : "client";
}

export function renderErrorPagesTab(draft, ctx) {
  const enabled = Boolean(draft.use_custom_error_pages ?? true);
  // disabled_error_pages: array of slugs that are individually disabled
  const disabledPages = Array.isArray(draft.disabled_error_pages) ? draft.disabled_error_pages : [];

  const rows = ERROR_PAGE_ENTRIES.map((e) => {
    const pageEnabled = !disabledPages.includes(e.slug);
    return `
      <div class="waf-error-pages-row${!enabled ? " waf-eprow-master-off" : ""}">
        <label class="waf-error-pages-check" title="${escapeHtml(ctx.t("sites.easy.errorpages.togglePage"))} ${escapeHtml(e.slug)}">
          <input
            type="checkbox"
            class="waf-ep-page-cb"
            data-ep-slug="${escapeHtml(e.slug)}"
            ${pageEnabled ? "checked" : ""}
            ${!enabled ? "disabled" : ""}
          >
        </label>
        <span class="waf-error-pages-label${!pageEnabled || !enabled ? " waf-eprow-off" : ""}">${escapeHtml(e.labelKey ? ctx.t(e.labelKey) : e.label)}${e.vendor ? ` <small class="muted">${escapeHtml(ctx.t("sites.easy.errorpages.vendor"))}</small>` : ""}${e.diagnostic ? ` <small class="muted">${escapeHtml(ctx.t("sites.easy.errorpages.diagnostic"))}</small>` : ""}</span>
        <button
          type="button"
          class="btn ghost btn-sm waf-error-pages-preview-btn"
          data-error-page-slug="${escapeHtml(e.slug)}"
          title="${escapeHtml(ctx.t("sites.easy.errorpages.preview"))} ${escapeHtml(e.slug)}"
          ${!enabled ? "disabled" : ""}>
          ${escapeHtml(ctx.t("sites.easy.errorpages.preview"))}
        </button>
      </div>
    `;
  }).reduce((groups, row, index) => {
    const category = errorPageCategory(ERROR_PAGE_ENTRIES[index].slug);
    (groups[category] ||= []).push(row);
    return groups;
  }, {});
  const groupedRows = [["client", "sites.easy.errorpages.category.client"], ["server", "sites.easy.errorpages.category.server"], ["proxy", "sites.easy.errorpages.category.proxy"], ["upstream", "sites.easy.errorpages.category.upstream"], ["waf", "sites.easy.errorpages.category.waf"]]
    .map(([category, key]) => rows[category]?.length ? `<h4 class="muted" style="margin:14px 0 6px;">${escapeHtml(ctx.t(key))}</h4>${rows[category].join("")}` : "").join("");

  return `
    <div class="waf-antibot-auth-grid">
      <section class="waf-subcard">
        <div class="waf-card-head">
          <h3>${escapeHtml(ctx.t("sites.easy.errorpages.frameTitle"))}</h3>
        </div>
        <div class="waf-card-body waf-stack" style="gap:16px;">
          <label class="waf-checkbox full">
            <input type="checkbox" id="service-use-custom-error-pages"${enabled ? " checked" : ""}>
            <span>${escapeHtml(ctx.t("sites.easy.errorpages.enable"))}</span>
          </label>
          <p class="muted" style="margin:0;">${escapeHtml(ctx.t("sites.easy.errorpages.hint"))}</p>
          <div class="waf-error-pages-list${!enabled ? " waf-disabled" : ""}" id="ep-list-wrap">
            <div class="waf-error-pages-header">
              <span class="waf-error-pages-header-label">${escapeHtml(ctx.t("sites.easy.errorpages.pageColumn"))}</span>
              <button type="button" class="btn ghost btn-sm" id="ep-enable-all" style="font-size:11px;padding:3px 10px;"${!enabled ? " disabled" : ""}>${escapeHtml(ctx.t("sites.easy.errorpages.enableAll"))}</button>
              <button type="button" class="btn ghost btn-sm" id="ep-disable-all" style="font-size:11px;padding:3px 10px;"${!enabled ? " disabled" : ""}>${escapeHtml(ctx.t("sites.easy.errorpages.disableAll"))}</button>
            </div>
            <div class="waf-error-pages-rows" id="ep-rows">
              ${groupedRows}
            </div>
          </div>
        </div>
      </section>
    </div>
  `;
}
