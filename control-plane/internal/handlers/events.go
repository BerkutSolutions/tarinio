package handlers

import (
	"net/http"
	"sync"
	"time"

	"waf/control-plane/internal/events"
)

type eventService interface {
	List() ([]events.Event, error)
}

type eventProbeService interface {
	Probe() error
}

type EventsHandler struct {
	events eventService
	cache  struct {
		mu    sync.Mutex
		items []events.Event
	}
}

const maxMonitoringEventsPageSize = 500

func NewEventsHandler(events eventService) *EventsHandler {
	return &EventsHandler{events: events}
}

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/events" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if isProbeRequest(r.URL.Query()) {
		if prober, ok := h.events.(eventProbeService); ok {
			if err := prober.Probe(); err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	items, err := h.events.List()
	if err != nil {
		if cached, ok := h.cachedEvents(); ok {
			offset, limit := monitoringEventsPage(r)
			total := len(cached)
			if offset >= total {
				cached = []events.Event{}
			} else {
				end := offset + limit
				if end > total {
					end = total
				}
				cached = cached[offset:end]
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"events": cached,
				"total":  total,
				"limit":  limit,
				"offset": offset,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	storage := CurrentStorageRetention()
	if storage.EventsDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -storage.EventsDays)
		filtered := make([]events.Event, 0, len(items))
		for _, item := range items {
			occurredAt, parseErr := time.Parse(time.RFC3339Nano, item.OccurredAt)
			if parseErr != nil {
				occurredAt, parseErr = time.Parse(time.RFC3339, item.OccurredAt)
			}
			if parseErr != nil || occurredAt.Before(cutoff) {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	h.storeEvents(items)
	offset, limit := monitoringEventsPage(r)
	total := len(items)
	if offset >= total {
		items = []events.Event{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		items = items[offset:end]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events":  items,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func monitoringEventsPage(r *http.Request) (int, int) {
	offset := parsePositiveInt(r.URL.Query().Get("offset"), 0)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), maxMonitoringEventsPageSize)
	if limit <= 0 || limit > maxMonitoringEventsPageSize {
		limit = maxMonitoringEventsPageSize
	}
	return offset, limit
}

func (h *EventsHandler) cachedEvents() ([]events.Event, bool) {
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()
	if len(h.cache.items) == 0 {
		return nil, false
	}
	return append([]events.Event(nil), h.cache.items...), true
}

func (h *EventsHandler) storeEvents(items []events.Event) {
	h.cache.mu.Lock()
	h.cache.items = append([]events.Event(nil), items...)
	h.cache.mu.Unlock()
}
