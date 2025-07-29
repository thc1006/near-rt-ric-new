/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHubBasicFunctionality(t *testing.T) {
	hub := NewWebSocketHub()

	// Test initial state
	if len(hub.clients) != 0 {
		t.Error("Expected no clients initially")
	}

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Test broadcasting a message with no clients (should not panic)
	hub.BroadcastMessage("test", map[string]string{"message": "hello"})

	// Give time for message processing
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocketHubClientRegistration(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create a mock WebSocket client
	client := &WebSocketClient{
		hub:  hub,
		conn: nil, // We'll use nil for this test
		send: make(chan []byte, 256),
	}

	// Register the client
	hub.register <- client

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Check that client was registered
	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(hub.clients))
	}

	if !hub.clients[client] {
		t.Error("Expected client to be registered")
	}

	// Unregister the client
	hub.unregister <- client

	// Give time for unregistration
	time.Sleep(100 * time.Millisecond)

	// Check that client was unregistered
	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(hub.clients))
	}
}

func TestWebSocketHubBroadcast(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create mock WebSocket clients
	client1 := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	client2 := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	// Register the clients
	hub.register <- client1
	hub.register <- client2

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Drain welcome messages
	<-client1.send // welcome message
	<-client2.send // welcome message

	// Broadcast a message
	testData := map[string]string{"test": "message"}
	hub.BroadcastMessage("test_event", testData)

	// Give time for message processing
	time.Sleep(100 * time.Millisecond)

	// Check that both clients received the message
	select {
	case msg := <-client1.send:
		var message map[string]interface{}
		if err := json.Unmarshal(msg, &message); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}

		if message["type"] != "test_event" {
			t.Errorf("Expected message type 'test_event', got %v", message["type"])
		}

		data, ok := message["data"].(map[string]interface{})
		if !ok {
			t.Error("Expected data to be an object")
		}

		if data["test"] != "message" {
			t.Errorf("Expected test message, got %v", data["test"])
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message on client1")
	}

	select {
	case msg := <-client2.send:
		var message map[string]interface{}
		if err := json.Unmarshal(msg, &message); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}

		if message["type"] != "test_event" {
			t.Errorf("Expected message type 'test_event', got %v", message["type"])
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message on client2")
	}
}

func TestWebSocketHubStop(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create and register a client
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Drain welcome message
	<-client.send

	// Verify client is registered
	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(hub.clients))
	}

	// Stop the hub
	hub.Stop()

	// Give time for shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify all clients were cleaned up
	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients after stop, got %d", len(hub.clients))
	}

	// Verify client send channel was closed
	select {
	case _, ok := <-client.send:
		if !ok {
			// Channel is closed, which is expected
		} else {
			t.Error("Expected client send channel to be closed")
		}
	default:
		// Channel might be closed but not readable yet, try again
		time.Sleep(50 * time.Millisecond)
		select {
		case _, ok := <-client.send:
			if ok {
				t.Error("Expected client send channel to be closed")
			}
		default:
			// Channel is closed and drained
		}
	}
}

