/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// SCTPConnectionState represents the state of an SCTP connection
type SCTPConnectionState string

const (
	SCTPStateDisconnected SCTPConnectionState = "DISCONNECTED"
	SCTPStateConnecting   SCTPConnectionState = "CONNECTING"
	SCTPStateConnected    SCTPConnectionState = "CONNECTED"
	SCTPStateAssociated   SCTPConnectionState = "ASSOCIATED"
	SCTPStateShuttingDown SCTPConnectionState = "SHUTTING_DOWN"
	SCTPStateError        SCTPConnectionState = "ERROR"
)

// SCTPAssociation represents an SCTP association with an E2 node
type SCTPAssociation struct {
	ID              string              `json:"id"`
	LocalAddress    string              `json:"localAddress"`
	LocalPort       uint16              `json:"localPort"`
	RemoteAddress   string              `json:"remoteAddress"`
	RemotePort      uint16              `json:"remotePort"`
	State           SCTPConnectionState `json:"state"`
	Streams         uint16              `json:"streams"`
	MaxInStreams    uint16              `json:"maxInStreams"`
	MaxOutStreams   uint16              `json:"maxOutStreams"`
	E2NodeID        string              `json:"e2NodeId"`
	CreatedAt       time.Time           `json:"createdAt"`
	LastActivity    time.Time           `json:"lastActivity"`
	BytesSent       uint64              `json:"bytesSent"`
	BytesReceived   uint64              `json:"bytesReceived"`
	MessagesSent    uint64              `json:"messagesSent"`
	MessagesReceived uint64             `json:"messagesReceived"`
	ErrorCount      uint64              `json:"errorCount"`
	LastError       string              `json:"lastError,omitempty"`
}

// SCTPConnectionManager manages SCTP connections for E2 nodes
type SCTPConnectionManager struct {
	mu           sync.RWMutex
	associations map[string]*SCTPAssociation
	listeners    map[string]*SCTPListener
	config       *SCTPConfig
	eventHandler SCTPEventHandler
}

// SCTPConfig holds configuration for SCTP connections
type SCTPConfig struct {
	ListenAddress     string        `json:"listenAddress"`
	ListenPort        uint16        `json:"listenPort"`
	MaxAssociations   uint32        `json:"maxAssociations"`
	HeartbeatInterval time.Duration `json:"heartbeatInterval"`
	ConnectTimeout    time.Duration `json:"connectTimeout"`
	MaxRetries        uint32        `json:"maxRetries"`
	MaxInStreams      uint16        `json:"maxInStreams"`
	MaxOutStreams     uint16        `json:"maxOutStreams"`
	BufferSize        uint32        `json:"bufferSize"`
}

// SCTPListener represents an SCTP listener
type SCTPListener struct {
	Address   string
	Port      uint16
	conn      net.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	manager   *SCTPConnectionManager
}

// SCTPEventHandler defines the interface for handling SCTP events
type SCTPEventHandler interface {
	OnAssociationEstablished(assoc *SCTPAssociation)
	OnAssociationClosed(assoc *SCTPAssociation)
	OnMessageReceived(assoc *SCTPAssociation, data []byte, stream uint16)
	OnError(assoc *SCTPAssociation, err error)
}

// DefaultSCTPEventHandler provides a default implementation of SCTPEventHandler
type DefaultSCTPEventHandler struct{}

func (h *DefaultSCTPEventHandler) OnAssociationEstablished(assoc *SCTPAssociation) {
	log.Printf("SCTP association established: %s (%s:%d -> %s:%d)", 
		assoc.ID, assoc.LocalAddress, assoc.LocalPort, assoc.RemoteAddress, assoc.RemotePort)
}

func (h *DefaultSCTPEventHandler) OnAssociationClosed(assoc *SCTPAssociation) {
	log.Printf("SCTP association closed: %s", assoc.ID)
}

func (h *DefaultSCTPEventHandler) OnMessageReceived(assoc *SCTPAssociation, data []byte, stream uint16) {
	log.Printf("SCTP message received on association %s, stream %d, length %d", 
		assoc.ID, stream, len(data))
}

func (h *DefaultSCTPEventHandler) OnError(assoc *SCTPAssociation, err error) {
	log.Printf("SCTP error on association %s: %v", assoc.ID, err)
}

// NewSCTPConnectionManager creates a new SCTP connection manager
func NewSCTPConnectionManager(config *SCTPConfig, eventHandler SCTPEventHandler) *SCTPConnectionManager {
	if config == nil {
		config = &SCTPConfig{
			ListenAddress:     "0.0.0.0",
			ListenPort:        36422, // Standard E2 port
			MaxAssociations:   1000,
			HeartbeatInterval: 30 * time.Second,
			ConnectTimeout:    10 * time.Second,
			MaxRetries:        3,
			MaxInStreams:      10,
			MaxOutStreams:     10,
			BufferSize:        65536,
		}
	}

	if eventHandler == nil {
		eventHandler = &DefaultSCTPEventHandler{}
	}

	return &SCTPConnectionManager{
		associations: make(map[string]*SCTPAssociation),
		listeners:    make(map[string]*SCTPListener),
		config:       config,
		eventHandler: eventHandler,
	}
}

