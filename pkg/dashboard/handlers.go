/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Component discovery handlers

// handleGetComponents returns all discovered components
func (s *Server) handleGetComponents(w http.ResponseWriter, r *http.Request) {
	components := s.discovery.GetComponents()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"components": components,
		"count":      len(components),
		"timestamp":  time.Now(),
	})
}

// handleGetComponent returns a specific component by ID
func (s *Server) handleGetComponent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	componentID := vars["id"]

	component, exists := s.discovery.GetComponent(componentID)
	if !exists {
		http.Error(w, "Component not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(component)
}

// E2 Manager handlers

// handleGetE2Nodes returns all E2 nodes
func (s *Server) handleGetE2Nodes(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual E2 node retrieval from E2 Manager
	// For now, return mock data
	e2nodes := []map[string]interface{}{
		{
			"id":             "gnb-001",
			"name":           "gNodeB-001",
			"type":           "gnb",
			"status":         "connected",
			"plmnId":         "001-01",
			"connectionTime": time.Now().Add(-1 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"e2nodes":   e2nodes,
		"count":     len(e2nodes),
		"timestamp": time.Now(),
	})
}

// handleGetE2Node returns a specific E2 node
func (s *Server) handleGetE2Node(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	// TODO: Implement actual E2 node retrieval from E2 Manager
	// For now, return mock data
	e2node := map[string]interface{}{
		"id":                 nodeID,
		"name":               "gNodeB-" + nodeID,
		"type":               "gnb",
		"status":             "connected",
		"plmnId":             "001-01",
		"connectionTime":     time.Now().Add(-1 * time.Hour),
		"supportedFunctions": []string{"KPM", "RC"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e2node)
}

// Subscription Manager handlers

// handleGetSubscriptions returns all subscriptions
func (s *Server) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual subscription retrieval from Subscription Manager
	// For now, return mock data
	subscriptions := []map[string]interface{}{
		{
			"id":            "sub-001",
			"e2nodeId":      "gnb-001",
			"ranFunctionId": 1,
			"status":        "active",
			"createdTime":   time.Now().Add(-30 * time.Minute),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscriptions": subscriptions,
		"count":         len(subscriptions),
		"timestamp":     time.Now(),
	})
}

// handleCreateSubscription creates a new subscription
func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual subscription creation via Subscription Manager
	// For now, return mock response
	subscription := map[string]interface{}{
		"id":            "sub-" + time.Now().Format("20060102150405"),
		"e2nodeId":      req["e2nodeId"],
		"ranFunctionId": req["ranFunctionId"],
		"status":        "active",
		"createdTime":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subscription)

	// Broadcast subscription update via WebSocket
	s.wsHub.BroadcastMessage("subscription_created", subscription)
}

// handleGetSubscription returns a specific subscription
func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subID := vars["id"]

	// TODO: Implement actual subscription retrieval from Subscription Manager
	// For now, return mock data
	subscription := map[string]interface{}{
		"id":            subID,
		"e2nodeId":      "gnb-001",
		"ranFunctionId": 1,
		"status":        "active",
		"createdTime":   time.Now().Add(-30 * time.Minute),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

// handleDeleteSubscription deletes a subscription
func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subID := vars["id"]

	// TODO: Implement actual subscription deletion via Subscription Manager

	w.WriteHeader(http.StatusNoContent)

	// Broadcast subscription deletion via WebSocket
	s.wsHub.BroadcastMessage("subscription_deleted", map[string]string{"id": subID})
}

// App Manager handlers

// handleGetXApps returns all deployed xApps
func (s *Server) handleGetXApps(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual xApp retrieval from App Manager
	// For now, return mock data
	xapps := []map[string]interface{}{
		{
			"name":         "hello-world",
			"version":      "1.0.0",
			"status":       "running",
			"instances":    1,
			"deployedTime": time.Now().Add(-2 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"xapps":     xapps,
		"count":     len(xapps),
		"timestamp": time.Now(),
	})
}

// handleDeployXApp deploys a new xApp
func (s *Server) handleDeployXApp(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual xApp deployment via App Manager
	// For now, return mock response
	xapp := map[string]interface{}{
		"name":         req["name"],
		"version":      req["version"],
		"status":       "deploying",
		"instances":    1,
		"deployedTime": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(xapp)

	// Broadcast xApp deployment via WebSocket
	s.wsHub.BroadcastMessage("xapp_deployed", xapp)
}

// handleGetXApp returns a specific xApp
func (s *Server) handleGetXApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	xappName := vars["name"]

	// TODO: Implement actual xApp retrieval from App Manager
	// For now, return mock data
	xapp := map[string]interface{}{
		"name":         xappName,
		"version":      "1.0.0",
		"status":       "running",
		"instances":    1,
		"deployedTime": time.Now().Add(-2 * time.Hour),
		"configuration": map[string]interface{}{
			"logLevel": "info",
			"replicas": 1,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(xapp)
}

// handleUndeployXApp undeploys an xApp
func (s *Server) handleUndeployXApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	xappName := vars["name"]

	// TODO: Implement actual xApp undeployment via App Manager

	w.WriteHeader(http.StatusNoContent)

	// Broadcast xApp undeployment via WebSocket
	s.wsHub.BroadcastMessage("xapp_undeployed", map[string]string{"name": xappName})
}
