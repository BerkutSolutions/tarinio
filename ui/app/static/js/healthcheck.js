import { api } from "./api.js";
import { checkEntryAccess, secureAppUrl } from "./guard.js";
import { applyTranslations, getLanguage, t } from "./i18n.js";

const MIN_STEP_MS = 150;
const CHECKS_CONCURRENCY = 4;
const CONTRACT_VERSION = "2026-04-06-healthcheck-v1";

const TAB_PROBES = [
  { id: "tab.dashboard", titleKey: "healthcheck.tab.dashboard", path: "/api/dashboard/stats" },
  { id: "tab.sites", titleKey: "healthcheck.tab.sites", path: "/api/sites" },
  { id: "tab.antiddos", titleKey: "healthcheck.tab.antiddos", path: "/api/anti-ddos/settings" },
  { id: "tab.owaspcrs", titleKey: "healthcheck.tab.owaspcrs", path: "/api/owasp-crs/status" },
  { id: "tab.tls", titleKey: "healthcheck.tab.tls", path: "/api/certificates" },
  { id: "tab.requests", titleKey: "healthcheck.tab.requests", path: "/api/requests" },
  { id: "tab.events", titleKey: "healthcheck.tab.events", path: "/api/events" },
  { id: "tab.bans", titleKey: "healthcheck.tab.bans", path: "/api/sites" },
  { id: "tab.administration", titleKey: "healthcheck.tab.administration", path: "/api/audit?limit=1" },
  { id: "tab.activity", titleKey: "healthcheck.tab.activity", path: "/api/audit?limit=1" },
  { id: "tab.settings", titleKey: "healthcheck.tab.settings", path: "/api/settings/runtime" },
  { id: "tab.profile", titleKey: "healthcheck.tab.profile", path: "/api/auth/me" },
];

const state = {
  checks: new Map(),
  compatItems: [],
};

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function iconStatusFor(status) {
  const normalized = String(status || "").trim();
  if (normalized === "ok" || normalized === "done") return "done";
  if (normalized === "running") return "running";
  if (normalized === "skipped") return "skipped";
  if (normalized === "failed" || normalized === "needs_reinit" || normalized === "broken") return "failed";
  if (normalized === "needs_attention") return "skipped";
  return "pending";
}

function setError(message) {
  const el = document.getElementById("healthcheck-error");
  if (!el) return;
  const text = String(message || "").trim();
  if (!text) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.hidden = false;
  el.textContent = text;
}

function setToggleCount(id, done, total) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = `${done}/${total}`;
}

function setToggleStatus(panelID, status) {
  const btn = document.querySelector(`[data-hc-toggle="${panelID}"]`);
  if (!btn) return;
  const icon = btn.querySelector(".hc-toggle-icon");
  if (!icon) return;
  icon.dataset.status = iconStatusFor(status);
}

function addRow(parent, id, title, subtitle, status) {
  const li = document.createElement("li");
  li.className = "hc-item";
  li.dataset.itemId = id;
  li.innerHTML = `
    <span class="healthcheck-step-icon" aria-hidden="true" data-status="${iconStatusFor(status)}"></span>
    <div class="hc-item-body">
      <div class="hc-item-title"></div>
      <div class="muted hc-item-sub"></div>
    </div>
  `;
  li.querySelector(".hc-item-title").textContent = String(title || "-");
  li.querySelector(".hc-item-sub").textContent = String(subtitle || "");
  parent.appendChild(li);
  return li;
}

function updateRow(row, subtitle, status) {
  if (!row) return;
  const icon = row.querySelector(".healthcheck-step-icon");
  if (icon) {
    icon.dataset.status = iconStatusFor(status);
  }
  const sub = row.querySelector(".hc-item-sub");
  if (sub) {
    sub.textContent = String(subtitle || "");
  }
}

function normalizeProbeError(error) {
  const status = Number(error?.status || 0);
  if (status === 403) {
    return { status: "skipped", desc: t("healthcheck.probe.forbidden") };
  }
  if (status === 404) {
    return { status: "failed", desc: t("healthcheck.probe.notFound") };
  }
  if (status === 401) {
    return { status: "failed", desc: t("healthcheck.probe.unauthorized") };
  }
  return { status: "failed", desc: String(error?.message || t("healthcheck.probe.failed")) };
}

async function probe(path) {
  try {
    await api.get(path);
    return { status: "done", desc: t("healthcheck.probe.ok") };
  } catch (error) {
    return normalizeProbeError(error);
  }
}

