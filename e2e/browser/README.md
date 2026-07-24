# WAF browser E2E

This suite is mutation-capable and refuses to start unless WAF_E2E_DISPOSABLE=1 and the base URL is localhost/127.0.0.1.

Run from e2e/browser: npm.cmd install, set WAF_E2E_DISPOSABLE=1, set WAF_BROWSER_BASE_URL=https://e2e-management.test:10443, optionally set WAF_BROWSER_EXECUTABLE, then npx.cmd playwright test. Chromium maps e2e-management.test to 127.0.0.1 inside the disposable runner.

Use scripts/run-e2e-tests.ps1 to create the disposable stack before running the browser suite. Do not point this runner at shared development data.
