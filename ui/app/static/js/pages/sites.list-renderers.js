import { escapeHtml } from "../ui.js";
import { normalizeArray } from "./sites.routing-merge.js";
import { normalizeStringArray } from "./sites.normalize.js";
import {
  BAD_BEHAVIOR_STATUS_OPTIONS,
  buildGeoCatalogFallback,
  regionDisplayLabel,
  regionDisplayName,
} from "./sites.geo-lists.js";

export function renderListEditor(field, label, values, placeholder = "", options = {}) {
  const safeValues = normalizeStringArray(values);
  const fullWidth = options.full !== false;
  const emptyLabel = options.emptyLabel || "No values yet";
  const fieldClass = fullWidth ? "waf-field full" : "waf-field";
  const presets = Array.isArray(options.presets) ? options.presets : [];
  const selectedTemplates = normalizeStringArray(options.selectedTemplates);
  const ctx = options.ctx || null;
  const quickTemplateLabel = ctx?.t ? ctx.t("sites.easy.listTemplates.quick") : "Quick templates";
  const selectedTemplateLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: selectedTemplates.length }) : `Selected: ${selectedTemplates.length}`;
  const selectedValueLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: safeValues.length }) : `Selected: ${safeValues.length}`;
  const selectedItemsLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: selectedTemplates.length + safeValues.length }) : `Selected: ${selectedTemplates.length + safeValues.length}`;
  const selectedTemplateSet = new Set(selectedTemplates);
  const availablePresets = presets.filter((preset) => {
    const presetID = String(preset?.id || "").trim();
    return presetID && !selectedTemplateSet.has(presetID);
  });
  const presetByID = new Map(presets.map((preset) => [String(preset?.id || "").trim(), preset]));
  const resolvePresetLabel = (preset) => String(ctx?.t && preset?.labelKey ? ctx.t(preset.labelKey) : (preset?.label || preset?.id || ""));
  return `
    <div class="${fieldClass} waf-list-editor" data-list-field="${escapeHtml(field)}">
      <label>${escapeHtml(label)}</label>
      <div class="waf-inline">
        <input id="list-input-${escapeHtml(field)}" placeholder="${escapeHtml(placeholder)}">
        <button class="btn ghost btn-sm" type="button" data-list-add="${escapeHtml(field)}">+</button>
      </div>
      ${presets.length ? `
        <div class="waf-template-picker">
          <details class="waf-status-dropdown">
            <summary>${escapeHtml(`${quickTemplateLabel} (${selectedTemplateLabel})`)}</summary>
            <div class="waf-status-options">
              ${availablePresets.map((preset) => `
                <button class="btn ghost btn-sm waf-template-option" type="button" data-list-template-apply="${escapeHtml(field)}" data-list-template-id="${escapeHtml(String(preset.id || ""))}">${escapeHtml(resolvePresetLabel(preset))}</button>
              `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
            </div>
          </details>
          <details class="waf-status-dropdown waf-list-selected-dropdown">
            <summary>${escapeHtml(`${ctx?.t ? ctx.t("sites.easy.listTemplates.add") : "Added"} (${selectedItemsLabel})`)}</summary>
            <div class="waf-status-options waf-list-selected-options">
              ${selectedTemplates.map((presetID) => {
                const preset = presetByID.get(presetID);
                if (!preset) {
                  return "";
                }
                return `
                  <div class="waf-list-selected-item">
                    <span class="waf-list-selected-value">${escapeHtml(resolvePresetLabel(preset))}</span>
                    <button class="waf-list-remove" type="button" data-list-template-remove="${escapeHtml(field)}" data-list-template-id="${escapeHtml(presetID)}">x</button>
                  </div>
                `;
              }).join("")}
              ${safeValues.map((value, index) => `
                <div class="waf-list-selected-item">
                  <span class="waf-list-selected-value">${escapeHtml(value)}</span>
                  <button class="waf-list-remove" type="button" data-list-remove="${escapeHtml(field)}" data-list-index="${index}">x</button>
                </div>
              `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
            </div>
            <div class="waf-note">${escapeHtml(selectedValueLabel)}</div>
          </details>
        </div>
      ` : ""}
      ${presets.length ? "" : `
        <div class="waf-inline">
          ${safeValues.map((value, index) => `
            <span class="badge badge-neutral">
              ${escapeHtml(value)}
              <button class="waf-list-remove" type="button" data-list-remove="${escapeHtml(field)}" data-list-index="${index}">x</button>
            </span>
          `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
        </div>
      `}
    </div>
  `;
}

export function renderCountryEditor(field, label, values, catalog, options = {}) {
  const safeValues = normalizeStringArray(values);
  const catalogOptions = normalizeStringArray(catalog).length ? normalizeStringArray(catalog) : buildGeoCatalogFallback();
  const fullWidth = options.full !== false;
  const emptyLabel = options.emptyLabel || "No values yet";
  const fieldClass = fullWidth ? "waf-field full" : "waf-field";
  const search = String(options.search || "").trim().toLowerCase();
  const selected = new Set(safeValues);
  const ctx = options.ctx || null;
  const selectedCountLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: safeValues.length }) : `Selected: ${safeValues.length}`;
  const searchPlaceholder = ctx?.t ? ctx.t("sites.easy.geo.searchPlaceholder") : "Search country or code";
  const filteredOptions = catalogOptions.filter((value) => {
    if (!search) {
      return true;
    }
    const haystack = `${value} ${regionDisplayLabel(value)} ${regionDisplayName(value)}`.toLowerCase();
    return haystack.includes(search);
  });
  const visibleOptions = filteredOptions.length ? filteredOptions : catalogOptions.filter((value) => selected.has(value));
  return `
    <div class="${fieldClass}">
      <label>${escapeHtml(label)}</label>
      <details class="waf-status-dropdown waf-country-picker" open>
        <summary>${escapeHtml(selectedCountLabel)}</summary>
        <input id="country-search-${escapeHtml(field)}" class="waf-country-search" placeholder="${escapeHtml(searchPlaceholder)}" value="${escapeHtml(search)}">
        <div class="waf-status-options waf-country-options">
          ${visibleOptions.map((value) => `
            <label class="waf-checkbox waf-status-option waf-country-option">
              <input type="checkbox" data-country-toggle="${escapeHtml(field)}" data-country-value="${escapeHtml(value)}"${selected.has(value) ? " checked" : ""}>
              <span>${escapeHtml(value)}</span>
              <span class="waf-note waf-country-option-name">${escapeHtml(regionDisplayLabel(value))}</span>
            </label>
          `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
        </div>
      </details>
      <div class="waf-inline">
        ${safeValues.map((value, index) => `
          <span class="badge badge-neutral">
            ${escapeHtml(regionDisplayLabel(value))}
            <button class="waf-list-remove" type="button" data-list-remove="${escapeHtml(field)}" data-list-index="${index}">x</button>
          </span>
        `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
      </div>
    </div>
  `;
}

export function renderStatusCodesEditor(selectedCodes, ctx) {
  const selected = new Set(normalizeArray(selectedCodes).map((item) => Number(item)).filter((item) => Number.isInteger(item)));
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.badStatusCodes"))}</label>
      <details class="waf-status-dropdown">
        <summary>${escapeHtml(ctx.t("sites.easy.selectedCount", { count: selected.size }))}</summary>
        <div class="waf-note">${escapeHtml(ctx.t("sites.easy.badStatusCodesHint"))}</div>
        <div class="waf-status-options">
          ${BAD_BEHAVIOR_STATUS_OPTIONS.map(([code, text]) => `
            <label class="waf-checkbox waf-status-option">
              <input type="checkbox" data-bad-code="${code}"${selected.has(code) ? " checked" : ""}>
              <span>${escapeHtml(`${code} ${text}`)}</span>
            </label>
          `).join("")}
        </div>
      </details>
    </div>
  `;
}
