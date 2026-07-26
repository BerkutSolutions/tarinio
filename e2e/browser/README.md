# WAF browser E2E

This suite is mutation-capable and refuses to start unless `WAF_E2E_DISPOSABLE=1` and the base URL is loopback or the dedicated `e2e-management.test` hostname.

`scripts/run-e2e-tests.sh` attaches Playwright to the current project's isolated Compose network, where only that stack's runtime owns the `e2e-management.test` DNS alias. For direct local execution from `e2e/browser`, install dependencies, set `WAF_E2E_DISPOSABLE=1` and `WAF_BROWSER_BASE_URL=https://e2e-management.test:10443`, optionally set `WAF_BROWSER_EXECUTABLE`, then run `npx.cmd playwright test`; the default Chromium resolver rule maps the hostname to loopback for this local mode.

Use scripts/run-e2e-tests.ps1 to create the disposable stack before running the browser suite. Do not point this runner at shared development data.
