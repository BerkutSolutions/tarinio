import { expect, type Locator, type Page, type Response } from "@playwright/test";

const ACTION_TIMEOUT = 30_000;
const NAVIGATION_TIMEOUT = 60_000;

export async function gotoWithNetworkRetry(page: Page, path: string) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 4; attempt++) {
    try {
      return await page.goto(path, { waitUntil: "domcontentloaded", timeout: NAVIGATION_TIMEOUT / 2 });
    } catch (error) {
      lastError = error;
      const message = String((error as Error)?.message || error);
      const transientPreDocumentFailure = /ERR_(TIMED_OUT|TOO_MANY_RETRIES|CONNECTION_RESET|CONNECTION_REFUSED|CONNECTION_CLOSED|NETWORK_CHANGED)/i.test(message) ||
        /(?:TimeoutError:\s*)?page\.goto: Timeout \d+ms exceeded/i.test(message) ||
        /is interrupted by another navigation to "chrome-error:\/\/chromewebdata\/"/i.test(message);
      if (!transientPreDocumentFailure || attempt === 3) throw error;
      await page.waitForTimeout(250 * (attempt + 1));
    }
  }
  throw lastError;
}

export async function openPage(page: Page, path: string, ready: string | Locator) {
  const target = typeof ready === "string" ? page.locator(ready) : ready;
  let lastError: unknown;
  for (let attempt = 0; attempt < 2; attempt++) {
    await gotoWithNetworkRetry(page, path);
    try {
      await expect(target).toBeVisible({ timeout: ACTION_TIMEOUT });
      return target;
    } catch (error) {
      lastError = error;
      if (attempt > 0) throw error;
      await page.waitForTimeout(250);
    }
  }
  throw lastError;
}

export function waitForAPIResponse(
  page: Page,
  method: string,
  path: string,
  payload?: (value: unknown) => boolean,
): Promise<Response> {
  return page.waitForResponse((response) => {
    const url = new URL(response.url());
    if (response.request().method() !== method || url.pathname !== path) return false;
    if (!payload) return true;
    try {
      return payload(response.request().postDataJSON());
    } catch {
      return false;
    }
  }, { timeout: ACTION_TIMEOUT });
}

export async function waitForToast(page: Page, text?: string | RegExp) {
  const toast = page.locator(".app-toast").last();
  await expect(toast).toBeVisible({ timeout: ACTION_TIMEOUT });
  if (text) await expect(toast).toContainText(text);
  return toast;
}

export async function waitForModal(modal: Locator, state: "open" | "closed" = "open") {
  if (state === "open") await expect(modal).toBeVisible({ timeout: ACTION_TIMEOUT });
  else await expect(modal).toBeHidden({ timeout: ACTION_TIMEOUT });
}

export async function waitForLoading(loading: Locator, state: "start" | "finish" = "finish") {
  if (state === "start") await expect(loading).toBeVisible({ timeout: ACTION_TIMEOUT });
  else await expect(loading).toBeHidden({ timeout: ACTION_TIMEOUT });
}

export async function waitForError(page: Page, text?: string | RegExp) {
  const error = page.locator(".alert:not([hidden]), .waf-error:not([hidden])").last();
  await expect(error).toBeVisible({ timeout: ACTION_TIMEOUT });
  if (text) await expect(error).toContainText(text);
  return error;
}

export async function waitForStableDOM(anchor: Locator, quietMilliseconds = 250) {
  await expect(anchor).toBeVisible({ timeout: ACTION_TIMEOUT });
  await anchor.evaluate((node, quiet) => new Promise<void>((resolve) => {
    const root = node.closest(".app-page-mount") || node.parentElement || document.body;
    let timer = window.setTimeout(done, quiet);
    const observer = new MutationObserver(() => {
      window.clearTimeout(timer);
      timer = window.setTimeout(done, quiet);
    });
    function done() {
      observer.disconnect();
      resolve();
    }
    observer.observe(root, { childList: true, subtree: true });
  }), quietMilliseconds);
}