function buildChecks() {
  return [
    {
      id: "session",
      title: t("healthcheck.check.session"),
      run: async () => {
        try {
          const me = await api.get("/api/auth/me");
          const username = me?.username || me?.user?.username || "-";
          return {
            status: "done",
            desc: t("healthcheck.session.authorized").replace("{user}", String(username)),
          };
        } catch (error) {
          return normalizeProbeError(error);
        }
      },
    },
    {
      id: "compat-contract",
      title: t("healthcheck.check.contract"),
      run: async () => ({
        status: "done",
        desc: t("healthcheck.contract.version").replace("{version}", CONTRACT_VERSION),
      }),
    },
    ...TAB_PROBES.map((item) => ({
      id: item.id,
      title: t(item.titleKey),
      run: async () => probe(item.path),
    })),
  ];
}

function updateChecksHeader() {
  const total = state.checks.size;
  let done = 0;
  let hasFailed = false;
  let hasRunning = false;
  state.checks.forEach((value) => {
    const status = String(value.status || "");
    if (status === "done" || status === "failed" || status === "skipped") {
      done++;
    }
    if (status === "failed") hasFailed = true;
    if (status === "running" || status === "pending") hasRunning = true;
  });
  setToggleCount("hc-checks-count", done, total);
  if (hasRunning) {
    setToggleStatus("hc-checks", "running");
  } else if (hasFailed) {
    setToggleStatus("hc-checks", "failed");
  } else {
    setToggleStatus("hc-checks", "done");
  }
}

function detailsLine(item) {
  const schemaApplied = Number(item?.applied_schema_version || 0);
  const schemaExpected = Number(item?.expected_schema_version || 0);
  const behaviorApplied = Number(item?.applied_behavior_version || 0);
  const behaviorExpected = Number(item?.expected_behavior_version || 0);
  return t("healthcheck.compat.versions")
    .replace("{1}", String(schemaApplied))
    .replace("{2}", String(schemaExpected))
    .replace("{3}", String(behaviorApplied))
    .replace("{4}", String(behaviorExpected));
}

function compatTitle(item) {
  const key = String(item?.title_i18n_key || "").trim();
  if (key) {
    const translated = t(key);
    if (translated && translated !== key) {
      return translated;
    }
  }
  return String(item?.module_id || "-");
}

function renderCompat(items) {
  const list = document.getElementById("healthcheck-compat");
  const summary = document.getElementById("hc-compat-summary");
  if (!list) return;
  list.innerHTML = "";
  state.compatItems = Array.isArray(items) ? items : [];
  setToggleCount("hc-compat-count", state.compatItems.length, state.compatItems.length);

  let hasFailed = false;
  let hasAttention = false;
  state.compatItems.forEach((item) => {
    const status = String(item?.status || "");
    if (status === "failed" || status === "broken" || status === "needs_reinit") {
      hasFailed = true;
    }
    if (status === "needs_attention") {
      hasAttention = true;
    }
    const title = compatTitle(item);
    const details = detailsLine(item);
    const errorText = String(item?.last_error || "").trim();
    const subtitle = errorText
      ? `${t("healthcheck.compat.status")}: ${status}. ${t("healthcheck.compat.error")}: ${errorText}`
      : `${t("healthcheck.compat.status")}: ${status}. ${details}`;
    const moduleID = String(item?.module_id || "-");
    const row = addRow(list, moduleID, title, subtitle, status);

    if (status === "needs_attention" || status === "needs_reinit" || status === "broken") {
      const body = row.querySelector(".hc-item-body");
      if (!body) return;
      const actions = document.createElement("div");
      actions.className = "healthcheck-actions";
      const fixBtn = document.createElement("button");
      fixBtn.type = "button";
      fixBtn.className = "btn ghost";
      fixBtn.textContent = t("healthcheck.compat.fix");

      const resultLine = document.createElement("div");
      resultLine.className = "muted hc-item-sub";
      resultLine.textContent = "";

      fixBtn.addEventListener("click", async () => {
        fixBtn.disabled = true;
        resultLine.textContent = t("healthcheck.compat.fixRunning");
        try {
          const response = await api.post("/api/app/compat/fix", { module_id: moduleID });
          const fixedItem = response?.item;
          const fixedStatus = String(fixedItem?.status || "done");
          const fixedError = String(fixedItem?.last_error || "").trim();
          const icon = row.querySelector(".healthcheck-step-icon");
          if (icon) {
            icon.dataset.status = iconStatusFor(fixedStatus);
          }
          if (fixedStatus === "ok" || fixedStatus === "done") {
            resultLine.textContent = t("healthcheck.compat.fixSuccess");
          } else {
            resultLine.textContent = fixedError
              ? t("healthcheck.compat.fixErrorWithDetails").replace("{details}", fixedError)
              : t("healthcheck.compat.fixErrorWithStatus").replace("{status}", fixedStatus);
          }
          if (response?.report?.items) {
            renderCompat(response.report.items);
          }
        } catch (error) {
          const icon = row.querySelector(".healthcheck-step-icon");
          if (icon) {
            icon.dataset.status = iconStatusFor("failed");
          }
          resultLine.textContent = t("healthcheck.compat.fixErrorWithDetails").replace(
            "{details}",
            String(error?.message || t("healthcheck.compat.fixUnknownError"))
          );
        } finally {
          fixBtn.disabled = false;
        }
      });
      actions.appendChild(fixBtn);
      body.appendChild(actions);
      body.appendChild(resultLine);
    }
  });

  if (summary) {
    summary.textContent = t("healthcheck.compat.summary").replace("{count}", String(state.compatItems.length));
  }

  if (hasFailed) {
    setToggleStatus("hc-compat", "failed");
  } else if (hasAttention) {
    setToggleStatus("hc-compat", "skipped");
  } else if (state.compatItems.length > 0) {
    setToggleStatus("hc-compat", "done");
  } else {
    setToggleStatus("hc-compat", "pending");
  }
}

