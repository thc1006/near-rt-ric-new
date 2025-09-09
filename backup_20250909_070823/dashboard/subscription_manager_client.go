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
	"net/url"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"github.com/oran/near-rt-ric-new/api/proto/submgr"
)

// SubscriptionManagerClient provides client interface for Subscription Manager component
type SubscriptionManagerClient struct {
	conn       *grpc.ClientConn
	grpcClient submgr.SubscriptionManagerClient
	httpClient *http.Client
	endpoint   string
}

// NewSubscriptionManagerClient creates a new Subscription Manager client
func NewSubscriptionManagerClient(conn *grpc.ClientConn, httpClient *http.Client, endpoint string) *SubscriptionManagerClient {
	var grpcClient submgr.SubscriptionManagerClient
	if conn != nil {
		grpcClient = submgr.NewSubscriptionManagerClient(conn)
	}
	
	return &SubscriptionManagerClient{
		conn:       conn,
		grpcClient: grpcClient,
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

// CreateSubscription creates a new subscription
func (c *SubscriptionManagerClient) CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.createSubscriptionViaGRPC(ctx, req)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("Subscription Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions", c.endpoint)
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Subscription Manager returned status %d", resp.StatusCode)
	}

	var subscriptionResp SubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&subscriptionResp); err != nil {
		return nil, fmt.Errorf("failed to decode subscription response: %w", err)
	}

	log.Printf("Successfully created subscription %s for xApp %s", subscriptionResp.SubscriptionID, req.XAppID)
	return &subscriptionResp, nil
}

// IsConnected checks if the Subscription Manager client is connected
func (c *SubscriptionManagerClient) IsConnected() bool {
	return c.conn != nil && c.conn.GetState().String() == "READY"
}

// GetSubscriptions retrieves subscriptions based on filter criteria
func (c *SubscriptionManagerClient) GetSubscriptions(ctx context.Context, filter *SubscriptionFilter) (*SubscriptionListResponse, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetSubscriptionsViaGRPC(ctx, filter)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}

	baseURL := fmt.Sprintf("%s/ric/v1/subscriptions", c.endpoint)
	
	// Build query parameters
	params := url.Values{}
	if filter != nil {
		if filter.E2NodeID != "" {
			params.Add("e2NodeId", filter.E2NodeID)
		}
		if filter.XAppID != "" {
			params.Add("xappId", filter.XAppID)
		}
		if filter.RANFunctionID != nil {
			params.Add("ranFunctionId", strconv.FormatUint(uint64(*filter.RANFunctionID), 10))
		}
		if filter.Status != "" {
			params.Add("status", string(filter.Status))
		}
		if filter.Limit > 0 {
			params.Add("limit", strconv.FormatUint(uint64(filter.Limit), 10))
		}
		if filter.Offset > 0 {
			params.Add("offset", strconv.FormatUint(uint64(filter.Offset), 10))
		}
	}

	fullURL := baseURL
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get subscriptions from Subscription Manager: %v", err)
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Subscription Manager returned status %d", resp.StatusCode)
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}

	var rawSubscriptions []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawSubscriptions); err != nil {
		log.Printf("Failed to decode subscriptions response: %v", err)
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}

	subscriptions := make([]Subscription, 0, len(rawSubscriptions))
	for _, rawSub := range rawSubscriptions {
		subscription, err := c.parseSubscription(rawSub)
		if err != nil {
			log.Printf("Failed to parse subscription: %v", err)
			continue
		}
		subscriptions = append(subscriptions, subscription)
	}

	return &SubscriptionListResponse{
		Subscriptions: subscriptions,
		Total:         uint32(len(subscriptions)),
	}, nil
}

// GetSubscription retrieves a specific subscription by ID
func (c *SubscriptionManagerClient) GetSubscription(ctx context.Context, subscriptionID SubscriptionID) (*Subscription, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetSubscriptionViaGRPC(ctx, subscriptionID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("Subscription Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions/%s", c.endpoint, subscriptionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Subscription Manager returned status %d", resp.StatusCode)
	}

	var rawSubscription map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawSubscription); err != nil {
		return nil, fmt.Errorf("failed to decode subscription response: %w", err)
	}

	return c.parseSubscription(rawSubscription)
}

