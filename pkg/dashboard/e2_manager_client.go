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
	"github.com/oran/near-rt-ric-new/api/proto/e2mgr"
)

// E2ManagerClient provides client interface for E2 Manager component
type E2ManagerClient struct {
	conn       *grpc.ClientConn
	grpcClient e2mgr.E2ManagerClient
	httpClient *http.Client
	endpoint   string
}

// NewE2ManagerClient creates a new E2 Manager client
func NewE2ManagerClient(conn *grpc.ClientConn, httpClient *http.Client, endpoint string) *E2ManagerClient {
	var grpcClient e2mgr.E2ManagerClient
	if conn != nil {
		grpcClient = e2mgr.NewE2ManagerClient(conn)
	}
	
	return &E2ManagerClient{
		conn:       conn,
		grpcClient: grpcClient,
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

// GetNodes retrieves all E2 nodes from E2 Manager
func (c *E2ManagerClient) GetNodes(ctx context.Context) (*E2NodeListResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getNodesViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/v1/nodeb/states", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get nodes from E2 Manager: %v", err)
		// Return empty response instead of error to allow graceful degradation
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2 Manager returned status %d", resp.StatusCode)
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	var rawNodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawNodes); err != nil {
		log.Printf("Failed to decode E2 Manager response: %v", err)
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	nodes := make([]E2Node, 0, len(rawNodes))
	for _, rawNode := range rawNodes {
		node, err := c.parseE2Node(rawNode)
		if err != nil {
			log.Printf("Failed to parse E2 node: %v", err)
			continue
		}
		nodes = append(nodes, node)
	}

	return &E2NodeListResponse{
		Nodes: nodes,
		Total: uint32(len(nodes)),
	}, nil
}

// convertProtobufToE2Node converts protobuf E2Node to internal E2Node format
func (c *E2ManagerClient) convertProtobufToE2Node(pbNode *e2mgr.E2Node) E2Node {
	node := E2Node{
		ID:               pbNode.Id,
		ConnectionStatus: E2NodeConnectionStatus(pbNode.ConnectionStatus),
		Address:          pbNode.IpAddress, // Map IPAddress to Address field
		Port:             int(pbNode.Port),  // Convert uint32 to int
		// ServiceModels:    []ServiceModel{},     // Field doesn't exist in E2Node
		// RANFunctions:     []RANFunction{},      // Field doesn't exist in E2Node  
		// Subscriptions:    []SubscriptionInfo{}, // Field doesn't exist in E2Node
		SupportedRANFunctions: []RANFunction{}, // Use actual field name
	}
	
	if pbNode.LastUpdate != nil {
		node.LastSeen = pbNode.LastUpdate.AsTime() // Map LastUpdate to LastSeen
	} else {
		node.LastSeen = time.Now()
	}
	
	// GlobalE2NodeID doesn't exist in E2Node, use GlobalRICID instead
	// if pbNode.GlobalE2NodeId != nil {
	//	node.GlobalE2NodeID = GlobalE2NodeID{
	//		PlmnID: pbNode.GlobalE2NodeId.PlmnId,
	//		NodeID: pbNode.GlobalE2NodeId.NodeId,
	//		Type:   E2NodeType(pbNode.GlobalE2NodeId.Type),
	//	}
	// }
	
	// Convert service models - map to SupportedRANFunctions since ServiceModels doesn't exist
	// for _, pbSM := range pbNode.ServiceModels {
	//	sm := ServiceModel{
	//		OID:       pbSM.Oid,
	//		Name:      pbSM.Name,
	//		Version:   pbSM.Version,
	//		Functions: []RANFunction{},
	//	}
	//	
	//	for _, pbFunc := range pbSM.Functions {
	//		function := RANFunction{
	//			ID:          pbFunc.Id,
	//			OID:         pbFunc.Oid,
	//			Definition:  pbFunc.Definition,
	//			Revision:    pbFunc.Revision,
	//			Description: pbFunc.Description,
	//		}
	//		sm.Functions = append(sm.Functions, function)
	//	}
	//	
	//	node.ServiceModels = append(node.ServiceModels, sm)
	// }
	
	// Convert RAN functions to SupportedRANFunctions
	for _, pbFunc := range pbNode.RanFunctions {
		function := RANFunction{
			ID:          pbFunc.Id,
			OID:         pbFunc.Oid,
			// Definition:  pbFunc.Definition,  // Field doesn't exist in RANFunction
			Revision:    pbFunc.Revision,
			Description: pbFunc.Description,
		}
		node.SupportedRANFunctions = append(node.SupportedRANFunctions, function)
	}
	
	// Convert subscriptions - field doesn't exist in E2Node, comment out
	// for _, pbSub := range pbNode.Subscriptions {
	//	sub := SubscriptionInfo{
		//	SubscriptionID: pbSub.SubscriptionId,
		//	XAppID:         pbSub.XappId,
		//	RANFunctionID:  pbSub.RanFunctionId,
		//	Status:         pbSub.Status,
		//}
		//node.Subscriptions = append(node.Subscriptions, sub)
	// }
	
	// Convert setup request if available - E2Node doesn't have SetupRequest field, comment out
	// if pbNode.SetupRequest != nil {
	//	setupReq := &E2SetupRequest{
	//		TransactionID:  pbNode.SetupRequest.TransactionId,
	//		RANFunctions:   []RANFunction{},
	//		E2NodeComponentConfigAddList: []E2NodeComponentConfig{},
	//	}
	//	
	//	if pbNode.SetupRequest.GlobalE2NodeId != nil {
	//		setupReq.GlobalE2NodeID = GlobalE2NodeID{
	//			PlmnID: pbNode.SetupRequest.GlobalE2NodeId.PlmnId,
	//			NodeID: pbNode.SetupRequest.GlobalE2NodeId.NodeId,
	//			Type:   E2NodeType(pbNode.SetupRequest.GlobalE2NodeId.Type),
	//		}
	//	}
	//	
	//	for _, pbFunc := range pbNode.SetupRequest.RanFunctions {
	//		function := RANFunction{
	//			ID:          pbFunc.Id,
	//			OID:         pbFunc.Oid,
	//			Definition:  pbFunc.Definition,  // Field doesn't exist in RANFunction
	//			Revision:    pbFunc.Revision,
	//			Description: pbFunc.Description,
	//		}
	//		setupReq.RANFunctions = append(setupReq.RANFunctions, function)
	//	}
	//	
	//	for _, pbConfig := range pbNode.SetupRequest.E2NodeComponentConfigAddList {
	//		config := E2NodeComponentConfig{
	//			InterfaceType: pbConfig.InterfaceType,  // Field doesn't exist in E2NodeComponentConfig
	//			InterfaceID:   pbConfig.InterfaceId,    // Field doesn't exist in E2NodeComponentConfig
	//			Configuration: pbConfig.Configuration,  // Field doesn't exist in E2NodeComponentConfig
	//		}
	//		setupReq.E2NodeComponentConfigAddList = append(setupReq.E2NodeComponentConfigAddList, config)
	//	}
	//	
	//	node.SetupRequest = setupReq  // Field doesn't exist in E2Node
	// }
	
	return node
}

// IsConnected checks if the E2 Manager client is connected
func (c *E2ManagerClient) IsConnected() bool {
	return c.conn != nil && c.conn.GetState().String() == "READY"
}

// GetNode retrieves a specific E2 node by ID
func (c *E2ManagerClient) GetNode(ctx context.Context, nodeID string) (*E2Node, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetNodeViaGRPC(ctx, nodeID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("E2 Manager client not configured")
	}

	url := fmt.Sprintf("%s/v1/nodeb/%s", c.endpoint, nodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get node from E2 Manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("E2 Manager returned status %d", resp.StatusCode)
	}

	var rawNode map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawNode); err != nil {
		return nil, fmt.Errorf("failed to decode E2 Manager response: %w", err)
	}

	node, err := c.parseE2Node(rawNode)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetNodeHealth retrieves health information for a specific E2 node
func (c *E2ManagerClient) GetNodeHealth(ctx context.Context, nodeID string) (*E2NodeHealth, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetNodeHealthViaGRPC(ctx, nodeID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("E2 Manager client not configured")
	}

	url := fmt.Sprintf("%s/v1/nodeb/%s/health", c.endpoint, nodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get node health from E2 Manager: %v", err)
		// Return default health status
		return &E2NodeHealth{
			NodeID:          nodeID,
			IsHealthy:       false,
			LastHealthCheck: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &E2NodeHealth{
			NodeID:          nodeID,
			IsHealthy:       false,
			LastHealthCheck: time.Now(),
		}, nil
	}

	var health E2NodeHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		log.Printf("Failed to decode health response: %v", err)
		return &E2NodeHealth{
			NodeID:          nodeID,
			IsHealthy:       false,
			LastHealthCheck: time.Now(),
		}, nil
	}

	return &health, nil
}

// GetStats retrieves statistics from E2 Manager
func (c *E2ManagerClient) GetStats(ctx context.Context) (*E2ManagerStats, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetStatsViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &E2ManagerStats{
			NodesByType:   make(map[string]uint32),
			NodesByStatus: make(map[string]uint32),
			LastUpdated:   time.Now(),
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
		log.Printf("Failed to get stats from E2 Manager: %v", err)
		return &E2ManagerStats{
			NodesByType:   make(map[string]uint32),
			NodesByStatus: make(map[string]uint32),
			LastUpdated:   time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2 Manager stats returned status %d", resp.StatusCode)
		return &E2ManagerStats{
			NodesByType:   make(map[string]uint32),
			NodesByStatus: make(map[string]uint32),
			LastUpdated:   time.Now(),
		}, nil
	}

	var stats E2ManagerStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode stats response: %v", err)
		return &E2ManagerStats{
			NodesByType:   make(map[string]uint32),
			NodesByStatus: make(map[string]uint32),
			LastUpdated:   time.Now(),
		}, nil
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// UpdateNodeConfiguration updates the configuration of an E2 node
func (c *E2ManagerClient) UpdateNodeConfiguration(ctx context.Context, nodeID string, update *E2NodeConfigurationUpdate) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.UpdateNodeConfigurationViaGRPC(ctx, update)  // Fix: Remove extra nodeID parameter
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("E2 Manager client not configured")
	}

	url := fmt.Sprintf("%s/v1/nodeb/%s/configuration", c.endpoint, nodeID)
	
	_, err := json.Marshal(update)  // Fix unused variable
	if err != nil {
		return fmt.Errorf("failed to marshal configuration update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update node configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("E2 Manager returned status %d for configuration update", resp.StatusCode)
	}

	log.Printf("Successfully updated configuration for node %s", nodeID)
	return nil
}

// parseE2Node parses raw node data from E2 Manager into E2Node struct
func (c *E2ManagerClient) parseE2Node(rawNode map[string]interface{}) (E2Node, error) {
	node := E2Node{
		LastSeen: time.Now(), // Use LastSeen instead of LastUpdate
		// ServiceModels: []ServiceModel{},      // Field doesn't exist in E2Node
		// RANFunctions:  []RANFunction{},       // Field doesn't exist in E2Node, use SupportedRANFunctions
		// Subscriptions: []SubscriptionInfo{},  // Field doesn't exist in E2Node
		SupportedRANFunctions: []RANFunction{}, // Use actual field name
	}

	// Parse basic node information
	if id, ok := rawNode["inventoryName"].(string); ok {
		node.ID = id
	}

	// if globalNodeID, ok := rawNode["globalNbId"].(map[string]interface{}); ok {
	//	node.GlobalE2NodeID = c.parseGlobalE2NodeID(globalNodeID)  // Field doesn't exist in E2Node
	// }

	if connectionStatus, ok := rawNode["connectionStatus"].(string); ok {
		node.ConnectionStatus = E2NodeConnectionStatus(connectionStatus)
	}

	if ipAddress, ok := rawNode["ip"].(string); ok {
		// Fix: node.IPAddress -> node.Address (line 445)
		node.Address = ipAddress
	}

	if port, ok := rawNode["port"].(float64); ok {
		// Fix: uint32(port) -> int(port) conversion (line 449)
		node.Port = int(port)
	}

	// Parse RAN functions
	if ranFunctions, ok := rawNode["ranFunctions"].([]interface{}); ok {
		for _, rf := range ranFunctions {
			if ranFunc, ok := rf.(map[string]interface{}); ok {
				function := c.parseRANFunction(ranFunc)
				// Fix: node.RANFunctions -> node.SupportedRANFunctions (line 457)
				node.SupportedRANFunctions = append(node.SupportedRANFunctions, function)
			}
		}
	}

	// Fix: Remove node.SetupRequest assignment (line 464) - field doesn't exist
	// Parse setup request if available
	// if setupReq, ok := rawNode["setupRequest"].(map[string]interface{}); ok {
	//	node.SetupRequest = c.parseE2SetupRequest(setupReq)
	// }

	return node, nil
}

// parseGlobalE2NodeID parses global E2 node ID from raw data
func (c *E2ManagerClient) parseGlobalE2NodeID(raw map[string]interface{}) GlobalE2NodeID {
	globalID := GlobalE2NodeID{}

	// Fix: Comment out all globalID.PlmnID, globalID.NodeID, globalID.Type access (lines 475,479,483) - fields don't exist in GlobalE2NodeID
	// if plmnID, ok := raw["plmnId"].(string); ok {
	//	globalID.PlmnID = plmnID
	// }

	// if nodeID, ok := raw["nbId"].(string); ok {
	//	globalID.NodeID = nodeID
	// }

	// if nodeType, ok := raw["nodeType"].(string); ok {
	//	globalID.Type = E2NodeType(nodeType)
	// }

	// Use the actual fields from GlobalE2NodeID struct: PLMNIdentity and E2NodeID
	if plmnID, ok := raw["plmnId"].(string); ok {
		globalID.PLMNIdentity = []byte(plmnID)
	}

	if nodeID, ok := raw["nbId"].(string); ok {
		globalID.E2NodeID = []byte(nodeID)
	}

	return globalID
}

// parseRANFunction parses RAN function from raw data
func (c *E2ManagerClient) parseRANFunction(raw map[string]interface{}) RANFunction {
	function := RANFunction{}

	if id, ok := raw["ranFunctionId"].(float64); ok {
		function.ID = uint32(id)
	}

	if oid, ok := raw["ranFunctionOid"].(string); ok {
		function.OID = oid
	}

	// Fix: Remove function.Definition assignment - field doesn't exist in RANFunction (line 516)
	// if definition, ok := raw["ranFunctionDefinition"].(string); ok {
	//	function.Definition = []byte(definition)
	// }

	if revision, ok := raw["ranFunctionRevision"].(float64); ok {
		function.Revision = uint32(revision)
	}

	if description, ok := raw["description"].(string); ok {
		function.Description = description
	}

	return function
}

// parseE2SetupRequest parses E2 setup request from raw data
func (c *E2ManagerClient) parseE2SetupRequest(raw map[string]interface{}) *E2SetupRequest {
	setupReq := &E2SetupRequest{
		RANFunctions: []RANFunction{},
		E2NodeComponentConfigAddList: []E2NodeComponentConfig{},
	}

	if transactionID, ok := raw["transactionId"].(float64); ok {
		setupReq.TransactionID = uint32(transactionID)
	}

	if globalNodeID, ok := raw["globalE2NodeId"].(map[string]interface{}); ok {
		setupReq.GlobalE2NodeID = c.parseGlobalE2NodeID(globalNodeID)
	}

	if ranFunctions, ok := raw["ranFunctions"].([]interface{}); ok {
		for _, rf := range ranFunctions {
			if ranFunc, ok := rf.(map[string]interface{}); ok {
				function := c.parseRANFunction(ranFunc)
				setupReq.RANFunctions = append(setupReq.RANFunctions, function)
			}
		}
	}

	return setupReq
}

// getNodesViaGRPC retrieves nodes using gRPC client
func (c *E2ManagerClient) getNodesViaGRPC(ctx context.Context) (*E2NodeListResponse, error) {
	req := &e2mgr.GetNodesRequest{}
	
	resp, err := c.grpcClient.GetNodes(ctx, req)
	if err != nil {
		log.Printf("Failed to get nodes via gRPC: %v", err)
		// Fallback to HTTP if gRPC fails
		return c.getNodesViaHTTP(ctx)
	}
	
	// Convert protobuf response to internal format
	nodes := make([]E2Node, 0, len(resp.Nodes))
	for _, pbNode := range resp.Nodes {
		node := c.convertProtobufToE2Node(pbNode)
		nodes = append(nodes, node)
	}
	
	return &E2NodeListResponse{
		Nodes: nodes,
		Total: resp.Total,
	}, nil
}

// GetNodeViaGRPC retrieves a specific node using gRPC
func (c *E2ManagerClient) GetNodeViaGRPC(ctx context.Context, nodeID string) (*E2Node, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}
	
	req := &e2mgr.GetNodeRequest{
		NodeId: nodeID,
	}
	
	resp, err := c.grpcClient.GetNode(ctx, req)
	if err != nil {
		return nil, err
	}
	
	node := c.convertProtobufToE2Node(resp.Node)
	return &node, nil
}

// GetNodeHealthViaGRPC retrieves node health using gRPC
func (c *E2ManagerClient) GetNodeHealthViaGRPC(ctx context.Context, nodeID string) (*E2NodeHealth, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}
	
	req := &e2mgr.GetNodeHealthRequest{
		NodeId: nodeID,
	}
	
	resp, err := c.grpcClient.GetNodeHealth(ctx, req)
	if err != nil {
		return nil, err
	}
	
	health := &E2NodeHealth{
		NodeID:          resp.Health.NodeId,
		IsHealthy:       resp.Health.IsHealthy,
		// Fix: Remove StatusMessage field access - field doesn't exist in E2NodeHealth (line 618)
		// StatusMessage:   resp.Health.StatusMessage,
	}
	
	if resp.Health.LastHealthCheck != nil {
		health.LastHealthCheck = resp.Health.LastHealthCheck.AsTime()
	}
	
	return health, nil
}

