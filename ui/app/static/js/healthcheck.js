import { api } from "./api.js";
import { checkEntryAccess, secureAppUrl } from "./guard.js";
import { applyTranslations, getLanguage, t } from "./i18n.js";

const MIN_STEP_MS = 150;
const CHECKS_CONCURRENCY = 4;
const PROBE_TIMEOUT_MS = 8000;
const PROBE_SLOW_MS = 2500;
const CONTRACT_VERSION = "2026-04-19-healthcheck-v3";

function todayDateKeyLocal() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

const TAB_PROBES = [
  { id: "tab.dashboard", titleKey: "healthcheck.tab.dashboard", path: "/api/dashboard/stats?probe=stats" },
  { id: "tab.sites", titleKey: "healthcheck.tab.sites", path: "/api/sites" },
  { id: "tab.revisions", titleKey: "healthcheck.tab.revisions", path: "/api/revisions" },
  { id: "tab.antiddos", titleKey: "healthcheck.tab.antiddos", path: "/api/anti-ddos/settings" },
  { id: "tab.owaspcrs", titleKey: "healthcheck.tab.owaspcrs", path: "/api/owasp-crs/status" },
  { id: "tab.tls", titleKey: "healthcheck.tab.tls", path: "/api/certificates" },
  { id: "tab.requests", titleKey: "healthcheck.tab.requests", path: () => `/api/dashboard/stats?probe=requests&day=${encodeURIComponent(todayDateKeyLocal())}` },
  { id: "tab.events", titleKey: "healthcheck.tab.events", path: "/api/dashboard/stats?probe=events" },
  { id: "tab.bans", titleKey: "healthcheck.tab.bans", path: "/api/sites" },
  { id: "tab.administration", titleKey: "healthcheck.tab.administration", path: "/api/administration/zero-trust/health" },
  { id: "tab.activity", titleKey: "healthcheck.tab.activity", path: "/api/audit?limit=1" },
  { id: "tab.settings", titleKey: "healthcheck.tab.settings", path: "/api/settings/runtime" },
  { id: "tab.profile", titleKey: "healthcheck.tab.profile", path: "/api/auth/me" },
];

