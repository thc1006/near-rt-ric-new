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

// TLSHandlers handles TLS and certificate management HTTP requests
type TLSHandlers struct {
	tlsManager  *TLSManager
	auditLogger *AuditLogger
}

// NewTLSHandlers creates new TLS handlers
func NewTLSHandlers(tlsManager *TLSManager, auditLogger *AuditLogger) *TLSHandlers {
	return &TLSHandlers{
		tlsManager:  tlsManager,
		auditLogger: auditLogger,
	}
}

// GetCertificateInfoHandler returns certificate information
func (th *TLSHandlers) GetCertificateInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	certType := vars["certType"]

	if certType == "" {
		th.respondWithError(w, http.StatusBadRequest, "Certificate type is required", "")
		return
	}

	info, err := th.tlsManager.GetCertificateInfo(certType)
	if err != nil {
		th.respondWithError(w, http.StatusNotFound, "Certificate not found", err.Error())
		return
	}

	th.respondWithJSON(w, http.StatusOK, info)
}

// GetAllCertificatesHandler returns information about all certificates
func (th *TLSHandlers) GetAllCertificatesHandler(w http.ResponseWriter, r *http.Request) {
	certificates := make(map[string]*CertificateInfo)

	certTypes := []string{"ca", "server", "client"}
	for _, certType := range certTypes {
		if info, err := th.tlsManager.GetCertificateInfo(certType); err == nil {
			certificates[certType] = info
		}
	}

	th.respondWithJSON(w, http.StatusOK, certificates)
}

// RegenerateCertificateHandler regenerates a certificate
func (th *TLSHandlers) RegenerateCertificateHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		th.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	certType := vars["certType"]

	if certType == "" {
		th.respondWithError(w, http.StatusBadRequest, "Certificate type is required", "")
		return
	}

	var err error
	switch certType {
	case "server":
		err = th.tlsManager.generateServerCertificate()
	case "client":
		err = th.tlsManager.generateClientCertificate()
	case "ca":
		th.respondWithError(w, http.StatusBadRequest, "CA certificate regeneration not supported", "")
		return
	default:
		th.respondWithError(w, http.StatusBadRequest, "Invalid certificate type", "")
		return
	}

	if err != nil {
		th.auditLogger.LogEvent(&AuditEvent{
			EventType: "certificate_regeneration_failed",
			UserID:    claims.UserID,
			Username:  claims.Username,
			Resource:  "certificates",
			Action:    "regenerate",
			Result:    ResultFailure,
			Details: map[string]interface{}{
				"certType": certType,
				"error":    err.Error(),
			},
			Timestamp: time.Now(),
		})

		th.respondWithError(w, http.StatusInternalServerError, "Certificate regeneration failed", err.Error())
		return
	}

	th.auditLogger.LogEvent(&AuditEvent{
		EventType: "certificate_regenerated",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "certificates",
		Action:    "regenerate",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"certType": certType,
		},
		Timestamp: time.Now(),
	})

	th.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Certificate regenerated successfully",
		"type":    certType,
	})
}

// GetTLSConfigHandler returns TLS configuration information
func (th *TLSHandlers) GetTLSConfigHandler(w http.ResponseWriter, r *http.Request) {
	serverConfig := th.tlsManager.GetServerTLSConfig()
	clientConfig := th.tlsManager.GetClientTLSConfig()

	configInfo := map[string]interface{}{
		"server": map[string]interface{}{
			"minVersion":   "TLS 1.3",
			"maxVersion":   "TLS 1.3",
			"clientAuth":   "RequireAndVerifyClientCert",
			"certificates": len(serverConfig.Certificates),
		},
		"client": map[string]interface{}{
			"minVersion":   "TLS 1.3",
			"maxVersion":   "TLS 1.3",
			"certificates": len(clientConfig.Certificates),
		},
		"cipherSuites": []string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"TLS_AES_128_GCM_SHA256",
		},
	}

	th.respondWithJSON(w, http.StatusOK, configInfo)
}

// CheckCertificateExpiryHandler checks certificate expiry status
func (th *TLSHandlers) CheckCertificateExpiryHandler(w http.ResponseWriter, r *http.Request) {
	expiryStatus := make(map[string]interface{})

	certTypes := []string{"ca", "server", "client"}
	for _, certType := range certTypes {
		if info, err := th.tlsManager.GetCertificateInfo(certType); err == nil {
			status := "valid"
			if info.IsExpired {
				status = "expired"
			} else if info.DaysToExpiry <= 30 {
				status = "expiring_soon"
			}

			expiryStatus[certType] = map[string]interface{}{
				"status":       status,
				"daysToExpiry": info.DaysToExpiry,
				"notAfter":     info.NotAfter,
				"isExpired":    info.IsExpired,
			}
		}
	}

	th.respondWithJSON(w, http.StatusOK, expiryStatus)
}

