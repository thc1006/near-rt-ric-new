/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// ServiceAccountHandlers handles service account-related HTTP requests
type ServiceAccountHandlers struct {
	serviceAccountManager *ServiceAccountManager
	auditLogger           *AuditLogger
}

// NewServiceAccountHandlers creates new service account handlers
func NewServiceAccountHandlers(serviceAccountManager *ServiceAccountManager, auditLogger *AuditLogger) *ServiceAccountHandlers {
	return &ServiceAccountHandlers{
		serviceAccountManager: serviceAccountManager,
		auditLogger:           auditLogger,
	}
}

// GetServiceAccountsHandler returns all service accounts (admin only)
func (sah *ServiceAccountHandlers) GetServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	serviceAccounts := sah.serviceAccountManager.GetAllServiceAccounts()
	sah.respondWithJSON(w, http.StatusOK, serviceAccounts)
}

// CreateServiceAccountHandler creates a new service account (admin only)
func (sah *ServiceAccountHandlers) CreateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	var request CreateServiceAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	serviceAccount, clientSecret, err := sah.serviceAccountManager.CreateServiceAccount(&request, claims.UserID)
	if err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Service account creation failed", err.Error())
		return
	}

	// Return service account with client credentials
	response := map[string]interface{}{
		"serviceAccount": serviceAccount,
		"clientId":       serviceAccount.ClientID,
		"clientSecret":   clientSecret,
	}

	sah.respondWithJSON(w, http.StatusCreated, response)
}

// GetServiceAccountHandler returns a specific service account (admin only)
func (sah *ServiceAccountHandlers) GetServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceAccountID := vars["serviceAccountId"]

	serviceAccount, err := sah.serviceAccountManager.GetServiceAccount(serviceAccountID)
	if err != nil {
		sah.respondWithError(w, http.StatusNotFound, "Service account not found", err.Error())
		return
	}

	sah.respondWithJSON(w, http.StatusOK, serviceAccount)
}

// UpdateServiceAccountHandler updates an existing service account (admin only)
func (sah *ServiceAccountHandlers) UpdateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	serviceAccountID := vars["serviceAccountId"]

	var request UpdateServiceAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	serviceAccount, err := sah.serviceAccountManager.UpdateServiceAccount(serviceAccountID, &request, claims.UserID)
	if err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Service account update failed", err.Error())
		return
	}

	sah.respondWithJSON(w, http.StatusOK, serviceAccount)
}

// DeleteServiceAccountHandler deletes a service account (admin only)
func (sah *ServiceAccountHandlers) DeleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	serviceAccountID := vars["serviceAccountId"]

	if err := sah.serviceAccountManager.DeleteServiceAccount(serviceAccountID, claims.UserID); err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Service account deletion failed", err.Error())
		return
	}

	sah.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Service account deleted successfully"})
}

// RotateServiceAccountSecretHandler rotates the client secret for a service account (admin only)
func (sah *ServiceAccountHandlers) RotateServiceAccountSecretHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sah.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	serviceAccountID := vars["serviceAccountId"]

	newClientSecret, err := sah.serviceAccountManager.RotateClientSecret(serviceAccountID, claims.UserID)
	if err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Secret rotation failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"message":      "Client secret rotated successfully",
		"clientSecret": newClientSecret,
	}

	sah.respondWithJSON(w, http.StatusOK, response)
}

// GetServiceAccountCredentialsHandler returns client credentials for a service account (admin only)
func (sah *ServiceAccountHandlers) GetServiceAccountCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceAccountID := vars["serviceAccountId"]

	clientID, clientSecret, err := sah.serviceAccountManager.GetServiceAccountCredentials(serviceAccountID)
	if err != nil {
		sah.respondWithError(w, http.StatusNotFound, "Service account credentials not found", err.Error())
		return
	}

	response := map[string]interface{}{
		"clientId":     clientID,
		"clientSecret": clientSecret,
	}

	sah.respondWithJSON(w, http.StatusOK, response)
}

// ServiceAccountTokenHandler generates a token for service account authentication
func (sah *ServiceAccountHandlers) ServiceAccountTokenHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sah.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Authenticate service account
	serviceAccount, err := sah.serviceAccountManager.AuthenticateServiceAccount(request.ClientID, request.ClientSecret)
	if err != nil {
		sah.respondWithError(w, http.StatusUnauthorized, "Authentication failed", err.Error())
		return
	}

	// Generate token
	token, expiresAt, err := sah.serviceAccountManager.GenerateServiceAccountToken(serviceAccount)
	if err != nil {
		sah.respondWithError(w, http.StatusInternalServerError, "Token generation failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"token":     token,
		"expiresAt": expiresAt,
		"tokenType": "Bearer",
	}

	sah.respondWithJSON(w, http.StatusOK, response)
}

// Helper methods

func (sah *ServiceAccountHandlers) respondWithError(w http.ResponseWriter, statusCode int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

func (sah *ServiceAccountHandlers) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}