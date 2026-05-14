import { escapeHtml, notify } from "../ui.js";
import { downloadBlob, renderScriptCard } from "./administration.helpers-base.js";

export function setupAdministrationScriptsRuntime(deps) {
  const { ctx, scriptsStatus, latestRuns } = deps;

  const renderScripts = (catalog) => {
    const scripts = Array.isArray(catalog?.scripts) ? catalog.scripts : [];
    if (!scripts.length) {
      scriptsStatus.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("administration.scripts.empty"))}</div>`;
      return;
    }
    scriptsStatus.innerHTML = `<div class="waf-stack">${scripts.map((script) => renderScriptCard(script, ctx, latestRuns.get(script.id) || null)).join("")}</div>`;

    scriptsStatus.querySelectorAll("[data-script-form]").forEach((formNode) => {
      formNode.addEventListener("submit", async (event) => {
        event.preventDefault();
        const scriptID = String(formNode.getAttribute("data-script-form") || "").trim();
        if (!scriptID) {
          return;
        }
        const runningNote = scriptsStatus.querySelector(`[data-script-running-note="${scriptID}"]`);
        if (runningNote) {
          runningNote.textContent = ctx.t("administration.scripts.running");
        }
        const input = {};
        formNode.querySelectorAll("[data-script-input]").forEach((inputNode) => {
          const key = String(inputNode.getAttribute("data-script-input") || "").trim();
          if (key) {
            input[key] = String(inputNode.value || "");
          }
        });
        try {
          const result = await ctx.api.post(`/api/administration/scripts/${encodeURIComponent(scriptID)}/run`, { input });
          latestRuns.set(scriptID, result);
          renderScripts(catalog);
          notify(ctx.t("administration.scripts.runSuccess"));
        } catch (error) {
          const resultNode = scriptsStatus.querySelector(`[data-script-result="${scriptID}"]`);
          if (resultNode) {
            resultNode.innerHTML = `<div class="alert">${escapeHtml(error?.message || ctx.t("administration.scripts.runError"))}</div>`;
          }
        } finally {
          if (runningNote) {
            runningNote.textContent = "";
          }
        }
      });
    });

    scriptsStatus.querySelectorAll("[data-script-download]").forEach((buttonNode) => {
      buttonNode.addEventListener("click", async () => {
        const runID = String(buttonNode.getAttribute("data-script-download") || "").trim();
        const fileName = String(buttonNode.getAttribute("data-script-download-name") || "script-output.tar.gz").trim();
        if (!runID) {
          return;
        }
        try {
          const response = await fetch(`/api/administration/scripts/runs/${encodeURIComponent(runID)}/download`, {
            method: "GET",
            credentials: "include",
            headers: { Accept: "application/gzip" }
          });
          if (!response.ok) {
            let message = `HTTP ${response.status}`;
            try {
              const payload = await response.json();
              if (payload?.error) {
                message = String(payload.error);
              }
            } catch {
              // ignore parse errors
            }
            throw new Error(message);
          }
          downloadBlob(fileName, await response.blob());
        } catch (error) {
          notify(error?.message || ctx.t("administration.scripts.downloadError"), "error");
        }
      });
    });
  };

  return { renderScripts };
}
