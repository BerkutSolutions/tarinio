package handlers

import (
	"net/http"

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
	writeJSON(w, http.StatusOK, map[string]any{
		"events": items,
	})
}

