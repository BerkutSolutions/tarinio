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
  if (!memoryPreferences) {
    const stored = readStoredPreferences();
    memoryPreferences = {
      ...defaults,
      ...(stored || {}),
      timeZone: detectTimeZone(),
      language: String(stored?.language || getLanguage() || defaults.language),
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

export function formatDateTimeInZone(value, timeZone) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: timeZone || loadPreferences().timeZone || defaults.timeZone,
    }).format(date);
  } catch {
    return date.toLocaleString();
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
