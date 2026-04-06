const POLL_MS = 20000;

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function timestamp(value) {
  const parsed = Date.parse(String(value || ""));
  return Number.isNaN(parsed) ? 0 : parsed;
}

export function createNotificationCenter(ctx) {
  let timer = null;
  let items = [];
  const dismissed = new Set();

  const listeners = new Set();
  const emit = () => {
    for (const listener of listeners) {
      listener(items.slice());
    }
  };

  const setItems = (next) => {
    const merged = normalizeList(next)
      .filter((item) => item && item.key && !dismissed.has(item.key))
      .sort((a, b) => Number(b.ts || 0) - Number(a.ts || 0));
    const dedup = [];
    const seen = new Set();
    for (const item of merged) {
      if (seen.has(item.key)) {
        continue;
      }
      seen.add(item.key);
      dedup.push(item);
    }
    items = dedup.slice(0, 50);
    emit();
  };

  const refresh = async () => {
    const next = [];

    try {
      const summary = await ctx.api.get("/api/reports/revisions");
      const latest = normalizeList(summary?.latest_revisions);
      const failed = latest.filter((row) => String(row?.last_apply_result || "").toLowerCase() === "failed");
      failed.slice(0, 5).forEach((row) => {
        const revisionId = String(row?.revision_id || "-");
        next.push({
          key: `jobs:${revisionId}`,
          title: ctx.t("notifications.jobsFailedTitle"),
          message: ctx.t("notifications.jobsFailedMessage", { revisionId }),
          targetPath: "/activity",
          ts: timestamp(row?.created_at),
        });
      });
    } catch {
      // no-op
    }

    setItems(next);
  };

  const start = () => {
    if (timer) {
      window.clearInterval(timer);
    }
    refresh().catch(() => {});
    timer = window.setInterval(() => {
      refresh().catch(() => {});
    }, POLL_MS);
  };

  const stop = () => {
    if (timer) {
      window.clearInterval(timer);
      timer = null;
    }
  };

  const subscribe = (listener) => {
    listeners.add(listener);
    listener(items.slice());
    return () => listeners.delete(listener);
  };

  const dismiss = (key) => {
    if (key) {
      dismissed.add(String(key));
      setItems(items);
    }
  };

  const clear = () => {
    items.forEach((item) => dismissed.add(item.key));
    setItems([]);
  };

  return {
    start,
    stop,
    refresh,
    subscribe,
    dismiss,
    clear,
    items: () => items.slice(),
  };
}
