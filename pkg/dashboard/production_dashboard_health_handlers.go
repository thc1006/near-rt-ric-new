package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// NewProductionHealthChecker initializes health checker
func NewProductionHealthChecker() *ProductionHealthChecker {
	return &ProductionHealthChecker{
		components: map[string]RICComponentHealth{
			"e2mgr": {
				ComponentName: "E2 Manager",
				Endpoint:      "ricplt-e2mgr:3800",
			},
			"a1mediator": {
				ComponentName: "A1 Mediator", 
				Endpoint:      "ricplt-a1mediator:10000",
			},
			"appmgr": {
				ComponentName: "App Manager",
				Endpoint:      "ricplt-appmgr:8080",
			},
			"submgr": {
				ComponentName: "Subscription Manager",
				Endpoint:      "ricplt-submgr:8080",
			},
			"rtmgr": {
				ComponentName: "RT Manager",
				Endpoint:      "ricplt-rtmgr:4560",
			},
		},
	}
}

// Check health of a specific RIC component
func (hc *ProductionHealthChecker) checkComponentHealth(componentName string) RICComponentHealth {
	component := hc.components[componentName]
	
	// Perform health check
	url := fmt.Sprintf("http://%s/health", component.Endpoint)
	client := &http.Client{Timeout: time.Second * 5}
	
	resp, err := client.Get(url)
	if err != nil {
		component.Status = "DOWN"
		component.Error = err.Error()
	} else {
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			component.Status = "UP"
		} else {
			component.Status = "DEGRADED"
			component.Error = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
		}
	}
	
	component.LastChecked = time.Now()
	return component
}

// Overall dashboard readiness handler
func (api *ProductionDashboardAPI) readinessHandler(w http.ResponseWriter, r *http.Request) {
	// Validate basic readiness
	running := atomic.LoadInt32(&api.running)
	if running == 0 {
		api.writeErrorResponse(w, http.StatusServiceUnavailable, "Dashboard not ready")
		return
	}

	// Check RIC component health
	healthChecker := NewProductionHealthChecker()
	componentStatuses := make(map[string]RICComponentHealth)
	
	requiredComponents := []string{
		"e2mgr", "a1mediator", "appmgr", "submgr", "rtmgr",
	}
	
	overallStatus := "READY"
	for _, comp := range requiredComponents {
		status := healthChecker.checkComponentHealth(comp)
		componentStatuses[comp] = status
		
		// If any component is down, mark overall status as not ready
		if status.Status != "UP" {
			overallStatus = "NOT_READY"
		}
	}

	// Prepare readiness response
	readinessResponse := map[string]interface{}{
		"status":     overallStatus,
		"components": componentStatuses,
		"timestamp":  time.Now(),
	}

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus != "READY" {
		statusCode = http.StatusServiceUnavailable
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(readinessResponse)
}

// Liveness probe handler
func (api *ProductionDashboardAPI) livenessHandler(w http.ResponseWriter, r *http.Request) {
	// Basic liveness check
	running := atomic.LoadInt32(&api.running)
	
	livenessResponse := map[string]interface{}{
		"status":    "ALIVE",
		"running":   running == 1,
		"timestamp": time.Now(),
	}

	if running == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(livenessResponse)
}