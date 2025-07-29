# WebSocket Implementation for Real-Time Updates

## Overview

This document describes the WebSocket implementation for task 4.2 of the O-RAN SC migration project. The implementation provides real-time updates for component status, alarms, and events in the React dashboard.

## Implementation Details

### 1. WebSocket Connection Management

**Location**: `src/services/api.js`

- **Connection URL**: `ws://localhost:8080/ws`
- **Auto-reconnection**: Exponential backoff with max 5 attempts
- **Connection lifecycle**: Proper handling of connect, disconnect, and error states
- **Message validation**: JSON parsing with error handling

### 2. Real-Time Message Processing

**Supported Message Types**:
- `component_status_update` - Component state changes
- `component_discovered` - New component detection
- `component_removed` - Component removal
- `e2node_connected` - E2 node connections
- `e2node_disconnected` - E2 node disconnections
- `subscription_created` - New subscriptions
- `subscription_deleted` - Subscription removal
- `subscription_failed` - Subscription failures
- `xapp_deployed` - xApp deployments
- `xapp_undeployed` - xApp removal
- `xapp_status_changed` - xApp state changes
- `alarm_raised` - New alarms
- `alarm_cleared` - Alarm resolution
- `system_event` - General system events

### 3. React Integration

**Location**: `src/hooks/useAPI.js`

The `useWebSocket` hook provides:
- Connection state management
- Message buffering (last 100 messages)
- Error handling and recovery
- Reconnection attempt tracking

**Location**: `src/App.js`

Real-time message processing triggers:
- API data refresh for relevant components
- Alarm management (add/remove)
- Real-time log updates (last 50 entries)

### 4. UI Components

**Connection Status Display**: `src/components/ErrorDisplay.js`
- Real-time connection status indicator
- Reconnection progress display
- Manual reconnect button
- Visual connection state indicators

**Enhanced CSS**: `src/App.css`
- Connection status styling
- Animated indicators for connecting state
- Responsive design for status display

## Features Implemented

### ✅ WebSocket Connection Lifecycle
- Automatic connection on app startup
- Graceful disconnection on app unmount
- Exponential backoff reconnection strategy
- Connection state monitoring

### ✅ Real-Time Message Processing
- JSON message parsing and validation
- Type-based message routing
- Error handling for malformed messages
- Message buffering and history

### ✅ Dashboard Integration
- Automatic API data refresh on relevant events
- Real-time alarm management
- Live log streaming
- Connection status display

### ✅ Error Handling
- Network error recovery
- Connection timeout handling
- Message parsing error handling
- User-friendly error display

## Testing

**Test Coverage**:
- WebSocket connection lifecycle tests
- Message processing tests
- Error handling tests
- Integration tests with React components

**Test Files**:
- `src/services/api.test.js` - API service tests
- `src/hooks/useWebSocket.test.js` - Hook tests
- `src/websocket.integration.test.js` - Integration tests

## Configuration

**Environment Variables**:
- `REACT_APP_WS_URL` - WebSocket endpoint URL (default: `ws://localhost:8080/ws`)
- `REACT_APP_API_BASE_URL` - API base URL (default: `http://localhost:8080/api/v1`)

## Usage

The WebSocket connection is automatically established when the React app loads. No manual configuration is required. The connection status is displayed in the header, and real-time updates are processed automatically.

### Manual Reconnection

Users can manually reconnect using the "Reconnect" button that appears when the connection is lost.

### Message Sending

The API service provides a `sendWebSocketMessage()` method for sending messages to the server:

```javascript
import dashboardAPI from './services/api';

// Send a message
const success = dashboardAPI.sendWebSocketMessage({
  type: 'custom_event',
  data: { message: 'Hello server' }
});
```

## Requirements Fulfilled

This implementation fulfills the requirements specified in task 4.2:

- ✅ **Connect to WebSocket endpoint at localhost:8080/ws** - Implemented with configurable URL
- ✅ **Handle WebSocket connection lifecycle** - Full lifecycle management with reconnection
- ✅ **Process real-time messages for component status, alarms, and events** - Comprehensive message processing

The implementation supports all O-RAN SC component types and provides a robust foundation for real-time dashboard updates.