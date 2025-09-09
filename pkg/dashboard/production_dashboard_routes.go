package dashboard

import (
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// configureHealthRoutes sets up comprehensive health and monitoring endpoints
func (api *ProductionDashboardAPI) configureHealthRoutes(router *mux.Router) {
	// Standard health and readiness probes
	router.HandleFunc("/health", api.healthHandler).Methods("GET")
	router.HandleFunc("/ready", api.readinessHandler).Methods("GET")
	router.HandleFunc("/live", api.livenessHandler).Methods("GET")

	// Detailed health routes
	healthRouter := router.PathPrefix("/api/health").Subrouter()
	healthRouter.HandleFunc("/ready", api.readinessHandler).Methods("GET")
	healthRouter.HandleFunc("/alive", api.livenessHandler).Methods("GET")
	healthRouter.HandleFunc("/components", api.componentHealthHandler).Methods("GET")

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Optional: Performance profiling endpoints (for debugging)
	if api.config.EnableProfiling {
		router.HandleFunc("/debug/pprof/", pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}

// componentHealthHandler provides detailed status of individual RIC components
func (api *ProductionDashboardAPI) componentHealthHandler(w http.ResponseWriter, r *http.Request) {
	healthChecker := NewProductionHealthChecker()
	componentStatuses := make(map[string]RICComponentHealth)
	
	requiredComponents := []string{
		"e2mgr", "a1mediator", "appmgr", "submgr", "rtmgr",
	}
	
	for _, comp := range requiredComponents {
		status := healthChecker.checkComponentHealth(comp)
		componentStatuses[comp] = status
	}

	api.writeJSONResponse(w, http.StatusOK, componentStatuses)
}

// Middleware for additional health checks
func (api *ProductionDashboardAPI) healthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pre-processing health checks
		if api.checkSystemHealth() {
			next.ServeHTTP(w, r)
		} else {
			api.writeErrorResponse(w, http.StatusServiceUnavailable, "System health check failed")
		}
	})
}

// Internal system health check
func (api *ProductionDashboardAPI) checkSystemHealth() bool {
	// Perform basic system health checks
	resourceCheck := api.checkResourceAvailability()
	componentCheck := api.checkCriticalComponents()
	
	return resourceCheck && componentCheck
}

// Check resource availability
func (api *ProductionDashboardAPI) checkResourceAvailability() bool {
	// Check CPU, memory, and other critical resources
	// This would use runtime package or system calls
	return true // Placeholder
}

// Check critical system components
func (api *ProductionDashboardAPI) checkCriticalComponents() bool {
	// Verify critical dashboard components are operational
	return true // Placeholder
}