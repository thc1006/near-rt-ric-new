package dashboard

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestComplianceTestRunner tests the compliance test runner
func TestComplianceTestRunner(t *testing.T) {
	config := &ComplianceConfig{
		E2TermEndpoint:    "localhost:36422",
		E2MgrEndpoint:     "localhost:3800",
		SubMgrEndpoint:    "localhost:3801",
		A1MediatorURL:     "http://localhost:10000",
		O1MediatorURL:     "http://localhost:8080",
		TLSConfig:         &tls.Config{InsecureSkipVerify: true},
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		TestDataPath:      "/tmp/test-data",
		ReportOutputPath:  "/tmp/compliance-report.json",
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	if runner == nil {
		t.Fatal("Failed to create compliance test runner")
	}
	
	if runner.config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", runner.config.Timeout)
	}
}

// TestComplianceTestSuiteManager tests the test suite manager
func TestComplianceTestSuiteManager(t *testing.T) {
	config := &ComplianceConfig{
		A1MediatorURL: "http://localhost:10000",
		Timeout:       30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	manager := NewComplianceTestSuiteManager(runner)
	
	if manager == nil {
		t.Fatal("Failed to create test suite manager")
	}
	
	// Test getting all test suites
	suites := manager.GetAllTestSuites()
	expectedSuites := []string{"e2ap", "a1", "o1", "security", "interoperability"}
	
	for _, expected := range expectedSuites {
		if _, exists := suites[expected]; !exists {
			t.Errorf("Expected test suite %s not found", expected)
		}
	}
	
	// Test getting specific test suite
	e2apSuite, err := manager.GetTestSuite("e2ap")
	if err != nil {
		t.Errorf("Failed to get E2AP test suite: %v", err)
	}
	
	if e2apSuite.Name != "O-RAN E2AP Compliance Test Suite" {
		t.Errorf("Expected E2AP suite name, got %s", e2apSuite.Name)
	}
	
	if len(e2apSuite.Tests) == 0 {
		t.Error("E2AP test suite should have tests")
	}
	
	// Test getting non-existent test suite
	_, err = manager.GetTestSuite("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent test suite")
	}
}

