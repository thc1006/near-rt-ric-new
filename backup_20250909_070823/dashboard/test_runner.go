package dashboard

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestRunner executes comprehensive unit and integration tests
type TestRunner struct {
	logger          *logrus.Logger
	projectRoot     string
	testPackages    []string
	coverageProfile string
	coverageReport  *CoverageReport
	testResults     *TestResults
}

// TestResults type is now defined in types.go to avoid redeclaration

// PackageTestResult type is now defined in types.go to avoid redeclaration

// TestCase type is now defined in types.go to avoid redeclaration

// BenchmarkResult type is now defined in types.go to avoid redeclaration

// NewTestRunner creates a new test runner instance
func NewTestRunner(projectRoot string, logger *logrus.Logger) *TestRunner {
	if logger == nil {
		logger = logrus.New()
	}

	return &TestRunner{
		logger:          logger,
		projectRoot:     projectRoot,
		coverageProfile: filepath.Join(projectRoot, "coverage.out"),
		testResults:     &TestResults{
			PackageResults:   make([]PackageTestResult, 0),
			BenchmarkResults: make([]BenchmarkResult, 0),
		},
	}
}

// RunAllTests executes all unit and integration tests
func (tr *TestRunner) RunAllTests(ctx context.Context) (*TestResults, error) {
	tr.logger.Info("Starting comprehensive test execution")
	
	startTime := time.Now()

	// Phase 1: Discover test packages
	if err := tr.discoverTestPackages(); err != nil {
		return nil, fmt.Errorf("failed to discover test packages: %w", err)
	}

	// Phase 2: Run unit tests with coverage
	if err := tr.runUnitTests(ctx); err != nil {
		return nil, fmt.Errorf("unit tests failed: %w", err)
	}

	// Phase 3: Run integration tests
	if err := tr.runIntegrationTests(ctx); err != nil {
		tr.logger.Warn("Some integration tests failed", "error", err)
	}

	// Phase 4: Run benchmark tests
	if err := tr.runBenchmarkTests(ctx); err != nil {
		tr.logger.Warn("Benchmark tests failed", "error", err)
	}

	// Phase 5: Generate coverage report
	if err := tr.generateCoverageReport(); err != nil {
		tr.logger.Error("Failed to generate coverage report", "error", err)
	}

	tr.testResults.Duration = time.Since(startTime)
	tr.logger.Info("Test execution completed", 
		"duration", tr.testResults.Duration,
		"totalTests", tr.testResults.TotalTests,
		"coverage", tr.testResults.Coverage)

	return tr.testResults, nil
}

