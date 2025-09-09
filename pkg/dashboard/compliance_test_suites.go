package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ComplianceTestSuiteManager manages compliance test suites
type ComplianceTestSuiteManager struct {
	runner *ComplianceTestRunner
	suites map[string]*ComplianceTestSuite
}

// NewComplianceTestSuiteManager creates a new test suite manager
func NewComplianceTestSuiteManager(runner *ComplianceTestRunner) *ComplianceTestSuiteManager {
	manager := &ComplianceTestSuiteManager{
		runner: runner,
		suites: make(map[string]*ComplianceTestSuite),
	}
	
	// Initialize all test suites
	manager.initializeTestSuites()
	
	return manager
}

// initializeTestSuites initializes all compliance test suites
func (m *ComplianceTestSuiteManager) initializeTestSuites() {
	m.suites["e2ap"] = m.createE2APTestSuite()
	m.suites["a1"] = m.createA1TestSuite()
	m.suites["o1"] = m.createO1TestSuite()
	m.suites["security"] = m.createSecurityTestSuite()
	m.suites["interoperability"] = m.createInteroperabilityTestSuite()
}

// createE2APTestSuite creates the E2AP compliance test suite
func (m *ComplianceTestSuiteManager) createE2APTestSuite() *ComplianceTestSuite {
	return &ComplianceTestSuite{
		Name:    "O-RAN E2AP Compliance Test Suite",
		Version: "1.0.0",
		Tests: []ComplianceTest{
			{
				ID:          "e2ap-001",
				Name:        "E2 Setup Procedure Compliance",
				Description: "Validates E2 Setup procedure according to O-RAN.WG3.E2AP-R003",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 8.2.1",
				Severity:    SeverityCritical,
				Tags:        []string{"e2ap", "setup", "procedure"},
			},
			{
				ID:          "e2ap-002",
				Name:        "ASN.1 PER Encoding Compliance",
				Description: "Validates ASN.1 PER encoding for all E2AP messages",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 9.1",
				Severity:    SeverityCritical,
				Tags:        []string{"e2ap", "asn1", "encoding"},
			},
			{
				ID:          "e2ap-003",
				Name:        "SCTP Transport Compliance",
				Description: "Validates SCTP multi-stream transport implementation",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 7.1",
				Severity:    SeverityHigh,
				Tags:        []string{"e2ap", "sctp", "transport"},
			},
			{
				ID:          "e2ap-004",
				Name:        "Service Model Support",
				Description: "Validates support for required service models (KPM, RC, NI)",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 8.4",
				Severity:    SeverityHigh,
				Tags:        []string{"e2ap", "service-model", "kmp", "rc", "ni"},
			},
			{
				ID:          "e2ap-005",
				Name:        "Subscription Procedures",
				Description: "Validates E2 subscription request/response/delete procedures",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 8.2.3",
				Severity:    SeverityHigh,
				Tags:        []string{"e2ap", "subscription", "procedure"},
			},
			{
				ID:          "e2ap-006",
				Name:        "RIC Control Procedures",
				Description: "Validates RIC control request/acknowledge procedures",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 8.2.4",
				Severity:    SeverityHigh,
				Tags:        []string{"e2ap", "control", "procedure"},
			},
			{
				ID:          "e2ap-007",
				Name:        "Error Handling Compliance",
				Description: "Validates proper error handling and cause codes",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 8.3",
				Severity:    SeverityMedium,
				Tags:        []string{"e2ap", "error", "handling"},
			},
			{
				ID:          "e2ap-008",
				Name:        "Message Validation",
				Description: "Validates E2AP message format and content validation",
				Category:    "e2ap",
				Requirement: "O-RAN.WG3.E2AP-R003 Section 9.2",
				Severity:    SeverityMedium,
				Tags:        []string{"e2ap", "validation", "message"},
			},
		},
		Metadata: map[string]string{
			"standard":    "O-RAN.WG3.E2AP-R003",
			"version":     "v03.00",
			"category":    "interface",
			"criticality": "high",
		},
	}
}

