package httpserver

import (
	"net/http"
	"strings"

	"waf/internal/observability"
)

const metricsTokenHeader = "X-TARINIO-Metrics-Token"

func metricsHandler(registry *observability.Registry, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		expected := strings.TrimSpace(token)
		presented := strings.TrimSpace(r.Header.Get(metricsTokenHeader))
		if presented == "" {
			presented = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if expected != "" && presented != expected {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := registry.WritePrometheus(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
