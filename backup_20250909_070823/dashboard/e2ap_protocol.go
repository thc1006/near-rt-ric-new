/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// E2AP Procedure State Machine States
type E2APProcedureState int

const (
	E2APStateIdle E2APProcedureState = iota
	E2APStateInitiated
	E2APStateWaitingResponse
	E2APStateCompleted
	E2APStateFailed
	E2APStateTimeout
)

// E2AP Transaction represents an ongoing E2AP transaction
type E2APTransaction struct {
	ID            string
	ProcedureCode uint8
	State         E2APProcedureState
	InitiatedAt   time.Time
	CompletedAt   *time.Time
	TimeoutAt     time.Time
	Request       *E2APMessage
	Response      *E2APMessage
	ErrorCause    *E2APCause
	Retries       int
	MaxRetries    int
}

// E2APProcedureHandler handles E2AP procedure state machines
type E2APProcedureHandler struct {
	mu           sync.RWMutex
	transactions map[string]*E2APTransaction
	encoder      *E2APEncoder
	messageBus   *RMRMessageBus
	nodeID       string
	
	// Procedure timeouts
	setupTimeout    time.Duration
	configTimeout   time.Duration
	resetTimeout    time.Duration
	controlTimeout  time.Duration
}

// E2APMessageValidator provides E2AP message validation
type E2APMessageValidator struct {
	encoder *E2APEncoder
}

// NewE2APProcedureHandler creates a new E2AP procedure handler
func NewE2APProcedureHandler(nodeID string, messageBus *RMRMessageBus) *E2APProcedureHandler {
	return &E2APProcedureHandler{
		transactions:  make(map[string]*E2APTransaction),
		encoder:       NewE2APEncoder(),
		messageBus:    messageBus,
		nodeID:        nodeID,
		setupTimeout:  30 * time.Second,
		configTimeout: 20 * time.Second,
		resetTimeout:  15 * time.Second,
		controlTimeout: 10 * time.Second,
	}
}

// HandleE2SetupProcedure handles E2 Setup procedure
func (h *E2APProcedureHandler) HandleE2SetupProcedure(ctx context.Context, setupReq *E2SetupRequestMessage) (*E2SetupResponseMessage, error) {
	transactionID := uuid.New().String()
	
	// Create transaction
	transaction := &E2APTransaction{
		ID:            transactionID,
		ProcedureCode: E2AP_PROCEDURE_E2_SETUP,
		State:         E2APStateInitiated,
		InitiatedAt:   time.Now(),
		TimeoutAt:     time.Now().Add(h.setupTimeout),
		MaxRetries:    3,
	}
	
	// Create E2AP message
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_INITIATING_MESSAGE,
		ProcedureCode: E2AP_PROCEDURE_E2_SETUP,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"transactionId":   setupReq.TransactionID,
			"globalE2NodeId":  setupReq.GlobalE2NodeID,
			"ranFunctions":    setupReq.RANFunctions,
			"componentConfig": setupReq.E2NodeComponentConfigAddList,
		},
	}
	
	transaction.Request = msg
	
	h.mu.Lock()
	h.transactions[transactionID] = transaction
	h.mu.Unlock()
	
	// Encode and send message
	encodedMsg, err := h.encoder.EncodeE2APMessage(msg)
	if err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_PROTOCOL,
			CauseValue: 1, // encoding-error
		})
		return nil, fmt.Errorf("failed to encode E2 Setup Request: %w", err)
	}
	
	// Send via RMR
	rmrMsg := &RMRMessage{
		MessageType:    RMR_MSG_E2AP_SETUP_REQ,
		TransactionID:  transactionID,
		Payload:        encodedMsg,
		Source:         h.nodeID,
		Timestamp:      time.Now(),
	}
	
	if err := h.messageBus.SendMessage(rmrMsg); err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_TRANSPORT,
			CauseValue: 1, // transport-resource-unavailable
		})
		return nil, fmt.Errorf("failed to send E2 Setup Request: %w", err)
	}
	
	h.updateTransactionState(transactionID, E2APStateWaitingResponse, nil)
	
	// Wait for response with timeout
	return h.waitForSetupResponse(ctx, transactionID)
}

