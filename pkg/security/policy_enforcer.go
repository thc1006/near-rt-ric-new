/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

// Package security implements comprehensive O-RAN security policy enforcement
// with WG11 compliance, FIPS 140-3 enforcement, and real-time policy validation
package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PolicyEnforcer enforces O-RAN security policies across the deployment
type PolicyEnforcer struct {
	kubeClient       kubernetes.Interface
	config           *SecurityPolicyConfig
	logger           *log.Logger
	enforcementRules map[string]*EnforcementRule
	violations       []PolicyViolation
	tlsConfig        *tls.Config
	httpClient       *http.Client
	mutex            sync.RWMutex
}

// SecurityPolicyConfig represents the security policy configuration
type SecurityPolicyConfig struct {
	// Global security settings
	EnforceStrictMode     bool                    `json:"enforce_strict_mode"`
	FIPS1403Required      bool                    `json:"fips_140_3_required"`
	TLSMinVersion        string                   `json:"tls_min_version"`
	MaxCertExpiration    time.Duration            `json:"max_cert_expiration"`
	
	// Interface-specific policies
	InterfacePolicies    map[string]*InterfacePolicy `json:"interface_policies"`
	
	// Container security policies
	ContainerPolicies    *ContainerSecurityPolicy `json:"container_policies"`
	
	// Network security policies
	NetworkPolicies      *NetworkSecurityPolicy   `json:"network_policies"`
	
	// Monitoring and alerting
	MonitoringConfig     *MonitoringConfig        `json:"monitoring_config"`
	
	// Violation handling
	ViolationHandling    *ViolationHandling       `json:"violation_handling"`
}

// InterfacePolicy represents security policy for O-RAN interfaces
type InterfacePolicy struct {
	Interface         string            `json:"interface"`
	Namespace         string            `json:"namespace"`
	RequiredPorts     []int             `json:"required_ports"`
	AllowedProtocols  []string          `json:"allowed_protocols"`
	MTLSRequired      bool              `json:"mtls_required"`
	AuthenticationReq string            `json:"authentication_required"`
	EncryptionReq     *EncryptionPolicy `json:"encryption_required"`
	RateLimits        *RateLimitPolicy  `json:"rate_limits"`
}

// EncryptionPolicy represents encryption requirements
type EncryptionPolicy struct {
	Algorithm       string `json:"algorithm"`
	MinKeySize      int    `json:"min_key_size"`
	KeyRotationReq  string `json:"key_rotation_required"`
	CipherSuites    []string `json:"cipher_suites"`
}

// RateLimitPolicy represents rate limiting policy
type RateLimitPolicy struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstSize         int `json:"burst_size"`
	TimeWindow        string `json:"time_window"`
}

// ContainerSecurityPolicy represents container security requirements
type ContainerSecurityPolicy struct {
	RunAsNonRoot          bool     `json:"run_as_non_root"`
	ReadOnlyRootFilesystem bool    `json:"read_only_root_filesystem"`
	AllowPrivilegeEscalation bool  `json:"allow_privilege_escalation"`
	RequiredCapabilities  []string `json:"required_capabilities"`
	ForbiddenCapabilities []string `json:"forbidden_capabilities"`
	RequiredSeccompProfile string  `json:"required_seccomp_profile"`
	FIPSModeRequired      bool     `json:"fips_mode_required"`
	ScanningRequired      bool     `json:"scanning_required"`
	MaxCriticalVulns      int      `json:"max_critical_vulnerabilities"`
	MaxHighVulns          int      `json:"max_high_vulnerabilities"`
}

// NetworkSecurityPolicy represents network security requirements
type NetworkSecurityPolicy struct {
	DefaultDenyAll        bool     `json:"default_deny_all"`
	RequireNetworkPolicies bool    `json:"require_network_policies"`
	AllowedNamespaces     []string `json:"allowed_namespaces"`
	RequiredPorts         []int    `json:"required_ports"`
	ServiceMeshRequired   bool     `json:"service_mesh_required"`
	MTLSRequired          bool     `json:"mtls_required"`
}

