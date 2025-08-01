# O-RAN SC gRPC Protocol Definitions

This directory contains the Protocol Buffer (protobuf) definitions for the O-RAN SC (Software Community) components used in the Near-RT RIC platform.

## Overview

The protobuf definitions provide standardized gRPC interfaces for communication between O-RAN SC components:

- **E2 Manager (E2M)**: Manages E2 node lifecycle, connection state, and configuration
- **Subscription Manager (SubMgr)**: Orchestrates E2 subscriptions between xApps and E2 nodes  
- **Routing Manager (RTMGR)**: Manages message routing and xApp registration

## Directory Structure

```
api/proto/
├── e2mgr/
│   ├── e2mgr.proto          # E2 Manager service definition
│   ├── e2mgr.pb.go          # Generated Go protobuf structs
│   └── e2mgr_grpc.pb.go     # Generated Go gRPC client/server
├── submgr/
│   ├── submgr.proto         # Subscription Manager service definition
│   ├── submgr.pb.go         # Generated Go protobuf structs
│   └── submgr_grpc.pb.go    # Generated Go gRPC client/server
├── rtmgr/
│   ├── rtmgr.proto          # Routing Manager service definition
│   ├── rtmgr.pb.go          # Generated Go protobuf structs
│   └── rtmgr_grpc.pb.go     # Generated Go gRPC client/server
└── README.md                # This file
```

## Service Definitions

### E2 Manager (e2mgr.proto)

The E2 Manager service provides the following operations:

**Node Management:**
- `GetNodes()` - Retrieve all E2 nodes with optional filtering
- `GetNode()` - Get specific E2 node by ID
- `UpdateNodeConfiguration()` - Update node configuration
- `DeleteNode()` - Remove E2 node

**Health and Statistics:**
- `GetNodeHealth()` - Get health status of specific node
- `GetStats()` - Retrieve E2 Manager statistics

**E2 Setup Procedures:**
- `HandleE2Setup()` - Process E2 Setup requests from nodes
- `HandleConfigurationUpdate()` - Handle configuration updates
- `HandleReset()` - Process E2 Reset procedures

### Subscription Manager (submgr.proto)

The Subscription Manager service provides:

**Subscription Lifecycle:**
- `CreateSubscription()` - Create new E2 subscription
- `GetSubscriptions()` - List subscriptions with filtering
- `GetSubscription()` - Get specific subscription details
- `UpdateSubscription()` - Modify existing subscription
- `DeleteSubscription()` - Remove subscription

**Indication Handling:**
- `GetIndications()` - Retrieve recent indications
- `StreamIndications()` - Stream live indications (server streaming)

**Monitoring:**
- `GetStats()` - Subscription statistics
- `GetSubscriptionHealth()` - Health status of subscriptions

### Routing Manager (rtmgr.proto)

The Routing Manager service provides:

**Route Management:**
- `CreateRoute()` - Create new routing rule
- `GetRoutes()` - List routes with filtering
- `GetRoute()` - Get specific route details
- `UpdateRoute()` - Modify existing route
- `DeleteRoute()` - Remove route

**Routing Table Operations:**
- `GetRoutingTable()` - Retrieve current routing table
- `UpdateRoutingTable()` - Update entire routing table

**xApp Registration:**
- `RegisterXApp()` - Register new xApp
- `UnregisterXApp()` - Remove xApp registration
- `GetXApps()` - List registered xApps

**Health and Statistics:**
- `GetStats()` - Routing statistics
- `GetHealth()` - Routing Manager health status

## Data Models

### E2 Node Model
```protobuf
message E2Node {
    string id = 1;
    GlobalE2NodeID global_e2_node_id = 2;
    string connection_status = 3;
    string ip_address = 4;
    uint32 port = 5;
    repeated ServiceModel service_models = 6;
    repeated RANFunction ran_functions = 7;
    google.protobuf.Timestamp last_update = 8;
    repeated SubscriptionInfo subscriptions = 9;
    E2SetupRequestData setup_request = 10;
}
```

