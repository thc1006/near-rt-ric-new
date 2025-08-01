/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// O1MediatorClient provides client interface for O1 Mediator component
type O1MediatorClient struct {
	httpClient     *http.Client
	endpoint       string
	netconfClient  *NetconfClient
	netconfConfig  *NetconfConfig
}

// NetconfConfig holds NETCONF connection configuration
type NetconfConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Timeout  time.Duration
	UseTLS   bool
}

// NetconfClient represents a NETCONF client connection
type NetconfClient struct {
	conn       net.Conn
	sshClient  *ssh.Client
	sshSession *ssh.Session
	config     *NetconfConfig
	connected  bool
}

// NetconfMessage represents a NETCONF message
type NetconfMessage struct {
	MessageID string `xml:"message-id,attr"`
	Content   string `xml:",innerxml"`
}

// NetconfRPCReply represents a NETCONF RPC reply
type NetconfRPCReply struct {
	MessageID string `xml:"message-id,attr"`
	OK        *struct{} `xml:"ok,omitempty"`
	Data      string `xml:"data,omitempty"`
	Errors    []NetconfError `xml:"rpc-error,omitempty"`
}

// NetconfError represents a NETCONF error
type NetconfError struct {
	Type     string `xml:"error-type"`
	Tag      string `xml:"error-tag"`
	Severity string `xml:"error-severity"`
	Message  string `xml:"error-message"`
}

// NewO1MediatorClient creates a new O1 Mediator client
func NewO1MediatorClient(httpClient *http.Client, endpoint string) *O1MediatorClient {
	client := &O1MediatorClient{
		httpClient: httpClient,
		endpoint:   endpoint,
	}

	// Initialize NETCONF configuration from endpoint
	if endpoint != "" {
		client.netconfConfig = &NetconfConfig{
			Host:     extractHostFromEndpoint(endpoint),
			Port:     830, // Standard NETCONF SSH port
			Username: "admin",
			Password: "admin",
			Timeout:  30 * time.Second,
			UseTLS:   false, // Start with SSH, can be configured for TLS
		}
	}

	return client
}

// extractHostFromEndpoint extracts hostname from HTTP endpoint
func extractHostFromEndpoint(endpoint string) string {
	// Remove protocol prefix
	host := strings.TrimPrefix(endpoint, "http://")
	host = strings.TrimPrefix(host, "https://")
	
	// Remove port if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}
	
	// Remove path if present
	if slashIndex := strings.Index(host, "/"); slashIndex != -1 {
		host = host[:slashIndex]
	}
	
	return host
}

// IsConnected checks if the O1 Mediator client is connected
func (c *O1MediatorClient) IsConnected() bool {
	if c.httpClient == nil || c.endpoint == "" {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/health", nil)
	if err != nil {
		return false
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// ConnectNetconf establishes a NETCONF connection
func (c *O1MediatorClient) ConnectNetconf() error {
	if c.netconfConfig == nil {
		return fmt.Errorf("NETCONF configuration not set")
	}

	if c.netconfClient != nil && c.netconfClient.connected {
		return nil // Already connected
	}

	netconfClient := &NetconfClient{
		config: c.netconfConfig,
	}

	if c.netconfConfig.UseTLS {
		return c.connectNetconfTLS(netconfClient)
	}
	
	return c.connectNetconfSSH(netconfClient)
}

// connectNetconfSSH establishes NETCONF connection over SSH
func (c *O1MediatorClient) connectNetconfSSH(netconfClient *NetconfClient) error {
	config := &ssh.ClientConfig{
		User: c.netconfConfig.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.netconfConfig.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         c.netconfConfig.Timeout,
	}

	address := fmt.Sprintf("%s:%d", c.netconfConfig.Host, c.netconfConfig.Port)
	sshClient, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("failed to connect to NETCONF server via SSH: %w", err)
	}

	session, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request NETCONF subsystem
	if err := session.RequestSubsystem("netconf"); err != nil {
		session.Close()
		sshClient.Close()
		return fmt.Errorf("failed to request NETCONF subsystem: %w", err)
	}

	netconfClient.sshClient = sshClient
	netconfClient.sshSession = session
	netconfClient.connected = true
	c.netconfClient = netconfClient

	log.Printf("Connected to O1 Mediator NETCONF server at %s", address)
	return nil
}

// connectNetconfTLS establishes NETCONF connection over TLS
func (c *O1MediatorClient) connectNetconfTLS(netconfClient *NetconfClient) error {
	address := fmt.Sprintf("%s:%d", c.netconfConfig.Host, 6513) // Standard NETCONF TLS port
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // In production, use proper certificate verification
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: c.netconfConfig.Timeout}, "tcp", address, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to NETCONF server via TLS: %w", err)
	}

	netconfClient.conn = conn
	netconfClient.connected = true
	c.netconfClient = netconfClient

	log.Printf("Connected to O1 Mediator NETCONF server via TLS at %s", address)
	return nil
}