// HandleE2ConfigurationUpdateProcedure handles E2 Node Configuration Update procedure
func (h *E2APProcedureHandler) HandleE2ConfigurationUpdateProcedure(ctx context.Context, configReq *E2NodeConfigurationUpdateMessage) error {
	transactionID := uuid.New().String()
	
	transaction := &E2APTransaction{
		ID:            transactionID,
		ProcedureCode: E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE,
		State:         E2APStateInitiated,
		InitiatedAt:   time.Now(),
		TimeoutAt:     time.Now().Add(h.configTimeout),
		MaxRetries:    2,
	}
	
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_INITIATING_MESSAGE,
		ProcedureCode: E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"transactionId":     configReq.TransactionID,
			"globalE2NodeId":    configReq.GlobalE2NodeID,
			"configAddList":     configReq.E2NodeComponentConfigAddList,
			"configUpdateList":  configReq.E2NodeComponentConfigUpdateList,
			"configRemovalList": configReq.E2NodeComponentConfigRemovalList,
		},
	}
	
	transaction.Request = msg
	
	h.mu.Lock()
	h.transactions[transactionID] = transaction
	h.mu.Unlock()
	
	encodedMsg, err := h.encoder.EncodeE2APMessage(msg)
	if err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_PROTOCOL,
			CauseValue: 1,
		})
		return fmt.Errorf("failed to encode E2 Configuration Update: %w", err)
	}
	
	rmrMsg := &RMRMessage{
		MessageType:   RMR_MSG_E2AP_CONFIG_UPDATE_REQ,
		TransactionID: transactionID,
		Payload:       encodedMsg,
		Source:        h.nodeID,
		Timestamp:     time.Now(),
	}
	
	if err := h.messageBus.SendMessage(rmrMsg); err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_TRANSPORT,
			CauseValue: 1,
		})
		return fmt.Errorf("failed to send E2 Configuration Update: %w", err)
	}
	
	h.updateTransactionState(transactionID, E2APStateWaitingResponse, nil)
	return h.waitForConfigResponse(ctx, transactionID)
}

// HandleE2ResetProcedure handles E2 Reset procedure
func (h *E2APProcedureHandler) HandleE2ResetProcedure(ctx context.Context, resetReq *E2ResetRequestMessage) error {
	transactionID := uuid.New().String()
	
	transaction := &E2APTransaction{
		ID:            transactionID,
		ProcedureCode: E2AP_PROCEDURE_RESET,
		State:         E2APStateInitiated,
		InitiatedAt:   time.Now(),
		TimeoutAt:     time.Now().Add(h.resetTimeout),
		MaxRetries:    1,
	}
	
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_INITIATING_MESSAGE,
		ProcedureCode: E2AP_PROCEDURE_RESET,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"transactionId": resetReq.TransactionID,
			"cause":         resetReq.Cause,
		},
	}
	
	transaction.Request = msg
	
	h.mu.Lock()
	h.transactions[transactionID] = transaction
	h.mu.Unlock()
	
	encodedMsg, err := h.encoder.EncodeE2APMessage(msg)
	if err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_PROTOCOL,
			CauseValue: 1,
		})
		return fmt.Errorf("failed to encode E2 Reset Request: %w", err)
	}
	
	rmrMsg := &RMRMessage{
		MessageType:   RMR_MSG_E2AP_RESET_REQ,
		TransactionID: transactionID,
		Payload:       encodedMsg,
		Source:        h.nodeID,
		Timestamp:     time.Now(),
	}
	
	if err := h.messageBus.SendMessage(rmrMsg); err != nil {
		h.updateTransactionState(transactionID, E2APStateFailed, &E2APCause{
			CauseType:  E2AP_CAUSE_TRANSPORT,
			CauseValue: 1,
		})
		return fmt.Errorf("failed to send E2 Reset Request: %w", err)
	}
	
	h.updateTransactionState(transactionID, E2APStateWaitingResponse, nil)
	return h.waitForResetResponse(ctx, transactionID)
}

