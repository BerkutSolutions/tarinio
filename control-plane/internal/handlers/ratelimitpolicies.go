package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/ratelimitpolicies"
)

type rateLimitPolicyService interface {
	Create(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
	List() ([]ratelimitpolicies.RateLimitPolicy, error)
	Update(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
	Delete(ctx context.Context, id string) error
}

type RateLimitPoliciesHandler struct {
	policies rateLimitPolicyService
}

func NewRateLimitPoliciesHandler(policies rateLimitPolicyService) *RateLimitPoliciesHandler {
	return &RateLimitPoliciesHandler{policies: policies}
}

func (h *RateLimitPoliciesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/rate-limit-policies" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/rate-limit-policies" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/rate-limit-policies/") && r.Method == http.MethodPut:
		h.update(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/rate-limit-policies/") && r.Method == http.MethodDelete:
		h.delete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *RateLimitPoliciesHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.policies.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "rate limit policy store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *RateLimitPoliciesHandler) create(w http.ResponseWriter, r *http.Request) {
	var item ratelimitpolicies.RateLimitPolicy
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	created, err := h.policies.Create(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *RateLimitPoliciesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/rate-limit-policies/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "rate limit policy id is required"})
		return
	}
	var item ratelimitpolicies.RateLimitPolicy
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item.ID = id
	updated, err := h.policies.Update(withActorIP(r), item)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *RateLimitPoliciesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/rate-limit-policies/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "rate limit policy id is required"})
		return
	}
	if err := h.policies.Delete(withActorIP(r), id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