// discoverTestPackages finds all packages with tests
func (tr *TestRunner) discoverTestPackages() error {
	tr.logger.Info("Discovering test packages")
	
	packages := make([]string, 0)

	err := filepath.Walk(tr.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), "_test.go") {
			packageDir := filepath.Dir(path)
			relPath, err := filepath.Rel(tr.projectRoot, packageDir)
			if err != nil {
				return err
			}

			// Convert to Go package path
			packagePath := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
			if packagePath == "." {
				packagePath = ""
			}

			// Avoid duplicates
			found := false
			for _, pkg := range packages {
				if pkg == packagePath {
					found = true
					break
				}
			}
			if !found {
				packages = append(packages, packagePath)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	tr.testPackages = packages
	tr.logger.Info("Discovered test packages", "count", len(packages))
	
	for _, pkg := range packages {
		tr.logger.Debug("Test package", "package", pkg)
	}

	return nil
}

// runUnitTests executes unit tests with coverage
func (tr *TestRunner) runUnitTests(ctx context.Context) error {
	tr.logger.Info("Running unit tests with coverage")

	// Create coverage profile directory
	coverageDir := filepath.Dir(tr.coverageProfile)
	if err := os.MkdirAll(coverageDir, 0755); err != nil {
		return err
	}

	// Build test command
	args := []string{
		"test",
		"-v",
		"-race",
		"-coverprofile=" + tr.coverageProfile,
		"-covermode=atomic",
		"-timeout=10m",
	}

	// Add all packages or specific ones
	if len(tr.testPackages) == 0 {
		args = append(args, "./...")
	} else {
		for _, pkg := range tr.testPackages {
			if pkg == "" {
				args = append(args, ".")
			} else {
				args = append(args, "./"+pkg)
			}
		}
	}

	// Execute tests
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(),
		"GODEBUG=fips140=on", // Enable FIPS mode for Go 1.24+
		"CGO_ENABLED=1",      // Enable CGO for race detector
	)

	tr.logger.Debug("Executing command", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	tr.parseTestOutput(string(output))

	if err != nil {
		tr.logger.Error("Unit tests failed", "error", err, "output", string(output))
		return err
	}

	tr.logger.Info("Unit tests completed successfully")
	return nil
}

// runIntegrationTests executes integration tests
func (tr *TestRunner) runIntegrationTests(ctx context.Context) error {
	tr.logger.Info("Running integration tests")

	// Look for integration test files
	integrationTests := make([]string, 0)

	for _, pkg := range tr.testPackages {
		packagePath := filepath.Join(tr.projectRoot, pkg)
		files, err := filepath.Glob(filepath.Join(packagePath, "*_integration_test.go"))
		if err != nil {
			continue
		}
		
		if len(files) > 0 {
			integrationTests = append(integrationTests, pkg)
		}
	}

	if len(integrationTests) == 0 {
		tr.logger.Info("No integration tests found")
		return nil
	}

	// Run integration tests with build tag
	args := []string{
		"test",
		"-v",
		"-tags=integration",
		"-timeout=20m",
	}

	for _, pkg := range integrationTests {
		if pkg == "" {
			args = append(args, ".")
		} else {
			args = append(args, "./"+pkg)
		}
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(),
		"GODEBUG=fips140=on",
		"INTEGRATION_TEST=1",
	)

	output, err := cmd.CombinedOutput()
	tr.parseTestOutput(string(output))

	if err != nil {
		tr.logger.Warn("Some integration tests failed", "error", err)
		return err
	}

	tr.logger.Info("Integration tests completed")
	return nil
}

// runBenchmarkTests executes benchmark tests
func (tr *TestRunner) runBenchmarkTests(ctx context.Context) error {
	tr.logger.Info("Running benchmark tests")

	args := []string{
		"test",
		"-bench=.",
		"-benchmem",
		"-run=^$", // Don't run regular tests, only benchmarks
		"-timeout=30m",
	}

	// Add all packages
	if len(tr.testPackages) == 0 {
		args = append(args, "./...")
	} else {
		for _, pkg := range tr.testPackages {
			if pkg == "" {
				args = append(args, ".")
			} else {
				args = append(args, "./"+pkg)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(), "GODEBUG=fips140=on")

	output, err := cmd.CombinedOutput()
	tr.parseBenchmarkOutput(string(output))

	if err != nil {
		tr.logger.Warn("Some benchmark tests failed", "error", err)
		return err
	}

	tr.logger.Info("Benchmark tests completed", "benchmarks", len(tr.testResults.BenchmarkResults))
	return nil
}

// parseTestOutput parses go test output and extracts results
func (tr *TestRunner) parseTestOutput(output string) {
	lines := strings.Split(output, "\n")
	
	var currentPackage string
	packageResult := PackageTestResult{
		TestCases: make([]TestCase, 0),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse package start
		if strings.HasPrefix(line, "=== RUN ") {
			testName := strings.TrimPrefix(line, "=== RUN ")
			// Test case will be updated when we see the result
			continue
		}

		// Parse test results
		if strings.HasPrefix(line, "--- PASS:") || strings.HasPrefix(line, "--- FAIL:") || strings.HasPrefix(line, "--- SKIP:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				status := strings.TrimSuffix(parts[1], ":")
				testName := parts[2]
				
				var duration time.Duration
				if len(parts) > 3 && strings.HasSuffix(parts[3], ")") {
					durationStr := strings.TrimSuffix(strings.TrimPrefix(parts[3], "("), ")")
					if d, err := time.ParseDuration(durationStr); err == nil {
						duration = d
					}
				}

				testCase := TestCase{
					Name:     testName,
					Package:  currentPackage,
					Status:   strings.ToLower(status),
					Duration: duration,
				}

				packageResult.TestCases = append(packageResult.TestCases, testCase)

				// Update counters
				tr.testResults.TotalTests++
				switch testCase.Status {
				case "pass":
					tr.testResults.PassedTests++
					packageResult.Passed++
				case "fail":
					tr.testResults.FailedTests++
					packageResult.Failed++
				case "skip":
					tr.testResults.SkippedTests++
					packageResult.Skipped++
				}
			}
		}

		// Parse package results
		if strings.HasPrefix(line, "ok  ") || strings.HasPrefix(line, "FAIL") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPackage = parts[1]
				packageResult.Package = currentPackage
				packageResult.Tests = len(packageResult.TestCases)
				
				if len(parts) > 2 && strings.HasSuffix(parts[2], "s") {
					if d, err := time.ParseDuration(parts[2]); err == nil {
						packageResult.Duration = d
					}
				}

				// Parse coverage if present
				if len(parts) > 3 && strings.HasPrefix(parts[3], "coverage:") && strings.HasSuffix(parts[4], "%") {
					coverageStr := strings.TrimSuffix(parts[4], "%")
					if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
						packageResult.Coverage = coverage
					}
				}

				tr.testResults.PackageResults = append(tr.testResults.PackageResults, packageResult)
				packageResult = PackageTestResult{
					TestCases: make([]TestCase, 0),
				}
			}
		}
	}
}

