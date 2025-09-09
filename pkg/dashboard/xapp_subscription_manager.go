// Package dashboard provides xApp subscription management capabilities
// for the O-RAN Near-RT RIC platform
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// XAppSubscriptionManager manages E2 subscriptions for xApps
type XAppSubscriptionManager struct {
	subscriptions    map[SubscriptionID]*ManagedSubscription
	subscriptionsByNode map[string][]SubscriptionID
	subscriptionsByXApp map[string][]SubscriptionID
	eventTriggers    map[SubscriptionID]*EventTrigger
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	messageBus       *RMRMessageBus
	e2tClient        *E2TClient
	submgrClient     *SubscriptionManagerClient
}

// ManagedSubscription represents a managed E2 subscription
type ManagedSubscription struct {
	ID               SubscriptionID         `json:"id"`
	XAppID           string                 `json:"xappId"`
	E2NodeID         string                 `json:"e2NodeId"`
	RANFunctionID    uint32                 `json:"ranFunctionId"`
	EventTrigger     *EventTrigger          `json:"eventTrigger"`
	Actions          []SubscriptionAction   `json:"actions"`
	Status           SubscriptionStatus     `json:"status"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	LastActivity     time.Time              `json:"lastActivity"`
	IndicationCount  int64                  `json:"indicationCount"`
	ErrorCount       int64                  `json:"errorCount"`
	Configuration    map[string]interface{} `json:"configuration,omitempty"`
}


// NOTE: EventTriggerType type moved to types.go to avoid redeclaration

// NOTE: EventTrigger type moved to subscription_models.go to avoid redeclaration

// NOTE: SubscriptionAction type moved to types.go to avoid redeclaration

// SubsequentAction represents a subsequent action in a subscription
type SubsequentAction struct {
	ID         uint32                `json:"id"`
	Type       ActionType            `json:"type"`
	Definition []byte                `json:"definition,omitempty"`
	TimeToWait time.Duration         `json:"timeToWait,omitempty"`
}

// NOTE: SubscriptionRequest type moved to xapp_communication_api.go to avoid redeclaration

// NOTE: SubscriptionResponse type moved to xapp_communication_api.go to avoid redeclaration

// NOTE: ActionResponse type moved to xapp_communication_api.go to avoid redeclaration

// ActionStatus represents the status of an action
type ActionStatus string

const (
	ActionStatusAccepted ActionStatus = "ACCEPTED"
	ActionStatusRejected ActionStatus = "REJECTED"
	ActionStatusPending  ActionStatus = "PENDING"
)

// SubscriptionStats represents subscription statistics
type SubscriptionStats struct {
	TotalSubscriptions    int64                          `json:"totalSubscriptions"`
	ActiveSubscriptions   int64                          `json:"activeSubscriptions"`
	PendingSubscriptions  int64                          `json:"pendingSubscriptions"`
	FailedSubscriptions   int64                          `json:"failedSubscriptions"`
	IndicationsReceived   int64                          `json:"indicationsReceived"`
	ByE2Node             map[string]int64               `json:"byE2Node"`
	ByXApp               map[string]int64               `json:"byXApp"`
	ByRANFunction        map[uint32]int64               `json:"byRanFunction"`
	EventTriggerTypes    map[EventTriggerType]int64     `json:"eventTriggerTypes"`
}

// NewXAppSubscriptionManager creates a new subscription manager
func NewXAppSubscriptionManager(messageBus *RMRMessageBus) *XAppSubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &XAppSubscriptionManager{
		subscriptions:       make(map[SubscriptionID]*ManagedSubscription),
		subscriptionsByNode: make(map[string][]SubscriptionID),
		subscriptionsByXApp: make(map[string][]SubscriptionID),
		eventTriggers:       make(map[SubscriptionID]*EventTrigger),
		ctx:                 ctx,
		cancel:              cancel,
		messageBus:          messageBus,
	}
}

// CreateSubscription creates a new E2 subscription
func (manager *XAppSubscriptionManager) CreateSubscription(req *SubscriptionRequest) (*SubscriptionResponse, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Generate subscription ID
	subscriptionID := manager.generateSubscriptionID()

	// Create managed subscription
	subscription := &ManagedSubscription{
		ID:            subscriptionID,
		XAppID:        req.XAppID,
		E2NodeID:      req.E2NodeID,
		RANFunctionID: req.RANFunctionID,
		EventTrigger:  req.EventTrigger,
		Actions:       req.Actions,
		Status:        SubscriptionStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		LastActivity:  time.Now(),
		Configuration: req.Configuration,
	}

	// Store subscription
	manager.subscriptions[subscriptionID] = subscription
	manager.eventTriggers[subscriptionID] = req.EventTrigger

	// Update indexes
	manager.subscriptionsByNode[req.E2NodeID] = append(manager.subscriptionsByNode[req.E2NodeID], subscriptionID)
	manager.subscriptionsByXApp[req.XAppID] = append(manager.subscriptionsByXApp[req.XAppID], subscriptionID)

	// Send subscription request to E2T
	if err := manager.sendSubscriptionRequest(subscription); err != nil {
		subscription.Status = SubscriptionStatusFailed
		log.Printf("Failed to send subscription request: %v", err)
		return &SubscriptionResponse{
			SubscriptionID: subscriptionID,
			Status:         SubscriptionStatusFailed,
			StatusMessage:  fmt.Sprintf("Failed to send request: %v", err),
			E2NodeID:       req.E2NodeID,
			RANFunctionID:  req.RANFunctionID,
			CreatedAt:      subscription.CreatedAt,
		}, err
	}

	// Create action responses
	actions := make([]ActionResponse, len(req.Actions))
	for i, action := range req.Actions {
		actions[i] = ActionResponse{
			ID:     action.ActionID,
			Status: ActionStatusAccepted,
		}
	}

	log.Printf("Created subscription %s for xApp %s on E2 node %s", subscriptionID, req.XAppID, req.E2NodeID)

	return &SubscriptionResponse{
		SubscriptionID: subscriptionID,
		Status:         SubscriptionStatusActive,
		StatusMessage:  "Subscription created successfully",
		Actions:        actions,
		E2NodeID:       req.E2NodeID,
		RANFunctionID:  req.RANFunctionID,
		CreatedAt:      subscription.CreatedAt,
	}, nil
}

// DeleteSubscription deletes an existing subscription
func (manager *XAppSubscriptionManager) DeleteSubscription(subscriptionID SubscriptionID) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	subscription, exists := manager.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// Send delete request to E2T
	if err := manager.sendDeleteRequest(subscription); err != nil {
		log.Printf("Failed to send delete request for subscription %s: %v", subscriptionID, err)
	}

	// Remove from indexes
	manager.removeFromIndex(manager.subscriptionsByNode, subscription.E2NodeID, subscriptionID)
	manager.removeFromIndex(manager.subscriptionsByXApp, subscription.XAppID, subscriptionID)

	// Remove subscription
	delete(manager.subscriptions, subscriptionID)
	delete(manager.eventTriggers, subscriptionID)

	log.Printf("Deleted subscription %s", subscriptionID)
	return nil
}

// GetSubscription retrieves a subscription by ID
func (manager *XAppSubscriptionManager) GetSubscription(subscriptionID SubscriptionID) (*ManagedSubscription, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	subscription, exists := manager.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// Return a copy to avoid race conditions
	subscriptionCopy := *subscription
	return &subscriptionCopy, nil
}

// GetSubscriptionsByXApp retrieves all subscriptions for a specific xApp
func (manager *XAppSubscriptionManager) GetSubscriptionsByXApp(xappID string) ([]*ManagedSubscription, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	subscriptionIDs, exists := manager.subscriptionsByXApp[xappID]
	if !exists {
		return []*ManagedSubscription{}, nil
	}

	subscriptions := make([]*ManagedSubscription, 0, len(subscriptionIDs))
	for _, id := range subscriptionIDs {
		if subscription, exists := manager.subscriptions[id]; exists {
			subscriptionCopy := *subscription
			subscriptions = append(subscriptions, &subscriptionCopy)
		}
	}

	return subscriptions, nil
}

// GetSubscriptionsByE2Node retrieves all subscriptions for a specific E2 node
func (manager *XAppSubscriptionManager) GetSubscriptionsByE2Node(e2NodeID string) ([]*ManagedSubscription, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	subscriptionIDs, exists := manager.subscriptionsByNode[e2NodeID]
	if !exists {
		return []*ManagedSubscription{}, nil
	}

	subscriptions := make([]*ManagedSubscription, 0, len(subscriptionIDs))
	for _, id := range subscriptionIDs {
		if subscription, exists := manager.subscriptions[id]; exists {
			subscriptionCopy := *subscription
			subscriptions = append(subscriptions, &subscriptionCopy)
		}
	}

	return subscriptions, nil
}

// GetAllSubscriptions retrieves all subscriptions
func (manager *XAppSubscriptionManager) GetAllSubscriptions() map[SubscriptionID]*ManagedSubscription {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	result := make(map[SubscriptionID]*ManagedSubscription)
	for id, subscription := range manager.subscriptions {
		subscriptionCopy := *subscription
		result[id] = &subscriptionCopy
	}

	return result
}

// UpdateSubscriptionStatus updates the status of a subscription
func (manager *XAppSubscriptionManager) UpdateSubscriptionStatus(subscriptionID SubscriptionID, status SubscriptionStatus, message string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	subscription, exists := manager.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	oldStatus := subscription.Status
	subscription.Status = status
	subscription.UpdatedAt = time.Now()

	if subscription.Configuration == nil {
		subscription.Configuration = make(map[string]interface{})
	}
	subscription.Configuration["statusMessage"] = message

	log.Printf("Subscription %s status changed from %s to %s: %s", subscriptionID, oldStatus, status, message)
	return nil
}

// HandleIndication processes an indication message for a subscription
func (manager *XAppSubscriptionManager) HandleIndication(indication *Indication) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	subscriptionID := indication.SubscriptionID
	subscription, exists := manager.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found for indication", subscriptionID)
	}

	// Update subscription activity
	subscription.LastActivity = time.Now()
	subscription.IndicationCount++

	log.Printf("Processed indication for subscription %s (count: %d)", subscriptionID, subscription.IndicationCount)
	return nil
}

// GetSubscriptionStats returns subscription statistics
func (manager *XAppSubscriptionManager) GetSubscriptionStats() *SubscriptionStats {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	stats := &SubscriptionStats{
		ByE2Node:          make(map[string]int64),
		ByXApp:            make(map[string]int64),
		ByRANFunction:     make(map[uint32]int64),
		EventTriggerTypes: make(map[EventTriggerType]int64),
	}

	for _, subscription := range manager.subscriptions {
		stats.TotalSubscriptions++

		switch subscription.Status {
		case SubscriptionStatusActive:
			stats.ActiveSubscriptions++
		case SubscriptionStatusPending:
			stats.PendingSubscriptions++
		case SubscriptionStatusFailed:
			stats.FailedSubscriptions++
		}

		stats.IndicationsReceived += subscription.IndicationCount
		stats.ByE2Node[subscription.E2NodeID]++
		stats.ByXApp[subscription.XAppID]++
		stats.ByRANFunction[subscription.RANFunctionID]++

		if trigger := subscription.EventTrigger; trigger != nil {
			stats.EventTriggerTypes[trigger.Type]++
		}
	}

	return stats
}

// Start starts the subscription manager
func (manager *XAppSubscriptionManager) Start() error {
	log.Println("Starting xApp subscription manager...")

	// Start subscription monitoring
	go manager.monitorSubscriptions()

	// Start cleanup routine
	go manager.cleanupRoutine()

	log.Println("xApp subscription manager started successfully")
	return nil
}

// Stop stops the subscription manager
func (manager *XAppSubscriptionManager) Stop() error {
	log.Println("Stopping xApp subscription manager...")

	manager.cancel()

	log.Println("xApp subscription manager stopped")
	return nil
}

// Private helper methods

// generateSubscriptionID generates a unique subscription ID
func (manager *XAppSubscriptionManager) generateSubscriptionID() SubscriptionID {
	return SubscriptionID(fmt.Sprintf("sub-%d", time.Now().UnixNano()))
}

// sendSubscriptionRequest sends a subscription request to E2T
func (manager *XAppSubscriptionManager) sendSubscriptionRequest(subscription *ManagedSubscription) error {
	if manager.messageBus == nil {
		return fmt.Errorf("message bus not available")
	}

	// Create RMR message for subscription request
	payload, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	msg := &RMRMessage{
		MessageType:   12010, // RIC_SUB_REQ
		TransactionID: string(subscription.ID),
		Source:        "dashboard",
		Target:        subscription.E2NodeID,
		Timestamp:     time.Now(),
		Payload:       payload,
	}

	return manager.messageBus.Send(msg)
}

// sendDeleteRequest sends a subscription delete request to E2T
func (manager *XAppSubscriptionManager) sendDeleteRequest(subscription *ManagedSubscription) error {
	if manager.messageBus == nil {
		return fmt.Errorf("message bus not available")
	}

	deleteReq := map[string]interface{}{
		"subscriptionId": subscription.ID,
		"e2NodeId":      subscription.E2NodeID,
		"ranFunctionId": subscription.RANFunctionID,
	}

	payload, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	msg := &RMRMessage{
		MessageType:   12020, // RIC_SUB_DEL_REQ
		TransactionID: string(subscription.ID),
		Source:        "dashboard",
		Target:        subscription.E2NodeID,
		Timestamp:     time.Now(),
		Payload:       payload,
	}

	return manager.messageBus.Send(msg)
}

// removeFromIndex removes a subscription ID from an index
func (manager *XAppSubscriptionManager) removeFromIndex(index map[string][]SubscriptionID, key string, subscriptionID SubscriptionID) {
	if subscriptionIDs, exists := index[key]; exists {
		for i, id := range subscriptionIDs {
			if id == subscriptionID {
				index[key] = append(subscriptionIDs[:i], subscriptionIDs[i+1:]...)
				break
			}
		}
		if len(index[key]) == 0 {
			delete(index, key)
		}
	}
}

// monitorSubscriptions monitors subscription health and activity
func (manager *XAppSubscriptionManager) monitorSubscriptions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-manager.ctx.Done():
			return
		case <-ticker.C:
			manager.checkSubscriptionHealth()
		}
	}
}

// checkSubscriptionHealth checks the health of all subscriptions
func (manager *XAppSubscriptionManager) checkSubscriptionHealth() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	now := time.Now()
	inactivityThreshold := 5 * time.Minute

	for id, subscription := range manager.subscriptions {
		if subscription.Status == SubscriptionStatusActive {
			if now.Sub(subscription.LastActivity) > inactivityThreshold {
				log.Printf("Subscription %s appears inactive (last activity: %v)", id, subscription.LastActivity)
				subscription.Status = SubscriptionStatusInactive
			}
		}
	}
}

// cleanupRoutine performs periodic cleanup of failed/cancelled subscriptions
func (manager *XAppSubscriptionManager) cleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-manager.ctx.Done():
			return
		case <-ticker.C:
			manager.cleanupFailedSubscriptions()
		}
	}
}

// cleanupFailedSubscriptions removes old failed/cancelled subscriptions
func (manager *XAppSubscriptionManager) cleanupFailedSubscriptions() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	now := time.Now()
	cleanupThreshold := 24 * time.Hour // Remove failed subscriptions after 24 hours

	var toDelete []SubscriptionID
	for id, subscription := range manager.subscriptions {
		if (subscription.Status == SubscriptionStatusFailed || subscription.Status == SubscriptionStatusCancelled) &&
			now.Sub(subscription.UpdatedAt) > cleanupThreshold {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		subscription := manager.subscriptions[id]
		manager.removeFromIndex(manager.subscriptionsByNode, subscription.E2NodeID, id)
		manager.removeFromIndex(manager.subscriptionsByXApp, subscription.XAppID, id)
		delete(manager.subscriptions, id)
		delete(manager.eventTriggers, id)
		log.Printf("Cleaned up subscription %s", id)
	}

	if len(toDelete) > 0 {
		log.Printf("Cleaned up %d failed/cancelled subscriptions", len(toDelete))
	}
}