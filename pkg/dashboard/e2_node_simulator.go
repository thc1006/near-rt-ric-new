/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ishidawataru/sctp"
)

// E2NodeSimulator simulates an E2 node for testing and development
type E2NodeSimulator struct {
	mu                sync.RWMutex
	nodeID            string
	globalE2NodeID    GlobalE2NodeID
	ricAddress        string
	ricPort           uint32
	localAddress      string
	localPort         uint32
	
	// SCTP connection
	conn              *sctp.SCTPConn
	isConnected       bool
	
	// E2AP protocol handler
	protocolHandler   *E2APProcedureHandler
	encoder           *E2APEncoder
	
	// Simulation state
	isRunning         bool
	ctx               context.Context
	cancel            context.CancelFunc
	
	// RAN Functions
	ranFunctions      []RANFunction
	serviceModels     []ServiceModel
	
	// Subscription management
	subscriptions     map[string]*SimulatedSubscription
	
	// Indication generation
	indicationTicker  *time.Ticker
	indicationRate    time.Duration
	
	// Configuration
	config            *E2NodeSimulatorConfig
}

// E2NodeSimulatorConfig represents configuration for E2 node simulator
type E2NodeSimulatorConfig struct {
	NodeID            string                 `json:"nodeId"`
	NodeType          E2NodeType            `json:"nodeType"`
	PlmnID            string                `json:"plmnId"`
	RICAddress        string                `json:"ricAddress"`
	RICPort           uint32                `json:"ricPort"`
	LocalAddress      string                `json:"localAddress"`
	LocalPort         uint32                `json:"localPort"`
	RANFunctions      []RANFunctionConfig   `json:"ranFunctions"`
	IndicationRate    time.Duration         `json:"indicationRate"`
	AutoConnect       bool                  `json:"autoConnect"`
	EnableKPM         bool                  `json:"enableKpm"`
	EnableRC          bool                  `json:"enableRc"`
	EnableNI          bool                  `json:"enableNi"`
}

// RANFunctionConfig represents RAN function configuration
type RANFunctionConfig struct {
	ID          uint32 `json:"id"`
	OID         string `json:"oid"`
	Description string `json:"description"`
	Revision    uint32 `json:"revision"`
}

// SimulatedSubscription represents a simulated subscription
type SimulatedSubscription struct {
	ID               string                `json:"id"`
	RANFunctionID    uint32               `json:"ranFunctionId"`
	EventTrigger     EventTrigger         `json:"eventTrigger"`
	Actions          []Action             `json:"actions"`
	Status           string               `json:"status"`
	CreatedAt        time.Time            `json:"createdAt"`
	LastIndication   time.Time            `json:"lastIndication"`
	IndicationCount  uint64               `json:"indicationCount"`
}

// NewE2NodeSimulator creates a new E2 node simulator
func NewE2NodeSimulator(config *E2NodeSimulatorConfig) *E2NodeSimulator {
	ctx, cancel := context.WithCancel(context.Background())
	
	simulator := &E2NodeSimulator{
		nodeID:           config.NodeID,
		ricAddress:       config.RICAddress,
		ricPort:          config.RICPort,
		localAddress:     config.LocalAddress,
		localPort:        config.LocalPort,
		subscriptions:    make(map[string]*SimulatedSubscription),
		indicationRate:   config.IndicationRate,
		ctx:              ctx,
		cancel:           cancel,
		config:           config,
		encoder:          NewE2APEncoder(),
	}
	
	// Initialize Global E2 Node ID
	simulator.globalE2NodeID = GlobalE2NodeID{
		PlmnID: config.PlmnID,
		NodeID: config.NodeID,
		Type:   config.NodeType,
	}
	
	// Initialize RAN functions based on configuration
	simulator.initializeRANFunctions()
	
	// Initialize service models
	simulator.initializeServiceModels()
	
	return simulator
}

