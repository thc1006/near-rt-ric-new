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
	"sync"
	"time"
)

// XAppCommunicationAPI provides communication APIs for xApps
type XAppCommunicationAPI struct {
	clientManager    *ClientManager
	messageBus       *RMRMessageBus
	rmrManager       *RMRManager
	subscriptionAPI  *SubscriptionAPI
	controlAPI       *ControlAPI
	platformAPI      *PlatformAPI
	messageHandlers  map[int]MessageHandler
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// NOTE: MessageHandler type moved to types.go to avoid redeclaration

// NOTE: RMRMessage type moved to types.go to avoid redeclaration

// SubscriptionAPI provides E2 subscription management APIs
type SubscriptionAPI struct {
	submgrClient *SubscriptionManagerClient
	messageBus   *RMRMessageBus
	mu           sync.RWMutex
}

// ControlAPI provides RIC control message APIs
type ControlAPI struct {
	e2tClient  *E2TClient
	messageBus *RMRMessageBus
	mu         sync.RWMutex
}

// PlatformAPI provides platform service APIs
type PlatformAPI struct {
	e2mgrClient *E2ManagerClient
	messageBus  *RMRMessageBus
	mu          sync.RWMutex
}

// RMRManager manages RMR messaging
type RMRManager struct {
	dataPort     int
	routePort    int
	routingTable map[int]string // message type -> destination
	mu           sync.RWMutex
}

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	E2NodeID       string                 `json:"e2NodeId"`
	RANFunctionID  uint32                 `json:"ranFunctionId"`
	ServiceModel   string                 `json:"serviceModel"`
	EventTrigger   EventTriggerDefinition `json:"eventTrigger"`
	Actions        []ActionDefinition     `json:"actions"`
	ClientEndpoint string                 `json:"clientEndpoint"`
}

// NOTE: EventTriggerDefinition type moved to subscription_models.go to avoid redeclaration

// ActionDefinition defines what action to take
type ActionDefinition struct {
	ID             uint32                 `json:"id"`
	Type           string                 `json:"type"`
	Definition     []byte                 `json:"definition"`
	SubsequentID   *uint32                `json:"subsequentId,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// SubscriptionResponse represents a subscription response
type SubscriptionResponse struct {
	SubscriptionID string        `json:"subscriptionId"`
	Status         string        `json:"status"`
	StatusMessage  string        `json:"statusMessage,omitempty"`
	E2NodeID       string        `json:"e2NodeId"`
	RANFunctionID  uint32        `json:"ranFunctionId"`
	Actions        []ActionResponse `json:"actions"`
}

// ActionResponse represents an action response
type ActionResponse struct {
	ID            uint32 `json:"id"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// ControlRequest represents a control request
type ControlRequest struct {
	E2NodeID          string `json:"e2NodeId"`
	RANFunctionID     uint32 `json:"ranFunctionId"`
	CallProcessID     string `json:"callProcessId,omitempty"`
	ControlHeader     []byte `json:"controlHeader"`
	ControlMessage    []byte `json:"controlMessage"`
	ControlAckRequest bool   `json:"controlAckRequest"`
}

// ControlResponse represents a control response
type ControlResponse struct {
	E2NodeID           string        `json:"e2NodeId"`
	RANFunctionID      uint32        `json:"ranFunctionId"`
	CallProcessID      string        `json:"callProcessId,omitempty"`
	Status             ControlStatus `json:"status"`
	StatusMessage      string        `json:"statusMessage,omitempty"`
	ControlOutcome     []byte        `json:"controlOutcome,omitempty"`
	Timestamp          time.Time     `json:"timestamp"`
}

// ControlStatus represents control status
type ControlStatus string

const (
	ControlStatusSuccess ControlStatus = "SUCCESS"
	ControlStatusFailure ControlStatus = "FAILURE"
	ControlStatusPending ControlStatus = "PENDING"
)

// NOTE: Indication type moved to types.go to avoid redeclaration

// NOTE: NodeStatus type moved to types.go to avoid redeclaration

// ServiceDescriptor describes a service provided by an xApp
type ServiceDescriptor struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Endpoints    []ServiceEndpoint      `json:"endpoints"`
	MessageTypes []int                  `json:"messageTypes"`
	Dependencies []ServiceDependency    `json:"dependencies"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// ServiceEndpoint represents a service endpoint
type ServiceEndpoint struct {
	Type        string `json:"type"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Path        string `json:"path,omitempty"`
}

