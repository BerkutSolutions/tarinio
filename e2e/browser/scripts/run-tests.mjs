import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const browserRoot = path.resolve(scriptDir, "..");
const playwrightCLI = path.join(browserRoot, "node_modules", "@playwright", "test", "cli.js");
const reportPath = process.env.WAF_BROWSER_RESULTS_FILE || "test-results/results.json";

const specs = process.argv.slice(2);
const selectedSpecs = specs.filter((argument) => /\.spec\.[cm]?[jt]sx?$/i.test(argument));
const options = specs.filter((argument) => !/\.spec\.[cm]?[jt]sx?$/i.test(argument));
if (selectedSpecs.length) console.log(`[browser-e2e] selected specs: ${selectedSpecs.join(", ")}`);
const playwright = spawnSync(process.execPath, [playwrightCLI, "test", ...options], {
  cwd: browserRoot,
  env: { ...process.env, WAF_BROWSER_SPECS: selectedSpecs.join("\n") },
  stdio: "inherit",
});
if (playwright.error) throw playwright.error;
if (playwright.status !== 0) process.exit(playwright.status ?? 1);

const verifier = spawnSync(process.execPath, [path.join(scriptDir, "verify-results.mjs"), reportPath], {
  cwd: browserRoot,
  env: process.env,
  stdio: "inherit",
});
if (verifier.error) throw verifier.error;
process.exit(verifier.status ?? 1);
