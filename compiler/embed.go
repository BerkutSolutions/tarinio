// Package compiler is the root package for the WAF compiler.
// It embeds all template files into the binary so the binary is
// self-contained and independent of the filesystem layout on the host.
package compiler

import "embed"

//go:embed all:templates
var TemplatesFS embed.FS
