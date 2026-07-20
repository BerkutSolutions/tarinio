package compiler

import (
	"strings"
	"testing"
)

func TestAuthPagePreview_UsesReorderedTemplateVariants(t *testing.T) {
	tests := []struct {
		variant string
		marker  string
	}{
		{variant: "v1", marker: "Minimal Ops"},
		{variant: "v5", marker: "Classic Split"},
		{variant: "v6", marker: "Aurora"},
		{variant: "v9", marker: "Light Login"},
	}

	for _, tc := range tests {
		t.Run(tc.variant, func(t *testing.T) {
			page, ok := AuthPagePreview(tc.variant)
			if !ok {
				t.Fatalf("preview %s is unavailable", tc.variant)
			}
			content := string(page)
			if !strings.Contains(content, `body class="`+tc.variant+`"`) || !strings.Contains(content, tc.marker) {
				t.Fatalf("preview %s does not contain the expected template marker %q", tc.variant, tc.marker)
			}
		})
	}
}
