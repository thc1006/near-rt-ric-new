package dashboard

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsManager handles Prometheus metrics collection and exposure
type MetricsManager struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// E2 Interface metrics
	e2NodesConnected        prometheus.Gauge
	e2SubscriptionsActive   prometheus.Gauge
	e2MessagesTotal         *prometheus.CounterVec
	e2IndicationProcessing  *prometheus.HistogramVec
	e2SubscriptionRequests  *prometheus.CounterVec
	e2SetupRequests         *prometheus.CounterVec

	// A1 Interface metrics
	a1PolicyTypesTotal      prometheus.Gauge
	a1PolicyInstancesTotal  prometheus.Gauge
	a1PolicyRequests        *prometheus.CounterVec
	a1PolicyProcessing      *prometheus.HistogramVec

	// O1 Interface metrics
	o1ConfigOperations      *prometheus.CounterVec
	o1AlarmEvents           *prometheus.CounterVec
	o1NetconfSessions       prometheus.Gauge
	o1OperationProcessing   *prometheus.HistogramVec

	// Subscription metrics
	subscriptionOperations  *prometheus.CounterVec
	subscriptionProcessing  *prometheus.HistogramVec

	// System metrics
	componentHealth         *prometheus.GaugeVec
	memoryUsage            *prometheus.GaugeVec
	cpuUsage               *prometheus.GaugeVec
	goroutinesCount        prometheus.Gauge

	// Security metrics
	authenticationAttempts  *prometheus.CounterVec
	authorizationChecks     *prometheus.CounterVec
	securityEvents          *prometheus.CounterVec

	registry *prometheus.Registry
}

// NewMetricsManager creates a new metrics manager
func NewMetricsManager() *MetricsManager {
	registry := prometheus.NewRegistry()

	mm := &MetricsManager{
		registry: registry,

		// HTTP metrics
		httpRequestsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		httpRequestsInFlight: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		// E2 Interface metrics
		e2NodesConnected: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "e2_nodes_connected",
				Help: "Number of connected E2 nodes",
			},
		),
		e2SubscriptionsActive: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "e2_subscriptions_active",
				Help: "Number of active E2 subscriptions",
			},
		),
		e2MessagesTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "e2ap_messages_total",
				Help: "Total number of E2AP messages processed",
			},
			[]string{"message_type", "direction", "node_id"},
		),
		e2IndicationProcessing: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "e2_indication_processing_duration_seconds",
				Help:    "Time spent processing E2 indications",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"node_id", "subscription_id"},
		),
		e2SubscriptionRequests: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "e2_subscription_requests_total",
				Help: "Total number of E2 subscription requests",
			},
			[]string{"status", "node_id"},
		),
		e2SetupRequests: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "e2_setup_requests_total",
				Help: "Total number of E2 setup requests",
			},
			[]string{"status", "node_id"},
		),

		// A1 Interface metrics
		a1PolicyTypesTotal: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "a1_policy_types_total",
				Help: "Total number of registered A1 policy types",
			},
		),
		a1PolicyInstancesTotal: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "a1_policy_instances_total",
				Help: "Total number of A1 policy instances",
			},
		),
		a1PolicyRequests: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "a1_policy_requests_total",
				Help: "Total number of A1 policy requests",
			},
			[]string{"operation", "policy_type_id", "status"},
		),
		a1PolicyProcessing: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "a1_policy_processing_duration_seconds",
				Help:    "Time spent processing A1 policy operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "policy_type_id"},
		),

		// O1 Interface metrics
		o1ConfigOperations: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_config_operations_total",
				Help: "Total number of O1 configuration operations",
			},
			[]string{"operation", "target", "status"},
		),
		o1AlarmEvents: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_alarm_events_total",
				Help: "Total number of O1 alarm events",
			},
			[]string{"severity", "source"},
		),
		o1NetconfSessions: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "o1_netconf_sessions_active",
				Help: "Number of active NETCONF sessions",
			},
		),
		o1OperationProcessing: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "o1_operation_processing_duration_seconds",
				Help:    "Time spent processing O1 operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "target"},
		),

		// Subscription metrics
		subscriptionOperations: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "subscription_operations_total",
				Help: "Total number of subscription operations",
			},
			[]string{"operation", "status"},
		),
		subscriptionProcessing: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "subscription_processing_duration_seconds",
				Help:    "Time spent processing subscription operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		// System metrics
		componentHealth: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "component_health",
				Help: "Health status of platform components (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),
		memoryUsage: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"component", "type"},
		),
		cpuUsage: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cpu_usage_percent",
				Help: "CPU usage percentage",
			},
			[]string{"component"},
		),
		goroutinesCount: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "goroutines_count",
				Help: "Number of goroutines",
			},
		),

		// Security metrics
		authenticationAttempts: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "authentication_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "status"},
		),
		authorizationChecks: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "authorization_checks_total",
				Help: "Total number of authorization checks",
			},
			[]string{"resource", "action", "status"},
		),
		securityEvents: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_events_total",
				Help: "Total number of security events",
			},
			[]string{"event_type", "severity"},
		),
	}

	// Register standard Go metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return mm
}