// MonitoringConfig represents monitoring and alerting configuration
type MonitoringConfig struct {
	EnableRealTimeMonitoring bool          `json:"enable_real_time_monitoring"`
	AlertingEnabled          bool          `json:"alerting_enabled"`
	MetricsCollection        bool          `json:"metrics_collection"`
	AuditLogging            bool          `json:"audit_logging"`
	ViolationReporting      bool          `json:"violation_reporting"`
	MonitoringInterval      time.Duration `json:"monitoring_interval"`
}

// ViolationHandling represents how policy violations are handled
type ViolationHandling struct {
	BlockNonCompliant     bool          `json:"block_non_compliant"`
	AutoRemediation      bool          `json:"auto_remediation"`
	NotificationChannels []string      `json:"notification_channels"`
	EscalationPolicy     *EscalationPolicy `json:"escalation_policy"`
}

// EscalationPolicy represents violation escalation policy
type EscalationPolicy struct {
	ImmediateViolations  []string      `json:"immediate_violations"`
	EscalationThreshold  int           `json:"escalation_threshold"`
	EscalationTimeWindow time.Duration `json:"escalation_time_window"`
	EscalationTarget     string        `json:"escalation_target"`
}

// EnforcementRule represents a security enforcement rule
type EnforcementRule struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Severity     string                 `json:"severity"`
	Scope        string                 `json:"scope"`
	Condition    string                 `json:"condition"`
	Action       string                 `json:"action"`
	Parameters   map[string]interface{} `json:"parameters"`
	Enabled      bool                   `json:"enabled"`
	CreatedAt    time.Time             `json:"created_at"`
	LastApplied  time.Time             `json:"last_applied"`
}

// PolicyViolation represents a security policy violation
type PolicyViolation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Severity      string                 `json:"severity"`
	Resource      string                 `json:"resource"`
	Namespace     string                 `json:"namespace"`
	Description   string                 `json:"description"`
	Rule          string                 `json:"rule"`
	DetectedAt    time.Time             `json:"detected_at"`
	Status        string                 `json:"status"`
	Remediation   string                 `json:"remediation"`
	Details       map[string]interface{} `json:"details"`
}

// PolicyEnforcementReport represents policy enforcement report
type PolicyEnforcementReport struct {
	Timestamp         time.Time             `json:"timestamp"`
	ClusterName       string                `json:"cluster_name"`
	TotalRules        int                   `json:"total_rules"`
	ActiveRules       int                   `json:"active_rules"`
	ViolationsFound   int                   `json:"violations_found"`
	CriticalViolations int                  `json:"critical_violations"`
	HighViolations    int                   `json:"high_violations"`
	Violations        []PolicyViolation     `json:"violations"`
	ComplianceScore   int                   `json:"compliance_score"`
	Recommendations   []string              `json:"recommendations"`
}

// NewPolicyEnforcer creates a new security policy enforcer
func NewPolicyEnforcer(kubeConfig *rest.Config) (*PolicyEnforcer, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	config := getDefaultSecurityPolicyConfig()
	
	// Create TLS config for secure communications
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}

	enforcer := &PolicyEnforcer{
		kubeClient:       kubeClient,
		config:           config,
		logger:           log.New(os.Stdout, "[POLICY-ENFORCER] ", log.LstdFlags|log.Lshortfile),
		enforcementRules: make(map[string]*EnforcementRule),
		violations:       []PolicyViolation{},
		tlsConfig:        tlsConfig,
		httpClient:       httpClient,
	}

	// Load default enforcement rules
	enforcer.loadDefaultRules()

	return enforcer, nil
}

