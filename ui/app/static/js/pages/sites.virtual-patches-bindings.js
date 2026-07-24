import { setError } from "../ui.js";

function patchPath(siteID) {
  return `/api/virtual-patches/${encodeURIComponent(siteID)}`;
}

function expirationToRFC3339(value) {
  const date = String(value || "").trim();
  return date ? `${date}T23:59:59Z` : "";
}

export function bindVirtualPatchesEditor(container, state, ctx, render) {
  const siteID = String(state.route?.siteID || state.draft?.id || "").trim();
  if (!siteID || !container.querySelector("#virtual-patches-editor")) return;

  if (state.virtualPatchesSiteID !== siteID) {
    state.virtualPatchesSiteID = siteID;
    state.virtualPatches = [];
    ctx.api.get(patchPath(siteID)).then((patches) => {
      if (state.virtualPatchesSiteID !== siteID) return;
      state.virtualPatches = Array.isArray(patches) ? patches : [];
      render();
    }).catch((error) => {
      setError(container.querySelector("#sites-feedback"), String(error?.message || error));
    });
    return;
  }

  container.querySelector("#vp-add-btn")?.addEventListener("click", async () => {
    const pattern = String(container.querySelector("#vp-pattern")?.value || "").trim();
    const target = String(container.querySelector("#vp-target")?.value || "uri").trim();
    const action = String(container.querySelector("#vp-action")?.value || "block").trim();
    const expiresAt = expirationToRFC3339(container.querySelector("#vp-expires")?.value);
    if (!pattern) {
      setError(container.querySelector("#sites-feedback"), "Virtual patch pattern is required");
      return;
    }
    try {
      const created = await ctx.api.post(`${patchPath(siteID)}?auto_apply=false`, {
        pattern, target, action, expires_at: expiresAt,
      });
      state.virtualPatches = [...(Array.isArray(state.virtualPatches) ? state.virtualPatches : []), created];
      render();
    } catch (error) {
      setError(container.querySelector("#sites-feedback"), String(error?.message || error));
    }
  });

  container.querySelectorAll("[data-vp-delete]").forEach((button) => {
    button.addEventListener("click", async () => {
      const patchID = String(button.dataset.vpDelete || "").trim();
      if (!patchID) return;
      try {
        await ctx.api.delete(`${patchPath(siteID)}/${encodeURIComponent(patchID)}?auto_apply=false`);
        state.virtualPatches = (state.virtualPatches || []).filter((patch) => String(patch?.id || "") !== patchID);
        render();
      } catch (error) {
        setError(container.querySelector("#sites-feedback"), String(error?.message || error));
      }
    });
  });
}
