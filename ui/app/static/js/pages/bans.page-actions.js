import { setError } from "../ui.js";
import { clearRateLimitCookies, normalizeIP } from "./bans.helpers.base.js";
import {
  clearDismissedBanRow,
  dismissBanRow,
  loadManualBanTimers,
  manualBanTimerRowKey,
  removeManualBanTimer,
  saveManualBanTimers,
  upsertManualBanTimer
} from "./bans.helpers.timers.js";
import { postBanAction } from "./bans.helpers.events.js";

export function bindBansPageActions({
  container,
  ctx,
  allServicesSiteID,
  latestSiteIDsRef,
  createSiteNode,
  createIPNode,
  createDurationNode,
  createStatusNode,
  createSubmitNode,
  extendDurationNode,
  extendStatusNode,
  extendSubmitNode,
  unbanStatusNode,
  unbanSubmitNode,
  extendDurations,
  modalControllers,
  renderRows
}) {
  container.querySelector("#bans-refresh")?.addEventListener("click", () => {
    renderRows();
  });
  container.querySelector("#bans-create")?.addEventListener("click", () => {
    modalControllers.openCreateModal();
  });

  createSubmitNode?.addEventListener("click", async () => {
    const siteID = String(createSiteNode?.value || "").trim();
    const ip = normalizeIP(createIPNode?.value || "");
    const durationSec = Number.parseInt(String(createDurationNode?.value || "0"), 10);
    if (!siteID || !ip) {
      setError(createStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    if (!Number.isFinite(durationSec) || durationSec < 0) {
      setError(createStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    try {
      createSubmitNode.disabled = true;
      const targets = siteID === allServicesSiteID ? [...latestSiteIDsRef.current] : [siteID];
      if (!targets.length) {
        setError(createStatusNode, ctx.t("bans.error.createValidation"));
        return;
      }
      await Promise.all(targets.map((targetSiteID) => postBanAction(ctx, targetSiteID, "primary", "ban", ip)));
      const timers = loadManualBanTimers();
      for (const targetSiteID of targets) {
        clearDismissedBanRow(targetSiteID, ip);
        if (durationSec > 0) {
          upsertManualBanTimer(timers, targetSiteID, ip, new Date(Date.now() + (durationSec * 1000)));
        } else {
          removeManualBanTimer(timers, targetSiteID, ip);
        }
      }
      saveManualBanTimers(timers);
      createIPNode.value = "";
      createDurationNode.value = String(extendDurations[0]?.seconds || 0);
      modalControllers.closeCreateModal();
      ctx.notify(ctx.t("toast.ipBanned"));
      await renderRows();
    } catch (error) {
      setError(createStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      createSubmitNode.disabled = false;
    }
  });

  extendSubmitNode?.addEventListener("click", async () => {
    const row = modalControllers.getExtendDraft()?.row;
    if (!row || !row.siteID || !row.ip) {
      setError(extendStatusNode, ctx.t("bans.error.action"));
      return;
    }
    const durationSec = Number.parseInt(String(extendDurationNode?.value || "0"), 10);
    if (!Number.isFinite(durationSec) || durationSec < 0) {
      setError(extendStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    const ip = normalizeIP(row.ip);
    try {
      extendSubmitNode.disabled = true;
      await postBanAction(ctx, row.siteID, "primary", "ban", ip);
      clearDismissedBanRow(row.siteID, ip);
      const timers = loadManualBanTimers();
      if (durationSec > 0) {
        const currentTimer = timers.get(manualBanTimerRowKey(row.siteID, ip));
        const currentUnbanAt = Date.parse(String(currentTimer?.unbanAt || row.expiresAt?.toISOString?.() || ""));
        const startFrom = Number.isNaN(currentUnbanAt) ? Date.now() : Math.max(Date.now(), currentUnbanAt);
        upsertManualBanTimer(timers, row.siteID, ip, new Date(startFrom + (durationSec * 1000)));
      } else {
        removeManualBanTimer(timers, row.siteID, ip);
      }
      saveManualBanTimers(timers);
      modalControllers.closeExtendModal();
      ctx.notify(ctx.t("toast.ipBanned"));
      await renderRows();
    } catch (error) {
      setError(extendStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      extendSubmitNode.disabled = false;
    }
  });

  unbanSubmitNode?.addEventListener("click", async () => {
    const row = modalControllers.getUnbanDraft()?.row;
    if (!row || !row.siteID || !row.ip) {
      setError(unbanStatusNode, ctx.t("bans.error.action"));
      return;
    }
    const ip = normalizeIP(row.ip);
    try {
      unbanSubmitNode.disabled = true;
      await postBanAction(ctx, row.siteID, "primary", "unban", ip);
      const timers = loadManualBanTimers();
      removeManualBanTimer(timers, row.siteID, ip);
      saveManualBanTimers(timers);
      dismissBanRow(row.siteID, ip);
      clearRateLimitCookies(row.siteID);
      modalControllers.closeUnbanModal();
      ctx.notify(ctx.t("toast.ipUnbanned"));
      await renderRows();
    } catch (error) {
      setError(unbanStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      unbanSubmitNode.disabled = false;
    }
  });
}
