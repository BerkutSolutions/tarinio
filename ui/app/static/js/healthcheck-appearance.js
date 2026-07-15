const themes = new Set(["variant-1", "variant-2", "variant-3", "variant-4", "variant-5"]);
const defaultTheme = "variant-1";

const group = (id, title, countID, body) => `
  <section class="hc-group hc-${id}">
    <button type="button" class="hc-toggle" data-hc-toggle="hc-${id}">
      <span class="healthcheck-step-icon hc-toggle-icon" aria-hidden="true" data-status="pending"></span>
      <span id="${countID}" class="hc-toggle-count">0/0</span>
      <span class="hc-toggle-title">${title}</span><span class="hc-chevron" aria-hidden="true"></span>
    </button>
    <div id="hc-${id}" class="hc-panel" hidden>${body}</div>
  </section>`;

const groups = () => ({
  checks: group("checks", "Проверки", "hc-checks-count", `<ul id="healthcheck-steps" class="hc-list"></ul>`),
  errors: group("errors", "Ошибки", "hc-errors-count", `<div id="hc-errors-summary" class="hc-summary"></div><ul id="healthcheck-errors" class="hc-list"></ul>`),
  compat: group("compat", "Совместимость", "hc-compat-count", `<div id="hc-compat-summary" class="hc-summary"></div><ul id="healthcheck-compat" class="hc-list"></ul>`),
});

function template(theme) {
  const parts = groups();
  const actions = `<div class="healthcheck-actions hc-page-actions"><button id="healthcheck-refresh" class="btn ghost" type="button">Обновить</button><button id="healthcheck-continue" class="btn primary" type="button">Перейти в панель</button></div>`;
  const alert = `<div id="healthcheck-error" class="alert" hidden></div>`;
  switch (theme) {
    case "variant-2":
      return `<section class="hc-shell"><header class="hc-brand"><img src="/static/logo800x300.png" alt="TARINIO"><span class="hc-status">Control room online</span></header><div class="hc-main"><section class="hc-hero"><div><h1>Контрольная комната</h1><p>Сводка готовности управления, API и защитных сервисов.</p></div><div class="hc-metric"><strong id="hc-live-checks">0/0</strong>проверок</div><div class="hc-metric"><strong id="hc-live-errors">0</strong>ошибок</div><div class="hc-metric"><strong id="hc-live-compat">—</strong>совместимость</div></section><div class="hc-checks-column">${parts.checks}${alert}</div><aside class="hc-side"><section class="hc-side-card">${parts.errors}</section><section class="hc-side-card">${parts.compat}</section>${actions}</aside></div></section>`;
    case "variant-3":
      return `<section class="hc-shell"><header class="hc-brand"><img src="/static/logo800x300.png" alt="TARINIO"></header><div class="hc-main"><section class="hc-assurance"><div class="ring" id="hc-live-ready">0</div><h1>Контур готовности</h1><p>Проверка защищённой сессии и сервисов управления.</p></section><section class="hc-progress">${parts.checks}</section>${parts.compat}<aside class="hc-console"><div class="hc-console-bar"><span class="hc-console-indicator" aria-hidden="true"></span><span id="hc-live-errors">0</span></div><div id="hc-errors-summary" class="hc-summary"></div><ul id="healthcheck-errors" class="hc-list"></ul></aside>${alert}${actions}</div></section>`;
    case "variant-4":
      return `<section class="hc-shell"><header class="hc-brand"><img src="/static/logo800x300.png" alt="TARINIO"><span class="hc-status">Runtime healthy</span></header><div class="hc-main"><header class="hc-heading"><div><h1>Проверка системы</h1><p>Диагностика сервисной консоли и доступных API.</p></div><span class="hc-summary">Management access / health scan</span></header>${parts.checks}${parts.errors}${parts.compat}${alert}${actions}</div></section>`;
    case "variant-5":
      return `<section class="hc-shell"><header class="hc-brand"><img src="/static/logo800x300.png" alt="TARINIO"><span class="hc-status">Live status board</span></header><div class="hc-main"><header class="hc-board-title"><h1>Статус системы</h1><span id="hc-live-updated">ПРОВЕРКА ЗАПУЩЕНА</span></header><section class="hc-health-grid"><article class="hc-tile"><small>Проверки</small><strong id="hc-live-checks">0/0</strong><em>● Живой результат</em></article><article class="hc-tile"><small>Ошибки</small><strong id="hc-live-errors">0</strong><em>● Журнал контейнеров</em></article><article class="hc-tile"><small>Совместимость</small><strong id="hc-live-compat">—</strong><em>● API-контракт</em></article></section><section class="hc-log" id="hc-live-log">[WAIT] Запуск живых проверок…</section><section class="hc-groups">${parts.checks}${parts.errors}${parts.compat}</section>${alert}${actions}</div></section>`;
    default:
      return `<section class="hc-shell"><header class="hc-brand"><img src="/static/logo800x300.png" alt="TARINIO"><span class="hc-status">Runtime healthy</span></header><div class="hc-main"><header class="hc-heading"><div><h1>Проверка системы</h1><p>Контроль готовности защищённой панели.</p></div><span class="hc-summary">Контур управления</span></header><div class="hc-layout"><aside class="hc-rail"><span>СЕССИЯ</span><strong id="hc-live-ready">0</strong><span>ПРОВЕРОК</span></aside><div class="hc-groups">${parts.checks}${parts.errors}${parts.compat}</div></div>${alert}${actions}</div></section>`;
  }
}

async function selectedTheme() {
  const requested = new URLSearchParams(window.location.search).get("appearance");
  if (themes.has(requested)) return requested;
  try {
    const response = await fetch("/api/public/healthcheck-appearance", { credentials: "same-origin", cache: "no-store" });
    const payload = response.ok ? await response.json() : null;
    const theme = String(payload?.healthcheck_appearance || "").trim();
    if (themes.has(theme)) return theme;
  } catch {
    // The working healthcheck must still be available with its default theme.
  }
  return defaultTheme;
}

export async function applyHealthcheckAppearance() {
  const theme = await selectedTheme();
  document.body.className = `healthcheck-page healthcheck-theme-${theme.slice(-1)}`;
  const root = document.getElementById("healthcheck-app");
  if (root) {
    root.innerHTML = template(theme);
  }
  return theme;
}

export function updateHealthcheckThemeSummary({ checksDone, checksTotal, errors, compatFailed, message }) {
  const checks = `${checksDone}/${checksTotal}`;
  document.querySelectorAll("#hc-live-checks").forEach((item) => { item.textContent = checks; });
  document.querySelectorAll("#hc-live-ready").forEach((item) => { item.textContent = String(checksDone); });
  document.querySelectorAll("#hc-live-errors").forEach((item) => { item.textContent = String(errors); });
  document.querySelectorAll("#hc-live-compat").forEach((item) => { item.textContent = compatFailed ? "WARN" : "OK"; });
  const updated = document.getElementById("hc-live-updated");
  if (updated) updated.textContent = `ОБНОВЛЕНО ${new Date().toLocaleTimeString()}`;
  const log = document.getElementById("hc-live-log");
  if (log && message) {
    const time = new Date().toLocaleTimeString();
    const lines = log.textContent.split("\n").filter(Boolean);
    lines.push(`[${time}] ${message}`);
    log.textContent = lines.slice(-12).join("\n");
    log.scrollTop = log.scrollHeight;
  }
}
