// Package dashboard provides types and interfaces for the O-RAN Near-RT RIC dashboard
package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusUnhealthy
	HealthStatusDegraded
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusUnhealthy:
		return "unhealthy"
	case HealthStatusDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// E2Node represents an E2 node in the RIC platform
type E2Node struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	Address          string              `json:"address"`
	Port             int                 `json:"port"`
	Status           E2NodeStatus        `json:"status"`
	PLMNIDs          []string            `json:"plmnIds"`
	GlobalE2NodeID   GlobalE2NodeID      `json:"globalE2NodeId"`
	SupportedRANFunctions []RANFunction   `json:"supportedRANFunctions"`
	Timestamp        time.Time           `json:"timestamp"`
	ConnectionInfo   *ConnectionInfo     `json:"connectionInfo,omitempty"`
	HealthStatus     HealthStatus        `json:"healthStatus"`
	Metrics          map[string]float64  `json:"metrics,omitempty"`
}

// E2NodeStatus represents the status of an E2 node
type E2NodeStatus int

const (
	E2NodeStatusUnknown E2NodeStatus = iota
	E2NodeStatusConnected
	E2NodeStatusDisconnected
	E2NodeStatusConnecting
)

// GlobalE2NodeID represents the global E2 node identifier
type GlobalE2NodeID struct {
	PLMNID       string `json:"plmnId"`
	NodeID       string `json:"nodeId"`
	NodeType     string `json:"nodeType"`
	CUDUFunction string `json:"cuduFunction,omitempty"`
}

// RANFunction represents a RAN function supported by an E2 node
type RANFunction struct {
	FunctionID       int    `json:"functionId"`
	ShortName        string `json:"shortName"`
	ServiceModelOID  string `json:"serviceModelOID"`
	Description      string `json:"description"`
	Revision         int    `json:"revision"`
}

// ConnectionInfo represents connection information for an E2 node
type ConnectionInfo struct {
	SCTPParams  SCTPParams  `json:"sctpParams"`
	EstablishedAt time.Time `json:"establishedAt"`
	LastActivity  time.Time `json:"lastActivity"`
}

// SCTPParams represents SCTP connection parameters
type SCTPParams struct {
	Port           int           `json:"port"`
	PPid           int           `json:"ppid"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"maxRetries"`
	HeartbeatDelay time.Duration `json:"heartbeatDelay"`
}

// RICRequest represents a RIC request message
type RICRequest struct {
	RequestID    RequestID    `json:"requestId"`
	RANFunctionID int          `json:"ranFunctionId"`
	CallProcessID string       `json:"callProcessId"`
	Payload      []byte       `json:"payload"`
	Timestamp    time.Time    `json:"timestamp"`
}

// RequestID represents a request identifier
type RequestID struct {
	RequestorID int `json:"requestorId"`
	InstanceID  int `json:"instanceId"`
}

// PolicyType represents a policy type in A1 interface
type PolicyType struct {
	PolicyTypeID PolicyTypeID               `json:"policy_type_id"`
	Name         string                     `json:"name"`
	Description  string                     `json:"description"`
	Schema       map[string]interface{}     `json:"policy_schema"`
	CreateSchema map[string]interface{}     `json:"create_schema,omitempty"`
}

// PolicyTypeID represents a policy type identifier
type PolicyTypeID string

