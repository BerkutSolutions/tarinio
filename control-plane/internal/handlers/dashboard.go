package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"waf/control-plane/internal/services"
)

type dashboardService interface {
	Stats() (services.DashboardStats, error)
	Probe(kind string, query url.Values) error
	DismissServiceErrors(errorIDs []string)
}

type DashboardHandler struct {
	service dashboardService
}

func NewDashboardHandler(service dashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// DELETE /api/dashboard/services/{name}/errors — скрыть ошибки сервиса
	if strings.HasPrefix(r.URL.Path, "/api/dashboard/services/") &&
		strings.HasSuffix(r.URL.Path, "/errors") &&
		r.Method == http.MethodDelete {
		h.handleDismissServiceErrors(w, r)
		return
	}
	if r.URL.Path != "/api/dashboard/stats" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if probeKind := strings.TrimSpace(r.URL.Query().Get("probe")); probeKind != "" {
		if err := h.service.Probe(probeKind, r.URL.Query()); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	stats, err := h.service.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleDismissServiceErrors обрабатывает DELETE /api/dashboard/services/{name}/errors
// Тело запроса: {"error_ids": ["id1", "id2"]} или пустое (скрыть все ошибки сервиса).
func (h *DashboardHandler) handleDismissServiceErrors(w http.ResponseWriter, r *http.Request) {
	// Извлекаем имя сервиса из пути.
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/dashboard/services/")
	trimmed = strings.TrimSuffix(trimmed, "/errors")
	serviceName := strings.TrimSpace(trimmed)
	if serviceName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "service name is required"})
		return
	}

	var body struct {
		ErrorIDs []string `json:"error_ids"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
	}

	// Если error_ids не переданы — нужно получить все ошибки сервиса из stats и скрыть их.
	if len(body.ErrorIDs) == 0 {
		stats, err := h.service.Stats()
		if err == nil {
			for _, svc := range stats.Services {
				if svc.Name != serviceName {
					continue
				}
				for _, e := range svc.UpstreamErrors {
					body.ErrorIDs = append(body.ErrorIDs, e.ID)
				}
				break
			}
		}
	}

	h.service.DismissServiceErrors(body.ErrorIDs)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dismissed": len(body.ErrorIDs)})
}

