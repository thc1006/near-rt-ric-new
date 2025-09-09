package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ComplianceHandlers provides HTTP handlers for compliance testing
type ComplianceHandlers struct {
	runner      *ComplianceTestRunner
	suiteManager *ComplianceTestSuiteManager
	logger      Logger
}

// NewComplianceHandlers creates new compliance handlers
func NewComplianceHandlers(config *ComplianceConfig, logger Logger) *ComplianceHandlers {
	runner := NewComplianceTestRunner(config, logger)
	suiteManager := NewComplianceTestSuiteManager(runner)
	
	return &ComplianceHandlers{
		runner:      runner,
		suiteManager: suiteManager,
		logger:      logger,
	}
}

// RegisterRoutes registers compliance testing routes
func (h *ComplianceHandlers) RegisterRoutes(router *mux.Router) {
	// Test suite management
	router.HandleFunc("/api/compliance/suites", h.GetTestSuites).Methods("GET")
	router.HandleFunc("/api/compliance/suites/{suite}", h.GetTestSuite).Methods("GET")
	router.HandleFunc("/api/compliance/suites/{suite}/tests", h.GetTestSuiteTests).Methods("GET")
	
	// Test execution
	router.HandleFunc("/api/compliance/run", h.RunAllTests).Methods("POST")
	router.HandleFunc("/api/compliance/run/{suite}", h.RunTestSuite).Methods("POST")
	router.HandleFunc("/api/compliance/run/test/{testId}", h.RunSingleTest).Methods("POST")
	
	// Test filtering
	router.HandleFunc("/api/compliance/run/tags", h.RunTestsByTags).Methods("POST")
	router.HandleFunc("/api/compliance/run/severity/{severity}", h.RunTestsBySeverity).Methods("POST")
	
	// Results and reporting
	router.HandleFunc("/api/compliance/results", h.GetComplianceResults).Methods("GET")
	router.HandleFunc("/api/compliance/results/{suite}", h.GetSuiteResults).Methods("GET")
	router.HandleFunc("/api/compliance/report", h.GenerateComplianceReport).Methods("GET")
	router.HandleFunc("/api/compliance/report/export", h.ExportComplianceReport).Methods("GET")
	
	// Test suite management
	router.HandleFunc("/api/compliance/suites/export", h.ExportTestSuites).Methods("GET")
	router.HandleFunc("/api/compliance/suites/import", h.ImportTestSuites).Methods("POST")
	
	// Health and status
	router.HandleFunc("/api/compliance/health", h.GetComplianceHealth).Methods("GET")
	router.HandleFunc("/api/compliance/status", h.GetComplianceStatus).Methods("GET")
}