// ProcessIncomingMessage processes incoming E2AP messages
func (h *E2APProcedureHandler) ProcessIncomingMessage(ctx context.Context, rmrMsg *RMRMessage) error {
	// Decode E2AP message
	e2apMsg, err := h.encoder.DecodeE2APMessage(rmrMsg.Payload)
	if err != nil {
		log.Printf("Failed to decode E2AP message: %v", err)
		return fmt.Errorf("failed to decode E2AP message: %w", err)
	}
	
	// Handle based on message type and PDU type
	switch e2apMsg.ProcedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		return h.handleSetupResponse(rmrMsg.TransactionID, e2apMsg)
	case E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE:
		return h.handleConfigResponse(rmrMsg.TransactionID, e2apMsg)
	case E2AP_PROCEDURE_RESET:
		return h.handleResetResponse(rmrMsg.TransactionID, e2apMsg)
	case E2AP_PROCEDURE_RIC_INDICATION:
		return h.handleIndication(e2apMsg)
	case E2AP_PROCEDURE_RIC_CONTROL:
		return h.handleControlResponse(rmrMsg.TransactionID, e2apMsg)
	default:
		log.Printf("Unknown E2AP procedure code: %d", e2apMsg.ProcedureCode)
		return fmt.Errorf("unknown E2AP procedure code: %d", e2apMsg.ProcedureCode)
	}
}

// GetMessageTypes returns the message types this handler processes
func (h *E2APProcedureHandler) GetMessageTypes() []uint32 {
	return []uint32{
		RMR_MSG_E2AP_SETUP_RESP,
		RMR_MSG_E2AP_SETUP_FAILURE,
		RMR_MSG_E2AP_CONFIG_UPDATE_RESP,
		RMR_MSG_E2AP_RESET_RESP,
		RMR_MSG_E2AP_INDICATION,
		RMR_MSG_E2AP_CONTROL_ACK,
		RMR_MSG_E2AP_CONTROL_FAILURE,
	}
}

// HandleMessage implements MessageHandler interface
func (h *E2APProcedureHandler) HandleMessage(ctx context.Context, msg *RMRMessage) error {
	return h.ProcessIncomingMessage(ctx, msg)
}

// Private methods

func (h *E2APProcedureHandler) updateTransactionState(transactionID string, state E2APProcedureState, cause *E2APCause) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	transaction, exists := h.transactions[transactionID]
	if !exists {
		return
	}
	
	transaction.State = state
	if cause != nil {
		transaction.ErrorCause = cause
	}
	
	if state == E2APStateCompleted || state == E2APStateFailed || state == E2APStateTimeout {
		now := time.Now()
		transaction.CompletedAt = &now
	}
}

func (h *E2APProcedureHandler) waitForSetupResponse(ctx context.Context, transactionID string) (*E2SetupResponseMessage, error) {
	timeout := time.NewTimer(h.setupTimeout)
	defer timeout.Stop()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			h.updateTransactionState(transactionID, E2APStateTimeout, nil)
			return nil, fmt.Errorf("E2 Setup procedure timeout")
		case <-ticker.C:
			h.mu.RLock()
			transaction, exists := h.transactions[transactionID]
			h.mu.RUnlock()
			
			if !exists {
				return nil, fmt.Errorf("transaction not found")
			}
			
			switch transaction.State {
			case E2APStateCompleted:
				if transaction.Response != nil {
					return h.parseSetupResponse(transaction.Response)
				}
				return nil, fmt.Errorf("completed transaction has no response")
			case E2APStateFailed:
				cause := "unknown error"
				if transaction.ErrorCause != nil {
					cause = fmt.Sprintf("cause type: %d, value: %d", 
						transaction.ErrorCause.CauseType, transaction.ErrorCause.CauseValue)
				}
				return nil, fmt.Errorf("E2 Setup failed: %s", cause)
			}
		}
	}
}