// RotateCertificatesHandler manually triggers certificate rotation
func (th *TLSHandlers) RotateCertificatesHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		th.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	// Trigger certificate rotation check
	th.tlsManager.checkAndRotateCertificates()

	th.auditLogger.LogEvent(&AuditEvent{
		EventType: "certificate_rotation_triggered",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "certificates",
		Action:    "rotate",
		Result:    ResultSuccess,
		Timestamp: time.Now(),
	})

	th.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Certificate rotation check completed",
	})
}

// GetCertificateStatsHandler returns certificate statistics
func (th *TLSHandlers) GetCertificateStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"totalCertificates": 0,
		"validCertificates": 0,
		"expiredCertificates": 0,
		"expiringSoonCertificates": 0,
		"certificates": make(map[string]interface{}),
	}

	certTypes := []string{"ca", "server", "client"}
	certificates := stats["certificates"].(map[string]interface{})

	for _, certType := range certTypes {
		if info, err := th.tlsManager.GetCertificateInfo(certType); err == nil {
			stats["totalCertificates"] = stats["totalCertificates"].(int) + 1

			if info.IsExpired {
				stats["expiredCertificates"] = stats["expiredCertificates"].(int) + 1
			} else if info.DaysToExpiry <= 30 {
				stats["expiringSoonCertificates"] = stats["expiringSoonCertificates"].(int) + 1
			} else {
				stats["validCertificates"] = stats["validCertificates"].(int) + 1
			}

			certificates[certType] = map[string]interface{}{
				"subject":      info.Subject,
				"notAfter":     info.NotAfter,
				"daysToExpiry": info.DaysToExpiry,
				"isExpired":    info.IsExpired,
				"isCA":         info.IsCA,
			}
		}
	}

	th.respondWithJSON(w, http.StatusOK, stats)
}

// ValidateTLSConfigHandler validates TLS configuration
func (th *TLSHandlers) ValidateTLSConfigHandler(w http.ResponseWriter, r *http.Request) {
	validation := map[string]interface{}{
		"valid": true,
		"issues": []string{},
		"recommendations": []string{},
	}

	issues := validation["issues"].([]string)
	recommendations := validation["recommendations"].([]string)

	// Check server certificate
	if serverInfo, err := th.tlsManager.GetCertificateInfo("server"); err != nil {
		issues = append(issues, "Server certificate not found or invalid")
		validation["valid"] = false
	} else {
		if serverInfo.IsExpired {
			issues = append(issues, "Server certificate is expired")
			validation["valid"] = false
		} else if serverInfo.DaysToExpiry <= 30 {
			recommendations = append(recommendations, "Server certificate expires soon, consider renewal")
		}
	}

	// Check client certificate
	if clientInfo, err := th.tlsManager.GetCertificateInfo("client"); err != nil {
		issues = append(issues, "Client certificate not found or invalid")
		validation["valid"] = false
	} else {
		if clientInfo.IsExpired {
			issues = append(issues, "Client certificate is expired")
			validation["valid"] = false
		} else if clientInfo.DaysToExpiry <= 30 {
			recommendations = append(recommendations, "Client certificate expires soon, consider renewal")
		}
	}

	// Check CA certificate
	if caInfo, err := th.tlsManager.GetCertificateInfo("ca"); err != nil {
		issues = append(issues, "CA certificate not found or invalid")
		validation["valid"] = false
	} else {
		if caInfo.IsExpired {
			issues = append(issues, "CA certificate is expired")
			validation["valid"] = false
		}
	}

	// Check TLS configuration
	serverConfig := th.tlsManager.GetServerTLSConfig()
	if serverConfig == nil {
		issues = append(issues, "Server TLS configuration is not available")
		validation["valid"] = false
	} else {
		if serverConfig.MinVersion < 0x0304 { // TLS 1.3
			recommendations = append(recommendations, "Consider using TLS 1.3 as minimum version")
		}
	}

	validation["issues"] = issues
	validation["recommendations"] = recommendations

	statusCode := http.StatusOK
	if !validation["valid"].(bool) {
		statusCode = http.StatusBadRequest
	}

	th.respondWithJSON(w, statusCode, validation)
}

// Helper methods

func (th *TLSHandlers) respondWithError(w http.ResponseWriter, statusCode int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

func (th *TLSHandlers) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}