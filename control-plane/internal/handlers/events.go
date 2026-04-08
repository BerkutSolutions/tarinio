package handlers

import (
	"net/http"
	"time"

	"waf/control-plane/internal/events"
)

type eventService interface {
	List() ([]events.Event, error)
}

type EventsHandler struct {
	events eventService
}

func NewEventsHandler(events eventService) *EventsHandler {
	return &EventsHandler{events: events}
}

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/events" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	items, err := h.events.List()
	if err != nil {
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
	writeJSON(w, http.StatusOK, map[string]any{
		"events": items,
	})
}
