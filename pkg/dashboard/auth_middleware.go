/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AuthMiddleware handles authentication and authorization for HTTP requests
type AuthMiddleware struct {
	authService *AuthService
	auditLogger *AuditLogger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *AuthService, auditLogger *AuditLogger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		auditLogger: auditLogger,
	}
}

// RequireAuth middleware that requires authentication
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := am.extractAndValidateToken(r)
		if err != nil {
			am.respondWithError(w, http.StatusUnauthorized, "Authentication required", err.Error())
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware that requires specific permission
func (am *AuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := am.extractAndValidateToken(r)
			if err != nil {
				am.respondWithError(w, http.StatusUnauthorized, "Authentication required", err.Error())
				return
			}

			// Check permission
			if !am.authService.CheckPermission(claims, resource, action) {
				am.auditLogger.LogAccessDenied(
					claims.UserID,
					claims.Username,
					resource,
					action,
					am.getClientIP(r),
					r.UserAgent(),
				)
				am.respondWithError(w, http.StatusForbidden, "Insufficient permissions", 
					"Access denied for resource: "+resource+", action: "+action)
				return
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware that requires specific role
func (am *AuthMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := am.extractAndValidateToken(r)
			if err != nil {
				am.respondWithError(w, http.StatusUnauthorized, "Authentication required", err.Error())
				return
			}

			// Check if user has required role
			hasRole := false
			for _, role := range claims.Roles {
				if role == requiredRole || role == "admin" { // Admin has access to everything
					hasRole = true
					break
				}
			}

			if !hasRole {
				am.auditLogger.LogAccessDenied(
					claims.UserID,
					claims.Username,
					"role",
					requiredRole,
					am.getClientIP(r),
					r.UserAgent(),
				)
				am.respondWithError(w, http.StatusForbidden, "Insufficient role", 
					"Required role: "+requiredRole)
				return
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth middleware that extracts auth info if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := am.extractAndValidateToken(r)
		
		// Add claims to request context (may be nil)
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminOnly middleware that requires admin role
func (am *AuthMiddleware) AdminOnly(next http.Handler) http.Handler {
	return am.RequireRole("admin")(next)
}

// OperatorOrAdmin middleware that requires operator or admin role
func (am *AuthMiddleware) OperatorOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := am.extractAndValidateToken(r)
		if err != nil {
			am.respondWithError(w, http.StatusUnauthorized, "Authentication required", err.Error())
			return
		}

		// Check if user has operator or admin role
		hasRole := false
		for _, role := range claims.Roles {
			if role == "operator" || role == "admin" {
				hasRole = true
				break
			}
		}

		if !hasRole {
			am.auditLogger.LogAccessDenied(
				claims.UserID,
				claims.Username,
				"role",
				"operator_or_admin",
				am.getClientIP(r),
				r.UserAgent(),
			)
			am.respondWithError(w, http.StatusForbidden, "Insufficient role", 
				"Required role: operator or admin")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAndValidateToken extracts and validates JWT token from request
func (am *AuthMiddleware) extractAndValidateToken(r *http.Request) (*JWTClaims, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	// Check for Bearer token format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := parts[1]
	
	// Validate token
	claims, err := am.authService.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// getClientIP extracts client IP address from request
func (am *AuthMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		// Remove port if present
		parts := strings.Split(ip, ":")
		if len(parts) > 1 {
			ip = strings.Join(parts[:len(parts)-1], ":")
		}
	}

	return ip
}

// respondWithError sends an error response
func (am *AuthMiddleware) respondWithError(w http.ResponseWriter, statusCode int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

// GetClaimsFromContext extracts JWT claims from request context
func GetClaimsFromContext(ctx context.Context) *JWTClaims {
	if claims, ok := ctx.Value("claims").(*JWTClaims); ok {
		return claims
	}
	return nil
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) string {
	if claims := GetClaimsFromContext(ctx); claims != nil {
		return claims.UserID
	}
	return ""
}

// GetUsernameFromContext extracts username from request context
func GetUsernameFromContext(ctx context.Context) string {
	if claims := GetClaimsFromContext(ctx); claims != nil {
		return claims.Username
	}
	return ""
}

// GetUserRolesFromContext extracts user roles from request context
func GetUserRolesFromContext(ctx context.Context) []string {
	if claims := GetClaimsFromContext(ctx); claims != nil {
		return claims.Roles
	}
	return nil
}

// HasPermissionInContext checks if the user in context has a specific permission
func HasPermissionInContext(ctx context.Context, authService *AuthService, resource, action string) bool {
	claims := GetClaimsFromContext(ctx)
	if claims == nil {
		return false
	}
	return authService.CheckPermission(claims, resource, action)
}

// HasRoleInContext checks if the user in context has a specific role
func HasRoleInContext(ctx context.Context, requiredRole string) bool {
	roles := GetUserRolesFromContext(ctx)
	for _, role := range roles {
		if role == requiredRole || role == "admin" {
			return true
		}
	}
	return false
}