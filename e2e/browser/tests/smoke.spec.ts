import { expect, test } from "../fixtures/auth";

const routes: Array<[string, string]> = [
  ["dashboard", "/dashboard"], ["services", "/services"], ["requests", "/requests"],
  ["bans", "/bans"], ["revisions", "/revisions"], ["anti-ddos", "/anti-ddos"],
  ["owasp-crs", "/owasp-crs"], ["tls", "/tls"], ["administration", "/administration"],
  ["events", "/events"], ["activity", "/activity"], ["settings", "/settings"],
];

for (const [name, route] of routes) {
  test("smoke." + name, async ({ authenticatedPage: page }) => {
    await page.goto(route, { waitUntil: "domcontentloaded", timeout: 60000 });
    await expect(page.locator("#content-area")).toBeVisible();
    await expect(page.locator("body")).not.toContainText("ServicesStableFacadeLoadError");
  });
}
