package e2ap

import (
	"fmt"
	"sync"
)

// E2APMessage represents an E2AP protocol message
type E2APMessage struct {
	Type    uint32
	Payload []byte
	ID      uint32
	Source  string
}

// E2APMessageType defines standard E2AP message types
type E2APMessageType int

const (
	E2_SETUP_REQUEST E2APMessageType = iota
	E2_SETUP_RESPONSE
	E2_NODE_CONFIG_UPDATE
	RIC_SUBSCRIPTION_REQUEST
	RIC_SUBSCRIPTION_RESPONSE
	RIC_INDICATION
	RIC_CONTROL_REQUEST
	RIC_CONTROL_RESPONSE
)

// E2APMessageHandler manages E2AP message processing
type E2APMessageHandler struct {
	mu                sync.RWMutex
	messageProcessors map[E2APMessageType]MessageProcessor
	defaultProcessor  MessageProcessor
	errorHandler      ErrorHandler
}

// MessageProcessor defines the signature for message processing functions
type MessageProcessor func(msg *E2APMessage) error

// ErrorHandler defines the signature for error handling functions
type ErrorHandler func(err error, msg *E2APMessage)

// NewE2APMessageHandler creates a new E2AP message handler
func NewE2APMessageHandler() *E2APMessageHandler {
	return &E2APMessageHandler{
		messageProcessors: make(map[E2APMessageType]MessageProcessor),
	}
}

// RegisterProcessor adds a message processor for a specific E2AP message type
func (h *E2APMessageHandler) RegisterProcessor(msgType E2APMessageType, processor MessageProcessor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messageProcessors[msgType] = processor
}

// SetDefaultProcessor sets a default processor for unregistered message types
func (h *E2APMessageHandler) SetDefaultProcessor(processor MessageProcessor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.defaultProcessor = processor
}

// SetErrorHandler configures a custom error handler
func (h *E2APMessageHandler) SetErrorHandler(handler ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorHandler = handler
}

// ProcessMessage handles incoming E2AP messages
func (h *E2APMessageHandler) ProcessMessage(msg *E2APMessage) error {
	h.mu.RLock()
	processor, ok := h.messageProcessors[E2APMessageType(msg.Type)]
	if !ok && h.defaultProcessor != nil {
		processor = h.defaultProcessor
	}
	h.mu.RUnlock()

	if processor == nil {
		err := fmt.Errorf("no processor found for message type %v", msg.Type)
		h.handleError(err, msg)
		return err
	}

	// Process the message
	err := processor(msg)
	if err != nil {
		h.handleError(err, msg)
		return err
	}

	return nil
}

// handleError manages error processing using the configured error handler
func (h *E2APMessageHandler) handleError(err error, msg *E2APMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.errorHandler != nil {
		h.errorHandler(err, msg)
	} else {
		// Default error logging
		fmt.Printf("E2AP Message Processing Error: %v\n", err)
	}
}

// CreateErrorResponse generates a standardized E2AP error response
func (h *E2APMessageHandler) CreateErrorResponse(originalMsg *E2APMessage, errorCause string) *E2APMessage {
	return &E2APMessage{
		Type: uint32(RIC_CONTROL_RESPONSE),
		Payload: []byte(fmt.Sprintf(`{
			"status": "error",
			"original_type": %d,
			"error_cause": "%s"
		}`, originalMsg.Type, errorCause)),
		ID:     originalMsg.ID,
		Source: "ric-e2term",
	}
}

// ValidateMessage performs basic validation on E2AP messages
func (h *E2APMessageHandler) ValidateMessage(msg *E2APMessage) error {
	if msg == nil {
		return fmt.Errorf("nil E2AP message")
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("empty message payload")
	}

	// Add more specific validation based on message type
	switch E2APMessageType(msg.Type) {
	case E2_SETUP_REQUEST:
		// Validate E2 Setup specific requirements
		return h.validateE2SetupRequest(msg)
	case RIC_SUBSCRIPTION_REQUEST:
		// Validate Subscription Request specific requirements
		return h.validateSubscriptionRequest(msg)
	default:
		return nil
	}
}

// validateE2SetupRequest performs specific validation for E2 Setup messages
func (h *E2APMessageHandler) validateE2SetupRequest(msg *E2APMessage) error {
	// Implement specific E2 Setup validation logic
	if len(msg.Payload) < 10 {
		return fmt.Errorf("E2 Setup request payload too short")
	}
	return nil
}

// validateSubscriptionRequest performs specific validation for Subscription Requests
func (h *E2APMessageHandler) validateSubscriptionRequest(msg *E2APMessage) error {
	// Implement specific Subscription Request validation logic
	if len(msg.Payload) < 20 {
		return fmt.Errorf("RIC Subscription request payload insufficient")
	}
	return nil
}

// CreateSetupResponse creates a standardized E2 Setup Response
func (h *E2APMessageHandler) CreateSetupResponse(ricID string, setupMsg *E2APMessage) *E2APMessage {
	return &E2APMessage{
		Type: uint32(E2_SETUP_RESPONSE),
		Payload: []byte(fmt.Sprintf(`{
			"ric_id": "%s",
			"status": "success",
			"global_ric_id": {"plmn_id": "001", "ric_id": "%s"},
			"ran_functions_accepted": []
		}`, ricID, ricID)),
		ID:     setupMsg.ID,
		Source: "ric-e2mgr",
	}
}