// PolicyInstance represents a policy instance
type PolicyInstance struct {
	PolicyInstanceID PolicyInstanceID       `json:"policy_instance_id"`
	PolicyTypeID     PolicyTypeID           `json:"policy_type_id"`
	PolicyData       map[string]interface{} `json:"policy_data"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	LastModified     time.Time              `json:"last_modified"`
}

// PolicyInstanceID represents a policy instance identifier
type PolicyInstanceID string

// A1PolicyClient interface for A1 policy operations
type A1PolicyClient interface {
	GetPolicyTypes() ([]PolicyType, error)
	GetPolicyType(typeID PolicyTypeID) (*PolicyType, error)
	CreatePolicyType(policyType *PolicyType) error
	DeletePolicyType(typeID PolicyTypeID) error
	GetPolicyInstances(typeID PolicyTypeID) ([]PolicyInstance, error)
	GetPolicyInstance(typeID PolicyTypeID, instanceID PolicyInstanceID) (*PolicyInstance, error)
	CreatePolicyInstance(instance *PolicyInstance) error
	UpdatePolicyInstance(instance *PolicyInstance) error
	DeletePolicyInstance(typeID PolicyTypeID, instanceID PolicyInstanceID) error
	GetPolicyStatus(typeID PolicyTypeID, instanceID PolicyInstanceID) (string, error)
}

// PerformanceMetrics represents performance metrics for components
type PerformanceMetrics struct {
	ComponentID      string                 `json:"componentId"`
	Timestamp        time.Time              `json:"timestamp"`
	CPUUsage         float64                `json:"cpuUsage"`
	MemoryUsage      int64                  `json:"memoryUsage"`
	NetworkIn        int64                  `json:"networkIn"`
	NetworkOut       int64                  `json:"networkOut"`
	RequestsPerSec   float64                `json:"requestsPerSec"`
	ResponseTime     time.Duration          `json:"responseTime"`
	ErrorRate        float64                `json:"errorRate"`
	ThroughputMbps   float64                `json:"throughputMbps"`
	Latency          time.Duration          `json:"latency"`
	CustomMetrics    map[string]interface{} `json:"customMetrics,omitempty"`
}

// XAppManager interface for xApp management operations
type XAppManager interface {
	GetXApps() ([]XApp, error)
	GetXApp(id string) (*XApp, error)
	DeployXApp(config *XAppConfig) error
	UndeployXApp(id string) error
	GetXAppStatus(id string) (*XAppStatus, error)
	GetXAppLogs(id string, lines int) ([]string, error)
}

// XApp represents an xApp in the RIC platform
type XApp struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Status      XAppStatus             `json:"status"`
	Config      map[string]interface{} `json:"config"`
	Instances   []XAppInstance         `json:"instances"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// XAppInstance represents an instance of an xApp
type XAppInstance struct {
	ID        string                 `json:"id"`
	PodName   string                 `json:"podName"`
	Status    string                 `json:"status"`
	NodeName  string                 `json:"nodeName"`
	Resources map[string]interface{} `json:"resources"`
	Health    HealthStatus           `json:"health"`
}

// AlertManager interface for alert management
type AlertManager interface {
	CreateAlert(alert *Alert) error
	GetAlerts(filters map[string]string) ([]Alert, error)
	UpdateAlert(id string, updates map[string]interface{}) error
	DeleteAlert(id string) error
	SubscribeToAlerts(callback func(*Alert)) error
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                  `json:"acknowledged"`
	AckBy       string                 `json:"ackBy,omitempty"`
	AckAt       *time.Time             `json:"ackAt,omitempty"`
}

// HealthChecker represents a health check component
type HealthChecker struct {
	ComponentID   string                 `json:"componentId"`
	Status        HealthStatus           `json:"status"`
	LastCheck     time.Time              `json:"lastCheck"`
	CheckInterval time.Duration          `json:"checkInterval"`
	Metrics       map[string]interface{} `json:"metrics"`
	Dependencies  []string               `json:"dependencies"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ComponentName string                 `json:"componentName"`
	Status        HealthStatus           `json:"status"`
	Message       string                 `json:"message"`
	Error         error                  `json:"error,omitempty"`
	CheckTime     time.Time              `json:"checkTime"`
	Duration      time.Duration          `json:"duration"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogOptions represents options for log retrieval
type LogOptions struct {
	Lines     int       `json:"lines"`
	Since     time.Time `json:"since"`
	Follow    bool      `json:"follow"`
	Filter    string    `json:"filter"`
	Level     string    `json:"level"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID      string    `json:"userId"`
	Username    string    `json:"username"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expiresAt"`
	IssuedAt    time.Time `json:"issuedAt"`
}

// Dashboard represents the main dashboard interface
type Dashboard interface {
	GetOverview() (*DashboardOverview, error)
	GetE2Nodes() ([]E2Node, error)
	GetXApps() ([]XApp, error)
	GetAlerts() ([]Alert, error)
	GetMetrics(duration time.Duration) (*SystemMetrics, error)
	GetHealthStatus() (*SystemHealth, error)
}

// DashboardOverview represents dashboard overview information
type DashboardOverview struct {
	TotalE2Nodes     int                    `json:"totalE2Nodes"`
	ConnectedE2Nodes int                    `json:"connectedE2Nodes"`
	TotalXApps       int                    `json:"totalXApps"`
	RunningXApps     int                    `json:"runningXApps"`
	ActiveAlerts     int                    `json:"activeAlerts"`
	SystemHealth     HealthStatus           `json:"systemHealth"`
	LastUpdated      time.Time              `json:"lastUpdated"`
	Statistics       map[string]interface{} `json:"statistics,omitempty"`
}

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	Timestamp       time.Time              `json:"timestamp"`
	CPUUsage        float64                `json:"cpuUsage"`
	MemoryUsage     float64                `json:"memoryUsage"`
	DiskUsage       float64                `json:"diskUsage"`
	NetworkTraffic  NetworkTrafficMetrics  `json:"networkTraffic"`
	E2Traffic       E2TrafficMetrics       `json:"e2Traffic"`
	ComponentMetrics map[string]interface{} `json:"componentMetrics,omitempty"`
}

// NetworkTrafficMetrics represents network traffic metrics
type NetworkTrafficMetrics struct {
	InboundBps  float64 `json:"inboundBps"`
	OutboundBps float64 `json:"outboundBps"`
	PacketsIn   int64   `json:"packetsIn"`
	PacketsOut  int64   `json:"packetsOut"`
	ErrorsIn    int64   `json:"errorsIn"`
	ErrorsOut   int64   `json:"errorsOut"`
}

// E2TrafficMetrics represents E2 interface traffic metrics
type E2TrafficMetrics struct {
	MessagesIn        int64   `json:"messagesIn"`
	MessagesOut       int64   `json:"messagesOut"`
	SubscriptionsActive int   `json:"subscriptionsActive"`
	ControlRequests   int64   `json:"controlRequests"`
	IndicationReports int64   `json:"indicationReports"`
	AvgLatency        float64 `json:"avgLatency"`
	ErrorRate         float64 `json:"errorRate"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	OverallStatus    HealthStatus                `json:"overallStatus"`
	ComponentHealth  map[string]HealthStatus     `json:"componentHealth"`
	DependencyCheck  map[string]bool             `json:"dependencyCheck"`
	HealthScore      float64                     `json:"healthScore"`
	LastCheck        time.Time                   `json:"lastCheck"`
	Issues           []HealthIssue               `json:"issues,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Component   string                 `json:"component"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolution  string                 `json:"resolution,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// BackupManager interface for backup operations
type BackupManager interface {
	CreateBackup(components []string) (*BackupInfo, error)
	RestoreBackup(backupID string) error
	ListBackups() ([]BackupInfo, error)
	DeleteBackup(backupID string) error
	GetBackupStatus(backupID string) (*BackupStatus, error)
}

// BackupInfo represents backup information
type BackupInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Components  []string  `json:"components"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
	Status      string    `json:"status"`
	Description string    `json:"description,omitempty"`
}

// BackupStatus represents backup status
type BackupStatus struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// TelemetryManager interface for telemetry operations
type TelemetryManager interface {
	StartTelemetryCollection() error
	StopTelemetryCollection() error
	GetTelemetryData(query TelemetryQuery) (*TelemetryData, error)
	ConfigureTelemetry(config *TelemetryConfig) error
	GetTelemetryConfig() (*TelemetryConfig, error)
}

// TelemetryQuery represents a telemetry query
type TelemetryQuery struct {
	Metrics   []string  `json:"metrics"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Interval  string    `json:"interval"`
	Filters   map[string]string `json:"filters,omitempty"`
}

// TelemetryData represents telemetry data
type TelemetryData struct {
	Query     TelemetryQuery                    `json:"query"`
	Results   map[string][]TelemetryDataPoint   `json:"results"`
	Timestamp time.Time                         `json:"timestamp"`
	Metadata  map[string]interface{}            `json:"metadata,omitempty"`
}

// TelemetryDataPoint represents a single telemetry data point
type TelemetryDataPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// TelemetryConfig represents telemetry configuration
type TelemetryConfig struct {
	Enabled          bool                   `json:"enabled"`
	CollectionInterval time.Duration       `json:"collectionInterval"`
	RetentionPeriod    time.Duration       `json:"retentionPeriod"`
	Metrics          []string               `json:"metrics"`
	Exporters        []string               `json:"exporters"`
	Config           map[string]interface{} `json:"config,omitempty"`
}

// Missing types that are needed but not defined elsewhere

// QueueMetrics represents queue performance metrics
type QueueMetrics struct {
	QueueName     string    `json:"queueName"`
	Size          int       `json:"size"`
	EnqueueRate   float64   `json:"enqueueRate"`
	DequeueRate   float64   `json:"dequeueRate"`
	AvgWaitTime   float64   `json:"avgWaitTime"`
	MaxWaitTime   float64   `json:"maxWaitTime"`
	Timestamp     time.Time `json:"timestamp"`
}

// Subscription represents an E2 subscription
type Subscription struct {
	ID              string                 `json:"id"`
	E2NodeID        string                 `json:"e2NodeId"`
	RANFunctionID   int                    `json:"ranFunctionId"`
	EventTriggers   []EventTrigger         `json:"eventTriggers"`
	Actions         []SubscriptionAction   `json:"actions"`
	Status          SubscriptionStatus     `json:"status"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastActivity    time.Time              `json:"lastActivity"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionAction represents an action within a subscription
type SubscriptionAction struct {
	ActionID    int                    `json:"actionId"`
	ActionType  string                 `json:"actionType"`
	Definition  map[string]interface{} `json:"definition"`
	SubsequentActions []SubscriptionAction `json:"subsequentActions,omitempty"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus int

const (
	SubscriptionStatusPending SubscriptionStatus = iota
	SubscriptionStatusActive
	SubscriptionStatusInactive
	SubscriptionStatusError
)

func (s SubscriptionStatus) String() string {
	switch s {
	case SubscriptionStatusPending:
		return "pending"
	case SubscriptionStatusActive:
		return "active"
	case SubscriptionStatusInactive:
		return "inactive"
	case SubscriptionStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// A1MediatorClient interface for A1 Mediator operations  
type A1MediatorClient interface {
	GetHealth(ctx context.Context) (*A1Health, error)
	GetPolicyTypes(ctx context.Context) (*PolicyTypeListResponse, error)
	GetPolicyType(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyType, error)
	CreatePolicyType(ctx context.Context, policyTypeID PolicyTypeID, request *PolicyTypeRequest) error
	DeletePolicyType(ctx context.Context, policyTypeID PolicyTypeID) error
	GetPolicyInstances(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyInstanceListResponse, error)
	GetPolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyInstance, error)
	CreatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, request *PolicyInstanceRequest) error
	UpdatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, update *PolicyInstanceUpdate) error
	DeletePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) error
	GetPolicyInstanceStatus(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyStatus, error)
	GetStats(ctx context.Context) (*A1Stats, error)
	ValidatePolicy(ctx context.Context, policyTypeID PolicyTypeID, policy json.RawMessage) (*PolicyValidationResult, error)
	IsConnected() bool
}

// A1MediatorClientImpl represents the concrete implementation of A1 Mediator client
type A1MediatorClientImpl struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
	Timeout    time.Duration
	mu         sync.RWMutex
}

// NewA1MediatorClientImpl creates a new A1 Mediator client implementation
func NewA1MediatorClientImpl(baseURL string) *A1MediatorClientImpl {
	return &A1MediatorClientImpl{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
}

// SetAuthToken sets the authentication token
func (c *A1MediatorClientImpl) SetAuthToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AuthToken = token
}


// A1MediatorClientImpl methods are defined in a1_mediator_client.go

// AsyncPackageProcessor represents an asynchronous package processor
type AsyncPackageProcessor interface {
	ProcessPackage(ctx context.Context, pkg Package) error
	GetProcessingStatus(packageID string) (ProcessingStatus, error)
	CancelProcessing(packageID string) error
	ListProcessingJobs() ([]ProcessingJob, error)
}

// Package represents a deployable package
type Package struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Content     []byte                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// ProcessingStatus represents the status of package processing
type ProcessingStatus struct {
	PackageID   string                 `json:"packageId"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"`
	Message     string                 `json:"message"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ProcessingJob represents a processing job
type ProcessingJob struct {
	ID          string                 `json:"id"`
	PackageID   string                 `json:"packageId"`
	Status      ProcessingStatus       `json:"status"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// ResourcePoolManager manages resource allocation and pooling
type ResourcePoolManager interface {
	AllocateResources(request ResourceRequest) (*ResourceAllocation, error)
	ReleaseResources(allocationID string) error
	GetResourceStatus() (*ResourceStatus, error)
	UpdateResourceLimits(limits ResourceLimits) error
	GetResourceUsage() (*ResourceUsage, error)
}

// ResourceRequest represents a request for resources
type ResourceRequest struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	CPU         float64                `json:"cpu"`
	Memory      int64                  `json:"memory"`
	Storage     int64                  `json:"storage"`
	Duration    time.Duration          `json:"duration"`
	Priority    int                    `json:"priority"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// AllocatedResources represents the actual allocated resources
type AllocatedResources struct {
	CPU     float64 `json:"cpu"`
	Memory  int64   `json:"memory"`
	Storage int64   `json:"storage"`
	NodeID  string  `json:"nodeId"`
}

// ResourceStatus represents overall resource status
type ResourceStatus struct {
	TotalResources     AllocatedResources             `json:"totalResources"`
	AvailableResources AllocatedResources             `json:"availableResources"`
	AllocatedResources AllocatedResources             `json:"allocatedResources"`
	Utilization        map[string]float64             `json:"utilization"`
	Allocations        []ResourceAllocation           `json:"allocations"`
	LastUpdated        time.Time                      `json:"lastUpdated"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	MaxCPU     float64                `json:"maxCpu"`
	MaxMemory  int64                  `json:"maxMemory"`
	MaxStorage int64                  `json:"maxStorage"`
	Quotas     map[string]interface{} `json:"quotas,omitempty"`
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUUsage     float64                `json:"cpuUsage"`
	MemoryUsage  int64                  `json:"memoryUsage"`
	StorageUsage int64                  `json:"storageUsage"`
	UsageHistory []ResourceUsagePoint   `json:"usageHistory,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ResourceUsagePoint represents a single usage data point
type ResourceUsagePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	CPUUsage     float64   `json:"cpuUsage"`
	MemoryUsage  int64     `json:"memoryUsage"`
	StorageUsage int64     `json:"storageUsage"`
}

// PerformanceDegradation represents performance degradation metrics
type PerformanceDegradation struct {
	ComponentID      string                 `json:"componentId"`
	MetricName       string                 `json:"metricName"`
	BaselineValue    float64                `json:"baselineValue"`
	CurrentValue     float64                `json:"currentValue"`
	DegradationPct   float64                `json:"degradationPct"`
	Threshold        float64                `json:"threshold"`
	Severity         string                 `json:"severity"`
	DetectedAt       time.Time              `json:"detectedAt"`
	Duration         time.Duration          `json:"duration"`
	Cause            string                 `json:"cause,omitempty"`
	Recommendation   string                 `json:"recommendation,omitempty"`
	AffectedServices []string               `json:"affectedServices,omitempty"`
}

// ConnectionStability represents connection stability metrics
type ConnectionStability struct {
	NodeID              string    `json:"nodeId"`
	ConnectionType      string    `json:"connectionType"`
	EstablishedAt       time.Time `json:"establishedAt"`
	LastDisconnection   *time.Time `json:"lastDisconnection,omitempty"`
	DisconnectionCount  int       `json:"disconnectionCount"`
	UptimePercentage    float64   `json:"uptimePercentage"`
	AverageLatency      float64   `json:"averageLatency"`
	PacketLossRate      float64   `json:"packetLossRate"`
	JitterMs            float64   `json:"jitterMs"`
	QualityScore        float64   `json:"qualityScore"`
	StabilityTrend      string    `json:"stabilityTrend"`
}

// SystemLimits represents system resource and operational limits
type SystemLimits struct {
	MaxConnections      int                    `json:"maxConnections"`
	MaxConcurrentJobs   int                    `json:"maxConcurrentJobs"`
	MaxMemoryUsage      int64                  `json:"maxMemoryUsage"`
	MaxCPUUsage         float64                `json:"maxCpuUsage"`
	MaxStorageUsage     int64                  `json:"maxStorageUsage"`
	MaxThroughputMbps   float64                `json:"maxThroughputMbps"`
	MaxLatencyMs        float64                `json:"maxLatencyMs"`
	RateLimits          map[string]RateLimit   `json:"rateLimits"`
	ResourceQuotas      map[string]interface{} `json:"resourceQuotas"`
	OperationalLimits   map[string]interface{} `json:"operationalLimits"`
	EnforcementPolicy   string                 `json:"enforcementPolicy"`
	LastUpdated         time.Time              `json:"lastUpdated"`
}

// RateLimit represents a rate limiting configuration
type RateLimit struct {
	RequestsPerSecond int           `json:"requestsPerSecond"`
	BurstSize         int           `json:"burstSize"`
	WindowSize        time.Duration `json:"windowSize"`
	Enabled           bool          `json:"enabled"`
}

// Indication represents an E2AP indication message
type Indication struct {
	RequestID         RequestID              `json:"requestId"`
	RANFunctionID     int                    `json:"ranFunctionId"`
	ActionID          int                    `json:"actionId"`
	IndicationSN      int64                  `json:"indicationSN"`
	IndicationType    IndicationType         `json:"indicationType"`
	IndicationHeader  []byte                 `json:"indicationHeader"`
	IndicationMessage []byte                 `json:"indicationMessage"`
	CallProcessID     string                 `json:"callProcessId,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// IndicationType represents the type of indication
type IndicationType int

const (
	IndicationTypeReport IndicationType = iota
	IndicationTypeInsert
)

func (i IndicationType) String() string {
	switch i {
	case IndicationTypeReport:
		return "report"
	case IndicationTypeInsert:
		return "insert"
	default:
		return "unknown"
	}
}

// ServiceProfile represents a network slice service profile
type ServiceProfile struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	ServiceType     string            `json:"serviceType"`
	Requirements    map[string]string `json:"requirements"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// NetworkSlice represents a network slice
type NetworkSlice struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	ServiceProfile  ServiceProfile         `json:"serviceProfile"`
	Configuration   map[string]interface{} `json:"configuration"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// PackageTask represents a package management task
type PackageTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	PackageID   string                 `json:"packageId"`
	Action      string                 `json:"action"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"createdAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

// PolicyManagerClient interface for A1 policy management
type PolicyManagerClient interface {
	GetPolicyTypes() ([]PolicyType, error)
	GetPolicyInstances(typeID PolicyTypeID) ([]PolicyInstance, error)
	CreatePolicyInstance(instance PolicyInstance) error
	DeletePolicyInstance(typeID PolicyTypeID, instanceID PolicyInstanceID) error
}

// LoadBalancingAlgorithm represents load balancing strategies
type LoadBalancingAlgorithm int

// Backend represents a backend server for load balancing
type Backend struct {
	ID           string        `json:"id"`
	Address      string        `json:"address"`
	Port         int           `json:"port"`
	Weight       int           `json:"weight"`
	IsHealthy    int32         `json:"isHealthy"`    // atomic
	CurrentConns int64         `json:"currentConns"` // atomic
	CPUUsage     float64       `json:"cpuUsage"`
	MemoryUsage  float64       `json:"memoryUsage"`
	ResponseTime time.Duration `json:"responseTime"`
	LastCheck    time.Time     `json:"lastCheck"`
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreaker provides circuit breaker functionality
type CircuitBreaker struct {
	name                string
	state               CircuitState
	maxFailures         int
	timeout             time.Duration
	resetTimeout        time.Duration
	halfOpenMaxCalls    int
	
	// Metrics
	failureCount        int
	successCount        int
	totalCalls          int64
	lastFailureTime     time.Time
	lastSuccessTime     time.Time
	nextAttempt         time.Time
	
	mu                  sync.RWMutex
}

// ValidationResult represents test validation result
type ValidationResult struct {
	TestName       string    `json:"testName"`
	RequirementMet bool      `json:"requirementMet"`
	ActualValue    float64   `json:"actualValue"`
	RequiredValue  float64   `json:"requiredValue"`
	PerformanceGap float64   `json:"performanceGap"`
	Details        string    `json:"details"`
	Timestamp      time.Time `json:"timestamp"`
}

// LatencyMetrics represents latency measurement data
type LatencyMetrics struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// EventTrigger type is defined in subscription_models.go

// SMO/Nephio Integration Layer Types

// PackageManagerClient handles Nephio package management operations
type PackageManagerClient struct {
	endpoint       string                 `json:"endpoint"`
	httpClient     *http.Client           `json:"-"`
	requestTimeout time.Duration          `json:"requestTimeout"`
	retryAttempts  int                    `json:"retryAttempts"`
	cache          map[string]interface{} `json:"-"`
	mu             sync.RWMutex           `json:"-"`
}

// NewPackageManagerClient creates a new package manager client
func NewPackageManagerClient(endpoint string) *PackageManagerClient {
	return &PackageManagerClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		requestTimeout: 30 * time.Second,
		retryAttempts:  3,
		cache:          make(map[string]interface{}),
	}
}

// Connect establishes connection to package manager
func (c *PackageManagerClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Implementation would establish connection to Nephio package manager
	return nil
}

// ResourceProvisionerClient handles O-Cloud resource provisioning
type ResourceProvisionerClient struct {
	endpoint         string                 `json:"endpoint"`
	httpClient       *http.Client           `json:"-"`
	requestTimeout   time.Duration          `json:"requestTimeout"`
	resourcePools    map[string]interface{} `json:"-"`
	provisionedCount uint64                 `json:"provisionedCount"`
	mu               sync.RWMutex           `json:"-"`
}

// NewResourceProvisionerClient creates a new resource provisioner client
func NewResourceProvisionerClient(endpoint string) *ResourceProvisionerClient {
	return &ResourceProvisionerClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		requestTimeout: 30 * time.Second,
		resourcePools:  make(map[string]interface{}),
	}
}

// Connect establishes connection to resource provisioner
func (c *ResourceProvisionerClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Implementation would establish connection to O-Cloud resource provisioner
	return nil
}

// SMOLoadBalancer provides load balancing for SMO components
type SMOLoadBalancer struct {
	strategy         string                 `json:"strategy"`
	backends         []*Backend             `json:"backends"`
	currentIndex     int32                  `json:"currentIndex"`    // atomic
	requestCount     uint64                 `json:"requestCount"`    // atomic
	healthChecker    *BackendHealthChecker  `json:"-"`
	running          bool                   `json:"running"`
	mu               sync.RWMutex           `json:"-"`
}

// NewSMOLoadBalancer creates a new SMO load balancer
func NewSMOLoadBalancer(strategy string) *SMOLoadBalancer {
	return &SMOLoadBalancer{
		strategy:      strategy,
		backends:      make([]*Backend, 0),
		healthChecker: &BackendHealthChecker{backends: make(map[string]*Backend)},
	}
}

// Start starts the SMO load balancer
func (lb *SMOLoadBalancer) Start(ctx context.Context) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.running = true
	return nil
}

// OptimizeForLatency optimizes the load balancer for latency
func (lb *SMOLoadBalancer) OptimizeForLatency() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.strategy = "latency_based"
}

// NephioLoadBalancer provides load balancing for Nephio components
type NephioLoadBalancer struct {
	strategy         string                 `json:"strategy"`
	backends         []*Backend             `json:"backends"`
	currentIndex     int32                  `json:"currentIndex"`    // atomic
	requestCount     uint64                 `json:"requestCount"`    // atomic
	healthChecker    *BackendHealthChecker  `json:"-"`
	running          bool                   `json:"running"`
	mu               sync.RWMutex           `json:"-"`
}

// NewNephioLoadBalancer creates a new Nephio load balancer
func NewNephioLoadBalancer(strategy string) *NephioLoadBalancer {
	return &NephioLoadBalancer{
		strategy:      strategy,
		backends:      make([]*Backend, 0),
		healthChecker: &BackendHealthChecker{backends: make(map[string]*Backend)},
	}
}

// Start starts the Nephio load balancer
func (lb *NephioLoadBalancer) Start(ctx context.Context) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.running = true
	return nil
}

// IntegrationCache provides caching for integration layer requests
type IntegrationCache struct {
	maxSize      int                    `json:"maxSize"`
	ttl          time.Duration          `json:"ttl"`
	data         map[string]cacheEntry  `json:"-"`
	hitCount     uint64                 `json:"hitCount"`     // atomic
	missCount    uint64                 `json:"missCount"`    // atomic
	mu           sync.RWMutex           `json:"-"`
}

type cacheEntry struct {
	value     interface{}
	timestamp time.Time
}

// NewIntegrationCache creates a new integration cache
func NewIntegrationCache(maxSize int, ttl time.Duration) *IntegrationCache {
	return &IntegrationCache{
		maxSize: maxSize,
		ttl:     ttl,
		data:    make(map[string]cacheEntry),
	}
}

// GetPolicyResponse retrieves a policy response from cache
func (c *IntegrationCache) GetPolicyResponse(policyID string) *SMOPolicyResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if entry, exists := c.data[policyID]; exists {
		if time.Since(entry.timestamp) < c.ttl {
			atomic.AddUint64(&c.hitCount, 1)
			if response, ok := entry.value.(*SMOPolicyResponse); ok {
				return response
			}
		}
	}
	atomic.AddUint64(&c.missCount, 1)
	return nil
}

// SetPolicyResponse stores a policy response in cache
func (c *IntegrationCache) SetPolicyResponse(policyID string, response *SMOPolicyResponse, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[policyID] = cacheEntry{
		value:     response,
		timestamp: time.Now(),
	}
}

// GetHitRatio returns the cache hit ratio
func (c *IntegrationCache) GetHitRatio() float64 {
	hits := atomic.LoadUint64(&c.hitCount)
	misses := atomic.LoadUint64(&c.missCount)
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// OptimizeForHitRatio optimizes the cache for hit ratio
func (c *IntegrationCache) OptimizeForHitRatio() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Remove expired entries
	now := time.Now()
	for key, entry := range c.data {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.data, key)
		}
	}
}

// IntegrationCircuitBreaker provides circuit breaker functionality for integration layer
type IntegrationCircuitBreaker struct {
	services        map[string]*CircuitBreaker `json:"-"`
	tripCount       uint64                     `json:"tripCount"`       // atomic
	successCount    uint64                     `json:"successCount"`    // atomic
	failureCount    uint64                     `json:"failureCount"`    // atomic
	mu              sync.RWMutex               `json:"-"`
}

// NewIntegrationCircuitBreaker creates a new integration circuit breaker
func NewIntegrationCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *IntegrationCircuitBreaker {
	return &IntegrationCircuitBreaker{
		services: make(map[string]*CircuitBreaker),
	}
}

// IsOpen checks if circuit breaker is open for a service
func (cb *IntegrationCircuitBreaker) IsOpen(service string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if breaker, exists := cb.services[service]; exists {
		return breaker.state == StateOpen
	}
	return false
}

// RecordFailure records a failure for a service
func (cb *IntegrationCircuitBreaker) RecordFailure(service string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if breaker, exists := cb.services[service]; exists {
		breaker.failureCount++
		if breaker.failureCount >= breaker.maxFailures {
			breaker.state = StateOpen
			atomic.AddUint64(&cb.tripCount, 1)
		}
	}
	atomic.AddUint64(&cb.failureCount, 1)
}

// RecordSuccess records a success for a service
func (cb *IntegrationCircuitBreaker) RecordSuccess(service string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if breaker, exists := cb.services[service]; exists {
		breaker.successCount++
		breaker.failureCount = 0
		if breaker.state == StateHalfOpen {
			breaker.state = StateClosed
		}
	}
	atomic.AddUint64(&cb.successCount, 1)
}

// GetTripCount returns the total trip count
func (cb *IntegrationCircuitBreaker) GetTripCount() uint64 {
	return atomic.LoadUint64(&cb.tripCount)
}

// IntegrationMetrics collects metrics for the integration layer
type IntegrationMetrics struct {
	requestsTotal       uint64                 `json:"requestsTotal"`       // atomic
	requestsSuccessful  uint64                 `json:"requestsSuccessful"`  // atomic
	requestsFailed      uint64                 `json:"requestsFailed"`      // atomic
	averageLatencyMs    float64                `json:"averageLatencyMs"`
	throughputRPS       float64                `json:"throughputRPS"`
	errorRate           float64                `json:"errorRate"`
	customMetrics       map[string]interface{} `json:"customMetrics"`
	lastUpdated         time.Time              `json:"lastUpdated"`
	mu                  sync.RWMutex           `json:"-"`
}

// NewIntegrationMetrics creates a new integration metrics collector
func NewIntegrationMetrics() *IntegrationMetrics {
	return &IntegrationMetrics{
		customMetrics: make(map[string]interface{}),
		lastUpdated:   time.Now(),
	}
}

// IntegrationPerformanceTracker tracks performance metrics for the integration layer
type IntegrationPerformanceTracker struct {
	latencyHistory      []LatencyMetrics       `json:"latencyHistory"`
	throughputHistory   []ThroughputSample     `json:"throughputHistory"`
	resourceUsage       ResourceConsumption    `json:"resourceUsage"`
	performanceTrend    string                 `json:"performanceTrend"`
	alertsGenerated     uint64                 `json:"alertsGenerated"`     // atomic
	optimizationsApplied uint64                `json:"optimizationsApplied"` // atomic
	lastAnalysis        time.Time              `json:"lastAnalysis"`
	mu                  sync.RWMutex           `json:"-"`
}

// NewIntegrationPerformanceTracker creates a new performance tracker
func NewIntegrationPerformanceTracker() *IntegrationPerformanceTracker {
	return &IntegrationPerformanceTracker{
		latencyHistory:    make([]LatencyMetrics, 0),
		throughputHistory: make([]ThroughputSample, 0),
		lastAnalysis:      time.Now(),
	}
}

// IntegrationHealthMonitor monitors the health of integration components
type IntegrationHealthMonitor struct {
	config              *IntegrationConfig     `json:"-"`
	componentHealth     map[string]HealthStatus `json:"componentHealth"`
	healthChecks        map[string]time.Time   `json:"healthChecks"`
	unhealthyComponents []string               `json:"unhealthyComponents"`
	healthScore         float64                `json:"healthScore"`
	alertsActive        uint64                 `json:"alertsActive"`        // atomic
	checksPerformed     uint64                 `json:"checksPerformed"`     // atomic
	lastHealthCheck     time.Time              `json:"lastHealthCheck"`
	mu                  sync.RWMutex           `json:"-"`
}

// NewIntegrationHealthMonitor creates a new health monitor
func NewIntegrationHealthMonitor(config *IntegrationConfig) *IntegrationHealthMonitor {
	return &IntegrationHealthMonitor{
		config:              config,
		componentHealth:     make(map[string]HealthStatus),
		healthChecks:        make(map[string]time.Time),
		unhealthyComponents: make([]string, 0),
		lastHealthCheck:     time.Now(),
	}
}

// CapacityPlanner manages resource capacity planning for O-Cloud
type CapacityPlanner struct {
	resourcePools       map[string]ResourcePool   `json:"resourcePools"`
	utilizationHistory  []ResourceUtilization     `json:"utilizationHistory"`
	forecastModels      map[string]interface{}    `json:"-"`
	scalingPolicies     map[string]ScalingPolicy  `json:"scalingPolicies"`
	currentUtilization  float64                   `json:"currentUtilization"`
	predictedDemand     float64                   `json:"predictedDemand"`
	recommendedActions  []string                  `json:"recommendedActions"`
	lastPlanningCycle   time.Time                 `json:"lastPlanningCycle"`
	mu                  sync.RWMutex              `json:"-"`
}

// ResourcePool represents a pool of resources
type ResourcePool struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	TotalCapacity     AllocatedResources     `json:"totalCapacity"`
	AllocatedCapacity AllocatedResources     `json:"allocatedCapacity"`
	AvailableCapacity AllocatedResources     `json:"availableCapacity"`
	Utilization       float64                `json:"utilization"`
	Status            string                 `json:"status"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceUtilization represents resource utilization over time
type ResourceUtilization struct {
	Timestamp   time.Time          `json:"timestamp"`
	PoolID      string             `json:"poolId"`
	Utilization float64            `json:"utilization"`
	Demand      float64            `json:"demand"`
	Metrics     map[string]float64 `json:"metrics"`
}

// SMOPolicyResponse is already defined in smo_nephio_integration_layer.go

// RAppDeploymentResponse represents a response to rApp deployment
type RAppDeploymentResponse struct {
	RAppID           string                 `json:"rAppId"`
	DeploymentID     string                 `json:"deploymentId"`
	Status           string                 `json:"status"`
	DeployedClusters []string               `json:"deployedClusters"`
	ResourceUsage    ResourceConsumption    `json:"resourceUsage"`
	Timestamp        time.Time              `json:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// PackageRevisionResponse represents a response to package revision operations
type PackageRevisionResponse struct {
	PackageName      string                 `json:"packageName"`
	Revision         string                 `json:"revision"`
	Status           string                 `json:"status"`
	ValidationErrors []string               `json:"validationErrors,omitempty"`
	DeploymentStatus string                 `json:"deploymentStatus"`
	Resources        []string               `json:"resources,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// OCloudResourceResponse represents a response to O-Cloud resource provisioning
type OCloudResourceResponse struct {
	ResourceID       string                 `json:"resourceId"`
	Status           string                 `json:"status"`
	AllocatedResources AllocatedResources   `json:"allocatedResources"`
	PlacementInfo    PlacementInfo          `json:"placementInfo"`
	EnergyMetrics    *EnergyMetrics         `json:"energyMetrics,omitempty"`
	SLACompliance    bool                   `json:"slaCompliance"`
	Timestamp        time.Time              `json:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// PlacementInfo represents resource placement information
type PlacementInfo struct {
	Zone             string                 `json:"zone"`
	Region           string                 `json:"region"`
	NodeID           string                 `json:"nodeId"`
	Constraints      []string               `json:"constraints"`
	OptimizationGoal string                 `json:"optimizationGoal"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// EnergyMetrics is already defined in smo_components.go

// SLARequirements represents SLA requirements for resources
type SLARequirements struct {
	Availability        float64                `json:"availability"`
	MaxLatencyMs        float64                `json:"maxLatencyMs"`
	MinThroughputMbps   float64                `json:"minThroughputMbps"`
	MaxDowntimeMinutes  int                    `json:"maxDowntimeMinutes"`
	RecoveryTimeSeconds int                    `json:"recoveryTimeSeconds"`
	CustomSLAs          map[string]interface{} `json:"customSLAs,omitempty"`
}

// ComputeResources represents compute resource specifications
type ComputeResources struct {
	CPU              float64                `json:"cpu"`
	Memory           int64                  `json:"memory"`
	VCPUs            int                    `json:"vcpus"`
	Architecture     string                 `json:"architecture"`
	ProcessorFeatures []string              `json:"processorFeatures"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NetworkResources represents network resource specifications
type NetworkResources struct {
	Bandwidth        int64                  `json:"bandwidth"`
	Interfaces       []NetworkInterface     `json:"interfaces"`
	QoSPolicies      []string               `json:"qosPolicies"`
	VLANs            []int                  `json:"vlans"`
	SecurityGroups   []string               `json:"securityGroups"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NetworkInterface represents a network interface specification
type NetworkInterface struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Bandwidth   int64                  `json:"bandwidth"`
	MTU         int                    `json:"mtu"`
	VLANID      int                    `json:"vlanId,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StorageResources represents storage resource specifications
type StorageResources struct {
	Size             int64                  `json:"size"`
	Type             string                 `json:"type"`
	IOPS             int                    `json:"iops"`
	Throughput       int64                  `json:"throughput"`
	Durability       string                 `json:"durability"`
	EncryptionLevel  string                 `json:"encryptionLevel"`
	BackupPolicy     string                 `json:"backupPolicy"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AcceleratorResources represents accelerator resource specifications (GPUs, FPGAs, etc.)
type AcceleratorResources struct {
	Type             string                 `json:"type"`
	Count            int                    `json:"count"`
	Model            string                 `json:"model"`
	Memory           int64                  `json:"memory"`
	ComputeCapability string                `json:"computeCapability"`
	DriverVersion    string                 `json:"driverVersion"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceAllocation is already defined in slice_e2_integration.go

// ResourceRequirements represents resource requirements for deployments
type ResourceRequirements struct {
	CPU              float64                `json:"cpu"`
	Memory           int64                  `json:"memory"`
	Storage          int64                  `json:"storage"`
	GPU              int                    `json:"gpu,omitempty"`
	NetworkBandwidth int64                  `json:"networkBandwidth,omitempty"`
	CustomRequirements map[string]interface{} `json:"customRequirements,omitempty"`
}

// MemorySample represents a memory usage sample for stability testing
type MemorySample struct {
	Timestamp time.Time `json:"timestamp"`
	UsageMB   float64   `json:"usageMB"`
	HeapMB    float64   `json:"heapMB"`
}

// CPUSample represents a CPU usage sample for stability testing
type CPUSample struct {
	Timestamp time.Time `json:"timestamp"`
	Percent   float64   `json:"percent"`
}

// ErrorRateSample represents an error rate sample for stability testing
type ErrorRateSample struct {
	Timestamp time.Time `json:"timestamp"`
	ErrorRate float64   `json:"errorRate"`
}

// EventTriggerType represents the type of event trigger
type EventTriggerType string

const (
	EventTriggerTypePeriodic    EventTriggerType = "PERIODIC"
	EventTriggerTypeOnChange    EventTriggerType = "ON_CHANGE"
	EventTriggerTypeImmediate   EventTriggerType = "IMMEDIATE"
	EventTriggerTypeConditional EventTriggerType = "CONDITIONAL"
)

// Test orchestrator types

// ComprehensiveLoadTest represents a comprehensive load test
type ComprehensiveLoadTest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Config      LoadTestConfig         `json:"config"`
	Status      string                 `json:"status"`
	StartTime   time.Time              `json:"startTime"`
	EndTime     *time.Time             `json:"endTime,omitempty"`
	Results     *LoadTestReport        `json:"results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NephioR5IntegrationTest represents a Nephio R5 integration test
type NephioR5IntegrationTest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Config      NephioR5Config         `json:"config"`
	Status      string                 `json:"status"`
	StartTime   time.Time              `json:"startTime"`
	EndTime     *time.Time             `json:"endTime,omitempty"`
	Results     *NephioTestReport      `json:"results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LoadTestConfig represents load test configuration
type LoadTestConfig struct {
	MaxConcurrentUsers   int                    `json:"maxConcurrentUsers"`
	TestDuration        time.Duration          `json:"testDuration"`
	RampUpDuration      time.Duration          `json:"rampUpDuration"`
	RampDownDuration    time.Duration          `json:"rampDownDuration"`
	RequestRate         int                    `json:"requestRate"`
	TargetEndpoints     []string               `json:"targetEndpoints"`
	TestScenarios       []string               `json:"testScenarios"`
	CustomParameters    map[string]interface{} `json:"customParameters,omitempty"`
}

// NephioR5Config represents Nephio R5 test configuration
type NephioR5Config struct {
	PorchEndpoint       string                 `json:"porchEndpoint"`
	OCloudEndpoint      string                 `json:"oCloudEndpoint"`
	PackageRepoEndpoint string                 `json:"packageRepoEndpoint"`
	TestPackages        []string               `json:"testPackages"`
	ResourceQuotas      map[string]interface{} `json:"resourceQuotas"`
	TestTimeout         time.Duration          `json:"testTimeout"`
	CustomParameters    map[string]interface{} `json:"customParameters,omitempty"`
}

// LoadTestReport represents load test results
type LoadTestReport struct {
	TestID              string                 `json:"testId"`
	TotalRequests       uint64                 `json:"totalRequests"`
	SuccessfulRequests  uint64                 `json:"successfulRequests"`
	FailedRequests      uint64                 `json:"failedRequests"`
	AverageLatencyMs    float64                `json:"averageLatencyMs"`
	MaxLatencyMs        float64                `json:"maxLatencyMs"`
	MinLatencyMs        float64                `json:"minLatencyMs"`
	ThroughputRPS       float64                `json:"throughputRPS"`
	ErrorRate           float64                `json:"errorRate"`
	P95LatencyMs        float64                `json:"p95LatencyMs"`
	P99LatencyMs        float64                `json:"p99LatencyMs"`
	ResourceUsage       ResourceConsumption    `json:"resourceUsage"`
	Errors              []string               `json:"errors,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// NephioTestReport represents Nephio test results
type NephioTestReport struct {
	TestID              string                 `json:"testId"`
	PackageDeployments  uint64                 `json:"packageDeployments"`
	SuccessfulDeployments uint64               `json:"successfulDeployments"`
	FailedDeployments   uint64                 `json:"failedDeployments"`
	ResourceAllocations uint64                 `json:"resourceAllocations"`
	AverageDeployTimeMs float64                `json:"averageDeployTimeMs"`
	EnergyEfficiency    float64                `json:"energyEfficiency"`
	Errors              []string               `json:"errors,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// Missing types that need to be added

// E2NodeConfigurationUpdate represents configuration update information
type E2NodeConfigurationUpdate struct {
	UpdateID    string     `json:"updateId"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requestedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// SimulatedSubscription represents a subscription in the simulator
type SimulatedSubscription struct {
	SubscriptionID   string               `json:"subscriptionId"`
	ServiceModelOID  string               `json:"serviceModelOid"`
	E2NodeID         string               `json:"e2NodeId"`
	Actions          []SubscriptionAction `json:"actions"`
	ReportingPeriod  time.Duration        `json:"reportingPeriod"`
	IsActive         bool                 `json:"isActive"`
	CreatedAt        time.Time            `json:"createdAt"`
}

// RMRMessage represents a message in the RMR (RIC Message Router) protocol
type RMRMessage struct {
	MessageType    uint32            `json:"messageType"`
	SubscriptionID string            `json:"subscriptionId,omitempty"`
	Payload        []byte            `json:"payload"`
	Source         string            `json:"source"`
	Destination    string            `json:"destination"`
	TransactionID  string            `json:"transactionId,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// E2SMKPMMetrics represents KPM service model metrics
type E2SMKPMMetrics struct {
	E2NodeID          string            `json:"e2NodeId"`
	MeasurementData   []MeasurementData `json:"measurementData"`
	GranularityPeriod int64             `json:"granularityPeriod"`
	Timestamp         time.Time         `json:"timestamp"`
	SubscriptionID    string            `json:"subscriptionId"`
}

// MeasurementData represents measurement data point
type MeasurementData struct {
	MeasurementID    uint32            `json:"measurementId"`
	MeasurementValue interface{}       `json:"measurementValue"`
	MeasurementType  string            `json:"measurementType"`
	Labels           map[string]string `json:"labels"`
}

// NETCONFClient represents a NETCONF protocol client
type NETCONFClient struct {
	Host           string            `json:"host"`
	Port           int               `json:"port"`
	Username       string            `json:"username"`
	Password       string            `json:"password,omitempty"`
	Timeout        time.Duration     `json:"timeout"`
	Capabilities   []string          `json:"capabilities"`
	SessionID      string            `json:"sessionId,omitempty"`
	Connected      bool              `json:"connected"`
	LastActivity   time.Time         `json:"lastActivity"`
}

// ProcessingStats tracks processing statistics
type ProcessingStats struct {
	MessagesProcessed uint64        `json:"messagesProcessed"`
	ProcessingTime    time.Duration `json:"processingTime"`
	ErrorCount        uint64        `json:"errorCount"`
	ThroughputMbps    float64       `json:"throughputMbps"`
}

// LatencyMeasurement represents a latency measurement operation
type LatencyMeasurement struct {
	OperationID   string                 `json:"operationId"`
	OperationType string                 `json:"operationType"`
	StartTime     time.Time              `json:"startTime"`
	EndTime       *time.Time             `json:"endTime,omitempty"`
	LatencyMs     float64                `json:"latencyMs"`
	Status        string                 `json:"status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ResponseTimeMetrics represents statistical metrics for response times
type ResponseTimeMetrics struct {
	Average       float64 `json:"average"`
	Min           float64 `json:"min"`
	Max           float64 `json:"max"`
	P50           float64 `json:"p50"`
	P95           float64 `json:"p95"`
	P99           float64 `json:"p99"`
	StandardDev   float64 `json:"standardDev"`
	TotalSamples  int     `json:"totalSamples"`
}

// PolicyConflict represents a policy conflict
type PolicyConflict struct {
	ConflictID          string                 `json:"conflictId"`
	ConflictType        string                 `json:"conflictType"`
	ConflictingPolicies []PolicyInstanceID     `json:"conflictingPolicies"`
	Description         string                 `json:"description"`
	Severity            string                 `json:"severity"`
	Resolution          string                 `json:"resolution,omitempty"`
	DetectedAt          time.Time              `json:"detectedAt"`
	ResolvedAt          *time.Time             `json:"resolvedAt,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyDistributionRequest represents a request to distribute a policy to xApps
type PolicyDistributionRequest struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	PolicyTypeID     PolicyTypeID     `json:"policy_type_id"`
	Policy           json.RawMessage  `json:"policy"`
	TargetXApps      []string         `json:"target_xapps"`
}

// PolicyComplianceRequest represents a request to check policy compliance
type PolicyComplianceRequest struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	XAppID           string           `json:"xapp_id"`
}

// XAppClient represents a client for communicating with an xApp
type XAppClient struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

// PolicyDistributionStatus represents the status of policy distribution
type PolicyDistributionStatus struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	XAppID           string           `json:"xapp_id"`
	Status           string           `json:"status"`
	Message          string           `json:"message,omitempty"`
	LastUpdate       time.Time        `json:"last_update"`
}

// PolicyComplianceReport represents a policy compliance report
type PolicyComplianceReport struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	XAppID           string           `json:"xapp_id"`
	ComplianceStatus string           `json:"compliance_status"`
	Violations       []string         `json:"violations,omitempty"`
	LastCheck        time.Time        `json:"last_check"`
}

// ProductionHealthChecker represents a production health checker
type ProductionHealthChecker struct {
	Components       map[string]RICComponentHealth `json:"components"`
	LastCheck        time.Time                     `json:"last_check"`
	CheckInterval    time.Duration                 `json:"check_interval"`
	AlertThresholds  map[string]float64            `json:"alert_thresholds"`
	mu               sync.RWMutex                  `json:"-"`
}

// RICComponentHealth represents the health status of a RIC component
type RICComponentHealth struct {
	ComponentName string                 `json:"component_name"`
	Status        HealthStatus           `json:"status"`
	LastCheck     time.Time              `json:"last_check"`
	ResponseTime  time.Duration          `json:"response_time"`
	ErrorRate     float64                `json:"error_rate"`
	Uptime        time.Duration          `json:"uptime"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// ServiceModelClient represents a client for service model operations
type ServiceModelClient struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

// ServiceModelSubscription represents a service model subscription
type ServiceModelSubscription struct {
	ID            string            `json:"id"`
	ServiceModel  string            `json:"serviceModel"`
	EventTypes    []string          `json:"eventTypes"`
	CreatedAt     time.Time         `json:"createdAt"`
	LastActivity  time.Time         `json:"lastActivity"`
	Active        bool              `json:"active"`
}

// ServiceModelEventType represents types of service model events
type ServiceModelEventType string

const (
	ServiceModelEventTypeRegistration   ServiceModelEventType = "registration"
	ServiceModelEventTypeDeregistration ServiceModelEventType = "deregistration"
	ServiceModelEventTypeUpdate         ServiceModelEventType = "update"
	ServiceModelEventTypeError          ServiceModelEventType = "error"
)

// ServiceModelEventHandler represents a handler function for service model events
type ServiceModelEventHandler func(event ServiceModelEvent)

// ServiceModelEvent represents a service model event
type ServiceModelEvent struct {
	Type      ServiceModelEventType  `json:"type"`
	ModelID   string                 `json:"modelId"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ServiceModelAPIConfig represents configuration for service model API
type ServiceModelAPIConfig struct {
	MaxConcurrentOps int           `json:"maxConcurrentOps"`
	RequestTimeout   time.Duration `json:"requestTimeout"`
	RetryAttempts    int           `json:"retryAttempts"`
	EnableEvents     bool          `json:"enableEvents"`
}

// ServiceModelAPI struct for service model operations
type ServiceModelAPI struct {
	mu               sync.RWMutex                                     `json:"-"`
	registry         *ServiceModelRegistry                            `json:"-"`
	clients          map[string]*ServiceModelClient                   `json:"-"`
	subscriptions    map[string]*ServiceModelSubscription             `json:"-"`
	eventHandlers    map[ServiceModelEventType][]ServiceModelEventHandler `json:"-"`
	config           *ServiceModelAPIConfig                           `json:"-"`
}

// ServiceModelCapabilities represents capabilities of a service model
type ServiceModelCapabilities struct {
	ServiceModelType      ServiceModelType `json:"serviceModelType"`
	Version               string           `json:"version"`
	SupportedOperations   []string         `json:"supportedOperations"`
	SupportedMessageTypes []string         `json:"supportedMessageTypes"`
	SupportsIndications   bool             `json:"supportsIndications"`
	SupportsControl       bool             `json:"supportsControl"`
	MaxConcurrentOps      int              `json:"maxConcurrentOps"`
	LastUpdated           time.Time        `json:"lastUpdated"`
}

// ServiceModelStatistics represents statistics for service models
type ServiceModelStatistics struct {
	ServiceModelType      ServiceModelType `json:"serviceModelType"`
	IndicationsProcessed  uint64           `json:"indicationsProcessed"`
	ControlsProcessed     uint64           `json:"controlsProcessed"`
	ValidationErrors      uint64           `json:"validationErrors"`
	ProcessingErrors      uint64           `json:"processingErrors"`
	AverageProcessingTime time.Duration    `json:"averageProcessingTime"`
	LastProcessedAt       time.Time        `json:"lastProcessedAt"`
	TotalProcessingTime   time.Duration    `json:"totalProcessingTime"`
}

// ServiceModelDefinition represents a service model definition
type ServiceModelDefinition struct {
	OID           string                   `json:"oid"`
	Name          string                   `json:"name"`
	Type          ServiceModelType         `json:"type"`
	Version       string                   `json:"version"`
	Description   string                   `json:"description"`
	Capabilities  []ServiceModelCapability `json:"capabilities"`
	RANFunctions  []RANFunction            `json:"ranFunctions"`
	LastUpdated   time.Time                `json:"lastUpdated"`
}

// ServiceModelCapability represents a capability of a service model
type ServiceModelCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// OptimizedHTTPClient represents an optimized HTTP client
type OptimizedHTTPClient struct {
	*http.Client
	MaxIdleConns        int           `json:"maxIdleConns"`
	MaxConnsPerHost     int           `json:"maxConnsPerHost"`
	IdleConnTimeout     time.Duration `json:"idleConnTimeout"`
	RequestTimeout      time.Duration `json:"requestTimeout"`
	MaxRetries          int           `json:"maxRetries"`
	RetryDelay          time.Duration `json:"retryDelay"`
}

// PolicyAPI represents a policy management API
type PolicyAPI struct {
	BaseURL    string                 `json:"baseUrl"`
	Client     *OptimizedHTTPClient   `json:"-"`
	Headers    map[string]string      `json:"headers"`
	Timeout    time.Duration          `json:"timeout"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// EnrichmentAPI represents an enrichment information API
type EnrichmentAPI struct {
	BaseURL    string                 `json:"baseUrl"`
	Client     *OptimizedHTTPClient   `json:"-"`
	Headers    map[string]string      `json:"headers"`
	Timeout    time.Duration          `json:"timeout"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// DMAAPClient represents a DMAAP client
type DMAAPClient struct {
	BaseURL     string                 `json:"baseUrl"`
	Client      *OptimizedHTTPClient   `json:"-"`
	TopicPrefix string                 `json:"topicPrefix"`
	ConsumerID  string                 `json:"consumerId"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// SubscriptionListResponse represents a response with subscription list
type SubscriptionListResponse struct {
	Subscriptions []Subscription `json:"subscriptions"`
	TotalCount    int            `json:"totalCount"`
	Page          int            `json:"page"`
	PageSize      int            `json:"pageSize"`
}

// SubscriptionUpdate represents an update to a subscription
type SubscriptionUpdate struct {
	SubscriptionID string                 `json:"subscriptionId"`
	Updates        map[string]interface{} `json:"updates"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	UpdatedBy      string                 `json:"updatedBy,omitempty"`
}

// Action represents a subscription action (simplified version)
type Action struct {
	ActionID   int                    `json:"actionId"`
	ActionType string                 `json:"actionType"`
	Definition map[string]interface{} `json:"definition"`
}

// NodeStatus represents the status of a node
type NodeStatus struct {
	NodeID       string                 `json:"nodeId"`
	Status       string                 `json:"status"`
	LastSeen     time.Time              `json:"lastSeen"`
	Health       HealthStatus           `json:"health"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Address      string                 `json:"address,omitempty"`
}




// Additional missing types for various components

// PolicyStatus represents the status of a policy
type PolicyStatus struct {
	PolicyID     string                 `json:"policyId"`
	Status       string                 `json:"status"`
	Enforced     bool                   `json:"enforced"`
	LastUpdated  time.Time              `json:"lastUpdated"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ComprehensiveComplianceReport represents a comprehensive compliance report
type ComprehensiveComplianceReport struct {
	ReportID          string                 `json:"reportId"`
	GeneratedAt       time.Time              `json:"generatedAt"`
	ComplianceScore   float64                `json:"complianceScore"`
	Standards         []StandardValidation   `json:"standards"`
	Violations        []ComplianceViolation  `json:"violations"`
	Recommendations   []string               `json:"recommendations"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// StandardValidation represents validation against a specific standard
type StandardValidation struct {
	StandardName    string                 `json:"standardName"`
	StandardVersion string                 `json:"standardVersion"`
	ComplianceScore float64                `json:"complianceScore"`
	RequirementsMet int                    `json:"requirementsMet"`
	TotalRequirements int                  `json:"totalRequirements"`
	Violations      []ComplianceViolation  `json:"violations"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceViolation is already defined in compliance_validator.go

// LatencyBenchmarkResult represents latency benchmark results
type LatencyBenchmarkResult struct {
	BenchmarkName    string        `json:"benchmarkName"`
	TotalRequests    uint64        `json:"totalRequests"`
	SuccessfulRequests uint64      `json:"successfulRequests"`
	AverageLatency   time.Duration `json:"averageLatency"`
	MinLatency       time.Duration `json:"minLatency"`
	MaxLatency       time.Duration `json:"maxLatency"`
	P50Latency       time.Duration `json:"p50Latency"`
	P95Latency       time.Duration `json:"p95Latency"`
	P99Latency       time.Duration `json:"p99Latency"`
	StandardDeviation time.Duration `json:"standardDeviation"`
	Timestamp        time.Time     `json:"timestamp"`
}

// ThroughputBenchmarkResult represents throughput benchmark results
type ThroughputBenchmarkResult struct {
	BenchmarkName       string    `json:"benchmarkName"`
	Duration            time.Duration `json:"duration"`
	TotalRequests       uint64    `json:"totalRequests"`
	SuccessfulRequests  uint64    `json:"successfulRequests"`
	FailedRequests      uint64    `json:"failedRequests"`
	RequestsPerSecond   float64   `json:"requestsPerSecond"`
	PeakThroughput      float64   `json:"peakThroughput"`
	AverageThroughput   float64   `json:"averageThroughput"`
	ThroughputVariance  float64   `json:"throughputVariance"`
	Timestamp           time.Time `json:"timestamp"`
}