// UpdateSubscription updates an existing subscription
func (c *SubscriptionManagerClient) UpdateSubscription(ctx context.Context, update *SubscriptionUpdate) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.UpdateSubscriptionViaGRPC(ctx, update)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Subscription Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions/%s", c.endpoint, update.SubscriptionID)
	
	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Subscription Manager returned status %d for update", resp.StatusCode)
	}

	log.Printf("Successfully updated subscription %s", update.SubscriptionID)
	return nil
}

// DeleteSubscription deletes a subscription
func (c *SubscriptionManagerClient) DeleteSubscription(ctx context.Context, subscriptionID SubscriptionID) error {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.DeleteSubscriptionViaGRPC(ctx, subscriptionID)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("Subscription Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions/%s", c.endpoint, subscriptionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Subscription Manager returned status %d for deletion", resp.StatusCode)
	}

	log.Printf("Successfully deleted subscription %s", subscriptionID)
	return nil
}

// GetStats retrieves statistics from Subscription Manager
func (c *SubscriptionManagerClient) GetStats(ctx context.Context) (*SubscriptionStats, error) {
	// Try gRPC client first if available
	if c.grpcClient != nil {
		return c.GetStatsViaGRPC(ctx)
	}
	
	// Fallback to HTTP client
	if c.httpClient == nil || c.endpoint == "" {
		return &SubscriptionStats{
			SubscriptionsByStatus: make(map[string]uint32),
			SubscriptionsByXApp:   make(map[string]uint32),
			LastUpdated:           time.Now(),
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
		log.Printf("Failed to get stats from Subscription Manager: %v", err)
		return &SubscriptionStats{
			SubscriptionsByStatus: make(map[string]uint32),
			SubscriptionsByXApp:   make(map[string]uint32),
			LastUpdated:           time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Subscription Manager stats returned status %d", resp.StatusCode)
		return &SubscriptionStats{
			SubscriptionsByStatus: make(map[string]uint32),
			SubscriptionsByXApp:   make(map[string]uint32),
			LastUpdated:           time.Now(),
		}, nil
	}

	var stats SubscriptionStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Printf("Failed to decode stats response: %v", err)
		return &SubscriptionStats{
			SubscriptionsByStatus: make(map[string]uint32),
			SubscriptionsByXApp:   make(map[string]uint32),
			LastUpdated:           time.Now(),
		}, nil
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// GetIndications retrieves recent indications for a subscription
func (c *SubscriptionManagerClient) GetIndications(ctx context.Context, subscriptionID SubscriptionID, limit uint32) ([]Indication, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return []Indication{}, nil
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions/%s/indications", c.endpoint, subscriptionID)
	if limit > 0 {
		url += fmt.Sprintf("?limit=%d", limit)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get indications from Subscription Manager: %v", err)
		return []Indication{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Subscription Manager indications returned status %d", resp.StatusCode)
		return []Indication{}, nil
	}

	var rawIndications []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawIndications); err != nil {
		log.Printf("Failed to decode indications response: %v", err)
		return []Indication{}, nil
	}

	indications := make([]Indication, 0, len(rawIndications))
	for _, rawInd := range rawIndications {
		indication, err := c.parseIndication(rawInd)
		if err != nil {
			log.Printf("Failed to parse indication: %v", err)
			continue
		}
		indications = append(indications, indication)
	}

	return indications, nil
}

// parseSubscription parses raw subscription data into Subscription struct
func (c *SubscriptionManagerClient) parseSubscription(raw map[string]interface{}) (Subscription, error) {
	subscription := Subscription{
		Actions: []Action{},
	}

	if id, ok := raw["subscriptionId"].(string); ok {
		subscription.ID = SubscriptionID(id)
	}

	if e2NodeID, ok := raw["e2NodeId"].(string); ok {
		subscription.E2NodeID = e2NodeID
	}

	if xappID, ok := raw["xappId"].(string); ok {
		subscription.XAppID = xappID
	}

	if ranFunctionID, ok := raw["ranFunctionId"].(float64); ok {
		subscription.RANFunctionID = uint32(ranFunctionID)
	}

	if status, ok := raw["status"].(string); ok {
		subscription.Status = SubscriptionStatus(status)
	}

	if errorMsg, ok := raw["errorMessage"].(string); ok {
		subscription.ErrorMessage = errorMsg
	}

	// Parse timestamps
	if createdAt, ok := raw["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			subscription.CreatedAt = t
		}
	}

	if updatedAt, ok := raw["updatedAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			subscription.UpdatedAt = t
		}
	}

	// Parse event trigger
	if eventTrigger, ok := raw["eventTrigger"].(map[string]interface{}); ok {
		subscription.EventTrigger = c.parseEventTrigger(eventTrigger)
	}

	// Parse actions
	if actions, ok := raw["actions"].([]interface{}); ok {
		for _, a := range actions {
			if action, ok := a.(map[string]interface{}); ok {
				parsedAction := c.parseAction(action)
				subscription.Actions = append(subscription.Actions, parsedAction)
			}
		}
	}

	return subscription, nil
}

// parseEventTrigger parses event trigger from raw data
func (c *SubscriptionManagerClient) parseEventTrigger(raw map[string]interface{}) EventTrigger {
	trigger := EventTrigger{}

	if triggerType, ok := raw["type"].(string); ok {
		trigger.Type = EventTriggerType(triggerType)
	}

	if definition, ok := raw["definition"].(string); ok {
		trigger.Definition = []byte(definition)
	}

	if period, ok := raw["period"].(float64); ok {
		duration := time.Duration(period) * time.Millisecond
		trigger.Period = &duration
	}

	return trigger
}

// parseAction parses action from raw data
func (c *SubscriptionManagerClient) parseAction(raw map[string]interface{}) Action {
	action := Action{}

	if id, ok := raw["id"].(float64); ok {
		action.ID = uint32(id)
	}

	if actionType, ok := raw["type"].(string); ok {
		action.Type = ActionType(actionType)
	}

	if definition, ok := raw["definition"].(string); ok {
		action.Definition = []byte(definition)
	}

	if subsequentAction, ok := raw["subsequentAction"].(map[string]interface{}); ok {
		action.SubsequentAction = c.parseSubsequentAction(subsequentAction)
	}

	return action
}

// parseSubsequentAction parses subsequent action from raw data
func (c *SubscriptionManagerClient) parseSubsequentAction(raw map[string]interface{}) *SubsequentAction {
	subAction := &SubsequentAction{}

	if actionType, ok := raw["type"].(string); ok {
		subAction.Type = ActionType(actionType)
	}

	if timeToWait, ok := raw["timeToWait"].(float64); ok {
		subAction.TimeToWait = uint32(timeToWait)
	}

	return subAction
}

// parseIndication parses indication from raw data
func (c *SubscriptionManagerClient) parseIndication(raw map[string]interface{}) (Indication, error) {
	indication := Indication{}

	if subscriptionID, ok := raw["subscriptionId"].(string); ok {
		indication.SubscriptionID = SubscriptionID(subscriptionID)
	}

	if e2NodeID, ok := raw["e2NodeId"].(string); ok {
		indication.E2NodeID = e2NodeID
	}

	if ranFunctionID, ok := raw["ranFunctionId"].(float64); ok {
		indication.RANFunctionID = uint32(ranFunctionID)
	}

	if actionID, ok := raw["actionId"].(float64); ok {
		indication.ActionID = uint32(actionID)
	}

	if indicationSN, ok := raw["indicationSn"].(float64); ok {
		indication.IndicationSN = uint32(indicationSN)
	}

	if header, ok := raw["indicationHeader"].(string); ok {
		indication.IndicationHeader = []byte(header)
	}

	if message, ok := raw["indicationMessage"].(string); ok {
		indication.IndicationMessage = []byte(message)
	}

	if callProcessID, ok := raw["callProcessId"].(string); ok {
		indication.CallProcessID = []byte(callProcessID)
	}

	if timestamp, ok := raw["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			indication.Timestamp = t
		}
	}

	return indication, nil
}

