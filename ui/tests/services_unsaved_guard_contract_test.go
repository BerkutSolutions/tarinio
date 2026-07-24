package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesUnsavedGuardCoversBackAndBeforeUnload(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "sites.unsaved-guard.js"))
	if err != nil {
		t.Fatalf("read unsaved guard: %v", err)
	}
	content := string(body)
	for _, marker := range []string{"beforeunload", "state.editorDirty", "confirmAction(message)", `form?.addEventListener("input"`, `form?.addEventListener("change"`} {
		if !strings.Contains(content, marker) {
			t.Fatalf("unsaved guard missing %q", marker)
		}
	}
}
