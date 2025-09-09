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
	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/oran/near-rt-ric-new/api/proto/rtmgr"
)

// RoutingManagerClient provides client interface for Routing Manager component
type RoutingManagerClient struct {
	conn       *grpc.ClientConn
	grpcClient rtmgr.RoutingManagerClient
	httpClient *http.Client
	endpoint   string
}

// NewRoutingManagerClient creates a new Routing Manager client
func NewRoutingManagerClient(conn *grpc.ClientConn, httpClient *http.Client, endpoint string) *RoutingManagerClient {
	var grpcClient rtmgr.RoutingManagerClient
	if conn != nil {
		grpcClient = rtmgr.NewRoutingManagerClient(conn)
	}
	
	return &RoutingManagerClient{
		conn:       conn,
		grpcClient: grpcClient,
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

// IsConnected checks if the Routing Manager client is connected
func (c *RoutingManagerClient) IsConnected() bool {
	return c.conn != nil && c.conn.GetState().String() == "READY"
}

// CreateRoute creates a new route
func (c *RoutingManagerClient) CreateRoute(ctx context.Context, route *Route) (*RouteResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.createRouteViaGRPC(ctx, route)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/routes", c.endpoint)
	
	jsonData, err := json.Marshal(route)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Routing Manager returned status %d", resp.StatusCode)
	}

	var routeResp RouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&routeResp); err != nil {
		return nil, fmt.Errorf("failed to decode route response: %w", err)
	}

	log.Printf("Successfully created route %s", routeResp.RouteID)
	return &routeResp, nil
}

// GetRoutes retrieves routes based on filter criteria
func (c *RoutingManagerClient) GetRoutes(ctx context.Context, filter *RouteFilter) (*RouteListResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getRoutesViaGRPC(ctx, filter)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &RouteListResponse{Routes: []Route{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/routes", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get routes from Routing Manager: %v", err)
		return &RouteListResponse{Routes: []Route{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Routing Manager returned status %d", resp.StatusCode)
		return &RouteListResponse{Routes: []Route{}, Total: 0}, nil
	}

	var rawRoutes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawRoutes); err != nil {
		log.Printf("Failed to decode routes response: %v", err)
		return &RouteListResponse{Routes: []Route{}, Total: 0}, nil
	}

	routes := make([]Route, 0, len(rawRoutes))
	for _, rawRoute := range rawRoutes {
		route, err := c.parseRoute(rawRoute)
		if err != nil {
			log.Printf("Failed to parse route: %v", err)
			continue
		}
		routes = append(routes, route)
	}

	return &RouteListResponse{
		Routes: routes,
		Total:  uint32(len(routes)),
	}, nil
}

// GetRoute retrieves a specific route by ID
func (c *RoutingManagerClient) GetRoute(ctx context.Context, routeID string) (*Route, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getRouteViaGRPC(ctx, routeID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/routes/%s", c.endpoint, routeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("route %s not found", routeID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Routing Manager returned status %d", resp.StatusCode)
	}

	var rawRoute map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawRoute); err != nil {
		return nil, fmt.Errorf("failed to decode route response: %w", err)
	}

	return c.parseRoute(rawRoute)
}

// UpdateRoute updates an existing route
func (c *RoutingManagerClient) UpdateRoute(ctx context.Context, routeID string, route *Route) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.updateRouteViaGRPC(ctx, routeID, route)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/routes/%s", c.endpoint, routeID)
	
	jsonData, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Routing Manager returned status %d for update", resp.StatusCode)
	}

	log.Printf("Successfully updated route %s", routeID)
	return nil
}

// DeleteRoute deletes a route
func (c *RoutingManagerClient) DeleteRoute(ctx context.Context, routeID string) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.deleteRouteViaGRPC(ctx, routeID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/routes/%s", c.endpoint, routeID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Routing Manager returned status %d for deletion", resp.StatusCode)
	}

	log.Printf("Successfully deleted route %s", routeID)
	return nil
}