func (h *E2APProcedureHandler) waitForConfigResponse(ctx context.Context, transactionID string) error {
	timeout := time.NewTimer(h.configTimeout)
	defer timeout.Stop()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			h.updateTransactionState(transactionID, E2APStateTimeout, nil)
			return fmt.Errorf("E2 Configuration Update procedure timeout")
		case <-ticker.C:
			h.mu.RLock()
			transaction, exists := h.transactions[transactionID]
			h.mu.RUnlock()
			
			if !exists {
				return fmt.Errorf("transaction not found")
			}
			
			switch transaction.State {
			case E2APStateCompleted:
				return nil
			case E2APStateFailed:
				cause := "unknown error"
				if transaction.ErrorCause != nil {
					cause = fmt.Sprintf("cause type: %d, value: %d", 
						transaction.ErrorCause.CauseType, transaction.ErrorCause.CauseValue)
				}
				return fmt.Errorf("E2 Configuration Update failed: %s", cause)
			}
		}
	}
}

func (h *E2APProcedureHandler) waitForResetResponse(ctx context.Context, transactionID string) error {
	timeout := time.NewTimer(h.resetTimeout)
	defer timeout.Stop()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			h.updateTransactionState(transactionID, E2APStateTimeout, nil)
			return fmt.Errorf("E2 Reset procedure timeout")
		case <-ticker.C:
			h.mu.RLock()
			transaction, exists := h.transactions[transactionID]
			h.mu.RUnlock()
			
			if !exists {
				return fmt.Errorf("transaction not found")
			}
			
			switch transaction.State {
			case E2APStateCompleted:
				return nil
			case E2APStateFailed:
				cause := "unknown error"
				if transaction.ErrorCause != nil {
					cause = fmt.Sprintf("cause type: %d, value: %d", 
						transaction.ErrorCause.CauseType, transaction.ErrorCause.CauseValue)
				}
				return fmt.Errorf("E2 Reset failed: %s", cause)
			}
		}
	}
}

func (h *E2APProcedureHandler) handleSetupResponse(transactionID string, msg *E2APMessage) error {
	h.mu.Lock()
	transaction, exists := h.transactions[transactionID]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("transaction %s not found", transactionID)
	}
	
	transaction.Response = msg
	h.mu.Unlock()
	
	if msg.PDUType == E2AP_PDU_SUCCESSFUL_OUTCOME {
		h.updateTransactionState(transactionID, E2APStateCompleted, nil)
		log.Printf("E2 Setup completed successfully for transaction %s", transactionID)
	} else if msg.PDUType == E2AP_PDU_UNSUCCESSFUL_OUTCOME {
		cause := &E2APCause{
			CauseType:  E2AP_CAUSE_E2_NODE,
			CauseValue: 1, // setup-failed
		}
		h.updateTransactionState(transactionID, E2APStateFailed, cause)
		log.Printf("E2 Setup failed for transaction %s", transactionID)
	}
	
	return nil
}

func (h *E2APProcedureHandler) handleConfigResponse(transactionID string, msg *E2APMessage) error {
	h.mu.Lock()
	transaction, exists := h.transactions[transactionID]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("transaction %s not found", transactionID)
	}
	
	transaction.Response = msg
	h.mu.Unlock()
	
	if msg.PDUType == E2AP_PDU_SUCCESSFUL_OUTCOME {
		h.updateTransactionState(transactionID, E2APStateCompleted, nil)
		log.Printf("E2 Configuration Update completed successfully for transaction %s", transactionID)
	} else {
		cause := &E2APCause{
			CauseType:  E2AP_CAUSE_E2_NODE,
			CauseValue: 2, // configuration-update-failed
		}
		h.updateTransactionState(transactionID, E2APStateFailed, cause)
		log.Printf("E2 Configuration Update failed for transaction %s", transactionID)
	}
	
	return nil
}

func (h *E2APProcedureHandler) handleResetResponse(transactionID string, msg *E2APMessage) error {
	h.mu.Lock()
	transaction, exists := h.transactions[transactionID]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("transaction %s not found", transactionID)
	}
	
	transaction.Response = msg
	h.mu.Unlock()
	
	h.updateTransactionState(transactionID, E2APStateCompleted, nil)
	log.Printf("E2 Reset completed for transaction %s", transactionID)
	
	return nil
}

func (h *E2APProcedureHandler) handleIndication(msg *E2APMessage) error {
	// Handle RIC Indication message
	log.Printf("Received RIC Indication message")
	// Process indication and forward to appropriate xApps
	return nil
}

