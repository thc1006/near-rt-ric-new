/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

// E2TClient provides client interface for E2 Termination component
type E2TClient struct {
	conn         *grpc.ClientConn
	httpClient   *http.Client
	endpoint     string
	sctpManager  *SCTPConnectionManager
}

// E2TStats represents statistics from E2 Termination
type E2TStats struct {
	ActiveConnections    uint32            `json:"activeConnections"`
	TotalConnections     uint32            `json:"totalConnections"`
	MessagesReceived     uint64            `json:"messagesReceived"`
	MessagesSent         uint64            `json:"messagesSent"`
	SetupRequests        uint64            `json:"setupRequests"`
	SetupResponses       uint64            `json:"setupResponses"`
	SetupFailures        uint64            `json:"setupFailures"`
	ConfigUpdates        uint64            `json:"configUpdates"`
	ResetRequests        uint64            `json:"resetRequests"`
	ErrorIndications     uint64            `json:"errorIndications"`
	ConnectionsByState   map[string]uint32 `json:"connectionsByState"`
	LastUpdated          time.Time         `json:"lastUpdated"`
}

// E2SetupResponse represents the response to an E2 setup request
type E2SetupResponse struct {
	TransactionID        uint32                    `json:"transactionId"`
	GlobalRICID          GlobalRICID               `json:"globalRicId"`
	RANFunctionsAccepted []RANFunctionAccepted     `json:"ranFunctionsAccepted"`
	RANFunctionsRejected []RANFunctionRejected     `json:"ranFunctionsRejected"`
	E2NodeComponentConfigUpdateAckList []E2NodeComponentConfigUpdateAck `json:"e2NodeComponentConfigUpdateAckList"`
}

// GlobalRICID represents the global RIC identifier
type GlobalRICID struct {
	PlmnID string `json:"plmnId"`
	RicID  string `json:"ricId"`
}

// RANFunctionAccepted represents an accepted RAN function
type RANFunctionAccepted struct {
	RANFunctionID uint32 `json:"ranFunctionId"`
	RANFunctionRevision uint32 `json:"ranFunctionRevision"`
}

// RANFunctionRejected represents a rejected RAN function
type RANFunctionRejected struct {
	RANFunctionID uint32 `json:"ranFunctionId"`
	Cause         string `json:"cause"`
}

