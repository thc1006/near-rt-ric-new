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

	"github.com/ishidawataru/sctp"
)

// E2NodeSimulatorConfig represents configuration for E2 node simulator
type E2NodeSimulatorConfig struct {
	NodeID         string
	PlmnID         []byte
	NodeType       string
	RICAddress     string
	RICPort        uint32
	LocalAddress   string
	LocalPort      uint32
	IndicationRate time.Duration
	AutoConnect    bool
	RANFunctions   []RANFunction
	ServiceModels  []ServiceModel
}

// NewE2NodeSimulator creates a new E2 node simulator instance
func NewE2NodeSimulator(config *E2NodeSimulatorConfig) *E2NodeSimulator {
	ctx, cancel := context.WithCancel(context.Background())
	
	simulator := &E2NodeSimulator{
		nodeID:         config.NodeID,
		ricAddress:     config.RICAddress,
		ricPort:        config.RICPort,
		localAddress:   config.LocalAddress,
		localPort:      config.LocalPort,
		subscriptions:  make(map[string]*SimulatedSubscription),
		ctx:            ctx,
		cancel:         cancel,
		encoder:        NewE2APEncoder(),
	}
	
	// Initialize Global E2 Node ID - fix field names to match GlobalE2NodeID struct
	simulator.globalE2NodeID = GlobalE2NodeID{
		PLMNIdentity: config.PlmnID, // Fix: PlmnID -> PLMNIdentity
		E2NodeID:     []byte(config.NodeID), // Fix: NodeID -> E2NodeID and convert to []byte
		// Remove Type field as it doesn't exist in GlobalE2NodeID struct
	}
	
	// Initialize RAN functions based on configuration
	simulator.initializeRANFunctions(config)
	
	// Initialize service models
	simulator.initializeServiceModels(config)
	
	return simulator
}

