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

	"github.com/google/uuid"
)

// XAppSubscriptionManager manages subscriptions for xApps
type XAppSubscriptionManager struct {
	mu              sync.RWMutex
	subscriptions   map[string]*XAppSubscription
	clientManager   *ClientManager
	messageBus      *RMRMessageBus
	subscriptionMap map[string]string // maps subscription ID to xApp instance ID
	isRunning       bool
	ctx             context.Context
	cancel          context.CancelFunc
}

// XAppSubscription represents a subscription created by an xApp
type XAppSubscription struct {
	ID               string                `json:"id"`
	XAppInstanceID   string                `json:"xappInstanceId"`
	E2NodeID         string                `json:"e2NodeId"`
	RANFunctionID    uint32                `json:"ranFunctionId"`
	ServiceModelType ServiceModelType      `json:"serviceModelType"`
	EventTrigger     *SubscriptionEventTrigger `json:"eventTrigger"`
	Actions          []SubscriptionAction  `json:"actions"`
	Status           SubscriptionStatus    `json:"status"`
	CreatedAt        time.Time             `json:"createdAt"`
	UpdatedAt        time.Time             `json:"updatedAt"`
	LastIndication   time.Time             `json:"lastIndication"`
	IndicationCount  uint64                `json:"indicationCount"`
	ErrorCount       uint64                `json:"errorCount"`
	LastError        string                `json:"lastError,omitempty"`
}

// SubscriptionEventTrigger represents the event trigger for a subscription
type SubscriptionEventTrigger struct {
	Type       EventTriggerType `json:"type"`
	Period     *time.Duration   `json:"period,omitempty"`
	Threshold  *float64         `json:"threshold,omitempty"`
	Definition []byte           `json:"definition,omitempty"`
}

// EventTriggerType represents the type of event trigger
type EventTriggerType string

const (
	EventTriggerPeriodic     EventTriggerType = "PERIODIC"
	EventTriggerOnChange     EventTriggerType = "ON_CHANGE"
	EventTriggerThreshold    EventTriggerType = "THRESHOLD"
	EventTriggerEventBased   EventTriggerType = "EVENT_BASED"
)

// SubscriptionAction represents an action in a subscription
type SubscriptionAction struct {
	ID         uint32                `json:"id"`
	Type       ActionType            `json:"type"`
	Definition []byte                `json:"definition,omitempty"`
	Subsequent *SubsequentAction     `json:"subsequent,omitempty"`
}

// ActionType represents the type of subscription action
type ActionType string

const (
	ActionTypeReport ActionType = "REPORT"
	ActionTypeInsert ActionType = "INSERT"
	ActionTypePolicy ActionType = "POLICY"
)