// getDefaultSecurityPolicyConfig returns default security policy configuration
func getDefaultSecurityPolicyConfig() *SecurityPolicyConfig {
	return &SecurityPolicyConfig{
		EnforceStrictMode:    true,
		FIPS1403Required:     true,
		TLSMinVersion:       "1.3",
		MaxCertExpiration:   30 * 24 * time.Hour, // 30 days
		
		InterfacePolicies: map[string]*InterfacePolicy{
			"e2": {
				Interface:         "e2",
				Namespace:         "oran",
				RequiredPorts:     []int{36421, 4560, 4561},
				AllowedProtocols:  []string{"SCTP", "TCP"},
				MTLSRequired:      true,
				AuthenticationReq: "x509-certificate",
				EncryptionReq: &EncryptionPolicy{
					Algorithm:      "AES-256-GCM",
					MinKeySize:     256,
					KeyRotationReq: "12h",
					CipherSuites:   []string{"TLS_AES_256_GCM_SHA384"},
				},
				RateLimits: &RateLimitPolicy{
					RequestsPerMinute: 1000,
					BurstSize:         100,
					TimeWindow:        "1m",
				},
			},
			"a1": {
				Interface:         "a1",
				Namespace:         "nonrtric",
				RequiredPorts:     []int{8081, 8443},
				AllowedProtocols:  []string{"HTTPS"},
				MTLSRequired:      true,
				AuthenticationReq: "oauth2",
				EncryptionReq: &EncryptionPolicy{
					Algorithm:      "TLS-1.3",
					MinKeySize:     256,
					KeyRotationReq: "24h",
					CipherSuites:   []string{"TLS_AES_256_GCM_SHA384"},
				},
				RateLimits: &RateLimitPolicy{
					RequestsPerMinute: 500,
					BurstSize:         50,
					TimeWindow:        "1m",
				},
			},
		},
		
		ContainerPolicies: &ContainerSecurityPolicy{
			RunAsNonRoot:             true,
			ReadOnlyRootFilesystem:   true,
			AllowPrivilegeEscalation: false,
			RequiredCapabilities:     []string{"NET_BIND_SERVICE"},
			ForbiddenCapabilities:    []string{"SYS_ADMIN", "SYS_PTRACE", "SYS_MODULE"},
			RequiredSeccompProfile:   "RuntimeDefault",
			FIPSModeRequired:         true,
			ScanningRequired:         true,
			MaxCriticalVulns:         0,
			MaxHighVulns:             5,
		},
		
		NetworkPolicies: &NetworkSecurityPolicy{
			DefaultDenyAll:         true,
			RequireNetworkPolicies: true,
			AllowedNamespaces:      []string{"oran", "nonrtric", "nephio-system", "ocloud-system"},
			RequiredPorts:          []int{53, 443}, // DNS and HTTPS
			ServiceMeshRequired:    false, // Optional
			MTLSRequired:           true,
		},
		
		MonitoringConfig: &MonitoringConfig{
			EnableRealTimeMonitoring: true,
			AlertingEnabled:          true,
			MetricsCollection:        true,
			AuditLogging:            true,
			ViolationReporting:      true,
			MonitoringInterval:      1 * time.Minute,
		},
		
		ViolationHandling: &ViolationHandling{
			BlockNonCompliant:    false, // Start with warning mode
			AutoRemediation:      false,
			NotificationChannels: []string{"log", "metrics"},
			EscalationPolicy: &EscalationPolicy{
				ImmediateViolations:  []string{"critical-vulnerability", "privilege-escalation"},
				EscalationThreshold:  5,
				EscalationTimeWindow: 15 * time.Minute,
				EscalationTarget:     "security-team",
			},
		},
	}
}