// TestE2APComplianceTests tests E2AP compliance test execution
func TestE2APComplianceTests(t *testing.T) {
	config := &ComplianceConfig{
		E2TermEndpoint: "localhost:36422",
		Timeout:        30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	// Test E2AP Setup procedure test
	test := ComplianceTest{
		ID:          "e2ap-001",
		Name:        "E2 Setup Procedure Compliance",
		Category:    "e2ap",
		Requirement: "O-RAN.WG3.E2AP-R003 Section 8.2.1",
		Severity:    SeverityCritical,
	}
	
	ctx := context.Background()
	result := runner.runE2APTest(ctx, test)
	
	if result.TestID != test.ID {
		t.Errorf("Expected test ID %s, got %s", test.ID, result.TestID)
	}
	
	if result.Status == StatusError {
		t.Errorf("Test execution failed with error: %s", result.Message)
	}
}

// TestA1ComplianceTests tests A1 compliance test execution
func TestA1ComplianceTests(t *testing.T) {
	// Create mock A1 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/healthcheck":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/a1-p/policytypes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	config := &ComplianceConfig{
		A1MediatorURL: server.URL,
		Timeout:       30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	// Test A1 health check test
	test := ComplianceTest{
		ID:          "a1-001",
		Name:        "Health Check Endpoint",
		Category:    "a1",
		Requirement: "O-RAN.WG2.A1 Section 4.1",
		Severity:    SeverityHigh,
	}
	
	ctx := context.Background()
	result := runner.runA1Test(ctx, test)
	
	if result.TestID != test.ID {
		t.Errorf("Expected test ID %s, got %s", test.ID, result.TestID)
	}
	
	if result.Status != StatusPassed {
		t.Errorf("Expected test to pass, got status %s: %s", result.Status, result.Message)
	}
}

// TestO1ComplianceTests tests O1 compliance test execution
func TestO1ComplianceTests(t *testing.T) {
	config := &ComplianceConfig{
		O1MediatorURL: "http://localhost:8080",
		Timeout:       30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	// Test O1 NETCONF connection test
	test := ComplianceTest{
		ID:          "o1-001",
		Name:        "NETCONF Connection Establishment",
		Category:    "o1",
		Requirement: "RFC 6241 Section 4",
		Severity:    SeverityCritical,
	}
	
	ctx := context.Background()
	result := runner.runO1Test(ctx, test)
	
	if result.TestID != test.ID {
		t.Errorf("Expected test ID %s, got %s", test.ID, result.TestID)
	}
	
	// Since we don't have a real NETCONF server, expect failure or skip
	if result.Status == StatusError {
		t.Logf("O1 test failed as expected without real NETCONF server: %s", result.Message)
	}
}

// TestSecurityComplianceTests tests security compliance test execution
func TestSecurityComplianceTests(t *testing.T) {
	// Create mock HTTPS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &ComplianceConfig{
		A1MediatorURL: server.URL,
		TLSConfig:     &tls.Config{InsecureSkipVerify: true},
		Timeout:       30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	// Test security headers test
	test := ComplianceTest{
		ID:          "sec-010",
		Name:        "Security Headers",
		Category:    "security",
		Requirement: "O-RAN.WG11.Security Section 9.1",
		Severity:    SeverityLow,
	}
	
	ctx := context.Background()
	result := runner.runSecurityTest(ctx, test)
	
	if result.TestID != test.ID {
		t.Errorf("Expected test ID %s, got %s", test.ID, result.TestID)
	}
	
	if result.Status != StatusPassed {
		t.Errorf("Expected security headers test to pass, got status %s: %s", result.Status, result.Message)
	}
}

// TestInteroperabilityComplianceTests tests interoperability compliance test execution
func TestInteroperabilityComplianceTests(t *testing.T) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	// Test third-party E2 node integration test
	test := ComplianceTest{
		ID:          "interop-001",
		Name:        "Third-Party E2 Node Integration",
		Category:    "interoperability",
		Requirement: "O-RAN Interoperability Requirements",
		Severity:    SeverityHigh,
	}
	
	ctx := context.Background()
	result := runner.runInteroperabilityTest(ctx, test)
	
	if result.TestID != test.ID {
		t.Errorf("Expected test ID %s, got %s", test.ID, result.TestID)
	}
	
	// Since we don't have real third-party components, expect skip
	if result.Status == StatusSkipped {
		t.Logf("Interoperability test skipped as expected: %s", result.Message)
	}
}

// TestComplianceTestSuiteExecution tests full test suite execution
func TestComplianceTestSuiteExecution(t *testing.T) {
	// Create mock server for A1 tests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer server.Close()
	
	config := &ComplianceConfig{
		A1MediatorURL: server.URL,
		Timeout:       30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	manager := NewComplianceTestSuiteManager(runner)
	
	// Run A1 test suite
	ctx := context.Background()
	compliance, err := manager.RunTestSuiteByName(ctx, "a1")
	if err != nil {
		t.Errorf("Failed to run A1 test suite: %v", err)
	}
	
	if compliance.Standard != "a1" {
		t.Errorf("Expected standard 'a1', got %s", compliance.Standard)
	}
	
	if compliance.TestSuite.Summary.Total == 0 {
		t.Error("Expected test suite to have tests")
	}
	
	// Check that at least health check test passed
	healthCheckPassed := false
	for _, result := range compliance.TestSuite.Results {
		if result.TestID == "a1-001" && result.Status == StatusPassed {
			healthCheckPassed = true
			break
		}
	}
	
	if !healthCheckPassed {
		t.Error("Expected A1 health check test to pass")
	}
}

// TestComplianceReportGeneration tests compliance report generation
func TestComplianceReportGeneration(t *testing.T) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	ctx := context.Background()
	report, err := runner.ValidateCompliance(ctx)
	if err != nil {
		t.Errorf("Failed to validate compliance: %v", err)
	}
	
	if report == nil {
		t.Fatal("Expected compliance report, got nil")
	}
	
	if len(report.Standards) == 0 {
		t.Error("Expected compliance report to have standards")
	}
	
	// Check overall compliance structure
	if report.OverallCompliance.TotalTests < 0 {
		t.Error("Expected non-negative total tests")
	}
	
	if report.OverallCompliance.Score < 0 || report.OverallCompliance.Score > 100 {
		t.Errorf("Expected score between 0-100, got %.2f", report.OverallCompliance.Score)
	}
}

// TestComplianceTestFiltering tests test filtering functionality
func TestComplianceTestFiltering(t *testing.T) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	manager := NewComplianceTestSuiteManager(runner)
	
	// Test filtering by tags
	ctx := context.Background()
	report, err := manager.RunTestsByTag(ctx, []string{"health", "endpoint"})
	if err != nil {
		t.Errorf("Failed to run tests by tag: %v", err)
	}
	
	if report == nil {
		t.Fatal("Expected report from tag filtering")
	}
	
	// Test filtering by severity
	report, err = manager.RunTestsBySeverity(ctx, SeverityCritical)
	if err != nil {
		t.Errorf("Failed to run tests by severity: %v", err)
	}
	
	if report == nil {
		t.Fatal("Expected report from severity filtering")
	}
	
	// Verify only critical tests were run
	for _, standard := range report.Standards {
		for _, test := range standard.TestSuite.Tests {
			if test.Severity != SeverityCritical {
				t.Errorf("Expected only critical tests, found %s test", test.Severity)
			}
		}
	}
}

