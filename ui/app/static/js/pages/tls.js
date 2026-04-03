import { escapeHtml, formatDate, statusBadge } from "../ui.js";

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function parseLineList(value) {
  return String(value || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
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

async function putWithPostFallback(ctx, path, payload) {
  try {
    await ctx.api.post(path, payload);
    return;
  } catch (postError) {
    if (postError?.status !== 404 && postError?.status !== 405) {
      throw postError;
    }
  }
  await ctx.api.put(path, payload);
}

async function syncLetsEncryptProfileOptions(ctx, siteID, options) {
  if (!siteID) {
    return;
  }
  const endpoint = `/api/easy-site-profiles/${encodeURIComponent(siteID)}`;
  let current = null;
  try {
    current = await ctx.api.get(endpoint);
  } catch (error) {
    if (error?.status !== 404) {
      throw error;
    }
  }
  const next = {
    ...(current || {}),
    site_id: siteID,
    front_service: {
      ...(current?.front_service || {}),
      auto_lets_encrypt: true,
      use_lets_encrypt_staging: Boolean(options?.useLetsEncryptStaging),
      use_lets_encrypt_wildcard: Boolean(options?.useLetsEncryptWildcard),
      certificate_authority_server: String(options?.certificateAuthorityServer || "letsencrypt")
    }
  };
  await putWithPostFallback(ctx, endpoint, next);
}

function renderCertificates(items, ctx) {
  if (!items.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("tls.empty.certificates"))}</div>`;
  }
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr><th>${escapeHtml(ctx.t("tls.table.id"))}</th><th>${escapeHtml(ctx.t("tls.table.commonName"))}</th><th>${escapeHtml(ctx.t("tls.table.san"))}</th><th>${escapeHtml(ctx.t("tls.table.status"))}</th><th>${escapeHtml(ctx.t("tls.table.notAfter"))}</th></tr></thead>
        <tbody>
          ${items.map((item) => `
            <tr>
              <td class="waf-code">${escapeHtml(item.id)}</td>
              <td>${escapeHtml(item.common_name)}</td>
              <td>${escapeHtml((item.san_list || []).join(", "))}</td>
              <td>${statusBadge(item.status)}</td>
              <td>${formatDate(item.not_after)}</td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderTLSConfigs(items, siteNamesByID, ctx) {
  if (!items.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("tls.empty.bindings"))}</div>`;
  }
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr><th>${escapeHtml(ctx.t("tls.table.site"))}</th><th>${escapeHtml(ctx.t("tls.table.certificate"))}</th><th>${escapeHtml(ctx.t("tls.table.updated"))}</th></tr></thead>
        <tbody>
          ${items.map((item) => `
            <tr>
              <td>${escapeHtml(siteNamesByID.get(String(item.site_id || "")) || item.site_id)}</td>
              <td>${escapeHtml(item.certificate_id)}</td>
              <td>${formatDate(item.updated_at)}</td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    </div>
  `;
}

export async function renderTLS(container, ctx) {
  container.innerHTML = `
    <div class="waf-page-stack waf-tls-page-stack">
      <div class="waf-grid two waf-grid-compact">
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.meta.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.meta.subtitle"))}</div></div></div>
          <div class="waf-card-body waf-stack">
            <form id="certificate-form" class="waf-form">
              <div class="waf-form-grid waf-form-grid-compact">
                <div class="waf-field"><label for="certificate-id">${escapeHtml(ctx.t("tls.field.id"))}</label><input id="certificate-id" required></div>
                <div class="waf-field"><label for="certificate-common-name">${escapeHtml(ctx.t("tls.field.commonName"))}</label><input id="certificate-common-name" required></div>
                <div class="waf-field full"><label for="certificate-san-list">${escapeHtml(ctx.t("tls.field.sanList"))}</label><textarea id="certificate-san-list" placeholder="${escapeHtml(ctx.t("tls.placeholder.sanList"))}"></textarea></div>
                <div class="waf-field"><label for="certificate-not-before">${escapeHtml(ctx.t("tls.field.notBeforeRfc"))}</label><input id="certificate-not-before" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="certificate-not-after">${escapeHtml(ctx.t("tls.field.notAfterRfc"))}</label><input id="certificate-not-after" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="certificate-status">${escapeHtml(ctx.t("tls.field.status"))}</label><select id="certificate-status"><option value="active">active</option><option value="expired">expired</option><option value="revoked">revoked</option></select></div>
              </div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("tls.action.createOrUpdate"))}</button>
                <button class="btn ghost btn-sm" type="button" id="certificate-delete">${escapeHtml(ctx.t("common.delete"))}</button>
                <button class="btn ghost btn-sm" type="button" id="certificate-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
              </div>
            </form>
            <div id="certificate-list"></div>
          </div>
        </section>
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.bindings.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.bindings.subtitle"))}</div></div></div>
          <div class="waf-card-body waf-stack">
            <form id="tls-config-form" class="waf-form">
              <div class="waf-form-grid waf-form-grid-compact">
                <div class="waf-field"><label for="tls-site-id">${escapeHtml(ctx.t("tls.field.siteId"))}</label><input id="tls-site-id" required></div>
                <div class="waf-field"><label for="tls-certificate-id">${escapeHtml(ctx.t("tls.field.certificateId"))}</label><input id="tls-certificate-id" required></div>
              </div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("tls.action.createOrUpdate"))}</button>
                <button class="btn ghost btn-sm" type="button" id="tls-config-delete">${escapeHtml(ctx.t("common.delete"))}</button>
                <button class="btn ghost btn-sm" type="button" id="tls-config-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
              </div>
            </form>
            <div id="tls-config-list"></div>
          </div>
        </section>
      </div>
      <div class="waf-grid two waf-grid-compact">
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.upload.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.upload.subtitle"))}</div></div></div>
          <div class="waf-card-body">
            <form id="certificate-upload-form" class="waf-form">
              <div class="waf-form-grid waf-form-grid-compact">
                <div class="waf-field"><label for="upload-certificate-id">${escapeHtml(ctx.t("tls.field.certificateId"))}</label><input id="upload-certificate-id"></div>
                <div class="waf-field"><label for="upload-common-name">${escapeHtml(ctx.t("tls.field.commonName"))}</label><input id="upload-common-name"></div>
                <div class="waf-field full"><label for="upload-san-list">${escapeHtml(ctx.t("tls.field.sanList"))}</label><textarea id="upload-san-list"></textarea></div>
                <div class="waf-field"><label for="upload-not-before">${escapeHtml(ctx.t("tls.field.notBefore"))}</label><input id="upload-not-before" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="upload-not-after">${escapeHtml(ctx.t("tls.field.notAfter"))}</label><input id="upload-not-after" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="upload-status">${escapeHtml(ctx.t("tls.field.status"))}</label><select id="upload-status"><option value="active">active</option><option value="expired">expired</option><option value="revoked">revoked</option></select></div>
                <div class="waf-field"><label for="certificate-file">${escapeHtml(ctx.t("tls.field.certificatePem"))}</label><input id="certificate-file" type="file" required></div>
                <div class="waf-field"><label for="private-key-file">${escapeHtml(ctx.t("tls.field.privateKeyPem"))}</label><input id="private-key-file" type="file" required></div>
              </div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("tls.action.upload"))}</button>
              </div>
            </form>
          </div>
        </section>
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.acme.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.acme.subtitle"))}</div></div></div>
          <div class="waf-card-body waf-stack">
            <form id="letsencrypt-form" class="waf-form">
              <div class="waf-form-grid waf-form-grid-compact">
                <div class="waf-field"><label for="le-site-id">${escapeHtml(ctx.t("tls.field.siteIdOptional"))}</label><input id="le-site-id" placeholder="${escapeHtml(ctx.t("tls.placeholder.siteId"))}"></div>
                <div class="waf-field"><label for="le-certificate-id">${escapeHtml(ctx.t("tls.field.certificateId"))}</label><input id="le-certificate-id" required></div>
                <div class="waf-field"><label for="le-common-name">${escapeHtml(ctx.t("tls.field.commonName"))}</label><input id="le-common-name" required></div>
                <div class="waf-field"><label for="le-ca-server">${escapeHtml(ctx.t("tls.field.caServer"))}</label><select id="le-ca-server"><option value="letsencrypt">letsencrypt</option><option value="zerossl">zerossl</option><option value="custom">custom</option></select></div>
                <label class="waf-checkbox">
                  <input id="le-use-staging" type="checkbox">
                  <span>${escapeHtml(ctx.t("tls.field.useLetsEncryptStaging"))}</span>
                </label>
                <label class="waf-checkbox">
                  <input id="le-use-wildcard" type="checkbox">
                  <span>${escapeHtml(ctx.t("tls.field.useLetsEncryptWildcard"))}</span>
                </label>
                <div class="waf-field full"><label for="le-san-list">${escapeHtml(ctx.t("tls.field.sanList"))}</label><textarea id="le-san-list"></textarea></div>
              </div>
              <div class="waf-note">${escapeHtml(ctx.t("tls.acme.note"))}</div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("tls.action.issue"))}</button>
                <button class="btn ghost btn-sm" type="button" id="letsencrypt-issue-bind">${escapeHtml(ctx.t("tls.action.issueAndBind"))}</button>
                <button class="btn ghost btn-sm" type="button" id="letsencrypt-renew">${escapeHtml(ctx.t("tls.action.renew"))}</button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>
  `;

  const load = async () => {
    const [certificates, tlsConfigs, sites, sitesSecondary] = await Promise.all([
      ctx.api.get("/api/certificates"),
      ctx.api.get("/api/tls-configs"),
      ctx.api.get("/api/sites"),
      tryGetJSON("/api-app/sites")
    ]);
    const siteNamesByID = new Map(
      [...normalizeList(sites), ...unwrapList(sitesSecondary, ["sites"])]
        .map((site) => [String(site?.id || ""), String(site?.primary_host || site?.id || "")])
        .filter(([id, host]) => id && host)
    );
    container.querySelector("#certificate-list").innerHTML = renderCertificates(certificates || [], ctx);
    container.querySelector("#tls-config-list").innerHTML = renderTLSConfigs(tlsConfigs || [], siteNamesByID, ctx);
  };

  container.querySelector("#certificate-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      id: container.querySelector("#certificate-id").value,
      common_name: container.querySelector("#certificate-common-name").value,
      san_list: parseLineList(container.querySelector("#certificate-san-list").value),
      not_before: container.querySelector("#certificate-not-before").value.trim(),
      not_after: container.querySelector("#certificate-not-after").value.trim(),
      status: container.querySelector("#certificate-status").value,
    };
    try {
      await ctx.api.post("/api/certificates", payload);
    } catch {
      await ctx.api.put(`/api/certificates/${encodeURIComponent(payload.id)}`, payload);
    }
    ctx.notify(ctx.t("tls.toast.metaSaved"));
    await load();
  });

  container.querySelector("#certificate-delete").addEventListener("click", async () => {
    const id = container.querySelector("#certificate-id").value.trim();
    if (!id) return;
    await ctx.api.delete(`/api/certificates/${encodeURIComponent(id)}`);
    ctx.notify(ctx.t("tls.toast.certificateDeleted"));
    await load();
  });

  container.querySelector("#certificate-refresh").addEventListener("click", load);

  container.querySelector("#tls-config-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      site_id: container.querySelector("#tls-site-id").value,
      certificate_id: container.querySelector("#tls-certificate-id").value,
    };
    try {
      await ctx.api.post("/api/tls-configs", payload);
    } catch {
      await ctx.api.put(`/api/tls-configs/${encodeURIComponent(payload.site_id)}`, payload);
    }
    ctx.notify(ctx.t("tls.toast.bindingSaved"));
    await load();
  });

  container.querySelector("#tls-config-delete").addEventListener("click", async () => {
    const siteID = container.querySelector("#tls-site-id").value.trim();
    if (!siteID) return;
    await ctx.api.delete(`/api/tls-configs/${encodeURIComponent(siteID)}`);
    ctx.notify(ctx.t("tls.toast.bindingDeleted"));
    await load();
  });

  container.querySelector("#tls-config-refresh").addEventListener("click", load);

  container.querySelector("#certificate-upload-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const formData = new FormData();
    formData.set("certificate_id", container.querySelector("#upload-certificate-id").value.trim());
    formData.set("common_name", container.querySelector("#upload-common-name").value.trim());
    formData.set("not_before", container.querySelector("#upload-not-before").value.trim());
    formData.set("not_after", container.querySelector("#upload-not-after").value.trim());
    formData.set("status", container.querySelector("#upload-status").value);
    const sans = parseLineList(container.querySelector("#upload-san-list").value);
    sans.forEach((san) => formData.append("san_list", san));
    formData.set("certificate_file", container.querySelector("#certificate-file").files[0]);
    formData.set("private_key_file", container.querySelector("#private-key-file").files[0]);
    await ctx.api.post("/api/certificate-materials/upload", formData);
    ctx.notify(ctx.t("tls.toast.uploaded"));
    await load();
  });

  const issueLetsEncrypt = async (bindSite) => {
    const certificateID = container.querySelector("#le-certificate-id").value.trim();
    const commonName = container.querySelector("#le-common-name").value.trim();
    const siteID = container.querySelector("#le-site-id").value.trim().toLowerCase();
    if (!certificateID || !commonName) {
      return;
    }
    await ctx.api.post("/api/certificates/acme/issue", {
      certificate_id: certificateID,
      common_name: commonName,
      san_list: parseLineList(container.querySelector("#le-san-list").value),
    });
    await syncLetsEncryptProfileOptions(ctx, siteID, {
      certificateAuthorityServer: container.querySelector("#le-ca-server").value,
      useLetsEncryptStaging: container.querySelector("#le-use-staging").checked,
      useLetsEncryptWildcard: container.querySelector("#le-use-wildcard").checked
    });
    if (bindSite && siteID) {
      const bindingPayload = {
        site_id: siteID,
        certificate_id: certificateID
      };
      try {
        await ctx.api.post("/api/tls-configs", bindingPayload);
      } catch {
        await ctx.api.put(`/api/tls-configs/${encodeURIComponent(siteID)}`, bindingPayload);
      }
      ctx.notify(ctx.t("tls.toast.issueAndBindJob"));
      await load();
      return;
    }
    ctx.notify(ctx.t("tls.toast.issueJob"));
    await load();
  };

  container.querySelector("#letsencrypt-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    await issueLetsEncrypt(false);
  });

  container.querySelector("#letsencrypt-issue-bind").addEventListener("click", async () => {
    await issueLetsEncrypt(true);
  });

  container.querySelector("#letsencrypt-renew").addEventListener("click", async () => {
    const id = container.querySelector("#le-certificate-id").value.trim();
    if (!id) return;
    await ctx.api.post(`/api/certificates/acme/renew/${encodeURIComponent(id)}`, {});
    ctx.notify(ctx.t("tls.toast.renewJob"));
  });

  await load();
}
