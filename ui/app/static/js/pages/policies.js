import { csvToList, escapeHtml, formatDate, listToMultiline, parseJSONLines, statusBadge } from "../ui.js";

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

function renderPolicyTable(items, columns) {
  if (!items.length) {
    return `<div class="waf-empty">No records yet.</div>`;
  }
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr>${columns.map((column) => `<th>${escapeHtml(column.label)}</th>`).join("")}</tr></thead>
        <tbody>
          ${items.map((item) => `
            <tr>${columns.map((column) => `<td>${column.render(item)}</td>`).join("")}</tr>
          `).join("")}
        </tbody>
      </table>
    </div>
  `;
}

export async function renderPolicies(container, ctx) {
  container.innerHTML = `
    <div class="waf-grid two">
      <section class="waf-card">
        <div class="waf-card-head"><div><h3>WAF policy</h3><div class="muted">Detection/blocking mode, CRS and structured overrides.</div></div></div>
        <div class="waf-card-body waf-stack">
          <form id="waf-policy-form" class="waf-form">
            <div class="waf-form-grid">
              <div class="waf-field"><label for="waf-id">ID</label><input id="waf-id" required></div>
              <div class="waf-field"><label for="waf-site-id">Site ID</label><input id="waf-site-id" required></div>
              <div class="waf-field"><label for="waf-mode">Mode</label><select id="waf-mode"><option value="detection">detection</option><option value="prevention">prevention</option></select></div>
              <label class="waf-checkbox"><input id="waf-enabled" type="checkbox" checked> enabled</label>
              <label class="waf-checkbox"><input id="waf-crs-enabled" type="checkbox" checked> CRS enabled</label>
              <div class="waf-field full"><label for="waf-custom-includes">Custom includes</label><textarea id="waf-custom-includes" placeholder="/etc/waf/custom/site-a.conf"></textarea></div>
              <div class="waf-field full"><label for="waf-rule-overrides">Rule overrides</label><textarea id="waf-rule-overrides" placeholder="942100=true&#10;949110=false"></textarea></div>
            </div>
            <div class="waf-actions">
              <button class="btn primary btn-sm" type="submit">Create / Update</button>
              <button class="btn ghost btn-sm" type="button" id="waf-delete">Delete</button>
            </div>
          </form>
          <div id="waf-policy-list"></div>
        </div>
      </section>
      <section class="waf-card">
        <div class="waf-card-head"><div><h3>Access policy</h3><div class="muted">Allowlist, denylist and manual ban-ready structure.</div></div></div>
        <div class="waf-card-body waf-stack">
          <form id="access-policy-form" class="waf-form">
            <div class="waf-form-grid">
              <div class="waf-field"><label for="access-id">ID</label><input id="access-id" required></div>
              <div class="waf-field"><label for="access-site-id">Site ID</label><input id="access-site-id" required></div>
              <label class="waf-checkbox"><input id="access-enabled" type="checkbox" checked> enabled</label>
              <div class="waf-field full"><label for="access-allowlist">Allowlist</label><textarea id="access-allowlist" placeholder="10.0.0.1&#10;10.0.0.0/24"></textarea></div>
              <div class="waf-field full"><label for="access-denylist">Denylist</label><textarea id="access-denylist" placeholder="203.0.113.10"></textarea></div>
            </div>
            <div class="waf-actions">
              <button class="btn primary btn-sm" type="submit">Create / Update</button>
              <button class="btn ghost btn-sm" type="button" id="access-delete">Delete</button>
            </div>
          </form>
          <div id="access-policy-list"></div>
        </div>
      </section>
    </div>
    <section class="waf-card">
      <div class="waf-card-head"><div><h3>Rate limit policy</h3><div class="muted">Compiler-friendly request rate declaration.</div></div></div>
      <div class="waf-card-body waf-stack">
        <form id="rate-limit-policy-form" class="waf-form">
          <div class="waf-form-grid three">
            <div class="waf-field"><label for="rate-limit-id">ID</label><input id="rate-limit-id" required></div>
            <div class="waf-field"><label for="rate-limit-site-id">Site ID</label><input id="rate-limit-site-id" required></div>
            <label class="waf-checkbox"><input id="rate-limit-enabled" type="checkbox" checked> enabled</label>
            <div class="waf-field"><label for="rate-limit-rps">Requests per second</label><input id="rate-limit-rps" type="number" min="0" value="10"></div>
            <div class="waf-field"><label for="rate-limit-burst">Burst</label><input id="rate-limit-burst" type="number" min="0" value="20"></div>
          </div>
          <div class="waf-actions">
            <button class="btn primary btn-sm" type="submit">Create / Update</button>
            <button class="btn ghost btn-sm" type="button" id="rate-limit-delete">Delete</button>
          </div>
        </form>
        <div id="rate-limit-policy-list"></div>
      </div>
    </section>
  `;

  const load = async () => {
    const [wafPolicies, accessPolicies, rateLimitPolicies, sites, sitesSecondary] = await Promise.all([
      ctx.api.get("/api/waf-policies"),
      ctx.api.get("/api/access-policies"),
      ctx.api.get("/api/rate-limit-policies"),
      ctx.api.get("/api/sites"),
      tryGetJSON("/api-app/sites")
    ]);
    const siteNamesByID = new Map(
      [...normalizeList(sites), ...unwrapList(sitesSecondary, ["sites"])]
        .map((site) => [String(site?.id || ""), String(site?.primary_host || site?.id || "")])
        .filter(([id, host]) => id && host)
    );
    const siteLabel = (siteID) => siteNamesByID.get(String(siteID || "")) || String(siteID || "");

    container.querySelector("#waf-policy-list").innerHTML = renderPolicyTable(wafPolicies || [], [
      { label: "ID", render: (item) => `<span class="waf-code">${escapeHtml(item.id)}</span>` },
      { label: "Service", render: (item) => escapeHtml(siteLabel(item.site_id)) },
      { label: "Enabled", render: (item) => statusBadge(item.enabled ? item.mode : "failed") },
      { label: "CRS", render: (item) => escapeHtml(String(item.crs_enabled)) },
      { label: "Updated", render: (item) => formatDate(item.updated_at) },
    ]);

    container.querySelector("#access-policy-list").innerHTML = renderPolicyTable(accessPolicies || [], [
      { label: "ID", render: (item) => `<span class="waf-code">${escapeHtml(item.id)}</span>` },
      { label: "Service", render: (item) => escapeHtml(siteLabel(item.site_id)) },
      { label: "Allowlist", render: (item) => escapeHtml((item.allowlist || []).join(", ")) },
      { label: "Denylist", render: (item) => escapeHtml((item.denylist || []).join(", ")) },
      { label: "Updated", render: (item) => formatDate(item.updated_at) },
    ]);

    container.querySelector("#rate-limit-policy-list").innerHTML = renderPolicyTable(rateLimitPolicies || [], [
      { label: "ID", render: (item) => `<span class="waf-code">${escapeHtml(item.id)}</span>` },
      { label: "Service", render: (item) => escapeHtml(siteLabel(item.site_id)) },
      { label: "Enabled", render: (item) => statusBadge(item.enabled ? "active" : "failed") },
      { label: "RPS", render: (item) => escapeHtml(item.limits?.requests_per_second ?? 0) },
      { label: "Burst", render: (item) => escapeHtml(item.limits?.burst ?? 0) },
      { label: "Updated", render: (item) => formatDate(item.updated_at) },
    ]);
  };

  container.querySelector("#waf-policy-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      id: container.querySelector("#waf-id").value,
      site_id: container.querySelector("#waf-site-id").value,
      enabled: container.querySelector("#waf-enabled").checked,
      mode: container.querySelector("#waf-mode").value,
      crs_enabled: container.querySelector("#waf-crs-enabled").checked,
      custom_rule_includes: csvToList(container.querySelector("#waf-custom-includes").value),
      rule_overrides: parseJSONLines(container.querySelector("#waf-rule-overrides").value),
    };
    try {
      await ctx.api.post("/api/waf-policies", payload);
    } catch {
      await ctx.api.put(`/api/waf-policies/${encodeURIComponent(payload.id)}`, payload);
    }
    ctx.notify("WAF policy saved");
    await load();
  });

  container.querySelector("#waf-delete").addEventListener("click", async () => {
    const id = container.querySelector("#waf-id").value.trim();
    if (!id) return;
    await ctx.api.delete(`/api/waf-policies/${encodeURIComponent(id)}`);
    ctx.notify("WAF policy deleted");
    await load();
  });

  container.querySelector("#access-policy-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      id: container.querySelector("#access-id").value,
      site_id: container.querySelector("#access-site-id").value,
      enabled: container.querySelector("#access-enabled").checked,
      allowlist: csvToList(container.querySelector("#access-allowlist").value),
      denylist: csvToList(container.querySelector("#access-denylist").value),
    };
    try {
      await ctx.api.post("/api/access-policies", payload);
    } catch {
      await ctx.api.put(`/api/access-policies/${encodeURIComponent(payload.id)}`, payload);
    }
    ctx.notify("Access policy saved");
    await load();
  });

  container.querySelector("#access-delete").addEventListener("click", async () => {
    const id = container.querySelector("#access-id").value.trim();
    if (!id) return;
    await ctx.api.delete(`/api/access-policies/${encodeURIComponent(id)}`);
    ctx.notify("Access policy deleted");
    await load();
  });

  container.querySelector("#rate-limit-policy-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      id: container.querySelector("#rate-limit-id").value,
      site_id: container.querySelector("#rate-limit-site-id").value,
      enabled: container.querySelector("#rate-limit-enabled").checked,
      limits: {
        requests_per_second: Number(container.querySelector("#rate-limit-rps").value || "0"),
        burst: Number(container.querySelector("#rate-limit-burst").value || "0"),
      },
    };
    try {
      await ctx.api.post("/api/rate-limit-policies", payload);
    } catch {
      await ctx.api.put(`/api/rate-limit-policies/${encodeURIComponent(payload.id)}`, payload);
    }
    ctx.notify("Rate limit policy saved");
    await load();
  });

  container.querySelector("#rate-limit-delete").addEventListener("click", async () => {
    const id = container.querySelector("#rate-limit-id").value.trim();
    if (!id) return;
    await ctx.api.delete(`/api/rate-limit-policies/${encodeURIComponent(id)}`);
    ctx.notify("Rate limit policy deleted");
    await load();
  });

  await load();
}