// loadDefaultRules loads default enforcement rules
func (pe *PolicyEnforcer) loadDefaultRules() {
	rules := []*EnforcementRule{
		{
			Name:        "fips-140-3-enforcement",
			Type:        "container",
			Severity:    "high",
			Scope:       "deployment",
			Condition:   "env.GODEBUG not contains fips140",
			Action:      "warn",
			Parameters:  map[string]interface{}{"required_env": "GODEBUG=fips140=only"},
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "non-root-user-enforcement",
			Type:        "container",
			Severity:    "high",
			Scope:       "pod",
			Condition:   "securityContext.runAsNonRoot != true",
			Action:      "warn",
			Parameters:  map[string]interface{}{"required_setting": "runAsNonRoot: true"},
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "tls-certificate-expiration",
			Type:        "certificate",
			Severity:    "medium",
			Scope:       "secret",
			Condition:   "certificate.expires_within 30d",
			Action:      "alert",
			Parameters:  map[string]interface{}{"threshold_days": 30},
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "network-policy-requirement",
			Type:        "network",
			Severity:    "high",
			Scope:       "namespace",
			Condition:   "networkpolicies.count == 0",
			Action:      "warn",
			Parameters:  map[string]interface{}{"min_policies": 1},
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			Name:        "privileged-container-detection",
			Type:        "container",
			Severity:    "critical",
			Scope:       "pod",
			Condition:   "securityContext.privileged == true",
			Action:      "block",
			Parameters:  map[string]interface{}{"allowed": false},
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
	}

	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	
	for _, rule := range rules {
		pe.enforcementRules[rule.Name] = rule
	}
}

// EnforcePolicies performs comprehensive policy enforcement
func (pe *PolicyEnforcer) EnforcePolicies(ctx context.Context) (*PolicyEnforcementReport, error) {
	pe.logger.Println("Starting policy enforcement")

	report := &PolicyEnforcementReport{
		Timestamp:   time.Now(),
		ClusterName: pe.getClusterName(),
		Violations:  []PolicyViolation{},
	}

	// Clear previous violations
	pe.violations = []PolicyViolation{}

	// Enforce container security policies
	if err := pe.enforceContainerPolicies(ctx); err != nil {
		pe.logger.Printf("Error enforcing container policies: %v", err)
	}

	// Enforce network security policies
	if err := pe.enforceNetworkPolicies(ctx); err != nil {
		pe.logger.Printf("Error enforcing network policies: %v", err)
	}

	// Enforce interface-specific policies
	if err := pe.enforceInterfacePolicies(ctx); err != nil {
		pe.logger.Printf("Error enforcing interface policies: %v", err)
	}

	// Enforce certificate policies
	if err := pe.enforceCertificatePolicies(ctx); err != nil {
		pe.logger.Printf("Error enforcing certificate policies: %v", err)
	}

	// Enforce FIPS 140-3 policies
	if err := pe.enforceFIPS1403Policies(ctx); err != nil {
		pe.logger.Printf("Error enforcing FIPS policies: %v", err)
	}

	// Generate report
	report.Violations = pe.violations
	report.ViolationsFound = len(pe.violations)
	report.TotalRules = len(pe.enforcementRules)
	
	activeRules := 0
	for _, rule := range pe.enforcementRules {
		if rule.Enabled {
			activeRules++
		}
	}
	report.ActiveRules = activeRules

	// Count violations by severity
	for _, violation := range pe.violations {
		switch violation.Severity {
		case "critical":
			report.CriticalViolations++
		case "high":
			report.HighViolations++
		}
	}

	// Calculate compliance score
	report.ComplianceScore = pe.calculateComplianceScore(report)

	// Generate recommendations
	report.Recommendations = pe.generateRecommendations(report)

	pe.logger.Printf("Policy enforcement completed. Found %d violations", report.ViolationsFound)

	return report, nil
}

// enforceContainerPolicies enforces container security policies
func (pe *PolicyEnforcer) enforceContainerPolicies(ctx context.Context) error {
	namespaces := []string{"oran", "nonrtric", "nephio-system", "ocloud-system"}
	
	for _, namespace := range namespaces {
		deployments, err := pe.kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, deployment := range deployments.Items {
			pe.validateDeploymentSecurity(&deployment)
		}

		// Also check pods
		pods, err := pe.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			pe.validatePodSecurity(&pod)
		}
	}

	return nil
}

