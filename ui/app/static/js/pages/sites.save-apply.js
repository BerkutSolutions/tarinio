export const siteSaveNoAutoApplyOptions = {
  headers: {
    "X-WAF-Auto-Apply-Disabled": "1"
  }
};

export async function compileAndApplySiteRevision(ctx, targetSiteIDs) {
  const body = {};
  if (Array.isArray(targetSiteIDs) && targetSiteIDs.length > 0) {
    body.target_site_ids = targetSiteIDs
      .map((id) => String(id || "").trim())
      .filter(Boolean);
  }
  const compileResponse = await ctx.api.post("/api/revisions/compile", body, siteSaveNoAutoApplyOptions);
  const revisionID = String(compileResponse?.revision?.id || "").trim();
  if (!revisionID) {
    throw new Error("site revision compile failed");
  }
  const applyResponse = await ctx.api.post(`/api/revisions/${encodeURIComponent(revisionID)}/apply`, {});
  if (String(applyResponse?.status || "").trim().toLowerCase() === "failed") {
    throw new Error(String(applyResponse?.result || "").trim() || "site revision apply failed");
  }
  return revisionID;
}
