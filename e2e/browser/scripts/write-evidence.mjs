import fs from "node:fs";
import path from "node:path";
import { validateStrictResults } from "./strict-results-lib.mjs";

const [reportArg, outputArg, suite = "browser"] = process.argv.slice(2);
if (!reportArg || !outputArg) throw new Error("usage: write-evidence.mjs <results.json> <output-dir> [suite]");
const reportPath = path.resolve(reportArg);
const outputDir = path.resolve(outputArg);
const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
const failures = validateStrictResults(report);
if (failures.length) throw new Error(`cannot publish invalid browser evidence: ${failures.join(", ")}`);

const tests = [];
const collect = (suites = []) => {
  for (const group of suites) {
    for (const spec of group.specs || []) {
      for (const test of spec.tests || []) {
        tests.push({ title: spec.title, project: test.projectName, status: test.status, results: test.results?.map((item) => ({ status: item.status, retry: item.retry, duration: item.duration })) || [] });
      }
    }
    collect(group.suites || []);
  }
};
collect(report.suites || []);
const evidence = {
  schema_version: 1,
  suite,
  generated_at: new Date().toISOString(),
  commit: process.env.CI_COMMIT_SHA || "local",
  pipeline_url: process.env.CI_PIPELINE_URL || "local",
  status: "passed",
  summary: { pass: Number(report.stats.expected), fail: 0, skip: 0 },
  browser: { flaky: 0, retries: 0, duration_ms: Number(report.stats.duration || 0), tests },
};
fs.mkdirSync(outputDir, { recursive: true });
fs.writeFileSync(path.join(outputDir, "e2e-evidence.json"), JSON.stringify(evidence, null, 2) + "\n");
fs.writeFileSync(path.join(outputDir, "e2e-evidence.md"), `# Browser E2E evidence\n\n- Suite: \`${suite}\`\n- Passed: ${evidence.summary.pass}\n- Failed: 0\n- Skipped: 0\n- Flaky: 0\n- Retries: 0\n`);