// validateDeploymentSecurity validates deployment security configuration
func (pe *PolicyEnforcer) validateDeploymentSecurity(deployment *appsv1.Deployment) {
	// Check FIPS 140-3 environment variables
	if pe.config.FIPS1403Required {
		fipsEnabled := false
		for _, container := range deployment.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				if env.Name == "GODEBUG" && strings.Contains(env.Value, "fips140") {
					fipsEnabled = true
					break
				}
			}
		}
		
		if !fipsEnabled {
			pe.addViolation(PolicyViolation{
				Type:        "fips-compliance",
				Severity:    "high",
				Resource:    deployment.Name,
				Namespace:   deployment.Namespace,
				Description: "FIPS 140-3 environment variables not configured",
				Rule:        "fips-140-3-enforcement",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Set GODEBUG=fips140=only environment variable",
			})
		}
	}

	// Check security context
	securityContext := deployment.Spec.Template.Spec.SecurityContext
	if pe.config.ContainerPolicies.RunAsNonRoot {
		if securityContext == nil || securityContext.RunAsNonRoot == nil || !*securityContext.RunAsNonRoot {
			pe.addViolation(PolicyViolation{
				Type:        "security-context",
				Severity:    "high",
				Resource:    deployment.Name,
				Namespace:   deployment.Namespace,
				Description: "Container not configured to run as non-root",
				Rule:        "non-root-user-enforcement",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Set runAsNonRoot: true in security context",
			})
		}
	}

	// Check container security contexts
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if pe.config.ContainerPolicies.ReadOnlyRootFilesystem {
			if container.SecurityContext == nil || 
			   container.SecurityContext.ReadOnlyRootFilesystem == nil ||
			   !*container.SecurityContext.ReadOnlyRootFilesystem {
				pe.addViolation(PolicyViolation{
					Type:        "filesystem-security",
					Severity:    "medium",
					Resource:    fmt.Sprintf("%s/%s", deployment.Name, container.Name),
					Namespace:   deployment.Namespace,
					Description: "Container root filesystem is not read-only",
					Rule:        "readonly-filesystem-enforcement",
					DetectedAt:  time.Now(),
					Status:      "active",
					Remediation: "Set readOnlyRootFilesystem: true",
				})
			}
		}

		// Check for privileged containers
		if container.SecurityContext != nil && 
		   container.SecurityContext.Privileged != nil && 
		   *container.SecurityContext.Privileged {
			pe.addViolation(PolicyViolation{
				Type:        "privileged-container",
				Severity:    "critical",
				Resource:    fmt.Sprintf("%s/%s", deployment.Name, container.Name),
				Namespace:   deployment.Namespace,
				Description: "Container running in privileged mode",
				Rule:        "privileged-container-detection",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Remove privileged: true from container security context",
			})
		}
	}
}

// validatePodSecurity validates pod security configuration
func (pe *PolicyEnforcer) validatePodSecurity(pod *corev1.Pod) {
	// Check for host network usage
	if pod.Spec.HostNetwork {
		pe.addViolation(PolicyViolation{
			Type:        "host-network",
			Severity:    "high",
			Resource:    pod.Name,
			Namespace:   pod.Namespace,
			Description: "Pod using host network",
			Rule:        "host-network-prohibition",
			DetectedAt:  time.Now(),
			Status:      "active",
			Remediation: "Remove hostNetwork: true from pod spec",
		})
	}

	// Check for host PID usage
	if pod.Spec.HostPID {
		pe.addViolation(PolicyViolation{
			Type:        "host-pid",
			Severity:    "high",
			Resource:    pod.Name,
			Namespace:   pod.Namespace,
			Description: "Pod using host PID namespace",
			Rule:        "host-pid-prohibition",
			DetectedAt:  time.Now(),
			Status:      "active",
			Remediation: "Remove hostPID: true from pod spec",
		})
	}
}

