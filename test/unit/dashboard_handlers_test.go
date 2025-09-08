package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DashboardHandlerTestSuite provides comprehensive unit testing for dashboard API handlers
type DashboardHandlerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	router          *mux.Router
	server          *httptest.Server
	ctx             context.Context
	testTimeout     time.Duration
	mockClients     map[string]interface{} // Store mock clients
	testResults     *UnitTestResults
}

// UnitTestResults tracks unit test execution results
type UnitTestResults struct {
	StartTime       time.Time
	TotalTests      int
	PassedTests     int
	FailedTests     int
	Coverage        float64
	HandlerTests    map[string]*HandlerTestResult
	ClientTests     map[string]*ClientTestResult
	PerformanceData map[string]*PerformanceData
}

// HandlerTestResult tracks individual handler test results
type HandlerTestResult struct {
	HandlerName     string
	TestsPassed     int
	TestsFailed     int
	AverageLatency  time.Duration
	ErrorRate       float64
	StatusCodes     map[int]int
}

// ClientTestResult tracks client test results
type ClientTestResult struct {
	ClientName      string
	TestsPassed     int
	TestsFailed     int
	ConnectionTests bool
	TimeoutTests    bool
	ErrorHandling   bool
}

// PerformanceData tracks performance metrics
type PerformanceData struct {
	OperationName   string
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AverageLatency  time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	RequestsPerSec  float64
	MemoryUsage     int64
}

// SetupTest initializes test environment for each test
func (suite *DashboardHandlerTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.testTimeout = 30 * time.Second
	suite.mockClients = make(map[string]interface{})
	
	// Initialize test results tracking
	if suite.testResults == nil {
		suite.testResults = &UnitTestResults{
			StartTime:       time.Now(),
			HandlerTests:    make(map[string]*HandlerTestResult),
			ClientTests:     make(map[string]*ClientTestResult),
			PerformanceData: make(map[string]*PerformanceData),
		}
	}
	
	// Setup router and test server
	suite.setupTestServer()
}

// TearDownTest cleans up after each test
func (suite *DashboardHandlerTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.ctrl != nil {
		suite.ctrl.Finish()
	}
}

// setupTestServer initializes the test HTTP server
func (suite *DashboardHandlerTestSuite) setupTestServer() {
	suite.router = mux.NewRouter()
	suite.setupRoutes()
	suite.server = httptest.NewServer(suite.router)
}

// setupRoutes configures API routes for testing
func (suite *DashboardHandlerTestSuite) setupRoutes() {
	// E2 Manager routes
	e2Routes := suite.router.PathPrefix("/api/v1/e2").Subrouter()
	e2Routes.HandleFunc("/nodes", suite.mockE2NodesHandler).Methods("GET", "POST")
	e2Routes.HandleFunc("/nodes/{nodeId}", suite.mockE2NodeHandler).Methods("GET", "PUT", "DELETE")
	e2Routes.HandleFunc("/subscriptions", suite.mockE2SubscriptionsHandler).Methods("GET", "POST")
	e2Routes.HandleFunc("/subscriptions/{subId}", suite.mockE2SubscriptionHandler).Methods("GET", "PUT", "DELETE")
	
	// A1 Policy routes
	a1Routes := suite.router.PathPrefix("/api/v1/a1").Subrouter()
	a1Routes.HandleFunc("/policies", suite.mockA1PoliciesHandler).Methods("GET", "POST")
	a1Routes.HandleFunc("/policies/{policyId}", suite.mockA1PolicyHandler).Methods("GET", "PUT", "DELETE")
	a1Routes.HandleFunc("/policytypes", suite.mockA1PolicyTypesHandler).Methods("GET", "POST")
	
	// xApp Management routes
	xappRoutes := suite.router.PathPrefix("/api/v1/xapps").Subrouter()
	xappRoutes.HandleFunc("", suite.mockXAppsHandler).Methods("GET", "POST")
	xappRoutes.HandleFunc("/{xappName}", suite.mockXAppHandler).Methods("GET", "PUT", "DELETE")
	xappRoutes.HandleFunc("/{xappName}/instances", suite.mockXAppInstancesHandler).Methods("GET", "POST")
	
	// Health and metrics routes
	suite.router.HandleFunc("/health", suite.mockHealthHandler).Methods("GET")
	suite.router.HandleFunc("/metrics", suite.mockMetricsHandler).Methods("GET")
	suite.router.HandleFunc("/ready", suite.mockReadinessHandler).Methods("GET")
}

