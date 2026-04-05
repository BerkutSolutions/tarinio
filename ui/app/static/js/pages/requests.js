import { escapeHtml, formatDate, setError, setLoading } from "../ui.js";

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function normalizeToken(value) {
  return String(value || "").trim().toLowerCase();
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

function parseRequestsJSONL(text) {
  const raw = String(text || "").trim();
  if (!raw) {
    return [];
  }
  if (raw.startsWith("[")) {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed.filter((row) => row && typeof row === "object") : [];
    } catch (error) {
      return [];
    }
  }
  const rows = [];
  for (const sourceLine of raw.split(/\r?\n/)) {
    const line = String(sourceLine || "").trim();
    if (!line) {
      continue;
    }
    try {
      const parsed = JSON.parse(line);
      if (parsed && typeof parsed === "object") {
        rows.push(parsed);
      }
    } catch (error) {
      rows.push({ stream: "archive", ingested_at: "", raw: line, entry: {} });
    }
  }
  return rows;
}

function compareValues(left, right, mode) {
  if (mode === "number") {
    return Number(left || 0) - Number(right || 0);
  }
  return String(left || "").localeCompare(String(right || ""), undefined, { sensitivity: "base" });
}

export async function renderRequests(container, ctx) {
  const state = {
    rows: [],
    filteredRows: [],
    search: "",
    sortBy: "timestamp",
    sortDirection: "desc",
    pageSize: 10,
    page: 1
  };

  const columns = [
    { id: "service", labelKey: "requests.col.service", mode: "string", value: (row) => String(row?.entry?.site || "") },
    { id: "timestamp", labelKey: "requests.col.time", mode: "string", value: (row) => String(row?.entry?.timestamp || row?.ingested_at || "") },
    { id: "request_id", labelKey: "requests.col.requestId", mode: "string", value: (row) => String(row?.entry?.request_id || "") },
    { id: "method", labelKey: "requests.col.method", mode: "string", value: (row) => String(row?.entry?.method || "") },
    { id: "uri", labelKey: "requests.col.uri", mode: "string", value: (row) => String(row?.entry?.uri || "") },
    { id: "status", labelKey: "requests.col.status", mode: "number", value: (row) => Number(row?.entry?.status || 0) },
    { id: "client_ip", labelKey: "requests.col.clientIp", mode: "string", value: (row) => String(row?.entry?.client_ip || "") },
    { id: "upstream", labelKey: "requests.col.upstream", mode: "string", value: (row) => String(row?.entry?.upstream_addr || "") },
    { id: "stream", labelKey: "requests.col.stream", mode: "string", value: (row) => String(row?.stream || "") }
  ];

  const getPaginationMeta = (total) => {
    const totalPages = Math.max(1, Math.ceil(total / state.pageSize));
    if (state.page > totalPages) {
      state.page = totalPages;
    }
    if (state.page < 1) {
      state.page = 1;
    }
    const start = total === 0 ? 0 : (state.page - 1) * state.pageSize;
    const end = Math.min(start + state.pageSize, total);
    return { totalPages, start, end };
  };

  const applyFiltersAndSort = () => {
    const search = normalizeToken(state.search);
    const sortColumn = columns.find((column) => column.id === state.sortBy) || columns[0];
    const factor = state.sortDirection === "asc" ? 1 : -1;
    const filtered = state.rows.filter((row) => {
      if (!search) {
        return true;
      }
      const haystack = [
        row?.entry?.site,
        row?.entry?.request_id,
        row?.entry?.method,
        row?.entry?.uri,
        row?.entry?.status,
        row?.entry?.client_ip,
        row?.entry?.upstream_addr,
        row?.stream
      ]
        .map((item) => String(item || "").toLowerCase())
        .join(" ");
      return haystack.includes(search);
    });
    filtered.sort((left, right) => factor * compareValues(sortColumn.value(left), sortColumn.value(right), sortColumn.mode));
    state.filteredRows = filtered;
  };

  const render = () => {
    applyFiltersAndSort();
    const meta = getPaginationMeta(state.filteredRows.length);
    const pageRows = state.filteredRows.slice(meta.start, meta.end);

    const renderRequestDetail = (row) => {
      const entry = row?.entry && typeof row.entry === "object" ? row.entry : {};
      const fields = [
        ["requests.detail.time", formatDate(String(entry.timestamp || row?.ingested_at || "")) || "-"],
        ["requests.detail.service", String(entry.site || "-")],
        ["requests.detail.requestId", String(entry.request_id || "-")],
        ["requests.detail.method", String(entry.method || "-")],
        ["requests.detail.uri", String(entry.uri || "-")],
        ["requests.detail.status", String(entry.status ?? "-")],
        ["requests.detail.clientIp", String(entry.client_ip || "-")],
        ["requests.detail.upstream", String(entry.upstream_addr || "-")],
        ["requests.detail.stream", String(row?.stream || "-")],
        ["requests.detail.referer", String(entry.referer || "-")],
        ["requests.detail.userAgent", String(entry.user_agent || "-")]
      ];
      return `
        <div class="waf-table-wrap">
          <table class="waf-table waf-detail-table">
            <tbody>
              ${fields.map(([labelKey, value]) => `
                <tr>
                  <th>${escapeHtml(ctx.t(labelKey))}</th>
                  <td><pre class="waf-code">${escapeHtml(value)}</pre></td>
                </tr>
              `).join("")}
            </tbody>
          </table>
        </div>
        <div class="waf-note">
          <div><strong>${escapeHtml(ctx.t("requests.detail.raw"))}</strong></div>
          <pre class="waf-code">${escapeHtml(JSON.stringify(row, null, 2))}</pre>
        </div>
      `;
    };

    container.innerHTML = `
      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("app.requests"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("requests.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" id="requests-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="waf-field">
            <label for="requests-search">${escapeHtml(ctx.t("requests.search"))}</label>
            <input id="requests-search" value="${escapeHtml(state.search)}" placeholder="${escapeHtml(ctx.t("requests.searchPlaceholder"))}">
          </div>
          <div id="requests-status" class="waf-empty">${escapeHtml(ctx.t("requests.total"))}: ${state.filteredRows.length}</div>
          <div class="waf-table-wrap">
            <table class="waf-table">
              <thead>
                <tr>
                  ${columns.map((column) => {
                    const isActive = state.sortBy === column.id;
                    const marker = isActive ? (state.sortDirection === "asc" ? " ▲" : " ▼") : "";
                    return `<th><button type="button" class="waf-table-sort" data-sort-col="${escapeHtml(column.id)}">${escapeHtml(ctx.t(column.labelKey))}${marker}</button></th>`;
                  }).join("")}
                </tr>
              </thead>
              <tbody>
                ${pageRows.length
                  ? pageRows.map((row, index) => `
                    <tr class="waf-table-row-clickable" data-request-row="${meta.start + index}" tabindex="0" role="button">
                      <td>${escapeHtml(String(row?.entry?.site || "-"))}</td>
                      <td>${escapeHtml(formatDate(String(row?.entry?.timestamp || row?.ingested_at || "")))}</td>
                      <td><span class="waf-code">${escapeHtml(String(row?.entry?.request_id || "-"))}</span></td>
                      <td>${escapeHtml(String(row?.entry?.method || "-"))}</td>
                      <td>${escapeHtml(String(row?.entry?.uri || "-"))}</td>
                      <td>${escapeHtml(String(row?.entry?.status ?? "-"))}</td>
                      <td>${escapeHtml(String(row?.entry?.client_ip || "-"))}</td>
                      <td>${escapeHtml(String(row?.entry?.upstream_addr || "-"))}</td>
                      <td>${escapeHtml(String(row?.stream || "-"))}</td>
                    </tr>
                  `).join("")
                  : `<tr><td colspan="${columns.length}"><div class="waf-empty">${escapeHtml(ctx.t("requests.empty"))}</div></td></tr>`}
              </tbody>
            </table>
          </div>
          <div class="waf-pager">
            <div class="waf-inline">
              <label for="requests-page-size">${escapeHtml(ctx.t("activity.filter.pageSize"))}</label>
              <select id="requests-page-size">
                <option value="10"${state.pageSize === 10 ? " selected" : ""}>10</option>
                <option value="25"${state.pageSize === 25 ? " selected" : ""}>25</option>
                <option value="50"${state.pageSize === 50 ? " selected" : ""}>50</option>
                <option value="100"${state.pageSize === 100 ? " selected" : ""}>100</option>
              </select>
            </div>
            <div class="waf-actions">
              ${buildPageButtons(meta.totalPages, state.page, "data-requests-page")}
            </div>
          </div>
        </div>
      </section>
      <div class="waf-modal waf-hidden" id="requests-detail-modal" role="dialog" aria-modal="true" aria-labelledby="requests-detail-title" tabindex="-1">
        <button class="waf-modal-overlay" type="button" data-requests-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
        <div class="waf-modal-card">
          <div class="waf-card-head">
            <div>
              <h3 id="requests-detail-title">${escapeHtml(ctx.t("requests.detail.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("requests.detail.subtitle"))}</div>
            </div>
            <button class="btn ghost btn-sm" type="button" data-requests-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
          </div>
          <div class="waf-card-body" id="requests-detail-content">
            <div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>
          </div>
        </div>
      </div>
    `;

    container.querySelector("#requests-refresh")?.addEventListener("click", load);
    container.querySelector("#requests-search")?.addEventListener("input", (event) => {
      state.search = event.target.value;
      state.page = 1;
      const cursor = Number(event.target.selectionStart || state.search.length);
      render();
      const nextInput = container.querySelector("#requests-search");
      if (nextInput) {
        nextInput.focus();
        nextInput.setSelectionRange(cursor, cursor);
      }
    });

    container.querySelector("#requests-page-size")?.addEventListener("change", (event) => {
      const nextSize = Number.parseInt(String(event.target?.value || "10"), 10);
      if (!Number.isFinite(nextSize) || nextSize <= 0) {
        return;
      }
      state.pageSize = nextSize;
      state.page = 1;
      render();
    });

    container.querySelectorAll("[data-requests-page]").forEach((button) => {
      button.addEventListener("click", () => {
        const nextPage = Number.parseInt(String(button.dataset.requestsPage || "1"), 10);
        if (!Number.isFinite(nextPage) || nextPage < 1) {
          return;
        }
        state.page = nextPage;
        render();
      });
    });

    container.querySelectorAll("[data-sort-col]").forEach((button) => {
      button.addEventListener("click", () => {
        const columnID = String(button.dataset.sortCol || "");
        if (!columnID) {
          return;
        }
        if (state.sortBy === columnID) {
          state.sortDirection = state.sortDirection === "asc" ? "desc" : "asc";
        } else {
          state.sortBy = columnID;
          state.sortDirection = columnID === "timestamp" ? "desc" : "asc";
        }
        state.page = 1;
        render();
      });
    });

    const detailModalNode = container.querySelector("#requests-detail-modal");
    const detailContentNode = container.querySelector("#requests-detail-content");
    const openDetails = (row) => {
      if (!detailModalNode || !detailContentNode) {
        return;
      }
      detailContentNode.innerHTML = renderRequestDetail(row);
      detailModalNode.classList.remove("waf-hidden");
      detailModalNode.focus();
    };

    const openRowFromTarget = (target) => {
      const rowNode = target?.closest?.("[data-request-row]");
      if (!rowNode) {
        return;
      }
      const index = Number(rowNode.dataset.requestRow || "-1");
      if (index < 0 || index >= state.filteredRows.length) {
        return;
      }
      openDetails(state.filteredRows[index]);
    };

    container.querySelector("tbody")?.addEventListener("click", (event) => {
      openRowFromTarget(event.target);
    });
    container.querySelector("tbody")?.addEventListener("pointerup", (event) => {
      if (event.button !== 0) {
        return;
      }
      openRowFromTarget(event.target);
    });
    container.querySelector("tbody")?.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" && event.key !== " ") {
        return;
      }
      const rowNode = event.target?.closest?.("[data-request-row]");
      if (!rowNode) {
        return;
      }
      event.preventDefault();
      openRowFromTarget(rowNode);
    });

    const closeDetails = () => detailModalNode?.classList.add("waf-hidden");
    detailModalNode?.querySelectorAll("[data-requests-detail-close='true']").forEach((node) => {
      node.addEventListener("click", closeDetails);
    });
    detailModalNode?.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        closeDetails();
      }
    });
  };

  const load = async () => {
    try {
      setLoading(container, ctx.t("requests.loading"));
      const response = await fetch("/api/requests", {
        method: "GET",
        credentials: "include",
        headers: { Accept: "text/plain" }
      });
      if (response.status === 404) {
        state.rows = [];
        state.page = 1;
        render();
        return;
      }
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const text = await response.text();
      state.rows = parseRequestsJSONL(text);
      state.page = 1;
      render();
    } catch (error) {
      setError(container, ctx.t("requests.error.load"));
    }
  };

  await load();
}
