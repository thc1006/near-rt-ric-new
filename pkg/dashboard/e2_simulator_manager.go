/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// E2SimulatorManager manages multiple E2 node simulators
type E2SimulatorManager struct {
	mu         sync.RWMutex
	simulators map[string]*E2NodeSimulator
	isRunning  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// E2SimulatorManagerConfig represents configuration for the simulator manager
type E2SimulatorManagerConfig struct {
	Simulators []E2NodeSimulatorConfig `json:"simulators"`
}

// NewE2SimulatorManager creates a new E2 simulator manager
func NewE2SimulatorManager() *E2SimulatorManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &E2SimulatorManager{
		simulators: make(map[string]*E2NodeSimulator),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the simulator manager
func (mgr *E2SimulatorManager) Start() error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	
	if mgr.isRunning {
		return fmt.Errorf("simulator manager is already running")
	}
	
	mgr.isRunning = true
	log.Println("E2 simulator manager started")
	return nil
}

// Stop stops the simulator manager and all simulators
func (mgr *E2SimulatorManager) Stop() error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	
	if !mgr.isRunning {
		return nil
	}
	
	mgr.cancel()
	
	// Stop all simulators
	for nodeID, simulator := range mgr.simulators {
		if err := simulator.Stop(); err != nil {
			log.Printf("Error stopping simulator %s: %v", nodeID, err)
		}
	}
	
	mgr.isRunning = false
	log.Println("E2 simulator manager stopped")
	return nil
}

// CreateSimulator creates a new E2 node simulator
func (mgr *E2SimulatorManager) CreateSimulator(config *E2NodeSimulatorConfig) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	
	if _, exists := mgr.simulators[config.NodeID]; exists {
		return fmt.Errorf("simulator with node ID %s already exists", config.NodeID)
	}
	
	simulator := NewE2NodeSimulator(config)
	mgr.simulators[config.NodeID] = simulator
	
	// Start the simulator if manager is running
	if mgr.isRunning {
		// Fix: Add missing *E2NodeSimulatorConfig parameter
		if err := simulator.Start(config); err != nil {
			delete(mgr.simulators, config.NodeID)
			return fmt.Errorf("failed to start simulator: %w", err)
		}
	}
	
	log.Printf("Created E2 node simulator: %s", config.NodeID)
	return nil
}

// RemoveSimulator removes an E2 node simulator
func (mgr *E2SimulatorManager) RemoveSimulator(nodeID string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	
	simulator, exists := mgr.simulators[nodeID]
	if !exists {
		return fmt.Errorf("simulator with node ID %s not found", nodeID)
	}
	
	if err := simulator.Stop(); err != nil {
		log.Printf("Error stopping simulator %s: %v", nodeID, err)
	}
	
	delete(mgr.simulators, nodeID)
	log.Printf("Removed E2 node simulator: %s", nodeID)
	return nil
}

// GetSimulator returns a specific simulator
func (mgr *E2SimulatorManager) GetSimulator(nodeID string) (*E2NodeSimulator, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	
	simulator, exists := mgr.simulators[nodeID]
	if !exists {
		return nil, fmt.Errorf("simulator with node ID %s not found", nodeID)
	}
	
	return simulator, nil
}

// GetAllSimulators returns all simulators
func (mgr *E2SimulatorManager) GetAllSimulators() map[string]*E2NodeSimulator {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	
	simulators := make(map[string]*E2NodeSimulator)
	for nodeID, simulator := range mgr.simulators {
		simulators[nodeID] = simulator
	}
	
	return simulators
}

// GetSimulatorStatus returns status of all simulators
func (mgr *E2SimulatorManager) GetSimulatorStatus() map[string]interface{} {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	
	status := make(map[string]interface{})
	for nodeID, simulator := range mgr.simulators {
		// Fix: Use correct method name GetStatus() 
		status[nodeID] = simulator.GetStatus()
	}
	
	return map[string]interface{}{
		"isRunning":        mgr.isRunning,
		"simulatorCount":   len(mgr.simulators),
		"simulators":       status,
	}
}

// ConnectSimulator connects a specific simulator to RIC
func (mgr *E2SimulatorManager) ConnectSimulator(nodeID string) error {
	mgr.mu.RLock()
	simulator, exists := mgr.simulators[nodeID]
	mgr.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("simulator with node ID %s not found", nodeID)
	}
	
	return simulator.Connect()
}

// DisconnectSimulator disconnects a specific simulator from RIC
func (mgr *E2SimulatorManager) DisconnectSimulator(nodeID string) error {
	mgr.mu.RLock()
	simulator, exists := mgr.simulators[nodeID]
	mgr.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("simulator with node ID %s not found", nodeID)
	}
	
	// Fix: Use correct method name disconnect() (lowercase)
	return simulator.disconnect()
}

// ConnectAllSimulators connects all simulators to RIC
func (mgr *E2SimulatorManager) ConnectAllSimulators() error {
	mgr.mu.RLock()
	simulators := make([]*E2NodeSimulator, 0, len(mgr.simulators))
	for _, simulator := range mgr.simulators {
		simulators = append(simulators, simulator)
	}
	mgr.mu.RUnlock()
	
	var lastErr error
	successCount := 0
	
	for _, simulator := range simulators {
		if err := simulator.Connect(); err != nil {
			lastErr = err
			log.Printf("Failed to connect simulator %s: %v", simulator.nodeID, err)
		} else {
			successCount++
		}
		
		// Brief delay between connections
		time.Sleep(500 * time.Millisecond)
	}
	
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to connect any simulators: %w", lastErr)
	}
	
	log.Printf("Connected %d out of %d simulators", successCount, len(simulators))
	return nil
}

