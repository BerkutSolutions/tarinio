export const siteSaveNoAutoApplyOptions = {
  headers: {
    "X-WAF-Auto-Apply-Disabled": "1"
  }
};

export async function compileAndApplySiteRevision(ctx) {
  const compileResponse = await ctx.api.post("/api/revisions/compile", {}, siteSaveNoAutoApplyOptions);
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
