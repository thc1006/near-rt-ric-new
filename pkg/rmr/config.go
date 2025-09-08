package rmr

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// RMRConfig represents the full RMR configuration
type RMRConfig struct {
	DefaultRoutes     []string               `yaml:"default_routes"`
	Routes           map[string][]string     `yaml:"routes"`
	ConnectionPool   ConnectionPoolConfig    `yaml:"connection_pool"`
	RetryPolicy      RetryPolicyConfig       `yaml:"retry_policy"`
	Telemetry        TelemetryConfig         `yaml:"telemetry"`

	routingTable *RoutingTable
	mu           sync.Mutex
}

type ConnectionPoolConfig struct {
	MaxConnections     int    `yaml:"max_connections"`
	IdleTimeout       string `yaml:"idle_timeout"`
	ConnectionTimeout string `yaml:"connection_timeout"`
}

type RetryPolicyConfig struct {
	MaxRetries         int    `yaml:"max_retries"`
	BackoffStrategy    string `yaml:"backoff_strategy"`
	InitialBackoff    string `yaml:"initial_backoff"`
	MaxBackoff        string `yaml:"max_backoff"`
}

type TelemetryConfig struct {
	Enabled           bool   `yaml:"enabled"`
	MetricsEndpoint   string `yaml:"metrics_endpoint"`
	LogRoutingErrors bool   `yaml:"log_routing_errors"`
}

// LoadConfig reads RMR configuration from a YAML file
func LoadConfig(configPath string) (*RMRConfig, error) {
	// Use absolute path
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %v", err)
	}

	// Read file contents
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Unmarshal YAML
	var config RMRConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Initialize routing table
	config.routingTable = NewRoutingTable()

	// Configure default routes
	if len(config.DefaultRoutes) > 0 {
		config.routingTable.SetDefaultRoutes(config.DefaultRoutes...)
	}

	// Configure specific routes
	for routeName, endpoints := range config.Routes {
		// Map route name to MessageType
		msgType := mapRouteNameToMessageType(routeName)
		if err := config.routingTable.AddRoute(msgType, endpoints...); err != nil {
			return nil, fmt.Errorf("failed to add route %s: %v", routeName, err)
		}
	}

	return &config, nil
}

// mapRouteNameToMessageType converts route names to MessageType
func mapRouteNameToMessageType(routeName string) MessageType {
	switch routeName {
	case "e2_setup":
		return E2_SETUP_REQUEST
	case "subscription":
		return SUBSCRIPTION_REQUEST
	case "indication":
		return INDICATION_MESSAGE
	case "ric_control":
		return RIC_CONTROL_REQUEST
	default:
		return 0 // Default/Unknown
	}
}

// GetRoutingTable returns the configured routing table
func (c *RMRConfig) GetRoutingTable() *RoutingTable {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.routingTable
}

// ApplyConfig allows dynamic configuration updates
func (c *RMRConfig) ApplyConfig(newConfig *RMRConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update routing table
	c.routingTable = newConfig.routingTable
	
	// Update other configuration parameters
	c.ConnectionPool = newConfig.ConnectionPool
	c.RetryPolicy = newConfig.RetryPolicy
	c.Telemetry = newConfig.Telemetry

	return nil
}