// createA1TestSuite creates the A1 compliance test suite
func (m *ComplianceTestSuiteManager) createA1TestSuite() *ComplianceTestSuite {
	return &ComplianceTestSuite{
		Name:    "O-RAN A1 Interface Compliance Test Suite",
		Version: "1.0.0",
		Tests: []ComplianceTest{
			{
				ID:          "a1-001",
				Name:        "Health Check Endpoint",
				Description: "Validates A1 health check endpoint compliance",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 4.1",
				Severity:    SeverityHigh,
				Tags:        []string{"a1", "health", "endpoint"},
			},
			{
				ID:          "a1-002",
				Name:        "Policy Type Management",
				Description: "Validates policy type CRUD operations",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 4.2",
				Severity:    SeverityCritical,
				Tags:        []string{"a1", "policy-type", "crud"},
			},
			{
				ID:          "a1-003",
				Name:        "Policy Instance Management",
				Description: "Validates policy instance lifecycle management",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 4.3",
				Severity:    SeverityCritical,
				Tags:        []string{"a1", "policy-instance", "lifecycle"},
			},
			{
				ID:          "a1-004",
				Name:        "JWT Authentication",
				Description: "Validates JWT-based authentication implementation",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 5.1",
				Severity:    SeverityHigh,
				Tags:        []string{"a1", "jwt", "authentication"},
			},
			{
				ID:          "a1-005",
				Name:        "RBAC Authorization",
				Description: "Validates role-based access control implementation",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 5.2",
				Severity:    SeverityHigh,
				Tags:        []string{"a1", "rbac", "authorization"},
			},
			{
				ID:          "a1-006",
				Name:        "JSON Schema Validation",
				Description: "Validates JSON schema validation for policy types and instances",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 4.4",
				Severity:    SeverityMedium,
				Tags:        []string{"a1", "json", "schema", "validation"},
			},
			{
				ID:          "a1-007",
				Name:        "Error Response Format",
				Description: "Validates proper error response format and status codes",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 6.1",
				Severity:    SeverityMedium,
				Tags:        []string{"a1", "error", "response"},
			},
			{
				ID:          "a1-008",
				Name:        "API Versioning",
				Description: "Validates API versioning compliance",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 3.1",
				Severity:    SeverityLow,
				Tags:        []string{"a1", "versioning", "api"},
			},
			{
				ID:          "a1-009",
				Name:        "Content Negotiation",
				Description: "Validates HTTP content negotiation support",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 3.2",
				Severity:    SeverityLow,
				Tags:        []string{"a1", "content", "negotiation"},
			},
			{
				ID:          "a1-010",
				Name:        "Rate Limiting",
				Description: "Validates rate limiting implementation",
				Category:    "a1",
				Requirement: "O-RAN.WG2.A1 Section 5.3",
				Severity:    SeverityLow,
				Tags:        []string{"a1", "rate", "limiting"},
			},
		},
		Metadata: map[string]string{
			"standard":    "O-RAN.WG2.A1",
			"version":     "v02.01",
			"category":    "interface",
			"criticality": "high",
		},
	}
}

