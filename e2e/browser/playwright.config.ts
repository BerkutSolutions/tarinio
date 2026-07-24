import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const baseURL = process.env.WAF_BROWSER_BASE_URL || "https://e2e-management.test:10443";
const executablePath = process.env.WAF_BROWSER_EXECUTABLE || "";
const configDirectory = path.dirname(fileURLToPath(import.meta.url));
const storageStatePath = path.resolve(configDirectory, ".auth/storage-state.json");
const outputDirectory = path.resolve(process.env.WAF_BROWSER_OUTPUT_DIR || path.join(configDirectory, "test-results"));
const resultsFile = path.resolve(process.env.WAF_BROWSER_RESULTS_FILE || path.join(outputDirectory, "results.json"));
const junitFile = process.env.WAF_BROWSER_JUNIT_FILE ? path.resolve(process.env.WAF_BROWSER_JUNIT_FILE) : "";
const workers = Number.parseInt(process.env.WAF_BROWSER_WORKERS || "1", 10);
const selectedSpecs = String(process.env.WAF_BROWSER_SPECS || "")
  .split(/\r?\n/)
  .map((value) => value.trim().replace(/\\/g, "/"))
  .filter(Boolean);
const selectedTestMatch = selectedSpecs.length
  ? selectedSpecs.map((spec) => new RegExp(`${path.basename(spec).replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`))
  : undefined;
if (process.env.WAF_E2E_DISPOSABLE !== "1") {
  throw new Error("Browser E2E requires WAF_E2E_DISPOSABLE=1 and a disposable stack.");
}
if (!/^https?:\/\/(127\.0\.0\.1|localhost|e2e-management\.test)(:\d+)?$/i.test(baseURL)) {
  throw new Error("Refusing non-local browser E2E base URL: " + baseURL);
}
if (!Number.isInteger(workers) || workers < 1) throw new Error("WAF_BROWSER_WORKERS must be a positive integer");

export default defineConfig({
  testDir: "./tests",
  timeout: 10 * 60 * 1000,
  navigationTimeout: 60000,
  expect: { timeout: 15000 },
  fullyParallel: false,
  retries: 0,
  forbidOnly: true,
  workers,
  outputDir: outputDirectory,
  reporter: [["list"], ["json", { outputFile: resultsFile }], ...(junitFile ? [["junit", { outputFile: junitFile }] as const] : [])],
  use: {
    baseURL,
    ignoreHTTPSErrors: true,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    launchOptions: {
      args: ["--host-resolver-rules=MAP e2e-management.test 127.0.0.1", "--no-first-run", "--no-default-browser-check", "--disable-session-crashed-bubble"],
      ...(executablePath ? { executablePath } : {}),
    },
  },
  projects: [
    { name: "setup", testMatch: /auth\.setup\.ts/ },
    {
      name: "desktop",
      dependencies: ["setup"],
      ...(selectedTestMatch ? { testMatch: selectedTestMatch } : {}),
      use: { ...devices["Desktop Chrome"], storageState: storageStatePath },
    },
    {
      name: "mobile",
      dependencies: ["setup"],
      ...(selectedTestMatch ? { testMatch: selectedTestMatch } : {}),
      use: { ...devices["Pixel 7"], storageState: storageStatePath },
    },
  ],
});
