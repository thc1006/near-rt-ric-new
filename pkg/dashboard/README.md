# Dashboard API Gateway

The Dashboard API Gateway is a Go-based REST API service that provides an abstraction layer between the React-based dashboard frontend and the O-RAN SC (Software Community) gRPC APIs. It enables real-time monitoring and management of O-RAN Near-RT RIC components.

## Features

### Core Functionality
- **REST API Gateway**: Abstracts O-RAN SC gRPC APIs into RESTful endpoints
- **Component Discovery**: Auto-discovers deployed O-RAN SC components
- **Real-time Updates**: WebSocket support for live component status updates
- **Health Monitoring**: Continuous health checking of O-RAN SC services

### Supported O-RAN SC Components
- **E2 Manager**: E2 interface management and E2 node connections
- **Subscription Manager**: E2 subscription lifecycle management
- **App Manager**: xApp deployment and lifecycle management
- **Routing Manager**: Message routing between RIC components (planned)

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React UI      │    │  Dashboard API   │    │  O-RAN SC       │
│                 │◄──►│  Gateway         │◄──►│  Components     │
│ - Components    │    │                  │    │                 │
│ - Real-time     │    │ - REST API       │    │ - E2 Manager    │
│ - WebSocket     │    │ - gRPC Clients   │    │ - Subscription  │
└─────────────────┘    │ - Discovery      │    │   Manager       │
                       │ - WebSocket Hub  │    │ - App Manager   │
                       └──────────────────┘    └─────────────────┘
```

## API Endpoints

### Component Discovery
- `GET /api/v1/components` - List all discovered components
- `GET /api/v1/components/{id}` - Get specific component details

### E2 Manager Integration
- `GET /api/v1/e2nodes` - List all connected E2 nodes
- `GET /api/v1/e2nodes/{id}` - Get specific E2 node details

### Subscription Manager Integration
- `GET /api/v1/subscriptions` - List all active subscriptions
- `POST /api/v1/subscriptions` - Create new subscription
- `GET /api/v1/subscriptions/{id}` - Get subscription details
- `DELETE /api/v1/subscriptions/{id}` - Delete subscription

### App Manager Integration
- `GET /api/v1/xapps` - List deployed xApps
- `POST /api/v1/xapps` - Deploy new xApp
- `GET /api/v1/xapps/{name}` - Get xApp details
- `DELETE /api/v1/xapps/{name}` - Undeploy xApp

### Real-time Updates
- `GET /ws` - WebSocket endpoint for real-time updates

### Health Check
- `GET /health` - Service health status

## Configuration

The service can be configured via command-line flags:

```bash
./dashboard-api \
  -port=8080 \
  -e2mgr-endpoint=localhost:3800 \
  -submgr-endpoint=localhost:3801 \
  -appmgr-endpoint=localhost:8080
```

### Environment Variables
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `DISCOVERY_INTERVAL`: Component discovery interval (default: 30s)

## WebSocket Messages

### Message Format
```json
{
  "type": "message_type",
  "data": { ... },
  "timestamp": "2023-01-01T00:00:00Z"
}
```

### Message Types
- `welcome`: Connection established
- `component_update`: Component status changes
- `subscription_created`: New subscription created
- `subscription_deleted`: Subscription removed
- `xapp_deployed`: xApp deployment event
- `xapp_undeployed`: xApp removal event

## Development

### Building
```bash
# Build binary
go build ./cmd/dashboard-api

# Build Docker image
make docker-build-dashboard-api

# Run tests
go test ./pkg/dashboard/...
```

### Testing
The package includes comprehensive tests covering:
- HTTP endpoint functionality
- CORS middleware
- Component discovery
- WebSocket hub operations

### Deployment
Deploy using Helm:
```bash
helm install dashboard-api ./helm/dashboard-api
```

## Integration with O-RAN SC

The gateway integrates with O-RAN SC components through:

1. **gRPC Connections**: Direct gRPC clients to E2 Manager and Subscription Manager
2. **REST APIs**: HTTP client for App Manager REST endpoints
3. **Service Discovery**: Kubernetes-based discovery of O-RAN SC services
4. **Health Monitoring**: Continuous health checks and reconnection logic

## Security

- **CORS Support**: Configurable cross-origin resource sharing
- **TLS Ready**: Supports TLS for gRPC connections (configurable)
- **Non-root Container**: Runs as non-root user in Docker
- **Health Checks**: Built-in health check endpoints

## Monitoring

The service exposes metrics and status information through:
- Health check endpoint (`/health`)
- Component status in discovery service
- Real-time updates via WebSocket
- Structured logging with configurable levels

## Future Enhancements

- **Authentication**: JWT/OAuth2 integration
- **Rate Limiting**: API rate limiting and throttling
- **Metrics Export**: Prometheus metrics endpoint
- **Circuit Breaker**: Resilience patterns for O-RAN SC connections
- **Caching**: Response caching for improved performance