// createO1TestSuite creates the O1 compliance test suite
func (m *ComplianceTestSuiteManager) createO1TestSuite() *ComplianceTestSuite {
	return &ComplianceTestSuite{
		Name:    "O-RAN O1 Interface Compliance Test Suite",
		Version: "1.0.0",
		Tests: []ComplianceTest{
			{
				ID:          "o1-001",
				Name:        "NETCONF Connection Establishment",
				Description: "Validates NETCONF connection establishment per RFC 6241",
				Category:    "o1",
				Requirement: "RFC 6241 Section 4",
				Severity:    SeverityCritical,
				Tags:        []string{"o1", "netconf", "connection"},
			},
			{
				ID:          "o1-002",
				Name:        "Capability Exchange",
				Description: "Validates NETCONF capability exchange",
				Category:    "o1",
				Requirement: "RFC 6241 Section 8.1",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "netconf", "capabilities"},
			},
			{
				ID:          "o1-003",
				Name:        "YANG Model Support",
				Description: "Validates O-RAN YANG model support",
				Category:    "o1",
				Requirement: "O-RAN.WG4 YANG Models",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "yang", "models"},
			},
			{
				ID:          "o1-004",
				Name:        "Configuration Operations",
				Description: "Validates NETCONF configuration operations",
				Category:    "o1",
				Requirement: "RFC 6241 Section 7",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "netconf", "configuration"},
			},
			{
				ID:          "o1-005",
				Name:        "Transaction Support",
				Description: "Validates NETCONF transaction support",
				Category:    "o1",
				Requirement: "RFC 6241 Section 8.3",
				Severity:    SeverityMedium,
				Tags:        []string{"o1", "netconf", "transactions"},
			},
			{
				ID:          "o1-006",
				Name:        "Validation Capabilities",
				Description: "Validates NETCONF validation capabilities",
				Category:    "o1",
				Requirement: "RFC 6241 Section 8.6",
				Severity:    SeverityMedium,
				Tags:        []string{"o1", "netconf", "validation"},
			},
			{
				ID:          "o1-007",
				Name:        "Fault Management",
				Description: "Validates FCAPS fault management functionality",
				Category:    "o1",
				Requirement: "O-RAN.WG4 Fault Management",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "fcaps", "fault"},
			},
			{
				ID:          "o1-008",
				Name:        "Performance Management",
				Description: "Validates FCAPS performance management functionality",
				Category:    "o1",
				Requirement: "O-RAN.WG4 Performance Management",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "fcaps", "performance"},
			},
			{
				ID:          "o1-009",
				Name:        "Security Management",
				Description: "Validates FCAPS security management functionality",
				Category:    "o1",
				Requirement: "O-RAN.WG4 Security Management",
				Severity:    SeverityHigh,
				Tags:        []string{"o1", "fcaps", "security"},
			},
			{
				ID:          "o1-010",
				Name:        "Backup and Restore",
				Description: "Validates configuration backup and restore capabilities",
				Category:    "o1",
				Requirement: "O-RAN.WG4 Configuration Management",
				Severity:    SeverityMedium,
				Tags:        []string{"o1", "backup", "restore"},
			},
		},
		Metadata: map[string]string{
			"standard":    "RFC 6241 + O-RAN.WG4",
			"version":     "1.1 + v01.00",
			"category":    "interface",
			"criticality": "high",
		},
	}
}

// createSecurityTestSuite creates the security compliance test suite
func (m *ComplianceTestSuiteManager) createSecurityTestSuite() *ComplianceTestSuite {
	return &ComplianceTestSuite{
		Name:    "O-RAN Security Compliance Test Suite",
		Version: "1.0.0",
		Tests: []ComplianceTest{
			{
				ID:          "sec-001",
				Name:        "TLS Implementation Compliance",
				Description: "Validates TLS 1.3 implementation and configuration",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 4.1",
				Severity:    SeverityCritical,
				Tags:        []string{"security", "tls", "encryption"},
			},
			{
				ID:          "sec-002",
				Name:        "Cipher Suite Compliance",
				Description: "Validates approved cipher suite usage",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 4.2",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "cipher", "encryption"},
			},
			{
				ID:          "sec-003",
				Name:        "Certificate Validation",
				Description: "Validates X.509 certificate handling and validation",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 4.3",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "certificate", "x509"},
			},
			{
				ID:          "sec-004",
				Name:        "Mutual TLS Authentication",
				Description: "Validates mutual TLS authentication implementation",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 4.4",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "mtls", "authentication"},
			},
			{
				ID:          "sec-005",
				Name:        "JWT Token Security",
				Description: "Validates JWT token security implementation",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 5.1",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "jwt", "token"},
			},
			{
				ID:          "sec-006",
				Name:        "RBAC Implementation",
				Description: "Validates role-based access control implementation",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 5.2",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "rbac", "authorization"},
			},
			{
				ID:          "sec-007",
				Name:        "Encryption Compliance",
				Description: "Validates data encryption at rest and in transit",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 6.1",
				Severity:    SeverityHigh,
				Tags:        []string{"security", "encryption", "data"},
			},
			{
				ID:          "sec-008",
				Name:        "Security Auditing",
				Description: "Validates security event logging and auditing",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 7.1",
				Severity:    SeverityMedium,
				Tags:        []string{"security", "audit", "logging"},
			},
			{
				ID:          "sec-009",
				Name:        "Vulnerability Protection",
				Description: "Validates protection against common vulnerabilities",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 8.1",
				Severity:    SeverityMedium,
				Tags:        []string{"security", "vulnerability", "protection"},
			},
			{
				ID:          "sec-010",
				Name:        "Security Headers",
				Description: "Validates HTTP security headers implementation",
				Category:    "security",
				Requirement: "O-RAN.WG11.Security Section 9.1",
				Severity:    SeverityLow,
				Tags:        []string{"security", "headers", "http"},
			},
		},
		Metadata: map[string]string{
			"standard":    "O-RAN.WG11.Security",
			"version":     "v01.00",
			"category":    "security",
			"criticality": "critical",
		},
	}
}

