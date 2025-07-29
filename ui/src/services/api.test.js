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
});