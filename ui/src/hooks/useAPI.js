/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import { useState, useEffect, useCallback } from 'react';
import dashboardAPI, { APIError } from '../services/api';

/**
 * Custom hook for managing API calls with loading states and error handling
 */
export function useAPI(apiCall, dependencies = []) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiCall();
      setData(result);
    } catch (err) {
      console.error('API call failed:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [apiCall, ...dependencies]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}

/**
 * Custom hook for managing WebSocket connections with automatic reconnection
 */
export function useWebSocket() {
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState([]);
  const [error, setError] = useState(null);

  const connect = useCallback(() => {
    dashboardAPI.connectWebSocket(
      (message) => {
        setMessages(prev => [...prev.slice(-99), message]); // Keep last 100 messages
      },
      (error) => {
        setError(error);
        setConnected(false);
      },
      (event) => {
        setConnected(false);
        if (event.code !== 1000) {
          setError(new Error(`WebSocket closed unexpectedly: ${event.reason || event.code}`));
        }
      }
    );
    setConnected(true);
    setError(null);
  }, []);

  const disconnect = useCallback(() => {
    dashboardAPI.disconnectWebSocket();
    setConnected(false);
  }, []);

  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  return { connected, messages, error, connect, disconnect };
}

/**
 * Custom hook for managing component discovery
 */
export function useComponents() {
  return useAPI(() => dashboardAPI.getComponents());
}

/**
 * Custom hook for managing E2 nodes
 */
export function useE2Nodes() {
  return useAPI(() => dashboardAPI.getE2Nodes());
}

/**
 * Custom hook for managing subscriptions
 */
export function useSubscriptions() {
  return useAPI(() => dashboardAPI.getSubscriptions());
}

/**
 * Custom hook for managing xApps
 */
export function useXApps() {
  return useAPI(() => dashboardAPI.getXApps());
}

/**
 * Custom hook for managing health status
 */
export function useHealth() {
  return useAPI(() => dashboardAPI.getHealth(), []);
}

/**
 * Utility function to format API errors for display
 */
export function formatAPIError(error) {
  if (error instanceof APIError) {
    switch (error.status) {
      case 0:
        return 'Network connection failed. Please check if the dashboard API is running.';
      case 404:
        return 'Resource not found.';
      case 500:
        return 'Internal server error. Please try again later.';
      case 503:
        return 'Service unavailable. The dashboard API may be starting up.';
      default:
        return error.message || 'An unexpected error occurred.';
    }
  }
  return error?.message || 'An unknown error occurred.';
}