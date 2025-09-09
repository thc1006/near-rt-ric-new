/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and authorization
type AuthService struct {
	users           map[string]*User
	serviceAccounts map[string]*ServiceAccount
	sessions        map[string]*Session
	jwtManager      *JWTManager
	rbacManager     *RBACManager
	auditLogger     *AuditLogger
	mutex           sync.RWMutex
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtManager *JWTManager, rbacManager *RBACManager, auditLogger *AuditLogger) *AuthService {
	auth := &AuthService{
		users:           make(map[string]*User),
		serviceAccounts: make(map[string]*ServiceAccount),
		sessions:        make(map[string]*Session),
		jwtManager:      jwtManager,
		rbacManager:     rbacManager,
		auditLogger:     auditLogger,
	}

	// Create default admin user
	auth.createDefaultAdmin()
	
	// Start session cleanup routine
	go auth.sessionCleanupRoutine()

	return auth
}

// createDefaultAdmin creates a default admin user
func (auth *AuthService) createDefaultAdmin() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to create default admin password: %v", err)
		return
	}

	admin := &User{
		ID:        "admin",
		Username:  "admin",
		Email:     "admin@oran-ric.org",
		FullName:  "System Administrator",
		Roles:     []string{"admin"},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	auth.users[admin.ID] = admin
	
	// Store password separately (in production, use proper password storage)
	auth.storePassword(admin.ID, string(hashedPassword))
}

// Login authenticates a user and returns a JWT token
func (auth *AuthService) Login(request *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	auth.mutex.RLock()
	var user *User
	for _, u := range auth.users {
		if u.Username == request.Username {
			user = u
			break
		}
	}
	auth.mutex.RUnlock()

	if user == nil {
		auth.auditLogger.LogEvent(&AuditEvent{
			EventType: EventTypeLoginFailed,
			Username:  request.Username,
			Resource:  "auth",
			Action:    "login",
			Result:    ResultFailure,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "user not found"},
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		auth.auditLogger.LogEvent(&AuditEvent{
			EventType: EventTypeLoginFailed,
			UserID:    user.ID,
			Username:  user.Username,
			Resource:  "auth",
			Action:    "login",
			Result:    ResultFailure,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "user inactive"},
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("user account is inactive")
	}

	// Verify password
	storedPassword := auth.getStoredPassword(user.ID)
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(request.Password)); err != nil {
		auth.auditLogger.LogEvent(&AuditEvent{
			EventType: EventTypeLoginFailed,
			UserID:    user.ID,
			Username:  user.Username,
			Resource:  "auth",
			Action:    "login",
			Result:    ResultFailure,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "invalid password"},
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("invalid credentials")
	}

	// Create session
	sessionID := auth.generateSessionID()
	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// Generate JWT token
	token, expiresAt, err := auth.jwtManager.GenerateToken(user, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session.Token = token
	session.ExpiresAt = expiresAt

	auth.mutex.Lock()
	auth.sessions[sessionID] = session
	user.LastLoginAt = &session.CreatedAt
	auth.mutex.Unlock()

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeLogin,
		UserID:    user.ID,
		Username:  user.Username,
		Resource:  "auth",
		Action:    "login",
		Result:    ResultSuccess,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	})

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	}, nil
}

// Logout invalidates a user session
func (auth *AuthService) Logout(sessionID string, ipAddress, userAgent string) error {
	auth.mutex.Lock()
	session, exists := auth.sessions[sessionID]
	if exists {
		delete(auth.sessions, sessionID)
	}
	auth.mutex.Unlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	user := auth.getUserByID(session.UserID)
	username := ""
	if user != nil {
		username = user.Username
	}

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeLogout,
		UserID:    session.UserID,
		Username:  username,
		Resource:  "auth",
		Action:    "logout",
		Result:    ResultSuccess,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	})

	return nil
}

