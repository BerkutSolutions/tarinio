import { getLanguage } from "./i18n.js";

const preferencesStorageKey = "waf.preferences";
const defaults = {
  language: "en",
  timeZone: "Europe/Moscow",
  autoLogout: false,
};
let memoryPreferences = null;

function detectTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || defaults.timeZone;
  } catch {
    return defaults.timeZone;
  }
}

function readStoredPreferences() {
  try {
    const raw = window.localStorage.getItem(preferencesStorageKey);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function writeStoredPreferences(value) {
  try {
    window.localStorage.setItem(preferencesStorageKey, JSON.stringify(value));
  } catch {
    // ignore storage failures
  }
}

export function loadPreferences() {
  const stored = readStoredPreferences();
  const currentLanguage = String(getLanguage() || stored?.language || defaults.language);
  if (!memoryPreferences || memoryPreferences.language !== currentLanguage) {
    memoryPreferences = {
      ...defaults,
      ...(stored || {}),
      timeZone: detectTimeZone(),
      language: currentLanguage,
      autoLogout: Boolean(stored?.autoLogout),
    };
  }
  return { ...memoryPreferences };
}

export function savePreferences(next) {
  const merged = {
    ...loadPreferences(),
    ...(next || {}),
  };
  memoryPreferences = { ...merged };
  writeStoredPreferences(memoryPreferences);
  return merged;
}

const localeByLanguage = {
  ru: "ru-RU",
  en: "en-US",
  de: "de-DE",
  sr: "sr-Cyrl-RS",
  zh: "zh-CN",
};

function intlLocaleForLanguage(language) {
  const normalized = String(language || "").toLowerCase();
  return localeByLanguage[normalized] || normalized || "en-US";
}

export function formatDateTimeInZone(value, timeZone) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  const prefs = loadPreferences();
  const locale = intlLocaleForLanguage(prefs.language || defaults.language);
  try {
    return new Intl.DateTimeFormat(locale, {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: timeZone || prefs.timeZone || defaults.timeZone,
    }).format(date);
  } catch {
    return date.toLocaleString(locale);
  }
}

export function formatDateOnly(value) {
  if (!value) {
    return "-";
  }
  // Accept "YYYY-MM-DD" strings or Date objects
  const date = value instanceof Date ? value : new Date(String(value) + "T00:00:00Z");
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  const prefs = loadPreferences();
  const locale = intlLocaleForLanguage(prefs.language || defaults.language);
  try {
    return new Intl.DateTimeFormat(locale, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      timeZone: "UTC",
    }).format(date);
  } catch {
    return String(value);
  }
}

export function availableTimeZones() {
  if (typeof Intl?.supportedValuesOf === "function") {
    try {
      return Intl.supportedValuesOf("timeZone");
    } catch {
      return [detectTimeZone(), "UTC"];
    }
  }
  return [detectTimeZone(), "UTC"];
}