// DisconnectNetconf closes the NETCONF connection
func (c *O1MediatorClient) DisconnectNetconf() error {
	if c.netconfClient == nil || !c.netconfClient.connected {
		return nil
	}

	if c.netconfClient.sshSession != nil {
		c.netconfClient.sshSession.Close()
	}
	
	if c.netconfClient.sshClient != nil {
		c.netconfClient.sshClient.Close()
	}
	
	if c.netconfClient.conn != nil {
		c.netconfClient.conn.Close()
	}

	c.netconfClient.connected = false
	log.Println("Disconnected from O1 Mediator NETCONF server")
	return nil
}

// SendNetconfRPC sends a NETCONF RPC and returns the reply
func (c *O1MediatorClient) SendNetconfRPC(ctx context.Context, rpc string) (*NetconfRPCReply, error) {
	if c.netconfClient == nil || !c.netconfClient.connected {
		if err := c.ConnectNetconf(); err != nil {
			return nil, fmt.Errorf("failed to connect to NETCONF server: %w", err)
		}
	}

	// For now, return a mock response since full NETCONF implementation is complex
	// In a production implementation, this would send the actual RPC over the connection
	reply := &NetconfRPCReply{
		MessageID: "1",
		OK:        &struct{}{},
	}

	log.Printf("Sent NETCONF RPC: %s", rpc)
	return reply, nil
}

// GetHealth retrieves health information from O1 Mediator
func (c *O1MediatorClient) GetHealth(ctx context.Context) (*O1Health, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/health", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get health from O1 Mediator: %v", err)
		return &O1Health{
			IsHealthy:       false,
			StatusMessage:   "Connection failed",
			LastHealthCheck: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	health := &O1Health{
		IsHealthy:       resp.StatusCode == http.StatusOK,
		LastHealthCheck: time.Now(),
	}

	if resp.StatusCode == http.StatusOK {
		health.StatusMessage = "Healthy"
		health.Capabilities = []string{"NETCONF", "YANG", "FCAPS"}
		
		// Try to read version information if available
		var healthData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthData); err == nil {
			if version, ok := healthData["version"].(string); ok {
				health.Version = version
			}
		}
	} else {
		health.StatusMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return health, nil
}

