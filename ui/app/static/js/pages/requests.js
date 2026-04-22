import { escapeHtml, formatDate, setError, setLoading } from "../ui.js";

const DEFAULT_REQUESTS_FETCH_LIMIT = 2000;

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function normalizeToken(value) {
  return String(value || "").trim().toLowerCase();
}

function normalizeHostLike(value) {
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
    } catch (_error) {
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
    } catch (_error) {
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

function parseTimestamp(value) {
  const stamp = Date.parse(String(value || ""));
  return Number.isFinite(stamp) ? new Date(stamp) : null;
}

function toDateKeyLocal(date) {
  if (!(date instanceof Date)) {
    return "";
  }
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function normalizeTimePreset(value) {
  const allowed = new Set(["all", "minute", "day", "month", "date", "datetime"]);
  const preset = String(value || "all").trim().toLowerCase();
  return allowed.has(preset) ? preset : "all";
}

function todayDateKeyLocal() {
  return toDateKeyLocal(new Date());
}

export async function renderRequests(container, ctx) {
  const signal = ctx?.signal;
  const isActive = typeof ctx?.isActive === "function" ? ctx.isActive : () => true;

  const state = {
    rows: [],
    filteredRows: [],
    search: "",
    sortBy: "timestamp",
    sortDirection: "desc",
    pageSize: 10,
    page: 1,
    fetchLimit: DEFAULT_REQUESTS_FETCH_LIMIT,
    selectedService: "",
    selectedMethod: "",
    selectedStatus: "",
    selectedTimePreset: "date",
    selectedDate: todayDateKeyLocal(),
    selectedDateTime: "",
    storageLogsDays: 14,
    loggingSummary: null,
    serviceOptions: [],
    methodOptions: [],
    statusOptions: []
  };

  const columns = [
    { id: "service", labelKey: "requests.col.service", mode: "string", value: (row) => String(row?.serviceDisplay || "") },
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

  const resolveServiceDisplay = (rawSite, siteHostMap) => {
    const source = String(rawSite || "").trim();
    if (!source) {
      return "-";
    }
    const mapped = siteHostMap.get(normalizeHostLike(source));
    if (mapped) {
      return mapped;
    }
    if (source.includes(".")) {
      return source;
    }
    if (/^[a-z0-9_]+$/i.test(source)) {
      return source.replace(/_/g, ".");
    }
    return source;
  };

  const rebuildOptions = () => {
    const services = new Set();
    const methods = new Set();
    const statuses = new Set();
    for (const row of state.rows) {
      const service = String(row?.serviceDisplay || "").trim();
      const method = String(row?.entry?.method || "").trim().toUpperCase();
      const status = String(row?.entry?.status ?? "").trim();
      if (service && service !== "-") {
        services.add(service);
      }
      if (method) {
        methods.add(method);
      }
      if (status) {
        statuses.add(status);
      }
    }
    state.serviceOptions = Array.from(services).sort((left, right) => left.localeCompare(right, undefined, { sensitivity: "base" }));
    state.methodOptions = Array.from(methods).sort((left, right) => left.localeCompare(right, undefined, { sensitivity: "base" }));
    state.statusOptions = Array.from(statuses).sort((left, right) => Number(left) - Number(right));
    if (state.selectedService && !state.serviceOptions.includes(state.selectedService)) {
      state.selectedService = "";
    }
    if (state.selectedMethod && !state.methodOptions.includes(state.selectedMethod)) {
      state.selectedMethod = "";
    }
    if (state.selectedStatus && !state.statusOptions.includes(state.selectedStatus)) {
      state.selectedStatus = "";
    }
  };

  const matchTimeFilter = (rowDate) => {
    const preset = normalizeTimePreset(state.selectedTimePreset);
    if (preset === "all") {
      return true;
    }
    if (!(rowDate instanceof Date)) {
      return false;
    }
    const now = Date.now();
    const rowStamp = rowDate.getTime();
    if (preset === "minute") {
      return rowStamp >= now - 60 * 1000;
    }
    if (preset === "day") {
      return rowStamp >= now - 24 * 60 * 60 * 1000;
    }
    if (preset === "month") {
      return rowStamp >= now - 30 * 24 * 60 * 60 * 1000;
    }
    if (preset === "date") {
      if (!state.selectedDate) {
        return true;
      }
      return toDateKeyLocal(rowDate) === state.selectedDate;
    }
    if (preset === "datetime") {
      if (!state.selectedDateTime) {
        return true;
      }
      const targetStamp = Date.parse(state.selectedDateTime);
      if (!Number.isFinite(targetStamp)) {
        return true;
      }
      return rowStamp >= targetStamp && rowStamp < (targetStamp + 60 * 1000);
    }
    return true;
  };

  const applyFiltersAndSort = () => {
    const search = normalizeToken(state.search);
    const sortColumn = columns.find((column) => column.id === state.sortBy) || columns[0];
    const factor = state.sortDirection === "asc" ? 1 : -1;
    const filtered = state.rows.filter((row) => {
      const service = String(row?.serviceDisplay || "").trim();
      const method = String(row?.entry?.method || "").trim().toUpperCase();
      const status = String(row?.entry?.status ?? "").trim();
      const rowDate = row?.timestampDate || null;
      if (state.selectedService && service !== state.selectedService) {
        return false;
      }
      if (state.selectedMethod && method !== state.selectedMethod) {
        return false;
      }
      if (state.selectedStatus && status !== state.selectedStatus) {
        return false;
      }
      if (!matchTimeFilter(rowDate)) {
        return false;
      }
      if (!search) {
        return true;
      }
      const haystack = [
        row?.serviceDisplay,
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
    if (!isActive()) {
      return;
    }
    applyFiltersAndSort();
    const meta = getPaginationMeta(state.filteredRows.length);
    const pageRows = state.filteredRows.slice(meta.start, meta.end);

    const renderRequestDetail = (row) => {
      const entry = row?.entry && typeof row.entry === "object" ? row.entry : {};
      const fields = [
        ["requests.detail.time", formatDate(String(entry.timestamp || row?.ingested_at || "")) || "-"],
        ["requests.detail.service", String(row?.serviceDisplay || "-")],
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

    const timePreset = normalizeTimePreset(state.selectedTimePreset);
    container.innerHTML = `
      <section class="waf-card no-page-card-head">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("app.requests"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("requests.subtitle"))}</div>
            <div class="waf-note">${escapeHtml(ctx.t("requests.storageTier", {
              hot: String(state.loggingSummary?.hot_backend || "file"),
              cold: String(state.loggingSummary?.cold_backend || "file"),
              retention: String(state.storageLogsDays || 14),
            }))}</div>
          </div>
          <button class="btn ghost btn-sm" id="requests-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="waf-form-grid requests-filter-row requests-filter-row-main">
            <div class="waf-field">
              <label for="requests-filter-service">${escapeHtml(ctx.t("requests.filter.service"))}</label>
              <select id="requests-filter-service">
                <option value="">${escapeHtml(ctx.t("requests.filter.all"))}</option>
                ${state.serviceOptions.map((item) => `<option value="${escapeHtml(item)}"${state.selectedService === item ? " selected" : ""}>${escapeHtml(item)}</option>`).join("")}
              </select>
            </div>
            <div class="waf-field">
              <label for="requests-filter-method">${escapeHtml(ctx.t("requests.filter.method"))}</label>
              <select id="requests-filter-method">
                <option value="">${escapeHtml(ctx.t("requests.filter.all"))}</option>
                ${state.methodOptions.map((item) => `<option value="${escapeHtml(item)}"${state.selectedMethod === item ? " selected" : ""}>${escapeHtml(item)}</option>`).join("")}
              </select>
            </div>
            <div class="waf-field">
              <label for="requests-filter-status">${escapeHtml(ctx.t("requests.filter.status"))}</label>
              <select id="requests-filter-status">
                <option value="">${escapeHtml(ctx.t("requests.filter.all"))}</option>
                ${state.statusOptions.map((item) => `<option value="${escapeHtml(item)}"${state.selectedStatus === item ? " selected" : ""}>${escapeHtml(item)}</option>`).join("")}
              </select>
            </div>
            <div class="waf-field">
              <label for="requests-filter-time">${escapeHtml(ctx.t("requests.filter.time"))}</label>
              <select id="requests-filter-time">
                <option value="all"${timePreset === "all" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.all"))}</option>
                <option value="minute"${timePreset === "minute" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.minute"))}</option>
                <option value="day"${timePreset === "day" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.day"))}</option>
                <option value="month"${timePreset === "month" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.month"))}</option>
                <option value="date"${timePreset === "date" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.date"))}</option>
                <option value="datetime"${timePreset === "datetime" ? " selected" : ""}>${escapeHtml(ctx.t("requests.time.datetime"))}</option>
              </select>
            </div>
          </div>
          <div class="waf-form-grid requests-filter-row requests-filter-row-secondary">
            <div class="waf-field${timePreset === "date" ? "" : " waf-hidden"}">
              <label for="requests-filter-date">${escapeHtml(ctx.t("requests.filter.date"))}</label>
              <input id="requests-filter-date" type="date" value="${escapeHtml(state.selectedDate)}">
            </div>
            <div class="waf-field${timePreset === "datetime" ? "" : " waf-hidden"}">
              <label for="requests-filter-datetime">${escapeHtml(ctx.t("requests.filter.datetime"))}</label>
              <input id="requests-filter-datetime" type="datetime-local" value="${escapeHtml(state.selectedDateTime)}">
            </div>
          </div>
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
                    const isActiveCol = state.sortBy === column.id;
                    const marker = isActiveCol ? (state.sortDirection === "asc" ? " ↑" : " ↓") : "";
                    return `<th><button type="button" class="waf-table-sort" data-sort-col="${escapeHtml(column.id)}">${escapeHtml(ctx.t(column.labelKey))}${marker}</button></th>`;
                  }).join("")}
                </tr>
              </thead>
              <tbody>
                ${pageRows.length
                  ? pageRows.map((row, index) => `
                    <tr class="waf-table-row-clickable" data-request-row="${meta.start + index}" tabindex="0" role="button">
                      <td>${escapeHtml(String(row?.serviceDisplay || "-"))}</td>
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
              <select id="requests-page-size" class="select select-compact">
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
    container.querySelector("#requests-filter-service")?.addEventListener("change", (event) => {
      state.selectedService = String(event.target?.value || "");
      state.page = 1;
      render();
    });
    container.querySelector("#requests-filter-method")?.addEventListener("change", (event) => {
      state.selectedMethod = String(event.target?.value || "").toUpperCase();
      state.page = 1;
      render();
    });
    container.querySelector("#requests-filter-status")?.addEventListener("change", (event) => {
      state.selectedStatus = String(event.target?.value || "");
      state.page = 1;
      render();
    });
    container.querySelector("#requests-filter-time")?.addEventListener("change", async (event) => {
      state.selectedTimePreset = normalizeTimePreset(event.target?.value || "all");
      state.page = 1;
      if (state.selectedTimePreset === "date") {
        await load();
        return;
      }
      render();
    });
    container.querySelector("#requests-filter-date")?.addEventListener("change", async (event) => {
      state.selectedDate = String(event.target?.value || "") || todayDateKeyLocal();
      state.page = 1;
      await load();
    });
    container.querySelector("#requests-filter-datetime")?.addEventListener("change", (event) => {
      state.selectedDateTime = String(event.target?.value || "");
      state.page = 1;
      render();
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
    if (!isActive()) {
      return;
    }
    try {
      setLoading(container, ctx.t("requests.loading"));
      const runtimeSettings = await ctx.api.get("/api/settings/runtime", { signal }).catch(() => null);
      const logsDays = Number(runtimeSettings?.storage?.logs_days || 14);
      if (Number.isFinite(logsDays) && logsDays > 0) {
        state.storageLogsDays = logsDays;
      }
      state.loggingSummary = runtimeSettings?.logging_summary || null;
      const params = new URLSearchParams();
      params.set("limit", String(state.fetchLimit));
      params.set("day", state.selectedDate || todayDateKeyLocal());
      params.set("retention_days", String(state.storageLogsDays || 14));
      const [response, sites] = await Promise.all([
        fetch(`/api/requests?${params.toString()}`, {
          method: "GET",
          credentials: "include",
          headers: { Accept: "text/plain" },
          signal
        }),
        ctx.api.get("/api/sites", { signal }).catch(() => []),
      ]);
      if (!isActive()) {
        return;
      }
      if (response.status === 404) {
        state.rows = [];
        state.page = 1;
        rebuildOptions();
        render();
        return;
      }
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const text = await response.text();
      if (!isActive()) {
        return;
      }
      const siteHostMap = new Map();
      for (const site of normalizeList(sites)) {
        const id = normalizeHostLike(site?.id);
        const host = String(site?.primary_host || "").trim();
        if (id && host) {
          siteHostMap.set(id, host);
        }
      }
      state.rows = parseRequestsJSONL(text).map((row) => {
        const entry = row?.entry && typeof row.entry === "object" ? row.entry : {};
        const serviceDisplay = resolveServiceDisplay(entry.site, siteHostMap);
        const timestampDate = parseTimestamp(entry.timestamp || row?.ingested_at);
        return { ...row, entry, serviceDisplay, timestampDate };
      });
      rebuildOptions();
      state.page = 1;
      render();
    } catch (error) {
      if (!isActive()) {
        return;
      }
      if (error?.name === "AbortError") {
        return;
      }
      setError(container, ctx.t("requests.error.load"));
    }
  };

  await load();
}
