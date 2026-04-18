package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"waf/control-plane/internal/services"
)

type dashboardContainerService interface {
	Overview() (services.DashboardContainerOverview, error)
	Logs(req services.DashboardContainerLogsRequest) (services.DashboardContainerLogs, error)
	Issues() (services.DashboardContainerIssuesSummary, error)
}

type DashboardContainersHandler struct {
	service dashboardContainerService
}

func NewDashboardContainersHandler(service dashboardContainerService) *DashboardContainersHandler {
	return &DashboardContainersHandler{service: service}
}

func (h *DashboardContainersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "container runtime service unavailable"})
		return
	}
	switch r.URL.Path {
	case "/api/dashboard/containers/overview":
		h.handleOverview(w, r)
	case "/api/dashboard/containers/logs":
		h.handleLogs(w, r)
	case "/api/dashboard/containers/issues":
		h.handleIssues(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *DashboardContainersHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	payload, err := h.service.Overview()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": strings.TrimSpace(err.Error())})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *DashboardContainersHandler) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tail := 1000
	if rawTail := strings.TrimSpace(r.URL.Query().Get("tail")); rawTail != "" {
		parsed, err := strconv.Atoi(rawTail)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tail must be integer"})
			return
		}
		tail = parsed
	}
	payload, err := h.service.Logs(services.DashboardContainerLogsRequest{
		Container: strings.TrimSpace(r.URL.Query().Get("container")),
		Since:     strings.TrimSpace(r.URL.Query().Get("since")),
		Tail:      tail,
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": strings.TrimSpace(err.Error())})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *DashboardContainersHandler) handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	payload, err := h.service.Issues()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": strings.TrimSpace(err.Error())})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