// Start starts the SCTP connection manager
func (m *SCTPConnectionManager) Start(ctx context.Context) error {
	log.Printf("Starting SCTP connection manager on %s:%d", m.config.ListenAddress, m.config.ListenPort)
	
	// Start the main listener
	listener, err := m.createListener(ctx, m.config.ListenAddress, m.config.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to create SCTP listener: %w", err)
	}

	listenerKey := fmt.Sprintf("%s:%d", m.config.ListenAddress, m.config.ListenPort)
	m.mu.Lock()
	m.listeners[listenerKey] = listener
	m.mu.Unlock()

	// Start accepting connections
	go m.acceptConnections(ctx, listener)

	// Start periodic maintenance
	go m.maintenanceLoop(ctx)

	return nil
}

// Stop stops the SCTP connection manager
func (m *SCTPConnectionManager) Stop() error {
	log.Println("Stopping SCTP connection manager")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all listeners
	for _, listener := range m.listeners {
		listener.cancel()
		if listener.conn != nil {
			listener.conn.Close()
		}
	}

	// Close all associations
	for _, assoc := range m.associations {
		m.closeAssociation(assoc)
	}

	m.listeners = make(map[string]*SCTPListener)
	m.associations = make(map[string]*SCTPAssociation)

	return nil
}

// createListener creates a new SCTP listener
func (m *SCTPConnectionManager) createListener(ctx context.Context, address string, port uint16) (*SCTPListener, error) {
	// For now, use TCP as a fallback since Go doesn't have native SCTP support
	// In a production environment, this would use a proper SCTP library
	addr := fmt.Sprintf("%s:%d", address, port)
	
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %s: %w", addr, err)
	}

	conn, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	listenerCtx, cancel := context.WithCancel(ctx)
	
	return &SCTPListener{
		Address: address,
		Port:    port,
		conn:    conn,
		ctx:     listenerCtx,
		cancel:  cancel,
		manager: m,
	}, nil
}

// acceptConnections accepts incoming SCTP connections
func (m *SCTPConnectionManager) acceptConnections(ctx context.Context, listener *SCTPListener) {
	defer listener.conn.Close()

	tcpListener := listener.conn.(*net.TCPListener)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-listener.ctx.Done():
			return
		default:
			// Set accept timeout
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			
			conn, err := tcpListener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is expected, continue accepting
				}
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			// Handle the new connection
			go m.handleConnection(ctx, conn)
		}
	}
}

// handleConnection handles a new SCTP connection
func (m *SCTPConnectionManager) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
	localAddr := conn.LocalAddr().(*net.TCPAddr)

	// Create association
	assoc := &SCTPAssociation{
		ID:               fmt.Sprintf("%s:%d-%s:%d", localAddr.IP, localAddr.Port, remoteAddr.IP, remoteAddr.Port),
		LocalAddress:     localAddr.IP.String(),
		LocalPort:        uint16(localAddr.Port),
		RemoteAddress:    remoteAddr.IP.String(),
		RemotePort:       uint16(remoteAddr.Port),
		State:            SCTPStateConnecting,
		MaxInStreams:     m.config.MaxInStreams,
		MaxOutStreams:    m.config.MaxOutStreams,
		CreatedAt:        time.Now(),
		LastActivity:     time.Now(),
	}

	// Register association
	m.mu.Lock()
	if uint32(len(m.associations)) >= m.config.MaxAssociations {
		m.mu.Unlock()
		log.Printf("Maximum associations reached, rejecting connection from %s", remoteAddr)
		return
	}
	m.associations[assoc.ID] = assoc
	m.mu.Unlock()

	// Update state to connected
	assoc.State = SCTPStateConnected
	m.eventHandler.OnAssociationEstablished(assoc)

	// Start message processing
	m.processMessages(ctx, conn, assoc)
}

// processMessages processes messages on an SCTP association
func (m *SCTPConnectionManager) processMessages(ctx context.Context, conn net.Conn, assoc *SCTPAssociation) {
	defer func() {
		m.mu.Lock()
		delete(m.associations, assoc.ID)
		m.mu.Unlock()
		
		assoc.State = SCTPStateDisconnected
		m.eventHandler.OnAssociationClosed(assoc)
	}()

	buffer := make([]byte, m.config.BufferSize)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read timeout
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			
			n, err := conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is expected, continue reading
				}
				log.Printf("Error reading from association %s: %v", assoc.ID, err)
				assoc.ErrorCount++
				assoc.LastError = err.Error()
				m.eventHandler.OnError(assoc, err)
				return
			}

			if n > 0 {
				// Update statistics
				assoc.BytesReceived += uint64(n)
				assoc.MessagesReceived++
				assoc.LastActivity = time.Now()

				// Process the message
				data := make([]byte, n)
				copy(data, buffer[:n])
				
				// For now, assume stream 0. In real SCTP, this would be extracted from the message
				m.eventHandler.OnMessageReceived(assoc, data, 0)
			}
		}
	}
}