const state = {
  checks: new Map(),
  compatItems: [],
  errorItems: [],
};

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function iconStatusFor(status) {
  const normalized = String(status || "").trim();
  if (normalized === "ok" || normalized === "done") return "done";
  if (normalized === "warning") return "warning";
  if (normalized === "running") return "running";
  if (normalized === "skipped") return "skipped";
  if (normalized === "failed" || normalized === "needs_reinit" || normalized === "broken") return "failed";
  if (normalized === "needs_attention") return "warning";
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
  if (error?.name === "AbortError") {
    return { status: "failed", desc: t("healthcheck.probe.timeout").replace("{ms}", String(PROBE_TIMEOUT_MS)) };
  }
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
  const targetPath = typeof path === "function" ? path() : path;
  const controller = new AbortController();
  const timeoutID = window.setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
  const startedAt = Date.now();
  try {
    await api.get(targetPath, { signal: controller.signal });
    const elapsed = Date.now() - startedAt;
    if (elapsed >= PROBE_SLOW_MS) {
      return {
        status: "needs_attention",
        desc: t("healthcheck.probe.slow")
          .replace("{ms}", String(elapsed))
          .replace("{threshold}", String(PROBE_SLOW_MS)),
      };
    }
    return { status: "done", desc: t("healthcheck.probe.ok") };
  } catch (error) {
    return normalizeProbeError(error);
  } finally {
    window.clearTimeout(timeoutID);
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
  let hasAttention = false;
  state.checks.forEach((value) => {
    const status = String(value.status || "");
    if (status === "done" || status === "failed" || status === "skipped" || status === "needs_attention") {
      done++;
    }
    if (status === "failed") hasFailed = true;
    if (status === "needs_attention") hasAttention = true;
    if (status === "running" || status === "pending") hasRunning = true;
  });
  setToggleCount("hc-checks-count", done, total);
  if (hasRunning) {
    setToggleStatus("hc-checks", "running");
  } else if (hasFailed) {
    setToggleStatus("hc-checks", "failed");
  } else if (hasAttention) {
    setToggleStatus("hc-checks", "skipped");
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

    const supportsFix = item?.supports_fix !== false;
    if (supportsFix && (status === "needs_attention" || status === "needs_reinit" || status === "broken")) {
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

function renderErrorIssues(items) {
  const list = document.getElementById("healthcheck-errors");
  const summary = document.getElementById("hc-errors-summary");
  if (!list) return;
  list.innerHTML = "";
  state.errorItems = Array.isArray(items) ? items : [];
  setToggleCount("hc-errors-count", state.errorItems.length, state.errorItems.length);

  if (summary) {
    summary.textContent = state.errorItems.length
      ? t("healthcheck.errors.summary").replace("{count}", String(state.errorItems.length))
      : t("healthcheck.errors.empty");
  }

  if (!state.errorItems.length) {
    setToggleStatus("hc-errors", "done");
    return;
  }

  let hasError = false;
  state.errorItems.forEach((item, index) => {
    const severity = String(item?.severity || "").toLowerCase() === "warning" ? "warning" : "error";
    const severityLabel = t(`events.severity.${severity}`);
    if (severity === "error") {
      hasError = true;
    }
    const li = document.createElement("li");
    li.className = "hc-item log-issue-item";
    li.dataset.itemId = `log-issue-${index}`;
    li.innerHTML = `
        <span class="healthcheck-step-icon" aria-hidden="true" data-status="${severity === "warning" ? "warning" : "failed"}"></span>
      <div class="hc-item-body">
        <div class="hc-item-title"></div>
        <div class="muted hc-item-sub"></div>
        <div class="hc-item-meta muted"></div>
      </div>
    `;
    const titleNode = li.querySelector(".hc-item-title");
    const subNode = li.querySelector(".hc-item-sub");
    const metaNode = li.querySelector(".hc-item-meta");
    if (titleNode) {
      titleNode.textContent = `${String(item?.container || "-")} | ${severityLabel}`;
    }
    if (subNode) {
      subNode.textContent = String(item?.sample_log || item?.normalized_log || "");
    }
    if (metaNode) {
      const parts = [
        `${t("healthcheck.errors.severity")}: ${severityLabel}`,
        `${t("healthcheck.errors.count")}: ${String(item?.count ?? 0)}`,
      ];
      if (item?.first_seen) {
        parts.push(`${t("healthcheck.errors.firstSeen")}: ${String(item.first_seen)}`);
      }
      if (item?.last_seen) {
        parts.push(`${t("healthcheck.errors.lastSeen")}: ${String(item.last_seen)}`);
      }
      metaNode.textContent = parts.join(" | ");
    }
    list.appendChild(li);
  });

  setToggleStatus("hc-errors", hasError ? "failed" : "warning");
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
    state.checks.set(spec.id, { status: result.status, desc: result.desc });
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

function buildProbeCompatIssues() {
  const issues = [];
  TAB_PROBES.forEach((probeSpec) => {
    const result = state.checks.get(probeSpec.id);
    const status = String(result?.status || "");
    const desc = String(result?.desc || "").trim();
    if (status !== "failed" && status !== "needs_attention") {
      return;
    }
    const title = t(probeSpec.titleKey);
    issues.push(`${title}: ${desc || status}`);
  });
  return issues;
}

async function loadCompat() {
  setToggleStatus("hc-compat", "running");
  try {
    const report = await api.get("/api/app/compat");
    const items = Array.isArray(report?.items) ? report.items.slice() : [];
    const issues = buildProbeCompatIssues();
    if (issues.length > 0) {
      const hasHardFailures = issues.some((line) => /401|404|502|503|504|timeout|unauthorized|not found|ошиб|недостаточно прав/i.test(line));
      items.push({
        module_id: "runtime-probes",
        title_i18n_key: "healthcheck.compat.runtimeProbes",
        status: hasHardFailures ? "broken" : "needs_attention",
        expected_schema_version: 1,
        applied_schema_version: 1,
        expected_behavior_version: 1,
        applied_behavior_version: 1,
        last_error: issues.join(" | "),
        supports_fix: false,
      });
    }
    renderCompat(items);
  } catch (error) {
    setError(String(error?.message || t("healthcheck.error.compat")));
    setToggleStatus("hc-compat", "failed");
  }
}

async function loadErrorIssues() {
  setToggleStatus("hc-errors", "running");
  try {
    const payload = await api.get("/api/dashboard/containers/issues");
    renderErrorIssues(Array.isArray(payload?.issues) ? payload.issues : []);
  } catch (error) {
    const list = document.getElementById("healthcheck-errors");
    const summary = document.getElementById("hc-errors-summary");
    if (list) {
      list.innerHTML = "";
      addRow(list, "hc-errors-load", t("healthcheck.section.errors"), String(error?.message || t("healthcheck.errors.loadError")), "failed");
    }
    if (summary) {
      summary.textContent = t("healthcheck.errors.loadError");
    }
    setToggleCount("hc-errors-count", 0, 0);
    setToggleStatus("hc-errors", "failed");
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
  await loadErrorIssues();
  await loadCompat();
}

bootstrap().catch((error) => {
  setError(String(error?.message || t("healthcheck.error.generic")));
});