// HTTP Metrics Methods

// RecordHTTPRequest records HTTP request metrics
func (mm *MetricsManager) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	mm.httpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
	mm.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// IncHTTPRequestsInFlight increments in-flight HTTP requests
func (mm *MetricsManager) IncHTTPRequestsInFlight() {
	mm.httpRequestsInFlight.Inc()
}

// DecHTTPRequestsInFlight decrements in-flight HTTP requests
func (mm *MetricsManager) DecHTTPRequestsInFlight() {
	mm.httpRequestsInFlight.Dec()
}

// E2 Interface Metrics Methods

// SetE2NodesConnected sets the number of connected E2 nodes
func (mm *MetricsManager) SetE2NodesConnected(count int) {
	mm.e2NodesConnected.Set(float64(count))
}

// SetE2SubscriptionsActive sets the number of active E2 subscriptions
func (mm *MetricsManager) SetE2SubscriptionsActive(count int) {
	mm.e2SubscriptionsActive.Set(float64(count))
}

// RecordE2Message records E2AP message metrics
func (mm *MetricsManager) RecordE2Message(messageType, direction, nodeID string) {
	mm.e2MessagesTotal.WithLabelValues(messageType, direction, nodeID).Inc()
}

// RecordE2IndicationProcessing records E2 indication processing time
func (mm *MetricsManager) RecordE2IndicationProcessing(nodeID, subscriptionID string, duration time.Duration) {
	mm.e2IndicationProcessing.WithLabelValues(nodeID, subscriptionID).Observe(duration.Seconds())
}

// RecordE2SubscriptionRequest records E2 subscription request metrics
func (mm *MetricsManager) RecordE2SubscriptionRequest(status, nodeID string) {
	mm.e2SubscriptionRequests.WithLabelValues(status, nodeID).Inc()
}

// RecordE2SetupRequest records E2 setup request metrics
func (mm *MetricsManager) RecordE2SetupRequest(status, nodeID string) {
	mm.e2SetupRequests.WithLabelValues(status, nodeID).Inc()
}

// A1 Interface Metrics Methods

// SetA1PolicyTypesTotal sets the total number of A1 policy types
func (mm *MetricsManager) SetA1PolicyTypesTotal(count int) {
	mm.a1PolicyTypesTotal.Set(float64(count))
}

// SetA1PolicyInstancesTotal sets the total number of A1 policy instances
func (mm *MetricsManager) SetA1PolicyInstancesTotal(count int) {
	mm.a1PolicyInstancesTotal.Set(float64(count))
}

// RecordA1PolicyRequest records A1 policy request metrics
func (mm *MetricsManager) RecordA1PolicyRequest(operation, policyTypeID, status string) {
	mm.a1PolicyRequests.WithLabelValues(operation, policyTypeID, status).Inc()
}

// RecordA1PolicyProcessing records A1 policy processing time
func (mm *MetricsManager) RecordA1PolicyProcessing(operation, policyTypeID string, duration time.Duration) {
	mm.a1PolicyProcessing.WithLabelValues(operation, policyTypeID).Observe(duration.Seconds())
}

