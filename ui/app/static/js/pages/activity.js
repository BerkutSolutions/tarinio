import { escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
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

function inferCategory(action) {
  if (action.startsWith("auth.")) {
    return "auth";
  }
  if (action.startsWith("revision.")) {
    return "rollout";
  }
  return "config";
}

function detailsText(details, ctx) {
  if (!details || (typeof details === "object" && !Object.keys(details).length)) {
    return ctx.t("activity.details.empty");
  }
  return typeof details === "string" ? details : JSON.stringify(details, null, 2);
}

function actionKey(action) {
  const normalized = String(action || "").trim().toLowerCase().replace(/[^a-z0-9_.-]+/g, "_");
  return normalized ? `activity.action.${normalized}` : "activity.action.unknown";
}

function summaryKey(summary) {
  const normalized = String(summary || "").trim().toLowerCase().replace(/[^a-z0-9_.-]+/g, "_");
  return normalized ? `activity.summary.${normalized}` : "activity.summary.unknown";
}

function translateAction(action, ctx) {
  const key = actionKey(action);
  const value = ctx.t(key);
  return value === key ? (action || ctx.t("activity.action.unknown")) : value;
}

function translateSummary(summary, ctx) {
  const key = summaryKey(summary);
  const value = ctx.t(key);
  if (value !== key) {
    return value;
  }
  return summary || ctx.t("activity.summary.unknown");
}

function applyPreset(container, preset) {
  const now = new Date();
  let from = "";
  if (preset === "hour") {
    from = new Date(now.getTime() - 60 * 60 * 1000);
  } else if (preset === "day") {
    from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  }
  container.querySelector("#audit-from").value = from ? from.toISOString().slice(0, 16) : "";
  container.querySelector("#audit-to").value = preset ? now.toISOString().slice(0, 16) : "";
}

export async function renderActivity(container, ctx) {
  container.innerHTML = `
    <section class="waf-card no-page-card-head">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("activity.title"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("activity.subtitle"))}</div>
        </div>
      </div>
      <div class="waf-card-body waf-stack">
        <form id="audit-filters" class="waf-filters">
          <div class="waf-actions">
            <button class="btn ghost btn-sm" type="button" data-preset="">${escapeHtml(ctx.t("common.allTime"))}</button>
            <button class="btn ghost btn-sm" type="button" data-preset="hour">${escapeHtml(ctx.t("common.lastHour"))}</button>
            <button class="btn ghost btn-sm" type="button" data-preset="day">${escapeHtml(ctx.t("common.last24Hours"))}</button>
          </div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="audit-category">${escapeHtml(ctx.t("activity.filter.actionType"))}</label>
              <select id="audit-category" class="select">
                <option value="">${escapeHtml(ctx.t("common.all"))}</option>
                <option value="auth">${escapeHtml(ctx.t("activity.category.auth"))}</option>
                <option value="config">${escapeHtml(ctx.t("activity.category.config"))}</option>
                <option value="rollout">${escapeHtml(ctx.t("activity.category.rollout"))}</option>
              </select>
            </div>
            <div class="waf-field">
              <label for="audit-actor">${escapeHtml(ctx.t("activity.filter.actor"))}</label>
              <input id="audit-actor" placeholder="${escapeHtml(ctx.t("activity.filter.actorPlaceholder"))}">
            </div>
            <div class="waf-field">
              <label for="audit-site">${escapeHtml(ctx.t("activity.filter.site"))}</label>
              <input id="audit-site" placeholder="${escapeHtml(ctx.t("activity.filter.sitePlaceholder"))}">
            </div>
            <div class="waf-field">
              <label for="audit-status">${escapeHtml(ctx.t("activity.filter.status"))}</label>
              <select id="audit-status" class="select">
                <option value="">${escapeHtml(ctx.t("common.all"))}</option>
                <option value="succeeded">${escapeHtml(ctx.t("status.succeeded"))}</option>
                <option value="failed">${escapeHtml(ctx.t("status.failed"))}</option>
              </select>
            </div>
            <div class="waf-field">
              <label for="audit-from">${escapeHtml(ctx.t("activity.filter.from"))}</label>
              <input id="audit-from" type="datetime-local">
            </div>
            <div class="waf-field">
              <label for="audit-to">${escapeHtml(ctx.t("activity.filter.to"))}</label>
              <input id="audit-to" type="datetime-local">
            </div>
            <div class="waf-field">
              <label for="audit-limit">${escapeHtml(ctx.t("activity.filter.pageSize"))}</label>
              <select id="audit-limit" class="select select-compact">
                <option value="25">25</option>
                <option value="50" selected>50</option>
                <option value="100">100</option>
              </select>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("activity.filter.apply"))}</button>
            <button class="btn ghost btn-sm" type="button" id="audit-reset">${escapeHtml(ctx.t("common.reset"))}</button>
          </div>
        </form>
        <div id="audit-results" class="waf-list"></div>
        <div class="waf-pager">
          <div id="audit-page-info" class="muted">-</div>
          <div class="waf-actions">
            <button class="btn ghost btn-sm" type="button" id="audit-prev">${escapeHtml(ctx.t("common.previous"))}</button>
            <button class="btn ghost btn-sm" type="button" id="audit-next">${escapeHtml(ctx.t("common.next"))}</button>
          </div>
        </div>
      </div>
    </section>
  `;

  const state = { offset: 0, total: 0 };
  const siteNamesByID = new Map();
  const siteIDsByHost = new Map();
  const form = container.querySelector("#audit-filters");
  const results = container.querySelector("#audit-results");

  const readQuery = () => {
    const limit = Number(container.querySelector("#audit-limit").value || "50");
    return {
      actor_user_id: container.querySelector("#audit-actor").value.trim(),
      site_id: container.querySelector("#audit-site").value.trim(),
      status: container.querySelector("#audit-status").value,
      from: container.querySelector("#audit-from").value ? new Date(container.querySelector("#audit-from").value).toISOString() : "",
      to: container.querySelector("#audit-to").value ? new Date(container.querySelector("#audit-to").value).toISOString() : "",
      limit,
      offset: state.offset,
      category: container.querySelector("#audit-category").value
    };
  };

  const renderItems = (items) => {
    results.innerHTML = items.length
      ? items.map((item) => `
          <div class="waf-list-item">
            <div class="waf-list-head">
              <div class="waf-list-title">${escapeHtml(translateAction(item.action, ctx))}</div>
              ${statusBadge(item.status)}
            </div>
            <div class="waf-inline">
              <span class="badge badge-neutral">${escapeHtml(ctx.t(`activity.category.${inferCategory(item.action || "")}`))}</span>
              <span>${escapeHtml(item.actor_user_id || ctx.t("common.system"))}</span>
              <span>${escapeHtml(siteNamesByID.get(String(item.site_id || "")) || item.site_id || "-")}</span>
            </div>
            <div>${escapeHtml(translateSummary(item.summary, ctx))}</div>
            <div class="waf-code">${escapeHtml(detailsText(item.details_json, ctx))}</div>
            <div class="muted">${formatDate(item.occurred_at)}</div>
          </div>
        `).join("")
      : `<div class="waf-empty">${escapeHtml(ctx.t("app.noData"))}</div>`;
  };

  const syncPager = (limit) => {
    container.querySelector("#audit-page-info").textContent = ctx.t("activity.pager", { offset: state.offset, total: state.total });
    container.querySelector("#audit-prev").disabled = state.offset <= 0;
    container.querySelector("#audit-next").disabled = state.offset + limit >= state.total;
  };

  const load = async () => {
    const query = readQuery();
    const params = new URLSearchParams();
    if (query.actor_user_id) params.set("actor_user_id", query.actor_user_id);
    if (query.site_id) {
      const normalizedSite = String(query.site_id || "").trim().toLowerCase();
      params.set("site_id", siteIDsByHost.get(normalizedSite) || query.site_id);
    }
    if (query.status) params.set("status", query.status);
    if (query.from) params.set("from", query.from);
    if (query.to) params.set("to", query.to);
    params.set("limit", String(query.limit));
    params.set("offset", String(query.offset));

    setLoading(results, ctx.t("activity.loading"));
    try {
      const result = await ctx.api.get(`/api/audit?${params.toString()}`);
      const items = (result.items || []).filter((item) => !query.category || inferCategory(item.action || "") === query.category);
      state.total = Number(result.total || 0);
      renderItems(items);
      syncPager(query.limit);
    } catch (error) {
      state.total = 0;
      setError(results, error.message || ctx.t("activity.error"));
      syncPager(query.limit);
    }
  };

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    state.offset = 0;
    await load();
  });

  ["#audit-category", "#audit-actor", "#audit-site", "#audit-status", "#audit-limit"].forEach((selector) => {
    container.querySelector(selector).addEventListener("change", async () => {
      state.offset = 0;
      await load();
    });
  });

  ["#audit-from", "#audit-to"].forEach((selector) => {
    container.querySelector(selector).addEventListener("change", async () => {
      state.offset = 0;
      await load();
    });
  });

  container.querySelectorAll("[data-preset]").forEach((button) => {
    button.addEventListener("click", async () => {
      applyPreset(container, button.dataset.preset);
      state.offset = 0;
      await load();
    });
  });

  container.querySelector("#audit-reset").addEventListener("click", async () => {
    form.reset();
    state.offset = 0;
    state.total = 0;
    await load();
  });

  container.querySelector("#audit-prev").addEventListener("click", async () => {
    const limit = Number(container.querySelector("#audit-limit").value || "50");
    state.offset = Math.max(0, state.offset - limit);
    await load();
  });

  container.querySelector("#audit-next").addEventListener("click", async () => {
    const limit = Number(container.querySelector("#audit-limit").value || "50");
    state.offset += limit;
    await load();
  });

  try {
    const [primarySites, secondarySites] = await Promise.all([
      ctx.api.get("/api/sites"),
      tryGetJSON("/api-app/sites")
    ]);
    const mergedSites = [...normalizeList(primarySites), ...unwrapList(secondarySites, ["sites"])];
    for (const site of mergedSites) {
      const id = String(site?.id || "").trim();
      const host = String(site?.primary_host || "").trim();
      if (!id) {
        continue;
      }
      if (host) {
        siteNamesByID.set(id, host);
        siteIDsByHost.set(host.toLowerCase(), id);
      } else {
        siteNamesByID.set(id, id);
      }
    }
  } catch (error) {
    // Keep fallback rendering by site ID.
  }

  await load();
}