// SubsequentAction represents a subsequent action
type SubsequentAction struct {
	Type       ActionType `json:"type"`
	TimeToWait uint32     `json:"timeToWait"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusPending SubscriptionStatus = "PENDING"
	SubscriptionStatusActive  SubscriptionStatus = "ACTIVE"
	SubscriptionStatusFailed  SubscriptionStatus = "FAILED"
	SubscriptionStatusDeleted SubscriptionStatus = "DELETED"
)

// NewXAppSubscriptionManager creates a new xApp subscription manager
func NewXAppSubscriptionManager(clientManager *ClientManager, messageBus *RMRMessageBus) *XAppSubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &XAppSubscriptionManager{
		subscriptions:   make(map[string]*XAppSubscription),
		clientManager:   clientManager,
		messageBus:      messageBus,
		subscriptionMap: make(map[string]string),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the subscription manager
func (sm *XAppSubscriptionManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.isRunning {
		return fmt.Errorf("subscription manager is already running")
	}
	
	// Register as RMR message handler for subscription-related messages
	sm.messageBus.RegisterMessageHandler(sm)
	
	// Start indication processor
	go sm.indicationProcessor()
	
	// Start subscription health monitor
	go sm.subscriptionHealthMonitor()
	
	sm.isRunning = true
	log.Println("xApp subscription manager started")
	return nil
}

// Stop stops the subscription manager
func (sm *XAppSubscriptionManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if !sm.isRunning {
		return nil
	}
	
	sm.cancel()
	sm.isRunning = false
	log.Println("xApp subscription manager stopped")
	return nil
}

// CreateSubscription creates a new subscription for an xApp
func (sm *XAppSubscriptionManager) CreateSubscription(xappInstanceID string, req *SubscriptionRequest) (*XAppSubscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	subscriptionID := uuid.New().String()
	
	subscription := &XAppSubscription{
		ID:               subscriptionID,
		XAppInstanceID:   xappInstanceID,
		E2NodeID:         req.E2NodeID,
		RANFunctionID:    req.RANFunctionID,
		ServiceModelType: req.ServiceModelType,
		EventTrigger:     req.EventTrigger,
		Actions:          req.Actions,
		Status:           SubscriptionStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// Store subscription
	sm.subscriptions[subscriptionID] = subscription
	sm.subscriptionMap[subscriptionID] = xappInstanceID
	
	// Send subscription request via RMR
	if err := sm.sendSubscriptionRequest(subscription); err != nil {
		subscription.Status = SubscriptionStatusFailed
		subscription.LastError = err.Error()
		subscription.UpdatedAt = time.Now()
		return subscription, fmt.Errorf("failed to send subscription request: %w", err)
	}
	
	log.Printf("Created subscription %s for xApp %s", subscriptionID, xappInstanceID)
	return subscription, nil
}

// DeleteSubscription deletes a subscription
func (sm *XAppSubscriptionManager) DeleteSubscription(subscriptionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}
	
	// Send subscription delete request via RMR
	if err := sm.sendSubscriptionDeleteRequest(subscription); err != nil {
		return fmt.Errorf("failed to send subscription delete request: %w", err)
	}
	
	// Update subscription status
	subscription.Status = SubscriptionStatusDeleted
	subscription.UpdatedAt = time.Now()
	
	// Remove from maps
	delete(sm.subscriptions, subscriptionID)
	delete(sm.subscriptionMap, subscriptionID)
	
	log.Printf("Deleted subscription %s", subscriptionID)
	return nil
}

// GetSubscription returns a subscription by ID
func (sm *XAppSubscriptionManager) GetSubscription(subscriptionID string) (*XAppSubscription, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}
	
	return subscription, nil
}

// ListSubscriptions returns all subscriptions for an xApp instance
func (sm *XAppSubscriptionManager) ListSubscriptions(xappInstanceID string) ([]*XAppSubscription, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var subscriptions []*XAppSubscription
	for _, subscription := range sm.subscriptions {
		if subscription.XAppInstanceID == xappInstanceID {
			subscriptions = append(subscriptions, subscription)
		}
	}
	
	return subscriptions, nil
}

// GetAllSubscriptions returns all subscriptions
func (sm *XAppSubscriptionManager) GetAllSubscriptions() map[string]*XAppSubscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	subscriptions := make(map[string]*XAppSubscription)
	for id, subscription := range sm.subscriptions {
		subscriptions[id] = &XAppSubscription{
			ID:               subscription.ID,
			XAppInstanceID:   subscription.XAppInstanceID,
			E2NodeID:         subscription.E2NodeID,
			RANFunctionID:    subscription.RANFunctionID,
			ServiceModelType: subscription.ServiceModelType,
			EventTrigger:     subscription.EventTrigger,
			Actions:          subscription.Actions,
			Status:           subscription.Status,
			CreatedAt:        subscription.CreatedAt,
			UpdatedAt:        subscription.UpdatedAt,
			LastIndication:   subscription.LastIndication,
			IndicationCount:  subscription.IndicationCount,
			ErrorCount:       subscription.ErrorCount,
			LastError:        subscription.LastError,
		}
	}
	
	return subscriptions
}

// HandleMessage implements MessageHandler interface for RMR messages
func (sm *XAppSubscriptionManager) HandleMessage(ctx context.Context, msg *RMRMessage) error {
	switch msg.MessageType {
	case RMR_MSG_E2AP_SUBSCRIPTION_RESP:
		return sm.handleSubscriptionResponse(msg)
	case RMR_MSG_E2AP_SUBSCRIPTION_FAILURE:
		return sm.handleSubscriptionFailure(msg)
	case RMR_MSG_E2AP_SUBSCRIPTION_DELETE_RESP:
		return sm.handleSubscriptionDeleteResponse(msg)
	case RMR_MSG_E2AP_INDICATION:
		return sm.handleIndication(msg)
	default:
		return fmt.Errorf("unsupported message type: %d", msg.MessageType)
	}
}

// GetMessageTypes returns the message types this handler processes
func (sm *XAppSubscriptionManager) GetMessageTypes() []uint32 {
	return []uint32{
		RMR_MSG_E2AP_SUBSCRIPTION_RESP,
		RMR_MSG_E2AP_SUBSCRIPTION_FAILURE,
		RMR_MSG_E2AP_SUBSCRIPTION_DELETE_RESP,
		RMR_MSG_E2AP_INDICATION,
	}
}

// Private methods

func (sm *XAppSubscriptionManager) sendSubscriptionRequest(subscription *XAppSubscription) error {
	// Create subscription request message
	subReq := map[string]interface{}{
		"subscriptionId": subscription.ID,
		"e2NodeId":       subscription.E2NodeID,
		"ranFunctionId":  subscription.RANFunctionID,
		"eventTrigger":   subscription.EventTrigger,
		"actions":        subscription.Actions,
	}
	
	payload, err := json.Marshal(subReq)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription request: %w", err)
	}
	
	// Send via RMR
	rmrMsg := &RMRMessage{
		MessageType:    RMR_MSG_E2AP_SUBSCRIPTION_REQ,
		SubscriptionID: subscription.ID,
		TransactionID:  uuid.New().String(),
		Payload:        payload,
		Source:         subscription.XAppInstanceID,
		Target:         "submgr",
		Timestamp:      time.Now(),
	}
	
	return sm.messageBus.SendMessage(rmrMsg)
}

func (sm *XAppSubscriptionManager) sendSubscriptionDeleteRequest(subscription *XAppSubscription) error {
	// Create subscription delete request message
	delReq := map[string]interface{}{
		"subscriptionId": subscription.ID,
		"e2NodeId":       subscription.E2NodeID,
		"ranFunctionId":  subscription.RANFunctionID,
	}
	
	payload, err := json.Marshal(delReq)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription delete request: %w", err)
	}
	
	// Send via RMR
	rmrMsg := &RMRMessage{
		MessageType:    RMR_MSG_E2AP_SUBSCRIPTION_DELETE_REQ,
		SubscriptionID: subscription.ID,
		TransactionID:  uuid.New().String(),
		Payload:        payload,
		Source:         subscription.XAppInstanceID,
		Target:         "submgr",
		Timestamp:      time.Now(),
	}
	
	return sm.messageBus.SendMessage(rmrMsg)
}

func (sm *XAppSubscriptionManager) handleSubscriptionResponse(msg *RMRMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	subscription, exists := sm.subscriptions[msg.SubscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", msg.SubscriptionID)
	}
	
	subscription.Status = SubscriptionStatusActive
	subscription.UpdatedAt = time.Now()
	
	log.Printf("Subscription %s activated successfully", msg.SubscriptionID)
	return nil
}

func (sm *XAppSubscriptionManager) handleSubscriptionFailure(msg *RMRMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	subscription, exists := sm.subscriptions[msg.SubscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", msg.SubscriptionID)
	}
	
	subscription.Status = SubscriptionStatusFailed
	subscription.UpdatedAt = time.Now()
	subscription.ErrorCount++
	
	// Parse failure reason from payload
	var failureInfo map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &failureInfo); err == nil {
		if reason, ok := failureInfo["reason"].(string); ok {
			subscription.LastError = reason
		}
	}
	
	log.Printf("Subscription %s failed: %s", msg.SubscriptionID, subscription.LastError)
	return nil
}

func (sm *XAppSubscriptionManager) handleSubscriptionDeleteResponse(msg *RMRMessage) error {
	log.Printf("Subscription %s delete confirmed", msg.SubscriptionID)
	return nil
}

func (sm *XAppSubscriptionManager) handleIndication(msg *RMRMessage) error {
	sm.mu.Lock()
	subscription, exists := sm.subscriptions[msg.SubscriptionID]
	if !exists {
		sm.mu.Unlock()
		return fmt.Errorf("subscription %s not found for indication", msg.SubscriptionID)
	}
	
	// Update subscription statistics
	subscription.LastIndication = time.Now()
	subscription.IndicationCount++
	sm.mu.Unlock()
	
	// Forward indication to xApp instance
	return sm.forwardIndicationToXApp(subscription, msg)
}

func (sm *XAppSubscriptionManager) forwardIndicationToXApp(subscription *XAppSubscription, msg *RMRMessage) error {
	// Create indication message for xApp
	indication := map[string]interface{}{
		"subscriptionId":   subscription.ID,
		"e2NodeId":         subscription.E2NodeID,
		"ranFunctionId":    subscription.RANFunctionID,
		"serviceModelType": subscription.ServiceModelType,
		"timestamp":        msg.Timestamp,
		"payload":          msg.Payload,
	}
	
	payload, err := json.Marshal(indication)
	if err != nil {
		return fmt.Errorf("failed to marshal indication for xApp: %w", err)
	}
	
	// Send to xApp via RMR
	xappMsg := &RMRMessage{
		MessageType:    RMR_MSG_E2AP_INDICATION,
		SubscriptionID: subscription.ID,
		TransactionID:  msg.TransactionID,
		Payload:        payload,
		Source:         "submgr",
		Target:         subscription.XAppInstanceID,
		Timestamp:      time.Now(),
	}
	
	return sm.messageBus.SendMessage(xappMsg)
}

func (sm *XAppSubscriptionManager) indicationProcessor() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			// Process any queued indications
			// This is a placeholder for more sophisticated indication processing
		}
	}
}

func (sm *XAppSubscriptionManager) subscriptionHealthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.checkSubscriptionHealth()
		}
	}
}

func (sm *XAppSubscriptionManager) checkSubscriptionHealth() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for _, subscription := range sm.subscriptions {
		if subscription.Status == SubscriptionStatusActive {
			// Check if subscription has been inactive for too long
			if now.Sub(subscription.LastIndication) > 5*time.Minute {
				log.Printf("Subscription %s appears inactive (last indication: %v)", 
					subscription.ID, subscription.LastIndication)
			}
		}
	}
}

// GetSubscriptionStatistics returns statistics for all subscriptions
func (sm *XAppSubscriptionManager) GetSubscriptionStatistics() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"totalSubscriptions": len(sm.subscriptions),
		"byStatus":           make(map[string]int),
		"byServiceModel":     make(map[string]int),
		"byXApp":             make(map[string]int),
	}
	
	statusCounts := make(map[string]int)
	serviceModelCounts := make(map[string]int)
	xappCounts := make(map[string]int)
	
	var totalIndications uint64
	var totalErrors uint64
	
	for _, subscription := range sm.subscriptions {
		statusCounts[string(subscription.Status)]++
		serviceModelCounts[string(subscription.ServiceModelType)]++
		xappCounts[subscription.XAppInstanceID]++
		totalIndications += subscription.IndicationCount
		totalErrors += subscription.ErrorCount
	}
	
	stats["byStatus"] = statusCounts
	stats["byServiceModel"] = serviceModelCounts
	stats["byXApp"] = xappCounts
	stats["totalIndications"] = totalIndications
	stats["totalErrors"] = totalErrors
	
	return stats
}

// SubscriptionRequest represents a request to create a subscription
type SubscriptionRequest struct {
	E2NodeID         string                    `json:"e2NodeId"`
	RANFunctionID    uint32                    `json:"ranFunctionId"`
	ServiceModelType ServiceModelType          `json:"serviceModelType"`
	EventTrigger     *SubscriptionEventTrigger `json:"eventTrigger"`
	Actions          []SubscriptionAction      `json:"actions"`
}