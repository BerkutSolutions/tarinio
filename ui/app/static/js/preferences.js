import { getLanguage } from "./i18n.js";

const key = "waf_ui_prefs";

const defaults = {
  language: "ru",
  timeZone: "Europe/Moscow",
  autoLogout: false,
};

function detectTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || defaults.timeZone;
  } catch {
    return defaults.timeZone;
  }
}

export function loadPreferences() {
  try {
    const raw = window.localStorage.getItem(key);
    const parsed = raw ? JSON.parse(raw) : {};
    return {
      ...defaults,
      timeZone: detectTimeZone(),
      language: getLanguage(),
      ...parsed,
    };
  } catch {
    return {
      ...defaults,
      timeZone: detectTimeZone(),
      language: getLanguage(),
    };
  }
}

export function savePreferences(next) {
  const merged = {
    ...loadPreferences(),
    ...(next || {}),
  };
  window.localStorage.setItem(key, JSON.stringify(merged));
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
