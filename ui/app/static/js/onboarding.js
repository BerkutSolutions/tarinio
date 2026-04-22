import { api } from "./api.js";
import { applyTranslations, getLanguage, t } from "./i18n.js";
import { checkEntryAccess, markOnboardingRedirecting } from "./guard.js";
import { escapeHtml, notify, setError, setLoading } from "./ui.js";

const state = {
  step: 1,
  setup: null,
  user: null,
  site: null,
  upstream: null,
  certificateID: "",
  tlsMode: "letsencrypt",
  applyInFlight: false,
  adminDraft: {
    username: "",
    email: "",
    password: ""
  },
  siteDraft: {
    host: "",
    tlsMode: "letsencrypt",
    accountEmail: "",
    challengeType: "http-01",
    dnsProvider: "cloudflare",
    dnsProviderEnv: "",
    dnsPropagationSeconds: 120,
    zeroSSLEABKID: "",
    zeroSSLEABHMAC: ""
  }
};

const onboardingDraftStorageKey = "waf_onboarding_draft";
const onboardingNoAutoApplyOptions = {
  headers: {
    "X-WAF-Auto-Apply-Disabled": "1"
  }
};
const stepRouteMap = {
  1: "/onboarding/user-creation",
  2: "/onboarding/site-tls",
  3: "/onboarding/confirm"
};

function persistDraft() {
  window.sessionStorage.setItem(onboardingDraftStorageKey, JSON.stringify({
    adminDraft: state.adminDraft,
    siteDraft: state.siteDraft,
    step: state.step
  }));
}

function restoreDraft() {
  try {
    const raw = window.sessionStorage.getItem(onboardingDraftStorageKey);
    if (!raw) {
      return;
    }
    const draft = JSON.parse(raw);
    state.adminDraft = { ...state.adminDraft, ...(draft.adminDraft || {}) };
    state.siteDraft = { ...state.siteDraft, ...(draft.siteDraft || {}) };
    if (draft.step >= 1 && draft.step <= 3) {
      state.step = draft.step;
    }
  } catch {
    window.sessionStorage.removeItem(onboardingDraftStorageKey);
  }
}

function clearDraft() {
  window.sessionStorage.removeItem(onboardingDraftStorageKey);
}

function stepFromPath() {
  const path = window.location.pathname.replace(/\/+$/, "");
  if (path.endsWith("/site-tls")) {
    return 2;
  }
  if (path.endsWith("/confirm")) {
    return 3;
  }
  return 1;
}

function syncStepRoute(replaceHistory = false) {
  const target = stepRouteMap[state.step] || stepRouteMap[1];
  if ((window.location.pathname || "").replace(/\/+$/, "") === target) {
    return;
  }
  if (replaceHistory) {
    window.history.replaceState({}, "", target);
    return;
  }
  window.history.pushState({}, "", target);
}

function httpsUrl(path = "/", hostOverride = "") {
  const host = String(hostOverride || window.location.hostname || "localhost").trim().toLowerCase() || "localhost";
  return `https://${host}${path}`;
}

function detectSiteHost() {
  const host = (window.location.hostname || "localhost").trim().toLowerCase();
  return host || "localhost";
}

function siteIDFromHost(host) {
  return (host || "site-1").toLowerCase().replace(/[^a-z0-9.-]+/g, "-").replace(/^-+|-+$/g, "") || "site-1";
}

function buildSitePayload() {
  const host = ((document.getElementById("site-host")?.value || state.siteDraft.host).trim() || detectSiteHost()).toLowerCase();
  return {
    id: siteIDFromHost(host),
    primary_host: host,
    enabled: true
  };
}

function buildUpstreamPayload(siteID) {
  return {
    id: `${siteID}-upstream`,
    site_id: siteID,
    host: window.WAF_ONBOARDING_UPSTREAM_HOST || "ui",
    port: Number(window.WAF_ONBOARDING_UPSTREAM_PORT || 80),
    scheme: "http"
  };
}

