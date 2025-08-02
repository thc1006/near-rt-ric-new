import { useState, useEffect, useCallback, useRef } from 'react';
import metricsService from '../services/metricsService';

// Hook for fetching and managing metrics data
export const useMetrics = (metricType, refreshInterval = 30000) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const intervalRef = useRef(null);
  const mountedRef = useRef(true);

  const fetchMetrics = useCallback(async () => {
    if (!mountedRef.current) return;

    try {
      setError(null);
      let result;

      switch (metricType) {
        case 'platform':
          result = await metricsService.getPlatformMetrics();
          break;
        case 'e2':
          result = await metricsService.getE2Metrics();
          break;
        case 'a1':
          result = await metricsService.getA1Metrics();
          break;
        case 'o1':
          result = await metricsService.getO1Metrics();
          break;
        case 'security':
          result = await metricsService.getSecurityMetrics();
          break;
        default:
          throw new Error(`Unknown metric type: ${metricType}`);
      }

      if (mountedRef.current) {
        setData(result);
        setLoading(false);
      }
    } catch (err) {
      if (mountedRef.current) {
        setError(err.message);
        setLoading(false);
      }
    }
  }, [metricType]);

  useEffect(() => {
    mountedRef.current = true;
    
    // Initial fetch
    fetchMetrics();

    // Set up polling
    if (refreshInterval > 0) {
      intervalRef.current = setInterval(fetchMetrics, refreshInterval);
    }

    return () => {
      mountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchMetrics, refreshInterval]);

  const refresh = useCallback(() => {
    setLoading(true);
    fetchMetrics();
  }, [fetchMetrics]);

  return { data, loading, error, refresh };
};

// Hook for real-time metrics updates via WebSocket
export const useRealTimeMetrics = (metricTypes = ['platform']) => {
  const [data, setData] = useState({});
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const subscriberKey = useRef(`subscriber_${Date.now()}_${Math.random()}`);

  useEffect(() => {
    const handleMetricsUpdate = (metrics) => {
      setData(prevData => ({
        ...prevData,
        ...metrics,
        lastUpdate: new Date().toISOString(),
      }));
      setConnected(true);
      setError(null);
    };

    // Subscribe to real-time updates
    metricsService.subscribe(subscriberKey.current, handleMetricsUpdate);

    return () => {
      metricsService.unsubscribe(subscriberKey.current);
    };
  }, []);

  return { data, connected, error };
};

// Hook for time series data (charts)
export const useTimeSeriesMetrics = (query, duration = '1h', step = '15s', refreshInterval = 60000) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const intervalRef = useRef(null);
  const mountedRef = useRef(true);

  const fetchTimeSeriesData = useCallback(async () => {
    if (!mountedRef.current || !query) return;

    try {
      setError(null);
      const result = await metricsService.getTimeSeriesData(query, duration, step);
      
      if (mountedRef.current) {
        setData(result);
        setLoading(false);
      }
    } catch (err) {
      if (mountedRef.current) {
        setError(err.message);
        setLoading(false);
      }
    }
  }, [query, duration, step]);

  useEffect(() => {
    mountedRef.current = true;
    
    if (query) {
      // Initial fetch
      fetchTimeSeriesData();

      // Set up polling
      if (refreshInterval > 0) {
        intervalRef.current = setInterval(fetchTimeSeriesData, refreshInterval);
      }
    }

    return () => {
      mountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchTimeSeriesData, refreshInterval]);

  const refresh = useCallback(() => {
    if (query) {
      setLoading(true);
      fetchTimeSeriesData();
    }
  }, [fetchTimeSeriesData, query]);

  return { data, loading, error, refresh };
};

// Hook for custom Prometheus queries
export const usePrometheusQuery = (query, refreshInterval = 30000) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const intervalRef = useRef(null);
  const mountedRef = useRef(true);

  const executeQuery = useCallback(async () => {
    if (!mountedRef.current || !query) return;

    try {
      setError(null);
      const result = await metricsService.queryMetrics(query);
      
      if (mountedRef.current) {
        setData(result);
        setLoading(false);
      }
    } catch (err) {
      if (mountedRef.current) {
        setError(err.message);
        setLoading(false);
      }
    }
  }, [query]);

  useEffect(() => {
    mountedRef.current = true;
    
    if (query) {
      // Initial fetch
      executeQuery();

      // Set up polling
      if (refreshInterval > 0) {
        intervalRef.current = setInterval(executeQuery, refreshInterval);
      }
    }

    return () => {
      mountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [executeQuery, refreshInterval]);

  const refresh = useCallback(() => {
    if (query) {
      setLoading(true);
      executeQuery();
    }
  }, [executeQuery, query]);

  return { data, loading, error, refresh };
};

// Hook for metric alerts
export const useMetricAlerts = () => {
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        // This would typically fetch from Alertmanager API
        // For now, we'll simulate some alerts based on metrics
        const platformMetrics = await metricsService.getPlatformMetrics();
        const e2Metrics = await metricsService.getE2Metrics();
        
        const simulatedAlerts = [];
        
        // Check component health
        if (platformMetrics.componentStatus?.result) {
          platformMetrics.componentStatus.result.forEach(metric => {
            if (parseFloat(metric.value[1]) === 0) {
              simulatedAlerts.push({
                id: `component_down_${metric.metric.job}`,
                severity: 'critical',
                summary: `Component ${metric.metric.job} is down`,
                description: `The ${metric.metric.job} component is not responding`,
                timestamp: new Date().toISOString(),
                status: 'firing',
              });
            }
          });
        }
        
        // Check E2 node connectivity
        if (e2Metrics.connectedNodes?.result?.[0]) {
          const nodeCount = parseFloat(e2Metrics.connectedNodes.result[0].value[1]);
          if (nodeCount === 0) {
            simulatedAlerts.push({
              id: 'no_e2_nodes',
              severity: 'critical',
              summary: 'No E2 nodes connected',
              description: 'There are currently no E2 nodes connected to the platform',
              timestamp: new Date().toISOString(),
              status: 'firing',
            });
          }
        }
        
        setAlerts(simulatedAlerts);
        setLoading(false);
      } catch (err) {
        setError(err.message);
        setLoading(false);
      }
    };

    fetchAlerts();
    const interval = setInterval(fetchAlerts, 60000); // Check every minute

    return () => clearInterval(interval);
  }, []);

  return { alerts, loading, error };
};