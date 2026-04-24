const defaultLanguage = "en";
const languageChangedEvent = "app:language-changed";
const languageStorageKey = "waf.language";
const dictionaries = new Map();
const supportedLanguages = ["en", "ru", "de", "sr", "zh"];
const languageCatalog = [
  { id: "en", label: "English" },
  { id: "ru", label: "Русский" },
  { id: "de", label: "Deutsch" },
  { id: "sr", label: "Српски" },
  { id: "zh", label: "中文" },
];
let currentLanguage = loadInitialLanguage();

function normalizeLanguage(language) {
  const value = String(language || "").trim().toLowerCase();
  if (!value) {
    return defaultLanguage;
  }
  const primary = value.split(/[-_]/)[0];
  return supportedLanguages.includes(primary) ? primary : defaultLanguage;
}

function detectBrowserLanguage() {
  const candidates = [];
  try {
    if (Array.isArray(navigator.languages)) {
      candidates.push(...navigator.languages);
    }
    if (navigator.language) {
      candidates.push(navigator.language);
    }
  } catch {
    return defaultLanguage;
  }
  for (const candidate of candidates) {
    const normalized = normalizeLanguage(candidate);
    if (supportedLanguages.includes(normalized)) {
      return normalized;
    }
  }
  return defaultLanguage;
}

export function getBrowserLanguage() {
  return detectBrowserLanguage();
}

function loadStoredLanguage() {
  try {
    return normalizeLanguage(window.localStorage.getItem(languageStorageKey) || "");
  } catch {
    return "";
  }
}

function loadInitialLanguage() {
  const stored = loadStoredLanguage();
  return stored || detectBrowserLanguage();
}

function persistLanguage(language) {
  try {
    window.localStorage.setItem(languageStorageKey, normalizeLanguage(language));
  } catch {
    // ignore storage failures
  }
}

function interpolate(template, params = {}) {
  return String(template).replace(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g, (match, key) => {
    return Object.prototype.hasOwnProperty.call(params, key) ? String(params[key]) : match;
  });
}

async function loadDictionary(language) {
  const normalized = normalizeLanguage(language);
  if (dictionaries.has(normalized)) {
    return dictionaries.get(normalized);
  }

  try {
    const url = new URL(`../i18n/${normalized}.json`, import.meta.url);
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`failed to load ${normalized}: ${response.status}`);
    }
    const dictionary = await response.json();
    dictionaries.set(normalized, dictionary);
    return dictionary;
  } catch (error) {
    console.warn("i18n dictionary load failed", normalized, error);
    const fallback = dictionaries.get(defaultLanguage);
    if (fallback) {
      dictionaries.set(normalized, fallback);
      return fallback;
    }
    const empty = {};
    dictionaries.set(normalized, empty);
    return empty;
  }
}

function translateFromDictionary(language, key, params = {}) {
  const normalized = normalizeLanguage(language);
  const primary = dictionaries.get(normalized) || {};
  const fallback = dictionaries.get(defaultLanguage) || {};
  const value = primary[key] ?? fallback[key] ?? key;
  return interpolate(value, params);
}

export function getLanguage() {
  return normalizeLanguage(currentLanguage);
}

export function availableLanguages() {
  return languageCatalog.map((item) => ({ ...item }));
}

export async function preloadTranslations(language = getLanguage()) {
  await loadDictionary(defaultLanguage);
  const normalized = normalizeLanguage(language);
  if (normalized !== defaultLanguage) {
    await loadDictionary(normalized);
  }
}

export function t(key, params = {}) {
  return translateFromDictionary(getLanguage(), key, params);
}

export async function applyTranslations(language = getLanguage()) {
  const normalized = normalizeLanguage(language);
  await preloadTranslations(normalized);

  document.querySelectorAll("[data-i18n]").forEach((element) => {
    element.textContent = translateFromDictionary(normalized, element.dataset.i18n);
  });

  document.querySelectorAll("[data-i18n-placeholder]").forEach((element) => {
    element.setAttribute("placeholder", translateFromDictionary(normalized, element.dataset.i18nPlaceholder));
  });

  document.querySelectorAll("[data-i18n-title]").forEach((element) => {
    element.setAttribute("title", translateFromDictionary(normalized, element.dataset.i18nTitle));
  });

  document.querySelectorAll("[data-i18n-aria-label]").forEach((element) => {
    element.setAttribute("aria-label", translateFromDictionary(normalized, element.dataset.i18nAriaLabel));
  });

  const switcher = document.getElementById("language-switcher");
  if (switcher) {
    switcher.value = normalized;
  }

  document.documentElement.lang = normalized;
}

export async function setLanguage(language) {
  const normalized = normalizeLanguage(language);
  const previous = getLanguage();
  currentLanguage = normalized;
  persistLanguage(normalized);
  await applyTranslations(normalized);
  if (previous !== normalized) {
    window.dispatchEvent(new CustomEvent(languageChangedEvent, { detail: { language: normalized } }));
  }
}

