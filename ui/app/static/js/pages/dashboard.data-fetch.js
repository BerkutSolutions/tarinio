function normalizeRequestRowsPayload(payload) {
  if (Array.isArray(payload)) {
    return payload.filter((row) => row && typeof row === "object");
  }
  const raw = String(payload || "").trim();
  if (!raw) {
    return [];
  }
  if (raw.startsWith("[")) {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed.filter((row) => row && typeof row === "object") : [];
    } catch (_error) {
      return [];
    }
  }
  const rows = [];
  for (const sourceLine of raw.split(/\r?\n/)) {
    const line = String(sourceLine || "").trim();
    if (!line) {
      continue;
    }
    try {
      const parsed = JSON.parse(line);
      if (parsed && typeof parsed === "object") {
        rows.push(parsed);
      }
    } catch (_error) {
      return [];
    }
  }
  return rows;
}

async function fetchRequestsRows() {
  try {
    const response = await fetch("/api/requests", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return [];
    }
    const text = await response.text();
    try {
      return normalizeRequestRowsPayload(JSON.parse(text));
    } catch (_error) {
      return normalizeRequestRowsPayload(text);
    }
  } catch (_error) {
    return [];
  }
}

async function fetchEventsRows() {
  try {
    const response = await fetch("/api/events", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return [];
    }
    const payload = await response.json();
    if (Array.isArray(payload)) return payload;
    if (Array.isArray(payload?.events)) return payload.events;
    if (Array.isArray(payload?.items)) return payload.items;
    return [];
  } catch (_error) {
    return [];
  }
}

async function fetchContainersOverview() {
  try {
    const response = await fetch("/api/dashboard/containers/overview", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return null;
    }
    return await response.json();
  } catch (_error) {
    return null;
  }
}

async function fetchContainerLogs(container, since = "", tail = 1000) {
  const params = new URLSearchParams();
  params.set("container", String(container || ""));
  if (since) {
    params.set("since", since);
  }
  if (tail > 0) {
    params.set("tail", String(tail));
  }
  const response = await fetch(`/api/dashboard/containers/logs?${params.toString()}`, {
    method: "GET",
    credentials: "include",
    headers: { Accept: "application/json" }
  });
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        message = String(payload.error);
      }
    } catch (_error) {
      // ignore parse errors
    }
    throw new Error(message);
  }
  return await response.json();
}

export {
  fetchRequestsRows,
  fetchEventsRows,
  fetchContainersOverview,
  fetchContainerLogs
};
