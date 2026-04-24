package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/antiddossuggestions"
)

type antiDDoSRuleSuggestionsService interface {
	List() ([]antiddossuggestions.Suggestion, error)
	Upsert(ctx context.Context, item antiddossuggestions.Suggestion) (antiddossuggestions.Suggestion, error)
	SetStatus(ctx context.Context, id, status string) (antiddossuggestions.Suggestion, error)
}

type AntiDDoSRuleSuggestionsHandler struct {
	service antiDDoSRuleSuggestionsService
}

func NewAntiDDoSRuleSuggestionsHandler(service antiDDoSRuleSuggestionsService) *AntiDDoSRuleSuggestionsHandler {
	return &AntiDDoSRuleSuggestionsHandler{service: service}
}

func (h *AntiDDoSRuleSuggestionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	basePath := "/api/anti-ddos/rule-suggestions"
	if r.URL.Path == basePath {
		switch r.Method {
		case http.MethodGet:
			h.list(w)
		case http.MethodPut, http.MethodPost:
			h.upsert(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.HasPrefix(r.URL.Path, basePath+"/") && strings.HasSuffix(r.URL.Path, "/status") {
		switch r.Method {
		case http.MethodPut, http.MethodPost:
			h.setStatus(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (h *AntiDDoSRuleSuggestionsHandler) list(w http.ResponseWriter) {
	items, err := h.service.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *AntiDDoSRuleSuggestionsHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var item antiddossuggestions.Suggestion
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	updated, err := h.service.Upsert(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type suggestionStatusRequest struct {
	Status string `json:"status"`
}

func (h *AntiDDoSRuleSuggestionsHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/anti-ddos/rule-suggestions/")
	id := strings.TrimSpace(strings.TrimSuffix(path, "/status"))
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "suggestion id is required"})
		return
	}
	var req suggestionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	updated, err := h.service.SetStatus(withActorIP(r), id, req.Status)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