// GetRoutingTable retrieves the current routing table
func (c *RoutingManagerClient) GetRoutingTable(ctx context.Context) (*RoutingTable, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getRoutingTableViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &RoutingTable{
			Entries:     []RouteEntry{},
			Version:     0,
			LastUpdated: time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/routing-table", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get routing table from Routing Manager: %v", err)
		return &RoutingTable{
			Entries:     []RouteEntry{},
			Version:     0,
			LastUpdated: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Routing Manager routing table returned status %d", resp.StatusCode)
		return &RoutingTable{
			Entries:     []RouteEntry{},
			Version:     0,
			LastUpdated: time.Now(),
		}, nil
	}

	var routingTable RoutingTable
	if err := json.NewDecoder(resp.Body).Decode(&routingTable); err != nil {
		log.Printf("Failed to decode routing table response: %v", err)
		return &RoutingTable{
			Entries:     []RouteEntry{},
			Version:     0,
			LastUpdated: time.Now(),
		}, nil
	}

	return &routingTable, nil
}

// RegisterXApp registers an xApp with the routing manager
func (c *RoutingManagerClient) RegisterXApp(ctx context.Context, xappInfo *XAppInfo) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.registerXAppViaGRPC(ctx, xappInfo)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/xapps", c.endpoint)
	
	jsonData, err := json.Marshal(xappInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal xApp info: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register xApp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Routing Manager returned status %d for xApp registration", resp.StatusCode)
	}

	log.Printf("Successfully registered xApp %s", xappInfo.Name)
	return nil
}

// UnregisterXApp unregisters an xApp from the routing manager
func (c *RoutingManagerClient) UnregisterXApp(ctx context.Context, xappName string) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.unregisterXAppViaGRPC(ctx, xappName)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Routing Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/xapps/%s", c.endpoint, xappName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unregister xApp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Routing Manager returned status %d for xApp unregistration", resp.StatusCode)
	}

	log.Printf("Successfully unregistered xApp %s", xappName)
	return nil
}

