package dashboard

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Enhanced health check with RIC component connectivity verification
func (api *ProductionDashboardAPI) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check overall system health
	healthStatus := map[string]interface{}{
		"status":     "healthy",
		"components": make(map[string]bool),
	}

	// List of RIC components to check
	components := []struct {
		name    string
		checkFn func() bool
	}{
		{"e2mgr", api.checkE2MgrHealth},
		{"a1mediator", api.checkA1MediatorHealth},
		{"appmgr", api.checkAppMgrHealth},
		{"submgr", api.checkSubMgrHealth},
		{"rtmgr", api.checkRtMgrHealth},
	}

	// Check each component
	overallHealth := true
	for _, comp := range components {
		healthy := comp.checkFn()
		healthStatus["components"].(map[string]bool)[comp.name] = healthy
		if !healthy {
			overallHealth = false
		}
	}

	// Set overall status
	if !overallHealth {
		healthStatus["status"] = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Write JSON response
	api.writeJSONResponse(w, http.StatusOK, healthStatus)
}

// Component health check methods
func (api *ProductionDashboardAPI) checkE2MgrHealth() bool {
	return api.checkRICComponentHealth("ricplt-e2mgr", 3800)
}

func (api *ProductionDashboardAPI) checkA1MediatorHealth() bool {
	return api.checkRICComponentHealth("ricplt-a1mediator", 10000)
}

func (api *ProductionDashboardAPI) checkAppMgrHealth() bool {
	return api.checkRICComponentHealth("ricplt-appmgr", 8080)
}

func (api *ProductionDashboardAPI) checkSubMgrHealth() bool {
	return api.checkRICComponentHealth("ricplt-submgr", 8080)
}

func (api *ProductionDashboardAPI) checkRtMgrHealth() bool {
	return api.checkRICComponentHealth("ricplt-rtmgr", 4560)
}

// Generic RIC component health check
func (api *ProductionDashboardAPI) checkRICComponentHealth(serviceName string, port int) bool {
	// Use internal network service discovery
	url := fmt.Sprintf("http://%s:%d/health", serviceName, port)

	// Create a client with a short timeout
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	// Perform health check
	resp, err := client.Get(url)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service": serviceName,
			"error":   err,
		}).Warn("RIC component health check failed")
		return false
	}
	defer resp.Body.Close()

	// Consider 200 OK as healthy
	return resp.StatusCode == http.StatusOK
}