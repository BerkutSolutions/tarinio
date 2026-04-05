import { escapeHtml } from "../ui.js";

export async function renderAbout(container, ctx) {
  container.innerHTML = `
    <div class="waf-page-stack" id="about-page">
      <section class="waf-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t("about.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("about.subtitle"))}</div>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div class="about-grid">
            <div class="about-logo-wrap">
              <img class="about-logo" src="/static/logo500x300.png" alt="Berkut Solutions - TARINIO">
            </div>
            <div class="about-content">
              <h4 class="about-name">${escapeHtml(ctx.t("about.projectName"))}</h4>
              <p class="about-description">${escapeHtml(ctx.t("about.projectDescription"))}</p>
              <p class="muted">${escapeHtml(ctx.t("about.version"))}: <strong id="about-version-value">${escapeHtml(ctx.t("about.versionFallback"))}</strong></p>
              <div class="about-links">
                <a class="btn primary btn-sm" id="about-project-link" href="https://github.com/BerkutSolutions/tarinio" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.project"))}</a>
                <a class="btn ghost btn-sm" href="https://github.com/BerkutSolutions" target="_blank" rel="noopener noreferrer">${escapeHtml(ctx.t("about.links.profile"))}</a>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  `;

  const versionNode = container.querySelector("#about-version-value");
  const projectLink = container.querySelector("#about-project-link");
  try {
    const meta = await ctx.api.get("/api/app/meta");
    versionNode.textContent = String(meta?.app_version || ctx.t("about.versionFallback"));
    if (projectLink) {
      const repo = String(meta?.repository_url || "").trim();
      projectLink.href = repo || "https://github.com/BerkutSolutions/tarinio";
    }
  } catch {
    versionNode.textContent = ctx.t("about.versionFallback");
    if (projectLink) {
      projectLink.href = "https://github.com/BerkutSolutions/tarinio";
    }
  }
}
