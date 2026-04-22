import { api } from "./api.js";
import { applyTranslations, getLanguage, setLanguage, t } from "./i18n.js";
import { checkEntryAccess, onboardingUrl, secureAppUrl } from "./guard.js";
import { BerkutWebAuthn } from "./webauthn.js";

const challengeStorageKey = "waf_login_2fa_challenge_id";
const nextStorageKey = "waf_login_next";

async function nextLocation() {
  const setup = await api.get("/api/setup/status");
  return setup.needs_bootstrap || !setup.has_active_revision
    ? onboardingUrl("/onboarding/user-creation")
    : secureAppUrl("/healthcheck");
}

function currentNext() {
  const value = new URLSearchParams(window.location.search).get("next") || "";
  return String(value || secureAppUrl("/healthcheck")).trim();
}

function showError(message) {
  const box = document.getElementById("login-alert");
  if (!box) {
    return;
  }
  const key = String(message || "").trim();
  const translated = key ? t(key) : "";
  box.hidden = false;
  box.textContent = translated && translated !== key ? translated : key || t("login.errorFailed");
}

function reasonMessage(reason) {
  const key = String(reason || "").trim();
  if (!key) return "";
  if (key === "session_expired") return t("login.reason.sessionExpired");
  if (key === "session_missing") return t("login.reason.sessionMissing");
  if (key === "session_invalid") return t("login.reason.sessionInvalid");
  if (key === "session_check_failed") return t("login.reason.sessionCheckFailed");
  return "";
}

function showReasonFromQuery() {
  const params = new URLSearchParams(window.location.search);
  const message = reasonMessage(params.get("reason") || "");
  if (message) {
    showError(message);
  }
}

function clearError() {
  const box = document.getElementById("login-alert");
  if (!box) {
    return;
  }
  box.hidden = true;
  box.textContent = "";
}

function webAuthnSupported() {
  return BerkutWebAuthn && BerkutWebAuthn.supported && BerkutWebAuthn.supported();
}

async function tryPasskeyLogin(usernameRaw) {
  const username = String(usernameRaw || "").trim();
  const begin = await api.post("/api/auth/passkeys/login/begin", { username });
  const options = begin?.options;
  const challengeID = String(begin?.challenge_id || "").trim();
  if (!options || !challengeID) {
    throw new Error("auth.passkeys.notAvailable");
  }
  const publicKey = BerkutWebAuthn.toPublicKeyRequestOptions(options);
  const credential = await navigator.credentials.get({ publicKey });
  const payload = BerkutWebAuthn.credentialToJSON(credential);
  const finish = await api.post("/api/auth/passkeys/login/finish", {
    challenge_id: challengeID,
    credential: payload,
  });
  return finish;
}

async function bootstrap() {
  await applyTranslations(getLanguage());
  document.title = t("login.pageTitle");

  try {
    const access = await checkEntryAccess("login");
    if (!access.allowed) {
      return;
    }
  } catch {
    // keep login screen visible
  }
  showReasonFromQuery();

  const switcher = document.getElementById("language-switcher");
  if (switcher) {
    switcher.value = getLanguage();
    switcher.addEventListener("change", async (event) => {
      await setLanguage(event.target.value);
    });
  }

  const form = document.getElementById("login-form");
  if (!form) {
    return;
  }

  const passkeyBtn = document.getElementById("login-passkey-btn");
  const usernameInput = document.getElementById("username");

  if (passkeyBtn) {
    passkeyBtn.hidden = !webAuthnSupported();
    passkeyBtn.addEventListener("click", async () => {
      clearError();
      passkeyBtn.disabled = true;
      try {
        const result = await tryPasskeyLogin(usernameInput?.value || "");
        if (result && result.requires_2fa) {
          sessionStorage.setItem(challengeStorageKey, result.challenge_id || "");
          sessionStorage.setItem(nextStorageKey, currentNext());
          window.location.href = secureAppUrl("/login/2fa");
          return;
        }
        window.location.href = await nextLocation();
      } catch (error) {
        const errorKey = BerkutWebAuthn.errorKey ? BerkutWebAuthn.errorKey(error) : "";
        if (Number(error?.status || 0) === 404) {
          showError(t("auth.passkeys.notAvailable"));
        } else {
          showError(errorKey || error?.message || t("login.errorFailed"));
        }
      } finally {
        passkeyBtn.disabled = false;
      }
    });
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    clearError();
    const submit = document.getElementById("login-submit-btn");
    if (submit) {
      submit.disabled = true;
    }
    try {
      const result = await api.post("/api/auth/login", {
        username: usernameInput?.value,
        password: document.getElementById("password")?.value,
      });
      if (result && result.requires_2fa) {
        sessionStorage.setItem(challengeStorageKey, result.challenge_id || "");
        sessionStorage.setItem(nextStorageKey, currentNext());
        window.location.href = secureAppUrl("/login/2fa");
        return;
      }
      window.location.href = await nextLocation();
    } catch (error) {
      if (error?.status === 401) {
        showError(t("login.errorInvalidCredentials"));
      } else {
        showError(error?.message || t("login.errorFailed"));
      }
    } finally {
      if (submit) {
        submit.disabled = false;
      }
    }
  });
}

bootstrap();