// Start starts the E2 node simulator
func (sim *E2NodeSimulator) Start() error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if sim.isRunning {
		return fmt.Errorf("E2 node simulator is already running")
	}
	
	// Start indication generation
	if sim.indicationRate > 0 {
		sim.indicationTicker = time.NewTicker(sim.indicationRate)
		go sim.indicationGenerator()
	}
	
	// Auto-connect if configured
	if sim.config.AutoConnect {
		go func() {
			time.Sleep(2 * time.Second) // Give some time for RIC to be ready
			if err := sim.Connect(); err != nil {
				log.Printf("Failed to auto-connect E2 node %s: %v", sim.nodeID, err)
			}
		}()
	}
	
	sim.isRunning = true
	log.Printf("E2 node simulator %s started", sim.nodeID)
	return nil
}

// Stop stops the E2 node simulator
func (sim *E2NodeSimulator) Stop() error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if !sim.isRunning {
		return nil
	}
	
	sim.cancel()
	
	// Stop indication generation
	if sim.indicationTicker != nil {
		sim.indicationTicker.Stop()
	}
	
	// Disconnect from RIC
	if sim.isConnected {
		sim.disconnect()
	}
	
	sim.isRunning = false
	log.Printf("E2 node simulator %s stopped", sim.nodeID)
	return nil
}

// Connect establishes SCTP connection with E2T
func (sim *E2NodeSimulator) Connect() error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if sim.isConnected {
		return fmt.Errorf("already connected")
	}
	
	// Resolve addresses
	ricAddr, err := sctp.ResolveSCTPAddr("sctp", fmt.Sprintf("%s:%d", sim.ricAddress, sim.ricPort))
	if err != nil {
		return fmt.Errorf("failed to resolve RIC address: %w", err)
	}
	
	localAddr, err := sctp.ResolveSCTPAddr("sctp", fmt.Sprintf("%s:%d", sim.localAddress, sim.localPort))
	if err != nil {
		return fmt.Errorf("failed to resolve local address: %w", err)
	}
	
	// Establish SCTP connection
	conn, err := sctp.DialSCTP("sctp", localAddr, ricAddr)
	if err != nil {
		return fmt.Errorf("failed to establish SCTP connection: %w", err)
	}
	
	sim.conn = conn
	sim.isConnected = true
	
	log.Printf("E2 node %s connected to RIC at %s:%d", sim.nodeID, sim.ricAddress, sim.ricPort)
	
	// Start message receiver
	go sim.messageReceiver()
	
	// Initiate E2 Setup procedure
	go func() {
		time.Sleep(1 * time.Second) // Brief delay
		if err := sim.sendE2SetupRequest(); err != nil {
			log.Printf("Failed to send E2 Setup Request: %v", err)
		}
	}()
	
	return nil
}

// Disconnect closes the SCTP connection
func (sim *E2NodeSimulator) Disconnect() error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	return sim.disconnect()
}

func (sim *E2NodeSimulator) disconnect() error {
	if !sim.isConnected {
		return nil
	}
	
	if sim.conn != nil {
		if err := sim.conn.Close(); err != nil {
			log.Printf("Error closing SCTP connection: %v", err)
		}
		sim.conn = nil
	}
	
	sim.isConnected = false
	log.Printf("E2 node %s disconnected from RIC", sim.nodeID)
	return nil
}

// GetStatus returns the current status of the simulator
func (sim *E2NodeSimulator) GetStatus() map[string]interface{} {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	return map[string]interface{}{
		"nodeId":           sim.nodeID,
		"isRunning":        sim.isRunning,
		"isConnected":      sim.isConnected,
		"ricAddress":       fmt.Sprintf("%s:%d", sim.ricAddress, sim.ricPort),
		"localAddress":     fmt.Sprintf("%s:%s", sim.localAddress, sim.localPort),
		"ranFunctions":     len(sim.ranFunctions),
		"subscriptions":    len(sim.subscriptions),
		"globalE2NodeId":   sim.globalE2NodeID,
	}
}