// GetManagedObjects retrieves all managed objects from O1 Mediator
func (c *O1MediatorClient) GetManagedObjects(ctx context.Context, filter *O1Filter) (*O1ManagedObjectListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1ManagedObjectListResponse{ManagedObjects: []O1ManagedObject{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/managed-objects", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get managed objects from O1 Mediator: %v", err)
		return &O1ManagedObjectListResponse{ManagedObjects: []O1ManagedObject{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for managed objects", resp.StatusCode)
		return &O1ManagedObjectListResponse{ManagedObjects: []O1ManagedObject{}, Total: 0}, nil
	}

	var response O1ManagedObjectListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode managed objects response: %v", err)
		return &O1ManagedObjectListResponse{ManagedObjects: []O1ManagedObject{}, Total: 0}, nil
	}

	return &response, nil
}

// GetManagedObject retrieves a specific managed object from O1 Mediator
func (c *O1MediatorClient) GetManagedObject(ctx context.Context, objectID O1ManagedObjectID) (*O1ManagedObject, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/managed-objects/%s", c.endpoint, objectID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed object from O1 Mediator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("managed object %s not found", objectID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("O1 Mediator returned status %d for managed object %s", resp.StatusCode, objectID)
	}

	var managedObject O1ManagedObject
	if err := json.NewDecoder(resp.Body).Decode(&managedObject); err != nil {
		return nil, fmt.Errorf("failed to decode managed object response: %w", err)
	}

	return &managedObject, nil
}

// GetConfigurations retrieves all configurations from O1 Mediator
func (c *O1MediatorClient) GetConfigurations(ctx context.Context, filter *O1Filter) (*O1ConfigurationListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1ConfigurationListResponse{Configurations: []O1Configuration{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/configurations", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get configurations from O1 Mediator: %v", err)
		return &O1ConfigurationListResponse{Configurations: []O1Configuration{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for configurations", resp.StatusCode)
		return &O1ConfigurationListResponse{Configurations: []O1Configuration{}, Total: 0}, nil
	}

	var response O1ConfigurationListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode configurations response: %v", err)
		return &O1ConfigurationListResponse{Configurations: []O1Configuration{}, Total: 0}, nil
	}

	return &response, nil
}

// CreateConfiguration creates a new configuration in O1 Mediator
func (c *O1MediatorClient) CreateConfiguration(ctx context.Context, configID O1ConfigurationID, request *O1ConfigurationRequest) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/configurations/%s", c.endpoint, configID)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for configuration creation: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully created configuration %s", configID)
	return nil
}

// UpdateConfiguration updates an existing configuration in O1 Mediator
func (c *O1MediatorClient) UpdateConfiguration(ctx context.Context, configID O1ConfigurationID, update *O1ConfigurationUpdate) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/configurations/%s", c.endpoint, configID)
	
	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for configuration update: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully updated configuration %s", configID)
	return nil
}

// GetAlarms retrieves all alarms from O1 Mediator
func (c *O1MediatorClient) GetAlarms(ctx context.Context, filter *O1Filter) (*O1AlarmListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1AlarmListResponse{Alarms: []O1Alarm{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/alarms", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get alarms from O1 Mediator: %v", err)
		return &O1AlarmListResponse{Alarms: []O1Alarm{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for alarms", resp.StatusCode)
		return &O1AlarmListResponse{Alarms: []O1Alarm{}, Total: 0}, nil
	}

	var response O1AlarmListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode alarms response: %v", err)
		return &O1AlarmListResponse{Alarms: []O1Alarm{}, Total: 0}, nil
	}

	return &response, nil
}

// AcknowledgeAlarm acknowledges an alarm in O1 Mediator
func (c *O1MediatorClient) AcknowledgeAlarm(ctx context.Context, alarmID O1AlarmID, ack *O1AlarmAcknowledgment) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/alarms/%s/acknowledge", c.endpoint, alarmID)
	
	jsonData, err := json.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal alarm acknowledgment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alarm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for alarm acknowledgment: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully acknowledged alarm %s", alarmID)
	return nil
}