// Start starts the E2 node simulator
func (sim *E2NodeSimulator) Start(config *E2NodeSimulatorConfig) error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if sim.isRunning {
		return fmt.Errorf("E2 node simulator is already running")
	}
	
	// Start indication generation
	if config.IndicationRate > 0 {
		sim.indicationTicker = time.NewTicker(config.IndicationRate) // Add indicationTicker field to E2NodeSimulator
		go sim.indicationGenerator()
	}
	
	// Auto-connect if configured
	if config.AutoConnect {
		go func() {
			time.Sleep(2 * time.Second) // Give some time for RIC to be ready
			if err := sim.Connect(); err != nil {
				log.Printf("Failed to auto-connect: %v", err)
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
		return fmt.Errorf("failed to connect to RIC: %w", err)
	}
	
	sim.conn = conn
	sim.isConnected = true
	
	// Start message handling
	go sim.messageHandler()
	
	// Send E2 Setup Request
	if err := sim.sendE2SetupRequest(); err != nil {
		sim.disconnect()
		return fmt.Errorf("failed to send E2 setup request: %w", err)
	}
	
	log.Printf("E2 node %s connected to RIC at %s", sim.nodeID, ricAddr)
	return nil
}

// disconnect closes the SCTP connection (lowercase method name)
func (sim *E2NodeSimulator) disconnect() error {
	if sim.conn != nil {
		if sctpConn, ok := sim.conn.(*sctp.SCTPConn); ok {
			if err := sctpConn.Close(); err != nil {
				return fmt.Errorf("failed to close SCTP connection: %w", err)
			}
		}
		sim.conn = nil
	}
	sim.isConnected = false
	return nil
}

// Disconnect public method that calls private disconnect method
func (sim *E2NodeSimulator) Disconnect() error {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	return sim.disconnect()
}

// IsConnected returns connection status
func (sim *E2NodeSimulator) IsConnected() bool {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	return sim.isConnected
}

// IsRunning returns running status
func (sim *E2NodeSimulator) IsRunning() bool {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	return sim.isRunning
}

// GetNodeID returns the node ID
func (sim *E2NodeSimulator) GetNodeID() string {
	return sim.nodeID
}

// GetStatus returns the status of the simulator - Fix: Add missing GetStatus method
func (sim *E2NodeSimulator) GetStatus() map[string]interface{} {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	return map[string]interface{}{
		"nodeId":        sim.nodeID,
		"isRunning":     sim.isRunning,
		"isConnected":   sim.isConnected,
		"subscriptions": len(sim.subscriptions),
		"ranFunctions":  len(sim.ranFunctions),
		"serviceModels": len(sim.serviceModels),
	}
}

// GetRANFunctions returns supported RAN functions
func (sim *E2NodeSimulator) GetRANFunctions() []RANFunction {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	return append([]RANFunction(nil), sim.ranFunctions...)
}

// GetServiceModels returns supported service models
func (sim *E2NodeSimulator) GetServiceModels() []ServiceModel {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	return append([]ServiceModel(nil), sim.serviceModels...)
}

// GetSubscriptions returns current subscriptions
func (sim *E2NodeSimulator) GetSubscriptions() map[string]*SimulatedSubscription {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	subscriptions := make(map[string]*SimulatedSubscription)
	for id, sub := range sim.subscriptions {
		subscriptions[id] = &SimulatedSubscription{
			SubscriptionID:   sub.SubscriptionID, // Fix: ID -> SubscriptionID
			ServiceModelOID:  sub.ServiceModelOID, // Fix: Add ServiceModelOID field
			E2NodeID:         sub.E2NodeID, // Fix: Add E2NodeID field
			Actions:          sub.Actions, // Keep Actions as it exists
			ReportingPeriod:  sub.ReportingPeriod, // Fix: Add ReportingPeriod field
			IsActive:         sub.IsActive, // Fix: Status -> IsActive
			CreatedAt:        sub.CreatedAt, // Keep CreatedAt as it exists
		}
	}
	
	return subscriptions
}

// Private methods

func (sim *E2NodeSimulator) initializeRANFunctions(config *E2NodeSimulatorConfig) {
	if len(config.RANFunctions) > 0 {
		sim.ranFunctions = append([]RANFunction(nil), config.RANFunctions...)
	} else {
		// Default RAN functions
		sim.ranFunctions = []RANFunction{
			{
				ID:          1,
				OID:         "1.3.6.1.4.1.53148.1.2.2.2",
				Revision:    1,
				Description: "KPM Service Model",
			},
			{
				ID:          2,
				OID:         "1.3.6.1.4.1.53148.1.2.2.3",
				Revision:    1,
				Description: "RC Service Model",
			},
		}
	}
}

func (sim *E2NodeSimulator) initializeServiceModels(config *E2NodeSimulatorConfig) {
	if len(config.ServiceModels) > 0 {
		sim.serviceModels = append([]ServiceModel(nil), config.ServiceModels...)
	} else {
		// Default service models
		sim.serviceModels = []ServiceModel{
			{
				OID:         "1.3.6.1.4.1.53148.1.2.2.2",
				Name:        "ORAN-E2SM-KPM",
				Version:     "v02.00",
				Description: "Key Performance Measurement",
			},
			{
				OID:         "1.3.6.1.4.1.53148.1.2.2.3",
				Name:        "ORAN-E2SM-RC",
				Version:     "v01.02",
				Description: "RAN Control",
			},
		}
	}
}

func (sim *E2NodeSimulator) sendE2SetupRequest() error {
	// Encode the message using simplified mock approach
	if sim.encoder == nil {
		return fmt.Errorf("encoder not initialized")
	}
	
	// For now, create a simple mock message since existing encoder has interface mismatches
	encodedMsg, err := sim.createMockE2SetupRequest()
	if err != nil {
		return fmt.Errorf("failed to create E2 setup request: %w", err)
	}
	
	// Send the message
	return sim.sendMessage(encodedMsg)
}

func (sim *E2NodeSimulator) createMockE2SetupRequest() ([]byte, error) {
	// Create a simple mock E2AP setup request message
	mockMessage := make([]byte, 100)
	binary.BigEndian.PutUint32(mockMessage[0:4], uint32(E2APMessageTypeSetupRequest))
	binary.BigEndian.PutUint32(mockMessage[4:8], 1) // transaction ID
	return mockMessage, nil
}

func (sim *E2NodeSimulator) sendMessage(message []byte) error {
	if !sim.isConnected || sim.conn == nil {
		return fmt.Errorf("not connected")
	}
	
	if sctpConn, ok := sim.conn.(*sctp.SCTPConn); ok {
		_, err := sctpConn.Write(message)
		return err
	}
	
	return fmt.Errorf("invalid connection type")
}

func (sim *E2NodeSimulator) messageHandler() {
	if !sim.isConnected || sim.conn == nil {
		return
	}
	
	sctpConn, ok := sim.conn.(*sctp.SCTPConn)
	if !ok {
		log.Printf("Invalid connection type for message handler")
		return
	}
	
	buffer := make([]byte, 4096)
	
	for sim.isConnected {
		n, err := sctpConn.Read(buffer)
		if err != nil {
			if sim.isConnected {
				log.Printf("Error reading from SCTP connection: %v", err)
			}
			break
		}
		
		if n > 0 {
			go sim.processMessage(buffer[:n])
		}
	}
}

func (sim *E2NodeSimulator) processMessage(message []byte) {
	// Basic message processing - in a real implementation, this would
	// decode E2AP messages and handle different message types
	
	if len(message) < 4 {
		return
	}
	
	// For simulation purposes, just log the message
	messageType := binary.BigEndian.Uint32(message[:4])
	log.Printf("E2 node %s received message type: %d", sim.nodeID, messageType)
}

func (sim *E2NodeSimulator) indicationGenerator() {
	if sim.indicationTicker == nil {
		return
	}
	
	for {
		select {
		case <-sim.indicationTicker.C:
			sim.generateIndications()
		case <-sim.ctx.Done():
			return
		}
	}
}

func (sim *E2NodeSimulator) generateIndications() {
	sim.mu.RLock()
	subscriptions := make([]*SimulatedSubscription, 0, len(sim.subscriptions))
	for _, sub := range sim.subscriptions {
		if sub.IsActive { // Fix: Status -> IsActive
			subscriptions = append(subscriptions, sub)
		}
	}
	sim.mu.RUnlock()
	
	// Generate indications for active subscriptions
	for _, sub := range subscriptions {
		indication := sim.createIndication(sub)
		if err := sim.sendIndication(indication); err != nil {
			log.Printf("Failed to send indication for subscription %s: %v", sub.SubscriptionID, err)
		}
	}
}

func (sim *E2NodeSimulator) createIndication(sub *SimulatedSubscription) *Indication {
	return &Indication{
		E2NodeID:      sim.nodeID,
		RANFunctionID: 1, // Default RAN function ID
		ActionID:      sub.Actions[0].ActionID, // Use first action ID
		IndicationSN:  1,
		IndicationHeader: []byte("simulation-header"),
		IndicationMessage: []byte("simulation-data"),
		Timestamp:     time.Now(),
	}
}

func (sim *E2NodeSimulator) sendIndication(indication *Indication) error {
	// In a real implementation, this would encode the indication as E2AP message
	// For simulation, just log it
	log.Printf("Sending indication from E2 node %s for RAN function %d", 
		indication.E2NodeID, indication.RANFunctionID)
	return nil
}