function normalizeHost(value) {
  return String(value || "").trim().toLowerCase().replace(/\.$/, "");
}

// The first site created by onboarding is the panel entrypoint. Persisting its
// host before the first compile makes the management safeguard deterministic
// and prevents CRS from protecting the control-plane API as an ordinary site.
export async function ensureOnboardingManagementHost(api, primaryHost, options = {}) {
  const host = normalizeHost(primaryHost);
  if (!host) {
    throw new Error("management host is required");
  }
  const current = await api.get("/api/settings/management-hosts");
  const configured = Array.isArray(current?.management_hosts) ? current.management_hosts : [];
  const normalized = configured.map(normalizeHost).filter(Boolean);
  if (normalized.includes(host)) {
    return current;
  }
  return api.put("/api/settings/management-hosts", {
    management_hosts: [...normalized, host],
    version: Number(current?.version || 0),
  }, options);
}
