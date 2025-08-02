import { API_BASE_URL } from './api';

class MetricsService {
  constructor() {
    this.prometheusUrl = `${API_BASE_URL}/prometheus`;
    this.websocketUrl = `${API_BASE_URL.replace('http', 'ws')}/ws/metrics`;
    this.websocket = null;
    this.subscribers = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
  }

  // Query Prometheus metrics
  async queryMetrics(query, time = null) {
    try {
      const params = new URLSearchParams({ query });
      if (time) {
        params.append('time', time);
      }

      const response = await fetch(`${this.prometheusUrl}/api/v1/query?${params}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.data;
    } catch (error) {
      console.error('Error querying metrics:', error);
      throw error;
    }
  }

  // Query Prometheus metrics over a time range
  async queryRangeMetrics(query, start, end, step = '15s') {
    try {
      const params = new URLSearchParams({
        query,
        start: start.toISOString(),
        end: end.toISOString(),
        step,
      });

      const response = await fetch(`${this.prometheusUrl}/api/v1/query_range?${params}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.data;
    } catch (error) {
      console.error('Error querying range metrics:', error);
      throw error;
    }
  }

  // Get platform overview metrics
  async getPlatformMetrics() {
    const queries = {
      componentStatus: 'up{job=~"e2term|e2mgr|submgr|rtmgr|appmgr|a1mediator|o1mediator|dashboard"}',
      httpRequestRate: 'rate(http_requests_total[5m])',
      memoryUsage: 'process_resident_memory_bytes{job=~"e2term|e2mgr|submgr|rtmgr|appmgr|a1mediator|o1mediator|dashboard"}',
      cpuUsage: 'rate(process_cpu_seconds_total{job=~"e2term|e2mgr|submgr|rtmgr|appmgr|a1mediator|o1mediator|dashboard"}[5m])',
    };

    const results = {};
    for (const [key, query] of Object.entries(queries)) {
      try {
        results[key] = await this.queryMetrics(query);
      } catch (error) {
        console.error(`Error fetching ${key}:`, error);
        results[key] = { result: [] };
      }
    }

    return results;
  }

  // Get E2 interface metrics
  async getE2Metrics() {
    const queries = {
      connectedNodes: 'e2_nodes_connected',
      activeSubscriptions: 'e2_subscriptions_active',
      messageRate: 'rate(e2ap_messages_total[5m])',
      indicationLatency: 'histogram_quantile(0.95, rate(e2_indication_processing_duration_seconds_bucket[5m]))',
      subscriptionSuccessRate: 'rate(e2_subscription_requests_total{status="success"}[5m]) / rate(e2_subscription_requests_total[5m])',
    };

    const results = {};
    for (const [key, query] of Object.entries(queries)) {
      try {
        results[key] = await this.queryMetrics(query);
      } catch (error) {
        console.error(`Error fetching E2 ${key}:`, error);
        results[key] = { result: [] };
      }
    }

    return results;
  }

  // Get A1 interface metrics
  async getA1Metrics() {
    const queries = {
      policyTypes: 'a1_policy_types_total',
      policyInstances: 'a1_policy_instances_total',
      requestRate: 'rate(a1_policy_requests_total[5m])',
      processingLatency: 'histogram_quantile(0.95, rate(a1_policy_processing_duration_seconds_bucket[5m]))',
    };

    const results = {};
    for (const [key, query] of Object.entries(queries)) {
      try {
        results[key] = await this.queryMetrics(query);
      } catch (error) {
        console.error(`Error fetching A1 ${key}:`, error);
        results[key] = { result: [] };
      }
    }

    return results;
  }

  // Get O1 interface metrics
  async getO1Metrics() {
    const queries = {
      configOperations: 'rate(o1_config_operations_total[5m])',
      alarmEvents: 'rate(o1_alarm_events_total[5m])',
      netconfSessions: 'o1_netconf_sessions_active',
      operationLatency: 'histogram_quantile(0.95, rate(o1_operation_processing_duration_seconds_bucket[5m]))',
    };

    const results = {};
    for (const [key, query] of Object.entries(queries)) {
      try {
        results[key] = await this.queryMetrics(query);
      } catch (error) {
        console.error(`Error fetching O1 ${key}:`, error);
        results[key] = { result: [] };
      }
    }

    return results;
  }

  // Get security metrics
  async getSecurityMetrics() {
    const queries = {
      authAttempts: 'rate(authentication_attempts_total[5m])',
      authFailures: 'rate(authentication_attempts_total{status="failure"}[5m])',
      authzChecks: 'rate(authorization_checks_total[5m])',
      securityEvents: 'rate(security_events_total[5m])',
    };

    const results = {};
    for (const [key, query] of Object.entries(queries)) {
      try {
        results[key] = await this.queryMetrics(query);
      } catch (error) {
        console.error(`Error fetching security ${key}:`, error);
        results[key] = { result: [] };
      }
    }

    return results;
  }

  // Get time series data for charts
  async getTimeSeriesData(query, duration = '1h', step = '15s') {
    const end = new Date();
    const start = new Date(end.getTime() - this.parseDuration(duration));
    
    return await this.queryRangeMetrics(query, start, end, step);
  }

  // Parse duration string (e.g., '1h', '30m', '24h')
  parseDuration(duration) {
    const match = duration.match(/^(\d+)([smhd])$/);
    if (!match) return 3600000; // Default to 1 hour

    const value = parseInt(match[1]);
    const unit = match[2];

    switch (unit) {
      case 's': return value * 1000;
      case 'm': return value * 60 * 1000;
      case 'h': return value * 60 * 60 * 1000;
      case 'd': return value * 24 * 60 * 60 * 1000;
      default: return 3600000;
    }
  }

  // WebSocket connection for real-time metrics
  connectWebSocket() {
    if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      this.websocket = new WebSocket(this.websocketUrl);

      this.websocket.onopen = () => {
        console.log('Metrics WebSocket connected');
        this.reconnectAttempts = 0;
        
        // Subscribe to real-time metrics
        this.websocket.send(JSON.stringify({
          type: 'subscribe',
          metrics: ['platform', 'e2', 'a1', 'o1', 'security']
        }));
      };

      this.websocket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.handleMetricsUpdate(data);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      this.websocket.onclose = () => {
        console.log('Metrics WebSocket disconnected');
        this.websocket = null;
        this.scheduleReconnect();
      };

      this.websocket.onerror = (error) => {
        console.error('Metrics WebSocket error:', error);
      };
    } catch (error) {
      console.error('Error creating WebSocket connection:', error);
      this.scheduleReconnect();
    }
  }

