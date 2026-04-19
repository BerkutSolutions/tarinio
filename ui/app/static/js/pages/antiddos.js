import { escapeHtml, formatDate, setError } from "../ui.js";

export async function renderAntiDDoS(container, ctx) {
  const upsertWithPostFallback = async (payload) => {
    try {
      await ctx.api.post("/api/anti-ddos/settings", payload);
    } catch (error) {
      if (error?.status !== 404 && error?.status !== 405) {
        throw error;
      }
      await ctx.api.put("/api/anti-ddos/settings", payload);
    }
  };
  const parsePorts = (value) => (value || "")
    .split(",")
    .map((item) => Number(item.trim()))
    .filter((item) => Number.isInteger(item) && item > 0 && item <= 65535);
  const portsToInput = (ports) => (Array.isArray(ports) ? ports.join(", ") : "80, 443");
  const normalizeToken = (value) => String(value || "").trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, "");
  const translateEventType = (type) => {
    const token = normalizeToken(type);
    const key = token ? `events.type.${token}` : "events.type.unknown";
    const value = ctx.t(key);
    return value === key ? (type || ctx.t("common.unknown")) : value;
  };
  const translateEventSummary = (item) => {
    const summaryToken = normalizeToken(item?.summary);
    if (summaryToken) {
      const summaryKey = `events.summary.${summaryToken}`;
      const summaryValue = ctx.t(summaryKey);
      if (summaryValue !== summaryKey) {
        return summaryValue;
      }
    }
    return String(item?.summary || "");
  };
  const translateEventSeverity = (severity) => {
    const token = normalizeToken(severity);
    const key = token ? `events.severity.${token}` : "events.severity.unknown";
    const value = ctx.t(key);
    return value === key ? (severity || ctx.t("common.unknown")) : value;
  };
  const detailsValue = (details, ...keys) => {
    for (const key of keys) {
      const value = details?.[key];
      if (value == null || value === "") {
        continue;
      }
      return value;
    }
    return "";
  };
  const toPrettyString = (value) => {
    if (value == null || value === "") {
      return "-";
    }
    if (typeof value === "object") {
      try {
        return JSON.stringify(value, null, 2);
      } catch (error) {
        return String(value);
      }
    }
    return String(value);
  };
  const renderLogDetail = (item) => {
    const details = (item && typeof item.details === "object" && item.details) ? item.details : {};
    const method = detailsValue(details, "method");
    const path = detailsValue(details, "path", "uri");
    const status = detailsValue(details, "status");
    const requestsPerSecond = detailsValue(details, "requests_second", "path_requests_sec");
    const requestLine = [method, path].filter(Boolean).join(" ");
    const fields = [
      { label: ctx.t("antiddos.model.logs.detail.time"), value: formatDate(String(item?.occurred_at || "")) },
      { label: ctx.t("antiddos.model.logs.detail.type"), value: translateEventType(item?.type) },
      { label: ctx.t("antiddos.model.logs.detail.severity"), value: translateEventSeverity(String(item?.severity || "")) || "-" },
      { label: ctx.t("antiddos.model.logs.detail.site"), value: String(item?.site_name || item?.site_id || "-") },
      { label: ctx.t("antiddos.model.logs.detail.summary"), value: translateEventSummary(item) || "-" },
      { label: ctx.t("antiddos.model.logs.detail.clientIp"), value: toPrettyString(detailsValue(details, "client_ip", "ip")) },
      { label: ctx.t("antiddos.model.logs.detail.request"), value: requestLine || "-" },
      { label: ctx.t("antiddos.model.logs.detail.status"), value: toPrettyString(status) },
      { label: ctx.t("antiddos.model.logs.detail.rate"), value: toPrettyString(requestsPerSecond) },
      { label: ctx.t("antiddos.model.logs.detail.referer"), value: toPrettyString(detailsValue(details, "referer", "referrer")) },
      { label: ctx.t("antiddos.model.logs.detail.userAgent"), value: toPrettyString(detailsValue(details, "user_agent", "device", "ua")) }
    ];
    return `
      <div class="waf-table-wrap">
        <table class="waf-table">
          <tbody>
            ${fields.map((field) => `
              <tr>
                <th>${escapeHtml(String(field.label || "-"))}</th>
                <td><pre class="waf-code">${escapeHtml(String(field.value || "-"))}</pre></td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
      <div class="waf-note">
        <div><strong>${escapeHtml(ctx.t("antiddos.model.logs.detail.raw"))}</strong></div>
        <pre class="waf-code">${escapeHtml(toPrettyString(details))}</pre>
      </div>
    `;
  };

  let logsTimer = null;
  const helpRows = [
    { labelKey: "antiddos.model.enabled", defaultValue: "true", hintKey: "antiddos.help.modelEnabled" },
    { labelKey: "antiddos.model.poll", defaultValue: "2", hintKey: "antiddos.help.modelPoll" },
    { labelKey: "antiddos.model.decay", defaultValue: "0.08", hintKey: "antiddos.help.modelDecay" },
    { labelKey: "antiddos.model.thresholdThrottle", defaultValue: "2.5", hintKey: "antiddos.help.modelThrottleThreshold" },
    { labelKey: "antiddos.model.thresholdDrop", defaultValue: "6.0", hintKey: "antiddos.help.modelDropThreshold" },
    { labelKey: "antiddos.model.hold", defaultValue: "60", hintKey: "antiddos.help.modelHoldSeconds" },
    { labelKey: "antiddos.model.throttleRps", defaultValue: "3", hintKey: "antiddos.help.modelThrottleRps" },
    { labelKey: "antiddos.model.throttleBurst", defaultValue: "6", hintKey: "antiddos.help.modelThrottleBurst" },
    { labelKey: "antiddos.model.throttleTarget", defaultValue: "REJECT", hintKey: "antiddos.help.modelThrottleTarget" },
    { labelKey: "antiddos.model.weight429", defaultValue: "1.0", hintKey: "antiddos.help.modelWeight429" },
    { labelKey: "antiddos.model.weight403", defaultValue: "1.8", hintKey: "antiddos.help.modelWeight403" },
    { labelKey: "antiddos.model.weight444", defaultValue: "2.2", hintKey: "antiddos.help.modelWeight444" },
    { labelKey: "antiddos.model.emergencyRps", defaultValue: "180", hintKey: "antiddos.help.modelEmergencyRps" },
    { labelKey: "antiddos.model.emergencyUnique", defaultValue: "40", hintKey: "antiddos.help.modelEmergencyUniqueIps" },
    { labelKey: "antiddos.model.emergencyPerIp", defaultValue: "60", hintKey: "antiddos.help.modelEmergencyPerIpRps" },
    { labelKey: "antiddos.model.weightEmergencyBotnet", defaultValue: "6.0", hintKey: "antiddos.help.modelWeightEmergencyBotnet" },
    { labelKey: "antiddos.model.weightEmergencySingle", defaultValue: "4.0", hintKey: "antiddos.help.modelWeightEmergencySingle" }
  ];

  const renderLogs = async () => {
    const node = container.querySelector("#antiddos-model-logs");
    if (!node) {
      return;
    }
    try {
      const payload = await ctx.api.get("/api/events");
      const events = Array.isArray(payload?.events) ? payload.events : [];
      const rows = events
        .filter((item) => ["security_waf", "security_rate_limit", "security_access"].includes(String(item?.type || "")))
        .sort((a, b) => {
          const left = Date.parse(String(a?.occurred_at || "")) || 0;
          const right = Date.parse(String(b?.occurred_at || "")) || 0;
          return right - left;
        })
        .slice(0, 80);
      if (!rows.length) {
        node.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("antiddos.model.logs.empty"))}</div>`;
        return;
      }
      node.innerHTML = `
        <div class="waf-table-wrap">
          <table class="waf-table">
            <thead>
              <tr>
                <th>${escapeHtml(ctx.t("antiddos.model.logs.col.time"))}</th>
                <th>${escapeHtml(ctx.t("antiddos.model.logs.col.type"))}</th>
                <th>${escapeHtml(ctx.t("antiddos.model.logs.col.site"))}</th>
                <th>${escapeHtml(ctx.t("antiddos.model.logs.col.summary"))}</th>
                <th>${escapeHtml(ctx.t("antiddos.model.logs.col.ip"))}</th>
              </tr>
            </thead>
            <tbody>
              ${rows.map((item, index) => `
                <tr class="waf-table-row-clickable" data-log-index="${index}" tabindex="0" role="button">
                  <td>${escapeHtml(formatDate(String(item?.occurred_at || "")))}</td>
                  <td>${escapeHtml(translateEventType(String(item?.type || "")))}</td>
                  <td>${escapeHtml(String(item?.site_name || item?.site_id || "-"))}</td>
                  <td>${escapeHtml(translateEventSummary(item) || "-")}</td>
                  <td>${escapeHtml(String(item?.details?.client_ip || item?.details?.ip || "-"))}</td>
                </tr>
              `).join("")}
            </tbody>
          </table>
        </div>
      `;
      const modalNode = container.querySelector("#antiddos-model-log-detail-modal");
      const modalBody = container.querySelector("#antiddos-model-log-detail-content");
      const openDetails = (item) => {
        if (!modalNode || !modalBody) {
          return;
        }
        modalBody.innerHTML = renderLogDetail(item);
        modalNode.classList.remove("waf-hidden");
        modalNode.focus();
      };
      const openRowFromTarget = (target) => {
        const rowNode = target?.closest?.("[data-log-index]");
        if (!rowNode) {
          return;
        }
        const index = Number(rowNode.dataset.logIndex || "-1");
        if (index < 0 || index >= rows.length) {
          return;
        }
        openDetails(rows[index]);
      };
      const bodyNode = node.querySelector("tbody");
      bodyNode?.addEventListener("click", (event) => {
        openRowFromTarget(event.target);
      });
      bodyNode?.addEventListener("pointerup", (event) => {
        if (event.button !== 0) {
          return;
        }
        openRowFromTarget(event.target);
      });
      bodyNode?.addEventListener("keydown", (event) => {
        if (event.key !== "Enter" && event.key !== " ") {
          return;
        }
        const rowNode = event.target?.closest?.("[data-log-index]");
        if (!rowNode) {
          return;
        }
        event.preventDefault();
        openRowFromTarget(rowNode);
      });
    } catch (error) {
      setError(node, ctx.t("antiddos.model.logs.error"));
    }
  };

  const render = (settings) => {
    if (logsTimer) {
      clearInterval(logsTimer);
      logsTimer = null;
    }

    container.innerHTML = `
      <section class="waf-card waf-antiddos-feed-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("antiddos.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("antiddos.subtitle"))}</div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <form id="antiddos-form" class="waf-form">
            <div class="waf-subcard waf-stack waf-antiddos-frame">
              <div class="waf-list-title">${escapeHtml(ctx.t("antiddos.scope.l4.title"))}</div>
              <div class="muted">${escapeHtml(ctx.t("antiddos.scope.l4.desc"))}</div>
              <div class="waf-form-grid waf-antiddos-grid">
                <label class="waf-checkbox"><input id="antiddos-use-l4" type="checkbox"${settings.use_l4_guard ? " checked" : ""}> ${escapeHtml(ctx.t("antiddos.field.useL4"))}</label>
                <div class="waf-field">
                  <label for="antiddos-chain-mode">${escapeHtml(ctx.t("antiddos.field.chainMode"))}</label>
                  <select id="antiddos-chain-mode">
                    ${["auto", "docker-user", "input"].map((mode) => `<option value="${mode}"${settings.chain_mode === mode ? " selected" : ""}>${mode}</option>`).join("")}
                  </select>
                </div>
                <div class="waf-field">
                  <label for="antiddos-target">${escapeHtml(ctx.t("antiddos.field.target"))}</label>
                  <select id="antiddos-target">
                    ${["DROP", "REJECT"].map((mode) => `<option value="${mode}"${settings.target === mode ? " selected" : ""}>${mode}</option>`).join("")}
                  </select>
                </div>
                <div class="waf-field">
                  <label for="antiddos-conn-limit">${escapeHtml(ctx.t("antiddos.field.connLimit"))}</label>
                  <input id="antiddos-conn-limit" type="number" min="1" value="${escapeHtml(String(settings.conn_limit || 200))}">
                </div>
                <div class="waf-field">
                  <label for="antiddos-rate-ps">${escapeHtml(ctx.t("antiddos.field.ratePerSecond"))}</label>
                  <input id="antiddos-rate-ps" type="number" min="1" value="${escapeHtml(String(settings.rate_per_second || 100))}">
                </div>
                <div class="waf-field">
                  <label for="antiddos-rate-burst">${escapeHtml(ctx.t("antiddos.field.rateBurst"))}</label>
                  <input id="antiddos-rate-burst" type="number" min="1" value="${escapeHtml(String(settings.rate_burst || 200))}">
                </div>
                <div class="waf-field">
                  <label for="antiddos-ports">${escapeHtml(ctx.t("antiddos.field.ports"))}</label>
                  <input id="antiddos-ports" value="${escapeHtml(portsToInput(settings.ports))}" placeholder="80, 443">
                </div>
                <div class="waf-field">
                  <label for="antiddos-destination-ip">${escapeHtml(ctx.t("antiddos.field.destinationIp"))}</label>
                  <input id="antiddos-destination-ip" value="${escapeHtml(settings.destination_ip || "")}" placeholder="172.18.0.4">
                </div>
              </div>
            </div>
            <div class="waf-subcard waf-stack waf-antiddos-frame">
              <div class="waf-list-title">${escapeHtml(ctx.t("antiddos.scope.l7.title"))}</div>
              <div class="muted">${escapeHtml(ctx.t("antiddos.scope.l7.desc"))}</div>
              <div class="waf-form-grid waf-antiddos-grid">
                <label class="waf-checkbox"><input id="antiddos-enforce-l7" type="checkbox"${settings.enforce_l7_rate_limit ? " checked" : ""}> ${escapeHtml(ctx.t("antiddos.field.enforceL7"))}</label>
                <div class="waf-field">
                  <label for="antiddos-l7-rps">${escapeHtml(ctx.t("antiddos.field.l7Rps"))}</label>
                  <input id="antiddos-l7-rps" type="number" min="1" value="${escapeHtml(String(settings.l7_requests_per_second || 100))}">
                </div>
                <div class="waf-field">
                  <label for="antiddos-l7-burst">${escapeHtml(ctx.t("antiddos.field.l7Burst"))}</label>
                  <input id="antiddos-l7-burst" type="number" min="0" value="${escapeHtml(String(settings.l7_burst || 200))}">
                </div>
                <div class="waf-field">
                  <label for="antiddos-l7-status">${escapeHtml(ctx.t("antiddos.field.l7StatusCode"))}</label>
                  <input id="antiddos-l7-status" type="number" min="100" max="599" value="${escapeHtml(String(settings.l7_status_code || 429))}">
                </div>
              </div>
            </div>
            <div class="waf-subcard waf-stack waf-antiddos-frame">
              <div class="waf-list-title">${escapeHtml(ctx.t("antiddos.scope.model.title"))}</div>
              <div class="waf-antiddos-title-row">
                <div class="muted">${escapeHtml(ctx.t("antiddos.scope.model.desc"))}</div>
                <button class="waf-help-icon-btn" type="button" id="antiddos-model-help-btn" title="${escapeHtml(ctx.t("antiddos.help.open"))}" aria-label="${escapeHtml(ctx.t("antiddos.help.open"))}">?</button>
              </div>
              <div class="waf-form-grid waf-antiddos-grid">
                <label class="waf-checkbox"><input id="model-enabled" type="checkbox"${settings.model_enabled !== false ? " checked" : ""}> ${escapeHtml(ctx.t("antiddos.model.enabled"))}</label>
                <div class="waf-field">
                  <label for="model-poll">${escapeHtml(ctx.t("antiddos.model.poll"))}</label>
                  <input id="model-poll" type="number" min="1" value="${escapeHtml(String(settings.model_poll_interval_seconds || 2))}">
                </div>
                <div class="waf-field">
                  <label for="model-decay">${escapeHtml(ctx.t("antiddos.model.decay"))}</label>
                  <input id="model-decay" type="number" min="0.001" step="0.001" value="${escapeHtml(String(settings.model_decay_lambda || 0.08))}">
                </div>
                <div class="waf-field">
                  <label for="model-threshold-throttle">${escapeHtml(ctx.t("antiddos.model.thresholdThrottle"))}</label>
                  <input id="model-threshold-throttle" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_throttle_threshold || 2.5))}">
                </div>
                <div class="waf-field">
                  <label for="model-threshold-drop">${escapeHtml(ctx.t("antiddos.model.thresholdDrop"))}</label>
                  <input id="model-threshold-drop" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_drop_threshold || 6.0))}">
                </div>
                <div class="waf-field">
                  <label for="model-hold">${escapeHtml(ctx.t("antiddos.model.hold"))}</label>
                  <input id="model-hold" type="number" min="1" value="${escapeHtml(String(settings.model_hold_seconds || 60))}">
                </div>
                <div class="waf-field">
                  <label for="model-throttle-rps">${escapeHtml(ctx.t("antiddos.model.throttleRps"))}</label>
                  <input id="model-throttle-rps" type="number" min="1" value="${escapeHtml(String(settings.model_throttle_rate_per_second || 3))}">
                </div>
                <div class="waf-field">
                  <label for="model-throttle-burst">${escapeHtml(ctx.t("antiddos.model.throttleBurst"))}</label>
                  <input id="model-throttle-burst" type="number" min="1" value="${escapeHtml(String(settings.model_throttle_burst || 6))}">
                </div>
                <div class="waf-field">
                  <label for="model-throttle-target">${escapeHtml(ctx.t("antiddos.model.throttleTarget"))}</label>
                  <select id="model-throttle-target">
                    ${["REJECT", "DROP"].map((mode) => `<option value="${mode}"${String(settings.model_throttle_target || "REJECT") === mode ? " selected" : ""}>${mode}</option>`).join("")}
                  </select>
                </div>
                <div class="waf-field">
                  <label for="model-weight-429">${escapeHtml(ctx.t("antiddos.model.weight429"))}</label>
                  <input id="model-weight-429" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_weight_429 || 1.0))}">
                </div>
                <div class="waf-field">
                  <label for="model-weight-403">${escapeHtml(ctx.t("antiddos.model.weight403"))}</label>
                  <input id="model-weight-403" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_weight_403 || 1.8))}">
                </div>
                <div class="waf-field">
                  <label for="model-weight-444">${escapeHtml(ctx.t("antiddos.model.weight444"))}</label>
                  <input id="model-weight-444" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_weight_444 || 2.2))}">
                </div>
                <div class="waf-field">
                  <label for="model-emergency-rps">${escapeHtml(ctx.t("antiddos.model.emergencyRps"))}</label>
                  <input id="model-emergency-rps" type="number" min="1" value="${escapeHtml(String(settings.model_emergency_rps || 180))}">
                </div>
                <div class="waf-field">
                  <label for="model-emergency-unique">${escapeHtml(ctx.t("antiddos.model.emergencyUnique"))}</label>
                  <input id="model-emergency-unique" type="number" min="1" value="${escapeHtml(String(settings.model_emergency_unique_ips || 40))}">
                </div>
                <div class="waf-field">
                  <label for="model-emergency-per-ip">${escapeHtml(ctx.t("antiddos.model.emergencyPerIp"))}</label>
                  <input id="model-emergency-per-ip" type="number" min="1" value="${escapeHtml(String(settings.model_emergency_per_ip_rps || 60))}">
                </div>
                <div class="waf-field">
                  <label for="model-weight-emergency-botnet">${escapeHtml(ctx.t("antiddos.model.weightEmergencyBotnet"))}</label>
                  <input id="model-weight-emergency-botnet" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_weight_emergency_botnet || 6.0))}">
                </div>
                <div class="waf-field">
                  <label for="model-weight-emergency-single">${escapeHtml(ctx.t("antiddos.model.weightEmergencySingle"))}</label>
                  <input id="model-weight-emergency-single" type="number" min="0.1" step="0.1" value="${escapeHtml(String(settings.model_weight_emergency_single || 4.0))}">
                </div>
              </div>
            </div>
            <div class="waf-actions">
              <button class="btn primary" type="submit">${escapeHtml(ctx.t("common.save"))}</button>
            </div>
            </form>
          </div>
        </section>

      <section class="waf-card waf-antiddos-live-feed-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("antiddos.model.logs.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("antiddos.model.logs.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" id="antiddos-model-logs-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
        <div class="waf-card-body">
          <div id="antiddos-model-logs"><div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div></div>
        </div>
      </section>
      <div class="waf-modal waf-hidden" id="antiddos-model-log-detail-modal" role="dialog" aria-modal="true" aria-labelledby="antiddos-model-log-detail-title" tabindex="-1">
        <button class="waf-modal-overlay" type="button" data-log-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
        <div class="waf-modal-card">
          <div class="waf-card-head">
            <div>
              <h3 id="antiddos-model-log-detail-title">${escapeHtml(ctx.t("antiddos.model.logs.detail.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("antiddos.model.logs.detail.subtitle"))}</div>
            </div>
            <button class="btn ghost btn-sm" type="button" data-log-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
          </div>
          <div class="waf-card-body" id="antiddos-model-log-detail-content">
            <div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>
          </div>
        </div>
      </div>
      <div class="waf-modal waf-hidden" id="antiddos-model-help-modal" role="dialog" aria-modal="true" aria-labelledby="antiddos-model-help-title" tabindex="-1">
        <button class="waf-modal-overlay" type="button" data-modal-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
        <div class="waf-modal-card">
          <div class="waf-card-head">
            <div>
              <h3 id="antiddos-model-help-title">${escapeHtml(ctx.t("antiddos.help.title"))}</h3>
              <div class="muted">${escapeHtml(ctx.t("antiddos.help.subtitle"))}</div>
            </div>
            <button class="btn ghost btn-sm" type="button" data-modal-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
          </div>
          <div class="waf-card-body">
            <div class="waf-table-wrap">
              <table class="waf-table">
                <thead>
                  <tr>
                    <th>${escapeHtml(ctx.t("antiddos.help.col.parameter"))}</th>
                    <th>${escapeHtml(ctx.t("antiddos.help.col.default"))}</th>
                    <th>${escapeHtml(ctx.t("antiddos.help.col.tuning"))}</th>
                  </tr>
                </thead>
                <tbody>
                  ${helpRows.map((item) => `
                    <tr>
                      <td>${escapeHtml(ctx.t(item.labelKey))}</td>
                      <td>${escapeHtml(item.defaultValue)}</td>
                      <td>${escapeHtml(ctx.t(item.hintKey))}</td>
                    </tr>
                  `).join("")}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    `;

    container.querySelector("#antiddos-form")?.addEventListener("submit", async (event) => {
      event.preventDefault();
      const payload = {
        use_l4_guard: container.querySelector("#antiddos-use-l4").checked,
        chain_mode: container.querySelector("#antiddos-chain-mode").value,
        target: container.querySelector("#antiddos-target").value,
        conn_limit: Number(container.querySelector("#antiddos-conn-limit").value || "200"),
        rate_per_second: Number(container.querySelector("#antiddos-rate-ps").value || "100"),
        rate_burst: Number(container.querySelector("#antiddos-rate-burst").value || "200"),
        ports: parsePorts(container.querySelector("#antiddos-ports").value),
        destination_ip: container.querySelector("#antiddos-destination-ip").value.trim(),
        enforce_l7_rate_limit: container.querySelector("#antiddos-enforce-l7").checked,
        l7_requests_per_second: Number(container.querySelector("#antiddos-l7-rps").value || "100"),
        l7_burst: Number(container.querySelector("#antiddos-l7-burst").value || "200"),
        l7_status_code: Number(container.querySelector("#antiddos-l7-status").value || "429"),
        model_enabled: container.querySelector("#model-enabled").checked,
        model_poll_interval_seconds: Number(container.querySelector("#model-poll").value || "2"),
        model_decay_lambda: Number(container.querySelector("#model-decay").value || "0.08"),
        model_throttle_threshold: Number(container.querySelector("#model-threshold-throttle").value || "2.5"),
        model_drop_threshold: Number(container.querySelector("#model-threshold-drop").value || "6.0"),
        model_hold_seconds: Number(container.querySelector("#model-hold").value || "60"),
        model_throttle_rate_per_second: Number(container.querySelector("#model-throttle-rps").value || "3"),
        model_throttle_burst: Number(container.querySelector("#model-throttle-burst").value || "6"),
        model_throttle_target: container.querySelector("#model-throttle-target").value,
        model_weight_429: Number(container.querySelector("#model-weight-429").value || "1.0"),
        model_weight_403: Number(container.querySelector("#model-weight-403").value || "1.8"),
        model_weight_444: Number(container.querySelector("#model-weight-444").value || "2.2"),
        model_emergency_rps: Number(container.querySelector("#model-emergency-rps").value || "180"),
        model_emergency_unique_ips: Number(container.querySelector("#model-emergency-unique").value || "40"),
        model_emergency_per_ip_rps: Number(container.querySelector("#model-emergency-per-ip").value || "60"),
        model_weight_emergency_botnet: Number(container.querySelector("#model-weight-emergency-botnet").value || "6.0"),
        model_weight_emergency_single: Number(container.querySelector("#model-weight-emergency-single").value || "4.0")
      };
      await upsertWithPostFallback(payload);
      ctx.notify(ctx.t("antiddos.toast.saved"));
      const updated = await ctx.api.get("/api/anti-ddos/settings");
      render(updated || payload);
      await renderLogs();
    });

    const modalNode = container.querySelector("#antiddos-model-help-modal");
    const detailModalNode = container.querySelector("#antiddos-model-log-detail-modal");
    const openModal = () => {
      if (!modalNode) {
        return;
      }
      modalNode.classList.remove("waf-hidden");
      modalNode.focus();
    };
    const closeModal = () => modalNode?.classList.add("waf-hidden");
    container.querySelector("#antiddos-model-help-btn")?.addEventListener("click", openModal);
    modalNode?.querySelectorAll("[data-modal-close='true']").forEach((node) => {
      node.addEventListener("click", closeModal);
    });
    modalNode?.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        closeModal();
      }
    });
    const closeDetailModal = () => detailModalNode?.classList.add("waf-hidden");
    detailModalNode?.querySelectorAll("[data-log-detail-close='true']").forEach((node) => {
      node.addEventListener("click", closeDetailModal);
    });
    detailModalNode?.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        closeDetailModal();
      }
    });

    container.querySelector("#antiddos-model-logs-refresh")?.addEventListener("click", renderLogs);
    renderLogs();
    logsTimer = window.setInterval(() => {
      if (!container.isConnected) {
        clearInterval(logsTimer);
        logsTimer = null;
        return;
      }
      renderLogs();
    }, 3000);
  };

  const settings = await ctx.api.get("/api/anti-ddos/settings");
  render(settings || {});
}