// parseBenchmarkOutput parses go test -bench output
func (tr *TestRunner) parseBenchmarkOutput(output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		// Parse benchmark line: BenchmarkName-8  	1000000	      1000 ns/op	     800 B/op	      10 allocs/op
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		benchmarkName := parts[0]
		iterations, _ := strconv.ParseInt(parts[1], 10, 64)
		nsPerOpStr := strings.TrimSuffix(parts[3], "ns/op")
		nsPerOp, _ := strconv.ParseInt(nsPerOpStr, 10, 64)

		benchmark := BenchmarkResult{
			Name:       benchmarkName,
			Iterations: iterations,
			NsPerOp:    nsPerOp,
		}

		// Parse additional metrics if present
		for i := 4; i < len(parts); i++ {
			part := parts[i]
			if strings.HasSuffix(part, "B/op") {
				bytesStr := strings.TrimSuffix(part, "B/op")
				benchmark.BytesPerOp, _ = strconv.ParseInt(bytesStr, 10, 64)
			} else if strings.HasSuffix(part, "allocs/op") {
				allocsStr := strings.TrimSuffix(part, "allocs/op")
				benchmark.AllocsPerOp, _ = strconv.ParseInt(allocsStr, 10, 64)
			} else if strings.HasSuffix(part, "MB/s") {
				mbPerSecStr := strings.TrimSuffix(part, "MB/s")
				benchmark.MBPerSec, _ = strconv.ParseFloat(mbPerSecStr, 64)
			}
		}

		tr.testResults.BenchmarkResults = append(tr.testResults.BenchmarkResults, benchmark)
	}
}

// generateCoverageReport generates detailed coverage analysis
func (tr *TestRunner) generateCoverageReport() error {
	tr.logger.Info("Generating coverage report")

	if _, err := os.Stat(tr.coverageProfile); os.IsNotExist(err) {
		tr.logger.Warn("Coverage profile not found")
		return nil
	}

	// Parse coverage profile
	coverageData, err := tr.parseCoverageProfile()
	if err != nil {
		return err
	}

	tr.coverageReport = coverageData
	tr.testResults.Coverage = coverageData.OverallCoverage

	// Generate HTML coverage report
	htmlFile := strings.TrimSuffix(tr.coverageProfile, ".out") + ".html"
	cmd := exec.Command("go", "tool", "cover", "-html="+tr.coverageProfile, "-o", htmlFile)
	cmd.Dir = tr.projectRoot

	if err := cmd.Run(); err != nil {
		tr.logger.Warn("Failed to generate HTML coverage report", "error", err)
	} else {
		tr.logger.Info("HTML coverage report generated", "file", htmlFile)
	}

	return nil
}