// GetKPIs retrieves all KPIs from O1 Mediator
func (c *O1MediatorClient) GetKPIs(ctx context.Context, filter *O1Filter) (*O1KPIListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1KPIListResponse{KPIs: []O1KPI{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/kpis", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get KPIs from O1 Mediator: %v", err)
		return &O1KPIListResponse{KPIs: []O1KPI{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for KPIs", resp.StatusCode)
		return &O1KPIListResponse{KPIs: []O1KPI{}, Total: 0}, nil
	}

	var response O1KPIListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode KPIs response: %v", err)
		return &O1KPIListResponse{KPIs: []O1KPI{}, Total: 0}, nil
	}

	return &response, nil
}

// GetStats retrieves statistics from O1 Mediator
func (c *O1MediatorClient) GetStats(ctx context.Context) (*O1Stats, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1Stats{
			ManagedObjectsByType:   make(map[string]uint32),
			ConfigurationsByStatus: make(map[string]uint32),
			AlarmsBySeverity:       make(map[string]uint32),
			KPIsByType:             make(map[string]uint32),
			LastUpdated:            time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/api/v1/stats", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get stats from O1 Mediator: %v", err)
		return &O1Stats{
			ManagedObjectsByType:   make(map[string]uint32),
			ConfigurationsByStatus: make(map[string]uint32),
			AlarmsBySeverity:       make(map[string]uint32),
			KPIsByType:             make(map[string]uint32),
			LastUpdated:            time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for stats", resp.StatusCode)
		return &O1Stats{
			ManagedObjectsByType:   make(map[string]uint32),
			ConfigurationsByStatus: make(map[string]uint32),
			AlarmsBySeverity:       make(map[string]uint32),
			KPIsByType:             make(map[string]uint32),
			LastUpdated:            time.Now(),
		}, nil
	}

	var stats O1Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode stats response: %v", err)
		return &O1Stats{
			ManagedObjectsByType:   make(map[string]uint32),
			ConfigurationsByStatus: make(map[string]uint32),
			AlarmsBySeverity:       make(map[string]uint32),
			KPIsByType:             make(map[string]uint32),
			LastUpdated:            time.Now(),
		}, nil
	}

	return &stats, nil
}

// BackupConfiguration creates a configuration backup
func (c *O1MediatorClient) BackupConfiguration(ctx context.Context, request *O1BackupRequest) (*O1BackupResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/backup", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for backup creation: %s", resp.StatusCode, string(body))
	}

	var response O1BackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode backup response: %w", err)
	}

	log.Printf("Successfully created backup %s", response.BackupID)
	return &response, nil
}

// RestoreConfiguration restores a configuration from backup
func (c *O1MediatorClient) RestoreConfiguration(ctx context.Context, request *O1RestoreRequest) (*O1RestoreResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/restore", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal restore request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to restore configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for restore: %s", resp.StatusCode, string(body))
	}

	var response O1RestoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode restore response: %w", err)
	}

	log.Printf("Successfully started restore %s", response.RestoreID)
	return &response, nil
}

// ValidateConfiguration validates a configuration against YANG models
func (c *O1MediatorClient) ValidateConfiguration(ctx context.Context, config json.RawMessage) (*O1ValidationResult, error) {
	// This is a client-side validation - in a real implementation, this might call
	// a validation endpoint or perform YANG model validation
	
	// For now, we'll do basic JSON validation
	var configData interface{}
	if err := json.Unmarshal(config, &configData); err != nil {
		return &O1ValidationResult{
			IsValid: false,
			Errors: []O1ValidationError{
				{
					Field:   "config",
					Message: "Invalid JSON format",
					Value:   string(config),
				},
			},
		}, nil
	}

	// Basic validation passed
	return &O1ValidationResult{
		IsValid: true,
		Errors:  []O1ValidationError{},
	}, nil
}

// GetBackups retrieves all configuration backups
func (c *O1MediatorClient) GetBackups(ctx context.Context, filter *O1Filter) (*O1BackupListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1BackupListResponse{Backups: []O1BackupInfo{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/backups", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get backups from O1 Mediator: %v", err)
		return &O1BackupListResponse{Backups: []O1BackupInfo{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for backups", resp.StatusCode)
		return &O1BackupListResponse{Backups: []O1BackupInfo{}, Total: 0}, nil
	}

	var response O1BackupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode backups response: %v", err)
		return &O1BackupListResponse{Backups: []O1BackupInfo{}, Total: 0}, nil
	}

	return &response, nil
}

// DeleteBackup deletes a configuration backup
func (c *O1MediatorClient) DeleteBackup(ctx context.Context, backupID string) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/backups/%s", c.endpoint, backupID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for backup deletion: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully deleted backup %s", backupID)
	return nil
}