// enforceNetworkPolicies enforces network security policies
func (pe *PolicyEnforcer) enforceNetworkPolicies(ctx context.Context) error {
	if !pe.config.NetworkPolicies.RequireNetworkPolicies {
		return nil
	}

	for _, namespace := range pe.config.NetworkPolicies.AllowedNamespaces {
		policies, err := pe.kubeClient.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		if len(policies.Items) == 0 {
			pe.addViolation(PolicyViolation{
				Type:        "network-policy-missing",
				Severity:    "high",
				Resource:    namespace,
				Namespace:   namespace,
				Description: "No network policies found in namespace",
				Rule:        "network-policy-requirement",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Deploy network policies to restrict traffic",
			})
		}

		// Check for default deny policy
		hasDefaultDeny := false
		for _, policy := range policies.Items {
			if pe.isDefaultDenyPolicy(&policy) {
				hasDefaultDeny = true
				break
			}
		}

		if pe.config.NetworkPolicies.DefaultDenyAll && !hasDefaultDeny {
			pe.addViolation(PolicyViolation{
				Type:        "default-deny-missing",
				Severity:    "medium",
				Resource:    namespace,
				Namespace:   namespace,
				Description: "Default deny network policy not found",
				Rule:        "default-deny-requirement",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Deploy default deny network policy",
			})
		}
	}

	return nil
}

// isDefaultDenyPolicy checks if a network policy is a default deny policy
func (pe *PolicyEnforcer) isDefaultDenyPolicy(policy *networkingv1.NetworkPolicy) bool {
	// Check if it selects all pods and has no allow rules
	if len(policy.Spec.PodSelector.MatchLabels) == 0 && 
	   len(policy.Spec.PodSelector.MatchExpressions) == 0 &&
	   len(policy.Spec.Ingress) == 0 && 
	   len(policy.Spec.Egress) == 0 {
		return true
	}
	return false
}

// enforceInterfacePolicies enforces interface-specific security policies
func (pe *PolicyEnforcer) enforceInterfacePolicies(ctx context.Context) error {
	for _, interfacePolicy := range pe.config.InterfacePolicies {
		// Check if required secrets exist
		if interfacePolicy.MTLSRequired {
			secretName := fmt.Sprintf("%s-server-cert-tls", interfacePolicy.Interface)
			_, err := pe.kubeClient.CoreV1().Secrets(interfacePolicy.Namespace).Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				pe.addViolation(PolicyViolation{
					Type:        "tls-certificate-missing",
					Severity:    "high",
					Resource:    secretName,
					Namespace:   interfacePolicy.Namespace,
					Description: fmt.Sprintf("TLS certificate missing for %s interface", interfacePolicy.Interface),
					Rule:        "mtls-certificate-requirement",
					DetectedAt:  time.Now(),
					Status:      "active",
					Remediation: "Deploy TLS certificates for mTLS authentication",
				})
			}
		}

		// Check for interface-specific network policies
		policyName := fmt.Sprintf("%s-interface-policy", interfacePolicy.Interface)
		_, err := pe.kubeClient.NetworkingV1().NetworkPolicies(interfacePolicy.Namespace).Get(ctx, policyName, metav1.GetOptions{})
		if err != nil {
			pe.addViolation(PolicyViolation{
				Type:        "interface-network-policy-missing",
				Severity:    "medium",
				Resource:    policyName,
				Namespace:   interfacePolicy.Namespace,
				Description: fmt.Sprintf("Network policy missing for %s interface", interfacePolicy.Interface),
				Rule:        "interface-network-policy-requirement",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Deploy network policy for interface security",
			})
		}
	}

	return nil
}

