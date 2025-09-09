/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWebSocketBroadcastMessageSuccess(t *testing.T) {
	hub := NewWebSocketHub()

	// Test broadcasting a valid message
	testData := map[string]string{"test": "message"}

	// Start the hub to consume messages
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(50 * time.Millisecond)

	// This should not panic and should work correctly
	hub.BroadcastMessage("test_event", testData)

	// Give time for message processing
	time.Sleep(50 * time.Millisecond)

	// The test passes if BroadcastMessage doesn't panic
	// We can't easily verify the message content without a client connected
}

func TestWebSocketHubStopWithoutStart(t *testing.T) {
	hub := NewWebSocketHub()

	// Test stopping a hub that was never started (should not panic)
	hub.Stop()

	// Verify the stop channel is closed
	select {
	case <-hub.stop:
		// Expected - channel should be closed
	default:
		t.Error("Expected stop channel to be closed")
	}
}

func TestClientManagerCloseWithNilConnections(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	cm := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
		// Connections are nil by default
	}

	// Test Close with nil connections (should not panic)
	cm.Close()
}

func TestNewServerSuccess(t *testing.T) {
	// Test NewServer with valid config
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if server == nil {
		t.Error("Expected server to be created")
	}

	// Clean up
	if server != nil {
		server.clients.Close()
	}
}

func TestDiscoveryServiceGetComponentsEmpty(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	clients := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
	}

	ds := NewDiscoveryService(clients)

	// Test GetComponents with empty components map
	components := ds.GetComponents()
	if len(components) != 0 {
		t.Errorf("Expected 0 components, got %d", len(components))
	}

	// Test GetComponent with non-existent ID
	component, exists := ds.GetComponent("nonexistent")
	if exists {
		t.Error("Expected component to not exist")
	}
	if component != nil {
		t.Error("Expected nil component for non-existent ID")
	}

	// Test GetComponentStatus with empty components
	status := ds.GetComponentStatus()
	if len(status) != 0 {
		t.Errorf("Expected 0 component statuses, got %d", len(status))
	}
}
func TestWebSocketHubRegistrationUnregistration(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(50 * time.Millisecond)

	// Create a client
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 1),
	}

	// Register the client
	hub.register <- client

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	// Drain welcome message
	<-client.send

	// Check that client was registered
	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(hub.clients))
	}

	// Unregister the client
	hub.unregister <- client

	// Give time for unregistration
	time.Sleep(50 * time.Millisecond)

	// Check that client was unregistered
	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(hub.clients))
	}
}

func TestWebSocketHubBroadcastToClients(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(50 * time.Millisecond)

	// Create a client
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 2), // Larger buffer for welcome + broadcast
	}

	// Register the client
	hub.register <- client

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	// Drain welcome message
	<-client.send

	// Broadcast a message
	hub.BroadcastMessage("test_broadcast", map[string]string{"message": "hello"})

	// Give time for message processing
	time.Sleep(50 * time.Millisecond)

	// Check that client received the broadcast message
	select {
	case msg := <-client.send:
		var message map[string]interface{}
		if err := json.Unmarshal(msg, &message); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}

		if message["type"] != "test_broadcast" {
			t.Errorf("Expected message type 'test_broadcast', got %v", message["type"])
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Expected client to receive broadcast message")
	}
}