// createInteroperabilityTestSuite creates the interoperability compliance test suite
func (m *ComplianceTestSuiteManager) createInteroperabilityTestSuite() *ComplianceTestSuite {
	return &ComplianceTestSuite{
		Name:    "O-RAN Interoperability Compliance Test Suite",
		Version: "1.0.0",
		Tests: []ComplianceTest{
			{
				ID:          "interop-001",
				Name:        "Third-Party E2 Node Integration",
				Description: "Validates integration with third-party E2 nodes",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityHigh,
				Tags:        []string{"interop", "e2", "third-party"},
			},
			{
				ID:          "interop-002",
				Name:        "SMO Integration",
				Description: "Validates integration with third-party SMO systems",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityHigh,
				Tags:        []string{"interop", "smo", "third-party"},
			},
			{
				ID:          "interop-003",
				Name:        "Multi-Vendor Compatibility",
				Description: "Validates compatibility across multiple vendors",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityHigh,
				Tags:        []string{"interop", "multi-vendor", "compatibility"},
			},
			{
				ID:          "interop-004",
				Name:        "Protocol Version Compatibility",
				Description: "Validates compatibility across protocol versions",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityMedium,
				Tags:        []string{"interop", "protocol", "version"},
			},
			{
				ID:          "interop-005",
				Name:        "Data Format Compatibility",
				Description: "Validates compatibility of data formats",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityMedium,
				Tags:        []string{"interop", "data", "format"},
			},
			{
				ID:          "interop-006",
				Name:        "Service Model Interoperability",
				Description: "Validates service model interoperability",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityHigh,
				Tags:        []string{"interop", "service-model", "compatibility"},
			},
			{
				ID:          "interop-007",
				Name:        "Cross-Vendor Policy Exchange",
				Description: "Validates policy exchange between different vendors",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityMedium,
				Tags:        []string{"interop", "policy", "cross-vendor"},
			},
			{
				ID:          "interop-008",
				Name:        "Management Interface Compatibility",
				Description: "Validates management interface compatibility",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityMedium,
				Tags:        []string{"interop", "management", "compatibility"},
			},
			{
				ID:          "interop-009",
				Name:        "Scalability with Third-Party",
				Description: "Validates scalability with third-party components",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityLow,
				Tags:        []string{"interop", "scalability", "third-party"},
			},
			{
				ID:          "interop-010",
				Name:        "Failover with Third-Party",
				Description: "Validates failover scenarios with third-party components",
				Category:    "interoperability",
				Requirement: "O-RAN Interoperability Requirements",
				Severity:    SeverityLow,
				Tags:        []string{"interop", "failover", "third-party"},
			},
		},
		Metadata: map[string]string{
			"standard":    "O-RAN Interoperability",
			"version":     "v01.00",
			"category":    "interoperability",
			"criticality": "medium",
		},
	}
}

// GetTestSuite returns a test suite by name
func (m *ComplianceTestSuiteManager) GetTestSuite(name string) (*ComplianceTestSuite, error) {
	suite, exists := m.suites[name]
	if !exists {
		return nil, fmt.Errorf("test suite %s not found", name)
	}
	return suite, nil
}

