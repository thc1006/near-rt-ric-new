package rmr

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Context represents an RMR messaging context
type Context struct {
	conn     net.Conn
	endpoint string
	active   bool
	mu       sync.RWMutex
}

// MessageType defines standard RMR message types for O-RAN RIC components
type MessageType int

const (
	E2_SETUP_REQUEST MessageType = iota
	E2_SETUP_RESPONSE
	E2_NODE_CONFIG_UPDATE
	SUBSCRIPTION_REQUEST
	SUBSCRIPTION_RESPONSE
	INDICATION_MESSAGE
	RIC_CONTROL_REQUEST
	RIC_CONTROL_RESPONSE
)

// RoutingTable manages RMR message routing configurations
type RoutingTable struct {
	mu              sync.RWMutex
	routes          map[MessageType][]string
	defaultRoutes   []string
	connectionPools map[string]*Context
}

// NewContext creates a new RMR context for the given endpoint
func NewContext(endpoint string) (*Context, error) {
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", endpoint, err)
	}

	return &Context{
		conn:     conn,
		endpoint: endpoint,
		active:   true,
	}, nil
}

// Send transmits data through the RMR context
func (c *Context) Send(data []byte, length int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return fmt.Errorf("context is closed")
	}

	_, err := c.conn.Write(data[:length])
	if err != nil {
		c.active = false
		return fmt.Errorf("failed to send data: %v", err)
	}

	return nil
}

// Close closes the RMR context
func (c *Context) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active && c.conn != nil {
		c.conn.Close()
		c.active = false
	}
}

// IsActive checks if the context is still active
func (c *Context) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.active
}

// NewRoutingTable creates a new RMR routing table
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		routes:          make(map[MessageType][]string),
		connectionPools: make(map[string]*Context),
	}
}

// AddRoute adds a routing entry for a specific message type
func (rt *RoutingTable) AddRoute(msgType MessageType, endpoints ...string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if len(endpoints) == 0 {
		return fmt.Errorf("must provide at least one endpoint")
	}

	rt.routes[msgType] = endpoints
	return nil
}

// GetRoutes retrieves routes for a specific message type
func (rt *RoutingTable) GetRoutes(msgType MessageType) []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return rt.routes[msgType]
}

// SetDefaultRoutes sets default fallback routes
func (rt *RoutingTable) SetDefaultRoutes(endpoints ...string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.defaultRoutes = endpoints
}

// RouteMessage routes a message based on its type and available endpoints
func (rt *RoutingTable) RouteMessage(msgType MessageType, msg []byte) error {
	rt.mu.RLock()
	routes := rt.routes[msgType]
	if len(routes) == 0 {
		routes = rt.defaultRoutes
	}
	rt.mu.RUnlock()

	if len(routes) == 0 {
		return fmt.Errorf("no routes available for message type %v", msgType)
	}

	// Implement load balancing or round-robin routing
	for _, endpoint := range routes {
		ctx, err := rt.getOrCreateConnection(endpoint)
		if err != nil {
			continue
		}

		// Send message via RMR
		if err := ctx.Send(msg, len(msg)); err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to route message for type %v", msgType)
}

// getOrCreateConnection manages connection pools for RMR endpoints
func (rt *RoutingTable) getOrCreateConnection(endpoint string) (*Context, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if ctx, exists := rt.connectionPools[endpoint]; exists && ctx.IsActive() {
		return ctx, nil
	}

	// Create new RMR context
	ctx, err := NewContext(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create RMR context for %s: %v", endpoint, err)
	}

	rt.connectionPools[endpoint] = ctx
	return ctx, nil
}

// RemoveRoute removes a routing entry
func (rt *RoutingTable) RemoveRoute(msgType MessageType) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.routes, msgType)
}

// GetActiveConnections returns the number of active connections
func (rt *RoutingTable) GetActiveConnections() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	active := 0
	for _, ctx := range rt.connectionPools {
		if ctx.IsActive() {
			active++
		}
	}
	return active
}

// Close cleanly shuts down all RMR connections
func (rt *RoutingTable) Close() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for _, ctx := range rt.connectionPools {
		ctx.Close()
	}
}

// HealthCheck verifies connectivity to all configured endpoints
func (rt *RoutingTable) HealthCheck() map[string]bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	status := make(map[string]bool)
	for endpoint, ctx := range rt.connectionPools {
		status[endpoint] = ctx.IsActive()
	}

	return status
}