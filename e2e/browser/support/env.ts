export function requiredE2EEnv(name: string) {
  const value = String(process.env[name] || "").trim();
  if (!value) throw new Error(`${name} is required for browser E2E`);
  return value;
}

export function runtimeBaseURL() {
  const value = requiredE2EEnv("WAF_E2E_RUNTIME_URL");
  const url = new URL(value);
  if (!/^(127\.0\.0\.1|localhost)$/i.test(url.hostname)) throw new Error(`Refusing non-local WAF_E2E_RUNTIME_URL: ${value}`);
  return url.toString().replace(/\/$/, "");
}