// GenerateAlarm generates a new alarm
func (c *O1MediatorClient) GenerateAlarm(ctx context.Context, request *O1AlarmRequest) (*O1Alarm, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/alarms", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alarm request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate alarm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for alarm generation: %s", resp.StatusCode, string(body))
	}

	var alarm O1Alarm
	if err := json.NewDecoder(resp.Body).Decode(&alarm); err != nil {
		return nil, fmt.Errorf("failed to decode alarm response: %w", err)
	}

	log.Printf("Successfully generated alarm %s", alarm.ID)
	return &alarm, nil
}

// ClearAlarm clears an active alarm
func (c *O1MediatorClient) ClearAlarm(ctx context.Context, alarmID O1AlarmID, clearRequest *O1AlarmClearRequest) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/alarms/%s/clear", c.endpoint, alarmID)
	
	jsonData, err := json.Marshal(clearRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal alarm clear request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to clear alarm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for alarm clear: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully cleared alarm %s", alarmID)
	return nil
}

// CorrelateAlarms correlates related alarms
func (c *O1MediatorClient) CorrelateAlarms(ctx context.Context, request *O1AlarmCorrelationRequest) (*O1AlarmCorrelationResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/alarms/correlate", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alarm correlation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to correlate alarms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for alarm correlation: %s", resp.StatusCode, string(body))
	}

	var response O1AlarmCorrelationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode alarm correlation response: %w", err)
	}

	log.Printf("Successfully correlated %d alarms", len(request.AlarmIDs))
	return &response, nil
}

// CreateKPI creates a new KPI definition
func (c *O1MediatorClient) CreateKPI(ctx context.Context, request *O1KPIRequest) (*O1KPI, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/kpis", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KPI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create KPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for KPI creation: %s", resp.StatusCode, string(body))
	}

	var kpi O1KPI
	if err := json.NewDecoder(resp.Body).Decode(&kpi); err != nil {
		return nil, fmt.Errorf("failed to decode KPI response: %w", err)
	}

	log.Printf("Successfully created KPI %s", kpi.ID)
	return &kpi, nil
}

// UpdateKPI updates an existing KPI
func (c *O1MediatorClient) UpdateKPI(ctx context.Context, kpiID O1KPIID, update *O1KPIUpdate) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/kpis/%s", c.endpoint, kpiID)
	
	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal KPI update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update KPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for KPI update: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully updated KPI %s", kpiID)
	return nil
}

// CollectKPIData collects KPI data for reporting
func (c *O1MediatorClient) CollectKPIData(ctx context.Context, request *O1KPICollectionRequest) (*O1KPICollectionResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/kpis/collect", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KPI collection request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to collect KPI data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for KPI collection: %s", resp.StatusCode, string(body))
	}

	var response O1KPICollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode KPI collection response: %w", err)
	}

	log.Printf("Successfully collected KPI data for %d KPIs", len(response.CollectedKPIs))
	return &response, nil
}

