import { expect, test } from "../fixtures/auth";
import { openPage, waitForStableDOM } from "../support/waits";

test("settings.runtime-toggle-persistence-restore", async ({ authenticatedPage: page }) => {
  await openPage(page, "/settings/general", "#settings-runtime-save");
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  const readRuntime = async () => page.evaluate(async () => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch("/api/settings/runtime", { credentials: "include", signal: controller.signal, headers: { Accept: "application/json" } });
      return { status: response.status, body: await response.text() };
    } finally { window.clearTimeout(timer); }
  });
  const originalResult = await readRuntime();
  expect(originalResult.status).toBe(200);
  const original = Boolean(JSON.parse(originalResult.body)?.update_checks_enabled);
  const next = !original;
  const toggle = page.locator("#settings-updates-enabled");
  try {
    await toggle.setChecked(next);
    await page.locator("#settings-runtime-save").click();
    await expect.poll(async () => Boolean(JSON.parse((await readRuntime()).body)?.update_checks_enabled), { timeout: 30000 }).toBe(next);
    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("#settings-updates-enabled")).toBeChecked({ checked: next });
  } finally {
    await openPage(page, "/settings/general", "#settings-runtime-save");
    await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
    await page.locator("#settings-updates-enabled").setChecked(original);
    await page.locator("#settings-runtime-save").click();
    await expect.poll(async () => Boolean(JSON.parse((await readRuntime()).body)?.update_checks_enabled), { timeout: 30000 }).toBe(original);
  }
});

test("settings.language-all-locales-persistence-restore", async ({ authenticatedPage: page }) => {
  const languages = ["en", "ru", "de", "sr", "zh"];
  const waitForSettingsDOM = async () => {
    await waitForStableDOM(page.locator("#settings-language-save"));
  };
  await openPage(page, "/settings/general", "#settings-language-save");
  await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
  await waitForSettingsDOM();
  const readLanguage = async () => page.evaluate(async () => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch("/api/settings/runtime", { credentials: "include", signal: controller.signal, headers: { Accept: "application/json" } });
      return { status: response.status, body: await response.text() };
    } finally {
      window.clearTimeout(timer);
    }
  });
  const originalResult = await readLanguage();
  expect(originalResult.status).toBe(200);
  const original = String(JSON.parse(originalResult.body)?.language || "en");
  const orderedLanguages = [...languages.filter((language) => language !== original), original];
  try {
    for (const language of orderedLanguages) {
      await waitForSettingsDOM();
      await page.locator("#settings-language-select").selectOption(language);
      await expect(page.locator("#settings-language-select")).toHaveValue(language);
      await page.locator("#settings-language-save").click();
      await expect.poll(async () => String(JSON.parse((await readLanguage()).body)?.language), { timeout: 30000 }).toBe(language);
      await page.reload({ waitUntil: "domcontentloaded" });
      await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
      await expect(page.locator("#settings-language-select")).toHaveValue(language);
    }
  } finally {
    const current = String(JSON.parse((await readLanguage()).body)?.language || "en");
    if (current !== original) {
      await openPage(page, "/settings/general", "#settings-language-save");
      await expect(page.locator("#settings-page")).toHaveAttribute("data-runtime-ready", "true");
      await waitForSettingsDOM();
      await page.locator("#settings-language-select").selectOption(original);
      await page.locator("#settings-language-save").click();
      await expect.poll(async () => String(JSON.parse((await readLanguage()).body)?.language), { timeout: 30000 }).toBe(original);
    }
  }
});