// GetAllTestSuites returns all available test suites
func (m *ComplianceTestSuiteManager) GetAllTestSuites() map[string]*ComplianceTestSuite {
	return m.suites
}

// RunAllTestSuites runs all compliance test suites
func (m *ComplianceTestSuiteManager) RunAllTestSuites(ctx context.Context) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Timestamp: time.Now(),
		Standards: make(map[string]StandardCompliance),
	}
	
	for name, suite := range m.suites {
		if err := m.runner.RunTestSuite(ctx, suite); err != nil {
			m.runner.logger.Error("Failed to run test suite", "suite", name, "error", err)
			continue
		}
		
		compliance := StandardCompliance{
			Standard:  name,
			Version:   suite.Version,
			TestSuite: *suite,
			Compliant: suite.Summary.Failed == 0,
			Score:     suite.Summary.Coverage,
			Issues:    m.extractIssuesFromSuite(suite),
		}
		
		report.Standards[name] = compliance
	}
	
	report.OverallCompliance = m.calculateOverallCompliance(report.Standards)
	
	return report, nil
}

// RunTestSuiteByName runs a specific test suite by name
func (m *ComplianceTestSuiteManager) RunTestSuiteByName(ctx context.Context, suiteName string) (*StandardCompliance, error) {
	suite, err := m.GetTestSuite(suiteName)
	if err != nil {
		return nil, err
	}
	
	if err := m.runner.RunTestSuite(ctx, suite); err != nil {
		return nil, fmt.Errorf("failed to run test suite %s: %w", suiteName, err)
	}
	
	compliance := &StandardCompliance{
		Standard:  suiteName,
		Version:   suite.Version,
		TestSuite: *suite,
		Compliant: suite.Summary.Failed == 0,
		Score:     suite.Summary.Coverage,
		Issues:    m.extractIssuesFromSuite(suite),
	}
	
	return compliance, nil
}

// RunTestsByTag runs tests filtered by tags
func (m *ComplianceTestSuiteManager) RunTestsByTag(ctx context.Context, tags []string) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Timestamp: time.Now(),
		Standards: make(map[string]StandardCompliance),
	}
	
	for name, suite := range m.suites {
		filteredSuite := m.filterTestsByTags(suite, tags)
		if len(filteredSuite.Tests) == 0 {
			continue
		}
		
		if err := m.runner.RunTestSuite(ctx, filteredSuite); err != nil {
			m.runner.logger.Error("Failed to run filtered test suite", "suite", name, "error", err)
			continue
		}
		
		compliance := StandardCompliance{
			Standard:  name,
			Version:   filteredSuite.Version,
			TestSuite: *filteredSuite,
			Compliant: filteredSuite.Summary.Failed == 0,
			Score:     filteredSuite.Summary.Coverage,
			Issues:    m.extractIssuesFromSuite(filteredSuite),
		}
		
		report.Standards[name] = compliance
	}
	
	report.OverallCompliance = m.calculateOverallCompliance(report.Standards)
	
	return report, nil
}

// RunTestsBySeverity runs tests filtered by severity
func (m *ComplianceTestSuiteManager) RunTestsBySeverity(ctx context.Context, severity TestSeverity) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Timestamp: time.Now(),
		Standards: make(map[string]StandardCompliance),
	}
	
	for name, suite := range m.suites {
		filteredSuite := m.filterTestsBySeverity(suite, severity)
		if len(filteredSuite.Tests) == 0 {
			continue
		}
		
		if err := m.runner.RunTestSuite(ctx, filteredSuite); err != nil {
			m.runner.logger.Error("Failed to run filtered test suite", "suite", name, "error", err)
			continue
		}
		
		compliance := StandardCompliance{
			Standard:  name,
			Version:   filteredSuite.Version,
			TestSuite: *filteredSuite,
			Compliant: filteredSuite.Summary.Failed == 0,
			Score:     filteredSuite.Summary.Coverage,
			Issues:    m.extractIssuesFromSuite(filteredSuite),
		}
		
		report.Standards[name] = compliance
	}
	
	report.OverallCompliance = m.calculateOverallCompliance(report.Standards)
	
	return report, nil
}