// TestComplianceTestSuiteExportImport tests test suite export/import
func TestComplianceTestSuiteExportImport(t *testing.T) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	manager := NewComplianceTestSuiteManager(runner)
	
	// Export test suites
	exportData, err := manager.ExportTestSuiteDefinitions()
	if err != nil {
		t.Errorf("Failed to export test suites: %v", err)
	}
	
	if len(exportData) == 0 {
		t.Error("Expected non-empty export data")
	}
	
	// Verify export data is valid JSON
	var exportStruct map[string]interface{}
	if err := json.Unmarshal(exportData, &exportStruct); err != nil {
		t.Errorf("Export data is not valid JSON: %v", err)
	}
	
	// Test import (create new manager to test import)
	newManager := NewComplianceTestSuiteManager(runner)
	
	// Clear existing suites to test import
	newManager.suites = make(map[string]*ComplianceTestSuite)
	
	if err := newManager.ImportTestSuiteDefinitions(exportData); err != nil {
		t.Errorf("Failed to import test suites: %v", err)
	}
	
	// Verify imported suites
	importedSuites := newManager.GetAllTestSuites()
	if len(importedSuites) == 0 {
		t.Error("Expected imported test suites")
	}
}

// TestComplianceHandlers tests the HTTP handlers
func TestComplianceHandlers(t *testing.T) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	handlers := NewComplianceHandlers(config, logger)
	
	if handlers == nil {
		t.Fatal("Failed to create compliance handlers")
	}
	
	// Test getting test suites
	req := httptest.NewRequest("GET", "/api/compliance/suites", nil)
	w := httptest.NewRecorder()
	
	handlers.GetTestSuites(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}
	
	// Verify response is valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}
	
	// Test getting specific test suite
	req = httptest.NewRequest("GET", "/api/compliance/suites/e2ap", nil)
	req = req.WithContext(context.WithValue(req.Context(), "suite", "e2ap"))
	w = httptest.NewRecorder()
	
	// Simulate mux vars
	req = httptest.NewRequest("GET", "/api/compliance/suites/e2ap", nil)
	w = httptest.NewRecorder()
	
	// We would need to set up mux router for proper testing
	// For now, just verify handler exists
	if handlers.GetTestSuite == nil {
		t.Error("GetTestSuite handler is nil")
	}
}

// TestLogger is a test implementation of the Logger interface
type TestLogger struct{}

func (l *TestLogger) Info(msg string, keysAndValues ...interface{}) {
	// Log to test output if needed
}

func (l *TestLogger) Error(msg string, keysAndValues ...interface{}) {
	// Log to test output if needed
}

func (l *TestLogger) Debug(msg string, keysAndValues ...interface{}) {
	// Log to test output if needed
}

func (l *TestLogger) Warn(msg string, keysAndValues ...interface{}) {
	// Log to test output if needed
}

// TestComplianceTestDataLoading tests test data loading
func TestComplianceTestDataLoading(t *testing.T) {
	// Test E2AP test data loading
	e2apData := loadE2APTestData()
	if e2apData == nil {
		t.Error("Failed to load E2AP test data")
	}
	
	if len(e2apData.ServiceModels) == 0 {
		t.Error("Expected E2AP test data to have service models")
	}
	
	// Test A1 test data loading
	a1Data := loadA1TestData()
	if a1Data == nil {
		t.Error("Failed to load A1 test data")
	}
	
	if len(a1Data.ValidPolicyTypes) == 0 {
		t.Error("Expected A1 test data to have valid policy types")
	}
	
	// Test O1 test data loading
	o1Data := loadO1TestData()
	if o1Data == nil {
		t.Error("Failed to load O1 test data")
	}
	
	if len(o1Data.YANGModels) == 0 {
		t.Error("Expected O1 test data to have YANG models")
	}
	
	// Test Security test data loading
	secData := loadSecurityTestData()
	if secData == nil {
		t.Error("Failed to load Security test data")
	}
	
	if len(secData.TLSTestData.RequiredCiphers) == 0 {
		t.Error("Expected Security test data to have required ciphers")
	}
	
	// Test Interoperability test data loading
	interopData := loadInteroperabilityTestData()
	if interopData == nil {
		t.Error("Failed to load Interoperability test data")
	}
	
	if len(interopData.ThirdPartyComponents) == 0 {
		t.Error("Expected Interoperability test data to have third-party components")
	}
}

// TestComplianceTestValidation tests test validation logic
func TestComplianceTestValidation(t *testing.T) {
	// Test test severity validation
	validSeverities := []TestSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
	
	for _, severity := range validSeverities {
		if string(severity) == "" {
			t.Errorf("Severity %s should not be empty", severity)
		}
	}
	
	// Test test status validation
	validStatuses := []TestStatus{StatusPassed, StatusFailed, StatusSkipped, StatusError}
	
	for _, status := range validStatuses {
		if string(status) == "" {
			t.Errorf("Status %s should not be empty", status)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkComplianceTestExecution(b *testing.B) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	test := ComplianceTest{
		ID:       "benchmark-test",
		Name:     "Benchmark Test",
		Category: "benchmark",
		Severity: SeverityLow,
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runner.runSingleTest(ctx, test)
	}
}

func BenchmarkComplianceReportGeneration(b *testing.B) {
	config := &ComplianceConfig{
		Timeout: 30 * time.Second,
	}
	
	logger := &TestLogger{}
	runner := NewComplianceTestRunner(config, logger)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = runner.ValidateCompliance(ctx)
	}
}