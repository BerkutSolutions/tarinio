const CRITICAL_STYLE_RETRY_DELAYS_MS = [250, 750];
const CRITICAL_STYLE_LOAD_TIMEOUT_MS = 30_000;

function wait(milliseconds) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

function isLoaded(link) {
  return Boolean(link?.sheet);
}

function reloadStylesheet(link, attempt) {
  return new Promise((resolve) => {
    let settled = false;
    const finish = (loaded) => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timer);
      link.removeEventListener("load", onLoad);
      link.removeEventListener("error", onError);
      resolve(loaded);
    };
    const onLoad = () => finish(isLoaded(link));
    const onError = () => finish(false);
    const timer = window.setTimeout(() => finish(isLoaded(link)), CRITICAL_STYLE_LOAD_TIMEOUT_MS);
    link.addEventListener("load", onLoad);
    link.addEventListener("error", onError);

    const retryURL = new URL(link.href, window.location.href);
    retryURL.searchParams.set("style_retry", `${Date.now()}-${attempt + 1}`);
    link.href = retryURL.toString();
  });
}

async function ensureStylesheet(link) {
  if (isLoaded(link)) return;
  for (let attempt = 0; attempt < CRITICAL_STYLE_RETRY_DELAYS_MS.length; attempt += 1) {
    await wait(CRITICAL_STYLE_RETRY_DELAYS_MS[attempt]);
    if (await reloadStylesheet(link, attempt)) return;
  }
  throw new Error(`critical stylesheet failed to load: ${link.getAttribute("href") || "unknown"}`);
}

export async function ensureCriticalStyles() {
  const links = Array.from(document.querySelectorAll('link[rel="stylesheet"][data-critical-style]'));
  await Promise.all(links.map(ensureStylesheet));
}