// ServiceDependency represents a service dependency
type ServiceDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

// NewXAppCommunicationAPI creates a new xApp communication API
func NewXAppCommunicationAPI() *XAppCommunicationAPI {
	ctx, cancel := context.WithCancel(context.Background())

	api := &XAppCommunicationAPI{
		clientManager:   NewClientManager(),
		messageBus:      NewRMRMessageBus(nil),
		messageHandlers: make(map[int]MessageHandler),
		ctx:             ctx,
		cancel:          cancel,
	}

	api.subscriptionAPI = &SubscriptionAPI{
		messageBus: api.messageBus,
	}

	api.controlAPI = &ControlAPI{
		messageBus: api.messageBus,
	}

	api.platformAPI = &PlatformAPI{
		messageBus: api.messageBus,
	}

	return api
}

// RegisterMessageHandler registers a message handler for specific message types
func (api *XAppCommunicationAPI) RegisterMessageHandler(messageType int, handler MessageHandler) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.messageHandlers[messageType] = handler
}

// SendMessage sends an RMR message
func (api *XAppCommunicationAPI) SendMessage(msg *RMRMessage) error {
	if api.messageBus == nil {
		return fmt.Errorf("message bus not initialized")
	}
	return api.messageBus.Send(msg)
}

// HandleMessage handles incoming RMR messages
func (api *XAppCommunicationAPI) HandleMessage(msg *RMRMessage) error {
	api.mu.RLock()
	handler, exists := api.messageHandlers[msg.MessageType]
	api.mu.RUnlock()

	if !exists {
		log.Printf("No handler registered for message type %d", msg.MessageType)
		return fmt.Errorf("no handler for message type %d", msg.MessageType)
	}

	return handler.HandleMessage(msg)
}

// Subscribe creates a subscription
func (api *XAppCommunicationAPI) Subscribe(req *SubscriptionRequest) (*SubscriptionResponse, error) {
	if api.subscriptionAPI == nil {
		return nil, fmt.Errorf("subscription API not initialized")
	}

	// Create subscription message
	subscriptionMsg := &RMRMessage{
		MessageType:   12010, // RIC_SUB_REQ
		TransactionID: generateTransactionID(),
		Timestamp:     time.Now(),
	}

	// Encode subscription request
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}
	subscriptionMsg.Payload = payload

	// Send subscription request
	if err := api.SendMessage(subscriptionMsg); err != nil {
		return nil, fmt.Errorf("failed to send subscription request: %w", err)
	}

	// Create response (simplified - in real implementation, wait for response)
	response := &SubscriptionResponse{
		SubscriptionID: generateSubscriptionID(),
		Status:         "ACTIVE",
		E2NodeID:       req.E2NodeID,
		RANFunctionID:  req.RANFunctionID,
		Actions:        make([]ActionResponse, len(req.Actions)),
	}

	for i, action := range req.Actions {
		response.Actions[i] = ActionResponse{
			ID:     action.ID,
			Status: "SUCCESS",
		}
	}

	return response, nil
}

// Unsubscribe removes a subscription
func (api *XAppCommunicationAPI) Unsubscribe(subscriptionID string) error {
	// Create unsubscribe message
	unsubscribeMsg := &RMRMessage{
		MessageType:   12011, // RIC_SUB_DEL_REQ
		TransactionID: generateTransactionID(),
		Timestamp:     time.Now(),
	}

	// Simple payload with subscription ID
	payload := map[string]string{"subscriptionId": subscriptionID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe request: %w", err)
	}
	unsubscribeMsg.Payload = payloadBytes

	return api.SendMessage(unsubscribeMsg)
}

