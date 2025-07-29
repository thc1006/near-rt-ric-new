/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWebSocketBroadcastMessageError(t *testing.T) {
	hub := NewWebSocketHub()

	// Test broadcasting a message that cannot be marshaled
	// Create a channel which cannot be marshaled to JSON
	invalidData := make(chan int)

	// This should not panic, but should log an error
	hub.BroadcastMessage("test_error", invalidData)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// The broadcast channel should be empty since marshaling failed
	select {
	case <-hub.broadcast:
		t.Error("Expected no message to be broadcast due to marshal error")
	default:
		// Expected - no message should be broadcast
	}
}

func TestWebSocketClientSendChannelFull(t *testing.T) {
	hub := NewWebSocketHub()

	// Create a client with a very small send buffer
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 1), // Very small buffer
	}

	// Fill the send buffer
	client.send <- []byte("test message")

	// Now try to send a ping response which should fail due to full buffer
	pingMsg := map[string]interface{}{
		"type": "ping",
	}

	msgBytes, _ := json.Marshal(pingMsg)

	// This should not panic even if the send channel is full
	client.handleClientMessage(msgBytes)

	// Verify the send channel is still full (message wasn't sent)
	select {
	case <-client.send:
		// Drain the original message
	default:
		t.Error("Expected send channel to have the original message")
	}
}

func TestWebSocketHubClientCleanupOnBroadcast(t *testing.T) {
	hub := NewWebSocketHub()

	// Start the hub
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Create a client with a send channel
	client := &WebSocketClient{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 1),
	}

	// Register the client
	hub.register <- client

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Drain welcome message
	<-client.send

	// Manually add client to hub and close its send channel to simulate disconnection
	hub.clients[client] = true
	close(client.send)

	// Try to broadcast a message
	hub.BroadcastMessage("test_cleanup", map[string]string{"test": "message"})

	// Give time for processing
	time.Sleep(200 * time.Millisecond)

	// The client should be automatically cleaned up
	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients after cleanup, got %d", len(hub.clients))
	}
}