// createSubscriptionViaGRPC creates a subscription using gRPC client
func (c *SubscriptionManagerClient) createSubscriptionViaGRPC(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResponse, error) {
	// Convert internal request to protobuf format
	pbReq := &submgr.CreateSubscriptionRequest{
		E2NodeId:      req.E2NodeID,
		XappId:        req.XAppID,
		RanFunctionId: req.RANFunctionID,
		TimeoutMs:     uint32(req.TimeoutMs),
	}
	
	// Convert event trigger
	if req.EventTrigger != nil {
		pbReq.EventTrigger = &submgr.EventTrigger{
			Type:       string(req.EventTrigger.Type),
			Definition: req.EventTrigger.Definition,
		}
		if req.EventTrigger.Period != nil {
			periodMs := uint32(req.EventTrigger.Period.Milliseconds())
			pbReq.EventTrigger.PeriodMs = &periodMs
		}
	}
	
	// Convert actions
	for _, action := range req.Actions {
		pbAction := &submgr.Action{
			Id:         action.ID,
			Type:       string(action.Type),
			Definition: action.Definition,
		}
		
		if action.SubsequentAction != nil {
			pbAction.SubsequentAction = &submgr.SubsequentAction{
				Type:       string(action.SubsequentAction.Type),
				TimeToWait: action.SubsequentAction.TimeToWait,
			}
		}
		
		pbReq.Actions = append(pbReq.Actions, pbAction)
	}
	
	resp, err := c.grpcClient.CreateSubscription(ctx, pbReq)
	if err != nil {
		log.Printf("Failed to create subscription via gRPC: %v", err)
		// Fallback to HTTP if gRPC fails
		return c.createSubscriptionViaHTTP(ctx, req)
	}
	
	return &SubscriptionResponse{
		SubscriptionID: SubscriptionID(resp.SubscriptionId),
		Success:        resp.Success,
		Message:        resp.Message,
	}, nil
}