// GetSubscriptions returns current subscriptions
func (sim *E2NodeSimulator) GetSubscriptions() map[string]*SimulatedSubscription {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	subscriptions := make(map[string]*SimulatedSubscription)
	for id, sub := range sim.subscriptions {
		subscriptions[id] = &SimulatedSubscription{
			ID:              sub.ID,
			RANFunctionID:   sub.RANFunctionID,
			EventTrigger:    sub.EventTrigger,
			Actions:         sub.Actions,
			Status:          sub.Status,
			CreatedAt:       sub.CreatedAt,
			LastIndication:  sub.LastIndication,
			IndicationCount: sub.IndicationCount,
		}
	}
	
	return subscriptions
}

// Private methods

func (sim *E2NodeSimulator) initializeRANFunctions() {
	sim.ranFunctions = make([]RANFunction, 0)
	
	for _, funcConfig := range sim.config.RANFunctions {
		ranFunc := RANFunction{
			ID:          funcConfig.ID,
			OID:         funcConfig.OID,
			Description: funcConfig.Description,
			Revision:    funcConfig.Revision,
			Definition:  sim.generateRANFunctionDefinition(funcConfig),
		}
		sim.ranFunctions = append(sim.ranFunctions, ranFunc)
	}
	
	// Add default RAN functions if none configured
	if len(sim.ranFunctions) == 0 {
		if sim.config.EnableKPM {
			sim.ranFunctions = append(sim.ranFunctions, RANFunction{
				ID:          1,
				OID:         "1.3.6.1.4.1.53148.1.2.2.2",
				Description: "E2SM-KPM RAN Function",
				Revision:    1,
				Definition:  sim.generateKPMRANFunctionDefinition(),
			})
		}
		
		if sim.config.EnableRC {
			sim.ranFunctions = append(sim.ranFunctions, RANFunction{
				ID:          2,
				OID:         "1.3.6.1.4.1.53148.1.2.2.3",
				Description: "E2SM-RC RAN Function",
				Revision:    1,
				Definition:  sim.generateRCRANFunctionDefinition(),
			})
		}
		
		if sim.config.EnableNI {
			sim.ranFunctions = append(sim.ranFunctions, RANFunction{
				ID:          3,
				OID:         "1.3.6.1.4.1.53148.1.2.2.4",
				Description: "E2SM-NI RAN Function",
				Revision:    1,
				Definition:  sim.generateNIRANFunctionDefinition(),
			})
		}
	}
}

func (sim *E2NodeSimulator) initializeServiceModels() {
	sim.serviceModels = make([]ServiceModel, 0)
	
	if sim.config.EnableKPM {
		sim.serviceModels = append(sim.serviceModels, ServiceModel{
			OID:         "1.3.6.1.4.1.53148.1.2.2.2",
			Name:        "E2SM-KPM",
			Version:     "v2.0",
			Description: "Key Performance Measurement Service Model",
			Functions:   []RANFunction{sim.ranFunctions[0]}, // Assuming KPM is first
		})
	}
	
	if sim.config.EnableRC {
		sim.serviceModels = append(sim.serviceModels, ServiceModel{
			OID:         "1.3.6.1.4.1.53148.1.2.2.3",
			Name:        "E2SM-RC",
			Version:     "v1.0",
			Description: "RAN Control Service Model",
			Functions:   []RANFunction{sim.ranFunctions[1]}, // Assuming RC is second
		})
	}
	
	if sim.config.EnableNI {
		sim.serviceModels = append(sim.serviceModels, ServiceModel{
			OID:         "1.3.6.1.4.1.53148.1.2.2.4",
			Name:        "E2SM-NI",
			Version:     "v1.0",
			Description: "Network Interface Service Model",
			Functions:   []RANFunction{sim.ranFunctions[2]}, // Assuming NI is third
		})
	}
}

