import { escapeHtml } from "../ui.js";
import { formatDateTimeInZone } from "../preferences.js";
import { readSecretFieldValue } from "./settings.logging-form.js";

function formatIndexUpdatedAt(value) {
  return formatDateTimeInZone(value);
}

export function renderStorageIndexes({
  ctx,
  storageIndexesNode,
  payload,
  storageIndexesStream,
  storageIndexesLimit,
  storageIndexesOffset,
  setStorageIndexesOffset,
  setStorageIndexesStream,
  setAlert,
  renderRuntime,
  api
}) {
  if (!storageIndexesNode) {
    return;
  }
  const items = Array.isArray(payload?.items) ? payload.items : [];
  const total = Number(payload?.total || items.length || 0);
  const limit = Number(payload?.limit || storageIndexesLimit || 10);
  const offset = Number(payload?.offset || storageIndexesOffset || 0);
  const stream = String(payload?.stream || storageIndexesStream || "requests");
  const currentPage = Math.max(1, Math.floor(offset / limit) + 1);
  const totalPages = Math.max(1, Math.ceil(total / Math.max(1, limit)));
  const pages = [];
  for (let page = 1; page <= Math.min(10, totalPages); page += 1) {
    pages.push(`<button type="button" class="btn ghost btn-sm${page === currentPage ? " active" : ""}" data-storage-index-page="${page}">${page}</button>`);
  }
  if (totalPages > 10) {
    pages.push(`<span class="muted">...</span>`);
    pages.push(`<button type="button" class="btn ghost btn-sm${currentPage === totalPages ? " active" : ""}" data-storage-index-page="${totalPages}">${totalPages}</button>`);
  }

  storageIndexesNode.innerHTML = `
      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("settings.storage.indexes.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("settings.storage.indexes.subtitle"))}</div>
          </div>
          <div class="tabs browser-tabs" role="tablist" aria-label="${escapeHtml(ctx.t("settings.storage.indexes.title"))}">
            <button type="button" class="tab-btn${stream === "requests" ? " active" : ""}" data-storage-index-stream="requests">${escapeHtml(ctx.t("app.requests"))}</button>
            <button type="button" class="tab-btn${stream === "events" ? " active" : ""}" data-storage-index-stream="events">${escapeHtml(ctx.t("app.events"))}</button>
            <button type="button" class="tab-btn${stream === "activity" ? " active" : ""}" data-storage-index-stream="activity">${escapeHtml(ctx.t("app.activity"))}</button>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="waf-empty">${escapeHtml(ctx.t("settings.storage.indexes.total"))}: ${total}</div>
          ${payload?.error ? `<div class="alert alert-danger">${escapeHtml(String(payload.error))}</div>` : ""}
          <div class="waf-table-wrap">
            <table class="waf-table">
              <thead>
                <tr>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.date"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.file"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.lines"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.size"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.updated"))}</th>
                  <th>${escapeHtml(ctx.t("settings.storage.indexes.col.actions"))}</th>
                </tr>
              </thead>
              <tbody>
                ${items.length
                  ? items.map((item) => `
                    <tr>
                      <td>${escapeHtml(String(item?.date || "-"))}</td>
                      <td>${escapeHtml(String(item?.file_name || "-"))}</td>
                      <td>${escapeHtml(String(item?.lines ?? 0))}</td>
                      <td>${escapeHtml(String(item?.size_bytes ?? 0))}</td>
                      <td>${escapeHtml(formatIndexUpdatedAt(item?.updated_at || "-"))}</td>
                      <td>
                        <button
                          class="btn ghost btn-sm"
                          type="button"
                          data-storage-index-delete="${escapeHtml(String(item?.date || ""))}"
                        >${escapeHtml(ctx.t("common.delete"))}</button>
                      </td>
                    </tr>
                  `).join("")
                  : `<tr><td colspan="6"><div class="waf-empty">${escapeHtml(ctx.t("settings.storage.indexes.empty"))}</div></td></tr>`}
              </tbody>
            </table>
          </div>
          <div class="waf-pager">
            <div class="muted">${escapeHtml(ctx.t("settings.storage.indexes.page"))}: ${currentPage}/${totalPages}</div>
            <div class="waf-actions">${pages.join("")}</div>
          </div>
        </div>
      </section>
    `;
  storageIndexesNode.querySelectorAll("[data-storage-index-page]").forEach((button) => {
    button.addEventListener("click", async () => {
      const page = Number.parseInt(String(button.dataset.storageIndexPage || "1"), 10);
      if (!Number.isFinite(page) || page < 1) {
        return;
      }
      setStorageIndexesOffset((page - 1) * storageIndexesLimit);
      await renderRuntime();
    });
  });
  storageIndexesNode.querySelectorAll("[data-storage-index-stream]").forEach((button) => {
    button.addEventListener("click", async () => {
      const nextStream = String(button.dataset.storageIndexStream || "requests");
      if (!nextStream || nextStream === storageIndexesStream) {
        return;
      }
      setStorageIndexesStream(nextStream);
      setStorageIndexesOffset(0);
      await renderRuntime();
    });
  });
  storageIndexesNode.querySelectorAll("[data-storage-index-delete]").forEach((button) => {
    button.addEventListener("click", async () => {
      const day = String(button.dataset.storageIndexDelete || "").trim();
      if (!day) {
        return;
      }
      if (!window.confirm(ctx.t("settings.storage.indexes.deleteConfirm", { date: day }))) {
        return;
      }
      setAlert("");
      try {
        await api.delete(`/api/settings/runtime/storage-indexes?stream=${encodeURIComponent(storageIndexesStream)}&date=${encodeURIComponent(day)}`);
        setAlert(ctx.t("settings.saved"), true);
        await renderRuntime();
      } catch (error) {
        setAlert(error?.message || ctx.t("common.error"));
      }
    });
  });
}

