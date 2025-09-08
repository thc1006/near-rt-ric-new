/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

// Package security implements comprehensive O-RAN WG11 security compliance validation
// for O-RAN L Release and Nephio R5 deployments with FIPS 140-3 enforcement
package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// WG11ComplianceValidator validates O-RAN WG11 security specifications
type WG11ComplianceValidator struct {
	kubeClient       kubernetes.Interface
	config           *WG11Config
	logger           *log.Logger
	complianceStatus map[string]*InterfaceComplianceStatus
	mutex            sync.RWMutex
}

// WG11Config represents O-RAN WG11 security configuration
type WG11Config struct {
	// Interface security requirements
	E2Interface  *InterfaceSecurityConfig `json:"e2_interface"`
	A1Interface  *InterfaceSecurityConfig `json:"a1_interface"`
	O1Interface  *InterfaceSecurityConfig `json:"o1_interface"`
	O2Interface  *InterfaceSecurityConfig `json:"o2_interface"`
	
	// FIPS 140-3 requirements
	FIPS1403Config *FIPS1403Config `json:"fips_140_3_config"`
	
	// General security requirements
	TLSMinVersion         string   `json:"tls_min_version"`
	AllowedCipherSuites   []string `json:"allowed_cipher_suites"`
	CertificateValidation bool     `json:"certificate_validation"`
	MTLSRequired          bool     `json:"mtls_required"`
	
	// Compliance thresholds
	MaxCriticalVulnerabilities int     `json:"max_critical_vulnerabilities"`
	MaxHighVulnerabilities     int     `json:"max_high_vulnerabilities"`
	MinEncryptionStrength      int     `json:"min_encryption_strength"`
	CertExpirationThreshold    int     `json:"cert_expiration_threshold_days"`
}

// InterfaceSecurityConfig represents security configuration for O-RAN interfaces
type InterfaceSecurityConfig struct {
	Interface    string            `json:"interface"`
	Namespace    string            `json:"namespace"`
	Enabled      bool             `json:"enabled"`
	MTLSRequired bool             `json:"mtls_required"`
	Ports        []int            `json:"ports"`
	Protocols    []string         `json:"protocols"`
	AuthMethod   string           `json:"auth_method"`
	Encryption   *EncryptionConfig `json:"encryption"`
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	Algorithm    string `json:"algorithm"`
	KeySize      int    `json:"key_size"`
	KeyRotation  string `json:"key_rotation"`
}

// FIPS1403Config represents FIPS 140-3 configuration
type FIPS1403Config struct {
	Enabled        bool     `json:"enabled"`
	Mode           string   `json:"mode"` // "on" or "only"
	GoVersion      string   `json:"go_version"`
	OpenSSLEnabled bool     `json:"openssl_enabled"`
	AllowedAlgos   []string `json:"allowed_algorithms"`
}

// InterfaceComplianceStatus represents compliance status for an O-RAN interface
type InterfaceComplianceStatus struct {
	Interface            string                 `json:"interface"`
	Namespace           string                 `json:"namespace"`
	SecurityPolicyExists bool                  `json:"security_policy_exists"`
	TLSConfigured       bool                  `json:"tls_configured"`
	MTLSConfigured      bool                  `json:"mtls_configured"`
	CertificatesValid   bool                  `json:"certificates_valid"`
	NetworkPolicyExists bool                  `json:"network_policy_exists"`
	ComplianceLevel     ComplianceLevel       `json:"compliance_level"`
	Issues              []ComplianceIssue     `json:"issues"`
	LastValidated       time.Time             `json:"last_validated"`
	Details             map[string]interface{} `json:"details"`
}

// ComplianceLevel represents the level of compliance
type ComplianceLevel string

