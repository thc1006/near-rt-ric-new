/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import { render, screen, waitFor, act } from '@testing-library/react';
import App from './App';
import dashboardAPI from './services/api';

// Mock the dashboard API
jest.mock('./services/api');

describe('WebSocket Integration Tests', () => {
  let mockWebSocketCallbacks = {};

  beforeEach(() => {
    // Reset all mocks
    jest.clearAllMocks();
    mockWebSocketCallbacks = {};
    
    // Setup default mock responses
    dashboardAPI.getComponents.mockResolvedValue({
      components: [
        { name: 'e2manager', type: 'e2manager', status: 'running' },
        { name: 'submgr', type: 'submgr', status: 'running' }
      ],
      count: 2
    });
    
    dashboardAPI.getE2Nodes.mockResolvedValue({
      e2nodes: [
        { id: 'gnb-001', type: 'gnb', status: 'connected', plmnId: '001-01' }
      ],
      count: 1
    });
    
    dashboardAPI.getSubscriptions.mockResolvedValue({
      subscriptions: [
        { id: 'sub-001', status: 'active', e2nodeId: 'gnb-001' }
      ],
      count: 1
    });
    
    dashboardAPI.getXApps.mockResolvedValue({
      xapps: [
        { name: 'hello-world', status: 'running', instances: 1, version: '1.0.0' }
      ],
      count: 1
    });
    
    dashboardAPI.getHealth.mockResolvedValue({ status: 'healthy' });
    
    // Mock WebSocket functionality
    dashboardAPI.connectWebSocket.mockImplementation((onMessage, onError, onClose, onOpen) => {
      mockWebSocketCallbacks.onMessage = onMessage;
      mockWebSocketCallbacks.onError = onError;
      mockWebSocketCallbacks.onClose = onClose;
      mockWebSocketCallbacks.onOpen = onOpen;
    });
    
    dashboardAPI.disconnectWebSocket.mockImplementation(() => {});
    dashboardAPI.isWebSocketConnected.mockReturnValue(true);
    dashboardAPI.getWebSocketState.mockReturnValue('OPEN');
    dashboardAPI.sendWebSocketMessage.mockReturnValue(true);
  });

  test('should connect to WebSocket on mount', () => {
    render(<App />);
    
    expect(dashboardAPI.connectWebSocket).toHaveBeenCalledWith(
      expect.any(Function), // onMessage
      expect.any(Function), // onError
      expect.any(Function), // onClose
      expect.any(Function)  // onOpen
    );
  });

  test('should process WebSocket component status updates', async () => {
    render(<App />);
    
    // Simulate WebSocket connection
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate component status update message
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'component_status_update',
        data: { name: 'e2manager', status: 'restarting' }
      });
    });
    
    // Should trigger component data refresh
    await waitFor(() => {
      expect(dashboardAPI.getComponents).toHaveBeenCalledTimes(2); // Initial + refresh
    });
  });

  test('should process WebSocket E2 node events', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate E2 node connection event
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'e2node_connected',
        data: { nodeId: 'gnb-002', type: 'gnb' }
      });
    });
    
    // Should trigger E2 nodes data refresh
    await waitFor(() => {
      expect(dashboardAPI.getE2Nodes).toHaveBeenCalledTimes(2);
    });
  });

  test('should process WebSocket subscription events', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate subscription created event
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'subscription_created',
        data: { subscriptionId: 'sub-002', e2nodeId: 'gnb-001' }
      });
    });
    
    // Should trigger subscriptions data refresh
    await waitFor(() => {
      expect(dashboardAPI.getSubscriptions).toHaveBeenCalledTimes(2);
    });
  });

  test('should process WebSocket xApp events', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate xApp deployment event
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'xapp_deployed',
        data: { name: 'traffic-steering', status: 'running' }
      });
    });
    
    // Should trigger xApps data refresh
    await waitFor(() => {
      expect(dashboardAPI.getXApps).toHaveBeenCalledTimes(2);
    });
  });

  test('should handle WebSocket alarm events', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate alarm raised event
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'alarm_raised',
        data: { 
          id: 'alarm-001', 
          severity: 'critical', 
          message: 'E2 connection lost' 
        }
      });
    });
    
    // Should display alarm in UI
    await waitFor(() => {
      expect(screen.getByText(/E2 connection lost/)).toBeInTheDocument();
      expect(screen.getByText(/CRITICAL/)).toBeInTheDocument();
    });
  });

  test('should handle WebSocket connection errors', async () => {
    render(<App />);
    
    // Simulate WebSocket error
    act(() => {
      mockWebSocketCallbacks.onError(new Error('WebSocket connection failed'));
    });
    
    // Should display connection error
    await waitFor(() => {
      expect(screen.getByText(/Connection Error/)).toBeInTheDocument();
    });
  });

  test('should display real-time logs from WebSocket messages', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Simulate system event
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'system_event',
        data: { message: 'System startup complete' }
      });
    });
    
    // Should display log entry
    await waitFor(() => {
      expect(screen.getByText(/System startup complete/)).toBeInTheDocument();
    });
  });

  test('should show WebSocket connection status', async () => {
    render(<App />);
    
    // Initially should show connecting or disconnected
    expect(screen.getByText(/Real-time Updates Active|Connecting|Disconnected/)).toBeInTheDocument();
    
    // Simulate connection open
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Should show connected status
    await waitFor(() => {
      expect(screen.getByText(/Real-time Updates Active/)).toBeInTheDocument();
    });
  });

  test('should handle WebSocket reconnection attempts', async () => {
    render(<App />);
    
    // Simulate connection open first
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Then simulate unexpected close
    act(() => {
      mockWebSocketCallbacks.onClose({ code: 1006, reason: 'Connection lost' });
    });
    
    // Should show reconnection status
    await waitFor(() => {
      expect(screen.getByText(/Reconnecting|Connection Error/)).toBeInTheDocument();
    });
  });

  test('should clear alarms when alarm_cleared event received', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // First raise an alarm
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'alarm_raised',
        data: { 
          id: 'alarm-001', 
          severity: 'warning', 
          message: 'Test alarm' 
        }
      });
    });
    
    await waitFor(() => {
      expect(screen.getByText(/Test alarm/)).toBeInTheDocument();
    });
    
    // Then clear the alarm
    act(() => {
      mockWebSocketCallbacks.onMessage({
        type: 'alarm_cleared',
        data: { id: 'alarm-001' }
      });
    });
    
    await waitFor(() => {
      expect(screen.queryByText(/Test alarm/)).not.toBeInTheDocument();
    });
  });

  test('should limit logs to 50 entries', async () => {
    render(<App />);
    
    act(() => {
      mockWebSocketCallbacks.onOpen();
    });
    
    // Send 55 log messages
    for (let i = 0; i < 55; i++) {
      act(() => {
        mockWebSocketCallbacks.onMessage({
          type: 'system_event',
          data: { message: `Log message ${i}` }
        });
      });
    }
    
    // Should only show the last 50 messages
    await waitFor(() => {
      expect(screen.queryByText(/Log message 0/)).not.toBeInTheDocument();
      expect(screen.getByText(/Log message 54/)).toBeInTheDocument();
    });
  });
});