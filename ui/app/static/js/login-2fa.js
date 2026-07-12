import { api } from "./api.js";
import { getBrowserLanguage, getLanguage, setLanguage, t } from "./i18n.js";
import { buildChallengeUrl, checkEntryAccess, onboardingUrl, secureAppUrl } from "./guard.js";
import { BerkutWebAuthn } from "./webauthn.js";

const challengeStorageKey = "waf_login_2fa_challenge_id";
const nextStorageKey = "waf_login_next";
const challengeRefreshMarkerKey = "waf_login_2fa_page_challenge_refresh_at";
const challengeRefreshCooldownMs = 8000;

async function nextLocation() {
  const setup = await api.get("/api/setup/status");
  return setup.needs_bootstrap || !setup.has_active_revision
    ? onboardingUrl("/onboarding/user-creation")
    : secureAppUrl("/healthcheck");
}

function getStoredChallenge() {
  return String(sessionStorage.getItem(challengeStorageKey) || "").trim();
}

function clearChallenge() {
  sessionStorage.removeItem(challengeStorageKey);
}

function showError(message) {
  const box = document.getElementById("login2fa-alert");
  if (!box) {
    return;
  }
  const key = String(message || "").trim();
  const translated = key ? t(key) : "";
  box.hidden = false;
  box.textContent = translated && translated !== key ? translated : key || t("login2fa.errorFailed");
}

function clearError() {
  const box = document.getElementById("login2fa-alert");
  if (!box) {
    return;
  }
  box.hidden = true;
  box.textContent = "";
}

function webAuthnSupported() {
  return BerkutWebAuthn && BerkutWebAuthn.supported && BerkutWebAuthn.supported();
}

function shouldRefreshChallenge(status) {
  const code = Number(status || 0);
  return code === 403 || code === 429;
}

function redirectToChallenge() {
  try {
    const lastAttempt = Number(sessionStorage.getItem(challengeRefreshMarkerKey) || 0);
    if (Number.isFinite(lastAttempt) && Date.now() - lastAttempt < challengeRefreshCooldownMs) {
      return false;
    }
    sessionStorage.setItem(challengeRefreshMarkerKey, String(Date.now()));
  } catch {
    // best-effort only
  }
  window.location.replace(buildChallengeUrl(window.location.pathname || "/login/2fa", window.location.search || ""));
  return true;
}

async function ensureChallengeAccess() {
  let response;
  try {
    response = await fetch("/api/auth/me", {
      credentials: "include",
      headers: { Accept: "application/json" },
    });
  } catch {
    return true;
  }
  if (shouldRefreshChallenge(response.status)) {
    redirectToChallenge();
    return false;
  }
  return true;
}

function installChallengeRecheckHandlers() {
  const handleReturnToPage = () => {
    if (document.visibilityState === "hidden") {
      return;
    }
    void ensureChallengeAccess();
  };
  document.addEventListener("visibilitychange", handleReturnToPage);
  window.addEventListener("focus", handleReturnToPage);
}

async function confirmWithCode() {
  const challengeID = getStoredChallenge();
  if (!challengeID) {
    throw new Error("auth.2fa.challengeMissing");
  }

  const codeEl = document.getElementById("twofa-code");
  const useRecovery = !!document.getElementById("twofa-use-recovery")?.checked;
  const value = String(codeEl?.value || "").trim();
  if (!value) {
    throw new Error("auth.2fa.codeRequired");
  }

  await api.post("/api/auth/login/2fa", {
    challenge_id: challengeID,
    code: useRecovery ? "" : value,
    recovery_code: useRecovery ? value : "",
  });
}

async function confirmWithPasskey() {
  const challengeID = getStoredChallenge();
  if (!challengeID) {
    throw new Error("auth.2fa.challengeMissing");
  }

  const begin = await api.post("/api/auth/login/2fa/passkey/begin", {
    challenge_id: challengeID,
  });
  const options = begin?.options;
  const webauthnChallengeID = String(begin?.webauthn_challenge_id || "").trim();
  if (!options || !webauthnChallengeID) {
    throw new Error("auth.passkeys.notAvailable");
  }

  const publicKey = BerkutWebAuthn.toPublicKeyRequestOptions(options);
  const credential = await navigator.credentials.get({ publicKey });
  const payload = BerkutWebAuthn.credentialToJSON(credential);

  await api.post("/api/auth/login/2fa/passkey/finish", {
    challenge_id: challengeID,
    webauthn_challenge_id: webauthnChallengeID,
    credential: payload,
  });
}

async function bootstrap() {
  await setLanguage(getBrowserLanguage());

  try {
    const access = await checkEntryAccess("login-2fa");
    if (!access.allowed) {
      return;
    }
  } catch {
    // keep page usable
  }
  installChallengeRecheckHandlers();
  if (!await ensureChallengeAccess()) {
    return;
  }

  const switcher = document.getElementById("language-switcher");
  if (switcher) {
    switcher.value = getLanguage();
    switcher.addEventListener("change", async (event) => {
      await setLanguage(event.target.value);
    });
  }

  const form = document.getElementById("login2fa-form");
  if (!form) {
    return;
  }

  if (!getStoredChallenge()) {
    showError(t("auth.2fa.challengeMissing"));
  }

  document.getElementById("login2fa-back")?.addEventListener("click", () => {
    clearChallenge();
    window.location.href = secureAppUrl("/login");
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    clearError();
    const submit = document.getElementById("login2fa-submit");
    if (submit) {
      submit.disabled = true;
    }
    try {
      if (!await ensureChallengeAccess()) {
        return;
      }
      await confirmWithCode();
      clearChallenge();
      window.location.href = await nextLocation();
    } catch (error) {
      if (shouldRefreshChallenge(error?.status) && redirectToChallenge()) {
        return;
      }
      if (error?.status === 401) {
        showError(t("login2fa.errorInvalidCode"));
      } else {
        showError(error?.message || t("login2fa.errorFailed"));
      }
    } finally {
      if (submit) {
        submit.disabled = false;
      }
    }
  });

  const passkeyBtn = document.getElementById("login2fa-passkey-btn");
  if (!passkeyBtn) {
    return;
  }

  passkeyBtn.hidden = !webAuthnSupported();
  passkeyBtn.addEventListener("click", async () => {
    clearError();
    passkeyBtn.disabled = true;
    try {
      if (!await ensureChallengeAccess()) {
        return;
      }
      await confirmWithPasskey();
      clearChallenge();
      window.location.href = await nextLocation();
    } catch (error) {
      const key = BerkutWebAuthn.errorKey ? BerkutWebAuthn.errorKey(error) : "";
      if (shouldRefreshChallenge(error?.status) && redirectToChallenge()) {
        return;
      }
      if (Number(error?.status || 0) === 404) {
        showError(t("auth.passkeys.notAvailable"));
      } else {
        showError(key || error?.message || t("login2fa.errorFailed"));
      }
    } finally {
      passkeyBtn.disabled = false;
    }
  });
}

bootstrap();
