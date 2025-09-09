/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// E2NodesHandler handles requests for E2 nodes
func (s *Server) E2NodesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	e2Client := s.clients.GetE2ManagerClient()
	if e2Client == nil {
		http.Error(w, "E2 Manager client not available", http.StatusServiceUnavailable)
		return
	}

	nodes, err := e2Client.GetNodes(ctx)
	if err != nil {
		log.Printf("Failed to get E2 nodes: %v", err)
		http.Error(w, "Failed to retrieve E2 nodes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(nodes); err != nil {
		log.Printf("Failed to encode E2 nodes response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// E2NodeHandler handles requests for a specific E2 node
func (s *Server) E2NodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]
	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	e2Client := s.clients.GetE2ManagerClient()
	if e2Client == nil {
		http.Error(w, "E2 Manager client not available", http.StatusServiceUnavailable)
		return
	}

	node, err := e2Client.GetNode(ctx, nodeID)
	if err != nil {
		log.Printf("Failed to get E2 node %s: %v", nodeID, err)
		http.Error(w, "Failed to retrieve E2 node", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(node); err != nil {
		log.Printf("Failed to encode E2 node response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// E2NodeHealthHandler handles requests for E2 node health
func (s *Server) E2NodeHealthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]
	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	e2Client := s.clients.GetE2ManagerClient()
	if e2Client == nil {
		http.Error(w, "E2 Manager client not available", http.StatusServiceUnavailable)
		return
	}

	health, err := e2Client.GetNodeHealth(ctx, nodeID)
	if err != nil {
		log.Printf("Failed to get E2 node health %s: %v", nodeID, err)
		http.Error(w, "Failed to retrieve E2 node health", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode E2 node health response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// E2NodeConfigurationHandler handles E2 node configuration updates
func (s *Server) E2NodeConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]
	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update E2NodeConfigurationUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// update.NodeID = nodeID // NodeID field doesn't exist in E2NodeConfigurationUpdate

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	e2Client := s.clients.GetE2ManagerClient()
	if e2Client == nil {
		http.Error(w, "E2 Manager client not available", http.StatusServiceUnavailable)
		return
	}

	if err := e2Client.UpdateNodeConfiguration(ctx, nodeID, &update); err != nil {
		log.Printf("Failed to update E2 node configuration %s: %v", nodeID, err)
		http.Error(w, "Failed to update E2 node configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Node configuration updated successfully",
		"nodeId":  nodeID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode configuration update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SubscriptionsHandler handles requests for subscriptions
func (s *Server) SubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	subClient := s.clients.GetSubscriptionManagerClient()
	if subClient == nil {
		http.Error(w, "Subscription Manager client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetSubscriptions(w, r, ctx, subClient)
	case http.MethodPost:
		s.handleCreateSubscription(w, r, ctx, subClient)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetSubscriptions handles GET requests for subscriptions
func (s *Server) handleGetSubscriptions(w http.ResponseWriter, r *http.Request, ctx context.Context, subClient *SubscriptionManagerClient) {
	// Parse query parameters for filtering
	filter := &SubscriptionFilter{}
	
	if e2NodeID := r.URL.Query().Get("e2NodeId"); e2NodeID != "" {
		filter.E2NodeID = e2NodeID
	}
	
	if xappID := r.URL.Query().Get("xappId"); xappID != "" {
		filter.XAppID = xappID
	}
	
	if ranFunctionIDStr := r.URL.Query().Get("ranFunctionId"); ranFunctionIDStr != "" {
		if ranFunctionID, err := strconv.ParseUint(ranFunctionIDStr, 10, 32); err == nil {
			ranFuncID := uint32(ranFunctionID)
			filter.RANFunctionID = &ranFuncID
		}
	}
	
	if status := r.URL.Query().Get("status"); status != "" {
		statusVal := SubscriptionStatus(status)
		filter.Status = &statusVal
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if _, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			// filter.Limit not available in SubscriptionFilter
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if _, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			// filter.Offset = uint32(offset) // Offset field doesn't exist in SubscriptionFilter
		}
	}

	subscriptions, err := subClient.GetSubscriptions(ctx, filter)
	if err != nil {
		log.Printf("Failed to get subscriptions: %v", err)
		http.Error(w, "Failed to retrieve subscriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		log.Printf("Failed to encode subscriptions response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateSubscription handles POST requests to create subscriptions
func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request, ctx context.Context, subClient *SubscriptionManagerClient) {
	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := subClient.CreateSubscription(ctx, &req)
	if err != nil {
		log.Printf("Failed to create subscription: %v", err)
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode subscription response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SubscriptionHandler handles requests for a specific subscription
func (s *Server) SubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriptionIDStr := vars["subscriptionId"]
	if subscriptionIDStr == "" {
		http.Error(w, "Subscription ID is required", http.StatusBadRequest)
		return
	}

	subscriptionID := SubscriptionID(subscriptionIDStr)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	subClient := s.clients.GetSubscriptionManagerClient()
	if subClient == nil {
		http.Error(w, "Subscription Manager client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetSubscription(w, r, ctx, subClient, subscriptionID)
	case http.MethodPut:
		s.handleUpdateSubscription(w, r, ctx, subClient, subscriptionID)
	case http.MethodDelete:
		s.handleDeleteSubscription(w, r, ctx, subClient, subscriptionID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetSubscription handles GET requests for a specific subscription
func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request, ctx context.Context, subClient *SubscriptionManagerClient, subscriptionID SubscriptionID) {
	subscription, err := subClient.GetSubscription(ctx, subscriptionID)
	if err != nil {
		log.Printf("Failed to get subscription %s: %v", subscriptionID, err)
		http.Error(w, "Failed to retrieve subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subscription); err != nil {
		log.Printf("Failed to encode subscription response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleUpdateSubscription handles PUT requests to update a subscription
func (s *Server) handleUpdateSubscription(w http.ResponseWriter, r *http.Request, ctx context.Context, subClient *SubscriptionManagerClient, subscriptionID SubscriptionID) {
	var update SubscriptionUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// update.SubscriptionID = subscriptionID // SubscriptionID field doesn't exist in SubscriptionUpdate

	if err := subClient.UpdateSubscription(ctx, &update); err != nil {
		log.Printf("Failed to update subscription %s: %v", subscriptionID, err)
		http.Error(w, "Failed to update subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":         "success",
		"message":        "Subscription updated successfully",
		"subscriptionId": subscriptionID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode subscription update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDeleteSubscription handles DELETE requests for a subscription
func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request, ctx context.Context, subClient *SubscriptionManagerClient, subscriptionID SubscriptionID) {
	if err := subClient.DeleteSubscription(ctx, subscriptionID); err != nil {
		log.Printf("Failed to delete subscription %s: %v", subscriptionID, err)
		http.Error(w, "Failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":         "success",
		"message":        "Subscription deleted successfully",
		"subscriptionId": subscriptionID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode subscription delete response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SubscriptionIndicationsHandler handles requests for subscription indications
func (s *Server) SubscriptionIndicationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subscriptionIDStr := vars["subscriptionId"]
	if subscriptionIDStr == "" {
		http.Error(w, "Subscription ID is required", http.StatusBadRequest)
		return
	}

	subscriptionID := SubscriptionID(subscriptionIDStr)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	subClient := s.clients.GetSubscriptionManagerClient()
	if subClient == nil {
		http.Error(w, "Subscription Manager client not available", http.StatusServiceUnavailable)
		return
	}

	// Parse limit parameter
	var limit uint32 = 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = uint32(parsedLimit)
		}
	}

	indications, err := subClient.GetIndications(ctx, subscriptionID, limit)
	if err != nil {
		log.Printf("Failed to get indications for subscription %s: %v", subscriptionID, err)
		http.Error(w, "Failed to retrieve indications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"subscriptionId": subscriptionID,
		"indications":    indications,
		"total":          len(indications),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode indications response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SCTPConnectionsHandler handles requests for SCTP connections
func (s *Server) SCTPConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	sctpManager := s.clients.GetSCTPManager()
	if sctpManager == nil {
		http.Error(w, "SCTP manager not available", http.StatusServiceUnavailable)
		return
	}

	associations := sctpManager.GetAssociations()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"associations": associations,
		"total":        len(associations),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode SCTP connections response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SCTPConnectionHandler handles requests for a specific SCTP connection
func (s *Server) SCTPConnectionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	associationID := vars["associationId"]
	if associationID == "" {
		http.Error(w, "Association ID is required", http.StatusBadRequest)
		return
	}

	sctpManager := s.clients.GetSCTPManager()
	if sctpManager == nil {
		http.Error(w, "SCTP manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		association, err := sctpManager.GetAssociation(associationID)
		if err != nil {
			log.Printf("Failed to get SCTP association %s: %v", associationID, err)
			http.Error(w, "Association not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(association); err != nil {
			log.Printf("Failed to encode SCTP association response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		if err := sctpManager.CloseAssociation(associationID); err != nil {
			log.Printf("Failed to close SCTP association %s: %v", associationID, err)
			http.Error(w, "Failed to close association", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":        "success",
			"message":       "Association closed successfully",
			"associationId": associationID,
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode SCTP close response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SCTPStatsHandler handles requests for SCTP statistics
func (s *Server) SCTPStatsHandler(w http.ResponseWriter, r *http.Request) {
	sctpManager := s.clients.GetSCTPManager()
	if sctpManager == nil {
		http.Error(w, "SCTP manager not available", http.StatusServiceUnavailable)
		return
	}

	stats := sctpManager.GetStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode SCTP stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// E2TStatsHandler handles requests for E2 Termination statistics
func (s *Server) E2TStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	e2tClient := s.clients.GetE2TClient()
	if e2tClient == nil {
		http.Error(w, "E2T client not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := e2tClient.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get E2T stats: %v", err)
		http.Error(w, "Failed to retrieve E2T stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode E2T stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// E2APMessagesHandler handles requests for E2AP messages
func (s *Server) E2APMessagesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	e2tClient := s.clients.GetE2TClient()
	if e2tClient == nil {
		http.Error(w, "E2T client not available", http.StatusServiceUnavailable)
		return
	}

	// Parse limit parameter
	var limit uint32 = 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = uint32(parsedLimit)
		}
	}

	messages, err := e2tClient.GetE2APMessages(ctx, limit)
	if err != nil {
		log.Printf("Failed to get E2AP messages: %v", err)
		http.Error(w, "Failed to retrieve E2AP messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"messages": messages,
		"total":    len(messages),
		"limit":    limit,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode E2AP messages response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}