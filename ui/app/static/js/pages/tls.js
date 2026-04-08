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

function parseKeyValueLines(value) {
  const map = {};
  const lines = String(value || "").split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }
    const delimiterIndex = trimmed.indexOf("=");
    if (delimiterIndex <= 0) {
      continue;
    }
    const key = trimmed.slice(0, delimiterIndex).trim().toUpperCase();
    const rawValue = trimmed.slice(delimiterIndex + 1).trim();
    if (!key || !rawValue) {
      continue;
    }
    map[key] = rawValue;
  }
  return map;
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

function downloadBlob(filename, blob) {
  const link = document.createElement("a");
  const href = URL.createObjectURL(blob);
  link.href = href;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(href);
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

function renderCertificates(items, selectedIDs, ctx) {
  if (!items.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("tls.empty.certificates"))}</div>`;
  }
  const allSelected = items.length > 0 && items.every((item) => selectedIDs.has(String(item?.id || "")));
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr><th><input type="checkbox" id="tls-cert-select-all" ${allSelected ? "checked" : ""}></th><th>${escapeHtml(ctx.t("tls.table.id"))}</th><th>${escapeHtml(ctx.t("tls.table.commonName"))}</th><th>${escapeHtml(ctx.t("tls.table.san"))}</th><th>${escapeHtml(ctx.t("tls.table.status"))}</th><th>${escapeHtml(ctx.t("tls.table.notAfter"))}</th></tr></thead>
        <tbody>
          ${items.map((item) => {
            const id = String(item?.id || "");
            const checked = selectedIDs.has(id) ? "checked" : "";
            return `
              <tr>
                <td><input type="checkbox" data-cert-select-id="${escapeHtml(id)}" ${checked}></td>
                <td class="waf-code">${escapeHtml(id)}</td>
                <td>${escapeHtml(item.common_name)}</td>
                <td>${escapeHtml((item.san_list || []).join(", "))}</td>
                <td>${statusBadge(item.status)}</td>
                <td>${formatDate(item.not_after)}</td>
              </tr>
            `;
          }).join("")}
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
  const selectedCertificateIDs = new Set();
  let certificatesState = [];

  container.innerHTML = `
    <div class="waf-page-stack waf-tls-page-stack">
      <div class="waf-grid two waf-grid-compact">
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.certificates.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.certificates.subtitle"))}</div></div></div>
          <div class="waf-card-body waf-stack">
            <form id="certificate-form" class="waf-form">
              <div class="waf-form-grid waf-form-grid-compact">
                <div class="waf-field"><label for="certificate-id">${escapeHtml(ctx.t("tls.field.id"))}</label><input id="certificate-id" required></div>
                <div class="waf-field"><label for="certificate-common-name">${escapeHtml(ctx.t("tls.field.commonName"))}</label><input id="certificate-common-name" required></div>
                <div class="waf-field full"><label for="certificate-san-list">${escapeHtml(ctx.t("tls.field.sanList"))}</label><textarea id="certificate-san-list" placeholder="${escapeHtml(ctx.t("tls.placeholder.sanList"))}"></textarea></div>
                <div class="waf-field"><label for="certificate-not-before">${escapeHtml(ctx.t("tls.field.notBeforeRfc"))}</label><input id="certificate-not-before" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="certificate-not-after">${escapeHtml(ctx.t("tls.field.notAfterRfc"))}</label><input id="certificate-not-after" placeholder="${escapeHtml(ctx.t("tls.placeholder.rfc3339"))}"></div>
                <div class="waf-field"><label for="certificate-status">${escapeHtml(ctx.t("tls.field.status"))}</label><select id="certificate-status"><option value="active">active</option><option value="inactive">inactive</option><option value="expired">expired</option><option value="revoked">revoked</option></select></div>
              </div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("tls.action.createOrUpdate"))}</button>
                <button class="btn ghost btn-sm" type="button" id="certificate-delete">${escapeHtml(ctx.t("common.delete"))}</button>
                <button class="btn ghost btn-sm" type="button" id="certificate-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
              </div>
            </form>
            <div class="waf-actions">
              <button class="btn ghost btn-sm" type="button" id="tls-certificate-import">${escapeHtml(ctx.t("tls.certificates.import"))}</button>
              <button class="btn ghost btn-sm" type="button" id="tls-certificate-export">${escapeHtml(ctx.t("tls.certificates.export"))}</button>
            </div>
            <input id="tls-certificate-import-archive" type="file" accept=".zip,application/zip,application/x-zip-compressed" class="waf-hidden">
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
        <section class="waf-card waf-card-compact">
          <div class="waf-card-head"><div><h3>${escapeHtml(ctx.t("tls.autoRenew.title"))}</h3><div class="muted">${escapeHtml(ctx.t("tls.autoRenew.subtitle"))}</div></div></div>
          <div class="waf-card-body waf-stack">
            <form id="tls-auto-renew-form" class="waf-form">
              <label class="waf-checkbox">
                <input id="tls-auto-renew-enabled" type="checkbox">
                <span>${escapeHtml(ctx.t("tls.autoRenew.enabled"))}</span>
              </label>
              <div class="waf-field">
                <label for="tls-auto-renew-days">${escapeHtml(ctx.t("tls.autoRenew.renewBeforeDays"))}</label>
                <input id="tls-auto-renew-days" type="number" min="1" max="365" step="1" value="30">
              </div>
              <div class="waf-actions">
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t("common.save"))}</button>
              </div>
            </form>
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
                <div class="waf-field"><label for="upload-status">${escapeHtml(ctx.t("tls.field.status"))}</label><select id="upload-status"><option value="active">active</option><option value="inactive">inactive</option><option value="expired">expired</option><option value="revoked">revoked</option></select></div>
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
                <div class="waf-field"><label for="le-account-email">${escapeHtml(ctx.t("tls.field.accountEmail"))}</label><input id="le-account-email" type="email" placeholder="admin@example.com"></div>
                <div class="waf-field"><label for="le-ca-server">${escapeHtml(ctx.t("tls.field.caServer"))}</label><select id="le-ca-server"><option value="letsencrypt">letsencrypt</option><option value="zerossl">zerossl</option><option value="custom">custom</option></select></div>
                <div class="waf-field full" data-acme-visible-for="custom">
                  <label for="le-custom-directory-url">${escapeHtml(ctx.t("tls.field.customDirectoryUrl"))}</label>
                  <input id="le-custom-directory-url" placeholder="https://acme.example.com/directory">
                </div>
                <div class="waf-field" data-acme-visible-for="zerossl">
                  <label for="le-zerossl-eab-kid">${escapeHtml(ctx.t("tls.field.zerosslEabKid"))}</label>
                  <input id="le-zerossl-eab-kid">
                </div>
                <div class="waf-field" data-acme-visible-for="zerossl">
                  <label for="le-zerossl-eab-hmac">${escapeHtml(ctx.t("tls.field.zerosslEabHmac"))}</label>
                  <input id="le-zerossl-eab-hmac" type="password">
                </div>
                <div class="waf-field">
                  <label for="le-challenge-type">${escapeHtml(ctx.t("tls.field.challengeType"))}</label>
                  <select id="le-challenge-type">
                    <option value="http-01">http-01</option>
                    <option value="dns-01">dns-01</option>
                  </select>
                </div>
                <div class="waf-field" data-acme-visible-for-challenge="dns-01">
                  <label for="le-dns-provider">${escapeHtml(ctx.t("tls.field.dnsProvider"))}</label>
                  <select id="le-dns-provider">
                    <option value="cloudflare">cloudflare</option>
                  </select>
                </div>
                <div class="waf-field" data-acme-visible-for-challenge="dns-01">
                  <label for="le-dns-propagation-seconds">${escapeHtml(ctx.t("tls.field.dnsPropagationSeconds"))}</label>
                  <input id="le-dns-propagation-seconds" type="number" min="0" step="1" value="120">
                </div>
                <div class="waf-field full" data-acme-visible-for-challenge="dns-01">
                  <label for="le-dns-provider-env">${escapeHtml(ctx.t("tls.field.dnsProviderEnv"))}</label>
                  <textarea id="le-dns-provider-env" placeholder="CLOUDFLARE_API_TOKEN=..."></textarea>
                </div>
                <div class="waf-field full" data-acme-visible-for-challenge="dns-01">
                  <label for="le-dns-resolvers">${escapeHtml(ctx.t("tls.field.dnsResolvers"))}</label>
                  <textarea id="le-dns-resolvers" placeholder="1.1.1.1:53&#10;8.8.8.8:53"></textarea>
                </div>
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

  const certImportArchiveInput = container.querySelector("#tls-certificate-import-archive");
  const selectedIDs = () => Array.from(selectedCertificateIDs.values());

  const bindCertificateSelection = (certificates) => {
    container.querySelectorAll("[data-cert-select-id]").forEach((node) => {
      node.addEventListener("change", () => {
        const id = String(node.getAttribute("data-cert-select-id") || "");
        if (!id) return;
        if (node.checked) {
          selectedCertificateIDs.add(id);
        } else {
          selectedCertificateIDs.delete(id);
        }
      });
    });
    container.querySelector("#tls-cert-select-all")?.addEventListener("change", (event) => {
      const checked = Boolean(event?.target?.checked);
      if (checked) {
        certificates.forEach((item) => {
          const id = String(item?.id || "");
          if (id) {
            selectedCertificateIDs.add(id);
          }
        });
      } else {
        selectedCertificateIDs.clear();
      }
      container.querySelectorAll("[data-cert-select-id]").forEach((node) => {
        node.checked = checked;
      });
    });
  };

  const loadAutoRenewSettings = async () => {
    let settings = { enabled: false, renew_before_days: 30 };
    try {
      const payload = await ctx.api.get("/api/tls/auto-renew");
      settings = {
        enabled: Boolean(payload?.enabled),
        renew_before_days: Number(payload?.renew_before_days || 30) || 30,
      };
    } catch {
      // fallback defaults
    }
    const enabledNode = container.querySelector("#tls-auto-renew-enabled");
    const daysNode = container.querySelector("#tls-auto-renew-days");
    if (enabledNode) {
      enabledNode.checked = settings.enabled;
    }
    if (daysNode) {
      daysNode.value = String(settings.renew_before_days);
    }
  };

  const load = async () => {
    const [certificates, tlsConfigs, sites, sitesSecondary] = await Promise.all([
      ctx.api.get("/api/certificates"),
      ctx.api.get("/api/tls-configs"),
      ctx.api.get("/api/sites"),
      tryGetJSON("/api-app/sites")
    ]);
    certificatesState = normalizeList(certificates || []);
    const existingIDs = new Set(certificatesState.map((item) => String(item?.id || "")).filter(Boolean));
    for (const id of Array.from(selectedCertificateIDs.values())) {
      if (!existingIDs.has(id)) {
        selectedCertificateIDs.delete(id);
      }
    }
    const siteNamesByID = new Map(
      [...normalizeList(sites), ...unwrapList(sitesSecondary, ["sites"])]
        .map((site) => [String(site?.id || ""), String(site?.primary_host || site?.id || "")])
        .filter(([id, host]) => id && host)
    );
    container.querySelector("#certificate-list").innerHTML = renderCertificates(certificatesState, selectedCertificateIDs, ctx);
    container.querySelector("#tls-config-list").innerHTML = renderTLSConfigs(tlsConfigs || [], siteNamesByID, ctx);
    bindCertificateSelection(certificatesState);
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
    await Promise.all([load(), loadAutoRenewSettings()]);
  });

  container.querySelector("#certificate-delete").addEventListener("click", async () => {
    const ids = selectedIDs();
    const typedID = container.querySelector("#certificate-id").value.trim();
    const targetIDs = ids.length ? ids : (typedID ? [typedID] : []);
    if (!targetIDs.length) {
      return;
    }
    try {
      const tlsConfigs = normalizeList(await ctx.api.get("/api/tls-configs"));
      for (const certificateID of targetIDs) {
        const linkedSiteIDs = tlsConfigs
          .filter((item) => String(item?.certificate_id || "").trim().toLowerCase() === certificateID.toLowerCase())
          .map((item) => String(item?.site_id || "").trim())
          .filter(Boolean);
        for (const siteID of linkedSiteIDs) {
          await ctx.api.delete(`/api/tls-configs/${encodeURIComponent(siteID)}`).catch((error) => {
            if (error?.status !== 404) {
              throw error;
            }
          });
        }
        await ctx.api.delete(`/api/certificates/${encodeURIComponent(certificateID)}`).catch((error) => {
          if (error?.status !== 404) {
            throw error;
          }
        });
        selectedCertificateIDs.delete(certificateID);
      }
      ctx.notify(ctx.t("tls.toast.certificateDeleted"));
      await Promise.all([load(), loadAutoRenewSettings()]);
    } catch (error) {
      ctx.notify(String(error?.message || error || ctx.t("common.error")), "error");
    }
  });

  container.querySelector("#certificate-refresh").addEventListener("click", load);

  container.querySelector("#tls-certificate-import")?.addEventListener("click", () => {
    certImportArchiveInput?.click();
  });

  certImportArchiveInput?.addEventListener("change", async () => {
    const archiveFile = certImportArchiveInput?.files?.[0] || null;
    if (!archiveFile) {
      return;
    }
    const formData = new FormData();
    formData.set("archive_file", archiveFile);
    try {
      const payload = await ctx.api.post("/api/certificate-materials/import-archive", formData);
      const importedCount = Number(payload?.imported_count || 0);
      ctx.notify(importedCount > 0 ? ctx.t("tls.certificates.importedArchive", { count: importedCount }) : ctx.t("tls.certificates.imported"));
      await Promise.all([load(), loadAutoRenewSettings()]);
    } catch (error) {
      ctx.notify(`${ctx.t("sites.tls.importFailed")}: ${String(error?.message || error)}`, "error");
    } finally {
      if (certImportArchiveInput) {
        certImportArchiveInput.value = "";
      }
    }
  });

  container.querySelector("#tls-certificate-export")?.addEventListener("click", async () => {
    const ids = selectedIDs();
    if (!ids.length) {
      ctx.notify(ctx.t("tls.certificates.selectAny"), "error");
      return;
    }
    try {
      const response = await fetch("/api/certificate-materials/export", {
        method: "POST",
        credentials: "include",
        headers: {
          Accept: "application/zip, application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ certificate_ids: ids })
      });
      if (!response.ok) {
        const bodyText = await response.text();
        let message = `HTTP ${response.status}`;
        if (bodyText) {
          try {
            const payload = JSON.parse(bodyText);
            message = String(payload?.error || payload?.message || message);
          } catch {
            message = bodyText;
          }
        }
        throw new Error(message);
      }
      const blob = await response.blob();
      downloadBlob(ids.length === 1 ? `${ids[0]}-materials.zip` : "certificate-materials.zip", blob);
      ctx.notify(ctx.t("tls.certificates.exportArchive"));
    } catch (error) {
      ctx.notify(`${ctx.t("sites.tls.exportFailed")}: ${String(error?.message || error)}`, "error");
    }
  });

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
    await Promise.all([load(), loadAutoRenewSettings()]);
  });

  container.querySelector("#tls-config-delete").addEventListener("click", async () => {
    const siteID = container.querySelector("#tls-site-id").value.trim();
    if (!siteID) return;
    await ctx.api.delete(`/api/tls-configs/${encodeURIComponent(siteID)}`);
    ctx.notify(ctx.t("tls.toast.bindingDeleted"));
    await Promise.all([load(), loadAutoRenewSettings()]);
  });

  container.querySelector("#tls-config-refresh").addEventListener("click", load);

  container.querySelector("#tls-auto-renew-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      enabled: Boolean(container.querySelector("#tls-auto-renew-enabled")?.checked),
      renew_before_days: Number(container.querySelector("#tls-auto-renew-days")?.value || 30),
    };
    await ctx.api.put("/api/tls/auto-renew", payload);
    ctx.notify(ctx.t("tls.autoRenew.saved"));
    await loadAutoRenewSettings();
  });

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
    await Promise.all([load(), loadAutoRenewSettings()]);
  });

  const toggleACMEConditionalFields = () => {
    const caServer = container.querySelector("#le-ca-server").value;
    const challengeType = container.querySelector("#le-challenge-type").value;
    container.querySelectorAll("[data-acme-visible-for]").forEach((node) => {
      const targets = String(node.getAttribute("data-acme-visible-for") || "").split(",").map((item) => item.trim()).filter(Boolean);
      node.style.display = targets.includes(caServer) ? "" : "none";
    });
    container.querySelectorAll("[data-acme-visible-for-challenge]").forEach((node) => {
      const targets = String(node.getAttribute("data-acme-visible-for-challenge") || "").split(",").map((item) => item.trim()).filter(Boolean);
      node.style.display = targets.includes(challengeType) ? "" : "none";
    });
    const stagingCheckbox = container.querySelector("#le-use-staging");
    if (stagingCheckbox) {
      if (caServer === "letsencrypt") {
        stagingCheckbox.disabled = false;
      } else {
        stagingCheckbox.checked = false;
        stagingCheckbox.disabled = true;
      }
    }
  };
  container.querySelector("#le-ca-server")?.addEventListener("change", toggleACMEConditionalFields);
  container.querySelector("#le-challenge-type")?.addEventListener("change", toggleACMEConditionalFields);
  toggleACMEConditionalFields();

  const issueLetsEncrypt = async (bindSite) => {
    const certificateID = container.querySelector("#le-certificate-id").value.trim();
    const commonName = container.querySelector("#le-common-name").value.trim();
    const siteID = container.querySelector("#le-site-id").value.trim().toLowerCase();
    const caServer = container.querySelector("#le-ca-server").value;
    const challengeType = container.querySelector("#le-challenge-type").value;
    const zeroSSLEABKID = container.querySelector("#le-zerossl-eab-kid").value.trim();
    const zeroSSLEABHMAC = container.querySelector("#le-zerossl-eab-hmac").value.trim();
    const dnsProvider = container.querySelector("#le-dns-provider").value.trim();
    const dnsProviderEnv = parseKeyValueLines(container.querySelector("#le-dns-provider-env").value);
    if (!certificateID || !commonName) {
      return;
    }
    if (caServer === "zerossl" && (!zeroSSLEABKID || !zeroSSLEABHMAC)) {
      ctx.notify(ctx.t("tls.toast.zerosslEabRequired"), "error");
      return;
    }
    if (challengeType === "dns-01" && !dnsProvider) {
      ctx.notify(ctx.t("tls.toast.dnsProviderRequired"), "error");
      return;
    }
    if (challengeType === "dns-01" && dnsProvider === "cloudflare" && !dnsProviderEnv.CLOUDFLARE_API_TOKEN && !dnsProviderEnv.CF_API_TOKEN) {
      ctx.notify(ctx.t("tls.toast.cloudflareTokenRequired"), "error");
      return;
    }
    await ctx.api.post("/api/certificates/acme/issue", {
      certificate_id: certificateID,
      common_name: commonName,
      san_list: parseLineList(container.querySelector("#le-san-list").value),
      certificate_authority_server: caServer,
      custom_directory_url: container.querySelector("#le-custom-directory-url").value.trim(),
      use_lets_encrypt_staging: container.querySelector("#le-use-staging").checked,
      account_email: container.querySelector("#le-account-email").value.trim(),
      challenge_type: challengeType,
      dns_provider: dnsProvider,
      dns_provider_env: dnsProviderEnv,
      dns_resolvers: parseLineList(container.querySelector("#le-dns-resolvers").value),
      dns_propagation_seconds: Number(container.querySelector("#le-dns-propagation-seconds").value || 0) || 0,
      zerossl_eab_kid: zeroSSLEABKID,
      zerossl_eab_hmac_key: zeroSSLEABHMAC
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
      await Promise.all([load(), loadAutoRenewSettings()]);
      return;
    }
    ctx.notify(ctx.t("tls.toast.issueJob"));
    await Promise.all([load(), loadAutoRenewSettings()]);
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

  await Promise.all([load(), loadAutoRenewSettings()]);
}