func (sim *E2NodeSimulator) sendE2SetupRequest() error {
	transactionID := rand.Uint32()
	
	// Create RAN function items for E2 Setup
	ranFunctionItems := make([]RANFunctionItem, len(sim.ranFunctions))
	for i, ranFunc := range sim.ranFunctions {
		ranFunctionItems[i] = RANFunctionItem{
			RANFunctionID:         ranFunc.ID,
			RANFunctionDefinition: ranFunc.Definition,
			RANFunctionRevision:   ranFunc.Revision,
			RANFunctionOID:        ranFunc.OID,
		}
	}
	
	// Create E2 Setup Request
	setupReq := &E2SetupRequestMessage{
		TransactionID:  transactionID,
		GlobalE2NodeID: sim.globalE2NodeID,
		RANFunctions:   ranFunctionItems,
		E2NodeComponentConfigAddList: []E2NodeComponentConfigAddItem{
			{
				E2NodeComponentInterfaceType: E2NodeComponentInterfaceTypeNG,
				E2NodeComponentID: E2NodeComponentID{
					E2NodeComponentTypeNG: &E2NodeComponentIDNG{
						AMFID: []byte{0x01, 0x02, 0x03},
					},
				},
				E2NodeComponentConfiguration: E2NodeComponentConfiguration{
					E2NodeComponentRequestPart:  []byte("request-part"),
					E2NodeComponentResponsePart: []byte("response-part"),
				},
			},
		},
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
	
	// Encode and send
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode E2 Setup Request: %w", err)
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		return fmt.Errorf("failed to send E2 Setup Request: %w", err)
	}
	
	log.Printf("E2 node %s sent E2 Setup Request (transaction: %d)", sim.nodeID, transactionID)
	return nil
}

func (sim *E2NodeSimulator) sendSCTPMessage(data []byte, streamID uint16) error {
	if !sim.isConnected || sim.conn == nil {
		return fmt.Errorf("not connected to RIC")
	}
	
	info := &sctp.SndRcvInfo{
		Stream: streamID,
		PPID:   1, // E2AP PPID
	}
	
	_, err := sim.conn.SCTPWrite(data, info)
	return err
}

func (sim *E2NodeSimulator) messageReceiver() {
	buffer := make([]byte, 4096)
	
	for sim.isConnected {
		n, info, err := sim.conn.SCTPRead(buffer)
		if err != nil {
			if sim.isConnected {
				log.Printf("Error reading SCTP message: %v", err)
			}
			break
		}
		
		if n > 0 {
			go sim.processIncomingMessage(buffer[:n], info)
		}
	}
}

func (sim *E2NodeSimulator) processIncomingMessage(data []byte, info *sctp.SndRcvInfo) {
	// Decode E2AP message
	msg, err := sim.encoder.DecodeE2APMessage(data)
	if err != nil {
		log.Printf("Failed to decode incoming E2AP message: %v", err)
		return
	}
	
	log.Printf("E2 node %s received E2AP message: procedure=%d, pdu=%d", 
		sim.nodeID, msg.ProcedureCode, msg.PDUType)
	
	// Handle based on procedure code
	switch msg.ProcedureCode {
	case E2AP_PROCEDURE_E2_SETUP:
		sim.handleE2SetupResponse(msg)
	case E2AP_PROCEDURE_RIC_SUBSCRIPTION:
		sim.handleRICSubscriptionRequest(msg)
	case E2AP_PROCEDURE_RIC_SUBSCRIPTION_DELETE:
		sim.handleRICSubscriptionDeleteRequest(msg)
	case E2AP_PROCEDURE_RIC_CONTROL:
		sim.handleRICControlRequest(msg)
	case E2AP_PROCEDURE_RESET:
		sim.handleResetRequest(msg)
	default:
		log.Printf("Unhandled E2AP procedure: %d", msg.ProcedureCode)
	}
}

func (sim *E2NodeSimulator) handleE2SetupResponse(msg *E2APMessage) {
	if msg.PDUType == E2AP_PDU_SUCCESSFUL_OUTCOME {
		log.Printf("E2 node %s: E2 Setup successful", sim.nodeID)
	} else if msg.PDUType == E2AP_PDU_UNSUCCESSFUL_OUTCOME {
		log.Printf("E2 node %s: E2 Setup failed", sim.nodeID)
	}
}

