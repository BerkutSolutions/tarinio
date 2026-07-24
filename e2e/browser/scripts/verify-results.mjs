import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { validateStrictResults } from "./strict-results-lib.mjs";

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const reportPath = path.resolve(process.argv[2] || process.env.WAF_BROWSER_RESULTS_FILE || path.join(root, "test-results", "results.json"));
if (!fs.existsSync(reportPath)) throw new Error(`Playwright JSON report is missing: ${reportPath}`);
const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
const failures = validateStrictResults(report);
if (failures.length > 0) throw new Error(`strict browser result gate failed: ${failures.join(", ")}`);
console.log(`strict browser result gate passed: ${report.stats.expected} passed, no skipped/flaky/retry`);
