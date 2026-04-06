const defaultLanguage = "ru";
const languageChangedEvent = "app:language-changed";
const dictionaries = new Map();
let currentLanguage = defaultLanguage;

function normalizeLanguage(language) {
  return language === "en" ? "en" : defaultLanguage;
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
  await applyTranslations(normalized);
  if (previous !== normalized) {
    window.dispatchEvent(new CustomEvent(languageChangedEvent, { detail: { language: normalized } }));
  }
}