// GetSubscriptionsViaGRPC retrieves subscriptions using gRPC
func (c *SubscriptionManagerClient) GetSubscriptionsViaGRPC(ctx context.Context, filter *SubscriptionFilter) (*SubscriptionListResponse, error) {
	if c.grpcClient == nil {
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}
	
	req := &submgr.GetSubscriptionsRequest{}
	
	if filter != nil {
		if filter.E2NodeID != "" {
			req.E2NodeId = &filter.E2NodeID
		}
		if filter.XAppID != "" {
			req.XappId = &filter.XAppID
		}
		if filter.RANFunctionID != nil {
			req.RanFunctionId = filter.RANFunctionID
		}
		if filter.Status != "" {
			status := string(filter.Status)
			req.Status = &status
		}
		if filter.Limit > 0 {
			req.Limit = &filter.Limit
		}
		if filter.Offset > 0 {
			req.Offset = &filter.Offset
		}
	}
	
	resp, err := c.grpcClient.GetSubscriptions(ctx, req)
	if err != nil {
		log.Printf("Failed to get subscriptions via gRPC: %v", err)
		return &SubscriptionListResponse{Subscriptions: []Subscription{}, Total: 0}, nil
	}
	
	subscriptions := make([]Subscription, 0, len(resp.Subscriptions))
	for _, pbSub := range resp.Subscriptions {
		subscription := c.convertProtobufToSubscription(pbSub)
		subscriptions = append(subscriptions, subscription)
	}
	
	return &SubscriptionListResponse{
		Subscriptions: subscriptions,
		Total:         resp.Total,
	}, nil
}

// GetSubscriptionViaGRPC retrieves a specific subscription using gRPC
func (c *SubscriptionManagerClient) GetSubscriptionViaGRPC(ctx context.Context, subscriptionID SubscriptionID) (*Subscription, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}
	
	req := &submgr.GetSubscriptionRequest{
		SubscriptionId: string(subscriptionID),
	}
	
	resp, err := c.grpcClient.GetSubscription(ctx, req)
	if err != nil {
		return nil, err
	}
	
	subscription := c.convertProtobufToSubscription(resp.Subscription)
	return &subscription, nil
}

// UpdateSubscriptionViaGRPC updates a subscription using gRPC
func (c *SubscriptionManagerClient) UpdateSubscriptionViaGRPC(ctx context.Context, update *SubscriptionUpdate) error {
	if c.grpcClient == nil {
		return fmt.Errorf("gRPC client not available")
	}
	
	req := &submgr.UpdateSubscriptionRequest{
		SubscriptionId: string(update.SubscriptionID),
	}
	
	// Convert event trigger
	if update.EventTrigger != nil {
		req.EventTrigger = &submgr.EventTrigger{
			Type:       string(update.EventTrigger.Type),
			Definition: update.EventTrigger.Definition,
		}
		if update.EventTrigger.Period != nil {
			periodMs := uint32(update.EventTrigger.Period.Milliseconds())
			req.EventTrigger.PeriodMs = &periodMs
		}
	}
	
	// Convert actions
	for _, action := range update.Actions {
		pbAction := &submgr.Action{
			Id:         action.ID,
			Type:       string(action.Type),
			Definition: action.Definition,
		}
		
		if action.SubsequentAction != nil {
			pbAction.SubsequentAction = &submgr.SubsequentAction{
				Type:       string(action.SubsequentAction.Type),
				TimeToWait: action.SubsequentAction.TimeToWait,
			}
		}
		
		req.Actions = append(req.Actions, pbAction)
	}
	
	resp, err := c.grpcClient.UpdateSubscription(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to update subscription: %s", resp.Message)
	}
	
	return nil
}