// GetXApps retrieves registered xApps
func (c *RoutingManagerClient) GetXApps(ctx context.Context) (*XAppListResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getXAppsViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &XAppListResponse{XApps: []XAppInfo{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/xapps", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get xApps from Routing Manager: %v", err)
		return &XAppListResponse{XApps: []XAppInfo{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Routing Manager returned status %d", resp.StatusCode)
		return &XAppListResponse{XApps: []XAppInfo{}, Total: 0}, nil
	}

	var rawXApps []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawXApps); err != nil {
		log.Printf("Failed to decode xApps response: %v", err)
		return &XAppListResponse{XApps: []XAppInfo{}, Total: 0}, nil
	}

	xapps := make([]XAppInfo, 0, len(rawXApps))
	for _, rawXApp := range rawXApps {
		xapp, err := c.parseXAppInfo(rawXApp)
		if err != nil {
			log.Printf("Failed to parse xApp info: %v", err)
			continue
		}
		xapps = append(xapps, xapp)
	}

	return &XAppListResponse{
		XApps: xapps,
		Total: uint32(len(xapps)),
	}, nil
}

// GetStats retrieves statistics from Routing Manager
func (c *RoutingManagerClient) GetStats(ctx context.Context) (*RoutingStats, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getStatsViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &RoutingStats{
			RoutesByMessageType: make(map[string]uint32),
			XAppsByStatus:       make(map[string]uint32),
			LastUpdated:         time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/stats", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get stats from Routing Manager: %v", err)
		return &RoutingStats{
			RoutesByMessageType: make(map[string]uint32),
			XAppsByStatus:       make(map[string]uint32),
			LastUpdated:         time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Routing Manager stats returned status %d", resp.StatusCode)
		return &RoutingStats{
			RoutesByMessageType: make(map[string]uint32),
			XAppsByStatus:       make(map[string]uint32),
			LastUpdated:         time.Now(),
		}, nil
	}

	var stats RoutingStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode stats response: %v", err)
		return &RoutingStats{
			RoutesByMessageType: make(map[string]uint32),
			XAppsByStatus:       make(map[string]uint32),
			LastUpdated:         time.Now(),
		}, nil
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// GetHealth retrieves health information from Routing Manager
func (c *RoutingManagerClient) GetHealth(ctx context.Context) (*RoutingHealth, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.getHealthViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &RoutingHealth{
			IsHealthy:        false,
			StatusMessage:    "Client not configured",
			LastHealthCheck:  time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/health", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get health from Routing Manager: %v", err)
		return &RoutingHealth{
			IsHealthy:        false,
			StatusMessage:    "Connection failed",
			LastHealthCheck:  time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	var health RoutingHealth
	if resp.StatusCode != http.StatusOK {
		health = RoutingHealth{
			IsHealthy:        false,
			StatusMessage:    fmt.Sprintf("HTTP %d", resp.StatusCode),
			LastHealthCheck:  time.Now(),
		}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			log.Printf("Failed to decode health response: %v", err)
			health = RoutingHealth{
				IsHealthy:        false,
				StatusMessage:    "Decode error",
				LastHealthCheck:  time.Now(),
			}
		}
	}

	return &health, nil
}

// gRPC implementation methods

func (c *RoutingManagerClient) createRouteViaGRPC(ctx context.Context, route *Route) (*RouteResponse, error) {
	pbRoute := c.convertRouteToProtobuf(route)
	
	req := &rtmgr.CreateRouteRequest{
		Route: pbRoute,
	}
	
	resp, err := c.grpcClient.CreateRoute(ctx, req)
	if err != nil {
		log.Printf("Failed to create route via gRPC: %v", err)
		return nil, err
	}
	
	return &RouteResponse{
		RouteID: resp.RouteId,
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

func (c *RoutingManagerClient) getRoutesViaGRPC(ctx context.Context, filter *RouteFilter) (*RouteListResponse, error) {
	req := &rtmgr.GetRoutesRequest{}
	
	if filter != nil {
		req.SourceXapp = &filter.SourceXApp
		req.TargetXapp = &filter.TargetXApp
		if filter.MessageType != nil {
			req.MessageType = filter.MessageType
		}
		if filter.Limit != nil {
			req.Limit = filter.Limit
		}
		if filter.Offset != nil {
			req.Offset = filter.Offset
		}
	}
	
	resp, err := c.grpcClient.GetRoutes(ctx, req)
	if err != nil {
		log.Printf("Failed to get routes via gRPC: %v", err)
		return &RouteListResponse{Routes: []Route{}, Total: 0}, nil
	}
	
	routes := make([]Route, 0, len(resp.Routes))
	for _, pbRoute := range resp.Routes {
		route := c.convertProtobufToRoute(pbRoute)
		routes = append(routes, route)
	}
	
	return &RouteListResponse{
		Routes: routes,
		Total:  resp.Total,
	}, nil
}

func (c *RoutingManagerClient) getRouteViaGRPC(ctx context.Context, routeID string) (*Route, error) {
	req := &rtmgr.GetRouteRequest{
		RouteId: routeID,
	}
	
	resp, err := c.grpcClient.GetRoute(ctx, req)
	if err != nil {
		return nil, err
	}
	
	route := c.convertProtobufToRoute(resp.Route)
	return &route, nil
}

func (c *RoutingManagerClient) updateRouteViaGRPC(ctx context.Context, routeID string, route *Route) error {
	pbRoute := c.convertRouteToProtobuf(route)
	
	req := &rtmgr.UpdateRouteRequest{
		RouteId: routeID,
		Route:   pbRoute,
	}
	
	resp, err := c.grpcClient.UpdateRoute(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to update route: %s", resp.Message)
	}
	
	return nil
}

func (c *RoutingManagerClient) deleteRouteViaGRPC(ctx context.Context, routeID string) error {
	req := &rtmgr.DeleteRouteRequest{
		RouteId: routeID,
	}
	
	resp, err := c.grpcClient.DeleteRoute(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to delete route: %s", resp.Message)
	}
	
	return nil
}

func (c *RoutingManagerClient) getRoutingTableViaGRPC(ctx context.Context) (*RoutingTable, error) {
	req := &rtmgr.GetRoutingTableRequest{}
	
	resp, err := c.grpcClient.GetRoutingTable(ctx, req)
	if err != nil {
		log.Printf("Failed to get routing table via gRPC: %v", err)
		return &RoutingTable{
			Entries:     []RouteEntry{},
			Version:     0,
			LastUpdated: time.Now(),
		}, nil
	}
	
	return c.convertProtobufToRoutingTable(resp.RoutingTable), nil
}

func (c *RoutingManagerClient) registerXAppViaGRPC(ctx context.Context, xappInfo *XAppInfo) error {
	pbXAppInfo := c.convertXAppInfoToProtobuf(xappInfo)
	
	req := &rtmgr.RegisterXAppRequest{
		XappInfo: pbXAppInfo,
	}
	
	resp, err := c.grpcClient.RegisterXApp(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to register xApp: %s", resp.Message)
	}
	
	return nil
}

func (c *RoutingManagerClient) unregisterXAppViaGRPC(ctx context.Context, xappName string) error {
	req := &rtmgr.UnregisterXAppRequest{
		XappName: xappName,
	}
	
	resp, err := c.grpcClient.UnregisterXApp(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to unregister xApp: %s", resp.Message)
	}
	
	return nil
}

func (c *RoutingManagerClient) getXAppsViaGRPC(ctx context.Context) (*XAppListResponse, error) {
	req := &rtmgr.GetXAppsRequest{}
	
	resp, err := c.grpcClient.GetXApps(ctx, req)
	if err != nil {
		log.Printf("Failed to get xApps via gRPC: %v", err)
		return &XAppListResponse{XApps: []XAppInfo{}, Total: 0}, nil
	}
	
	xapps := make([]XAppInfo, 0, len(resp.Xapps))
	for _, pbXApp := range resp.Xapps {
		xapp := c.convertProtobufToXAppInfo(pbXApp)
		xapps = append(xapps, xapp)
	}
	
	return &XAppListResponse{
		XApps: xapps,
		Total: resp.Total,
	}, nil
}

func (c *RoutingManagerClient) getStatsViaGRPC(ctx context.Context) (*RoutingStats, error) {
	req := &rtmgr.GetStatsRequest{}
	
	resp, err := c.grpcClient.GetStats(ctx, req)
	if err != nil {
		log.Printf("Failed to get stats via gRPC: %v", err)
		return &RoutingStats{
			RoutesByMessageType: make(map[string]uint32),
			XAppsByStatus:       make(map[string]uint32),
			LastUpdated:         time.Now(),
		}, nil
	}
	
	return c.convertProtobufToRoutingStats(resp.Stats), nil
}

func (c *RoutingManagerClient) getHealthViaGRPC(ctx context.Context) (*RoutingHealth, error) {
	req := &rtmgr.GetHealthRequest{}
	
	resp, err := c.grpcClient.GetHealth(ctx, req)
	if err != nil {
		log.Printf("Failed to get health via gRPC: %v", err)
		return &RoutingHealth{
			IsHealthy:        false,
			StatusMessage:    "gRPC call failed",
			LastHealthCheck:  time.Now(),
		}, nil
	}
	
	return c.convertProtobufToRoutingHealth(resp.Health), nil
}

// Conversion helper methods

func (c *RoutingManagerClient) convertRouteToProtobuf(route *Route) *rtmgr.Route {
	pbRoute := &rtmgr.Route{
		Id:           route.ID,
		SourceXapp:   route.SourceXApp,
		TargetXapp:   route.TargetXApp,
		MessageType:  route.MessageType,
		SubscriptionId: route.SubscriptionID,
	}
	
	if route.Policy != nil {
		pbRoute.Policy = &rtmgr.RoutePolicy{
			Type:       route.Policy.Type,
			Parameters: route.Policy.Parameters,
		}
	}
	
	if !route.CreatedAt.IsZero() {
		pbRoute.CreatedAt = timestamppb.New(route.CreatedAt)
	}
	
	if !route.UpdatedAt.IsZero() {
		pbRoute.UpdatedAt = timestamppb.New(route.UpdatedAt)
	}
	
	return pbRoute
}

func (c *RoutingManagerClient) convertProtobufToRoute(pbRoute *rtmgr.Route) Route {
	route := Route{
		ID:             pbRoute.Id,
		SourceXApp:     pbRoute.SourceXapp,
		TargetXApp:     pbRoute.TargetXapp,
		MessageType:    pbRoute.MessageType,
		SubscriptionID: pbRoute.SubscriptionId,
	}
	
	if pbRoute.Policy != nil {
		route.Policy = &RoutePolicy{
			Type:       pbRoute.Policy.Type,
			Parameters: pbRoute.Policy.Parameters,
		}
	}
	
	if pbRoute.CreatedAt != nil {
		route.CreatedAt = pbRoute.CreatedAt.AsTime()
	}
	
	if pbRoute.UpdatedAt != nil {
		route.UpdatedAt = pbRoute.UpdatedAt.AsTime()
	}
	
	return route
}

func (c *RoutingManagerClient) convertProtobufToRoutingTable(pbTable *rtmgr.RoutingTable) *RoutingTable {
	table := &RoutingTable{
		Version: pbTable.Version,
		Entries: make([]RouteEntry, 0, len(pbTable.Entries)),
	}
	
	if pbTable.LastUpdated != nil {
		table.LastUpdated = pbTable.LastUpdated.AsTime()
	}
	
	for _, pbEntry := range pbTable.Entries {
		entry := RouteEntry{
			MessageType:     pbEntry.MessageType,
			SourceEndpoint:  pbEntry.SourceEndpoint,
			TargetEndpoints: pbEntry.TargetEndpoints,
		}
		
		if pbEntry.Policy != nil {
			entry.Policy = &RoutePolicy{
				Type:       pbEntry.Policy.Type,
				Parameters: pbEntry.Policy.Parameters,
			}
		}
		
		table.Entries = append(table.Entries, entry)
	}
	
	return table
}

func (c *RoutingManagerClient) convertXAppInfoToProtobuf(xappInfo *XAppInfo) *rtmgr.XAppInfo {
	pbXApp := &rtmgr.XAppInfo{
		Name:      xappInfo.Name,
		Version:   xappInfo.Version,
		Namespace: xappInfo.Namespace,
		Endpoints: xappInfo.Endpoints,
		Config:    xappInfo.Config,
		Status:    xappInfo.Status,
	}
	
	if !xappInfo.RegisteredAt.IsZero() {
		pbXApp.RegisteredAt = timestamppb.New(xappInfo.RegisteredAt)
	}
	
	if !xappInfo.LastHeartbeat.IsZero() {
		pbXApp.LastHeartbeat = timestamppb.New(xappInfo.LastHeartbeat)
	}
	
	return pbXApp
}

func (c *RoutingManagerClient) convertProtobufToXAppInfo(pbXApp *rtmgr.XAppInfo) XAppInfo {
	xapp := XAppInfo{
		Name:      pbXApp.Name,
		Version:   pbXApp.Version,
		Namespace: pbXApp.Namespace,
		Endpoints: pbXApp.Endpoints,
		Config:    pbXApp.Config,
		Status:    pbXApp.Status,
	}
	
	if pbXApp.RegisteredAt != nil {
		xapp.RegisteredAt = pbXApp.RegisteredAt.AsTime()
	}
	
	if pbXApp.LastHeartbeat != nil {
		xapp.LastHeartbeat = pbXApp.LastHeartbeat.AsTime()
	}
	
	return xapp
}

func (c *RoutingManagerClient) convertProtobufToRoutingStats(pbStats *rtmgr.RoutingStats) *RoutingStats {
	stats := &RoutingStats{
		TotalRoutes:         pbStats.TotalRoutes,
		ActiveXApps:         pbStats.ActiveXapps,
		RoutesByMessageType: pbStats.RoutesByMessageType,
		XAppsByStatus:       pbStats.XappsByStatus,
		MessagesRouted:      pbStats.MessagesRouted,
		RoutingErrors:       pbStats.RoutingErrors,
	}
	
	if pbStats.LastUpdated != nil {
		stats.LastUpdated = pbStats.LastUpdated.AsTime()
	}
	
	return stats
}

func (c *RoutingManagerClient) convertProtobufToRoutingHealth(pbHealth *rtmgr.RoutingHealth) *RoutingHealth {
	health := &RoutingHealth{
		IsHealthy:         pbHealth.IsHealthy,
		StatusMessage:     pbHealth.StatusMessage,
		ActiveConnections: pbHealth.ActiveConnections,
		FailedRoutes:      pbHealth.FailedRoutes,
	}
	
	if pbHealth.LastHealthCheck != nil {
		health.LastHealthCheck = pbHealth.LastHealthCheck.AsTime()
	}
	
	return health
}

// HTTP parsing helper methods

func (c *RoutingManagerClient) parseRoute(raw map[string]interface{}) (Route, error) {
	route := Route{}

	if id, ok := raw["id"].(string); ok {
		route.ID = id
	}

	if sourceXApp, ok := raw["sourceXapp"].(string); ok {
		route.SourceXApp = sourceXApp
	}

	if targetXApp, ok := raw["targetXapp"].(string); ok {
		route.TargetXApp = targetXApp
	}

	if messageType, ok := raw["messageType"].(float64); ok {
		route.MessageType = uint32(messageType)
	}

	if subscriptionID, ok := raw["subscriptionId"].(string); ok {
		route.SubscriptionID = subscriptionID
	}

	// Parse timestamps
	if createdAt, ok := raw["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			route.CreatedAt = t
		}
	}

	if updatedAt, ok := raw["updatedAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			route.UpdatedAt = t
		}
	}

	// Parse policy
	if policy, ok := raw["policy"].(map[string]interface{}); ok {
		route.Policy = c.parseRoutePolicy(policy)
	}

	return route, nil
}

func (c *RoutingManagerClient) parseRoutePolicy(raw map[string]interface{}) *RoutePolicy {
	policy := &RoutePolicy{
		Parameters: make(map[string]string),
	}

	if policyType, ok := raw["type"].(string); ok {
		policy.Type = policyType
	}

	if parameters, ok := raw["parameters"].(map[string]interface{}); ok {
		for key, value := range parameters {
			if strValue, ok := value.(string); ok {
				policy.Parameters[key] = strValue
			}
		}
	}

	return policy
}

func (c *RoutingManagerClient) parseXAppInfo(raw map[string]interface{}) (XAppInfo, error) {
	xapp := XAppInfo{
		Config: make(map[string]string),
	}

	if name, ok := raw["name"].(string); ok {
		xapp.Name = name
	}

	if version, ok := raw["version"].(string); ok {
		xapp.Version = version
	}

	if namespace, ok := raw["namespace"].(string); ok {
		xapp.Namespace = namespace
	}

	if status, ok := raw["status"].(string); ok {
		xapp.Status = status
	}

	// Parse endpoints
	if endpoints, ok := raw["endpoints"].([]interface{}); ok {
		for _, endpoint := range endpoints {
			if endpointStr, ok := endpoint.(string); ok {
				xapp.Endpoints = append(xapp.Endpoints, endpointStr)
			}
		}
	}

	// Parse config
	if config, ok := raw["config"].(map[string]interface{}); ok {
		for key, value := range config {
			if strValue, ok := value.(string); ok {
				xapp.Config[key] = strValue
			}
		}
	}

	// Parse timestamps
	if registeredAt, ok := raw["registeredAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, registeredAt); err == nil {
			xapp.RegisteredAt = t
		}
	}

	if lastHeartbeat, ok := raw["lastHeartbeat"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastHeartbeat); err == nil {
			xapp.LastHeartbeat = t
		}
	}

	return xapp, nil
}