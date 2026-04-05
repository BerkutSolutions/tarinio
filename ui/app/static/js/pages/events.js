import { escapeHtml, setError, setLoading } from "../ui.js";

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function normalizeToken(value) {
  return String(value || "").trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, "");
}

function unwrapList(payload, keys = []) {
  if (Array.isArray(payload)) {
    return payload;
  }
  for (const key of keys) {
    if (Array.isArray(payload?.[key])) {
      return payload[key];
    }
  }
  return [];
}

async function tryGetJSON(path) {
  try {
    const response = await fetch(path, {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return null;
    }
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  } catch (error) {
    return null;
  }
}

function translateEventType(type, ctx) {
  const token = normalizeToken(type);
  const key = token ? `events.type.${token}` : "events.type.unknown";
  const value = ctx.t(key);
  return value === key ? (type || ctx.t("common.unknown")) : value;
}

function translateEventSeverity(severity, ctx) {
  const token = normalizeToken(severity);
  const key = token ? `events.severity.${token}` : "events.severity.unknown";
  const value = ctx.t(key);
  return value === key ? (severity || ctx.t("common.unknown")) : value;
}

function translateEventSummary(item, ctx) {
  const summaryToken = normalizeToken(item?.summary);
  if (summaryToken) {
    const summaryKey = `events.summary.${summaryToken}`;
    const summaryValue = ctx.t(summaryKey);
    if (summaryValue !== summaryKey) {
      return summaryValue;
    }
  }
  return String(item?.summary || "");
}

function normalizeSiteID(value) {
  return String(value || "").trim().toLowerCase().replace(/_/g, "-");
}

function shouldSkipInternalSite(siteID) {
  const token = normalizeSiteID(siteID);
  return token === "control-plane-access" || token === "control-plane" || token === "ui";
}

function buildPageButtons(totalPages, currentPage, dataAttr) {
  const pages = [];
  for (let page = 1; page <= Math.min(10, totalPages); page += 1) {
    pages.push(`<button type="button" class="btn ghost btn-sm${page === currentPage ? " active" : ""}" ${dataAttr}="${page}">${page}</button>`);
  }
  if (totalPages > 10) {
    pages.push(`<span class="muted">...</span>`);
    pages.push(`<button type="button" class="btn ghost btn-sm${totalPages === currentPage ? " active" : ""}" ${dataAttr}="${totalPages}">${totalPages}</button>`);
  }
  return pages.join("");
}

export async function renderEvents(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.events"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("events.subtitle"))}</div>
        </div>
      </div>
      <div class="waf-card-body waf-stack">
        <form id="events-filters" class="waf-filters">
          <div class="waf-form-grid three">
            <div class="waf-field"><label>${escapeHtml(ctx.t("events.filter.type"))}</label><input id="events-type" placeholder="apply_failed"></div>
            <div class="waf-field"><label>${escapeHtml(ctx.t("events.filter.severity"))}</label><input id="events-severity" placeholder="error"></div>
            <div class="waf-field"><label>${escapeHtml(ctx.t("events.filter.site"))}</label><input id="events-site" placeholder="${escapeHtml(ctx.t("events.filter.sitePlaceholder"))}"></div>
            <div class="waf-field"><label>${escapeHtml(ctx.t("events.filter.from"))}</label><input id="events-from" type="datetime-local"></div>
            <div class="waf-field"><label>${escapeHtml(ctx.t("events.filter.to"))}</label><input id="events-to" type="datetime-local"></div>
          </div>
        </form>
        <div id="events-status"></div>
        <div id="events-list"></div>
      </div>
    </section>
    <div class="waf-modal waf-hidden" id="events-detail-modal" role="dialog" aria-modal="true" aria-labelledby="events-detail-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-events-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="events-detail-title">${escapeHtml(ctx.t("events.detail.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("events.detail.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-events-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body" id="events-detail-content">
          <div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>
        </div>
      </div>
    </div>
  `;

  const status = container.querySelector("#events-status");
  const list = container.querySelector("#events-list");
  const detailModalNode = container.querySelector("#events-detail-modal");
  const detailContentNode = container.querySelector("#events-detail-content");
  const readFilters = () => ({
    type: String(container.querySelector("#events-type")?.value || "").trim().toLowerCase(),
    severity: String(container.querySelector("#events-severity")?.value || "").trim().toLowerCase(),
    site: String(container.querySelector("#events-site")?.value || "").trim().toLowerCase(),
    from: String(container.querySelector("#events-from")?.value || "").trim(),
    to: String(container.querySelector("#events-to")?.value || "").trim()
  });
  const toEpoch = (value) => {
    if (!value) {
      return null;
    }
    const t = Date.parse(value);
    return Number.isNaN(t) ? null : t;
  };

  let currentFiltered = [];
  let currentSiteNamesByID = new Map();
  const pagingState = {
    pageSize: 10,
    page: 1
  };

  const getPaginationMeta = (total) => {
    const totalPages = Math.max(1, Math.ceil(total / pagingState.pageSize));
    if (pagingState.page > totalPages) {
      pagingState.page = totalPages;
    }
    if (pagingState.page < 1) {
      pagingState.page = 1;
    }
    const start = total === 0 ? 0 : (pagingState.page - 1) * pagingState.pageSize;
    const end = Math.min(start + pagingState.pageSize, total);
    return { totalPages, start, end };
  };

  const renderRows = (items, siteNamesByID) => {
    if (!Array.isArray(items) || items.length === 0) {
      list.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("events.empty"))}</div>`;
      return;
    }
    const meta = getPaginationMeta(items.length);
    const pageItems = items.slice(meta.start, meta.end);
    list.innerHTML = `
      <div class="waf-table-wrap">
        <table class="waf-table">
          <thead>
            <tr>
              <th>${escapeHtml(ctx.t("events.col.time"))}</th>
              <th>${escapeHtml(ctx.t("events.col.type"))}</th>
              <th>${escapeHtml(ctx.t("events.col.severity"))}</th>
              <th>${escapeHtml(ctx.t("events.col.site"))}</th>
              <th>${escapeHtml(ctx.t("events.col.summary"))}</th>
            </tr>
          </thead>
          <tbody>
            ${pageItems.map((item, index) => `
              <tr class="waf-table-row-clickable" data-event-row="${meta.start + index}" tabindex="0" role="button">
                <td>${escapeHtml(String(item.occurred_at || ""))}</td>
                <td>${escapeHtml(translateEventType(item.type, ctx))}</td>
                <td>${escapeHtml(translateEventSeverity(item.severity, ctx))}</td>
                <td>${escapeHtml(siteNamesByID.get(String(item.site_id || "")) || String(item.site_id || "-"))}</td>
                <td>${escapeHtml(translateEventSummary(item, ctx))}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
      <div class="waf-pager">
        <div class="waf-inline">
          <label for="events-page-size">${escapeHtml(ctx.t("activity.filter.pageSize"))}</label>
          <select id="events-page-size">
            <option value="10"${pagingState.pageSize === 10 ? " selected" : ""}>10</option>
            <option value="25"${pagingState.pageSize === 25 ? " selected" : ""}>25</option>
            <option value="50"${pagingState.pageSize === 50 ? " selected" : ""}>50</option>
            <option value="100"${pagingState.pageSize === 100 ? " selected" : ""}>100</option>
          </select>
        </div>
        <div class="waf-actions">
          ${buildPageButtons(meta.totalPages, pagingState.page, "data-events-page")}
        </div>
      </div>
    `;

    list.querySelector("#events-page-size")?.addEventListener("change", (event) => {
      const nextSize = Number.parseInt(String(event.target?.value || "10"), 10);
      if (!Number.isFinite(nextSize) || nextSize <= 0) {
        return;
      }
      pagingState.pageSize = nextSize;
      pagingState.page = 1;
      renderRows(items, siteNamesByID);
      status.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("events.total"))}: ${items.length}</div>`;
    });

    list.querySelectorAll("[data-events-page]").forEach((button) => {
      button.addEventListener("click", () => {
        const nextPage = Number.parseInt(String(button.dataset.eventsPage || "1"), 10);
        if (!Number.isFinite(nextPage) || nextPage < 1) {
          return;
        }
        pagingState.page = nextPage;
        renderRows(items, siteNamesByID);
      });
    });
  };

  const renderEventDetail = (item, siteNamesByID) => {
    const details = item?.details && typeof item.details === "object" ? item.details : {};
    const fields = [
      ["events.detail.field.time", String(item?.occurred_at || "-")],
      ["events.detail.field.type", translateEventType(item?.type, ctx) || "-"],
      ["events.detail.field.severity", translateEventSeverity(item?.severity, ctx) || "-"],
      ["events.detail.field.site", siteNamesByID.get(String(item?.site_id || "")) || String(item?.site_id || "-")],
      ["events.detail.field.source", String(item?.source_component || "-")],
      ["events.detail.field.summary", translateEventSummary(item, ctx) || "-"],
      ["events.detail.field.relatedRevision", String(item?.related_revision_id || "-")],
      ["events.detail.field.relatedJob", String(item?.related_job_id || "-")],
      ["events.detail.field.relatedCertificate", String(item?.related_certificate_id || "-")],
      ["events.detail.field.relatedRule", String(item?.related_rule_id || "-")]
    ];

    return `
      <div class="waf-table-wrap">
        <table class="waf-table waf-detail-table">
          <tbody>
            ${fields.map(([labelKey, value]) => `
              <tr>
                <th>${escapeHtml(ctx.t(labelKey))}</th>
                <td><pre class="waf-code">${escapeHtml(String(value ?? "-"))}</pre></td>
              </tr>
            `).join("")}
            <tr>
              <th>${escapeHtml(ctx.t("events.detail.field.details"))}</th>
              <td><pre class="waf-code">${escapeHtml(JSON.stringify(details, null, 2) || "{}")}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    `;
  };

  const applyFilters = (items, siteNamesByID) => {
    const f = readFilters();
    const fromEpoch = toEpoch(f.from);
    const toEpochValue = toEpoch(f.to);
    const filtered = (items || []).filter((item) => {
      const type = String(translateEventType(item.type, ctx) || item.type || "").toLowerCase();
      const severity = String(translateEventSeverity(item.severity, ctx) || item.severity || "").toLowerCase();
      const site = String(siteNamesByID.get(String(item.site_id || "")) || item.site_id || "").toLowerCase();
      const occurred = toEpoch(String(item.occurred_at || ""));
      if (f.type && !type.includes(f.type)) {
        return false;
      }
      if (f.severity && !severity.includes(f.severity)) {
        return false;
      }
      if (f.site && !site.includes(f.site)) {
        return false;
      }
      if (fromEpoch !== null && occurred !== null && occurred < fromEpoch) {
        return false;
      }
      if (toEpochValue !== null && occurred !== null && occurred > toEpochValue) {
        return false;
      }
      return true;
    });
    pagingState.page = 1;
    currentFiltered = filtered;
    currentSiteNamesByID = siteNamesByID;
    renderRows(filtered, siteNamesByID);
    status.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("events.total"))}: ${filtered.length}</div>`;
  };

  const openDetails = (item) => {
    if (!detailModalNode || !detailContentNode) {
      return;
    }
    detailContentNode.innerHTML = renderEventDetail(item, currentSiteNamesByID);
    detailModalNode.classList.remove("waf-hidden");
    detailModalNode.focus();
  };

  const openRowFromTarget = (target) => {
    const rowNode = target?.closest?.("[data-event-row]");
    if (!rowNode) {
      return;
    }
    const index = Number(rowNode.dataset.eventRow || "-1");
    if (index < 0 || index >= currentFiltered.length) {
      return;
    }
    openDetails(currentFiltered[index]);
  };

  list.addEventListener("click", (event) => {
    openRowFromTarget(event.target);
  });
  list.addEventListener("pointerup", (event) => {
    if (event.button !== 0) {
      return;
    }
    openRowFromTarget(event.target);
  });
  list.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    const rowNode = event.target?.closest?.("[data-event-row]");
    if (!rowNode) {
      return;
    }
    event.preventDefault();
    openRowFromTarget(event.target);
  });

  const closeDetails = () => detailModalNode?.classList.add("waf-hidden");
  detailModalNode?.querySelectorAll("[data-events-detail-close='true']").forEach((node) => {
    node.addEventListener("click", closeDetails);
  });
  detailModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeDetails();
    }
  });

  setLoading(status, ctx.t("events.loading"));
  try {
    const [eventsResponse, sitesResponse, secondarySites] = await Promise.all([
      ctx.api.get("/api/events"),
      ctx.api.get("/api/sites"),
      tryGetJSON("/api-app/sites")
    ]);
    const allSites = [
      ...normalizeList(sitesResponse),
      ...unwrapList(secondarySites, ["sites"])
    ];
    const siteNamesByID = new Map(
      allSites
        .map((site) => [String(site?.id || ""), String(site?.primary_host || site?.id || "").trim()])
        .filter(([id, host]) => id && host)
    );
    const items = normalizeList(eventsResponse?.events)
      .filter((item) => !shouldSkipInternalSite(item?.site_id))
      .slice()
      .sort((a, b) => String(b.occurred_at || "").localeCompare(String(a.occurred_at || "")));
    applyFilters(items, siteNamesByID);
    container.querySelector("#events-filters")?.addEventListener("input", () => applyFilters(items, siteNamesByID));
  } catch (error) {
    setError(status, ctx.t("events.error.load"));
    renderRows([], new Map());
  }
}
