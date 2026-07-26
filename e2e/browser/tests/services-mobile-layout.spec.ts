import { expect, test } from "../fixtures/auth";
import { openPage } from "../support/waits";

const tabs = ["front", "upstream", "httpheaders", "traffic", "blocking", "antibot", "geo", "modsec", "websocket", "virtualpatches", "errorpages"];

test("services.editor-all-tabs-responsive-layout", async ({ authenticatedPage: page }) => {
  test.setTimeout(3 * 60_000);
  await openPage(page, "/services/new", page.locator("#service-editor-form"));
  for (const tab of tabs) {
    await page.locator(`[data-wizard-tab="${tab}"]`).click();
    const panel = page.locator(`[data-tab-panel="${tab}"]`);
    await expect(panel).toBeVisible();
    const geometry = await panel.evaluate((element) => {
      const viewportWidth = document.documentElement.clientWidth;
      const documentOverflow = document.documentElement.scrollWidth - viewportWidth;
      const panelOverflow = element.scrollWidth - element.clientWidth;
      const outside = Array.from(element.querySelectorAll("button, input, select, textarea"))
        .filter((control) => {
          const style = window.getComputedStyle(control);
          const rect = control.getBoundingClientRect();
          const intersectsViewportY = rect.bottom > 0 && rect.top < document.documentElement.clientHeight;
          return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && intersectsViewportY && (rect.left < -1 || rect.right > viewportWidth + 1);
        })
        .map((control) => control.id || control.getAttribute("data-ep-slug") || control.tagName);
      const widest = Array.from(element.querySelectorAll("*"))
        .map((node) => ({
          marker: node.id || node.getAttribute("data-list-field") || node.className || node.tagName,
          overflow: node.scrollWidth - node.clientWidth,
          width: node.scrollWidth,
        }))
        .filter((item) => item.overflow > 1)
        .sort((left, right) => right.overflow - left.overflow)
        .slice(0, 8);
      return { documentOverflow, panelOverflow, outside, widest };
    });
    expect(geometry.documentOverflow, `${tab} document overflow`).toBeLessThanOrEqual(1);
    expect(geometry.panelOverflow, `${tab} panel overflow: ${JSON.stringify(geometry.widest)}`).toBeLessThanOrEqual(1);
    expect(geometry.outside, `${tab} controls outside viewport`).toEqual([]);
  }
});
