package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"waf/control-plane/internal/services"
	"waf/control-plane/internal/users"
)

type scimService interface {
	AuthenticateSCIMBearer(raw string) error
	ListSCIMUsers() ([]services.SCIMUserRecord, error)
	GetSCIMUser(id string) (services.SCIMUserRecord, bool, error)
	ProvisionSCIMUser(ctx context.Context, input services.SCIMProvisionUserInput) (users.User, error)
	DeactivateSCIMUser(id string) (services.SCIMUserRecord, error)
	ListSCIMGroups() ([]services.SCIMGroupRecord, error)
}

type SCIMHandler struct {
	service scimService
}

func NewSCIMHandler(service scimService) *SCIMHandler {
	return &SCIMHandler{service: service}
}

func (h *SCIMHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.service.AuthenticateSCIMBearer(strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": err.Error()})
		return
	}
	switch {
	case r.URL.Path == "/scim/v2/ServiceProviderConfig" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
			"patch":   map[string]any{"supported": true},
			"filter":  map[string]any{"supported": true},
		})
	case r.URL.Path == "/scim/v2/Users" && r.Method == http.MethodGet:
		h.listUsers(w, r)
	case r.URL.Path == "/scim/v2/Users" && r.Method == http.MethodPost:
		h.upsertUser(w, r)
	case strings.HasPrefix(r.URL.Path, "/scim/v2/Users/") && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		h.upsertUser(w, r)
	case strings.HasPrefix(r.URL.Path, "/scim/v2/Users/") && r.Method == http.MethodDelete:
		h.deleteUser(w, r)
	case strings.HasPrefix(r.URL.Path, "/scim/v2/Users/") && r.Method == http.MethodGet:
		h.getUser(w, r)
	case r.URL.Path == "/scim/v2/Groups" && r.Method == http.MethodGet:
		h.listGroups(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *SCIMHandler) listUsers(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.ListSCIMUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"Resources":    mapSCIMUsers(items),
		"totalResults": len(items),
	})
}

func (h *SCIMHandler) getUser(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/scim/v2/Users/"), "/")
	item, ok, err := h.service.GetSCIMUser(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, mapSCIMUser(item))
}

func (h *SCIMHandler) upsertUser(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid request body"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/scim/v2/Users/"), "/")
	input := services.SCIMProvisionUserInput{
		ExternalID: firstNonEmptySCIM(scimString(payload, "externalId"), id),
		UserName:   firstNonEmptySCIM(scimString(payload, "userName"), id),
		Email:      scimPrimaryEmail(payload),
		FullName:   scimStringPath(payload, "name", "formatted"),
		Department: scimStringPath(payload, "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User", "department"),
		Position:   scimStringPath(payload, "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User", "title"),
		Active:     scimBool(payload, "active", true),
		Groups:     scimGroups(payload),
	}
	user, err := h.service.ProvisionSCIMUser(withActorIP(r), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		status = http.StatusCreated
	}
	writeJSON(w, status, mapSCIMUser(services.SCIMUserRecord{
		ID:         user.ID,
		UserName:   user.Username,
		ExternalID: user.ExternalID,
		Active:     user.IsActive,
		Email:      user.Email,
		FullName:   user.FullName,
		Department: user.Department,
		Position:   user.Position,
		Groups:     append([]string(nil), user.ExternalGroups...),
	}))
}

func (h *SCIMHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/scim/v2/Users/"), "/")
	if _, err := h.service.DeactivateSCIMUser(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SCIMHandler) listGroups(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.ListSCIMGroups()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	resources := make([]map[string]any, 0, len(items))
	for _, item := range items {
		resources = append(resources, map[string]any{
			"id":          item.ID,
			"displayName": item.DisplayName,
			"members":     []any{},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"Resources":    resources,
		"totalResults": len(resources),
	})
}

func mapSCIMUsers(items []services.SCIMUserRecord) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, mapSCIMUser(item))
	}
	return out
}

func mapSCIMUser(item services.SCIMUserRecord) map[string]any {
	return map[string]any{
		"id":         item.ID,
		"userName":   item.UserName,
		"externalId": item.ExternalID,
		"active":     item.Active,
		"name":       map[string]any{"formatted": item.FullName},
		"emails":     []map[string]any{{"value": item.Email, "primary": true}},
		"groups":     mapSCIMGroups(item.Groups),
		"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": map[string]any{
			"department": item.Department,
			"title":      item.Position,
		},
	}
}

func mapSCIMGroups(items []string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"display": item, "value": item})
	}
	return out
}

func scimString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func scimStringPath(payload map[string]any, key string, nested string) string {
	value, ok := payload[key].(map[string]any)
	if !ok {
		return ""
	}
	return scimString(value, nested)
}

func scimBool(payload map[string]any, key string, fallback bool) bool {
	value, ok := payload[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func scimPrimaryEmail(payload map[string]any) string {
	items, ok := payload["emails"].([]any)
	if !ok {
		return ""
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if primary, _ := row["primary"].(bool); primary {
			return scimString(row, "value")
		}
	}
	if len(items) == 0 {
		return ""
	}
	row, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	return scimString(row, "value")
}

func scimGroups(payload map[string]any) []string {
	items, ok := payload["groups"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text := firstNonEmptySCIM(scimString(row, "display"), scimString(row, "value"))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func firstNonEmptySCIM(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
