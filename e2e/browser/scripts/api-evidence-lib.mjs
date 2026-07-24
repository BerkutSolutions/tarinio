const STATUS_PRIORITY = { pass: 1, skip: 2, fail: 3 };

function normalizeStatus(status) {
  if (status === "passed") return "pass";
  if (status === "skipped") return "skip";
  if (status === "failed") return "fail";
  return ["pass", "skip", "fail"].includes(status) ? status : "";
}

function recordStatus(results, name, status) {
  const normalized = normalizeStatus(status);
  if (!name || !normalized) return;
  const previous = results.get(name);
  if (!previous || STATUS_PRIORITY[normalized] > STATUS_PRIORITY[previous]) {
    results.set(name, normalized);
  }
}

export function parseGoTestJSONLines(source) {
  const results = new Map();
  for (const line of String(source).split(/\r?\n/)) {
    if (!line.trim()) continue;
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue;
    }
    recordStatus(results, event.Test, event.Action);
  }
  return results;
}

export function parseGoEvidenceReport(report) {
  const results = new Map();
  for (const item of report?.tests || []) recordStatus(results, item?.name, item?.status);
  return results;
}

export function mergeGoEvidence(target, source) {
  for (const [name, status] of source) recordStatus(target, name, status);
  return target;
}

export function validateGoExecution(goResults) {
  const failed = [];
  const skipped = [];
  for (const [name, status] of goResults) {
    if (status === "fail") failed.push(name);
    if (status === "skip") skipped.push(name);
  }
  return { failed, skipped };
}

export function apiCoverageByTab(registry, goResults) {
  const rows = registry.filter((item) => item.layer === "api-runtime");
  return [...new Set(rows.map((item) => item.tab))].map((tab) => {
    const workflows = rows.filter((item) => item.tab === tab);
    const passed = workflows.filter((item) =>
      item.goTests?.length && item.goTests.every((name) => goResults.get(name) === "pass")
    ).length;
    return {
      tab,
      passed,
      total: workflows.length,
      percent: workflows.length ? Math.round((passed / workflows.length) * 100) : 0,
    };
  });
}

export function validateApiEvidence(registry, goResults) {
  const missingMappings = [];
  const missingTests = [];
  const failedTests = [];
  const skippedTests = [];
  for (const workflow of registry.filter((item) => item.layer === "api-runtime")) {
    if (!workflow.goTests?.length) {
      missingMappings.push(workflow.id);
      continue;
    }
    for (const name of workflow.goTests) {
      const status = goResults.get(name);
      if (!status) missingTests.push(`${workflow.id}:${name}`);
      else if (status === "fail") failedTests.push(`${workflow.id}:${name}`);
      else if (status === "skip") skippedTests.push(`${workflow.id}:${name}`);
    }
  }
  return { missingMappings, missingTests, failedTests, skippedTests };
}
