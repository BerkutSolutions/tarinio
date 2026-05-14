package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIJSExports_RegressionContracts(t *testing.T) {
	type exportCase struct {
		name    string
		file    string
		markers []string
	}

	cases := []exportCase{
		{
			name: "dashboard layout exports",
			file: filepath.Join("..", "app", "static", "js", "pages", "dashboard.layout-core.js"),
			markers: []string{
				"const GRID = 20;",
				"const WIDGETS = [",
				"export {",
				"  GRID,",
				"  WIDGETS,",
			},
		},
		{
			name: "sites page main exports",
			file: filepath.Join("..", "app", "static", "js", "pages", "sites.page-main-core.js"),
			markers: []string{
				"function applyImportPayload(",
				"function importServicesJSON(",
				"function parseRawDraft(",
				"applyImportPayload,",
				"importServicesJSON,",
			},
		},
		{
			name: "administration script card export",
			file: filepath.Join("..", "app", "static", "js", "pages", "administration.helpers-base.js"),
			markers: []string{
				"export function renderScriptCard(",
				"data-script-form=",
				"data-script-result=",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("read %s: %v", tc.file, err)
			}
			content := string(raw)
			for _, marker := range tc.markers {
				if !strings.Contains(content, marker) {
					t.Fatalf("missing marker %q in %s", marker, tc.file)
				}
			}
		})
	}
}