// DisconnectAllSimulators disconnects all simulators from RIC
func (mgr *E2SimulatorManager) DisconnectAllSimulators() error {
	mgr.mu.RLock()
	simulators := make([]*E2NodeSimulator, 0, len(mgr.simulators))
	for _, simulator := range mgr.simulators {
		simulators = append(simulators, simulator)
	}
	mgr.mu.RUnlock()
	
	for _, simulator := range simulators {
		// Fix: Use correct method name disconnect() (lowercase)  
		simulator.disconnect()
	}
	
	log.Printf("Disconnected all simulators")
	return nil
}

// CreateDefaultSimulators creates a set of default simulators for testing
func (mgr *E2SimulatorManager) CreateDefaultSimulators(ricAddress string, ricPort uint32) error {
	defaultConfigs := []E2NodeSimulatorConfig{
		{
			NodeID:         "gnb-001",
			NodeType:       string(E2NodeTypeGNB),
			PlmnID:         []byte("001001"),
			RICAddress:     ricAddress,
			RICPort:        ricPort,
			LocalAddress:   "127.0.0.1",
			LocalPort:      36421,
			IndicationRate: 5 * time.Second,
			AutoConnect:    true,
			RANFunctions: []RANFunction{
				{
					ID:          1,
					OID:         "1.3.6.1.4.1.53148.1.2.2.2",
					Description: "E2SM-KPM",
					Revision:    1,
				},
				{
					ID:          2,
					OID:         "1.3.6.1.4.1.53148.1.2.2.3",
					Description: "E2SM-RC",
					Revision:    1,
				},
			},
		},
		{
			NodeID:         "gnb-002",
			NodeType:       string(E2NodeTypeGNB),
			PlmnID:         []byte("001001"),
			RICAddress:     ricAddress,
			RICPort:        ricPort,
			LocalAddress:   "127.0.0.1",
			LocalPort:      36422,
			IndicationRate: 3 * time.Second,
			AutoConnect:    true,
			RANFunctions: []RANFunction{
				{
					ID:          1,
					OID:         "1.3.6.1.4.1.53148.1.2.2.2",
					Description: "E2SM-KPM",
					Revision:    1,
				},
				{
					ID:          3,
					OID:         "1.3.6.1.4.1.53148.1.2.2.4",
					Description: "E2SM-NI",
					Revision:    1,
				},
			},
		},
		{
			NodeID:         "o-cu-001",
			NodeType:       string(E2NodeTypeOCU),
			PlmnID:         []byte("001001"),
			RICAddress:     ricAddress,
			RICPort:        ricPort,
			LocalAddress:   "127.0.0.1",
			LocalPort:      36423,
			IndicationRate: 10 * time.Second,
			AutoConnect:    false, // Manual connection for testing
			RANFunctions: []RANFunction{
				{
					ID:          1,
					OID:         "1.3.6.1.4.1.53148.1.2.2.2",
					Description: "E2SM-KPM",
					Revision:    1,
				},
				{
					ID:          2,
					OID:         "1.3.6.1.4.1.53148.1.2.2.3",
					Description: "E2SM-RC",
					Revision:    1,
				},
				{
					ID:          3,
					OID:         "1.3.6.1.4.1.53148.1.2.2.4",
					Description: "E2SM-NI",
					Revision:    1,
				},
			},
		},
	}
	
	for _, config := range defaultConfigs {
		if err := mgr.CreateSimulator(&config); err != nil {
			log.Printf("Failed to create default simulator %s: %v", config.NodeID, err)
		}
	}
	
	log.Printf("Created %d default simulators", len(defaultConfigs))
	return nil
}

// GetSimulatorSubscriptions returns subscriptions for a specific simulator
func (mgr *E2SimulatorManager) GetSimulatorSubscriptions(nodeID string) (map[string]*SimulatedSubscription, error) {
	simulator, err := mgr.GetSimulator(nodeID)
	if err != nil {
		return nil, err
	}
	
	return simulator.GetSubscriptions(), nil
}

// GetAllSubscriptions returns subscriptions for all simulators
func (mgr *E2SimulatorManager) GetAllSubscriptions() map[string]map[string]*SimulatedSubscription {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	
	allSubscriptions := make(map[string]map[string]*SimulatedSubscription)
	for nodeID, simulator := range mgr.simulators {
		allSubscriptions[nodeID] = simulator.GetSubscriptions()
	}
	
	return allSubscriptions
}

// GetStats returns statistics for the simulator manager
func (mgr *E2SimulatorManager) GetStats() map[string]interface{} {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	
	totalSubscriptions := 0
	connectedSimulators := 0
	runningSimulators := 0
	
	for _, simulator := range mgr.simulators {
		status := simulator.GetStatus()
		if connected, ok := status["isConnected"].(bool); ok && connected {
			connectedSimulators++
		}
		if running, ok := status["isRunning"].(bool); ok && running {
			runningSimulators++
		}
		if subscriptions, ok := status["subscriptions"].(int); ok {
			totalSubscriptions += subscriptions
		}
	}
	
	return map[string]interface{}{
		"totalSimulators":     len(mgr.simulators),
		"runningSimulators":   runningSimulators,
		"connectedSimulators": connectedSimulators,
		"totalSubscriptions":  totalSubscriptions,
		"isManagerRunning":    mgr.isRunning,
	}
}