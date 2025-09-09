// Package dashboard provides types and interfaces for the O-RAN Near-RT RIC dashboard
package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
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

// GetPolicyTypes retrieves all policy types
func (c *A1MediatorClientImpl) GetPolicyTypes() ([]PolicyType, error) {
	return []PolicyType{}, nil
}

// GetPolicyType retrieves a specific policy type
func (c *A1MediatorClientImpl) GetPolicyType(typeID PolicyTypeID) (*PolicyType, error) {
	return &PolicyType{}, nil
}

// CreatePolicyType creates a new policy type
func (c *A1MediatorClientImpl) CreatePolicyType(policyType *PolicyType) error {
	return nil
}

// DeletePolicyType deletes a policy type
func (c *A1MediatorClientImpl) DeletePolicyType(typeID PolicyTypeID) error {
	return nil
}

// GetPolicyInstances retrieves all policy instances for a type
func (c *A1MediatorClientImpl) GetPolicyInstances(typeID PolicyTypeID) ([]PolicyInstance, error) {
	return []PolicyInstance{}, nil
}

// GetPolicyInstance retrieves a specific policy instance
func (c *A1MediatorClientImpl) GetPolicyInstance(typeID PolicyTypeID, instanceID PolicyInstanceID) (*PolicyInstance, error) {
	return &PolicyInstance{}, nil
}

// CreatePolicyInstance creates a new policy instance
func (c *A1MediatorClientImpl) CreatePolicyInstance(instance *PolicyInstance) error {
	return nil
}

// UpdatePolicyInstance updates an existing policy instance
func (c *A1MediatorClientImpl) UpdatePolicyInstance(instance *PolicyInstance) error {
	return nil
}

// DeletePolicyInstance deletes a policy instance
func (c *A1MediatorClientImpl) DeletePolicyInstance(typeID PolicyTypeID, instanceID PolicyInstanceID) error {
	return nil
}

// GetPolicyStatus retrieves the status of a policy instance
func (c *A1MediatorClientImpl) GetPolicyStatus(typeID PolicyTypeID, instanceID PolicyInstanceID) (string, error) {
	return "active", nil
}

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