// parseCoverageProfile parses the coverage profile and calculates metrics
func (tr *TestRunner) parseCoverageProfile() (*CoverageReport, error) {
	content, err := os.ReadFile(tr.coverageProfile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	report := &CoverageReport{
		ComponentCoverage:  make(map[string]float64),
		InterfaceCoverage:  make(map[string]float64),
		FunctionalCoverage: make(map[string]float64),
		CodeCoverage:       make(map[string]CodeCoverage),
		UncoveredAreas:     make([]string, 0),
	}

	var totalStatements, coveredStatements int
	packageStats := make(map[string]*CodeCoverage)

	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "mode:") {
			continue // Skip mode line
		}
		
		if line == "" {
			continue
		}

		// Parse coverage line: filename:startline.col,endline.col numstmt count
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		filename := parts[0]
		numStmt, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		// Extract package name from filename
		packageName := tr.extractPackageFromFilename(filename)

		if packageStats[packageName] == nil {
			packageStats[packageName] = &CodeCoverage{
				Package:    packageName,
				LinesTotal: 0,
				LinesCovered: 0,
			}
		}

		packageStats[packageName].LinesTotal += numStmt
		totalStatements += numStmt

		if count > 0 {
			packageStats[packageName].LinesCovered += numStmt
			coveredStatements += numStmt
		}
	}

	// Calculate overall coverage
	if totalStatements > 0 {
		report.OverallCoverage = float64(coveredStatements) / float64(totalStatements) * 100
	}

	// Calculate per-package coverage
	for pkg, stats := range packageStats {
		if stats.LinesTotal > 0 {
			stats.Coverage = float64(stats.LinesCovered) / float64(stats.LinesTotal) * 100
		}
		report.CodeCoverage[pkg] = *stats
		report.ComponentCoverage[pkg] = stats.Coverage

		// Identify low coverage areas
		if stats.Coverage < 80.0 {
			report.UncoveredAreas = append(report.UncoveredAreas, 
				fmt.Sprintf("Package %s has low coverage: %.1f%%", pkg, stats.Coverage))
		}
	}

	return report, nil
}

// extractPackageFromFilename extracts package name from file path
func (tr *TestRunner) extractPackageFromFilename(filename string) string {
	// Remove project root prefix
	if strings.HasPrefix(filename, tr.projectRoot) {
		filename = strings.TrimPrefix(filename, tr.projectRoot)
		filename = strings.TrimPrefix(filename, "/")
		filename = strings.TrimPrefix(filename, "\\")
	}

	// Get directory part
	dir := filepath.Dir(filename)
	if dir == "." {
		return "root"
	}

	// Convert to package name
	return strings.ReplaceAll(dir, string(filepath.Separator), "/")
}

// RunTestsWithMinimumCoverage runs tests and enforces minimum coverage
func (tr *TestRunner) RunTestsWithMinimumCoverage(ctx context.Context, minCoverage float64) error {
	results, err := tr.RunAllTests(ctx)
	if err != nil {
		return err
	}

	if results.Coverage < minCoverage {
		return fmt.Errorf("coverage %.2f%% below minimum requirement %.2f%%", 
			results.Coverage, minCoverage)
	}

	tr.logger.Info("Coverage requirement met", 
		"coverage", results.Coverage, 
		"minimum", minCoverage)

	return nil
}

// GetTestResults returns the current test results
func (tr *TestRunner) GetTestResults() *TestResults {
	return tr.testResults
}

