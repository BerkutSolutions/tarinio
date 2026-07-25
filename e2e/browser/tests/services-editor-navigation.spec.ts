import { expect, test } from "../fixtures/auth";
import { e2eID } from "../support/isolation";
import { openPage } from "../support/waits";

const openNewServiceEditor = (page: import("@playwright/test").Page) =>
  openPage(page, "/services/new", page.locator("#service-editor-form #service-host"));

test("services.editor-draft-mode-parity-and-back-cancel", async ({ authenticatedPage: page }, testInfo) => {
  test.setTimeout(3 * 60_000);
  page.setDefaultTimeout(15_000);
  const firstID = e2eID(testInfo, "e2e-draft-a");
  const secondID = e2eID(testInfo, "e2e-draft-b");
  const listSiteIDs = async () => page.evaluate(async () => {
    const response = await fetch("/api/sites", { credentials: "include", headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`sites readback returned ${response.status}`);
    const payload = await response.json();
    const items = Array.isArray(payload) ? payload : (payload?.items || []);
    return items.map((item: { id?: string }) => String(item.id || ""));
  });

  await test.step("preserve draft across Easy and Raw", async () => {
    await openNewServiceEditor(page);
    await page.locator("#service-host").fill(`${firstID}.test`);
    await page.locator("#service-id").fill(firstID);
    await expect(page.locator("#service-editor-form")).toHaveAttribute("data-unsaved", "true");
    await page.locator('[data-wizard-tab="upstream"]').click();
    await expect(page.locator("#service-editor-form")).toHaveAttribute("data-unsaved", "true");
    await page.locator("#service-upstream-host").fill("upstream-echo");
    await page.locator('[data-mode-tab="raw"]').click();
    const raw = page.locator("#service-raw-env");
    await expect(raw).toContainText(firstID);
    await expect(raw).toContainText("upstream-echo");
    await raw.fill((await raw.inputValue()).replace(`${firstID}.test`, `${firstID}.updated.test`));
    await page.locator('[data-mode-tab="easy"]').click();
    await expect(page.locator("#service-editor-form")).toHaveAttribute("data-unsaved", "true");
    await expect(page.locator("#service-host")).toHaveValue(`${firstID}.updated.test`);
    await expect(page.locator("#service-id")).toHaveValue(firstID);
    await page.locator('[data-wizard-tab="upstream"]').click();
    await expect(page.locator("#service-upstream-host")).toHaveValue("upstream-echo");
  });

  await test.step("top Back cancels without API mutation", async () => {
    let cancelMessage = "";
    page.once("dialog", async (dialog) => { cancelMessage = dialog.message(); await dialog.dismiss(); });
    await page.locator("#service-back").click();
    expect(cancelMessage).not.toBe("");
    await expect(page).toHaveURL(/\/services\/new$/);
    await expect(page.locator("#service-id")).toHaveValue(firstID);
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#service-back").click();
    await expect(page).toHaveURL(/\/services$/);
    expect(await listSiteIDs()).not.toContain(firstID);
  });

  await test.step("bottom Back cancels without API mutation", async () => {
    await openNewServiceEditor(page);
    await page.locator("#service-host").fill(`${secondID}.test`);
    await page.locator("#service-id").fill(secondID);
    await page.locator('[data-wizard-tab="errorpages"]').click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#service-back-bottom").click();
    await expect(page).toHaveURL(/\/services$/);
    expect(await listSiteIDs()).not.toContain(secondID);
  });
});

test("services.editor-keyboard-labels-and-beforeunload", async ({ authenticatedPage: page }) => {
  await openNewServiceEditor(page);
  const host = page.locator("#service-host");
  await page.locator('label[for="service-host"]').click();
  await expect(host).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.locator("#service-id")).toBeFocused();
  await host.fill("unsaved-keyboard.test");
  const beforeUnloadPrevented = await page.evaluate(() => {
    const event = new Event("beforeunload", { cancelable: true });
    window.dispatchEvent(event);
    return event.defaultPrevented;
  });
  expect(beforeUnloadPrevented).toBe(true);
  let dialogMessage = "";
  page.once("dialog", async (dialog) => { dialogMessage = dialog.message(); await dialog.dismiss(); });
  await page.locator("#service-back").click();
  expect(dialogMessage).not.toBe("");
  await expect(page).toHaveURL(/\/services\/new$/);
  await expect(host).toHaveValue("unsaved-keyboard.test");
});
