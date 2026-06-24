const AUTO_BACKEND_ENDPOINTS = {
  opensearch: "http://opensearch:9200",
  clickhouse: "http://clickhouse:8123",
  vault: "http://vault:8200",
};

function normalizedAutoDefaults() {
  return new Set(Object.values(AUTO_BACKEND_ENDPOINTS).map((value) => String(value).trim().toLowerCase()));
}

function shouldReplaceWithAutoDefault(input) {
  const value = String(input?.value || "").trim().toLowerCase();
  return value === "" || normalizedAutoDefaults().has(value);
}

function detectContainer(overview, needle) {
  const containers = Array.isArray(overview?.containers) ? overview.containers : [];
  const normalizedNeedle = String(needle || "").trim().toLowerCase();
  return containers.some((item) => {
    const name = String(item?.name || "").trim().toLowerCase();
    const image = String(item?.image || "").trim().toLowerCase();
    return name.includes(normalizedNeedle) || image.includes(normalizedNeedle);
  });
}

function resolveBackendEndpoint(overview, backend) {
  const normalized = String(backend || "").trim().toLowerCase();
  if (!normalized) {
    return "";
  }
  if (normalized === "opensearch" && detectContainer(overview, "opensearch")) {
    return AUTO_BACKEND_ENDPOINTS.opensearch;
  }
  if (normalized === "clickhouse" && detectContainer(overview, "clickhouse")) {
    return AUTO_BACKEND_ENDPOINTS.clickhouse;
  }
  if (normalized === "vault" && detectContainer(overview, "vault")) {
    return AUTO_BACKEND_ENDPOINTS.vault;
  }
  return AUTO_BACKEND_ENDPOINTS[normalized] || "";
}

export function syncLoggingEndpointDefaults({ overview, loggingHotBackend, loggingColdBackend, loggingOpenSearchEndpoint, loggingEndpoint, loggingVaultAddress }) {
  if (loggingOpenSearchEndpoint && shouldReplaceWithAutoDefault(loggingOpenSearchEndpoint)) {
    loggingOpenSearchEndpoint.value = resolveBackendEndpoint(overview, loggingHotBackend?.value || "opensearch");
  }
  if (loggingEndpoint && shouldReplaceWithAutoDefault(loggingEndpoint)) {
    loggingEndpoint.value = resolveBackendEndpoint(overview, loggingColdBackend?.value || "clickhouse");
  }
  if (loggingVaultAddress && shouldReplaceWithAutoDefault(loggingVaultAddress)) {
    loggingVaultAddress.value = resolveBackendEndpoint(overview, "vault");
  }
}

export function bindSecretFieldToggles(container, ctx, items) {
  if (!container || !Array.isArray(items)) {
    return;
  }
  items.forEach(({ inputId, buttonId }) => {
    const input = container.querySelector(`#${inputId}`);
    const button = container.querySelector(`#${buttonId}`);
    if (!input || !button) {
      return;
    }
    const sync = () => {
      const hidden = input.type === "password";
      button.textContent = ctx.t(hidden ? "common.show" : "common.hide");
      button.setAttribute("aria-pressed", hidden ? "false" : "true");
    };
    sync();
    button.addEventListener("click", () => {
      input.type = input.type === "password" ? "text" : "password";
      sync();
    });
  });
}