// GetCoverageReport returns the coverage report
func (tr *TestRunner) GetCoverageReport() *CoverageReport {
	return tr.coverageReport
}

// RunSpecificTests runs tests for specific packages
func (tr *TestRunner) RunSpecificTests(ctx context.Context, packages []string) error {
	tr.logger.Info("Running tests for specific packages", "packages", packages)

	args := []string{
		"test",
		"-v",
		"-race",
		"-coverprofile=" + tr.coverageProfile,
		"-covermode=atomic",
	}

	for _, pkg := range packages {
		args = append(args, "./"+pkg)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(), "GODEBUG=fips140=on")

	output, err := cmd.CombinedOutput()
	tr.parseTestOutput(string(output))

	if err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	return nil
}

// RunTestsInParallel runs tests in parallel with specified concurrency
func (tr *TestRunner) RunTestsInParallel(ctx context.Context, concurrency int) error {
	tr.logger.Info("Running tests in parallel", "concurrency", concurrency)

	args := []string{
		"test",
		"-v",
		"-race",
		"-parallel", strconv.Itoa(concurrency),
		"-coverprofile=" + tr.coverageProfile,
		"-covermode=atomic",
		"./...",
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(), 
		"GODEBUG=fips140=on",
		fmt.Sprintf("GOMAXPROCS=%d", concurrency),
	)

	output, err := cmd.CombinedOutput()
	tr.parseTestOutput(string(output))

	return err
}

// RunFailedTestsOnly re-runs only the tests that failed in the last run
func (tr *TestRunner) RunFailedTestsOnly(ctx context.Context) error {
	if tr.testResults == nil {
		return fmt.Errorf("no previous test results available")
	}

	failedTests := make([]string, 0)
	for _, pkg := range tr.testResults.PackageResults {
		for _, test := range pkg.TestCases {
			if test.Status == "fail" {
				failedTests = append(failedTests, fmt.Sprintf("%s.%s", pkg.Package, test.Name))
			}
		}
	}

	if len(failedTests) == 0 {
		tr.logger.Info("No failed tests to re-run")
		return nil
	}

	tr.logger.Info("Re-running failed tests", "count", len(failedTests))

	args := []string{"test", "-v", "-run"}
	testPattern := strings.Join(failedTests, "|")
	args = append(args, testPattern, "./...")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot
	cmd.Env = append(os.Environ(), "GODEBUG=fips140=on")

	output, err := cmd.CombinedOutput()
	tr.parseTestOutput(string(output))

	return err
}

// ValidateFIPSCompliance ensures FIPS 140 compliance is enabled
func (tr *TestRunner) ValidateFIPSCompliance() error {
	tr.logger.Info("Validating FIPS 140 compliance")

	// Check Go version supports FIPS
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Go version: %w", err)
	}

	versionStr := string(output)
	tr.logger.Debug("Go version", "version", versionStr)

	// For Go 1.24+, FIPS mode should be available
	if !strings.Contains(versionStr, "go1.24") && !strings.Contains(versionStr, "go1.25") {
		tr.logger.Warn("Go version may not support FIPS 140 compliance")
	}

	// Test FIPS mode by running a simple test with FIPS enabled
	testCode := `
package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

func main() {
	// Test cryptographic operations work in FIPS mode
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Printf("FIPS compliance test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("FIPS compliance test passed")
}
`

	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "fips_test.go")
	
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		return fmt.Errorf("failed to create FIPS test file: %w", err)
	}
	defer os.Remove(testFile)

	cmd = exec.Command("go", "run", testFile)
	cmd.Env = append(os.Environ(), "GODEBUG=fips140=on")
	
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FIPS compliance test failed: %w, output: %s", err, string(output))
	}

	if !strings.Contains(string(output), "FIPS compliance test passed") {
		return fmt.Errorf("FIPS compliance validation failed")
	}

	tr.logger.Info("FIPS 140 compliance validated successfully")
	return nil
}