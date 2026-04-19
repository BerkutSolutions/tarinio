package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/users"
)

type administrationUserStore interface {
	Get(id string) (users.User, bool, error)
	FindByUsername(username string) (users.User, bool, error)
	List() ([]users.User, error)
	Create(user users.User) (users.User, error)
	Update(user users.User) (users.User, error)
}

type administrationRoleStore interface {
	Get(id string) (roles.Role, bool, error)
	List() ([]roles.Role, error)
	Create(role roles.Role) (roles.Role, error)
	Update(role roles.Role) (roles.Role, error)
	Delete(id string) error
}

type AdministrationUsersHandler struct {
	users    administrationUserStore
	roles    administrationRoleStore
	sessions administrationSessionStore
}

type AdministrationRolesHandler struct {
	roles administrationRoleStore
	users administrationUserStore
}

type ZeroTrustHealthHandler struct {
	users administrationUserStore
	roles administrationRoleStore
}

type administrationUserPayload struct {
	ID         string   `json:"id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	FullName   string   `json:"full_name"`
	Department string   `json:"department"`
	Position   string   `json:"position"`
	Password   string   `json:"password"`
	IsActive   *bool    `json:"is_active"`
	RoleIDs    []string `json:"role_ids"`
}

type administrationRolePayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func NewAdministrationUsersHandler(userStore administrationUserStore, roleStore administrationRoleStore) *AdministrationUsersHandler {
	return &AdministrationUsersHandler{users: userStore, roles: roleStore}
}

type administrationSessionStore interface {
	DeleteSessionsByUser(userID string) error
}

func NewAdministrationUsersHandlerWithSessions(userStore administrationUserStore, roleStore administrationRoleStore, sessionStore administrationSessionStore) *AdministrationUsersHandler {
	return &AdministrationUsersHandler{users: userStore, roles: roleStore, sessions: sessionStore}
}

func NewAdministrationRolesHandler(roleStore administrationRoleStore, userStore administrationUserStore) *AdministrationRolesHandler {
	return &AdministrationRolesHandler{roles: roleStore, users: userStore}
}

func NewZeroTrustHealthHandler(userStore administrationUserStore, roleStore administrationRoleStore) *ZeroTrustHealthHandler {
	return &ZeroTrustHealthHandler{users: userStore, roles: roleStore}
}

func (h *AdministrationUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/administration/users" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/administration/users" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/users/") && r.Method == http.MethodGet:
		h.get(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/users/") && r.Method == http.MethodPut:
		h.update(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *AdministrationRolesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/administration/roles" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/administration/roles" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/roles/") && r.Method == http.MethodGet:
		h.get(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/roles/") && r.Method == http.MethodPut:
		h.update(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *ZeroTrustHealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	roleItems, err := h.roles.List()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded",
			"error":  "roles store unavailable",
		})
		return
	}
	userItems, err := h.users.List()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded",
			"error":  "users store unavailable",
		})
		return
	}
	rolesByID := map[string]roles.Role{}
	for _, item := range roleItems {
		rolesByID[item.ID] = item
	}
	missingDefaults := []string{}
	for _, item := range []string{"admin", "auditor", "manager", "soc"} {
		if _, ok := rolesByID[item]; !ok {
			missingDefaults = append(missingDefaults, item)
		}
	}
	adminRole, hasAdminRole := rolesByID["admin"]
	unknownRoleRefs := 0
	inactiveAdmins := 0
	for _, user := range userItems {
		for _, roleID := range user.RoleIDs {
			if _, ok := rolesByID[roleID]; !ok && roleID != "admin" {
				unknownRoleRefs++
			}
		}
		if isBuiltInAdmin(user) && !user.IsActive {
			inactiveAdmins++
		}
	}
	status := "ok"
	if len(missingDefaults) > 0 || !hasAdminRole || len(adminRole.Permissions) < len(rbac.AllPermissions()) || unknownRoleRefs > 0 || inactiveAdmins > 0 {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": status,
		"components": map[string]any{
			"users": map[string]any{
				"status": statusFromBool(len(userItems) > 0 || inactiveAdmins == 0),
				"count":  len(userItems),
			},
			"roles": map[string]any{
				"status":                  statusFromBool(hasAdminRole && len(missingDefaults) == 0),
				"count":                   len(roleItems),
				"missing_default_roles":   missingDefaults,
				"unknown_role_references": unknownRoleRefs,
			},
			"zero_trust": map[string]any{
				"status":            statusFromBool(hasAdminRole && len(adminRole.Permissions) >= len(rbac.AllPermissions()) && inactiveAdmins == 0),
				"admin_permissions": len(adminRole.Permissions),
				"known_permissions": len(rbac.AllPermissions()),
				"inactive_admins":   inactiveAdmins,
			},
		},
	})
}

func (h *AdministrationUsersHandler) list(w http.ResponseWriter, _ *http.Request) {
	items, err := h.users.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": itemsToUserViews(items),
	})
}

func (h *AdministrationUsersHandler) get(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/administration/users/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "user id is required"})
		return
	}
	item, ok, err := h.users.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, userToView(item))
}

func (h *AdministrationUsersHandler) create(w http.ResponseWriter, r *http.Request) {
	var payload administrationUserPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	roleIDs, err := h.normalizeRoleIDs(payload.RoleIDs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(roleIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at least one role is required"})
		return
	}
	if strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "password is required"})
		return
	}
	passwordHash, err := users.HashPassword(payload.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	isActive := true
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}
	user, err := h.users.Create(users.User{
		ID:           firstNonEmpty(payload.ID, payload.Username),
		Username:     payload.Username,
		Email:        payload.Email,
		FullName:     payload.FullName,
		Department:   payload.Department,
		Position:     payload.Position,
		PasswordHash: passwordHash,
		IsActive:     isActive,
		RoleIDs:      roleIDs,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, userToView(user))
}

func (h *AdministrationUsersHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/administration/users/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "user id is required"})
		return
	}
	current, ok, err := h.users.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "user not found"})
		return
	}
	var payload administrationUserPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	roleIDs, err := h.normalizeRoleIDs(payload.RoleIDs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	current.Email = payload.Email
	current.FullName = payload.FullName
	current.Department = payload.Department
	current.Position = payload.Position
	if payload.IsActive != nil {
		current.IsActive = *payload.IsActive
	}
	current.RoleIDs = roleIDs
	passwordChanged := false
	if strings.TrimSpace(payload.Password) != "" {
		passwordHash, err := users.HashPassword(payload.Password)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		current.PasswordHash = passwordHash
		passwordChanged = true
	}
	if isBuiltInAdmin(current) {
		current.IsActive = true
		current.RoleIDs = ensureRole(current.RoleIDs, "admin")
	}
	updated, err := h.users.Update(current)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if h.sessions != nil && passwordChanged {
		if err := h.sessions.DeleteSessionsByUser(updated.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, userToView(updated))
}

func (h *AdministrationUsersHandler) normalizeRoleIDs(items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	roleItems, err := h.roles.List()
	if err != nil {
		return nil, err
	}
	allowed := map[string]struct{}{}
	for _, item := range roleItems {
		allowed[item.ID] = struct{}{}
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		roleID := rbac.NormalizeRoleID(item)
		if roleID == "" {
			continue
		}
		if _, ok := allowed[roleID]; !ok {
			return nil, fmt.Errorf("unknown role %s", roleID)
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		seen[roleID] = struct{}{}
		out = append(out, roleID)
	}
	return out, nil
}

func (h *AdministrationRolesHandler) list(w http.ResponseWriter, _ *http.Request) {
	items, err := h.roles.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"roles":                 items,
		"available_permissions": permissionNames(rbac.AllPermissions()),
	})
}

func (h *AdministrationRolesHandler) get(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/administration/roles/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "role id is required"})
		return
	}
	item, ok, err := h.roles.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "role not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *AdministrationRolesHandler) create(w http.ResponseWriter, r *http.Request) {
	var payload administrationRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item, err := h.roles.Create(roles.Role{
		ID:          payload.ID,
		Name:        payload.Name,
		Permissions: stringsToPermissions(payload.Permissions),
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *AdministrationRolesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/administration/roles/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "role id is required"})
		return
	}
	current, ok, err := h.roles.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "role not found"})
		return
	}
	var payload administrationRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	current.Name = payload.Name
	current.Permissions = stringsToPermissions(payload.Permissions)
	updated, err := h.roles.Update(current)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func userToView(item users.User) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"username":      item.Username,
		"email":         item.Email,
		"full_name":     item.FullName,
		"department":    item.Department,
		"position":      item.Position,
		"is_active":     item.IsActive,
		"role_ids":      append([]string(nil), item.RoleIDs...),
		"totp_enabled":  item.TOTPEnabled,
		"last_login_at": item.LastLoginAt,
		"created_at":    item.CreatedAt,
		"updated_at":    item.UpdatedAt,
		"is_builtin":    isBuiltInAdmin(item),
	}
}

func itemsToUserViews(items []users.User) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, userToView(item))
	}
	return out
}

func stringsToPermissions(items []string) []rbac.Permission {
	out := make([]rbac.Permission, 0, len(items))
	seen := map[rbac.Permission]struct{}{}
	for _, item := range items {
		permission := rbac.Permission(strings.TrimSpace(item))
		if permission == "" {
			continue
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		out = append(out, permission)
	}
	return out
}

func permissionNames(items []rbac.Permission) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, string(item))
	}
	return out
}

func isBuiltInAdmin(item users.User) bool {
	return strings.EqualFold(item.ID, "admin") || strings.EqualFold(item.Username, "admin")
}

func ensureRole(items []string, roleID string) []string {
	normalizedRoleID := rbac.NormalizeRoleID(roleID)
	for _, item := range items {
		if rbac.NormalizeRoleID(item) == normalizedRoleID {
			return items
		}
	}
	return append(items, normalizedRoleID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func statusFromBool(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}

func actorFromRequest(r *http.Request) (auth.SessionView, bool) {
	return auth.SessionFromContext(r.Context())
}
