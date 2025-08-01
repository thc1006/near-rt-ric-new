/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import { renderHook, act } from '@testing-library/react';
import { useWebSocket } from './useAPI';
import dashboardAPI from '../services/api';

// Mock the dashboard API
jest.mock('../services/api', () => ({
  connectWebSocket: jest.fn(),
  disconnectWebSocket: jest.fn(),
  getWebSocketState: jest.fn(),
  sendWebSocketMessage: jest.fn(),
}));

describe('useWebSocket hook', () => {
  let mockCallbacks = {};

  beforeEach(() => {
    jest.clearAllMocks();
    mockCallbacks = {};
    
    // Use fake timers to control intervals
    jest.useFakeTimers();
    
    // Set test timeout
    jest.setTimeout(5000);
    
    // Mock connectWebSocket to capture callbacks
    dashboardAPI.connectWebSocket.mockImplementation((onMessage, onError, onClose, onOpen) => {
      mockCallbacks.onMessage = onMessage;
      mockCallbacks.onError = onError;
      mockCallbacks.onClose = onClose;
      mockCallbacks.onOpen = onOpen;
    });
    
    dashboardAPI.getWebSocketState.mockReturnValue('CLOSED');
    dashboardAPI.sendWebSocketMessage.mockReturnValue(true);
  });

  afterEach(() => {
    // Clean up timers
    jest.clearAllTimers();
    jest.useRealTimers();
  });

  it('should initialize with default state', () => {
    const { result } = renderHook(() => useWebSocket());

    expect(result.current.connected).toBe(false);
    expect(result.current.connecting).toBe(false);
    expect(result.current.messages).toEqual([]);
    expect(result.current.error).toBe(null);
    expect(result.current.connectionState).toBe('CLOSED');
    expect(result.current.reconnectAttempts).toBe(0);
  });

  it('should connect on mount', () => {
    renderHook(() => useWebSocket());

    expect(dashboardAPI.connectWebSocket).toHaveBeenCalledWith(
      expect.any(Function),
      expect.any(Function),
      expect.any(Function),
      expect.any(Function)
    );
  });

  it('should handle WebSocket open event', () => {
    const { result } = renderHook(() => useWebSocket());

    act(() => {
      mockCallbacks.onOpen();
    });

    expect(result.current.connected).toBe(true);
    expect(result.current.connecting).toBe(false);
    expect(result.current.error).toBe(null);
    expect(result.current.reconnectAttempts).toBe(0);
  });

  it('should handle WebSocket messages', () => {
    const { result } = renderHook(() => useWebSocket());

    const testMessage = {
      type: 'component_status_update',
      data: { name: 'e2manager', status: 'running' }
    };

    act(() => {
      mockCallbacks.onMessage(testMessage);
    });

    expect(result.current.messages).toContain(testMessage);
    expect(result.current.lastMessage).toEqual(testMessage);
  });

  it('should handle WebSocket errors', () => {
    const { result } = renderHook(() => useWebSocket());

    const testError = new Error('Connection failed');

    act(() => {
      mockCallbacks.onError(testError);
    });

    expect(result.current.error).toEqual(testError);
    expect(result.current.connected).toBe(false);
    expect(result.current.connecting).toBe(false);
  });

  it('should handle WebSocket close events', () => {
    const { result } = renderHook(() => useWebSocket());

    // First connect
    act(() => {
      mockCallbacks.onOpen();
    });

    expect(result.current.connected).toBe(true);

    // Then close unexpectedly
    act(() => {
      mockCallbacks.onClose({ code: 1006, reason: 'Connection lost' });
    });

    expect(result.current.connected).toBe(false);
    expect(result.current.connecting).toBe(false);
    expect(result.current.error).toEqual(expect.any(Error));
    expect(result.current.reconnectAttempts).toBe(1);
  });

  it('should handle clean close without error', () => {
    const { result } = renderHook(() => useWebSocket());

    act(() => {
      mockCallbacks.onClose({ code: 1000, reason: 'Normal closure' });
    });

    expect(result.current.connected).toBe(false);
    expect(result.current.error).toBe(null);
    expect(result.current.reconnectAttempts).toBe(0);
  });

  it('should send messages through API', () => {
    const { result } = renderHook(() => useWebSocket());

    const testMessage = { type: 'test', data: 'hello' };
    
    act(() => {
      const success = result.current.sendMessage(testMessage);
      expect(success).toBe(true);
    });

    expect(dashboardAPI.sendWebSocketMessage).toHaveBeenCalledWith(testMessage);
  });

  it('should disconnect on unmount', () => {
    const { unmount } = renderHook(() => useWebSocket());

    unmount();

    expect(dashboardAPI.disconnectWebSocket).toHaveBeenCalled();
  });

  it('should limit messages to 100', () => {
    const { result } = renderHook(() => useWebSocket());

    // Add 105 messages
    act(() => {
      for (let i = 0; i < 105; i++) {
        mockCallbacks.onMessage({ type: 'test', data: i });
      }
    });

    expect(result.current.messages).toHaveLength(100);
    expect(result.current.messages[0].data).toBe(5); // First 5 should be removed
    expect(result.current.messages[99].data).toBe(104); // Last message should be 104
  });

  it('should clear error when receiving messages', () => {
    const { result } = renderHook(() => useWebSocket());

    // Set an error first
    act(() => {
      mockCallbacks.onError(new Error('Test error'));
    });

    expect(result.current.error).toBeTruthy();

    // Receive a message
    act(() => {
      mockCallbacks.onMessage({ type: 'test', data: 'hello' });
    });

    expect(result.current.error).toBe(null);
  });
});