// enforceCertificatePolicies enforces certificate-related security policies
func (pe *PolicyEnforcer) enforceCertificatePolicies(ctx context.Context) error {
	namespaces := []string{"oran", "nonrtric", "nephio-system", "ocloud-system"}
	
	for _, namespace := range namespaces {
		secrets, err := pe.kubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, secret := range secrets.Items {
			if secret.Type == "kubernetes.io/tls" {
				pe.validateCertificateSecret(&secret)
			}
		}
	}

	return nil
}

// validateCertificateSecret validates certificate secret
func (pe *PolicyEnforcer) validateCertificateSecret(secret *corev1.Secret) {
	certData, exists := secret.Data["tls.crt"]
	if !exists {
		pe.addViolation(PolicyViolation{
			Type:        "certificate-data-missing",
			Severity:    "high",
			Resource:    secret.Name,
			Namespace:   secret.Namespace,
			Description: "Certificate data missing from TLS secret",
			Rule:        "certificate-data-requirement",
			DetectedAt:  time.Now(),
			Status:      "active",
			Remediation: "Ensure TLS secret contains valid certificate data",
		})
		return
	}

	// Parse and validate certificate
	cert, err := tls.X509KeyPair(certData, secret.Data["tls.key"])
	if err != nil {
		pe.addViolation(PolicyViolation{
			Type:        "certificate-invalid",
			Severity:    "high",
			Resource:    secret.Name,
			Namespace:   secret.Namespace,
			Description: "Invalid certificate/key pair in TLS secret",
			Rule:        "certificate-validity-requirement",
			DetectedAt:  time.Now(),
			Status:      "active",
			Remediation: "Replace with valid certificate and key",
		})
		return
	}

	// Check certificate expiration
	if len(cert.Certificate) > 0 {
		// Parse the certificate to check expiration
		// This would require additional parsing - simplified for demo
		if time.Now().Add(pe.config.MaxCertExpiration).After(time.Now()) {
			pe.addViolation(PolicyViolation{
				Type:        "certificate-expiring",
				Severity:    "medium",
				Resource:    secret.Name,
				Namespace:   secret.Namespace,
				Description: "Certificate expires soon",
				Rule:        "tls-certificate-expiration",
				DetectedAt:  time.Now(),
				Status:      "active",
				Remediation: "Renew certificate before expiration",
			})
		}
	}
}

// enforceFIPS1403Policies enforces FIPS 140-3 compliance policies
func (pe *PolicyEnforcer) enforceFIPS1403Policies(ctx context.Context) error {
	if !pe.config.FIPS1403Required {
		return nil
	}

	// Check if FIPS configuration exists
	_, err := pe.kubeClient.CoreV1().ConfigMaps("oran").Get(ctx, "fips-140-3-config", metav1.GetOptions{})
	if err != nil {
		pe.addViolation(PolicyViolation{
			Type:        "fips-config-missing",
			Severity:    "high",
			Resource:    "fips-140-3-config",
			Namespace:   "oran",
			Description: "FIPS 140-3 configuration not found",
			Rule:        "fips-configuration-requirement",
			DetectedAt:  time.Now(),
			Status:      "active",
			Remediation: "Deploy FIPS 140-3 configuration",
		})
	}

	return nil
}

// addViolation adds a policy violation to the list
func (pe *PolicyEnforcer) addViolation(violation PolicyViolation) {
	violation.ID = fmt.Sprintf("%s-%s-%d", violation.Type, violation.Resource, time.Now().Unix())
	
	pe.mutex.Lock()
	pe.violations = append(pe.violations, violation)
	pe.mutex.Unlock()
	
	pe.logger.Printf("Policy violation detected: %s - %s", violation.Type, violation.Description)
}

