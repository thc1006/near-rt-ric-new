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
	"sync"
	"time"
)

// ServiceAccountManager manages service accounts for component authentication
type ServiceAccountManager struct {
	serviceAccounts map[string]*ServiceAccount
	clientSecrets   map[string]string // clientID -> secret
	jwtManager      *JWTManager
	rbacManager     *RBACManager
	auditLogger     *AuditLogger
	mutex           sync.RWMutex
}

// NewServiceAccountManager creates a new service account manager
func NewServiceAccountManager(jwtManager *JWTManager, rbacManager *RBACManager, auditLogger *AuditLogger) *ServiceAccountManager {
	sam := &ServiceAccountManager{
		serviceAccounts: make(map[string]*ServiceAccount),
		clientSecrets:   make(map[string]string),
		jwtManager:      jwtManager,
		rbacManager:     rbacManager,
		auditLogger:     auditLogger,
	}

	// Create default service accounts for O-RAN SC components
	sam.createDefaultServiceAccounts()

	return sam
}

// createDefaultServiceAccounts creates default service accounts for O-RAN SC components
func (sam *ServiceAccountManager) createDefaultServiceAccounts() {
	defaultAccounts := []struct {
		name        string
		description string
		roles       []string
	}{
		{
			name:        "e2mgr-service",
			description: "E2 Manager service account",
			roles:       []string{"operator"},
		},
		{
			name:        "submgr-service",
			description: "Subscription Manager service account",
			roles:       []string{"operator"},
		},
		{
			name:        "e2term-service",
			description: "E2 Termination service account",
			roles:       []string{"operator"},
		},
		{
			name:        "a1mediator-service",
			description: "A1 Mediator service account",
			roles:       []string{"policy-manager"},
		},
		{
			name:        "o1mediator-service",
			description: "O1 Mediator service account",
			roles:       []string{"admin"},
		},
		{
			name:        "rtmgr-service",
			description: "Routing Manager service account",
			roles:       []string{"operator"},
		},
		{
			name:        "appmgr-service",
			description: "Application Manager service account",
			roles:       []string{"xapp-developer"},
		},
	}

	for _, account := range defaultAccounts {
		request := &CreateServiceAccountRequest{
			Name:        account.name,
			Description: account.description,
			Roles:       account.roles,
		}

		_, err := sam.CreateServiceAccount(request, "system")
		if err != nil {
			fmt.Printf("Failed to create default service account %s: %v\n", account.name, err)
		}
	}
}

// CreateServiceAccount creates a new service account
func (sam *ServiceAccountManager) CreateServiceAccount(request *CreateServiceAccountRequest, creatorID string) (*ServiceAccount, string, error) {
	// Validate role assignment
	if err := sam.rbacManager.ValidateRoleAssignment(request.Roles); err != nil {
		return nil, "", fmt.Errorf("invalid role assignment: %w", err)
	}

	// Generate unique IDs
	serviceAccountID := sam.generateServiceAccountID()
	clientID := sam.generateClientID()
	clientSecret := sam.generateClientSecret()

	serviceAccount := &ServiceAccount{
		ID:          serviceAccountID,
		Name:        request.Name,
		Description: request.Description,
		ClientID:    clientID,
		Roles:       request.Roles,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	sam.mutex.Lock()
	sam.serviceAccounts[serviceAccountID] = serviceAccount
	sam.clientSecrets[clientID] = clientSecret
	sam.mutex.Unlock()

	sam.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeServiceAccountCreated,
		UserID:    creatorID,
		Resource:  "service_accounts",
		Action:    "create",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"serviceAccountId":   serviceAccountID,
			"serviceAccountName": request.Name,
			"clientId":           clientID,
		},
		Timestamp: time.Now(),
	})

	return serviceAccount, clientSecret, nil
}

// UpdateServiceAccount updates an existing service account
func (sam *ServiceAccountManager) UpdateServiceAccount(serviceAccountID string, request *UpdateServiceAccountRequest, updaterID string) (*ServiceAccount, error) {
	sam.mutex.Lock()
	serviceAccount, exists := sam.serviceAccounts[serviceAccountID]
	if !exists {
		sam.mutex.Unlock()
		return nil, fmt.Errorf("service account not found")
	}

	// Update fields
	if request.Name != nil {
		serviceAccount.Name = *request.Name
	}
	if request.Description != nil {
		serviceAccount.Description = *request.Description
	}
	if request.Roles != nil {
		if err := sam.rbacManager.ValidateRoleAssignment(request.Roles); err != nil {
			sam.mutex.Unlock()
			return nil, fmt.Errorf("invalid role assignment: %w", err)
		}
		serviceAccount.Roles = request.Roles
	}
	if request.IsActive != nil {
		serviceAccount.IsActive = *request.IsActive
	}

	serviceAccount.UpdatedAt = time.Now()
	sam.mutex.Unlock()

	sam.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeServiceAccountUpdated,
		UserID:    updaterID,
		Resource:  "service_accounts",
		Action:    "update",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"serviceAccountId":   serviceAccountID,
			"serviceAccountName": serviceAccount.Name,
		},
		Timestamp: time.Now(),
	})

	return serviceAccount, nil
}

