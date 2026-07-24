import { readFile } from "node:fs/promises";
import { coverageByTab, coverageRegressions, validateRegistry } from "./coverage-lib.mjs";
import { validateStrictResults } from "./strict-results-lib.mjs";
import {
  apiCoverageByTab,
  mergeGoEvidence,
  parseGoEvidenceReport,
  parseGoTestJSONLines,
  validateApiEvidence,
  validateGoExecution,
} from "./api-evidence-lib.mjs";

const registrySource = await readFile(new URL("../registry/workflows.ts", import.meta.url), "utf8");
const registry = registrySource.split(/\r?\n/).map((line) => {
  const id = line.match(/id: "([^"]+)"/);
  const tab = line.match(/tab: "([^"]+)"/);
  const title = line.match(/title: "([^"]+)"/);
  const layer = line.match(/layer: "([^"]+)"/);
  const aliases = line.match(/testTitles: \[([^\]]*)\]/);
  const goTests = line.match(/goTests: \[([^\]]*)\]/);
  return id && tab && title && layer ? {
    id: id[1], tab: tab[1], title: title[1], layer: layer[1],
    testTitles: aliases ? [...aliases[1].matchAll(/"([^"]+)"/g)].map((item) => item[1]) : [],
    goTests: goTests ? [...goTests[1].matchAll(/"([^"]+)"/g)].map((item) => item[1]) : [],
  } : null;
}).filter(Boolean);
if (!registry.length) {
  throw new Error("Workflow registry is empty or has an unsupported format");
}
const baseline = JSON.parse(await readFile(new URL("../registry/baseline.json", import.meta.url), "utf8"));
const registryValidation = validateRegistry(registry.map((item) => item.id), baseline.workflowIds || []);
if (registryValidation.duplicateIds.length || registryValidation.missingIds.length || registryValidation.unexpectedIds.length) {
  throw new Error("registry baseline mismatch: " + JSON.stringify(registryValidation));
}
const specs = [];
function collect(suites) {
  for (const suite of suites || []) {
    for (const spec of suite.specs || []) specs.push(spec);
    collect(suite.suites);
  }
}
const browserReportPaths = [];
const apiEvidencePaths = [];
for (let index = 2; index < process.argv.length; index += 1) {
  if (process.argv[index] === "--api-evidence") {
    const evidencePath = process.argv[index + 1];
    if (!evidencePath || evidencePath.startsWith("--")) throw new Error("--api-evidence requires a path");
    apiEvidencePaths.push(evidencePath);
    index += 1;
  } else {
    browserReportPaths.push(process.argv[index]);
  }
}
if (!browserReportPaths.length) browserReportPaths.push(process.env.WAF_BROWSER_RESULTS_FILE || "test-results/results.json");
for (const reportPath of browserReportPaths) {
  const result = JSON.parse(await readFile(reportPath, "utf8"));
  const strictFailures = validateStrictResults(result);
  if (strictFailures.length) throw new Error(`invalid browser evidence ${reportPath}: ${strictFailures.join(", ")}`);
  collect(result.suites);
}
const passedTitles = new Set(specs.filter((spec) => spec.ok).map((spec) => String(spec.title || "")));
const browserRegistry = registry.filter((item) => item.layer === "browser-ui");
const coverageRows = coverageByTab(registry, passedTitles);
for (const row of coverageRows) console.log("coverage " + row.tab + ": " + row.passed + "/" + row.total + " (" + row.percent + "%)");
const regressions = coverageRegressions(coverageRows, baseline.minimumBrowserCoverageByTab || {});
if (regressions.length) throw new Error("browser coverage regression: " + JSON.stringify(regressions));
const apiRows = registry.filter((item) => item.layer === "api-runtime");
if (apiRows.length) {
  if (!apiEvidencePaths.length) throw new Error("api-runtime registry requires at least one --api-evidence Go result");
  const goResults = new Map();
  for (const evidencePath of apiEvidencePaths) {
    const source = await readFile(evidencePath, "utf8");
    let parsed;
    try { parsed = JSON.parse(source); } catch { parsed = null; }
    const evidence = parsed?.tests ? parseGoEvidenceReport(parsed) : parseGoTestJSONLines(source);
    const execution = validateGoExecution(evidence);
    if (execution.failed.length || execution.skipped.length ||
        (parsed?.tests && (parsed.status !== "passed" || Number(parsed.summary?.fail || 0) || Number(parsed.summary?.skip || 0)))) {
      throw new Error(`invalid Go execution evidence ${evidencePath}: ${JSON.stringify(execution)}`);
    }
    mergeGoEvidence(goResults, evidence);
  }
  const apiValidation = validateApiEvidence(registry, goResults);
  if (Object.values(apiValidation).some((items) => items.length)) {
    throw new Error("invalid api-runtime evidence: " + JSON.stringify(apiValidation));
  }
  const apiCoverageRows = apiCoverageByTab(registry, goResults);
  for (const row of apiCoverageRows) console.log("api-runtime coverage " + row.tab + ": " + row.passed + "/" + row.total + " (" + row.percent + "%)");
  const apiRegressions = coverageRegressions(apiCoverageRows, baseline.minimumApiCoverageByTab || {});
  if (apiRegressions.length) throw new Error("api-runtime coverage regression: " + JSON.stringify(apiRegressions));
}
