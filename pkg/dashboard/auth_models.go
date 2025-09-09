/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"time"
)

// User represents a system user
type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	FullName    string     `json:"fullName"`
	Roles       []string   `json:"roles"`
	IsActive    bool       `json:"isActive"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

// Role represents a system role with permissions
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	IsSystem    bool         `json:"isSystem"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// Permission represents a specific permission
type Permission struct {
	ID          string `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// ServiceAccount represents a service account for component authentication
type ServiceAccount struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ClientID    string     `json:"clientId"`
	Roles       []string   `json:"roles"`
	IsActive    bool       `json:"isActive"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID      string   `json:"userId"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"sessionId"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	Issuer      string   `json:"iss"`
	Subject     string   `json:"sub"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      User      `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Username string   `json:"username" validate:"required"`
	Email    string   `json:"email" validate:"required,email"`
	FullName string   `json:"fullName" validate:"required"`
	Password string   `json:"password" validate:"required,min=8"`
	Roles    []string `json:"roles"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Email    *string  `json:"email,omitempty" validate:"omitempty,email"`
	FullName *string  `json:"fullName,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	IsActive *bool    `json:"isActive,omitempty"`
}

// CreateRoleRequest represents a role creation request
type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest represents a role update request
type UpdateRoleRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// CreateServiceAccountRequest represents a service account creation request
type CreateServiceAccountRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

// UpdateServiceAccountRequest represents a service account update request
type UpdateServiceAccountRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	IsActive    *bool    `json:"isActive,omitempty"`
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"eventType"`
	UserID    string                 `json:"userId"`
	Username  string                 `json:"username"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	IPAddress string                 `json:"ipAddress"`
	UserAgent string                 `json:"userAgent"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}

// Predefined system roles
var (
	SystemRoles = []Role{
		{
			ID:          "admin",
			Name:        "Administrator",
			Description: "Full system access",
			IsSystem:    true,
			Permissions: []Permission{
				{ID: "system:*", Resource: "*", Action: "*", Description: "Full system access"},
			},
		},
		{
			ID:          "operator",
			Name:        "Network Operator",
			Description: "Network operations and monitoring",
			IsSystem:    true,
			Permissions: []Permission{
				{ID: "e2:read", Resource: "e2", Action: "read", Description: "Read E2 interface data"},
				{ID: "e2:write", Resource: "e2", Action: "write", Description: "Manage E2 interface"},
				{ID: "subscriptions:read", Resource: "subscriptions", Action: "read", Description: "Read subscriptions"},
				{ID: "subscriptions:write", Resource: "subscriptions", Action: "write", Description: "Manage subscriptions"},
				{ID: "policies:read", Resource: "policies", Action: "read", Description: "Read policies"},
				{ID: "policies:write", Resource: "policies", Action: "write", Description: "Manage policies"},
				{ID: "xapps:read", Resource: "xapps", Action: "read", Description: "Read xApp data"},
				{ID: "xapps:write", Resource: "xapps", Action: "write", Description: "Manage xApps"},
			},
		},
		{
			ID:          "viewer",
			Name:        "Read-Only Viewer",
			Description: "Read-only access to system data",
			IsSystem:    true,
			Permissions: []Permission{
				{ID: "e2:read", Resource: "e2", Action: "read", Description: "Read E2 interface data"},
				{ID: "subscriptions:read", Resource: "subscriptions", Action: "read", Description: "Read subscriptions"},
				{ID: "policies:read", Resource: "policies", Action: "read", Description: "Read policies"},
				{ID: "xapps:read", Resource: "xapps", Action: "read", Description: "Read xApp data"},
				{ID: "o1:read", Resource: "o1", Action: "read", Description: "Read O1 management data"},
			},
		},
		{
			ID:          "policy-manager",
			Name:        "Policy Manager",
			Description: "Policy management access",
			IsSystem:    true,
			Permissions: []Permission{
				{ID: "policies:read", Resource: "policies", Action: "read", Description: "Read policies"},
				{ID: "policies:write", Resource: "policies", Action: "write", Description: "Manage policies"},
				{ID: "policies:delete", Resource: "policies", Action: "delete", Description: "Delete policies"},
			},
		},
		{
			ID:          "xapp-developer",
			Name:        "xApp Developer",
			Description: "xApp development and deployment access",
			IsSystem:    true,
			Permissions: []Permission{
				{ID: "xapps:read", Resource: "xapps", Action: "read", Description: "Read xApp data"},
				{ID: "xapps:write", Resource: "xapps", Action: "write", Description: "Manage xApps"},
				{ID: "xapps:deploy", Resource: "xapps", Action: "deploy", Description: "Deploy xApps"},
				{ID: "subscriptions:read", Resource: "subscriptions", Action: "read", Description: "Read subscriptions"},
				{ID: "subscriptions:write", Resource: "subscriptions", Action: "write", Description: "Manage subscriptions"},
			},
		},
	}
)

// Event types for audit logging
const (
	EventTypeLogin                 = "login"
	EventTypeLogout                = "logout"
	EventTypeLoginFailed           = "login_failed"
	EventTypePasswordChanged       = "password_changed"
	EventTypeUserCreated           = "user_created"
	EventTypeUserUpdated           = "user_updated"
	EventTypeUserDeleted           = "user_deleted"
	EventTypeRoleCreated           = "role_created"
	EventTypeRoleUpdated           = "role_updated"
	EventTypeRoleDeleted           = "role_deleted"
	EventTypePermissionGranted     = "permission_granted"
	EventTypePermissionRevoked     = "permission_revoked"
	EventTypeAccessDenied          = "access_denied"
	EventTypeServiceAccountCreated = "service_account_created"
	EventTypeServiceAccountUpdated = "service_account_updated"
	EventTypeServiceAccountDeleted = "service_account_deleted"
)

// Result types for audit events
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultDenied  = "denied"
)
