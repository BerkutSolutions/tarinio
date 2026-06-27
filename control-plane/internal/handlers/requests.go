package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type requestCollector interface {
	Collect() ([]map[string]any, error)
}

type requestCollectorWithOptions interface {
	CollectWithOptions(values url.Values) ([]map[string]any, error)
}

type requestCollectorProber interface {
	Probe(values url.Values) error
}

type RequestsHandler struct {
	collector requestCollector
	cache     struct {
		mu   sync.Mutex
		rows []map[string]any
	}
}

func NewRequestsHandler(collector requestCollector) *RequestsHandler {
	h := &RequestsHandler{collector: collector}
	h.cache.rows = []map[string]any{}
	return h
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
	if isProbeRequest(r.URL.Query()) {
		if prober, ok := h.collector.(requestCollectorProber); ok {
			if err := prober.Probe(r.URL.Query()); err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	var (
		items              []map[string]any
		err                error
		collectorPaginates bool
	)
	if advanced, ok := h.collector.(requestCollectorWithOptions); ok {
		query := r.URL.Query()
		if strings.TrimSpace(query.Get("retention_days")) == "" {
			storage := CurrentStorageRetention()
			query.Set("retention_days", strconv.Itoa(storage.LogsDays))
		}
		items, err = advanced.CollectWithOptions(query)
		collectorPaginates = true
	} else {
		items, err = h.collector.Collect()
	}
	if err != nil {
		if cached, ok := h.cachedRows(); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if !collectorPaginates {
		items = applyOffsetLimit(items, r.URL.Query())
	}
	h.storeRows(items)
	writeJSON(w, http.StatusOK, items)
}

func (h *RequestsHandler) cachedRows() ([]map[string]any, bool) {
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()
	if h.cache.rows == nil {
		return nil, false
	}
	return append([]map[string]any(nil), h.cache.rows...), true
}

func (h *RequestsHandler) storeRows(items []map[string]any) {
	h.cache.mu.Lock()
	h.cache.rows = append([]map[string]any(nil), items...)
	h.cache.mu.Unlock()
}

func applyOffsetLimit(items []map[string]any, query url.Values) []map[string]any {
	total := len(items)
	if total == 0 {
		return items
	}
	offset := parsePositiveInt(query.Get("offset"), 0)
	limit := parsePositiveInt(query.Get("limit"), 0)
	if offset >= total {
		return []map[string]any{}
	}
	if offset < 0 {
		offset = 0
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func parsePositiveInt(raw string, fallback int) int {
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}

func isProbeRequest(query url.Values) bool {
	switch strings.ToLower(strings.TrimSpace(query.Get("probe"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