func (sim *E2NodeSimulator) handleRICSubscriptionRequest(msg *E2APMessage) {
	// Extract subscription details and create simulated subscription
	subscriptionID := uuid.New().String()
	
	subscription := &SimulatedSubscription{
		ID:              subscriptionID,
		RANFunctionID:   1, // Default to first RAN function
		Status:          "ACTIVE",
		CreatedAt:       time.Now(),
		IndicationCount: 0,
	}
	
	sim.mu.Lock()
	sim.subscriptions[subscriptionID] = subscription
	sim.mu.Unlock()
	
	log.Printf("E2 node %s: Created subscription %s", sim.nodeID, subscriptionID)
	
	// Send subscription response
	sim.sendRICSubscriptionResponse(subscriptionID, true)
}

func (sim *E2NodeSimulator) handleRICSubscriptionDeleteRequest(msg *E2APMessage) {
	// Extract subscription ID and delete subscription
	// For simulation, we'll just remove the first subscription
	sim.mu.Lock()
	for id := range sim.subscriptions {
		delete(sim.subscriptions, id)
		log.Printf("E2 node %s: Deleted subscription %s", sim.nodeID, id)
		break
	}
	sim.mu.Unlock()
	
	// Send subscription delete response
	sim.sendRICSubscriptionDeleteResponse(true)
}

func (sim *E2NodeSimulator) handleRICControlRequest(msg *E2APMessage) {
	log.Printf("E2 node %s: Received RIC Control Request", sim.nodeID)
	
	// Send control acknowledgment
	sim.sendRICControlAck()
}

func (sim *E2NodeSimulator) handleResetRequest(msg *E2APMessage) {
	log.Printf("E2 node %s: Received Reset Request", sim.nodeID)
	
	// Clear all subscriptions
	sim.mu.Lock()
	sim.subscriptions = make(map[string]*SimulatedSubscription)
	sim.mu.Unlock()
	
	// Send reset response
	sim.sendResetResponse()
}

func (sim *E2NodeSimulator) sendRICSubscriptionResponse(subscriptionID string, success bool) {
	// Create and send RIC Subscription Response
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_SUCCESSFUL_OUTCOME,
		ProcedureCode: E2AP_PROCEDURE_RIC_SUBSCRIPTION,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"subscriptionId": subscriptionID,
			"success":        success,
		},
	}
	
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		log.Printf("Failed to encode RIC Subscription Response: %v", err)
		return
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		log.Printf("Failed to send RIC Subscription Response: %v", err)
	}
}

func (sim *E2NodeSimulator) sendRICSubscriptionDeleteResponse(success bool) {
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_SUCCESSFUL_OUTCOME,
		ProcedureCode: E2AP_PROCEDURE_RIC_SUBSCRIPTION_DELETE,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"success": success,
		},
	}
	
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		log.Printf("Failed to encode RIC Subscription Delete Response: %v", err)
		return
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		log.Printf("Failed to send RIC Subscription Delete Response: %v", err)
	}
}

func (sim *E2NodeSimulator) sendRICControlAck() {
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_SUCCESSFUL_OUTCOME,
		ProcedureCode: E2AP_PROCEDURE_RIC_CONTROL,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"success": true,
		},
	}
	
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		log.Printf("Failed to encode RIC Control Ack: %v", err)
		return
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		log.Printf("Failed to send RIC Control Ack: %v", err)
	}
}

func (sim *E2NodeSimulator) sendResetResponse() {
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_SUCCESSFUL_OUTCOME,
		ProcedureCode: E2AP_PROCEDURE_RESET,
		Criticality:   E2AP_CRITICALITY_REJECT,
		Value: map[string]interface{}{
			"success": true,
		},
	}
	
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		log.Printf("Failed to encode Reset Response: %v", err)
		return
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		log.Printf("Failed to send Reset Response: %v", err)
	}
}

func (sim *E2NodeSimulator) indicationGenerator() {
	for {
		select {
		case <-sim.ctx.Done():
			return
		case <-sim.indicationTicker.C:
			sim.generateIndications()
		}
	}
}