// DeleteServiceAccount deletes a service account
func (sam *ServiceAccountManager) DeleteServiceAccount(serviceAccountID string, deleterID string) error {
	sam.mutex.Lock()
	serviceAccount, exists := sam.serviceAccounts[serviceAccountID]
	if !exists {
		sam.mutex.Unlock()
		return fmt.Errorf("service account not found")
	}

	clientID := serviceAccount.ClientID
	delete(sam.serviceAccounts, serviceAccountID)
	delete(sam.clientSecrets, clientID)
	sam.mutex.Unlock()

	sam.auditLogger.LogEvent(&AuditEvent{
		EventType: EventTypeServiceAccountDeleted,
		UserID:    deleterID,
		Resource:  "service_accounts",
		Action:    "delete",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"serviceAccountId":   serviceAccountID,
			"serviceAccountName": serviceAccount.Name,
			"clientId":           clientID,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// GetServiceAccount retrieves a service account by ID
func (sam *ServiceAccountManager) GetServiceAccount(serviceAccountID string) (*ServiceAccount, error) {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	serviceAccount, exists := sam.serviceAccounts[serviceAccountID]
	if !exists {
		return nil, fmt.Errorf("service account not found")
	}

	return serviceAccount, nil
}

// GetServiceAccountByClientID retrieves a service account by client ID
func (sam *ServiceAccountManager) GetServiceAccountByClientID(clientID string) (*ServiceAccount, error) {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	for _, serviceAccount := range sam.serviceAccounts {
		if serviceAccount.ClientID == clientID {
			return serviceAccount, nil
		}
	}

	return nil, fmt.Errorf("service account not found")
}

// GetAllServiceAccounts retrieves all service accounts
func (sam *ServiceAccountManager) GetAllServiceAccounts() []*ServiceAccount {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	serviceAccounts := make([]*ServiceAccount, 0, len(sam.serviceAccounts))
	for _, serviceAccount := range sam.serviceAccounts {
		serviceAccounts = append(serviceAccounts, serviceAccount)
	}

	return serviceAccounts
}

// AuthenticateServiceAccount authenticates a service account using client credentials
func (sam *ServiceAccountManager) AuthenticateServiceAccount(clientID, clientSecret string) (*ServiceAccount, error) {
	sam.mutex.RLock()
	storedSecret, exists := sam.clientSecrets[clientID]
	sam.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Verify client secret
	if !sam.verifyClientSecret(clientSecret, storedSecret) {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Get service account
	serviceAccount, err := sam.GetServiceAccountByClientID(clientID)
	if err != nil {
		return nil, err
	}

	if !serviceAccount.IsActive {
		return nil, fmt.Errorf("service account is inactive")
	}

	// Update last used timestamp
	sam.mutex.Lock()
	now := time.Now()
	serviceAccount.LastUsedAt = &now
	sam.mutex.Unlock()

	return serviceAccount, nil
}

// GenerateServiceAccountToken generates a JWT token for a service account
func (sam *ServiceAccountManager) GenerateServiceAccountToken(serviceAccount *ServiceAccount) (string, time.Time, error) {
	return sam.jwtManager.GenerateServiceAccountToken(serviceAccount)
}

// RotateClientSecret rotates the client secret for a service account
func (sam *ServiceAccountManager) RotateClientSecret(serviceAccountID string, rotatorID string) (string, error) {
	sam.mutex.Lock()
	serviceAccount, exists := sam.serviceAccounts[serviceAccountID]
	if !exists {
		sam.mutex.Unlock()
		return "", fmt.Errorf("service account not found")
	}

	// Generate new client secret
	newClientSecret := sam.generateClientSecret()
	sam.clientSecrets[serviceAccount.ClientID] = newClientSecret
	serviceAccount.UpdatedAt = time.Now()
	sam.mutex.Unlock()

	sam.auditLogger.LogEvent(&AuditEvent{
		EventType: "service_account_secret_rotated",
		UserID:    rotatorID,
		Resource:  "service_accounts",
		Action:    "rotate_secret",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"serviceAccountId":   serviceAccountID,
			"serviceAccountName": serviceAccount.Name,
			"clientId":           serviceAccount.ClientID,
		},
		Timestamp: time.Now(),
	})

	return newClientSecret, nil
}

// ValidateServiceAccountToken validates a service account token
func (sam *ServiceAccountManager) ValidateServiceAccountToken(tokenString string) (*JWTClaims, error) {
	claims, err := sam.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Verify service account exists and is active
	serviceAccount, err := sam.GetServiceAccount(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("service account not found")
	}

	if !serviceAccount.IsActive {
		return nil, fmt.Errorf("service account is inactive")
	}

	return claims, nil
}

// Helper methods

func (sam *ServiceAccountManager) generateServiceAccountID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "sa-" + hex.EncodeToString(bytes)
}

func (sam *ServiceAccountManager) generateClientID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (sam *ServiceAccountManager) generateClientSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (sam *ServiceAccountManager) verifyClientSecret(provided, stored string) bool {
	// In production, use proper secret hashing and comparison
	// For now, using simple comparison
	providedHash := sha256.Sum256([]byte(provided))
	storedHash := sha256.Sum256([]byte(stored))
	return hex.EncodeToString(providedHash[:]) == hex.EncodeToString(storedHash[:])
}

// GetServiceAccountCredentials returns the client credentials for a service account (admin only)
func (sam *ServiceAccountManager) GetServiceAccountCredentials(serviceAccountID string) (string, string, error) {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	serviceAccount, exists := sam.serviceAccounts[serviceAccountID]
	if !exists {
		return "", "", fmt.Errorf("service account not found")
	}

	clientSecret, exists := sam.clientSecrets[serviceAccount.ClientID]
	if !exists {
		return "", "", fmt.Errorf("client secret not found")
	}

	return serviceAccount.ClientID, clientSecret, nil
}