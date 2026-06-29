import { escapeHtml } from "../ui.js";

// Full list of WAF error page entries: slug → label + default enabled.
const ERROR_PAGE_ENTRIES = [
  { slug: "400", label: "400 Bad Request" },
  { slug: "401", label: "401 Unauthorized" },
  { slug: "402", label: "402 Payment Required" },
  { slug: "403", label: "403 Forbidden" },
  { slug: "404", label: "404 Not Found" },
  { slug: "405", label: "405 Method Not Allowed" },
  { slug: "406", label: "406 Not Acceptable" },
  { slug: "407", label: "407 Proxy Authentication Required" },
  { slug: "408", label: "408 Request Timeout" },
  { slug: "409", label: "409 Conflict" },
  { slug: "410", label: "410 Gone" },
  { slug: "411", label: "411 Length Required" },
  { slug: "412", label: "412 Precondition Failed" },
  { slug: "413", label: "413 Payload Too Large" },
  { slug: "414", label: "414 URI Too Long" },
  { slug: "415", label: "415 Unsupported Media Type" },
  { slug: "416", label: "416 Range Not Satisfiable" },
  { slug: "417", label: "417 Expectation Failed" },
  { slug: "418", label: "418 I'm a Teapot" },
  { slug: "421", label: "421 Misdirected Request" },
  { slug: "422", label: "422 Unprocessable Entity" },
  { slug: "423", label: "423 Locked" },
  { slug: "424", label: "424 Failed Dependency" },
  { slug: "425", label: "425 Too Early" },
  { slug: "426", label: "426 Upgrade Required" },
  { slug: "428", label: "428 Precondition Required" },
  { slug: "429", label: "429 Too Many Requests" },
  { slug: "431", label: "431 Request Header Fields Too Large" },
  { slug: "444", label: "444 No Response" },
  { slug: "451", label: "451 Geo Block" },
  { slug: "500", label: "500 Internal Server Error" },
  { slug: "501", label: "501 Not Implemented" },
  { slug: "502", label: "502 Bad Gateway" },
  { slug: "503", label: "503 Service Unavailable" },
  { slug: "504", label: "504 Gateway Timeout" },
  { slug: "505", label: "505 HTTP Version Not Supported" },
  { slug: "507", label: "507 Insufficient Storage" },
  { slug: "508", label: "508 Loop Detected" },
  { slug: "510", label: "510 Not Extended" },
  { slug: "511", label: "511 Network Authentication Required" },
];

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
        <span class="waf-error-pages-label${!pageEnabled || !enabled ? " waf-eprow-off" : ""}">${escapeHtml(e.label)}</span>
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
  }).join("");

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
              ${rows}
            </div>
          </div>
        </div>
      </section>
    </div>
  `;
}
