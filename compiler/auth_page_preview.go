package compiler

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type authPagePreviewData struct {
	BasicVerifyURI string
	TemplateName   string
}

// AuthPagePreview renders an isolated Basic Auth appearance preview from the
// same embedded template used by generated site artifacts.
func AuthPagePreview(templateName string) ([]byte, bool) {
	templateName = strings.ToLower(strings.TrimSpace(templateName))
	if !strings.HasPrefix(templateName, "v") || len(templateName) != 2 || templateName[1] < '1' || templateName[1] > '9' {
		return nil, false
	}
	content, err := TemplatesFS.ReadFile("templates/errors/auth-themed.html.tmpl")
	if err != nil {
		return nil, false
	}
	page, err := template.New("auth-preview").Parse(string(content))
	if err != nil {
		return nil, false
	}
	var output bytes.Buffer
	if err := page.Execute(&output, authPagePreviewData{BasicVerifyURI: "/auth/verify/basic", TemplateName: templateName}); err != nil {
		return nil, false
	}
	return output.Bytes(), true
}

func authPagePreviewError(templateName string) error {
	return fmt.Errorf("auth page preview %q is unavailable", templateName)
}
