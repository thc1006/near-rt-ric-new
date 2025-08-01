/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import dashboardAPI, { APIError } from './api';

// Mock fetch for testing
global.fetch = jest.fn();

describe('DashboardAPI', () => {
  beforeEach(() => {
    fetch.mockClear();
  });

  describe('request method', () => {
    it('should make successful API calls', async () => {
      const mockResponse = { components: [], count: 0 };
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await dashboardAPI.request('/components');
      
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/components',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      );
      expect(result).toEqual(mockResponse);
    });

    it('should handle HTTP errors', async () => {
      fetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({ message: 'Resource not found' }),
      });

      await expect(dashboardAPI.request('/components/invalid')).rejects.toThrow(APIError);
    });

    it('should handle network errors', async () => {
      fetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(dashboardAPI.request('/components')).rejects.toThrow(APIError);
    });
  });

  describe('API methods', () => {
    it('should call getComponents', async () => {
      const mockResponse = { components: [], count: 0 };
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await dashboardAPI.getComponents();
      
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/components',
        expect.any(Object)
      );
      expect(result).toEqual(mockResponse);
    });

    it('should call getE2Nodes', async () => {
      const mockResponse = { e2nodes: [], count: 0 };
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await dashboardAPI.getE2Nodes();
      
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/e2nodes',
        expect.any(Object)
      );
      expect(result).toEqual(mockResponse);
    });

    it('should call createSubscription with POST', async () => {
      const subscriptionData = { e2nodeId: 'gnb-001', ranFunctionId: 1 };
      const mockResponse = { id: 'sub-001', ...subscriptionData };
      
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await dashboardAPI.createSubscription(subscriptionData);
      
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/subscriptions',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(subscriptionData),
        })
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe('WebSocket functionality', () => {
    let mockWebSocket;
    
    beforeEach(() => {
      // Mock WebSocket
      mockWebSocket = {
        send: jest.fn(),
        close: jest.fn(),
        readyState: WebSocket.OPEN,
        onopen: null,
        onmessage: null,
        onerror: null,
        onclose: null
      };
      
      global.WebSocket = jest.fn(() => mockWebSocket);
      global.WebSocket.CONNECTING = 0;
      global.WebSocket.OPEN = 1;
      global.WebSocket.CLOSING = 2;
      global.WebSocket.CLOSED = 3;
    });

    afterEach(() => {
      dashboardAPI.disconnectWebSocket();
    });

    it('should connect to WebSocket and send subscription message', () => {
      const onMessage = jest.fn();
      const onError = jest.fn();
      const onClose = jest.fn();
      const onOpen = jest.fn();

      dashboardAPI.connectWebSocket(onMessage, onError, onClose, onOpen);

      expect(WebSocket).toHaveBeenCalledWith('ws://localhost:8080/ws');
      
      // Simulate connection open
      mockWebSocket.onopen();
      
      expect(onOpen).toHaveBeenCalled();
      expect(mockWebSocket.send).toHaveBeenCalledWith(
        expect.stringContaining('"type":"subscribe"')
      );
    });

    it('should handle WebSocket messages', () => {
      const onMessage = jest.fn();
      const onError = jest.fn();
      const onClose = jest.fn();
      const onOpen = jest.fn();

      dashboardAPI.connectWebSocket(onMessage, onError, onClose, onOpen);

      const testMessage = {
        type: 'component_status_update',
        data: { name: 'e2manager', status: 'running' }
      };

      // Simulate message received
      mockWebSocket.onmessage({ data: JSON.stringify(testMessage) });

      expect(onMessage).toHaveBeenCalledWith(testMessage);
    });

    it('should handle WebSocket errors', () => {
      const onMessage = jest.fn();
      const onError = jest.fn();
      const onClose = jest.fn();
      const onOpen = jest.fn();

      dashboardAPI.connectWebSocket(onMessage, onError, onClose, onOpen);

      const testError = new Error('Connection failed');
      mockWebSocket.onerror(testError);

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
    });

    it('should handle WebSocket close and attempt reconnection', (done) => {
      const onMessage = jest.fn();
      const onError = jest.fn();
      const onClose = jest.fn((event) => {
        expect(event).toEqual({ code: 1006, reason: 'Connection lost' });
        done();
      });
      const onOpen = jest.fn();

      dashboardAPI.connectWebSocket(onMessage, onError, onClose, onOpen);

      // Simulate unexpected close
      setTimeout(() => {
        mockWebSocket.onclose({ code: 1006, reason: 'Connection lost' });
      }, 100);
    });

    it('should send WebSocket messages when connected', () => {
      dashboardAPI.connectWebSocket(jest.fn(), jest.fn(), jest.fn(), jest.fn());
      
      const testMessage = { type: 'test', data: 'hello' };
      const result = dashboardAPI.sendWebSocketMessage(testMessage);

      expect(result).toBe(true);
      expect(mockWebSocket.send).toHaveBeenCalledWith(JSON.stringify(testMessage));
    });

    it('should not send messages when disconnected', () => {
      mockWebSocket.readyState = WebSocket.CLOSED;
      
      const testMessage = { type: 'test', data: 'hello' };
      const result = dashboardAPI.sendWebSocketMessage(testMessage);

      expect(result).toBe(false);
      expect(mockWebSocket.send).not.toHaveBeenCalled();
    });

    it('should return correct WebSocket state', () => {
      dashboardAPI.ws = mockWebSocket;
      
      mockWebSocket.readyState = WebSocket.CONNECTING;
      expect(dashboardAPI.getWebSocketState()).toBe('CONNECTING');

      mockWebSocket.readyState = WebSocket.OPEN;
      expect(dashboardAPI.getWebSocketState()).toBe('OPEN');

      mockWebSocket.readyState = WebSocket.CLOSING;
      expect(dashboardAPI.getWebSocketState()).toBe('CLOSING');

      mockWebSocket.readyState = WebSocket.CLOSED;
      expect(dashboardAPI.getWebSocketState()).toBe('CLOSED');
    });

    it('should disconnect WebSocket cleanly', () => {
      dashboardAPI.connectWebSocket(jest.fn(), jest.fn(), jest.fn(), jest.fn());
      
      dashboardAPI.disconnectWebSocket();

      expect(mockWebSocket.close).toHaveBeenCalledWith(1000, 'Client disconnect');
    });
  });
});