// GetTestSuites returns all available test suites
func (h *ComplianceHandlers) GetTestSuites(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Getting all test suites")
	
	suites := h.suiteManager.GetAllTestSuites()
	
	response := make(map[string]interface{})
	for name, suite := range suites {
		response[name] = map[string]interface{}{
			"name":        suite.Name,
			"version":     suite.Version,
			"testCount":   len(suite.Tests),
			"metadata":    suite.Metadata,
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTestSuite returns a specific test suite
func (h *ComplianceHandlers) GetTestSuite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suiteName := vars["suite"]
	
	h.logger.Info("Getting test suite", "suite", suiteName)
	
	suite, err := h.suiteManager.GetTestSuite(suiteName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Test suite not found: %v", err), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suite)
}

// GetTestSuiteTests returns tests for a specific suite
func (h *ComplianceHandlers) GetTestSuiteTests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suiteName := vars["suite"]
	
	h.logger.Info("Getting test suite tests", "suite", suiteName)
	
	suite, err := h.suiteManager.GetTestSuite(suiteName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Test suite not found: %v", err), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suite.Tests)
}

// RunAllTests runs all compliance test suites
func (h *ComplianceHandlers) RunAllTests(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Running all compliance tests")
	
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	
	report, err := h.suiteManager.RunAllTestSuites(ctx)
	if err != nil {
		h.logger.Error("Failed to run all test suites", "error", err)
		http.Error(w, fmt.Sprintf("Failed to run tests: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// RunTestSuite runs a specific test suite
func (h *ComplianceHandlers) RunTestSuite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suiteName := vars["suite"]
	
	h.logger.Info("Running test suite", "suite", suiteName)
	
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	
	compliance, err := h.suiteManager.RunTestSuiteByName(ctx, suiteName)
	if err != nil {
		h.logger.Error("Failed to run test suite", "suite", suiteName, "error", err)
		http.Error(w, fmt.Sprintf("Failed to run test suite: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compliance)
}

// RunSingleTest runs a single compliance test
func (h *ComplianceHandlers) RunSingleTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]
	
	h.logger.Info("Running single test", "testId", testID)
	
	// Find the test in all suites
	var test *ComplianceTest
	var suiteName string
	
	for name, suite := range h.suiteManager.GetAllTestSuites() {
		for _, t := range suite.Tests {
			if t.ID == testID {
				test = &t
				suiteName = name
				break
			}
		}
		if test != nil {
			break
		}
	}
	
	if test == nil {
		http.Error(w, fmt.Sprintf("Test %s not found", testID), http.StatusNotFound)
		return
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	
	result := h.runner.runSingleTest(ctx, *test)
	
	response := map[string]interface{}{
		"testId":    testID,
		"suite":     suiteName,
		"result":    result,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RunTestsByTags runs tests filtered by tags
func (h *ComplianceHandlers) RunTestsByTags(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Tags []string `json:"tags"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	h.logger.Info("Running tests by tags", "tags", request.Tags)
	
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
	defer cancel()
	
	report, err := h.suiteManager.RunTestsByTag(ctx, request.Tags)
	if err != nil {
		h.logger.Error("Failed to run tests by tags", "tags", request.Tags, "error", err)
		http.Error(w, fmt.Sprintf("Failed to run tests: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// RunTestsBySeverity runs tests filtered by severity
func (h *ComplianceHandlers) RunTestsBySeverity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	severityStr := vars["severity"]
	
	severity := TestSeverity(severityStr)
	if severity != SeverityCritical && severity != SeverityHigh && 
	   severity != SeverityMedium && severity != SeverityLow {
		http.Error(w, "Invalid severity level", http.StatusBadRequest)
		return
	}
	
	h.logger.Info("Running tests by severity", "severity", severity)
	
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
	defer cancel()
	
	report, err := h.suiteManager.RunTestsBySeverity(ctx, severity)
	if err != nil {
		h.logger.Error("Failed to run tests by severity", "severity", severity, "error", err)
		http.Error(w, fmt.Sprintf("Failed to run tests: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetComplianceResults returns overall compliance results
func (h *ComplianceHandlers) GetComplianceResults(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Getting compliance results")
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	
	report, err := h.runner.ValidateCompliance(ctx)
	if err != nil {
		h.logger.Error("Failed to get compliance results", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get results: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetSuiteResults returns results for a specific suite
func (h *ComplianceHandlers) GetSuiteResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	suiteName := vars["suite"]
	
	h.logger.Info("Getting suite results", "suite", suiteName)
	
	suite, err := h.suiteManager.GetTestSuite(suiteName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Test suite not found: %v", err), http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"suite":     suiteName,
		"results":   suite.Results,
		"summary":   suite.Summary,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GenerateComplianceReport generates a comprehensive compliance report
func (h *ComplianceHandlers) GenerateComplianceReport(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Generating compliance report")
	
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	
	report, err := h.suiteManager.RunAllTestSuites(ctx)
	if err != nil {
		h.logger.Error("Failed to generate compliance report", "error", err)
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}
	
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	case "html":
		h.generateHTMLReport(w, report)
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

// ExportComplianceReport exports compliance report in various formats
func (h *ComplianceHandlers) ExportComplianceReport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	h.logger.Info("Exporting compliance report", "format", format)
	
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	
	report, err := h.suiteManager.RunAllTestSuites(ctx)
	if err != nil {
		h.logger.Error("Failed to export compliance report", "error", err)
		http.Error(w, fmt.Sprintf("Failed to export report: %v", err), http.StatusInternalServerError)
		return
	}
	
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("oran-compliance-report-%s.%s", timestamp, format)
	
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	case "csv":
		h.generateCSVReport(w, report)
	default:
		http.Error(w, "Unsupported export format", http.StatusBadRequest)
	}
}

// ExportTestSuites exports test suite definitions
func (h *ComplianceHandlers) ExportTestSuites(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Exporting test suites")
	
	data, err := h.suiteManager.ExportTestSuiteDefinitions()
	if err != nil {
		h.logger.Error("Failed to export test suites", "error", err)
		http.Error(w, fmt.Sprintf("Failed to export: %v", err), http.StatusInternalServerError)
		return
	}
	
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("oran-test-suites-%s.json", timestamp)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(data)
}

// ImportTestSuites imports test suite definitions
func (h *ComplianceHandlers) ImportTestSuites(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Importing test suites")
	
	if r.ContentLength > 10*1024*1024 { // 10MB limit
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}
	
	data := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(data); err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	
	if err := h.suiteManager.ImportTestSuiteDefinitions(data); err != nil {
		h.logger.Error("Failed to import test suites", "error", err)
		http.Error(w, fmt.Sprintf("Failed to import: %v", err), http.StatusBadRequest)
		return
	}
	
	response := map[string]interface{}{
		"message":   "Test suites imported successfully",
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetComplianceHealth returns compliance testing health status
func (h *ComplianceHandlers) GetComplianceHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]interface{}{
			"testRunner":    "operational",
			"suiteManager":  "operational",
			"testSuites":    len(h.suiteManager.GetAllTestSuites()),
		},
	}
	
	// Check component connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	if err := h.checkComponentConnectivity(ctx); err != nil {
		health["status"] = "degraded"
		health["issues"] = []string{err.Error()}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// GetComplianceStatus returns current compliance status
func (h *ComplianceHandlers) GetComplianceStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"timestamp": time.Now(),
		"testSuites": make(map[string]interface{}),
	}
	
	for name, suite := range h.suiteManager.GetAllTestSuites() {
		suiteStatus := map[string]interface{}{
			"name":      suite.Name,
			"version":   suite.Version,
			"testCount": len(suite.Tests),
			"lastRun":   nil,
		}
		
		if len(suite.Results) > 0 {
			suiteStatus["lastRun"] = suite.Results[0].Timestamp
			suiteStatus["summary"] = suite.Summary
		}
		
		status["testSuites"].(map[string]interface{})[name] = suiteStatus
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper methods

func (h *ComplianceHandlers) generateHTMLReport(w http.ResponseWriter, report *ComplianceReport) {
	w.Header().Set("Content-Type", "text/html")
	
	html := `<!DOCTYPE html>
<html>
<head>
    <title>O-RAN Compliance Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .standard { margin: 20px 0; border: 1px solid #ddd; padding: 15px; border-radius: 5px; }
        .passed { color: green; }
        .failed { color: red; }
        .score { font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>O-RAN Compliance Report</h1>
        <p>Generated: %s</p>
    </div>
    
    <div class="summary">
        <h2>Overall Compliance</h2>
        <p>Score: <span class="score">%.1f%%</span></p>
        <p>Status: <span class="%s">%s</span></p>
        <p>Total Tests: %d | Passed: %d | Failed: %d</p>
    </div>
    
    <div class="standards">
        <h2>Standards Compliance</h2>
        %s
    </div>
</body>
</html>`
	
	overallStatus := "failed"
	overallClass := "failed"
	if report.OverallCompliance.Compliant {
		overallStatus = "compliant"
		overallClass = "passed"
	}
	
	standardsHTML := ""
	for name, standard := range report.Standards {
		status := "failed"
		class := "failed"
		if standard.Compliant {
			status = "compliant"
			class = "passed"
		}
		
		standardHTML := fmt.Sprintf(`
        <div class="standard">
            <h3>%s</h3>
            <p>Version: %s</p>
            <p>Score: <span class="score">%.1f%%</span></p>
            <p>Status: <span class="%s">%s</span></p>
            <p>Tests: %d | Passed: %d | Failed: %d</p>
        </div>`,
			standard.Standard,
			standard.Version,
			standard.Score,
			class,
			status,
			standard.TestSuite.Summary.Total,
			standard.TestSuite.Summary.Passed,
			standard.TestSuite.Summary.Failed,
		)
		
		standardsHTML += standardHTML
	}
	
	finalHTML := fmt.Sprintf(html,
		report.Timestamp.Format("2006-01-02 15:04:05"),
		report.OverallCompliance.Score,
		overallClass,
		overallStatus,
		report.OverallCompliance.TotalTests,
		report.OverallCompliance.PassedTests,
		report.OverallCompliance.FailedTests,
		standardsHTML,
	)
	
	w.Write([]byte(finalHTML))
}

func (h *ComplianceHandlers) generateCSVReport(w http.ResponseWriter, report *ComplianceReport) {
	w.Header().Set("Content-Type", "text/csv")
	
	// CSV header
	csv := "Standard,Version,Score,Compliant,Total Tests,Passed Tests,Failed Tests\n"
	
	// Overall compliance
	csv += fmt.Sprintf("Overall,N/A,%.1f,%t,%d,%d,%d\n",
		report.OverallCompliance.Score,
		report.OverallCompliance.Compliant,
		report.OverallCompliance.TotalTests,
		report.OverallCompliance.PassedTests,
		report.OverallCompliance.FailedTests,
	)
	
	// Individual standards
	for _, standard := range report.Standards {
		csv += fmt.Sprintf("%s,%s,%.1f,%t,%d,%d,%d\n",
			standard.Standard,
			standard.Version,
			standard.Score,
			standard.Compliant,
			standard.TestSuite.Summary.Total,
			standard.TestSuite.Summary.Passed,
			standard.TestSuite.Summary.Failed,
		)
	}
	
	w.Write([]byte(csv))
}

func (h *ComplianceHandlers) checkComponentConnectivity(ctx context.Context) error {
	// Check E2T connectivity
	if h.runner.config.E2TermEndpoint != "" {
		// In a real implementation, this would check actual connectivity
	}
	
	// Check A1 Mediator connectivity
	if h.runner.config.A1MediatorURL != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", h.runner.config.A1MediatorURL+"/a1-p/healthcheck", nil)
		if err != nil {
			return fmt.Errorf("failed to create A1 health check request: %w", err)
		}
		
		resp, err := h.runner.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("A1 Mediator connectivity check failed: %w", err)
		}
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("A1 Mediator health check returned status %d", resp.StatusCode)
		}
	}
	
	// Check O1 Mediator connectivity
	if h.runner.config.O1MediatorURL != "" {
		// In a real implementation, this would check NETCONF connectivity
	}
	
	return nil
}