### Subscription Model
```protobuf
message Subscription {
    string id = 1;
    string e2_node_id = 2;
    string xapp_id = 3;
    uint32 ran_function_id = 4;
    EventTrigger event_trigger = 5;
    repeated Action actions = 6;
    string status = 7;
    string error_message = 8;
    google.protobuf.Timestamp created_at = 9;
    google.protobuf.Timestamp updated_at = 10;
}
```

### Route Model
```protobuf
message Route {
    string id = 1;
    string source_xapp = 2;
    string target_xapp = 3;
    uint32 message_type = 4;
    string subscription_id = 5;
    RoutePolicy policy = 6;
    google.protobuf.Timestamp created_at = 7;
    google.protobuf.Timestamp updated_at = 8;
}
```

## Code Generation

The Go client and server stubs are generated using the Protocol Buffers compiler (`protoc`) with the Go plugins:

```bash
# Generate Go protobuf structs and gRPC stubs
make generate-proto

# Or run the script directly
bash scripts/generate-proto.sh
```

### Prerequisites

- Protocol Buffers compiler (`protoc`)
- Go protobuf plugin (`protoc-gen-go`)
- Go gRPC plugin (`protoc-gen-go-grpc`)

Install on Ubuntu/Debian:
```bash
sudo apt-get install protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Install on macOS:
```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Client Usage

### E2 Manager Client Example

```go
import (
    "context"
    "google.golang.org/grpc"
    "github.com/oran/near-rt-ric-new/api/proto/e2mgr"
)

// Create gRPC connection
conn, err := grpc.Dial("e2mgr:8080", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Create client
client := e2mgr.NewE2ManagerClient(conn)

// Get all nodes
resp, err := client.GetNodes(context.Background(), &e2mgr.GetNodesRequest{})
if err != nil {
    log.Fatal(err)
}

for _, node := range resp.Nodes {
    fmt.Printf("Node: %s, Status: %s\n", node.Id, node.ConnectionStatus)
}
```

### Subscription Manager Client Example

```go
import (
    "context"
    "google.golang.org/grpc"
    "github.com/oran/near-rt-ric-new/api/proto/submgr"
)

// Create client
conn, _ := grpc.Dial("submgr:8080", grpc.WithInsecure())
client := submgr.NewSubscriptionManagerClient(conn)

// Create subscription
req := &submgr.CreateSubscriptionRequest{
    E2NodeId:      "node-1",
    XappId:        "my-xapp",
    RanFunctionId: 1,
    EventTrigger: &submgr.EventTrigger{
        Type:       "periodic",
        Definition: []byte("{}"),
        PeriodMs:   uint32Ptr(1000),
    },
    Actions: []*submgr.Action{
        {
            Id:         1,
            Type:       "report",
            Definition: []byte("{}"),
        },
    },
}

resp, err := client.CreateSubscription(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created subscription: %s\n", resp.SubscriptionId)
```

## Integration with Dashboard

The dashboard API clients in `pkg/dashboard/` use these protobuf definitions to communicate with O-RAN SC components:

- `E2ManagerClient` - Uses `e2mgr.E2ManagerClient`
- `SubscriptionManagerClient` - Uses `submgr.SubscriptionManagerClient`  
- `RoutingManagerClient` - Uses `rtmgr.RoutingManagerClient`

Each client provides both gRPC and HTTP fallback implementations for maximum compatibility.

## Standards Compliance

These protobuf definitions are designed to be compatible with:

- O-RAN.WG3.E2AP-R003 (E2 Application Protocol)
- O-RAN.WG2.A1 (A1 Interface Specification)
- O-RAN SC reference implementations

## Versioning

The protobuf definitions follow semantic versioning. Breaking changes to the API will result in a major version increment.

Current versions:
- E2 Manager API: v1.0
- Subscription Manager API: v1.0  
- Routing Manager API: v1.0