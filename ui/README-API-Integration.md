# Dashboard API Integration

This document describes the API integration implemented for the O-RAN Interactive Operations Console React dashboard.

## Overview

The React dashboard has been migrated from using mock data to consuming real API endpoints from the Dashboard API Gateway running on `localhost:8080/api/v1`.

## Architecture

### API Service Layer (`src/services/api.js`)

- **DashboardAPI Class**: Singleton service for making HTTP requests to the dashboard gateway
- **Error Handling**: Custom `APIError` class for structured error handling
- **WebSocket Support**: Real-time updates via WebSocket connection to `localhost:8080/ws`
- **Automatic Reconnection**: WebSocket reconnection with exponential backoff

### Custom Hooks (`src/hooks/useAPI.js`)

- **useAPI**: Generic hook for API calls with loading states and error handling
- **useWebSocket**: WebSocket connection management with automatic reconnection
- **useComponents**: Hook for component discovery API
- **useE2Nodes**: Hook for E2 node management API
- **useSubscriptions**: Hook for subscription management API
- **useXApps**: Hook for xApp management API
- **useHealth**: Hook for health check API

### Error Handling Components

- **ErrorBoundary**: React error boundary for catching and displaying React errors
- **ErrorDisplay**: Component for displaying API errors with retry functionality
- **LoadingDisplay**: Loading state component with spinner
- **ConnectionStatus**: WebSocket connection status indicator

## API Endpoints

The dashboard consumes the following REST API endpoints:

### Component Discovery
- `GET /api/v1/components` - Get all discovered components
- `GET /api/v1/components/{id}` - Get specific component details

### E2 Manager Integration
- `GET /api/v1/e2nodes` - Get all E2 nodes
- `GET /api/v1/e2nodes/{id}` - Get specific E2 node details

### Subscription Manager Integration
- `GET /api/v1/subscriptions` - Get all subscriptions
- `POST /api/v1/subscriptions` - Create new subscription
- `GET /api/v1/subscriptions/{id}` - Get specific subscription
- `DELETE /api/v1/subscriptions/{id}` - Delete subscription

### App Manager Integration
- `GET /api/v1/xapps` - Get all deployed xApps
- `POST /api/v1/xapps` - Deploy new xApp
- `GET /api/v1/xapps/{name}` - Get specific xApp details
- `DELETE /api/v1/xapps/{name}` - Undeploy xApp

### Health Check
- `GET /health` - API health status

### WebSocket
- `WS /ws` - Real-time updates for component status, subscriptions, and xApp deployments

## Configuration

### Environment Variables

- `REACT_APP_API_BASE_URL`: Dashboard API base URL (default: `http://localhost:8080/api/v1`)
- `REACT_APP_WS_URL`: WebSocket URL (default: `ws://localhost:8080/ws`)

### Example `.env` file:
```
REACT_APP_API_BASE_URL=http://localhost:8080/api/v1
REACT_APP_WS_URL=ws://localhost:8080/ws
```

## Error Handling

### Network Errors
- Connection failures display user-friendly error messages
- Retry functionality available for failed requests
- Automatic fallback to cached data when possible

### API Errors
- HTTP status codes mapped to meaningful error messages
- 404: Resource not found
- 500: Internal server error
- 503: Service unavailable

### WebSocket Errors
- Automatic reconnection with exponential backoff
- Connection status indicator in header
- Graceful degradation when WebSocket unavailable

## Data Flow

1. **Component Discovery**: Dashboard auto-discovers O-RAN SC components on load
2. **Real-time Updates**: WebSocket provides live updates for component status changes
3. **KPI Processing**: E2 nodes, xApps, and subscription data processed into KPIs
4. **Alarm Generation**: API errors and connection issues generate dashboard alarms
5. **Log Streaming**: WebSocket messages displayed as real-time logs

## Testing

### Unit Tests
- API service methods tested with mocked fetch
- Error handling scenarios covered
- WebSocket connection logic tested

### Integration Tests
- React components tested with mocked API responses
- Error states and loading states verified
- User interactions tested

### Running Tests
```bash
npm test
```

## Development

### Starting the Dashboard
```bash
npm start
```

### Building for Production
```bash
npm run build
```

### Prerequisites
- Dashboard API Gateway must be running on `localhost:8080`
- O-RAN SC components should be deployed and accessible via the gateway

## Migration Notes

### Replaced Mock Data
- Static network function list → Dynamic component discovery
- Random KPI generation → Real metrics from O-RAN SC APIs
- Static alarms → Dynamic error-based alarms
- Simulated logs → Real-time WebSocket message logs

### Preserved Functionality
- Dashboard layout and styling maintained
- Auto-refresh capability (now every 30 seconds)
- Real-time updates (now via WebSocket instead of intervals)
- Error handling and user feedback

## Future Enhancements

1. **Authentication**: Add JWT token support for secured API access
2. **Caching**: Implement intelligent caching for improved performance
3. **Offline Mode**: Add service worker for offline functionality
4. **Advanced Filtering**: Add filtering and search capabilities for large datasets
5. **Custom Dashboards**: Allow users to customize dashboard panels and layouts