// O1 Interface Metrics Methods

// RecordO1ConfigOperation records O1 configuration operation metrics
func (mm *MetricsManager) RecordO1ConfigOperation(operation, target, status string) {
	mm.o1ConfigOperations.WithLabelValues(operation, target, status).Inc()
}

// RecordO1AlarmEvent records O1 alarm event metrics
func (mm *MetricsManager) RecordO1AlarmEvent(severity, source string) {
	mm.o1AlarmEvents.WithLabelValues(severity, source).Inc()
}

// SetO1NetconfSessions sets the number of active NETCONF sessions
func (mm *MetricsManager) SetO1NetconfSessions(count int) {
	mm.o1NetconfSessions.Set(float64(count))
}

// RecordO1OperationProcessing records O1 operation processing time
func (mm *MetricsManager) RecordO1OperationProcessing(operation, target string, duration time.Duration) {
	mm.o1OperationProcessing.WithLabelValues(operation, target).Observe(duration.Seconds())
}

// Subscription Metrics Methods

// RecordSubscriptionOperation records subscription operation metrics
func (mm *MetricsManager) RecordSubscriptionOperation(operation, status string) {
	mm.subscriptionOperations.WithLabelValues(operation, status).Inc()
}

// RecordSubscriptionProcessing records subscription processing time
func (mm *MetricsManager) RecordSubscriptionProcessing(operation string, duration time.Duration) {
	mm.subscriptionProcessing.WithLabelValues(operation).Observe(duration.Seconds())
}

// System Metrics Methods

// SetComponentHealth sets the health status of a component
func (mm *MetricsManager) SetComponentHealth(component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	mm.componentHealth.WithLabelValues(component).Set(value)
}

// SetMemoryUsage sets memory usage metrics
func (mm *MetricsManager) SetMemoryUsage(component, memType string, bytes int64) {
	mm.memoryUsage.WithLabelValues(component, memType).Set(float64(bytes))
}

// SetCPUUsage sets CPU usage metrics
func (mm *MetricsManager) SetCPUUsage(component string, percent float64) {
	mm.cpuUsage.WithLabelValues(component).Set(percent)
}

// SetGoroutinesCount sets the number of goroutines
func (mm *MetricsManager) SetGoroutinesCount(count int) {
	mm.goroutinesCount.Set(float64(count))
}

// Security Metrics Methods

// RecordAuthenticationAttempt records authentication attempt metrics
func (mm *MetricsManager) RecordAuthenticationAttempt(method, status string) {
	mm.authenticationAttempts.WithLabelValues(method, status).Inc()
}

// RecordAuthorizationCheck records authorization check metrics
func (mm *MetricsManager) RecordAuthorizationCheck(resource, action, status string) {
	mm.authorizationChecks.WithLabelValues(resource, action, status).Inc()
}

// RecordSecurityEvent records security event metrics
func (mm *MetricsManager) RecordSecurityEvent(eventType, severity string) {
	mm.securityEvents.WithLabelValues(eventType, severity).Inc()
}

// HTTP Handler Methods

// Handler returns the Prometheus metrics HTTP handler
func (mm *MetricsManager) Handler() http.Handler {
	return promhttp.HandlerFor(mm.registry, promhttp.HandlerOpts{})
}

// HTTPMiddleware creates HTTP middleware for metrics collection
func (mm *MetricsManager) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mm.IncHTTPRequestsInFlight()
		defer mm.DecHTTPRequestsInFlight()

		// Wrap response writer to capture status code
		wrapped := &ResponseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		mm.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter is now using the ResponseWriterWrapper from types.go to avoid redeclaration

// WriteHeader method is now defined in types.go with ResponseWriterWrapper

// Global metrics manager instance
var GlobalMetrics *MetricsManager

// InitializeMetrics initializes global metrics
func InitializeMetrics() {
	GlobalMetrics = NewMetricsManager()
}

// GetMetricsHandler returns the global metrics HTTP handler
func GetMetricsHandler() http.Handler {
	if GlobalMetrics == nil {
		InitializeMetrics()
	}
	return GlobalMetrics.Handler()
}