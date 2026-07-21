package tests

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSitesSimpleRawSwitchContracts(t *testing.T) {
	command := "node"
	if runtime.GOOS == "windows" {
		command = "node.exe"
	}
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("%s is required for the simple/raw switch contract", command)
	}

	script := `
import { bindDetailCore } from "./ui/app/static/js/pages/sites.detail-core-bindings.js";
import { defaultSiteDraft } from "./ui/app/static/js/pages/sites.draft-core.js";
import { draftToEnvText } from "./ui/app/static/js/pages/sites.import-pipeline.js";
const rawButton = { dataset: { modeTab: "raw" }, addEventListener(_type, handler) { this.handler = handler; } };
const container = {
  querySelector(selector) { return selector === "#sites-feedback" ? {} : null; },
  querySelectorAll(selector) { return selector === "[data-mode-tab]" ? [rawButton] : []; },
  addEventListener() {},
};
const state = { editorMode: "easy", rawEnvText: "WAF_SITE_UPSTREAM_HOST=127.0.0.1\n", rawMissingFields: [], route: { mode: "detail" }, draft: defaultSiteDraft() };
const currentDraft = defaultSiteDraft();
for (const [field, value] of Object.entries(currentDraft)) {
  if (typeof value === "string") currentDraft[field] = "current-" + field;
  else if (typeof value === "boolean") currentDraft[field] = !value;
  else if (typeof value === "number") currentDraft[field] = value + 23;
  else if (Array.isArray(value)) currentDraft[field] = ["current-" + field];
}
currentDraft.upstream_scheme = "http";
currentDraft.upstream_host = "privatebin";
currentDraft.upstream_port = 8080;
bindDetailCore(container, state, { t: () => "error" }, {
  go() {}, render() {}, getDraft: () => currentDraft, parseRawDraft() {},
  syncStateDraftFromForm: () => { state.draft = currentDraft; },
  draftToEnvText, ensureControlPlaneAccessManagementMethods: (draft) => draft,
  normalizeAutoSiteID: () => "", syncDerivedFieldsFromID() {}, normalizeServiceProfile: (value) => value,
  applyServiceProfilePresetToDraft: (draft) => draft, toggleCertificateImportActions() {}, highlightSelector() {},
});
rawButton.handler();
const expected = draftToEnvText(currentDraft);
if (state.editorMode !== "raw") throw new Error("editor did not switch to raw");
if (state.rawEnvText !== expected) throw new Error("raw state is stale");
if (!state.rawEnvText.includes("WAF_SITE_UPSTREAM_HOST=privatebin") || state.rawEnvText.includes("WAF_SITE_UPSTREAM_HOST=127.0.0.1")) {
  throw new Error("raw upstream does not match simple draft");
}
const { envToDraft } = await import("./ui/app/static/js/pages/sites.import-pipeline.js");
const { shouldUpsertBaseResources } = await import("./ui/app/static/js/pages/sites.access-upsert.js");
const rawDraft = envToDraft(state.rawEnvText).draft;
const persistedSite = { id: rawDraft.id, primary_host: rawDraft.primary_host, enabled: rawDraft.enabled };
const staleUpstream = { id: rawDraft.upstream_id, site_id: rawDraft.id, scheme: "http", host: "127.0.0.1", port: 8080 };
if (!shouldUpsertBaseResources(rawDraft, persistedSite, staleUpstream, null)) {
  throw new Error("raw-to-simple save would skip updated upstream");
}
`
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	cmd := exec.Command(command, "--input-type=module", "--eval", script)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("simple-to-raw switch contract failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
}