// SendMessage sends a message on an SCTP association
func (m *SCTPConnectionManager) SendMessage(associationID string, data []byte, stream uint16) error {
	m.mu.RLock()
	assoc, exists := m.associations[associationID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("association %s not found", associationID)
	}

	if assoc.State != SCTPStateConnected && assoc.State != SCTPStateAssociated {
		return fmt.Errorf("association %s is not connected (state: %s)", associationID, assoc.State)
	}

	// In a real implementation, this would send via the actual SCTP connection
	// For now, we'll simulate the send operation
	assoc.BytesSent += uint64(len(data))
	assoc.MessagesSent++
	assoc.LastActivity = time.Now()

	log.Printf("Sent message on association %s, stream %d, length %d", associationID, stream, len(data))
	return nil
}

// GetAssociation retrieves an SCTP association by ID
func (m *SCTPConnectionManager) GetAssociation(associationID string) (*SCTPAssociation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assoc, exists := m.associations[associationID]
	if !exists {
		return nil, fmt.Errorf("association %s not found", associationID)
	}

	// Return a copy to avoid race conditions
	assocCopy := *assoc
	return &assocCopy, nil
}

// GetAssociations retrieves all SCTP associations
func (m *SCTPConnectionManager) GetAssociations() []*SCTPAssociation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	associations := make([]*SCTPAssociation, 0, len(m.associations))
	for _, assoc := range m.associations {
		// Return copies to avoid race conditions
		assocCopy := *assoc
		associations = append(associations, &assocCopy)
	}

	return associations
}

// GetAssociationsByE2Node retrieves SCTP associations for a specific E2 node
func (m *SCTPConnectionManager) GetAssociationsByE2Node(e2NodeID string) []*SCTPAssociation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var associations []*SCTPAssociation
	for _, assoc := range m.associations {
		if assoc.E2NodeID == e2NodeID {
			// Return a copy to avoid race conditions
			assocCopy := *assoc
			associations = append(associations, &assocCopy)
		}
	}

	return associations
}

// CloseAssociation closes an SCTP association
func (m *SCTPConnectionManager) CloseAssociation(associationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	assoc, exists := m.associations[associationID]
	if !exists {
		return fmt.Errorf("association %s not found", associationID)
	}

	return m.closeAssociation(assoc)
}

// closeAssociation closes an SCTP association (internal method)
func (m *SCTPConnectionManager) closeAssociation(assoc *SCTPAssociation) error {
	assoc.State = SCTPStateShuttingDown
	
	// In a real implementation, this would properly close the SCTP association
	log.Printf("Closing SCTP association %s", assoc.ID)
	
	delete(m.associations, assoc.ID)
	assoc.State = SCTPStateDisconnected
	
	m.eventHandler.OnAssociationClosed(assoc)
	return nil
}

// maintenanceLoop performs periodic maintenance tasks
func (m *SCTPConnectionManager) maintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performMaintenance()
		}
	}
}

// performMaintenance performs maintenance tasks like heartbeat and cleanup
func (m *SCTPConnectionManager) performMaintenance() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	staleThreshold := now.Add(-2 * m.config.HeartbeatInterval)

	// Check for stale associations
	for id, assoc := range m.associations {
		if assoc.LastActivity.Before(staleThreshold) {
			log.Printf("Association %s appears stale, last activity: %v", id, assoc.LastActivity)
			// In a real implementation, this would send a heartbeat or close the association
		}
	}
}

// GetStats returns statistics about SCTP connections
func (m *SCTPConnectionManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"totalAssociations":    len(m.associations),
		"activeListeners":      len(m.listeners),
		"maxAssociations":      m.config.MaxAssociations,
		"associationsByState":  make(map[string]int),
		"totalBytesSent":       uint64(0),
		"totalBytesReceived":   uint64(0),
		"totalMessagesSent":    uint64(0),
		"totalMessagesReceived": uint64(0),
		"totalErrors":          uint64(0),
	}

	stateCount := make(map[string]int)
	var totalBytesSent, totalBytesReceived, totalMessagesSent, totalMessagesReceived, totalErrors uint64

	for _, assoc := range m.associations {
		stateCount[string(assoc.State)]++
		totalBytesSent += assoc.BytesSent
		totalBytesReceived += assoc.BytesReceived
		totalMessagesSent += assoc.MessagesSent
		totalMessagesReceived += assoc.MessagesReceived
		totalErrors += assoc.ErrorCount
	}

	stats["associationsByState"] = stateCount
	stats["totalBytesSent"] = totalBytesSent
	stats["totalBytesReceived"] = totalBytesReceived
	stats["totalMessagesSent"] = totalMessagesSent
	stats["totalMessagesReceived"] = totalMessagesReceived
	stats["totalErrors"] = totalErrors

	return stats
}