func (h *E2APProcedureHandler) handleControlResponse(transactionID string, msg *E2APMessage) error {
	// Handle RIC Control response
	log.Printf("Received RIC Control response for transaction %s", transactionID)
	return nil
}

func (h *E2APProcedureHandler) parseSetupResponse(msg *E2APMessage) (*E2SetupResponseMessage, error) {
	// Parse E2 Setup Response from E2AP message
	response := &E2SetupResponseMessage{}
	
	if transactionID, ok := msg.Value["transactionId"].(uint32); ok {
		response.TransactionID = transactionID
	}
	
	if globalRICID, ok := msg.Value["globalRicId"].(GlobalRICID); ok {
		response.GlobalRICID = globalRICID
	}
	
	// Parse accepted and rejected RAN functions
	if accepted, ok := msg.Value["ranFunctionsAccepted"].([]RANFunctionIDItem); ok {
		response.RANFunctionsAccepted = accepted
	}
	
	if rejected, ok := msg.Value["ranFunctionsRejected"].([]RANFunctionIDCauseItem); ok {
		response.RANFunctionsRejected = rejected
	}
	
	return response, nil
}

// NewE2APMessageValidator creates a new E2AP message validator
func NewE2APMessageValidator() *E2APMessageValidator {
	return &E2APMessageValidator{
		encoder: NewE2APEncoder(),
	}
}

// ValidateE2APMessage validates an E2AP message for conformance
func (v *E2APMessageValidator) ValidateE2APMessage(msg *E2APMessage) error {
	// Validate PDU type
	if msg.PDUType > E2AP_PDU_UNSUCCESSFUL_OUTCOME {
		return fmt.Errorf("invalid PDU type: %d", msg.PDUType)
	}
	
	// Validate procedure code
	if msg.ProcedureCode < E2AP_PROCEDURE_E2_SETUP || msg.ProcedureCode > E2AP_PROCEDURE_E2_CONNECTION_UPDATE {
		return fmt.Errorf("invalid procedure code: %d", msg.ProcedureCode)
	}
	
	// Validate criticality
	if msg.Criticality > E2AP_CRITICALITY_NOTIFY {
		return fmt.Errorf("invalid criticality: %d", msg.Criticality)
	}
	
	// Validate message structure based on procedure code
	switch msg.ProcedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		return v.validateE2SetupMessage(msg)
	case E2AP_PROCEDURE_E2_NODE_CONFIG_UPDATE:
		return v.validateConfigUpdateMessage(msg)
	case E2AP_PROCEDURE_RESET:
		return v.validateResetMessage(msg)
	}
	
	return nil
}

func (v *E2APMessageValidator) validateE2SetupMessage(msg *E2APMessage) error {
	if msg.PDUType == E2AP_PDU_INITIATING_MESSAGE {
		// Validate E2 Setup Request
		if _, ok := msg.Value["transactionId"]; !ok {
			return fmt.Errorf("missing transaction ID in E2 Setup Request")
		}
		if _, ok := msg.Value["globalE2NodeId"]; !ok {
			return fmt.Errorf("missing global E2 node ID in E2 Setup Request")
		}
		if _, ok := msg.Value["ranFunctions"]; !ok {
			return fmt.Errorf("missing RAN functions in E2 Setup Request")
		}
	}
	return nil
}

func (v *E2APMessageValidator) validateConfigUpdateMessage(msg *E2APMessage) error {
	if msg.PDUType == E2AP_PDU_INITIATING_MESSAGE {
		// Validate E2 Node Configuration Update
		if _, ok := msg.Value["transactionId"]; !ok {
			return fmt.Errorf("missing transaction ID in E2 Configuration Update")
		}
	}
	return nil
}

func (v *E2APMessageValidator) validateResetMessage(msg *E2APMessage) error {
	if msg.PDUType == E2AP_PDU_INITIATING_MESSAGE {
		// Validate E2 Reset Request
		if _, ok := msg.Value["transactionId"]; !ok {
			return fmt.Errorf("missing transaction ID in E2 Reset Request")
		}
		if _, ok := msg.Value["cause"]; !ok {
			return fmt.Errorf("missing cause in E2 Reset Request")
		}
	}
	return nil
}