// E2NodeComponentConfigUpdateAck represents acknowledgment of component config update
type E2NodeComponentConfigUpdateAck struct {
	E2NodeComponentInterfaceType string `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            string `json:"e2NodeComponentId"`
	UpdateOutcome                string `json:"updateOutcome"`
}

// E2APMessage represents an E2AP protocol message
type E2APMessage struct {
	MessageType    string    `json:"messageType"`
	ProcedureCode  uint8     `json:"procedureCode"`
	Criticality    string    `json:"criticality"`
	TransactionID  uint32    `json:"transactionId"`
	Payload        []byte    `json:"payload"`
	Timestamp      time.Time `json:"timestamp"`
	SourceAddress  string    `json:"sourceAddress"`
	DestAddress    string    `json:"destAddress"`
	AssociationID  string    `json:"associationId"`
}

// NewE2TClient creates a new E2 Termination client
func NewE2TClient(conn *grpc.ClientConn, httpClient *http.Client, endpoint string, sctpManager *SCTPConnectionManager) *E2TClient {
	return &E2TClient{
		conn:        conn,
		httpClient:  httpClient,
		endpoint:    endpoint,
		sctpManager: sctpManager,
	}
}

// GetStats retrieves statistics from E2 Termination
func (c *E2TClient) GetStats(ctx context.Context) (*E2TStats, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &E2TStats{
			ConnectionsByState: make(map[string]uint32),
			LastUpdated:        time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/v1/stats", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get stats from E2T: %v", err)
		return &E2TStats{
			ConnectionsByState: make(map[string]uint32),
			LastUpdated:        time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2T returned status %d for stats", resp.StatusCode)
		return &E2TStats{
			ConnectionsByState: make(map[string]uint32),
			LastUpdated:        time.Now(),
		}, nil
	}

	var stats E2TStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode E2T stats response: %v", err)
		return &E2TStats{
			ConnectionsByState: make(map[string]uint32),
			LastUpdated:        time.Now(),
		}, nil
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// GetConnections retrieves active E2 connections from E2T
func (c *E2TClient) GetConnections(ctx context.Context) ([]*SCTPAssociation, error) {
	if c.sctpManager != nil {
		return c.sctpManager.GetAssociations(), nil
	}

	if c.httpClient == nil || c.endpoint == "" {
		return []*SCTPAssociation{}, nil
	}

	url := fmt.Sprintf("%s/v1/connections", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get connections from E2T: %v", err)
		return []*SCTPAssociation{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2T returned status %d for connections", resp.StatusCode)
		return []*SCTPAssociation{}, nil
	}

	var connections []*SCTPAssociation
	if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
		log.Printf("Failed to decode E2T connections response: %v", err)
		return []*SCTPAssociation{}, nil
	}

	return connections, nil
}

// SendE2APMessage sends an E2AP message through E2T
func (c *E2TClient) SendE2APMessage(ctx context.Context, message *E2APMessage) error {
	if c.sctpManager != nil {
		// Send via SCTP manager
		return c.sctpManager.SendMessage(message.AssociationID, message.Payload, 0)
	}

	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("E2T client not configured")
	}

	url := fmt.Sprintf("%s/v1/messages", c.endpoint)
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal E2AP message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send E2AP message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("E2T returned status %d for message send", resp.StatusCode)
	}

	log.Printf("Successfully sent E2AP message type %s", message.MessageType)
	return nil
}

// ProcessE2SetupRequest processes an E2 setup request
func (c *E2TClient) ProcessE2SetupRequest(ctx context.Context, setupReq *E2SetupRequest) (*E2SetupResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("E2T client not configured")
	}

	url := fmt.Sprintf("%s/v1/setup", c.endpoint)
	
	jsonData, err := json.Marshal(setupReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal E2 setup request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to process E2 setup request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("E2T returned status %d for setup request", resp.StatusCode)
	}

	var setupResp E2SetupResponse
	if err := json.NewDecoder(resp.Body).Decode(&setupResp); err != nil {
		return nil, fmt.Errorf("failed to decode E2 setup response: %w", err)
	}

	log.Printf("Successfully processed E2 setup request for transaction %d", setupReq.TransactionID)
	return &setupResp, nil
}

// ResetE2Node sends an E2 reset request to a node
func (c *E2TClient) ResetE2Node(ctx context.Context, nodeID string, cause string) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("E2T client not configured")
	}

	url := fmt.Sprintf("%s/v1/nodes/%s/reset", c.endpoint, nodeID)
	
	resetReq := map[string]interface{}{
		"nodeId": nodeID,
		"cause":  cause,
	}

	jsonData, err := json.Marshal(resetReq)
	if err != nil {
		return fmt.Errorf("failed to marshal reset request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send reset request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("E2T returned status %d for reset request", resp.StatusCode)
	}

	log.Printf("Successfully sent reset request to node %s", nodeID)
	return nil
}

// GetE2APMessages retrieves recent E2AP messages
func (c *E2TClient) GetE2APMessages(ctx context.Context, limit uint32) ([]*E2APMessage, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return []*E2APMessage{}, nil
	}

	url := fmt.Sprintf("%s/v1/messages?limit=%d", c.endpoint, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get E2AP messages from E2T: %v", err)
		return []*E2APMessage{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2T returned status %d for messages", resp.StatusCode)
		return []*E2APMessage{}, nil
	}

	var messages []*E2APMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		log.Printf("Failed to decode E2AP messages response: %v", err)
		return []*E2APMessage{}, nil
	}

	return messages, nil
}

// IsConnected checks if the E2T client is connected
func (c *E2TClient) IsConnected() bool {
	if c.conn != nil {
		return c.conn.GetState().String() == "READY"
	}
	return c.httpClient != nil && c.endpoint != ""
}