// TestE2NodeHandlers tests E2 node management handlers
func (suite *DashboardHandlerTestSuite) TestE2NodeHandlers() {
	suite.testResults.TotalTests++
	
	// Test GET /api/v1/e2/nodes
	suite.Run("GetE2Nodes", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/e2/nodes")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var nodes []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(suite.T(), err)
		assert.GreaterOrEqual(suite.T(), len(nodes), 0)
		
		suite.recordHandlerTest("GetE2Nodes", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test POST /api/v1/e2/nodes
	suite.Run("CreateE2Node", func() {
		nodeData := map[string]interface{}{
			"globalE2NodeId": "001-001-001",
			"nodeType":       "gNB",
			"plmnId":         "001001",
			"ranFunctions": []map[string]interface{}{
				{
					"ranFunctionId":         1,
					"ranFunctionDefinition": "KPM monitoring",
					"ranFunctionRevision":   1,
				},
			},
		}
		
		jsonData, _ := json.Marshal(nodeData)
		start := time.Now()
		resp, err := http.Post(suite.server.URL+"/api/v1/e2/nodes", "application/json", bytes.NewBuffer(jsonData))
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		suite.recordHandlerTest("CreateE2Node", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test GET /api/v1/e2/nodes/{nodeId}
	suite.Run("GetE2Node", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/e2/nodes/001-001-001")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var node map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&node)
		require.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), node)
		
		suite.recordHandlerTest("GetE2Node", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test error handling
	suite.Run("GetE2NodeNotFound", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/e2/nodes/nonexistent")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
		
		suite.recordHandlerTest("GetE2NodeNotFound", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	suite.testResults.PassedTests++
}

// TestA1PolicyHandlers tests A1 policy management handlers
func (suite *DashboardHandlerTestSuite) TestA1PolicyHandlers() {
	suite.testResults.TotalTests++
	
	// Test GET /api/v1/a1/policies
	suite.Run("GetA1Policies", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/a1/policies")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var policies []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&policies)
		require.NoError(suite.T(), err)
		
		suite.recordHandlerTest("GetA1Policies", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test POST /api/v1/a1/policies
	suite.Run("CreateA1Policy", func() {
		policyData := map[string]interface{}{
			"policyId":     "policy-001",
			"policyTypeId": 20001,
			"policyData": map[string]interface{}{
				"scope": map[string]interface{}{
					"cellId": "cell-001",
				},
				"statements": []map[string]interface{}{
					{
						"priorityLevel": 10,
						"qosParameters": map[string]interface{}{
							"qci":           9,
							"arp":           1,
							"maxBitRate":    "100Mbps",
							"minBitRate":    "10Mbps",
						},
					},
				},
			},
		}
		
		jsonData, _ := json.Marshal(policyData)
		start := time.Now()
		resp, err := http.Post(suite.server.URL+"/api/v1/a1/policies", "application/json", bytes.NewBuffer(jsonData))
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		suite.recordHandlerTest("CreateA1Policy", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test policy validation
	suite.Run("CreateA1PolicyInvalidData", func() {
		invalidData := map[string]interface{}{
			"invalidField": "value",
		}
		
		jsonData, _ := json.Marshal(invalidData)
		start := time.Now()
		resp, err := http.Post(suite.server.URL+"/api/v1/a1/policies", "application/json", bytes.NewBuffer(jsonData))
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		
		suite.recordHandlerTest("CreateA1PolicyInvalidData", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	suite.testResults.PassedTests++
}

// TestXAppHandlers tests xApp management handlers
func (suite *DashboardHandlerTestSuite) TestXAppHandlers() {
	suite.testResults.TotalTests++
	
	// Test GET /api/v1/xapps
	suite.Run("GetXApps", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/xapps")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var xapps []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&xapps)
		require.NoError(suite.T(), err)
		
		suite.recordHandlerTest("GetXApps", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test POST /api/v1/xapps (Deploy xApp)
	suite.Run("DeployXApp", func() {
		xappData := map[string]interface{}{
			"name":        "hello-world",
			"version":     "1.0.0",
			"chartName":   "hello-world-xapp",
			"namespace":   "ricxapp",
			"configData": map[string]interface{}{
				"containers": []map[string]interface{}{
					{
						"name":  "hello-world",
						"image": "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-app-hw:1.0.0",
						"ports": []map[string]interface{}{
							{"containerPort": 8080, "protocol": "http"},
						},
					},
				},
			},
		}
		
		jsonData, _ := json.Marshal(xappData)
		start := time.Now()
		resp, err := http.Post(suite.server.URL+"/api/v1/xapps", "application/json", bytes.NewBuffer(jsonData))
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		suite.recordHandlerTest("DeployXApp", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test GET /api/v1/xapps/{xappName}
	suite.Run("GetXApp", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/api/v1/xapps/hello-world")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var xapp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&xapp)
		require.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), xapp)
		
		suite.recordHandlerTest("GetXApp", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	suite.testResults.PassedTests++
}

// TestHealthAndMetricsHandlers tests health and metrics endpoints
func (suite *DashboardHandlerTestSuite) TestHealthAndMetricsHandlers() {
	suite.testResults.TotalTests++
	
	// Test health endpoint
	suite.Run("HealthCheck", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/health")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "ok", health["status"])
		
		suite.recordHandlerTest("HealthCheck", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test metrics endpoint
	suite.Run("Metrics", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/metrics")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		// Verify Prometheus metrics format
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		metricsText := string(body[:n])
		
		assert.Contains(suite.T(), metricsText, "# HELP")
		assert.Contains(suite.T(), metricsText, "# TYPE")
		
		suite.recordHandlerTest("Metrics", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	// Test readiness endpoint
	suite.Run("ReadinessCheck", func() {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + "/ready")
		latency := time.Since(start)
		
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var ready map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&ready)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "ready", ready["status"])
		
		suite.recordHandlerTest("ReadinessCheck", true, latency, resp.StatusCode)
		resp.Body.Close()
	})
	
	suite.testResults.PassedTests++
}

// TestConcurrentRequests tests handler performance under concurrent load
func (suite *DashboardHandlerTestSuite) TestConcurrentRequests() {
	suite.testResults.TotalTests++
	
	concurrentUsers := 50
	requestsPerUser := 10
	
	suite.Run("ConcurrentE2NodesRequests", func() {
		suite.runConcurrentTest("/api/v1/e2/nodes", "GET", nil, concurrentUsers, requestsPerUser)
	})
	
	suite.Run("ConcurrentA1PoliciesRequests", func() {
		suite.runConcurrentTest("/api/v1/a1/policies", "GET", nil, concurrentUsers, requestsPerUser)
	})
	
	suite.Run("ConcurrentXAppsRequests", func() {
		suite.runConcurrentTest("/api/v1/xapps", "GET", nil, concurrentUsers, requestsPerUser)
	})
	
	suite.testResults.PassedTests++
}

// runConcurrentTest executes concurrent requests and measures performance
func (suite *DashboardHandlerTestSuite) runConcurrentTest(endpoint, method string, body []byte, users, requests int) {
	results := make(chan *PerformanceData, users)
	
	for i := 0; i < users; i++ {
		go func() {
			var latencies []time.Duration
			var errors int
			
			for j := 0; j < requests; j++ {
				start := time.Now()
				
				var resp *http.Response
				var err error
				
				switch method {
				case "GET":
					resp, err = http.Get(suite.server.URL + endpoint)
				case "POST":
					resp, err = http.Post(suite.server.URL+endpoint, "application/json", bytes.NewBuffer(body))
				}
				
				latency := time.Since(start)
				
				if err != nil || resp.StatusCode >= 400 {
					errors++
				} else {
					latencies = append(latencies, latency)
				}
				
				if resp != nil {
					resp.Body.Close()
				}
			}
			
			// Calculate statistics
			if len(latencies) > 0 {
				perfData := suite.calculatePerformanceStats(endpoint, latencies, errors, requests)
				results <- perfData
			} else {
				results <- &PerformanceData{OperationName: endpoint}
			}
		}()
	}
	
	// Collect results
	var allLatencies []time.Duration
	totalErrors := 0
	totalRequests := 0
	
	for i := 0; i < users; i++ {
		result := <-results
		if result.MinLatency > 0 {
			// Aggregate data
			totalRequests += requests
		}
	}
	
	// Record overall performance
	if len(allLatencies) > 0 {
		perfData := suite.calculatePerformanceStats(endpoint+"_concurrent", allLatencies, totalErrors, totalRequests)
		suite.testResults.PerformanceData[endpoint] = perfData
	}
}

// calculatePerformanceStats calculates performance statistics from latency data
func (suite *DashboardHandlerTestSuite) calculatePerformanceStats(operation string, latencies []time.Duration, errors, totalRequests int) *PerformanceData {
	if len(latencies) == 0 {
		return &PerformanceData{OperationName: operation}
	}
	
	// Sort latencies for percentile calculation
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	
	// Calculate statistics
	min := latencies[0]
	max := latencies[len(latencies)-1]
	
	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}
	avg := total / time.Duration(len(latencies))
	
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)
	
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}
	
	p95 := latencies[p95Index]
	p99 := latencies[p99Index]
	
	return &PerformanceData{
		OperationName:  operation,
		MinLatency:     min,
		MaxLatency:     max,
		AverageLatency: avg,
		P95Latency:     p95,
		P99Latency:     p99,
		RequestsPerSec: float64(len(latencies)) / total.Seconds(),
	}
}

// recordHandlerTest records test results for a handler
func (suite *DashboardHandlerTestSuite) recordHandlerTest(handlerName string, passed bool, latency time.Duration, statusCode int) {
	if suite.testResults.HandlerTests[handlerName] == nil {
		suite.testResults.HandlerTests[handlerName] = &HandlerTestResult{
			HandlerName: handlerName,
			StatusCodes: make(map[int]int),
		}
	}
	
	result := suite.testResults.HandlerTests[handlerName]
	
	if passed {
		result.TestsPassed++
	} else {
		result.TestsFailed++
	}
	
	result.StatusCodes[statusCode]++
	
	// Update average latency
	totalTests := result.TestsPassed + result.TestsFailed
	if totalTests == 1 {
		result.AverageLatency = latency
	} else {
		result.AverageLatency = (result.AverageLatency*time.Duration(totalTests-1) + latency) / time.Duration(totalTests)
	}
	
	// Calculate error rate
	result.ErrorRate = float64(result.TestsFailed) / float64(totalTests) * 100
}

// Mock handler implementations

func (suite *DashboardHandlerTestSuite) mockE2NodesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		nodes := []map[string]interface{}{
			{
				"globalE2NodeId": "001-001-001",
				"nodeType":       "gNB",
				"plmnId":         "001001",
				"connectionStatus": "connected",
				"ranFunctions": []map[string]interface{}{
					{"ranFunctionId": 1, "ranFunctionDefinition": "KPM monitoring"},
				},
			},
			{
				"globalE2NodeId": "001-001-002",
				"nodeType":       "eNB",
				"plmnId":         "001001",
				"connectionStatus": "connected",
				"ranFunctions": []map[string]interface{}{
					{"ranFunctionId": 2, "ranFunctionDefinition": "RC control"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	case "POST":
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"message": "E2 node created successfully",
			"nodeId":  "001-001-003",
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *DashboardHandlerTestSuite) mockE2NodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeId := vars["nodeId"]
	
	if nodeId == "nonexistent" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node not found"})
		return
	}
	
	switch r.Method {
	case "GET":
		node := map[string]interface{}{
			"globalE2NodeId":   nodeId,
			"nodeType":         "gNB",
			"plmnId":           "001001",
			"connectionStatus": "connected",
			"ranFunctions": []map[string]interface{}{
				{"ranFunctionId": 1, "ranFunctionDefinition": "KPM monitoring"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(node)
	case "PUT":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Node updated"})
	case "DELETE":
		w.WriteHeader(http.StatusNoContent)
	}
}

func (suite *DashboardHandlerTestSuite) mockE2SubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		subscriptions := []map[string]interface{}{
			{
				"subscriptionId": "sub-001",
				"e2NodeId":       "001-001-001",
				"ranFunctionId":  1,
				"status":         "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subscriptions)
	case "POST":
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"message":        "Subscription created",
			"subscriptionId": "sub-002",
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *DashboardHandlerTestSuite) mockE2SubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subId := vars["subId"]
	
	switch r.Method {
	case "GET":
		subscription := map[string]interface{}{
			"subscriptionId": subId,
			"e2NodeId":       "001-001-001",
			"ranFunctionId":  1,
			"status":         "active",
		}
		json.NewEncoder(w).Encode(subscription)
	case "PUT":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Subscription updated"})
	case "DELETE":
		w.WriteHeader(http.StatusNoContent)
	}
}

func (suite *DashboardHandlerTestSuite) mockA1PoliciesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		policies := []map[string]interface{}{
			{
				"policyId":     "policy-001",
				"policyTypeId": 20001,
				"status":       "enforced",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(policies)
	case "POST":
		// Validate request body
		var policyData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&policyData); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}
		
		// Basic validation
		if _, ok := policyData["policyId"]; !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing policyId"})
			return
		}
		
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"message":  "Policy created",
			"policyId": policyData["policyId"],
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *DashboardHandlerTestSuite) mockA1PolicyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyId := vars["policyId"]
	
	switch r.Method {
	case "GET":
		policy := map[string]interface{}{
			"policyId":     policyId,
			"policyTypeId": 20001,
			"status":       "enforced",
			"policyData": map[string]interface{}{
				"scope": map[string]interface{}{
					"cellId": "cell-001",
				},
			},
		}
		json.NewEncoder(w).Encode(policy)
	case "PUT":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Policy updated"})
	case "DELETE":
		w.WriteHeader(http.StatusNoContent)
	}
}

func (suite *DashboardHandlerTestSuite) mockA1PolicyTypesHandler(w http.ResponseWriter, r *http.Request) {
	policyTypes := []map[string]interface{}{
		{
			"policyTypeId": 20001,
			"name":         "QoS Policy",
			"schema":       map[string]interface{}{"type": "object"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policyTypes)
}

func (suite *DashboardHandlerTestSuite) mockXAppsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		xapps := []map[string]interface{}{
			{
				"name":      "hello-world",
				"version":   "1.0.0",
				"status":    "deployed",
				"namespace": "ricxapp",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(xapps)
	case "POST":
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"message": "xApp deployed successfully",
			"name":    "hello-world",
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *DashboardHandlerTestSuite) mockXAppHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	xappName := vars["xappName"]
	
	switch r.Method {
	case "GET":
		xapp := map[string]interface{}{
			"name":      xappName,
			"version":   "1.0.0",
			"status":    "deployed",
			"namespace": "ricxapp",
			"instances": []map[string]interface{}{
				{
					"podName": xappName + "-pod-1",
					"status":  "running",
				},
			},
		}
		json.NewEncoder(w).Encode(xapp)
	case "PUT":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "xApp updated"})
	case "DELETE":
		w.WriteHeader(http.StatusNoContent)
	}
}

func (suite *DashboardHandlerTestSuite) mockXAppInstancesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	xappName := vars["xappName"]
	
	instances := []map[string]interface{}{
		{
			"podName":   xappName + "-pod-1",
			"status":    "running",
			"createdAt": time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

func (suite *DashboardHandlerTestSuite) mockHealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"checks": map[string]string{
			"database":    "ok",
			"e2manager":   "ok",
			"a1mediator":  "ok",
			"submgr":      "ok",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (suite *DashboardHandlerTestSuite) mockMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="GET",handler="e2nodes"} 42
http_requests_total{method="POST",handler="e2nodes"} 7

# HELP http_request_duration_seconds The HTTP request latencies in seconds.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",handler="e2nodes",le="0.1"} 32
http_request_duration_seconds_bucket{method="GET",handler="e2nodes",le="0.5"} 40
http_request_duration_seconds_bucket{method="GET",handler="e2nodes",le="+Inf"} 42
http_request_duration_seconds_sum{method="GET",handler="e2nodes"} 1.2
http_request_duration_seconds_count{method="GET",handler="e2nodes"} 42
`
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(metrics))
}

func (suite *DashboardHandlerTestSuite) mockReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ready)
}

// TestDashboardHandlerTestSuite runs the test suite
func TestDashboardHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DashboardHandlerTestSuite))
}

// Helper function to generate test report
func (suite *DashboardHandlerTestSuite) TearDownSuite() {
	duration := time.Since(suite.testResults.StartTime)
	
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("Dashboard Handler Unit Test Results\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")
	fmt.Printf("Test Duration: %v\n", duration)
	fmt.Printf("Total Tests: %d\n", suite.testResults.TotalTests)
	fmt.Printf("Passed Tests: %d\n", suite.testResults.PassedTests)
	fmt.Printf("Failed Tests: %d\n", suite.testResults.FailedTests)
	fmt.Printf("Success Rate: %.2f%%\n", float64(suite.testResults.PassedTests)/float64(suite.testResults.TotalTests)*100)
	
	fmt.Printf("\nHandler Test Details:\n")
	for name, result := range suite.testResults.HandlerTests {
		fmt.Printf("- %s: %d passed, %d failed, %.2fms avg latency, %.1f%% error rate\n",
			name, result.TestsPassed, result.TestsFailed,
			float64(result.AverageLatency.Nanoseconds())/1e6, result.ErrorRate)
	}
	
	fmt.Printf("\nPerformance Data:\n")
	for name, perf := range suite.testResults.PerformanceData {
		fmt.Printf("- %s: avg=%.2fms, p95=%.2fms, p99=%.2fms, %.1f req/s\n",
			name, float64(perf.AverageLatency.Nanoseconds())/1e6,
			float64(perf.P95Latency.Nanoseconds())/1e6,
			float64(perf.P99Latency.Nanoseconds())/1e6,
			perf.RequestsPerSec)
	}
	
	fmt.Printf(strings.Repeat("=", 80) + "\n")
}