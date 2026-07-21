package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSitesRawEditor_RoundTripsEveryDraftField(t *testing.T) {
	script := `
import { defaultSiteDraft } from "./ui/app/static/js/pages/sites.draft-core.js";
import { draftToEnvText, envToDraft } from "./ui/app/static/js/pages/sites.import-pipeline.js";
const draft = defaultSiteDraft();
for (const [field, value] of Object.entries(draft)) {
  if (typeof value === "string") draft[field] = "raw-parity-" + field;
  else if (typeof value === "boolean") draft[field] = !value;
  else if (typeof value === "number") draft[field] = value + 17;
  else if (Array.isArray(value)) draft[field] = ["raw-parity-" + field];
}
const env = draftToEnvText(draft);
const parsed = envToDraft(env);
if (parsed.missingFields.length) throw new Error("missing fields: " + parsed.missingFields.join(", "));
for (const field of Object.keys(draft)) {
  if (JSON.stringify(draft[field]) !== JSON.stringify(parsed.draft[field])) {
    throw new Error(field + " mismatch: " + JSON.stringify(draft[field]) + " != " + JSON.stringify(parsed.draft[field]));
  }
}
`
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	cmd := nodeESMCommand(t, script)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("raw editor round-trip failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
}
