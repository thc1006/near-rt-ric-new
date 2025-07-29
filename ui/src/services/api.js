/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

/**
 * API service layer for Dashboard API Gateway
 * Handles HTTP requests to the dashboard gateway and error handling
 */

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';
const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

class APIError extends Error {
  constructor(message, status, response) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.response = response;
  }
}

class DashboardAPI {
  constructor(baseUrl = API_BASE_URL) {
    this.baseUrl = baseUrl;
    this.ws = null;
    this.wsReconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000; // Start with 1 second
  }

  // Generic HTTP request method with error handling
  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers
        },
        ...options
      });

      if (!response.ok) {
        let errorMessage = `HTTP error! status: ${response.status}`;
        let errorData = null;
        
        try {
          errorData = await response.json();
          errorMessage = errorData.message || errorMessage;
        } catch (e) {
          // If response is not JSON, use status text
          errorMessage = response.statusText || errorMessage;
        }
        
        throw new APIError(errorMessage, response.status, errorData);
      }

      return response.json();
    } catch (error) {
      if (error instanceof APIError) {
        throw error;
      }
      
      // Network or other errors
      throw new APIError(
        `Network error: ${error.message}`,
        0,
        null
      );
    }
  }

  // Component Discovery API
  async getComponents() {
    return this.request('/components');
  }

  async getComponent(id) {
    return this.request(`/components/${id}`);
  }

  // E2 Manager API
  async getE2Nodes() {
    return this.request('/e2nodes');
  }

  async getE2Node(id) {
    return this.request(`/e2nodes/${id}`);
  }

  // Subscription Manager API
  async getSubscriptions() {
    return this.request('/subscriptions');
  }

  async createSubscription(subscriptionData) {
    return this.request('/subscriptions', {
      method: 'POST',
      body: JSON.stringify(subscriptionData)
    });
  }

  async getSubscription(id) {
    return this.request(`/subscriptions/${id}`);
  }

  async deleteSubscription(id) {
    return this.request(`/subscriptions/${id}`, {
      method: 'DELETE'
    });
  }

  // App Manager API
  async getXApps() {
    return this.request('/xapps');
  }

  async deployXApp(xappData) {
    return this.request('/xapps', {
      method: 'POST',
      body: JSON.stringify(xappData)
    });
  }

  async getXApp(name) {
    return this.request(`/xapps/${name}`);
  }

  async undeployXApp(name) {
    return this.request(`/xapps/${name}`, {
      method: 'DELETE'
    });
  }

  async restartXApp(name) {
    return this.request(`/xapps/${name}/restart`, {
      method: 'POST'
    });
  }

  async scaleXApp(name, instances) {
    return this.request(`/xapps/${name}/scale`, {
      method: 'POST',
      body: JSON.stringify({ instances })
    });
  }

  async getXAppLogs(name, lines = 100) {
    return this.request(`/xapps/${name}/logs?lines=${lines}`);
  }

  // Health check API
  async getHealth() {
    const response = await fetch(`${this.baseUrl.replace('/api/v1', '')}/health`);
    if (!response.ok) {
      throw new APIError(`Health check failed: ${response.status}`, response.status);
    }
    return response.json();
  }

  // WebSocket connection management
  connectWebSocket(onMessage, onError, onClose, onOpen) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected');
      return;
    }

    // Close existing connection if any
    if (this.ws) {
      this.ws.close();
    }

    try {
      console.log(`Connecting to WebSocket at ${WS_URL}`);
      this.ws = new WebSocket(WS_URL);
      
      this.ws.onopen = () => {
        console.log('WebSocket connected successfully');
        this.wsReconnectAttempts = 0;
        this.reconnectDelay = 1000;
        
        // Send subscription message for real-time updates
        const subscriptionMessage = {
          type: 'subscribe',
          data: {
            events: [
              'component_status_update',
              'component_discovered',
              'component_removed',
              'e2node_connected',
              'e2node_disconnected',
              'subscription_created',
              'subscription_deleted',
              'subscription_failed',
              'xapp_deployed',
              'xapp_undeployed',
              'xapp_status_changed',
              'alarm_raised',
              'alarm_cleared',
              'system_event'
            ]
          }
        };
        
        this.ws.send(JSON.stringify(subscriptionMessage));
        
        if (onOpen) onOpen();
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('WebSocket message received:', message);
          
          // Validate message structure
          if (message && typeof message === 'object' && message.type) {
            if (onMessage) onMessage(message);
          } else {
            console.warn('Invalid WebSocket message format:', message);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error, event.data);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        if (onError) onError(new Error('WebSocket connection error'));
      };

      this.ws.onclose = (event) => {
        console.log(`WebSocket disconnected: code=${event.code}, reason=${event.reason || 'No reason provided'}`);
        
        if (onClose) onClose(event);
        
        // Attempt to reconnect if not a clean close and within retry limits
        if (event.code !== 1000 && this.wsReconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect(onMessage, onError, onClose, onOpen);
        } else if (this.wsReconnectAttempts >= this.maxReconnectAttempts) {
          console.error('Max WebSocket reconnection attempts reached');
          if (onError) onError(new Error('Max reconnection attempts reached'));
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      if (onError) onError(error);
    }
  }

  scheduleReconnect(onMessage, onError, onClose, onOpen) {
    this.wsReconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.wsReconnectAttempts - 1), 30000); // Exponential backoff with max 30s
    
    console.log(`Scheduling WebSocket reconnection ${this.wsReconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      if (this.wsReconnectAttempts <= this.maxReconnectAttempts) {
        this.connectWebSocket(onMessage, onError, onClose, onOpen);
      }
    }, delay);
  }

  disconnectWebSocket() {
    if (this.ws) {
      console.log('Disconnecting WebSocket');
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
    // Reset reconnection state
    this.wsReconnectAttempts = 0;
  }

  isWebSocketConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }

  getWebSocketState() {
    if (!this.ws) return 'CLOSED';
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'CONNECTING';
      case WebSocket.OPEN:
        return 'OPEN';
      case WebSocket.CLOSING:
        return 'CLOSING';
      case WebSocket.CLOSED:
        return 'CLOSED';
      default:
        return 'UNKNOWN';
    }
  }

  sendWebSocketMessage(message) {
    if (this.isWebSocketConnected()) {
      try {
        this.ws.send(JSON.stringify(message));
        console.log('WebSocket message sent:', message);
        return true;
      } catch (error) {
        console.error('Failed to send WebSocket message:', error);
        return false;
      }
    } else {
      console.warn('Cannot send WebSocket message: connection not open');
      return false;
    }
  }
}

// Create singleton instance
const dashboardAPI = new DashboardAPI();

export default dashboardAPI;
export { APIError };