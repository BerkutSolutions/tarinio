package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"waf/compiler"
)

// errorPagePreviewAllowedSlugs is the set of valid preview slugs.
// Keys must match *.preview.html filenames under compiler/templates/errors/.
var errorPagePreviewAllowedSlugs = map[string]string{
	"400":        "400.preview.html",
	"401":        "401.preview.html",
	"402":        "402.preview.html",
	"403":        "403.preview.html",
	"404":        "404.preview.html",
	"405":        "405.preview.html",
	"406":        "406.preview.html",
	"407":        "407.preview.html",
	"408":        "408.preview.html",
	"409":        "409.preview.html",
	"410":        "410.preview.html",
	"411":        "411.preview.html",
	"412":        "412.preview.html",
	"413":        "413.preview.html",
	"414":        "414.preview.html",
	"415":        "415.preview.html",
	"416":        "416.preview.html",
	"417":        "417.preview.html",
	"418":        "418.preview.html",
	"421":        "421.preview.html",
	"422":        "422.preview.html",
	"423":        "423.preview.html",
	"424":        "424.preview.html",
	"425":        "425.preview.html",
	"426":        "426.preview.html",
	"428":        "428.preview.html",
	"429":        "429.preview.html",
	"431":        "431.preview.html",
	"444":        "444.preview.html",
	"451":        "geo_block.preview.html",
	"500":        "500.preview.html",
	"501":        "501.preview.html",
	"502":        "502.preview.html",
	"503":        "503.preview.html",
	"504":        "504.preview.html",
	"505":        "505.preview.html",
	"507":        "507.preview.html",
	"508":        "508.preview.html",
	"510":        "510.preview.html",
	"511":        "511.preview.html",
	"geo-block":  "geo_block.preview.html",
	// antibot challenge page variants
	"antibot-v1": "antibot-v1.preview.html",
	"antibot-v2": "antibot-v2.preview.html",
	"antibot-v3": "antibot-v3.preview.html",
	"antibot-v4": "antibot-v4.preview.html",
	"antibot-v5": "antibot-v5.preview.html",
	// captcha challenge page variants
	"captcha-v1": "captcha-v1.preview.html",
	"captcha-v2": "captcha-v2.preview.html",
	"captcha-v3": "captcha-v3.preview.html",
	"captcha-v4": "captcha-v4.preview.html",
	"captcha-v5": "captcha-v5.preview.html",
}

var reSlug = regexp.MustCompile(`^[a-z0-9\-]+$`)

// ErrorPagePreviewHandler serves WAF error page HTML previews from the
// embedded TemplatesFS. No filesystem access at runtime — previews work
// identically in Docker and locally without WAF_ERROR_PAGES_TEMPLATES_DIR.
// Only accessible to authenticated users (wrapped with withAuth in server.go).
// Path: GET /api/error-pages/preview/{slug}
type ErrorPagePreviewHandler struct{}

func NewErrorPagePreviewHandler(_ string) *ErrorPagePreviewHandler {
	return &ErrorPagePreviewHandler{}
}

func (h *ErrorPagePreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract slug from path: /api/error-pages/preview/{slug}
	slug := strings.TrimPrefix(r.URL.Path, "/api/error-pages/preview/")
	slug = strings.Trim(slug, "/")

	if !reSlug.MatchString(slug) || slug == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filename, ok := errorPagePreviewAllowedSlugs[slug]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Read from embedded FS — always works regardless of container filesystem layout.
	data, err := compiler.TemplatesFS.ReadFile("templates/errors/" + filename)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