function bindAccordion() {
  document.querySelectorAll("[data-hc-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const panelID = btn.getAttribute("data-hc-toggle");
      const panel = panelID ? document.getElementById(panelID) : null;
      if (!panel) return;
      const opened = !panel.hidden;
      panel.hidden = opened;
      btn.dataset.open = opened ? "0" : "1";
    });
  });
}

async function runChecks() {
  const list = document.getElementById("healthcheck-steps");
  if (!list) return;
  const checks = buildChecks();
  list.innerHTML = "";
  state.checks.clear();

  const rows = new Map();
  checks.forEach((check) => {
    state.checks.set(check.id, { status: "pending" });
    rows.set(check.id, addRow(list, check.id, check.title, t("healthcheck.pending"), "pending"));
  });
  updateChecksHeader();

  const runOne = async (spec) => {
    const row = rows.get(spec.id);
    state.checks.set(spec.id, { status: "running" });
    updateRow(row, t("healthcheck.running"), "running");
    updateChecksHeader();

    const startedAt = Date.now();
    let result = { status: "failed", desc: t("healthcheck.probe.failed") };
    try {
      result = await spec.run();
    } catch (error) {
      result = { status: "failed", desc: String(error?.message || t("healthcheck.probe.failed")) };
    }
    const elapsed = Date.now() - startedAt;
    if (elapsed < MIN_STEP_MS) {
      await sleep(MIN_STEP_MS - elapsed);
    }
    state.checks.set(spec.id, { status: result.status });
    updateRow(row, result.desc, result.status);
    updateChecksHeader();
    if (spec.id === "session" && result.status !== "done") {
      setError(t("healthcheck.error.session"));
    }
  };

  const session = checks.find((item) => item.id === "session");
  if (session) {
    await runOne(session);
    if (state.checks.get("session")?.status !== "done") return;
  }

  const others = checks.filter((item) => item.id !== "session");
  let cursor = 0;
  const workers = Array.from({ length: Math.max(1, Math.min(CHECKS_CONCURRENCY, others.length || 1)) }, () =>
    (async () => {
      while (cursor < others.length) {
        const index = cursor;
        cursor += 1;
        const spec = others[index];
        if (!spec) continue;
        await runOne(spec);
      }
    })()
  );
  await Promise.all(workers);
}

async function loadCompat() {
  setToggleStatus("hc-compat", "running");
  try {
    const report = await api.get("/api/app/compat");
    renderCompat(report?.items || []);
  } catch (error) {
    setError(String(error?.message || t("healthcheck.error.compat")));
    setToggleStatus("hc-compat", "failed");
  }
}

async function bootstrap() {
  await applyTranslations(getLanguage());
  document.title = t("healthcheck.pageTitle");

  const access = await checkEntryAccess("app");
  if (!access.allowed) return;

  document.getElementById("healthcheck-continue")?.addEventListener("click", () => {
    window.location.href = secureAppUrl("/dashboard");
  });

  bindAccordion();

  const checksPanel = document.getElementById("hc-checks");
  if (checksPanel) checksPanel.hidden = false;
  document.querySelectorAll("[data-hc-toggle]").forEach((btn) => {
    const panelID = btn.getAttribute("data-hc-toggle");
    const panel = panelID ? document.getElementById(panelID) : null;
    if (!panel) return;
    btn.dataset.open = panel.hidden ? "0" : "1";
  });

  await runChecks();
  await loadCompat();
}

bootstrap().catch((error) => {
  setError(String(error?.message || t("healthcheck.error.generic")));
});