  // Handle real-time metrics updates
  handleMetricsUpdate(data) {
    if (data.type === 'metrics_update') {
      // Notify all subscribers
      this.subscribers.forEach((callback, key) => {
        try {
          callback(data.metrics);
        } catch (error) {
          console.error(`Error in metrics subscriber ${key}:`, error);
        }
      });
    }
  }

  // Subscribe to real-time metrics updates
  subscribe(key, callback) {
    this.subscribers.set(key, callback);
    
    // Connect WebSocket if not already connected
    if (!this.websocket || this.websocket.readyState !== WebSocket.OPEN) {
      this.connectWebSocket();
    }
  }

  // Unsubscribe from real-time metrics updates
  unsubscribe(key) {
    this.subscribers.delete(key);
    
    // Close WebSocket if no subscribers
    if (this.subscribers.size === 0 && this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }
  }

  // Schedule WebSocket reconnection
  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
    this.reconnectAttempts++;

    setTimeout(() => {
      console.log(`Attempting to reconnect WebSocket (attempt ${this.reconnectAttempts})`);
      this.connectWebSocket();
    }, delay);
  }

  // Disconnect WebSocket
  disconnect() {
    if (this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }
    this.subscribers.clear();
  }

  // Format metric value for display
  formatMetricValue(value, unit = '') {
    if (typeof value !== 'number') return 'N/A';

    if (unit === 'bytes') {
      return this.formatBytes(value);
    } else if (unit === 'seconds') {
      return this.formatDuration(value * 1000);
    } else if (unit === 'percent') {
      return `${(value * 100).toFixed(1)}%`;
    } else if (value >= 1000000) {
      return `${(value / 1000000).toFixed(1)}M`;
    } else if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}K`;
    } else {
      return value.toFixed(2);
    }
  }

  // Format bytes for display
  formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  }

  // Format duration for display
  formatDuration(ms) {
    if (ms < 1000) return `${ms.toFixed(0)}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
    return `${(ms / 3600000).toFixed(1)}h`;
  }
}

// Export singleton instance
export default new MetricsService();