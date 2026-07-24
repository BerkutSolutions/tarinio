import { expect, test } from "@playwright/test";
import { gotoWithNetworkRetry, openPage, waitForAPIResponse, waitForError, waitForLoading, waitForModal, waitForStableDOM, waitForToast } from "../support/waits";

test("infrastructure.wait-helpers", async ({ page }) => {
  await page.setContent(`
    <main class="app-page-mount">
      <button id="anchor">ready</button>
      <div id="loading">loading</div>
      <div id="modal" hidden>modal</div>
      <div class="app-toast" hidden>saved</div>
      <div class="alert" hidden>failed</div>
    </main>
  `);
  await page.locator("#loading").evaluate((node) => window.setTimeout(() => { node.hidden = true; }, 20));
  await waitForLoading(page.locator("#loading"));
  await page.locator("#modal").evaluate((node) => { node.hidden = false; });
  await waitForModal(page.locator("#modal"));
  await page.locator(".app-toast").evaluate((node) => { node.hidden = false; });
  await waitForToast(page, "saved");
  await page.locator(".alert").evaluate((node) => { node.hidden = false; });
  await waitForError(page, "failed");
  await page.locator("#anchor").evaluate((node) => window.setTimeout(() => node.after(document.createElement("span")), 20));
  await waitForStableDOM(page.locator("#anchor"), 50);
  await expect(page.locator("#anchor")).toBeVisible();
});

test("infrastructure.bounded-navigation-retry", async ({ page }) => {
  let attempts = 0;
  await page.route("**/retry-page", async (route) => {
    attempts++;
    if (attempts === 1) await route.abort("connectionreset");
    else await route.fulfill({ contentType: "text/html", body: "<main id='retry-ready'>ready</main>" });
  });
  await gotoWithNetworkRetry(page, "https://e2e-management.test/retry-page");
  await expect(page.locator("#retry-ready")).toBeVisible();
  expect(attempts).toBe(2);
});

test("infrastructure.navigation-and-api-waits", async ({ page }) => {
  await page.route("**/fixture-page", (route) => route.fulfill({ contentType: "text/html", body: "<button id='ready'>ready</button><button id='save'>save</button>" }));
  await page.route("**/api/fixture", async (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ saved: true }) }));
  await openPage(page, "https://e2e-management.test/fixture-page", "#ready");
  await page.locator("#save").evaluate((button) => button.addEventListener("click", () => { void fetch("/api/fixture", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ value: "expected" }) }); }));
  const responsePromise = waitForAPIResponse(page, "PUT", "/api/fixture", (payload) => (payload as { value?: string })?.value === "expected");
  await page.locator("#save").click();
  expect((await responsePromise).ok()).toBeTruthy();
});