func (sim *E2NodeSimulator) generateIndications() {
	sim.mu.RLock()
	subscriptions := make([]*SimulatedSubscription, 0, len(sim.subscriptions))
	for _, sub := range sim.subscriptions {
		if sub.Status == "ACTIVE" {
			subscriptions = append(subscriptions, sub)
		}
	}
	sim.mu.RUnlock()
	
	for _, sub := range subscriptions {
		sim.sendRICIndication(sub)
	}
}

func (sim *E2NodeSimulator) sendRICIndication(subscription *SimulatedSubscription) {
	// Generate indication data based on service model
	var indicationHeader, indicationMessage []byte
	
	switch subscription.RANFunctionID {
	case 1: // KPM
		indicationHeader, indicationMessage = sim.generateKPMIndication()
	case 2: // RC
		indicationHeader, indicationMessage = sim.generateRCIndication()
	case 3: // NI
		indicationHeader, indicationMessage = sim.generateNIIndication()
	default:
		indicationHeader, indicationMessage = sim.generateGenericIndication()
	}
	
	msg := &E2APMessage{
		PDUType:       E2AP_PDU_INITIATING_MESSAGE,
		ProcedureCode: E2AP_PROCEDURE_RIC_INDICATION,
		Criticality:   E2AP_CRITICALITY_IGNORE,
		Value: map[string]interface{}{
			"subscriptionId":     subscription.ID,
			"ranFunctionId":      subscription.RANFunctionID,
			"indicationHeader":   indicationHeader,
			"indicationMessage":  indicationMessage,
			"indicationSN":       subscription.IndicationCount + 1,
		},
	}
	
	encodedMsg, err := sim.encoder.EncodeE2APMessage(msg)
	if err != nil {
		log.Printf("Failed to encode RIC Indication: %v", err)
		return
	}
	
	if err := sim.sendSCTPMessage(encodedMsg, 0); err != nil {
		log.Printf("Failed to send RIC Indication: %v", err)
		return
	}
	
	// Update subscription statistics
	sim.mu.Lock()
	subscription.IndicationCount++
	subscription.LastIndication = time.Now()
	sim.mu.Unlock()
}

// RAN Function definition generators
func (sim *E2NodeSimulator) generateRANFunctionDefinition(config RANFunctionConfig) []byte {
	// Generate a simple RAN function definition
	return []byte(fmt.Sprintf("RAN-Function-Definition-%d-%s", config.ID, config.Description))
}

func (sim *E2NodeSimulator) generateKPMRANFunctionDefinition() []byte {
	return []byte("E2SM-KPM-RANfunction-Description")
}

func (sim *E2NodeSimulator) generateRCRANFunctionDefinition() []byte {
	return []byte("E2SM-RC-RANfunction-Description")
}

func (sim *E2NodeSimulator) generateNIRANFunctionDefinition() []byte {
	return []byte("E2SM-NI-RANfunction-Description")
}

// Indication generators
func (sim *E2NodeSimulator) generateKPMIndication() ([]byte, []byte) {
	header := []byte("KMP-Indication-Header")
	message := []byte(fmt.Sprintf("KMP-Indication-Message-%d", time.Now().Unix()))
	return header, message
}

func (sim *E2NodeSimulator) generateRCIndication() ([]byte, []byte) {
	header := []byte("RC-Indication-Header")
	message := []byte(fmt.Sprintf("RC-Indication-Message-%d", time.Now().Unix()))
	return header, message
}

func (sim *E2NodeSimulator) generateNIIndication() ([]byte, []byte) {
	header := []byte("NI-Indication-Header")
	message := []byte(fmt.Sprintf("NI-Indication-Message-%d", time.Now().Unix()))
	return header, message
}

func (sim *E2NodeSimulator) generateGenericIndication() ([]byte, []byte) {
	header := []byte("Generic-Indication-Header")
	message := []byte(fmt.Sprintf("Generic-Indication-Message-%d", time.Now().Unix()))
	return header, message
}