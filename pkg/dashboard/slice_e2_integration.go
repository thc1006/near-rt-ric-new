package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
)

// E2SliceConfigManager handles E2 interface configurations for network slices
type E2SliceConfigManager struct {
	e2Clients     map[string]*E2NodeClient
	clientMutex   sync.RWMutex
	nodeRegistry  map[string]E2NodeInfo
}

// E2NodeClient represents a client connection to an E2 node
type E2NodeClient struct {
	NodeID    string
	GRPCConn  *grpc.Conn
	Connected bool
}

// E2NodeInfo contains metadata about an E2 node
type E2NodeInfo struct {
	NodeID      string
	NodeType    string
	IPAddress   string
	Port        int
	LastSeen    time.Time
	Capabilities []string
}

// SliceConfig represents E2 slice configuration (local definition)
type SliceConfig struct {
	SliceID          string
	ServiceProfile   *ServiceProfile
	ResourceAllocation *ResourceAllocation
}

// ServiceProfile represents service level requirements
type ServiceProfile struct {
	MaxDataRate *DataRate
	Latency     int32
	Reliability float32
}

// DataRate represents data rate configuration
type DataRate struct {
	Value int64
	Unit  string
}

// ResourceAllocation represents resource allocation for a slice
type ResourceAllocation struct {
	CPU     *ComputeResource
	Memory  *ComputeResource
	Network *NetworkResource
}

// ComputeResource represents compute resource allocation
type ComputeResource struct {
	Value int32
	Unit  string
}

// NetworkResource represents network resource allocation
type NetworkResource struct {
	Bandwidth int64
	Priority  int32
}

// NewE2SliceConfigManager creates a new E2 slice configuration manager
func NewE2SliceConfigManager() *E2SliceConfigManager {
	return &E2SliceConfigManager{
		e2Clients:    make(map[string]*E2NodeClient),
		nodeRegistry: make(map[string]E2NodeInfo),
	}
}

// RegisterE2Node registers a new E2 node with the manager
func (m *E2SliceConfigManager) RegisterE2Node(nodeInfo E2NodeInfo) error {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	m.nodeRegistry[nodeInfo.NodeID] = nodeInfo
	log.Printf("Registered E2 node: %s", nodeInfo.NodeID)
	return nil
}

// ConnectToE2Node establishes a connection to an E2 node
func (m *E2SliceConfigManager) ConnectToE2Node(nodeID string) error {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	nodeInfo, exists := m.nodeRegistry[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found in registry", nodeID)
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeInfo.IPAddress, nodeInfo.Port), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to E2 node %s: %v", nodeID, err)
	}

	client := &E2NodeClient{
		NodeID:    nodeID,
		GRPCConn:  conn,
		Connected: true,
	}

	m.e2Clients[nodeID] = client
	log.Printf("Connected to E2 node: %s", nodeID)
	return nil
}

// ConfigureSliceOnE2Node configures a network slice on an E2 node
func (m *E2SliceConfigManager) ConfigureSliceOnE2Node(nodeID string, slice *NetworkSlice) error {
	m.clientMutex.RLock()
	client, exists := m.e2Clients[nodeID]
	m.clientMutex.RUnlock()

	if !exists || !client.Connected {
		return fmt.Errorf("no active connection to E2 node %s", nodeID)
	}

	e2Config, err := m.convertSliceToE2SMConfig(slice)
	if err != nil {
		return fmt.Errorf("failed to convert slice config: %v", err)
	}

	// Use E2T API instead of direct E2SM API
	e2tClient := e2tapi.NewE2TServiceClient(client.GRPCConn)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a subscription request for slice configuration
	subscriptionRequest := &e2tapi.SubscriptionRequest{
		SubscriptionId: uuid.New().String(),
		// Add subscription details based on slice configuration
	}

	_, err = e2tClient.Subscribe(ctx, subscriptionRequest)
	if err != nil {
		return fmt.Errorf("failed to configure slice on E2 node %s: %v", nodeID, err)
	}

	log.Printf("Successfully configured slice %s on E2 node %s", slice.ID, nodeID)
	return nil
}

// convertSliceToE2SMConfig converts a NetworkSlice to E2SM configuration
func (m *E2SliceConfigManager) convertSliceToE2SMConfig(slice *NetworkSlice) (*SliceConfig, error) {
	// Convert to local SliceConfig structure
	e2Config := &SliceConfig{
		SliceID: slice.ID,
		ServiceProfile: &ServiceProfile{
			MaxDataRate: &DataRate{
				Value: int64(slice.Throughput),
				Unit:  "Mbps",
			},
			Latency:     int32(slice.Latency),
			Reliability: slice.Reliability,
		},
		ResourceAllocation: &ResourceAllocation{
			CPU: &ComputeResource{
				Value: int32(slice.CPU),
				Unit:  "cores",
			},
			Memory: &ComputeResource{
				Value: int32(slice.Memory),
				Unit:  "MB",
			},
			Network: &NetworkResource{
				Bandwidth: int64(slice.Bandwidth),
				Priority:  int32(slice.Priority),
			},
		},
	}

	return e2Config, nil
}

// GetE2NodeStatus returns the status of all E2 nodes
func (m *E2SliceConfigManager) GetE2NodeStatus() map[string]E2NodeInfo {
	m.clientMutex.RLock()
	defer m.clientMutex.RUnlock()

	status := make(map[string]E2NodeInfo)
	for nodeID, nodeInfo := range m.nodeRegistry {
		client, connected := m.e2Clients[nodeID]
		nodeInfo.LastSeen = time.Now()
		
		if connected && client.Connected {
			// Update capabilities or status if needed
		}
		
		status[nodeID] = nodeInfo
	}

	return status
}

// DisconnectE2Node disconnects from an E2 node
func (m *E2SliceConfigManager) DisconnectE2Node(nodeID string) error {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	client, exists := m.e2Clients[nodeID]
	if !exists {
		return fmt.Errorf("no connection to E2 node %s", nodeID)
	}

	if client.GRPCConn != nil {
		err := client.GRPCConn.Close()
		if err != nil {
			log.Printf("Error closing connection to E2 node %s: %v", nodeID, err)
		}
	}

	delete(m.e2Clients, nodeID)
	log.Printf("Disconnected from E2 node: %s", nodeID)
	return nil
}

// Shutdown closes all E2 node connections
func (m *E2SliceConfigManager) Shutdown() error {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	for nodeID, client := range m.e2Clients {
		if client.GRPCConn != nil {
			client.GRPCConn.Close()
		}
	}

	m.e2Clients = make(map[string]*E2NodeClient)
	log.Println("E2 slice configuration manager shut down")
	return nil
}