// ValidateToken validates a JWT token and returns claims
func (auth *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims, err := auth.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Check if session exists and is valid
	auth.mutex.RLock()
	session, exists := auth.sessions[claims.SessionID]
	auth.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		auth.mutex.Lock()
		delete(auth.sessions, claims.SessionID)
		auth.mutex.Unlock()
		return nil, fmt.Errorf("session expired")
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token
func (auth *AuthService) RefreshToken(request *RefreshTokenRequest) (*LoginResponse, error) {
	claims, err := auth.ValidateToken(request.Token)
	if err != nil {
		return nil, fmt.Errorf("invalid token for refresh: %w", err)
	}

	user := auth.getUserByID(claims.UserID)
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new token
	newToken, expiresAt, err := auth.jwtManager.GenerateToken(user, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	// Update session
	auth.mutex.Lock()
	if session, exists := auth.sessions[claims.SessionID]; exists {
		session.Token = newToken
		session.ExpiresAt = expiresAt
	}
	auth.mutex.Unlock()

	return &LoginResponse{
		Token:     newToken,
		ExpiresAt: expiresAt,
		User:      *user,
	}, nil
}

// CreateUser creates a new user
func (auth *AuthService) CreateUser(request *CreateUserRequest, creatorID string) (*User, error) {
	// Validate role assignment
	if err := auth.rbacManager.ValidateRoleAssignment(request.Roles); err != nil {
		return nil, fmt.Errorf("invalid role assignment: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := auth.generateUserID()
	user := &User{
		ID:        userID,
		Username:  request.Username,
		Email:     request.Email,
		FullName:  request.FullName,
		Roles:     request.Roles,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	auth.mutex.Lock()
	auth.users[userID] = user
	auth.mutex.Unlock()

	// Store password
	auth.storePassword(userID, string(hashedPassword))

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeUserCreated,
		UserID:    creatorID,
		Resource:  "users",
		Action:    "create",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"createdUserId": userID,
			"username":      request.Username,
		},
		Timestamp: time.Now(),
	})

	return user, nil
}

// UpdateUser updates an existing user
func (auth *AuthService) UpdateUser(userID string, request *UpdateUserRequest, updaterID string) (*User, error) {
	auth.mutex.Lock()
	user, exists := auth.users[userID]
	if !exists {
		auth.mutex.Unlock()
		return nil, fmt.Errorf("user not found")
	}

	// Update fields
	if request.Email != nil {
		user.Email = *request.Email
	}
	if request.FullName != nil {
		user.FullName = *request.FullName
	}
	if request.Roles != nil {
		if err := auth.rbacManager.ValidateRoleAssignment(request.Roles); err != nil {
			auth.mutex.Unlock()
			return nil, fmt.Errorf("invalid role assignment: %w", err)
		}
		user.Roles = request.Roles
	}
	if request.IsActive != nil {
		user.IsActive = *request.IsActive
	}

	user.UpdatedAt = time.Now()
	auth.mutex.Unlock()

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeUserUpdated,
		UserID:    updaterID,
		Resource:  "users",
		Action:    "update",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"updatedUserId": userID,
			"username":      user.Username,
		},
		Timestamp: time.Now(),
	})

	return user, nil
}

// DeleteUser deletes a user
func (auth *AuthService) DeleteUser(userID string, deleterID string) error {
	auth.mutex.Lock()
	user, exists := auth.users[userID]
	if !exists {
		auth.mutex.Unlock()
		return fmt.Errorf("user not found")
	}

	// Don't allow deletion of admin user
	if user.Username == "admin" {
		auth.mutex.Unlock()
		return fmt.Errorf("cannot delete admin user")
	}

	delete(auth.users, userID)
	auth.mutex.Unlock()

	// Invalidate all sessions for this user
	auth.invalidateUserSessions(userID)

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeUserDeleted,
		UserID:    deleterID,
		Resource:  "users",
		Action:    "delete",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"deletedUserId": userID,
			"username":      user.Username,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// GetUser retrieves a user by ID
func (auth *AuthService) GetUser(userID string) (*User, error) {
	auth.mutex.RLock()
	user, exists := auth.users[userID]
	auth.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// GetAllUsers retrieves all users
func (auth *AuthService) GetAllUsers() []*User {
	auth.mutex.RLock()
	defer auth.mutex.RUnlock()

	users := make([]*User, 0, len(auth.users))
	for _, user := range auth.users {
		users = append(users, user)
	}

	return users
}

// ChangePassword changes a user's password
func (auth *AuthService) ChangePassword(userID string, request *ChangePasswordRequest) error {
	user := auth.getUserByID(userID)
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	storedPassword := auth.getStoredPassword(userID)
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(request.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Store new password
	auth.storePassword(userID, string(hashedPassword))

	auth.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypePasswordChanged,
		UserID:    userID,
		Username:  user.Username,
		Resource:  "auth",
		Action:    "change_password",
		Result:    ResultSuccess,
		Timestamp: time.Now(),
	})

	return nil
}

// CheckPermission checks if a user has permission to perform an action
func (auth *AuthService) CheckPermission(claims *JWTClaims, resource, action string) bool {
	return auth.rbacManager.CheckPermissionWithClaims(claims, resource, action)
}

// Helper methods

func (auth *AuthService) getUserByID(userID string) *User {
	auth.mutex.RLock()
	defer auth.mutex.RUnlock()
	return auth.users[userID]
}

func (auth *AuthService) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (auth *AuthService) generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (auth *AuthService) invalidateUserSessions(userID string) {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	for sessionID, session := range auth.sessions {
		if session.UserID == userID {
			delete(auth.sessions, sessionID)
		}
	}
}

func (auth *AuthService) sessionCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		auth.cleanupExpiredSessions()
	}
}

func (auth *AuthService) cleanupExpiredSessions() {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range auth.sessions {
		if now.After(session.ExpiresAt) {
			delete(auth.sessions, sessionID)
		}
	}
}

// In-memory password storage (for demo purposes)
// In production, use proper secure storage
var passwordStore = make(map[string]string)
var passwordMutex sync.RWMutex

func (auth *AuthService) storePassword(userID, hashedPassword string) {
	passwordMutex.Lock()
	defer passwordMutex.Unlock()
	passwordStore[userID] = hashedPassword
}

func (auth *AuthService) getStoredPassword(userID string) string {
	passwordMutex.RLock()
	defer passwordMutex.RUnlock()
	return passwordStore[userID]
}