package handlers

import (
	"net/http"

	"waf/control-plane/internal/services"
)

type reportService interface {
	RevisionSummary() (services.ReportSummary, error)
}

type ReportsHandler struct {
	reports reportService
}

func NewReportsHandler(reports reportService) *ReportsHandler {
	return &ReportsHandler{reports: reports}
}

func (h *ReportsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/reports/revisions" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	summary, err := h.reports.RevisionSummary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