function buildCertificateID(siteID) {
  return `${siteID}-tls`;
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

function buildUpstreamHostURL(upstream) {
  const scheme = String(upstream?.scheme || "http").toLowerCase() === "https" ? "https" : "http";
  const host = String(upstream?.host || "upstream-server").trim() || "upstream-server";
  const port = Number(upstream?.port || (scheme === "https" ? 443 : 80));
  if (port > 0) {
    return `${scheme}://${host}:${port}`;
  }
  return `${scheme}://${host}`;
}

async function ensureFirstInitEasyProfileTemplate(site, upstream, tlsMode, acmeAccountEmail = "") {
  const endpoint = `/api/easy-site-profiles/${encodeURIComponent(site.id)}`;
  const current = await api.get(endpoint, onboardingNoAutoApplyOptions);
  const selectedTLSMode = String(tlsMode || "letsencrypt").toLowerCase();
  const isSelfSigned = selectedTLSMode === "self-signed";
  const selectedCA = selectedTLSMode === "zerossl" ? "zerossl" : "letsencrypt";
  const next = {
    ...current,
    site_id: site.id,
    front_service: {
      ...(current?.front_service || {}),
      server_name: site.primary_host,
      auto_lets_encrypt: !isSelfSigned,
      certificate_authority_server: selectedCA,
      acme_account_email: String(acmeAccountEmail || current?.front_service?.acme_account_email || "").trim().toLowerCase()
    },
    upstream_routing: {
      ...(current?.upstream_routing || {}),
      use_reverse_proxy: true,
      reverse_proxy_host: buildUpstreamHostURL(upstream),
      reverse_proxy_url: String(current?.upstream_routing?.reverse_proxy_url || "/").startsWith("/")
        ? String(current?.upstream_routing?.reverse_proxy_url || "/")
        : "/"
    }
  };
  return api.put(endpoint, next, onboardingNoAutoApplyOptions);
}

async function compileAndApplyRevision() {
  setLoading(document.getElementById("onboarding-feedback"), t("onboarding.apply.loading"));
  const compileResponse = await api.post("/api/revisions/compile", {}, onboardingNoAutoApplyOptions);
  const revisionID = String(compileResponse?.revision?.id || "").trim();
  if (!revisionID) {
    throw new Error(t("onboarding.error.apply"));
  }
  setLoading(document.getElementById("onboarding-feedback"), t("onboarding.apply.loading"));
  await api.post(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, {});
  state.setup = await loadSetupStatus();
  if (!state.setup?.has_active_revision) {
    throw new Error(t("onboarding.error.apply"));
  }
}

function currentTLSMode() {
  const zeroSSL = document.getElementById("tls-zerossl");
  const selfSigned = document.getElementById("tls-self-signed");
  if (!selfSigned) {
    return state.siteDraft.tlsMode || "letsencrypt";
  }
  if (zeroSSL?.checked) {
    return "zerossl";
  }
  return selfSigned.checked ? "self-signed" : "letsencrypt";
}

function syncStage2ACMEFields() {
  const tlsMode = currentTLSMode();
  const challengeType = String(document.getElementById("onboarding-acme-challenge")?.value || state.siteDraft.challengeType || "http-01");
  const acmeOptions = document.getElementById("onboarding-acme-options");
  if (acmeOptions) {
    acmeOptions.classList.toggle("waf-hidden", tlsMode === "self-signed");
  }
  document.querySelectorAll("[data-onboarding-visible-tls]").forEach((node) => {
    const targets = String(node.getAttribute("data-onboarding-visible-tls") || "").split(",").map((item) => item.trim()).filter(Boolean);
    node.classList.toggle("waf-hidden", !targets.includes(tlsMode));
  });
  document.querySelectorAll("[data-onboarding-visible-challenge]").forEach((node) => {
    const targets = String(node.getAttribute("data-onboarding-visible-challenge") || "").split(",").map((item) => item.trim()).filter(Boolean);
    node.classList.toggle("waf-hidden", !targets.includes(challengeType) || tlsMode === "self-signed");
  });
}

function setApplyBusy(busy) {
  state.applyInFlight = busy;
  document.getElementById("onboarding-apply").disabled = busy;
  document.getElementById("onboarding-next").disabled = busy;
  document.getElementById("onboarding-back").disabled = busy || state.step === 1;
}

function updateStepUI() {
  document.querySelectorAll("[data-step-indicator]").forEach((node) => {
    const step = Number(node.dataset.stepIndicator);
    node.classList.toggle("is-active", step === state.step);
    node.classList.toggle("is-done", step < state.step);
  });
  document.querySelectorAll("[data-step]").forEach((node) => {
    node.classList.toggle("waf-hidden", Number(node.dataset.step) !== state.step);
  });

  document.getElementById("onboarding-step-badge").textContent = t("onboarding.stepBadge", { step: state.step });
  document.getElementById("onboarding-back").disabled = state.applyInFlight || state.step === 1;
  document.getElementById("onboarding-next").classList.toggle("waf-hidden", state.step === 3);
  document.getElementById("onboarding-next").disabled = state.applyInFlight;
  document.getElementById("onboarding-apply").classList.toggle("waf-hidden", state.step !== 3);
  document.getElementById("onboarding-apply").disabled = state.applyInFlight;
  syncStepRoute();
  persistDraft();
}

function clearFeedback() {
  document.getElementById("onboarding-feedback").innerHTML = "";
}

function configureStage1() {
  const note = document.getElementById("admin-setup-note");
  const username = document.getElementById("admin-username");
  const email = document.getElementById("admin-email");
  const password = document.getElementById("admin-password");
  const confirm = document.getElementById("admin-password-confirm");

  if (state.setup?.needs_bootstrap) {
    note.textContent = t("onboarding.admin.noteBootstrap");
    username.value = state.adminDraft.username || "";
    email.value = state.adminDraft.email || "";
    password.value = state.adminDraft.password || "";
    confirm.value = state.adminDraft.password || "";
    username.readOnly = false;
    email.readOnly = false;
    password.readOnly = false;
    confirm.readOnly = false;
    return;
  }

  note.textContent = t("onboarding.admin.noteExisting");
  username.value = state.user?.username || state.user?.id || state.adminDraft.username || "admin";
  email.value = state.user?.email || state.adminDraft.email || "";
  username.readOnly = Boolean(state.user);
  email.readOnly = Boolean(state.user);
  password.value = state.adminDraft.password || "";
  confirm.value = state.adminDraft.password || "";
  password.readOnly = false;
  confirm.readOnly = false;
}

function configureStage2() {
  document.getElementById("site-host").value = state.siteDraft.host || state.site?.primary_host || detectSiteHost();
  const tlsMode = state.siteDraft.tlsMode || state.tlsMode || "letsencrypt";
  document.getElementById("tls-self-signed").checked = tlsMode === "self-signed";
  document.getElementById("tls-zerossl").checked = tlsMode === "zerossl";
  document.getElementById("tls-letsencrypt").checked = tlsMode !== "self-signed" && tlsMode !== "zerossl";
  document.getElementById("onboarding-acme-email").value = state.siteDraft.accountEmail || "";
  document.getElementById("onboarding-acme-challenge").value = state.siteDraft.challengeType || "http-01";
  document.getElementById("onboarding-dns-provider").value = state.siteDraft.dnsProvider || "cloudflare";
  document.getElementById("onboarding-dns-provider-env").value = state.siteDraft.dnsProviderEnv || "";
  document.getElementById("onboarding-dns-propagation").value = String(state.siteDraft.dnsPropagationSeconds || 120);
  document.getElementById("onboarding-zerossl-eab-kid").value = state.siteDraft.zeroSSLEABKID || "";
  document.getElementById("onboarding-zerossl-eab-hmac").value = state.siteDraft.zeroSSLEABHMAC || "";
  syncStage2ACMEFields();
}

function syncPasswordToggle(button, visible) {
  button.dataset.visible = visible ? "true" : "false";
  button.setAttribute("aria-pressed", visible ? "true" : "false");
  button.title = t(visible ? "onboarding.admin.passwordToggleHide" : "onboarding.admin.passwordToggleShow");
  button.querySelector(".waf-password-icon-eye")?.classList.toggle("waf-hidden", visible);
  button.querySelector(".waf-password-icon-off")?.classList.toggle("waf-hidden", !visible);
}

function togglePasswordField(button) {
  const input = document.getElementById(button.dataset.target);
  if (!input) {
    return;
  }
  const visible = input.type === "text";
  input.type = visible ? "password" : "text";
  syncPasswordToggle(button, !visible);
}

function maskedPassword() {
  if (!state.adminDraft.password) {
    return t("common.notShown");
  }
  return "\u2022".repeat(Math.max(8, state.adminDraft.password.length));
}

function confirmPasswordValue() {
  return state.adminDraft.password || t("common.notShown");
}

function renderSummary() {
  const host = state.site?.primary_host || "-";
  const container = document.getElementById("onboarding-summary");
  container.innerHTML = `
    <div class="waf-list-item waf-summary-card">
      <div class="waf-list-head">
        <div class="waf-list-title">${escapeHtml(t("onboarding.confirm.adminAccount"))}</div>
        <button class="btn ghost btn-sm" type="button" id="summary-edit-admin">${escapeHtml(t("common.edit"))}</button>
      </div>
      <div class="waf-summary-grid">
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.username"))}</div>
          <div>${escapeHtml(state.user?.username || state.adminDraft.username || "-")}</div>
        </div>
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.email"))}</div>
          <div>${escapeHtml(state.user?.email || state.adminDraft.email || t("common.notSet"))}</div>
        </div>
        <div class="waf-field full">
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.password"))}</div>
          <div class="waf-inline">
            <span id="summary-password-value" class="waf-code">${escapeHtml(maskedPassword())}</span>
            <button class="btn ghost btn-sm" type="button" id="summary-toggle-password">${escapeHtml(t("common.show"))}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="waf-list-item waf-summary-card">
      <div class="waf-list-head">
        <div class="waf-list-title">${escapeHtml(t("onboarding.confirm.siteAndUpstream"))}</div>
        <button class="btn ghost btn-sm" type="button" id="summary-edit-site">${escapeHtml(t("common.edit"))}</button>
      </div>
      <div class="waf-summary-grid">
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.siteHost"))}</div>
          <div>${escapeHtml(host)}</div>
        </div>
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.siteId"))}</div>
          <div class="waf-code">${escapeHtml(state.site?.id || "-")}</div>
        </div>
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.upstreamTarget"))}</div>
          <div>${escapeHtml(state.upstream?.host || "-")}:${escapeHtml(String(state.upstream?.port || "-"))}</div>
        </div>
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.upstreamId"))}</div>
          <div class="waf-code">${escapeHtml(state.upstream?.id || "-")}</div>
        </div>
      </div>
    </div>
    <div class="waf-list-item waf-summary-card">
      <div class="waf-list-head"><div class="waf-list-title">${escapeHtml(t("onboarding.confirm.tls"))}</div></div>
      <div class="waf-summary-grid">
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.certificateMode"))}</div>
          <div>${escapeHtml(t(state.tlsMode === "self-signed" ? "onboarding.confirm.tlsSelfSignedMode" : state.tlsMode === "zerossl" ? "onboarding.confirm.tlsZeroSSLMode" : "onboarding.confirm.tlsManagedMode"))}</div>
        </div>
        <div>
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.certificateId"))}</div>
          <div class="waf-code">${escapeHtml(state.certificateID || "-")}</div>
        </div>
        <div class="waf-field full">
          <div class="waf-note">${escapeHtml(t("onboarding.confirm.afterApply"))}</div>
          <div>${escapeHtml(t("onboarding.confirm.afterApplyText", { url: `https://${host}` }))}</div>
        </div>
      </div>
    </div>
  `;

  container.querySelector("#summary-edit-admin")?.addEventListener("click", () => {
    state.step = 1;
    updateStepUI();
  });
  container.querySelector("#summary-edit-site")?.addEventListener("click", () => {
    state.step = 2;
    updateStepUI();
  });
  container.querySelector("#summary-toggle-password")?.addEventListener("click", (event) => {
    const value = container.querySelector("#summary-password-value");
    const visible = event.target.dataset.visible === "true";
    value.textContent = visible ? maskedPassword() : confirmPasswordValue();
    event.target.textContent = t(visible ? "common.show" : "common.hide");
    event.target.dataset.visible = visible ? "false" : "true";
  });
}

async function loadSetupStatus() {
  state.setup = await api.get("/api/setup/status");
  return state.setup;
}

async function ensureAuthIfNeeded() {
  if (state.setup?.needs_bootstrap || !state.setup?.has_active_revision) {
    return null;
  }
  try {
    const me = await api.get("/api/auth/me");
    state.user = me;
    return me;
  } catch (error) {
    if (error.status === 401) {
      return null;
    }
    throw error;
  }
}

async function bootstrapAdminIfNeeded() {
  const username = document.getElementById("admin-username").value.trim();
  const email = document.getElementById("admin-email").value.trim();
  const password = document.getElementById("admin-password").value;
  const confirm = document.getElementById("admin-password-confirm").value;

  state.adminDraft = { username, email, password };

  if (!state.setup?.needs_bootstrap) {
    if (!username) {
      throw new Error(t("onboarding.validation.usernameRequired"));
    }
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      throw new Error(t("onboarding.validation.emailInvalid"));
    }
    if (password && password !== confirm) {
      throw new Error(t("onboarding.validation.passwordConfirmMismatch"));
    }
    persistDraft();
    return {
      username,
      email
    };
  }

  if (!username) {
    throw new Error(t("onboarding.validation.usernameRequired"));
  }
  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    throw new Error(t("onboarding.validation.emailInvalid"));
  }
  if (!password) {
    throw new Error(t("onboarding.validation.passwordRequired"));
  }
  if (password !== confirm) {
    throw new Error(t("onboarding.validation.passwordConfirmMismatch"));
  }
  persistDraft();
  return {
    username,
    email
  };
}

async function ensureSiteAndTLS() {
  setLoading(document.getElementById("onboarding-feedback"), t("onboarding.apply.createResources"));

  const site = buildSitePayload();
  const upstream = buildUpstreamPayload(site.id);
  const certificateID = buildCertificateID(site.id);
  const tlsMode = currentTLSMode();
  const acmeEmail = String(document.getElementById("onboarding-acme-email")?.value || state.siteDraft.accountEmail || "").trim();
  const challengeType = String(document.getElementById("onboarding-acme-challenge")?.value || state.siteDraft.challengeType || "http-01").trim();
  const dnsProvider = String(document.getElementById("onboarding-dns-provider")?.value || state.siteDraft.dnsProvider || "cloudflare").trim();
  const dnsProviderEnvRaw = String(document.getElementById("onboarding-dns-provider-env")?.value || state.siteDraft.dnsProviderEnv || "");
  const dnsPropagationSeconds = Number(document.getElementById("onboarding-dns-propagation")?.value || state.siteDraft.dnsPropagationSeconds || 120) || 0;
  const zeroSSLEABKID = String(document.getElementById("onboarding-zerossl-eab-kid")?.value || state.siteDraft.zeroSSLEABKID || "").trim();
  const zeroSSLEABHMAC = String(document.getElementById("onboarding-zerossl-eab-hmac")?.value || state.siteDraft.zeroSSLEABHMAC || "").trim();

  if (tlsMode !== "self-signed" && !acmeEmail) {
    throw new Error(t("onboarding.validation.acmeEmailRequired"));
  }
  if (tlsMode === "zerossl" && (!zeroSSLEABKID || !zeroSSLEABHMAC)) {
    throw new Error(t("onboarding.validation.zerosslEabRequired"));
  }
  if (tlsMode !== "self-signed" && challengeType === "dns-01") {
    if (!dnsProvider) {
      throw new Error(t("onboarding.validation.dnsProviderRequired"));
    }
    const env = parseKeyValueLines(dnsProviderEnvRaw);
    if (dnsProvider === "cloudflare" && !env.CLOUDFLARE_API_TOKEN && !env.CF_API_TOKEN) {
      throw new Error(t("onboarding.validation.cloudflareTokenRequired"));
    }
  }

  const [sitesResponse, upstreamsResponse, certificatesResponse, tlsConfigsResponse] = await Promise.all([
    api.get("/api/sites"),
    api.get("/api/upstreams"),
    api.get("/api/certificates"),
    api.get("/api/tls-configs")
  ]);
  const sites = Array.isArray(sitesResponse) ? sitesResponse : [];
  const upstreams = Array.isArray(upstreamsResponse) ? upstreamsResponse : [];
  const certificates = Array.isArray(certificatesResponse) ? certificatesResponse : [];
  const tlsConfigs = Array.isArray(tlsConfigsResponse) ? tlsConfigsResponse : [];
  const isDefaultUpstreamRequiredError = (error) =>
    String(error?.message || "").toLowerCase().includes("default upstream is required");

  if (!sites.some((item) => item.id === site.id)) {
    try {
      await api.post("/api/sites", site, onboardingNoAutoApplyOptions);
    } catch (error) {
      if (!isDefaultUpstreamRequiredError(error)) {
        throw error;
      }
      const refreshedSites = await api.get("/api/sites");
      const siteExists = Array.isArray(refreshedSites) && refreshedSites.some((item) => item.id === site.id);
      if (!siteExists) {
        throw error;
      }
    }
  }
  if (!upstreams.some((item) => item.id === upstream.id)) {
    try {
      await api.post("/api/upstreams", upstream, onboardingNoAutoApplyOptions);
    } catch (error) {
      const conflict = error?.status === 409 || String(error?.message || "").toLowerCase().includes("already exists");
      if (!conflict) {
        throw error;
      }
      await api.put(`/api/upstreams/${encodeURIComponent(upstream.id)}`, upstream, onboardingNoAutoApplyOptions);
    }
  }
  if (tlsMode === "self-signed") {
    await api.post("/api/certificates/self-signed/issue", {
      certificate_id: certificateID,
      common_name: site.primary_host,
      san_list: []
    }, onboardingNoAutoApplyOptions);
  } else {
    await api.post("/api/certificates/acme/issue", {
      certificate_id: certificateID,
      common_name: site.primary_host,
      san_list: [],
      certificate_authority_server: tlsMode === "zerossl" ? "zerossl" : "letsencrypt",
      account_email: acmeEmail,
      challenge_type: challengeType,
      dns_provider: challengeType === "dns-01" ? dnsProvider : "",
      dns_provider_env: challengeType === "dns-01" ? parseKeyValueLines(dnsProviderEnvRaw) : {},
      dns_resolvers: [],
      dns_propagation_seconds: challengeType === "dns-01" ? dnsPropagationSeconds : 0,
      zerossl_eab_kid: tlsMode === "zerossl" ? zeroSSLEABKID : "",
      zerossl_eab_hmac_key: tlsMode === "zerossl" ? zeroSSLEABHMAC : ""
    }, onboardingNoAutoApplyOptions);
  }

  if (!tlsConfigs.some((item) => item.site_id === site.id)) {
    const tlsPayload = {
      site_id: site.id,
      certificate_id: certificateID
    };
    try {
      await api.post("/api/tls-configs", tlsPayload, onboardingNoAutoApplyOptions);
    } catch (error) {
      const conflict = error?.status === 409 || String(error?.message || "").toLowerCase().includes("already exists");
      if (!conflict) {
        throw error;
      }
      await api.put(`/api/tls-configs/${encodeURIComponent(site.id)}`, tlsPayload, onboardingNoAutoApplyOptions);
    }
  }
  await ensureFirstInitEasyProfileTemplate(site, upstream, tlsMode, acmeEmail);

  state.site = sites.find((item) => item.id === site.id) || site;
  state.upstream = upstreams.find((item) => item.id === upstream.id) || upstream;
  state.certificateID = certificates.find((item) => item.id === certificateID)?.id || certificateID;
  state.tlsMode = tlsMode;
  state.siteDraft.accountEmail = acmeEmail;
  state.siteDraft.challengeType = challengeType;
  state.siteDraft.dnsProvider = dnsProvider;
  state.siteDraft.dnsProviderEnv = dnsProviderEnvRaw;
  state.siteDraft.dnsPropagationSeconds = dnsPropagationSeconds;
  state.siteDraft.zeroSSLEABKID = zeroSSLEABKID;
  state.siteDraft.zeroSSLEABHMAC = zeroSSLEABHMAC;
  persistDraft();
}

async function runApply() {
  if (state.applyInFlight) {
    return;
  }

  setApplyBusy(true);
  try {
    setLoading(document.getElementById("onboarding-feedback"), t("onboarding.apply.loading"));

    if (state.setup?.needs_bootstrap) {
      const session = await api.post("/api/auth/bootstrap", {
        username: state.adminDraft.username,
        email: state.adminDraft.email,
        password: state.adminDraft.password
      });
      state.user = session.user;
      state.setup = await api.get("/api/setup/status");
    } else if (!state.user) {
      if (!state.adminDraft.username) {
        throw new Error(t("onboarding.validation.usernameRequired"));
      }
      if (!state.adminDraft.password) {
        throw new Error(t("onboarding.validation.passwordRequired"));
      }
      const login = await api.post("/api/auth/login", {
        username: state.adminDraft.username,
        password: state.adminDraft.password
      });
      if (login?.requires_2fa) {
        throw new Error(t("login.error2faRequired"));
      }
      state.user = login?.session?.user || null;
    }

    await ensureSiteAndTLS();
    await compileAndApplyRevision();
    renderSummary();

    notify(t("toast.initialSetupCompleted"));
    clearDraft();
    markOnboardingRedirecting();
    await api.post("/api/auth/logout", {});
    setLoading(document.getElementById("onboarding-feedback"), t("onboarding.apply.redirecting"));
    window.setTimeout(() => {
      window.location.replace(httpsUrl("/login", state.site?.primary_host || ""));
    }, 1200);
  } catch (error) {
    throw error;
  } finally {
    setApplyBusy(false);
  }
}

async function bootstrap() {
  await applyTranslations(getLanguage());

  const access = await checkEntryAccess("onboarding");
  if (!access.allowed) {
    return;
  }

  restoreDraft();
  state.step = stepFromPath();

  await loadSetupStatus();
  const me = access.user || await ensureAuthIfNeeded();

  if (me) {
    state.user = me;
    state.adminDraft.username = me.username || me.id || "";
    state.adminDraft.email = me.email || "";
  }

  if (state.siteDraft.host) {
    state.tlsMode = state.siteDraft.tlsMode || "letsencrypt";
    state.site = buildSitePayload();
    state.upstream = buildUpstreamPayload(state.site.id);
    state.certificateID = buildCertificateID(state.site.id);
  }

  configureStage1();
  configureStage2();

  document.querySelectorAll(".waf-password-toggle").forEach((button) => {
    syncPasswordToggle(button, false);
    button.addEventListener("click", () => togglePasswordField(button));
  });

  document.getElementById("admin-password").addEventListener("input", (event) => {
    state.adminDraft.password = event.target.value;
    persistDraft();
  });

  document.getElementById("admin-username").addEventListener("input", (event) => {
    state.adminDraft.username = event.target.value.trim();
    persistDraft();
  });

  document.getElementById("admin-email").addEventListener("input", (event) => {
    state.adminDraft.email = event.target.value.trim();
    persistDraft();
  });

  document.getElementById("site-host").addEventListener("input", (event) => {
    state.siteDraft.host = event.target.value.trim().toLowerCase();
    persistDraft();
  });

  document.querySelectorAll('input[name="tls-mode"]').forEach((node) => {
    node.addEventListener("change", () => {
      state.siteDraft.tlsMode = currentTLSMode();
      syncStage2ACMEFields();
      persistDraft();
    });
  });
  document.getElementById("onboarding-acme-email")?.addEventListener("input", (event) => {
    state.siteDraft.accountEmail = event.target.value.trim();
    persistDraft();
  });
  document.getElementById("onboarding-acme-challenge")?.addEventListener("change", (event) => {
    state.siteDraft.challengeType = event.target.value;
    syncStage2ACMEFields();
    persistDraft();
  });
  document.getElementById("onboarding-dns-provider")?.addEventListener("change", (event) => {
    state.siteDraft.dnsProvider = event.target.value;
    persistDraft();
  });
  document.getElementById("onboarding-dns-provider-env")?.addEventListener("input", (event) => {
    state.siteDraft.dnsProviderEnv = event.target.value;
    persistDraft();
  });
  document.getElementById("onboarding-dns-propagation")?.addEventListener("input", (event) => {
    state.siteDraft.dnsPropagationSeconds = Number(event.target.value || 0) || 0;
    persistDraft();
  });
  document.getElementById("onboarding-zerossl-eab-kid")?.addEventListener("input", (event) => {
    state.siteDraft.zeroSSLEABKID = event.target.value.trim();
    persistDraft();
  });
  document.getElementById("onboarding-zerossl-eab-hmac")?.addEventListener("input", (event) => {
    state.siteDraft.zeroSSLEABHMAC = event.target.value.trim();
    persistDraft();
  });

  window.addEventListener("popstate", () => {
    state.step = stepFromPath();
    updateStepUI();
  });

  document.getElementById("onboarding-back").addEventListener("click", () => {
    if (state.applyInFlight) {
      return;
    }
    if (state.step > 1) {
      state.step -= 1;
      updateStepUI();
    }
  });

  document.getElementById("onboarding-next").addEventListener("click", async () => {
    clearFeedback();
    if (state.step === 1) {
      try {
        await bootstrapAdminIfNeeded();
        state.step = 2;
        updateStepUI();
      } catch (error) {
        setError(document.getElementById("onboarding-feedback"), error.message || t("onboarding.error.adminCreate"));
      }
      return;
    }

    if (state.step === 2) {
      try {
        state.siteDraft.host = document.getElementById("site-host").value.trim().toLowerCase() || detectSiteHost();
        state.siteDraft.tlsMode = currentTLSMode();
        state.siteDraft.accountEmail = document.getElementById("onboarding-acme-email")?.value.trim() || "";
        state.siteDraft.challengeType = document.getElementById("onboarding-acme-challenge")?.value || "http-01";
        state.siteDraft.dnsProvider = document.getElementById("onboarding-dns-provider")?.value || "cloudflare";
        state.siteDraft.dnsProviderEnv = document.getElementById("onboarding-dns-provider-env")?.value || "";
        state.siteDraft.dnsPropagationSeconds = Number(document.getElementById("onboarding-dns-propagation")?.value || 0) || 0;
        state.siteDraft.zeroSSLEABKID = document.getElementById("onboarding-zerossl-eab-kid")?.value.trim() || "";
        state.siteDraft.zeroSSLEABHMAC = document.getElementById("onboarding-zerossl-eab-hmac")?.value.trim() || "";
        if (state.siteDraft.tlsMode !== "self-signed" && !state.siteDraft.accountEmail) {
          throw new Error(t("onboarding.validation.acmeEmailRequired"));
        }
        if (state.siteDraft.tlsMode === "zerossl" && (!state.siteDraft.zeroSSLEABKID || !state.siteDraft.zeroSSLEABHMAC)) {
          throw new Error(t("onboarding.validation.zerosslEabRequired"));
        }
        if (state.siteDraft.tlsMode !== "self-signed" && state.siteDraft.challengeType === "dns-01") {
          if (!state.siteDraft.dnsProvider) {
            throw new Error(t("onboarding.validation.dnsProviderRequired"));
          }
          const env = parseKeyValueLines(state.siteDraft.dnsProviderEnv);
          if (state.siteDraft.dnsProvider === "cloudflare" && !env.CLOUDFLARE_API_TOKEN && !env.CF_API_TOKEN) {
            throw new Error(t("onboarding.validation.cloudflareTokenRequired"));
          }
        }
        state.tlsMode = state.siteDraft.tlsMode;
        state.site = buildSitePayload();
        state.upstream = buildUpstreamPayload(state.site.id);
        state.certificateID = buildCertificateID(state.site.id);
        renderSummary();
        state.step = 3;
        updateStepUI();
      } catch (error) {
        setError(document.getElementById("onboarding-feedback"), error.message || t("onboarding.error.siteTlsCreate"));
      }
    }
  });

  document.getElementById("onboarding-apply").addEventListener("click", async () => {
    clearFeedback();
    try {
      await runApply();
    } catch (error) {
      setError(document.getElementById("onboarding-feedback"), error.message || t("onboarding.error.apply"));
    }
  });

  if (state.step === 3 && state.site) {
    renderSummary();
  }

  updateStepUI();
}

bootstrap();
