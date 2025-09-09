/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// AuthHandlers handles authentication-related HTTP requests
type AuthHandlers struct {
	authService *AuthService
	rbacManager *RBACManager
	auditLogger *AuditLogger
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authService *AuthService, rbacManager *RBACManager, auditLogger *AuditLogger) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		rbacManager: rbacManager,
		auditLogger: auditLogger,
	}
}

// LoginHandler handles user login requests
func (ah *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	response, err := ah.authService.Login(&request, ah.getClientIP(r), r.UserAgent())
	if err != nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Login failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, response)
}

// LogoutHandler handles user logout requests
func (ah *AuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	if err := ah.authService.Logout(claims.SessionID, ah.getClientIP(r), r.UserAgent()); err != nil {
		ah.respondWithError(w, http.StatusInternalServerError, "Logout failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// RefreshTokenHandler handles token refresh requests
func (ah *AuthHandlers) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var request RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	response, err := ah.authService.RefreshToken(&request)
	if err != nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Token refresh failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, response)
}

// GetCurrentUserHandler returns current user information
func (ah *AuthHandlers) GetCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	user, err := ah.authService.GetUser(claims.UserID)
	if err != nil {
		ah.respondWithError(w, http.StatusNotFound, "User not found", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, user)
}

// ChangePasswordHandler handles password change requests
func (ah *AuthHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	var request ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := ah.authService.ChangePassword(claims.UserID, &request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Password change failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// GetUsersHandler returns all users (admin only)
func (ah *AuthHandlers) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := ah.authService.GetAllUsers()
	ah.respondWithJSON(w, http.StatusOK, users)
}

// CreateUserHandler creates a new user (admin only)
func (ah *AuthHandlers) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	var request CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	user, err := ah.authService.CreateUser(&request, claims.UserID)
	if err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "User creation failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusCreated, user)
}

// UpdateUserHandler updates an existing user (admin only)
func (ah *AuthHandlers) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	var request UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	user, err := ah.authService.UpdateUser(userID, &request, claims.UserID)
	if err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "User update failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, user)
}

// DeleteUserHandler deletes a user (admin only)
func (ah *AuthHandlers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userId"]

	if err := ah.authService.DeleteUser(userID, claims.UserID); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "User deletion failed", err.Error())
		return
	}

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// GetRolesHandler returns all roles
func (ah *AuthHandlers) GetRolesHandler(w http.ResponseWriter, r *http.Request) {
	roles := ah.rbacManager.GetAllRoles()
	ah.respondWithJSON(w, http.StatusOK, roles)
}

// CreateRoleHandler creates a new role (admin only)
func (ah *AuthHandlers) CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	var request CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Convert permission IDs to Permission objects
	permissions := make([]Permission, len(request.Permissions))
	for i, permID := range request.Permissions {
		permissions[i] = Permission{
			ID:          permID,
			Resource:    "", // Will be filled based on permission ID
			Action:      "", // Will be filled based on permission ID
			Description: "",
		}
	}

	role := &Role{
		ID:          request.Name, // Use name as ID for simplicity
		Name:        request.Name,
		Description: request.Description,
		Permissions: permissions,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ah.rbacManager.CreateRole(role); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Role creation failed", err.Error())
		return
	}

	ah.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeRoleCreated,
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "roles",
		Action:    "create",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"roleId":   role.ID,
			"roleName": role.Name,
		},
		Timestamp: time.Now(),
	})

	ah.respondWithJSON(w, http.StatusCreated, role)
}

// UpdateRoleHandler updates an existing role (admin only)
func (ah *AuthHandlers) UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	roleID := vars["roleId"]

	var request UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Convert to Role object for update
	updates := &Role{}
	if request.Name != nil {
		updates.Name = *request.Name
	}
	if request.Description != nil {
		updates.Description = *request.Description
	}
	if request.Permissions != nil {
		permissions := make([]Permission, len(request.Permissions))
		for i, permID := range request.Permissions {
			permissions[i] = Permission{ID: permID}
		}
		updates.Permissions = permissions
	}
	updates.UpdatedAt = time.Now()

	if err := ah.rbacManager.UpdateRole(roleID, updates); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Role update failed", err.Error())
		return
	}

	ah.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeRoleUpdated,
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "roles",
		Action:    "update",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"roleId": roleID,
		},
		Timestamp: time.Now(),
	})

	role, _ := ah.rbacManager.GetRole(roleID)
	ah.respondWithJSON(w, http.StatusOK, role)
}

// DeleteRoleHandler deletes a role (admin only)
func (ah *AuthHandlers) DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	roleID := vars["roleId"]

	if err := ah.rbacManager.DeleteRole(roleID); err != nil {
		ah.respondWithError(w, http.StatusBadRequest, "Role deletion failed", err.Error())
		return
	}

	ah.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeRoleDeleted,
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "roles",
		Action:    "delete",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"roleId": roleID,
		},
		Timestamp: time.Now(),
	})

	ah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Role deleted successfully"})
}

// GetPermissionsHandler returns all permissions
func (ah *AuthHandlers) GetPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	permissions := ah.rbacManager.GetAllPermissions()
	ah.respondWithJSON(w, http.StatusOK, permissions)
}

// GetAuditEventsHandler returns audit events (admin only)
func (ah *AuthHandlers) GetAuditEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	filter := &AuditEventFilter{}
	if eventType := query.Get("eventType"); eventType != "" {
		filter.EventType = eventType
	}
	if userID := query.Get("userId"); userID != "" {
		filter.UserID = userID
	}
	if username := query.Get("username"); username != "" {
		filter.Username = username
	}
	if resource := query.Get("resource"); resource != "" {
		filter.Resource = resource
	}
	if action := query.Get("action"); action != "" {
		filter.Action = action
	}
	if result := query.Get("result"); result != "" {
		filter.Result = result
	}

	events := ah.auditLogger.GetEvents(filter)
	ah.respondWithJSON(w, http.StatusOK, events)
}

// GetAuditStatsHandler returns audit statistics (admin only)
func (ah *AuthHandlers) GetAuditStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := ah.auditLogger.GetEventStats()
	ah.respondWithJSON(w, http.StatusOK, stats)
}

// GetUserPermissionsHandler returns permissions for a specific user
func (ah *AuthHandlers) GetUserPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	user, err := ah.authService.GetUser(userID)
	if err != nil {
		ah.respondWithError(w, http.StatusNotFound, "User not found", err.Error())
		return
	}

	permissions := ah.rbacManager.GetUserPermissions(user.Roles)
	ah.respondWithJSON(w, http.StatusOK, permissions)
}

// Helper methods

func (ah *AuthHandlers) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func (ah *AuthHandlers) respondWithError(w http.ResponseWriter, statusCode int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

func (ah *AuthHandlers) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