func TestWebSocketClientHandleMessage(t *testing.T) {
	hub := NewWebSocketHub()
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	// Test ping message
	pingMessage := map[string]interface{}{
		"type": "ping",
	}

	msgBytes, err := json.Marshal(pingMessage)
	if err != nil {
		t.Fatal(err)
	}

	client.handleClientMessage(msgBytes)

	// Check for pong response
	select {
	case response := <-client.send:
		var pongMsg map[string]interface{}
		if err := json.Unmarshal(response, &pongMsg); err != nil {
			t.Errorf("Failed to unmarshal pong response: %v", err)
		}

		if pongMsg["type"] != "pong" {
			t.Errorf("Expected pong response, got %v", pongMsg["type"])
		}

		if _, ok := pongMsg["timestamp"]; !ok {
			t.Error("Expected timestamp in pong response")
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for pong response")
	}

	// Test subscribe message
	subscribeMessage := map[string]interface{}{
		"type": "subscribe",
		"data": "component_updates",
	}

	msgBytes, err = json.Marshal(subscribeMessage)
	if err != nil {
		t.Fatal(err)
	}

	client.handleClientMessage(msgBytes)

	// Test invalid JSON
	client.handleClientMessage([]byte("invalid json"))

	// Test message without type
	noTypeMessage := map[string]interface{}{
		"data": "test",
	}

	msgBytes, err = json.Marshal(noTypeMessage)
	if err != nil {
		t.Fatal(err)
	}

	client.handleClientMessage(msgBytes)

	// Test unknown message type
	unknownMessage := map[string]interface{}{
		"type": "unknown",
		"data": "test",
	}

	msgBytes, err = json.Marshal(unknownMessage)
	if err != nil {
		t.Fatal(err)
	}

	client.handleClientMessage(msgBytes)
}

func TestWebSocketEndToEnd(t *testing.T) {
	// Create a test server
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Give time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Read welcome message
	_, welcomeMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}

	var welcome map[string]interface{}
	if err := json.Unmarshal(welcomeMsg, &welcome); err != nil {
		t.Errorf("Failed to unmarshal welcome message: %v", err)
	}

	if welcome["type"] != "welcome" {
		t.Errorf("Expected welcome message, got %v", welcome["type"])
	}

	// Send ping message
	pingMsg := map[string]interface{}{
		"type": "ping",
	}

	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Failed to send ping message: %v", err)
	}

	// Read pong response
	_, pongMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read pong message: %v", err)
	}

	var pong map[string]interface{}
	if err := json.Unmarshal(pongMsg, &pong); err != nil {
		t.Errorf("Failed to unmarshal pong message: %v", err)
	}

	if pong["type"] != "pong" {
		t.Errorf("Expected pong message, got %v", pong["type"])
	}

	// Test broadcasting from server side
	go func() {
		time.Sleep(100 * time.Millisecond)
		server.wsHub.BroadcastMessage("test_broadcast", map[string]string{"message": "hello"})
	}()

	// Read broadcast message
	_, broadcastMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read broadcast message: %v", err)
	}

	var broadcast map[string]interface{}
	if err := json.Unmarshal(broadcastMsg, &broadcast); err != nil {
		t.Errorf("Failed to unmarshal broadcast message: %v", err)
	}

	if broadcast["type"] != "test_broadcast" {
		t.Errorf("Expected test_broadcast message, got %v", broadcast["type"])
	}
}

func TestWebSocketConnectionLimits(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create multiple clients to test concurrent connections
	numClients := 10
	clients := make([]*WebSocketClient, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = &WebSocketClient{
			hub:  hub,
			conn: nil,
			send: make(chan []byte, 256),
		}

		hub.register <- clients[i]
	}

	// Give time for all registrations
	time.Sleep(200 * time.Millisecond)

	// Drain welcome messages
	for _, client := range clients {
		<-client.send // welcome message
	}

	// Check that all clients were registered
	if len(hub.clients) != numClients {
		t.Errorf("Expected %d clients, got %d", numClients, len(hub.clients))
	}

	// Broadcast a message to all clients
	hub.BroadcastMessage("mass_test", map[string]int{"client_count": numClients})

	// Give time for message processing
	time.Sleep(200 * time.Millisecond)

	// Verify all clients received the message
	for i, client := range clients {
		select {
		case msg := <-client.send:
			var message map[string]interface{}
			if err := json.Unmarshal(msg, &message); err != nil {
				t.Errorf("Client %d: Failed to unmarshal message: %v", i, err)
			}

			if message["type"] != "mass_test" {
				t.Errorf("Client %d: Expected message type 'mass_test', got %v", i, message["type"])
			}

		case <-time.After(2 * time.Second):
			t.Errorf("Client %d: Timeout waiting for message", i)
		}
	}

	// Unregister all clients
	for _, client := range clients {
		hub.unregister <- client
	}

	// Give time for unregistration
	time.Sleep(200 * time.Millisecond)

	// Check that all clients were unregistered
	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients after unregistration, got %d", len(hub.clients))
	}
}

// Benchmark WebSocket operations
func BenchmarkWebSocketBroadcast(b *testing.B) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create some clients
	numClients := 100
	for i := 0; i < numClients; i++ {
		client := &WebSocketClient{
			hub:  hub,
			conn: nil,
			send: make(chan []byte, 256),
		}
		hub.register <- client
	}

	// Give time for registration
	time.Sleep(200 * time.Millisecond)

	testData := map[string]string{"test": "message"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastMessage("benchmark_test", testData)
	}
}

func BenchmarkWebSocketClientHandleMessage(b *testing.B) {
	hub := NewWebSocketHub()
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}

	pingMessage := map[string]interface{}{
		"type": "ping",
	}

	msgBytes, _ := json.Marshal(pingMessage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.handleClientMessage(msgBytes)
		// Drain the send channel to prevent blocking
		select {
		case <-client.send:
		default:
		}
	}
}