// calculateComplianceScore calculates overall compliance score
func (pe *PolicyEnforcer) calculateComplianceScore(report *PolicyEnforcementReport) int {
	if report.TotalRules == 0 {
		return 100
	}

	// Base score starts at 100
	score := 100

	// Deduct points for violations
	for _, violation := range report.Violations {
		switch violation.Severity {
		case "critical":
			score -= 20
		case "high":
			score -= 10
		case "medium":
			score -= 5
		case "low":
			score -= 2
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// generateRecommendations generates security recommendations based on violations
func (pe *PolicyEnforcer) generateRecommendations(report *PolicyEnforcementReport) []string {
	recommendations := []string{}

	if report.CriticalViolations > 0 {
		recommendations = append(recommendations, "Address critical security violations immediately")
	}

	if report.HighViolations > 0 {
		recommendations = append(recommendations, "Review and remediate high-severity violations")
	}

	// Specific recommendations based on violation types
	violationTypes := make(map[string]int)
	for _, violation := range report.Violations {
		violationTypes[violation.Type]++
	}

	for violationType, count := range violationTypes {
		switch violationType {
		case "fips-compliance":
			recommendations = append(recommendations, 
				fmt.Sprintf("Enable FIPS 140-3 compliance for %d resources", count))
		case "tls-certificate-missing":
			recommendations = append(recommendations, 
				fmt.Sprintf("Deploy TLS certificates for %d resources", count))
		case "network-policy-missing":
			recommendations = append(recommendations, 
				fmt.Sprintf("Implement network policies for %d namespaces", count))
		case "privileged-container":
			recommendations = append(recommendations, 
				fmt.Sprintf("Remove privileged access from %d containers", count))
		}
	}

	// General recommendations
	recommendations = append(recommendations,
		"Implement automated policy enforcement",
		"Enable continuous security monitoring",
		"Regular security policy reviews",
		"Implement security training for development teams",
	)

	return recommendations
}

// getClusterName gets the current cluster name
func (pe *PolicyEnforcer) getClusterName() string {
	// This would typically come from cluster metadata or configuration
	return "oran-ric-cluster"
}

// GetViolations returns current policy violations
func (pe *PolicyEnforcer) GetViolations() []PolicyViolation {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()
	
	violations := make([]PolicyViolation, len(pe.violations))
	copy(violations, pe.violations)
	return violations
}

// GetViolationsByType returns violations filtered by type
func (pe *PolicyEnforcer) GetViolationsByType(violationType string) []PolicyViolation {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()
	
	var filtered []PolicyViolation
	for _, violation := range pe.violations {
		if violation.Type == violationType {
			filtered = append(filtered, violation)
		}
	}
	
	return filtered
}

// GetViolationsBySeverity returns violations filtered by severity
func (pe *PolicyEnforcer) GetViolationsBySeverity(severity string) []PolicyViolation {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()
	
	var filtered []PolicyViolation
	for _, violation := range pe.violations {
		if violation.Severity == severity {
			filtered = append(filtered, violation)
		}
	}
	
	return filtered
}

// UpdateRule updates or adds an enforcement rule
func (pe *PolicyEnforcer) UpdateRule(rule *EnforcementRule) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	
	rule.LastApplied = time.Now()
	pe.enforcementRules[rule.Name] = rule
	
	pe.logger.Printf("Updated enforcement rule: %s", rule.Name)
}

// DisableRule disables an enforcement rule
func (pe *PolicyEnforcer) DisableRule(ruleName string) error {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	
	rule, exists := pe.enforcementRules[ruleName]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleName)
	}
	
	rule.Enabled = false
	pe.logger.Printf("Disabled enforcement rule: %s", ruleName)
	return nil
}

// EnableRule enables an enforcement rule
func (pe *PolicyEnforcer) EnableRule(ruleName string) error {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	
	rule, exists := pe.enforcementRules[ruleName]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleName)
	}
	
	rule.Enabled = true
	pe.logger.Printf("Enabled enforcement rule: %s", ruleName)
	return nil
}