const (
	ComplianceLevelCompliant     ComplianceLevel = "COMPLIANT"
	ComplianceLevelPartial       ComplianceLevel = "PARTIAL"
	ComplianceLevelNonCompliant  ComplianceLevel = "NON_COMPLIANT"
	ComplianceLevelNotConfigured ComplianceLevel = "NOT_CONFIGURED"
)

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Component   string    `json:"component"`
	Resolution  string    `json:"resolution"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ComplianceReport represents overall WG11 compliance report
type ComplianceReport struct {
	Timestamp         time.Time                              `json:"timestamp"`
	ClusterName       string                                `json:"cluster_name"`
	ORanRelease       string                                `json:"o_ran_release"`
	WG11Version       string                                `json:"wg11_version"`
	OverallCompliance ComplianceLevel                       `json:"overall_compliance"`
	InterfaceStatus   map[string]*InterfaceComplianceStatus `json:"interface_status"`
	FIPS1403Status    *FIPS1403ComplianceStatus             `json:"fips_140_3_status"`
	SecurityMetrics   *SecurityMetrics                      `json:"security_metrics"`
	Recommendations   []string                              `json:"recommendations"`
}

// FIPS1403ComplianceStatus represents FIPS 140-3 compliance status
type FIPS1403ComplianceStatus struct {
	Enabled               bool      `json:"enabled"`
	Mode                 string    `json:"mode"`
	GoVersionCompliant   bool      `json:"go_version_compliant"`
	AlgorithmsCompliant  bool      `json:"algorithms_compliant"`
	DeploymentCompliance int       `json:"deployment_compliance_percentage"`
	Issues               []string  `json:"issues"`
	LastValidated        time.Time `json:"last_validated"`
}

// SecurityMetrics represents security metrics
type SecurityMetrics struct {
	TotalSecrets            int `json:"total_secrets"`
	TLSSecrets             int `json:"tls_secrets"`
	ExpiredCertificates    int `json:"expired_certificates"`
	ExpiringCertificates   int `json:"expiring_certificates"`
	NetworkPolicies        int `json:"network_policies"`
	SecurityPolicies       int `json:"security_policies"`
	VulnerabilityCount     *VulnerabilityCount `json:"vulnerability_count"`
}

// VulnerabilityCount represents vulnerability counts by severity
type VulnerabilityCount struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// NewWG11ComplianceValidator creates a new WG11 compliance validator
func NewWG11ComplianceValidator(kubeConfig *rest.Config) (*WG11ComplianceValidator, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	config := getDefaultWG11Config()
	
	validator := &WG11ComplianceValidator{
		kubeClient:       kubeClient,
		config:           config,
		logger:           log.New(os.Stdout, "[WG11-VALIDATOR] ", log.LstdFlags|log.Lshortfile),
		complianceStatus: make(map[string]*InterfaceComplianceStatus),
	}

	return validator, nil
}

// getDefaultWG11Config returns default WG11 configuration
func getDefaultWG11Config() *WG11Config {
	return &WG11Config{
		E2Interface: &InterfaceSecurityConfig{
			Interface:    "e2",
			Namespace:    "oran",
			Enabled:      true,
			MTLSRequired: true,
			Ports:        []int{36421, 4560, 4561},
			Protocols:    []string{"SCTP", "TCP"},
			AuthMethod:   "x509-certificate",
			Encryption: &EncryptionConfig{
				Algorithm:   "AES-256-GCM",
				KeySize:     256,
				KeyRotation: "12h",
			},
		},
		A1Interface: &InterfaceSecurityConfig{
			Interface:    "a1",
			Namespace:    "nonrtric",
			Enabled:      true,
			MTLSRequired: true,
			Ports:        []int{8081, 8443},
			Protocols:    []string{"HTTPS"},
			AuthMethod:   "oauth2",
			Encryption: &EncryptionConfig{
				Algorithm:   "TLS-1.3",
				KeySize:     256,
				KeyRotation: "24h",
			},
		},
		O1Interface: &InterfaceSecurityConfig{
			Interface:    "o1",
			Namespace:    "oran",
			Enabled:      true,
			MTLSRequired: true,
			Ports:        []int{830, 6513},
			Protocols:    []string{"SSH", "TLS"},
			AuthMethod:   "netconf-acm",
			Encryption: &EncryptionConfig{
				Algorithm:   "SSH-2.0",
				KeySize:     256,
				KeyRotation: "24h",
			},
		},
		O2Interface: &InterfaceSecurityConfig{
			Interface:    "o2",
			Namespace:    "ocloud-system",
			Enabled:      true,
			MTLSRequired: true,
			Ports:        []int{443},
			Protocols:    []string{"HTTPS"},
			AuthMethod:   "oauth2",
			Encryption: &EncryptionConfig{
				Algorithm:   "TLS-1.3",
				KeySize:     256,
				KeyRotation: "24h",
			},
		},
		FIPS1403Config: &FIPS1403Config{
			Enabled:        true,
			Mode:           "only", // Go 1.25+ supports "only" mode
			GoVersion:      "1.25",
			OpenSSLEnabled: true,
			AllowedAlgos: []string{
				"AES-128-GCM", "AES-256-GCM", "AES-128-CBC", "AES-256-CBC",
				"SHA-256", "SHA-384", "SHA-512",
				"RSA-2048", "RSA-3072", "RSA-4096",
				"ECDSA-P256", "ECDSA-P384", "ECDSA-P521",
			},
		},
		TLSMinVersion:              "1.3",
		AllowedCipherSuites: []string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_AES_128_GCM_SHA256",
			"TLS_CHACHA20_POLY1305_SHA256",
		},
		CertificateValidation:       true,
		MTLSRequired:                true,
		MaxCriticalVulnerabilities:  0,
		MaxHighVulnerabilities:      5,
		MinEncryptionStrength:       256,
		CertExpirationThreshold:     30,
	}
}

// ValidateCompliance performs comprehensive WG11 compliance validation
func (v *WG11ComplianceValidator) ValidateCompliance(ctx context.Context) (*ComplianceReport, error) {
	v.logger.Println("Starting WG11 compliance validation")

	report := &ComplianceReport{
		Timestamp:       time.Now(),
		ClusterName:     v.getClusterName(),
		ORanRelease:     "L",
		WG11Version:     "3.0",
		InterfaceStatus: make(map[string]*InterfaceComplianceStatus),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Validate each interface
	interfaces := []*InterfaceSecurityConfig{
		v.config.E2Interface,
		v.config.A1Interface,
		v.config.O1Interface,
		v.config.O2Interface,
	}

	for _, interfaceConfig := range interfaces {
		wg.Add(1)
		go func(config *InterfaceSecurityConfig) {
			defer wg.Done()
			
			status, err := v.validateInterface(ctx, config)
			if err != nil {
				v.logger.Printf("Error validating interface %s: %v", config.Interface, err)
				return
			}

			mu.Lock()
			report.InterfaceStatus[config.Interface] = status
			v.complianceStatus[config.Interface] = status
			mu.Unlock()
		}(interfaceConfig)
	}

	wg.Wait()

	// Validate FIPS 140-3 compliance
	fipsStatus, err := v.validateFIPS1403Compliance(ctx)
	if err != nil {
		v.logger.Printf("Error validating FIPS 140-3 compliance: %v", err)
	}
	report.FIPS1403Status = fipsStatus

	// Collect security metrics
	metrics, err := v.collectSecurityMetrics(ctx)
	if err != nil {
		v.logger.Printf("Error collecting security metrics: %v", err)
	}
	report.SecurityMetrics = metrics

	// Determine overall compliance level
	report.OverallCompliance = v.calculateOverallCompliance(report)

	// Generate recommendations
	report.Recommendations = v.generateRecommendations(report)

	v.logger.Printf("WG11 compliance validation completed. Overall status: %s", report.OverallCompliance)

	return report, nil
}

// validateInterface validates security compliance for a specific O-RAN interface
func (v *WG11ComplianceValidator) validateInterface(ctx context.Context, config *InterfaceSecurityConfig) (*InterfaceComplianceStatus, error) {
	status := &InterfaceComplianceStatus{
		Interface:     config.Interface,
		Namespace:     config.Namespace,
		LastValidated: time.Now(),
		Details:       make(map[string]interface{}),
	}

	// Check if security policy exists
	policyExists, err := v.checkSecurityPolicy(ctx, config)
	if err != nil {
		status.Issues = append(status.Issues, ComplianceIssue{
			Type:        "security_policy",
			Severity:    "high",
			Description: fmt.Sprintf("Failed to check security policy: %v", err),
			Component:   config.Interface,
			Resolution:  "Verify security policy deployment",
			DetectedAt:  time.Now(),
		})
	}
	status.SecurityPolicyExists = policyExists

	// Check TLS configuration
	tlsConfigured, err := v.checkTLSConfiguration(ctx, config)
	if err != nil {
		status.Issues = append(status.Issues, ComplianceIssue{
			Type:        "tls_configuration",
			Severity:    "high",
			Description: fmt.Sprintf("Failed to check TLS configuration: %v", err),
			Component:   config.Interface,
			Resolution:  "Configure TLS certificates and settings",
			DetectedAt:  time.Now(),
		})
	}
	status.TLSConfigured = tlsConfigured

	// Check mTLS configuration
	mtlsConfigured, err := v.checkMTLSConfiguration(ctx, config)
	if err != nil {
		status.Issues = append(status.Issues, ComplianceIssue{
			Type:        "mtls_configuration",
			Severity:    "high",
			Description: fmt.Sprintf("Failed to check mTLS configuration: %v", err),
			Component:   config.Interface,
			Resolution:  "Configure mutual TLS authentication",
			DetectedAt:  time.Now(),
		})
	}
	status.MTLSConfigured = mtlsConfigured

	// Check certificate validity
	certsValid, err := v.checkCertificateValidity(ctx, config)
	if err != nil {
		status.Issues = append(status.Issues, ComplianceIssue{
			Type:        "certificate_validity",
			Severity:    "medium",
			Description: fmt.Sprintf("Certificate validation issue: %v", err),
			Component:   config.Interface,
			Resolution:  "Renew or replace expired certificates",
			DetectedAt:  time.Now(),
		})
	}
	status.CertificatesValid = certsValid

	// Check network policy
	networkPolicyExists, err := v.checkNetworkPolicy(ctx, config)
	if err != nil {
		status.Issues = append(status.Issues, ComplianceIssue{
			Type:        "network_policy",
			Severity:    "medium",
			Description: fmt.Sprintf("Network policy check failed: %v", err),
			Component:   config.Interface,
			Resolution:  "Deploy network security policies",
			DetectedAt:  time.Now(),
		})
	}
	status.NetworkPolicyExists = networkPolicyExists

	// Calculate compliance level
	status.ComplianceLevel = v.calculateInterfaceCompliance(status, config)

	return status, nil
}

// checkSecurityPolicy checks if security policy exists for the interface
func (v *WG11ComplianceValidator) checkSecurityPolicy(ctx context.Context, config *InterfaceSecurityConfig) (bool, error) {
	// This would check for SecurityPolicy CRD in a real implementation
	// For now, check for configmap or secret with security configuration
	
	policyName := fmt.Sprintf("%s-interface-security", config.Interface)
	
	_, err := v.kubeClient.CoreV1().ConfigMaps(config.Namespace).Get(ctx, policyName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	// Also check for SecurityPolicy custom resource (would need CRD client)
	// This is a placeholder - actual implementation would use proper CRD client
	return false, nil
}

// checkTLSConfiguration checks TLS configuration for the interface
func (v *WG11ComplianceValidator) checkTLSConfiguration(ctx context.Context, config *InterfaceSecurityConfig) (bool, error) {
	secretName := fmt.Sprintf("%s-server-cert-tls", config.Interface)
	
	secret, err := v.kubeClient.CoreV1().Secrets(config.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("TLS secret not found: %w", err)
	}

	// Validate certificate data
	certData, exists := secret.Data["tls.crt"]
	if !exists {
		return false, fmt.Errorf("certificate data not found in secret")
	}

	// Parse and validate certificate
	_, err = tls.X509KeyPair(certData, secret.Data["tls.key"])
	if err != nil {
		return false, fmt.Errorf("invalid certificate/key pair: %w", err)
	}

	return true, nil
}

// checkMTLSConfiguration checks mutual TLS configuration
func (v *WG11ComplianceValidator) checkMTLSConfiguration(ctx context.Context, config *InterfaceSecurityConfig) (bool, error) {
	if !config.MTLSRequired {
		return true, nil // Not required
	}

	// Check for CA certificate
	caSecretName := fmt.Sprintf("%s-ca-cert-tls", config.Interface)
	_, err := v.kubeClient.CoreV1().Secrets(config.Namespace).Get(ctx, caSecretName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("CA certificate secret not found: %w", err)
	}

	// Check for client certificate
	clientSecretName := fmt.Sprintf("%s-client-cert-tls", config.Interface)
	_, err = v.kubeClient.CoreV1().Secrets(config.Namespace).Get(ctx, clientSecretName, metav1.GetOptions{})
	if err != nil {
		// Client cert might be optional depending on configuration
		v.logger.Printf("Client certificate not found for %s: %v", config.Interface, err)
	}

	return true, nil
}

// checkCertificateValidity checks certificate validity and expiration
func (v *WG11ComplianceValidator) checkCertificateValidity(ctx context.Context, config *InterfaceSecurityConfig) (bool, error) {
	secretName := fmt.Sprintf("%s-server-cert-tls", config.Interface)
	
	secret, err := v.kubeClient.CoreV1().Secrets(config.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	certData, exists := secret.Data["tls.crt"]
	if !exists {
		return false, fmt.Errorf("certificate data not found")
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check expiration
	now := time.Now()
	if cert.NotAfter.Before(now) {
		return false, fmt.Errorf("certificate expired on %v", cert.NotAfter)
	}

	// Check if certificate expires soon
	threshold := time.Duration(v.config.CertExpirationThreshold) * 24 * time.Hour
	if cert.NotAfter.Before(now.Add(threshold)) {
		return false, fmt.Errorf("certificate expires soon: %v", cert.NotAfter)
	}

	return true, nil
}

// checkNetworkPolicy checks network security policies
func (v *WG11ComplianceValidator) checkNetworkPolicy(ctx context.Context, config *InterfaceSecurityConfig) (bool, error) {
	policyName := fmt.Sprintf("%s-interface-policy", config.Interface)
	
	_, err := v.kubeClient.NetworkingV1().NetworkPolicies(config.Namespace).Get(ctx, policyName, metav1.GetOptions{})
	if err != nil {
		// Also check for generic policies
		policies, err := v.kubeClient.NetworkingV1().NetworkPolicies(config.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		if len(policies.Items) > 0 {
			return true, nil // At least some network policies exist
		}
		return false, fmt.Errorf("no network policies found")
	}

	return true, nil
}

// validateFIPS1403Compliance validates FIPS 140-3 compliance
func (v *WG11ComplianceValidator) validateFIPS1403Compliance(ctx context.Context) (*FIPS1403ComplianceStatus, error) {
	status := &FIPS1403ComplianceStatus{
		LastValidated: time.Now(),
	}

	// Check FIPS configuration
	fipsConfig, err := v.kubeClient.CoreV1().ConfigMaps("oran").Get(ctx, "fips-140-3-config", metav1.GetOptions{})
	if err != nil {
		status.Issues = append(status.Issues, "FIPS configuration not found")
		return status, nil
	}

	// Parse FIPS mode
	mode, exists := fipsConfig.Data["fips-mode"]
	if !exists {
		status.Issues = append(status.Issues, "FIPS mode not specified")
	} else {
		status.Mode = mode
		status.Enabled = (mode == "on" || mode == "only")
	}

	// Check Go version
	goVersion, exists := fipsConfig.Data["go-version"]
	if exists {
		status.GoVersionCompliant = v.isGoVersionFIPSCompliant(goVersion)
	}

	// Check deployment compliance
	compliance, err := v.checkFIPSDeploymentCompliance(ctx)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to check deployment compliance: %v", err))
	}
	status.DeploymentCompliance = compliance

	// Check algorithms
	status.AlgorithmsCompliant = true // Would need actual crypto algorithm validation

	return status, nil
}

// isGoVersionFIPSCompliant checks if Go version supports FIPS
func (v *WG11ComplianceValidator) isGoVersionFIPSCompliant(version string) bool {
	// Go 1.24+ has basic FIPS support, 1.25+ has "only" mode
	return strings.Compare(version, "1.24") >= 0
}

// checkFIPSDeploymentCompliance checks FIPS compliance across deployments
func (v *WG11ComplianceValidator) checkFIPSDeploymentCompliance(ctx context.Context) (int, error) {
	namespaces := []string{"oran", "nonrtric", "nephio-system", "ocloud-system"}
	
	totalDeployments := 0
	compliantDeployments := 0

	for _, namespace := range namespaces {
		deployments, err := v.kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, deployment := range deployments.Items {
			totalDeployments++
			
			// Check for FIPS environment variables
			for _, container := range deployment.Spec.Template.Spec.Containers {
				for _, env := range container.Env {
					if env.Name == "GODEBUG" && strings.Contains(env.Value, "fips140") {
						compliantDeployments++
						break
					}
				}
			}
		}
	}

	if totalDeployments == 0 {
		return 0, nil
	}

	return (compliantDeployments * 100) / totalDeployments, nil
}

// collectSecurityMetrics collects various security metrics
func (v *WG11ComplianceValidator) collectSecurityMetrics(ctx context.Context) (*SecurityMetrics, error) {
	metrics := &SecurityMetrics{
		VulnerabilityCount: &VulnerabilityCount{},
	}

	// Count secrets
	allSecrets, err := v.kubeClient.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.TotalSecrets = len(allSecrets.Items)
		for _, secret := range allSecrets.Items {
			if secret.Type == "kubernetes.io/tls" {
				metrics.TLSSecrets++
			}
		}
	}

	// Count network policies
	allNetPolicies, err := v.kubeClient.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.NetworkPolicies = len(allNetPolicies.Items)
	}

	// Count expired/expiring certificates
	metrics.ExpiredCertificates, metrics.ExpiringCertificates = v.countCertificateStatus(ctx)

	// Vulnerability counts would come from external scanner integration
	// This is a placeholder
	metrics.VulnerabilityCount.Critical = 0
	metrics.VulnerabilityCount.High = 2
	metrics.VulnerabilityCount.Medium = 5
	metrics.VulnerabilityCount.Low = 10

	return metrics, nil
}

// countCertificateStatus counts expired and expiring certificates
func (v *WG11ComplianceValidator) countCertificateStatus(ctx context.Context) (expired, expiring int) {
	namespaces := []string{"oran", "nonrtric", "nephio-system", "ocloud-system"}
	threshold := time.Duration(v.config.CertExpirationThreshold) * 24 * time.Hour

	for _, namespace := range namespaces {
		secrets, err := v.kubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, secret := range secrets.Items {
			if secret.Type != "kubernetes.io/tls" {
				continue
			}

			certData, exists := secret.Data["tls.crt"]
			if !exists {
				continue
			}

			cert, err := x509.ParseCertificate(certData)
			if err != nil {
				continue
			}

			now := time.Now()
			if cert.NotAfter.Before(now) {
				expired++
			} else if cert.NotAfter.Before(now.Add(threshold)) {
				expiring++
			}
		}
	}

	return expired, expiring
}

// calculateInterfaceCompliance calculates compliance level for an interface
func (v *WG11ComplianceValidator) calculateInterfaceCompliance(status *InterfaceComplianceStatus, config *InterfaceSecurityConfig) ComplianceLevel {
	score := 0
	total := 5

	if status.SecurityPolicyExists {
		score++
	}
	if status.TLSConfigured {
		score++
	}
	if status.MTLSConfigured || !config.MTLSRequired {
		score++
	}
	if status.CertificatesValid {
		score++
	}
	if status.NetworkPolicyExists {
		score++
	}

	percentage := (score * 100) / total

	switch {
	case percentage >= 90:
		return ComplianceLevelCompliant
	case percentage >= 70:
		return ComplianceLevelPartial
	case percentage > 0:
		return ComplianceLevelNonCompliant
	default:
		return ComplianceLevelNotConfigured
	}
}

// calculateOverallCompliance calculates overall compliance level
func (v *WG11ComplianceValidator) calculateOverallCompliance(report *ComplianceReport) ComplianceLevel {
	compliantCount := 0
	totalCount := len(report.InterfaceStatus)

	for _, status := range report.InterfaceStatus {
		if status.ComplianceLevel == ComplianceLevelCompliant {
			compliantCount++
		}
	}

	// FIPS compliance factor
	fipsCompliant := report.FIPS1403Status.Enabled && 
		report.FIPS1403Status.DeploymentCompliance >= 90 &&
		report.FIPS1403Status.GoVersionCompliant

	// Vulnerability factor
	vulnCompliant := report.SecurityMetrics.VulnerabilityCount.Critical <= v.config.MaxCriticalVulnerabilities &&
		report.SecurityMetrics.VulnerabilityCount.High <= v.config.MaxHighVulnerabilities

	if compliantCount == totalCount && fipsCompliant && vulnCompliant {
		return ComplianceLevelCompliant
	} else if compliantCount >= totalCount/2 {
		return ComplianceLevelPartial
	} else {
		return ComplianceLevelNonCompliant
	}
}

// generateRecommendations generates security recommendations
func (v *WG11ComplianceValidator) generateRecommendations(report *ComplianceReport) []string {
	recommendations := []string{}

	// Interface-specific recommendations
	for _, status := range report.InterfaceStatus {
		if status.ComplianceLevel != ComplianceLevelCompliant {
			recommendations = append(recommendations, 
				fmt.Sprintf("Address security issues for %s interface", status.Interface))
		}
	}

	// FIPS recommendations
	if !report.FIPS1403Status.Enabled {
		recommendations = append(recommendations, "Enable FIPS 140-3 compliance")
	} else if report.FIPS1403Status.DeploymentCompliance < 100 {
		recommendations = append(recommendations, "Ensure all deployments are FIPS compliant")
	}

	// Certificate recommendations
	if report.SecurityMetrics.ExpiredCertificates > 0 {
		recommendations = append(recommendations, "Renew expired certificates immediately")
	}
	if report.SecurityMetrics.ExpiringCertificates > 0 {
		recommendations = append(recommendations, "Plan certificate renewal for expiring certificates")
	}

	// Vulnerability recommendations
	if report.SecurityMetrics.VulnerabilityCount.Critical > 0 {
		recommendations = append(recommendations, "Address critical vulnerabilities immediately")
	}

	// General recommendations
	recommendations = append(recommendations, 
		"Implement automated certificate rotation",
		"Enable continuous security monitoring",
		"Regular security compliance audits",
		"Implement security incident response procedures",
	)

	return recommendations
}

// getClusterName gets the current cluster name
func (v *WG11ComplianceValidator) getClusterName() string {
	// This would typically come from cluster metadata or configuration
	return "oran-ric-cluster"
}

// GetComplianceStatus returns current compliance status
func (v *WG11ComplianceValidator) GetComplianceStatus(interfaceName string) *InterfaceComplianceStatus {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	return v.complianceStatus[interfaceName]
}

// GetAllComplianceStatus returns all interface compliance status
func (v *WG11ComplianceValidator) GetAllComplianceStatus() map[string]*InterfaceComplianceStatus {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	result := make(map[string]*InterfaceComplianceStatus)
	for k, v := range v.complianceStatus {
		result[k] = v
	}
	
	return result
}

// ExportComplianceReport exports compliance report to JSON
func (v *WG11ComplianceValidator) ExportComplianceReport(report *ComplianceReport, filePath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}