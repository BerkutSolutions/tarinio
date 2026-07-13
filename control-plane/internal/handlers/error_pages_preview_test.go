package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorPagePreviewHandlerServesCanonicalFallbackCodes(t *testing.T) {
	handler := NewErrorPagePreviewHandler("")
	for _, slug := range []string{"403", "421", "451", "494", "499", "506", "520", "526", "geo_block", "geo-block"} {
		request := httptest.NewRequest(http.MethodGet, "/api/error-pages/preview/"+slug, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("preview %s: got %d", slug, response.Code)
		}
		if strings.Contains(response.Body.String(), "HTTP 400") && slug != "403" {
			t.Fatalf("preview %s must not reuse the 400 page", slug)
		}
		if slug == "495" && !strings.Contains(response.Body.String(), "SSL Certificate Error") {
			t.Fatalf("preview 495 must use its dedicated title")
		}
		for _, locale := range []string{"en:", "ru:", "de:", "sr:", "zh:"} {
			if !strings.Contains(response.Body.String(), locale) {
				t.Fatalf("preview %s lacks %s", slug, locale)
			}
		}
	}
}

func TestErrorPagePreviewHandlerRejectsUnknownSlug(t *testing.T) {
	handler := NewErrorPagePreviewHandler("")
	request := httptest.NewRequest(http.MethodGet, "/api/error-pages/preview/527", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown preview must return 404, got %d", response.Code)
	}
}