// SendControl sends a control message
func (api *XAppCommunicationAPI) SendControl(req *ControlRequest) (*ControlResponse, error) {
	if api.controlAPI == nil {
		return nil, fmt.Errorf("control API not initialized")
	}

	// Create control message
	controlMsg := &RMRMessage{
		MessageType:   12040, // RIC_CONTROL_REQ
		TransactionID: generateTransactionID(),
		Timestamp:     time.Now(),
	}

	// Encode control request
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal control request: %w", err)
	}
	controlMsg.Payload = payload

	// Send control request
	if err := api.SendMessage(controlMsg); err != nil {
		return nil, fmt.Errorf("failed to send control request: %w", err)
	}

	// Create response (simplified - in real implementation, wait for response)
	response := &ControlResponse{
		E2NodeID:       req.E2NodeID,
		RANFunctionID:  req.RANFunctionID,
		CallProcessID:  req.CallProcessID,
		Status:         ControlStatusSuccess,
		Timestamp:      time.Now(),
	}

	return response, nil
}

// GetNodeStatus retrieves E2 node status
func (api *XAppCommunicationAPI) GetNodeStatus(nodeID string) (*NodeStatus, error) {
	if api.platformAPI == nil {
		return nil, fmt.Errorf("platform API not initialized")
	}

	// Create node status request
	statusMsg := &RMRMessage{
		MessageType:   12050, // NODE_STATUS_REQ
		TransactionID: generateTransactionID(),
		Timestamp:     time.Now(),
	}

	payload := map[string]string{"nodeId": nodeID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node status request: %w", err)
	}
	statusMsg.Payload = payloadBytes

	// Send request
	if err := api.SendMessage(statusMsg); err != nil {
		return nil, fmt.Errorf("failed to send node status request: %w", err)
	}

	// Return mock status (in real implementation, wait for response)
	return &NodeStatus{
		NodeID:           nodeID,
		ConnectionStatus: "CONNECTED",
		RANFunctions:     []RANFunction{},
		ServiceModels:    []ServiceModel{},
		LastUpdate:       time.Now(),
	}, nil
}

// RegisterService registers an xApp service
func (api *XAppCommunicationAPI) RegisterService(service *ServiceDescriptor) error {
	if api.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}

	// Create service registration message
	regMsg := &RMRMessage{
		MessageType:   12060, // SERVICE_REG
		TransactionID: generateTransactionID(),
		Timestamp:     time.Now(),
	}

	payload, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service descriptor: %w", err)
	}
	regMsg.Payload = payload

	return api.SendMessage(regMsg)
}

// Start starts the communication API
func (api *XAppCommunicationAPI) Start() error {
	if api.messageBus != nil {
		if err := api.messageBus.Start(); err != nil {
			return fmt.Errorf("failed to start message bus: %w", err)
		}
	}

	if api.clientManager != nil {
		if err := api.clientManager.Start(); err != nil {
			return fmt.Errorf("failed to start client manager: %w", err)
		}
	}

	log.Println("xApp Communication API started successfully")
	return nil
}

// Stop stops the communication API
func (api *XAppCommunicationAPI) Stop() error {
	api.cancel()

	if api.messageBus != nil {
		if err := api.messageBus.Stop(); err != nil {
			log.Printf("Failed to stop message bus: %v", err)
		}
	}

	if api.clientManager != nil {
		if err := api.clientManager.Stop(); err != nil {
			log.Printf("Failed to stop client manager: %v", err)
		}
	}

	log.Println("xApp Communication API stopped")
	return nil
}

// Helper functions

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	return fmt.Sprintf("tx-%d", time.Now().UnixNano())
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub-%d", time.Now().UnixNano())
}