export function renderLoggingStatusText(ctx, logging, summary) {
  const hotBackend = String(summary?.hot_backend || logging?.hot?.backend || "file");
  const coldBackend = String(summary?.cold_backend || logging?.cold?.backend || "file");
  const secretProvider = String(summary?.secret_provider || logging?.secret_provider || "vault");
  const hotRetention = Number(logging?.retention?.hot_days || 30);
  const coldRetention = Number(logging?.retention?.cold_days || 730);
  if (hotBackend === "opensearch" && coldBackend === "clickhouse") {
    return ctx.t("settings.logging.status.dual", {
      hotDays: hotRetention,
      coldDays: coldRetention,
      secretProvider,
    });
  }
  if (hotBackend === "opensearch" && coldBackend === "opensearch") {
    return ctx.t("settings.logging.status.opensearch_full", {
      endpoint: String(logging?.opensearch?.endpoint || "-"),
      hotDays: hotRetention,
      coldDays: coldRetention,
    });
  }
  if (hotBackend === "opensearch" || coldBackend === "opensearch") {
    return ctx.t("settings.logging.status.opensearch", {
      endpoint: String(logging?.opensearch?.endpoint || "-"),
      retention: hotBackend === "opensearch" ? hotRetention : coldRetention,
    });
  }
  if (coldBackend === "clickhouse") {
    return ctx.t("settings.logging.status.clickhouse", {
      endpoint: String(logging?.clickhouse?.endpoint || "-"),
      database: String(logging?.clickhouse?.database || "waf_logs"),
      table: String(logging?.clickhouse?.table || "request_logs"),
    });
  }
  return ctx.t("settings.logging.status.file");
}

export function buildLoggingPayload(nodes) {
  const {
    loggingHotBackend,
    loggingColdBackend,
    storageHotIndexDays,
    loggingHotRetention,
    storageColdIndexDays,
    loggingColdRetention,
    loggingRouteRequestsHot,
    loggingRouteRequestsCold,
    loggingRouteEventsHot,
    loggingRouteEventsCold,
    loggingRouteActivityHot,
    loggingRouteActivityCold,
    loggingRouteFallback,
    loggingSecretProvider,
    loggingVaultEnabled,
    loggingVaultAddress,
    loggingVaultMount,
    loggingVaultPathPrefix,
    loggingVaultToken,
    loggingVaultTLSSkipVerify,
    loggingOpenSearchEndpoint,
    loggingOpenSearchPrefix,
    loggingOpenSearchUsername,
    loggingOpenSearchPassword,
    loggingOpenSearchAPIKey,
    loggingEndpoint,
    loggingDatabase,
    loggingTable,
    loggingUsername,
    loggingPassword,
    loggingMigrationEnabled
  } = nodes;
  return {
    logging: {
      backend: String(loggingHotBackend?.value || "opensearch") === "opensearch"
        ? "opensearch"
        : (String(loggingColdBackend?.value || "opensearch") === "clickhouse" ? "clickhouse" : (String(loggingColdBackend?.value || "opensearch") === "opensearch" ? "opensearch" : "file")),
      hot: {
        backend: String(loggingHotBackend?.value || "opensearch"),
      },
      cold: {
        backend: String(loggingColdBackend?.value || "opensearch"),
      },
      retention: {
        hot_days: Number(storageHotIndexDays?.value || loggingHotRetention?.value || "30"),
        cold_days: Number(storageColdIndexDays?.value || loggingColdRetention?.value || "730"),
      },
      routing: {
        write_requests_to_hot: !!loggingRouteRequestsHot?.checked,
        write_requests_to_cold: !!loggingRouteRequestsCold?.checked,
        write_events_to_hot: !!loggingRouteEventsHot?.checked,
        write_events_to_cold: !!loggingRouteEventsCold?.checked,
        write_activity_to_hot: !!loggingRouteActivityHot?.checked,
        write_activity_to_cold: !!loggingRouteActivityCold?.checked,
        keep_local_fallback: !!loggingRouteFallback?.checked,
      },
      secret_provider: String(loggingSecretProvider?.value || "vault"),
      vault: {
        enabled: !!loggingVaultEnabled?.checked,
        address: String(loggingVaultAddress?.value || "").trim(),
        mount: String(loggingVaultMount?.value || "secret").trim(),
        path_prefix: String(loggingVaultPathPrefix?.value || "tarinio").trim(),
        token: readSecretFieldValue(loggingVaultToken),
        tls_skip_verify: !!loggingVaultTLSSkipVerify?.checked,
      },
      opensearch: {
        endpoint: String(loggingOpenSearchEndpoint?.value || "").trim(),
        index_prefix: String(loggingOpenSearchPrefix?.value || "waf-hot").trim(),
        username: String(loggingOpenSearchUsername?.value || "").trim(),
        password: readSecretFieldValue(loggingOpenSearchPassword),
        api_key: readSecretFieldValue(loggingOpenSearchAPIKey),
      },
      clickhouse: {
        endpoint: String(loggingEndpoint?.value || "").trim(),
        database: String(loggingDatabase?.value || "waf_logs").trim(),
        table: String(loggingTable?.value || "request_logs").trim(),
        username: String(loggingUsername?.value || "").trim(),
        password: readSecretFieldValue(loggingPassword),
        migration_enabled: !!loggingMigrationEnabled?.checked,
      },
    },
  };
}