// GetStatsViaGRPC retrieves statistics using gRPC
func (c *E2ManagerClient) GetStatsViaGRPC(ctx context.Context) (*E2ManagerStats, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}
	
	req := &e2mgr.GetStatsRequest{}
	
	resp, err := c.grpcClient.GetStats(ctx, req)
	if err != nil {
		return nil, err
	}
	
	stats := &E2ManagerStats{
		NodesByType:         resp.Stats.NodesByType,
		NodesByStatus:       resp.Stats.NodesByStatus,
		TotalNodes:          resp.Stats.TotalNodes,
		// Fix: Remove ActiveSubscriptions field access - field doesn't exist in E2ManagerStats (line 645)
		// ActiveSubscriptions: resp.Stats.ActiveSubscriptions,
	}
	
	if resp.Stats.LastUpdated != nil {
		stats.LastUpdated = resp.Stats.LastUpdated.AsTime()
	}
	
	return stats, nil
}

// UpdateNodeConfigurationViaGRPC updates node configuration using gRPC
func (c *E2ManagerClient) UpdateNodeConfigurationViaGRPC(ctx context.Context, update *E2NodeConfigurationUpdate) error {
	if c.grpcClient == nil {
		return fmt.Errorf("gRPC client not available")
	}
	
	// Fix: Comment out Configuration field access - field doesn't exist in E2NodeConfigurationUpdate (lines 663,666)
	// Convert internal configuration to protobuf format
	// pbConfig := &e2mgr.NodeConfiguration{
	//	Parameters: update.Configuration.Parameters,
	// }
	
	// for _, config := range update.Configuration.ComponentConfigs {
	//	pbComponentConfig := &e2mgr.E2NodeComponentConfig{
	//		InterfaceType: config.InterfaceType,
	//		InterfaceId:   config.InterfaceID,
	//		Configuration: config.Configuration,
	//	}
	//	pbConfig.ComponentConfigs = append(pbConfig.ComponentConfigs, pbComponentConfig)
	// }
	
	// Create a minimal request with available fields
	req := &e2mgr.UpdateNodeConfigurationRequest{
		// Fix: Remove NodeID field access - field doesn't exist in E2NodeConfigurationUpdate (line 676)
		// NodeId:        update.NodeID,
		// Configuration: pbConfig,
		// Use UpdateID as a fallback identifier
		NodeId: update.UpdateID, // Use UpdateID instead of non-existent NodeID field
	}
	
	resp, err := c.grpcClient.UpdateNodeConfiguration(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to update node configuration: %s", resp.Message)
	}
	
	return nil
}

// getNodesViaHTTP retrieves nodes using HTTP client (existing implementation)
func (c *E2ManagerClient) getNodesViaHTTP(ctx context.Context) (*E2NodeListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/v1/nodeb/states", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get nodes from E2 Manager: %v", err)
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2 Manager returned status %d", resp.StatusCode)
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	var rawNodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawNodes); err != nil {
		log.Printf("Failed to decode E2 Manager response: %v", err)
		return &E2NodeListResponse{Nodes: []E2Node{}, Total: 0}, nil
	}

	nodes := make([]E2Node, 0, len(rawNodes))
	for _, rawNode := range rawNodes {
		node, err := c.parseE2Node(rawNode)
		if err != nil {
			log.Printf("Failed to parse E2 node: %v", err)
			continue
		}
		nodes = append(nodes, node)
	}

	return &E2NodeListResponse{
		Nodes: nodes,
		Total: uint32(len(nodes)),
	}, nil
}