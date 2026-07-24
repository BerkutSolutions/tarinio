import { expect, test } from "../fixtures/auth";

const helpControls = [
  ["front", "service-front-main-help-btn", "service-front-main-help-modal"],
  ["front", "service-front-mtls-help-btn", "service-front-mtls-help-modal"],
  ["upstream", "service-upstream-chapter-help-btn", "service-upstream-chapter-help-modal"],
  ["upstream", "service-upstream-headers-help-btn", "service-upstream-headers-help-modal"],
  ["upstream", "service-upstream-mtls-help-btn", "service-upstreamMtls-chapter-help-modal"],
  ["httpheaders", "service-http-chapter-help-btn", "service-http-chapter-help-modal"],
  ["httpheaders", "service-headers-chapter-help-btn", "service-headers-chapter-help-modal"],
  ["blocking", "service-blocking-chapter-help-btn", "service-blocking-chapter-help-modal"],
  ["antibot", "service-antibot-chapter-help-btn", "service-antibot-chapter-help-modal"],
  ["antibot", "service-antibot-help-btn", "service-antibot-help-modal"],
  ["antibot", "service-auth-help-btn", "service-auth-help-modal"],
  ["geo", "service-geo-chapter-help-btn", "service-geo-chapter-help-modal"],
  ["modsec", "service-modsec-chapter-help-btn", "service-modsec-chapter-help-modal"],
  ["websocket", "service-websocket-chapter-help-btn", "service-websocket-chapter-help-modal"],
  ["virtualpatches", "service-virtualpatches-chapter-help-btn", "service-virtualpatches-chapter-help-modal"],
  ["traffic", "service-traffic-badbehavior-help-btn", "service-traffic-badbehavior-help-modal"],
  ["traffic", "service-traffic-limits-help-btn", "service-traffic-limits-help-modal"],
  ["traffic", "service-traffic-blacklist-help-btn", "service-traffic-blacklist-help-modal"],
  ["traffic", "service-traffic-allowlist-help-btn", "service-traffic-allowlist-help-modal"],
  ["traffic", "service-traffic-dnsbl-help-btn", "service-traffic-dnsbl-help-modal"],
];

test("services.editor-help-modals-all-controls", async ({ authenticatedPage: page }) => {
  test.setTimeout(5 * 60_000);
  page.setDefaultTimeout(15_000);
  await page.goto("/services/new", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("#service-editor-form")).toBeVisible();
  for (const [tab, buttonID, modalID] of helpControls) {
    const tabButton = page.locator(`[data-wizard-tab="${tab}"]`);
    await expect(tabButton, `${tab} wizard tab must be rendered`).toHaveCount(1);
    await tabButton.click();
    const button = page.locator(`#${buttonID}`);
    await expect(button, `${buttonID} must be rendered`).toHaveCount(1);
    await button.click();
    const modal = page.locator(`#${modalID}`);
    await expect(modal, `${modalID} must open`).toBeVisible();
    await expect(modal).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
    await button.click();
    await expect(modal).toBeVisible();
    await modal.locator("[data-help-close]").last().click();
    await expect(modal).toBeHidden();
  }
});
