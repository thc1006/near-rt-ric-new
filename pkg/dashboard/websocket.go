/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

// WebSocketHub maintains the set of active clients and broadcasts messages to the clients
type WebSocketHub struct {
	// Registered clients
	clients map[*WebSocketClient]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *WebSocketClient

	// Unregister requests from clients
	unregister chan *WebSocketClient

	// Stop channel
	stop chan struct{}
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		clients:    make(map[*WebSocketClient]bool),
		stop:       make(chan struct{}),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	log.Info("Starting WebSocket hub")

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Infof("WebSocket client connected. Total clients: %d", len(h.clients))

			// Send welcome message
			welcome := map[string]interface{}{
				"type":      "welcome",
				"message":   "Connected to O-RAN Dashboard API",
				"timestamp": time.Now(),
			}
			if data, err := json.Marshal(welcome); err == nil {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Infof("WebSocket client disconnected. Total clients: %d", len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client send channel is full or closed, remove client
					delete(h.clients, client)
					// Close channel safely - only if it's still in our clients map
					go func(c *WebSocketClient) {
						defer func() {
							if r := recover(); r != nil {
								// Channel was already closed, ignore
							}
						}()
						close(c.send)
					}(client)
				}
			}

		case <-h.stop:
			log.Info("Stopping WebSocket hub")
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			return
		}
	}
}

// Stop stops the WebSocket hub
func (h *WebSocketHub) Stop() {
	close(h.stop)
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *WebSocketHub) BroadcastMessage(messageType string, data interface{}) {
	message := map[string]interface{}{
		"type":      messageType,
		"data":      data,
		"timestamp": time.Now(),
	}

	if jsonData, err := json.Marshal(message); err == nil {
		h.broadcast <- jsonData
	} else {
		log.Errorf("Failed to marshal WebSocket message: %v", err)
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("WebSocket error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		// Handle incoming messages from clients
		c.handleClientMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage handles messages received from WebSocket clients
func (c *WebSocketClient) handleClientMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Errorf("Failed to unmarshal client message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		log.Warn("Received message without type field")
		return
	}

	switch msgType {
	case "ping":
		// Respond to ping with pong
		response := map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now(),
		}
		if data, err := json.Marshal(response); err == nil {
			select {
			case c.send <- data:
			default:
				// Client send channel is full, close the connection
				close(c.send)
			}
		}

	case "subscribe":
		// Handle subscription requests for specific data types
		log.Infof("Client subscribed to updates: %v", msg["data"])

	default:
		log.Warnf("Unknown message type received: %s", msgType)
	}
}