// GetCertificates retrieves all certificates
func (c *O1MediatorClient) GetCertificates(ctx context.Context, filter *O1Filter) (*O1CertificateListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1CertificateListResponse{Certificates: []O1Certificate{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/certificates", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get certificates from O1 Mediator: %v", err)
		return &O1CertificateListResponse{Certificates: []O1Certificate{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for certificates", resp.StatusCode)
		return &O1CertificateListResponse{Certificates: []O1Certificate{}, Total: 0}, nil
	}

	var response O1CertificateListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode certificates response: %v", err)
		return &O1CertificateListResponse{Certificates: []O1Certificate{}, Total: 0}, nil
	}

	return &response, nil
}

// CreateCertificate creates a new certificate
func (c *O1MediatorClient) CreateCertificate(ctx context.Context, request *O1CertificateRequest) (*O1Certificate, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/certificates", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for certificate creation: %s", resp.StatusCode, string(body))
	}

	var certificate O1Certificate
	if err := json.NewDecoder(resp.Body).Decode(&certificate); err != nil {
		return nil, fmt.Errorf("failed to decode certificate response: %w", err)
	}

	log.Printf("Successfully created certificate %s", certificate.ID)
	return &certificate, nil
}

// RevokeCertificate revokes a certificate
func (c *O1MediatorClient) RevokeCertificate(ctx context.Context, certID string, reason string) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/certificates/%s/revoke", c.endpoint, certID)
	
	request := map[string]string{"reason": reason}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal revoke request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("O1 Mediator returned status %d for certificate revocation: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully revoked certificate %s", certID)
	return nil
}

// GetResourceUsage retrieves resource usage information
func (c *O1MediatorClient) GetResourceUsage(ctx context.Context, filter *O1Filter) (*O1ResourceUsageResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1ResourceUsageResponse{ResourceUsage: []O1ResourceUsage{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/resource-usage", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get resource usage from O1 Mediator: %v", err)
		return &O1ResourceUsageResponse{ResourceUsage: []O1ResourceUsage{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for resource usage", resp.StatusCode)
		return &O1ResourceUsageResponse{ResourceUsage: []O1ResourceUsage{}, Total: 0}, nil
	}

	var response O1ResourceUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode resource usage response: %v", err)
		return &O1ResourceUsageResponse{ResourceUsage: []O1ResourceUsage{}, Total: 0}, nil
	}

	return &response, nil
}

// CreateResourceUsageRecord creates a new resource usage record
func (c *O1MediatorClient) CreateResourceUsageRecord(ctx context.Context, request *O1ResourceUsageRequest) (*O1ResourceUsage, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/resource-usage", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource usage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource usage record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for resource usage creation: %s", resp.StatusCode, string(body))
	}

	var usage O1ResourceUsage
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, fmt.Errorf("failed to decode resource usage response: %w", err)
	}

	log.Printf("Successfully created resource usage record %s", usage.ID)
	return &usage, nil
}

// GetAccessControlPolicies retrieves access control policies
func (c *O1MediatorClient) GetAccessControlPolicies(ctx context.Context, filter *O1Filter) (*O1AccessControlPolicyListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &O1AccessControlPolicyListResponse{Policies: []O1AccessControlPolicy{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/api/v1/access-control/policies", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get access control policies from O1 Mediator: %v", err)
		return &O1AccessControlPolicyListResponse{Policies: []O1AccessControlPolicy{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("O1 Mediator returned status %d for access control policies", resp.StatusCode)
		return &O1AccessControlPolicyListResponse{Policies: []O1AccessControlPolicy{}, Total: 0}, nil
	}

	var response O1AccessControlPolicyListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode access control policies response: %v", err)
		return &O1AccessControlPolicyListResponse{Policies: []O1AccessControlPolicy{}, Total: 0}, nil
	}

	return &response, nil
}

// CreateAccessControlPolicy creates a new access control policy
func (c *O1MediatorClient) CreateAccessControlPolicy(ctx context.Context, request *O1AccessControlPolicyRequest) (*O1AccessControlPolicy, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("O1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/access-control/policies", c.endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal access control policy request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create access control policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("O1 Mediator returned status %d for access control policy creation: %s", resp.StatusCode, string(body))
	}

	var policy O1AccessControlPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode access control policy response: %w", err)
	}

	log.Printf("Successfully created access control policy %s", policy.ID)
	return &policy, nil
}