package handlers

import "net/http"

type requestCollector interface {
	Collect() ([]map[string]any, error)
}

type RequestsHandler struct {
	collector requestCollector
}

func NewRequestsHandler(collector requestCollector) *RequestsHandler {
	return &RequestsHandler{collector: collector}
}

func (h *RequestsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.collector == nil {
		writeJSON(w, http.StatusOK, []map[string]any{})
		return
	}
	items, err := h.collector.Collect()
	if err != nil {
		writeJSON(w, http.StatusOK, []map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}