// DeleteSubscriptionViaGRPC deletes a subscription using gRPC
func (c *SubscriptionManagerClient) DeleteSubscriptionViaGRPC(ctx context.Context, subscriptionID SubscriptionID) error {
	if c.grpcClient == nil {
		return fmt.Errorf("gRPC client not available")
	}
	
	req := &submgr.DeleteSubscriptionRequest{
		SubscriptionId: string(subscriptionID),
	}
	
	resp, err := c.grpcClient.DeleteSubscription(ctx, req)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("failed to delete subscription: %s", resp.Message)
	}
	
	return nil
}

// GetStatsViaGRPC retrieves statistics using gRPC
func (c *SubscriptionManagerClient) GetStatsViaGRPC(ctx context.Context) (*SubscriptionStats, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}
	
	req := &submgr.GetStatsRequest{}
	
	resp, err := c.grpcClient.GetStats(ctx, req)
	if err != nil {
		return nil, err
	}
	
	stats := &SubscriptionStats{
		SubscriptionsByStatus:      resp.Stats.SubscriptionsByStatus,
		SubscriptionsByXApp:        resp.Stats.SubscriptionsByXapp,
		SubscriptionsByRANFunction: resp.Stats.SubscriptionsByRanFunction,
		TotalSubscriptions:         resp.Stats.TotalSubscriptions,
		ActiveIndicationsPerSecond: resp.Stats.ActiveIndicationsPerSecond,
	}
	
	if resp.Stats.LastUpdated != nil {
		stats.LastUpdated = resp.Stats.LastUpdated.AsTime()
	}
	
	return stats, nil
}

// convertProtobufToSubscription converts protobuf subscription to internal format
func (c *SubscriptionManagerClient) convertProtobufToSubscription(pbSub *submgr.Subscription) Subscription {
	subscription := Subscription{
		ID:            SubscriptionID(pbSub.Id),
		E2NodeID:      pbSub.E2NodeId,
		XAppID:        pbSub.XappId,
		RANFunctionID: pbSub.RanFunctionId,
		Status:        SubscriptionStatus(pbSub.Status),
		ErrorMessage:  pbSub.ErrorMessage,
		Actions:       []Action{},
	}
	
	if pbSub.CreatedAt != nil {
		subscription.CreatedAt = pbSub.CreatedAt.AsTime()
	}
	
	if pbSub.UpdatedAt != nil {
		subscription.UpdatedAt = pbSub.UpdatedAt.AsTime()
	}
	
	// Convert event trigger
	if pbSub.EventTrigger != nil {
		subscription.EventTrigger = EventTrigger{
			Type:       EventTriggerType(pbSub.EventTrigger.Type),
			Definition: pbSub.EventTrigger.Definition,
		}
		
		if pbSub.EventTrigger.PeriodMs != nil {
			period := time.Duration(*pbSub.EventTrigger.PeriodMs) * time.Millisecond
			subscription.EventTrigger.Period = &period
		}
	}
	
	// Convert actions
	for _, pbAction := range pbSub.Actions {
		action := Action{
			ID:         pbAction.Id,
			Type:       ActionType(pbAction.Type),
			Definition: pbAction.Definition,
		}
		
		if pbAction.SubsequentAction != nil {
			action.SubsequentAction = &SubsequentAction{
				Type:       ActionType(pbAction.SubsequentAction.Type),
				TimeToWait: pbAction.SubsequentAction.TimeToWait,
			}
		}
		
		subscription.Actions = append(subscription.Actions, action)
	}
	
	return subscription
}

// createSubscriptionViaHTTP creates a subscription using HTTP client (existing implementation)
func (c *SubscriptionManagerClient) createSubscriptionViaHTTP(ctx context.Context, req *SubscriptionRequest) (*SubscriptionResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("Subscription Manager client not configured")
	}

	url := fmt.Sprintf("%s/ric/v1/subscriptions", c.endpoint)
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Subscription Manager returned status %d", resp.StatusCode)
	}

	var subscriptionResp SubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&subscriptionResp); err != nil {
		return nil, fmt.Errorf("failed to decode subscription response: %w", err)
	}

	log.Printf("Successfully created subscription %s for xApp %s", subscriptionResp.SubscriptionID, req.XAppID)
	return &subscriptionResp, nil
}