// filterTestsByTags filters tests by tags
func (m *ComplianceTestSuiteManager) filterTestsByTags(suite *ComplianceTestSuite, tags []string) *ComplianceTestSuite {
	filteredSuite := &ComplianceTestSuite{
		Name:     suite.Name + " (Filtered by Tags)",
		Version:  suite.Version,
		Tests:    make([]ComplianceTest, 0),
		Metadata: suite.Metadata,
	}
	
	for _, test := range suite.Tests {
		for _, filterTag := range tags {
			for _, testTag := range test.Tags {
				if testTag == filterTag {
					filteredSuite.Tests = append(filteredSuite.Tests, test)
					break
				}
			}
		}
	}
	
	return filteredSuite
}

// filterTestsBySeverity filters tests by severity
func (m *ComplianceTestSuiteManager) filterTestsBySeverity(suite *ComplianceTestSuite, severity TestSeverity) *ComplianceTestSuite {
	filteredSuite := &ComplianceTestSuite{
		Name:     suite.Name + " (Filtered by Severity)",
		Version:  suite.Version,
		Tests:    make([]ComplianceTest, 0),
		Metadata: suite.Metadata,
	}
	
	for _, test := range suite.Tests {
		if test.Severity == severity {
			filteredSuite.Tests = append(filteredSuite.Tests, test)
		}
	}
	
	return filteredSuite
}

// extractIssuesFromSuite extracts compliance issues from test suite results
func (m *ComplianceTestSuiteManager) extractIssuesFromSuite(suite *ComplianceTestSuite) []ComplianceIssue {
	issues := make([]ComplianceIssue, 0)
	
	for _, result := range suite.Results {
		if result.Status == StatusFailed {
			// Find the corresponding test
			var test *ComplianceTest
			for _, t := range suite.Tests {
				if t.ID == result.TestID {
					test = &t
					break
				}
			}
			
			if test != nil {
				issue := ComplianceIssue{
					TestID:      result.TestID,
					Severity:    string(test.Severity),
					Description: result.Message,
					Requirement: test.Requirement,
					Evidence:    nil, // result.Evidence field doesn't exist
				}
				issues = append(issues, issue)
			}
		}
	}
	
	return issues
}

// calculateOverallCompliance computes overall compliance metrics
func (m *ComplianceTestSuiteManager) calculateOverallCompliance(standards map[string]StandardCompliance) ComplianceResult {
	overall := ComplianceResult{}
	
	totalScore := 0.0
	standardCount := 0
	
	for _, standard := range standards {
		overall.TotalTests += standard.TestCount
		overall.PassedTests += standard.Passed
		overall.FailedTests += standard.Failed
		
		totalScore += standard.Score
		standardCount++
	}
	
	if standardCount > 0 {
		overall.Score = totalScore / float64(standardCount)
	}
	
	overall.Compliant = overall.FailedTests == 0
	
	return overall
}

// ExportTestSuiteDefinitions exports test suite definitions to JSON
func (m *ComplianceTestSuiteManager) ExportTestSuiteDefinitions() ([]byte, error) {
	export := struct {
		TestSuites map[string]*ComplianceTestSuite `json:"testSuites"`
		Metadata   map[string]interface{}          `json:"metadata"`
	}{
		TestSuites: m.suites,
		Metadata: map[string]interface{}{
			"exportTime": time.Now(),
			"version":    "1.0.0",
			"framework":  "O-RAN Compliance Testing Framework",
		},
	}
	
	return json.MarshalIndent(export, "", "  ")
}

// ImportTestSuiteDefinitions imports test suite definitions from JSON
func (m *ComplianceTestSuiteManager) ImportTestSuiteDefinitions(data []byte) error {
	var import_ struct {
		TestSuites map[string]*ComplianceTestSuite `json:"testSuites"`
		Metadata   map[string]interface{}          `json:"metadata"`
	}
	
	if err := json.Unmarshal(data, &import_); err != nil {
		return fmt.Errorf("failed to unmarshal test suite definitions: %w", err)
	}
	
	// Merge imported test suites
	for name, suite := range import_.TestSuites {
		m.suites[name] = suite
	}
	
	return nil
}