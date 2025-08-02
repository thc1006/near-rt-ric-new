import React, { useState, useEffect } from 'react';
import { useMetrics, useRealTimeMetrics, useTimeSeriesMetrics, useMetricAlerts } from '../hooks/useMetrics';
import metricsService from '../services/metricsService';
import './MetricsDashboard.css';

// Metric card component
const MetricCard = ({ title, value, unit, trend, status, loading, error }) => {
  const getStatusColor = (status) => {
    switch (status) {
      case 'healthy': return '#4CAF50';
      case 'warning': return '#FF9800';
      case 'critical': return '#F44336';
      default: return '#9E9E9E';
    }
  };

  if (loading) {
    return (
      <div className="metric-card loading">
        <div className="metric-card-header">
          <h3>{title}</h3>
        </div>
        <div className="metric-card-content">
          <div className="loading-spinner"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="metric-card error">
        <div className="metric-card-header">
          <h3>{title}</h3>
        </div>
        <div className="metric-card-content">
          <span className="error-text">Error: {error}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="metric-card" style={{ borderLeftColor: getStatusColor(status) }}>
      <div className="metric-card-header">
        <h3>{title}</h3>
        {status && <span className={`status-indicator ${status}`}></span>}
      </div>
      <div className="metric-card-content">
        <div className="metric-value">
          {metricsService.formatMetricValue(value, unit)}
        </div>
        {trend && (
          <div className={`metric-trend ${trend > 0 ? 'up' : trend < 0 ? 'down' : 'stable'}`}>
            {trend > 0 ? '↗' : trend < 0 ? '↘' : '→'} {Math.abs(trend).toFixed(1)}%
          </div>
        )}
      </div>
    </div>
  );
};

// Time series chart component (simplified)
const TimeSeriesChart = ({ title, query, duration = '1h', height = 200 }) => {
  const { data, loading, error } = useTimeSeriesMetrics(query, duration);
  
  if (loading) {
    return (
      <div className="chart-container" style={{ height }}>
        <h3>{title}</h3>
        <div className="loading-spinner"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="chart-container error" style={{ height }}>
        <h3>{title}</h3>
        <span className="error-text">Error: {error}</span>
      </div>
    );
  }

  // This is a simplified representation - in a real implementation,
  // you would use a charting library like Chart.js, D3, or Recharts
  const renderSimpleChart = () => {
    if (!data?.result?.[0]?.values) {
      return <div className="no-data">No data available</div>;
    }

    const values = data.result[0].values;
    const maxValue = Math.max(...values.map(v => parseFloat(v[1])));
    const minValue = Math.min(...values.map(v => parseFloat(v[1])));
    const range = maxValue - minValue || 1;

    return (
      <div className="simple-chart">
        <svg width="100%" height={height - 60}>
          <polyline
            points={values.map((value, index) => {
              const x = (index / (values.length - 1)) * 100;
              const y = ((maxValue - parseFloat(value[1])) / range) * (height - 100) + 20;
              return `${x}%,${y}`;
            }).join(' ')}
            fill="none"
            stroke="#2196F3"
            strokeWidth="2"
          />
        </svg>
        <div className="chart-labels">
          <span>Min: {metricsService.formatMetricValue(minValue)}</span>
          <span>Max: {metricsService.formatMetricValue(maxValue)}</span>
        </div>
      </div>
    );
  };

  return (
    <div className="chart-container" style={{ height }}>
      <h3>{title}</h3>
      {renderSimpleChart()}
    </div>
  );
};

// Alert panel component
const AlertPanel = () => {
  const { alerts, loading, error } = useMetricAlerts();

  if (loading) {
    return (
      <div className="alert-panel">
        <h3>Active Alerts</h3>
        <div className="loading-spinner"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="alert-panel">
        <h3>Active Alerts</h3>
        <div className="error-text">Error loading alerts: {error}</div>
      </div>
    );
  }

  return (
    <div className="alert-panel">
      <h3>Active Alerts ({alerts.length})</h3>
      {alerts.length === 0 ? (
        <div className="no-alerts">No active alerts</div>
      ) : (
        <div className="alert-list">
          {alerts.map(alert => (
            <div key={alert.id} className={`alert-item ${alert.severity}`}>
              <div className="alert-header">
                <span className="alert-severity">{alert.severity.toUpperCase()}</span>
                <span className="alert-time">
                  {new Date(alert.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div className="alert-summary">{alert.summary}</div>
              <div className="alert-description">{alert.description}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Platform overview section
const PlatformOverview = () => {
  const { data: platformData, loading, error } = useMetrics('platform');
  const { data: realTimeData } = useRealTimeMetrics(['platform']);

  // Combine static and real-time data
  const data = { ...platformData, ...realTimeData };

  const getComponentStatus = () => {
    if (!data?.componentStatus?.result) return [];
    
    return data.componentStatus.result.map(metric => ({
      name: metric.metric.job,
      status: parseFloat(metric.value[1]) === 1 ? 'healthy' : 'critical',
      value: parseFloat(metric.value[1]),
    }));
  };

  const getAverageValue = (metricData) => {
    if (!metricData?.result) return 0;
    const values = metricData.result.map(m => parseFloat(m.value[1]));
    return values.reduce((sum, val) => sum + val, 0) / values.length;
  };

  const components = getComponentStatus();
  const healthyComponents = components.filter(c => c.status === 'healthy').length;
  const totalComponents = components.length;

  return (
    <div className="platform-overview">
      <h2>Platform Overview</h2>
      <div className="metrics-grid">
        <MetricCard
          title="Component Health"
          value={totalComponents > 0 ? `${healthyComponents}/${totalComponents}` : 'N/A'}
          status={healthyComponents === totalComponents ? 'healthy' : 'critical'}
          loading={loading}
          error={error}
        />
        <MetricCard
          title="HTTP Request Rate"
          value={getAverageValue(data?.httpRequestRate)}
          unit="req/s"
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Memory Usage"
          value={getAverageValue(data?.memoryUsage)}
          unit="bytes"
          loading={loading}
          error={error}
        />
        <MetricCard
          title="CPU Usage"
          value={getAverageValue(data?.cpuUsage) * 100}
          unit="percent"
          loading={loading}
          error={error}
        />
      </div>
      
      <div className="charts-grid">
        <TimeSeriesChart
          title="HTTP Request Rate"
          query="rate(http_requests_total[5m])"
          duration="1h"
        />
        <TimeSeriesChart
          title="Memory Usage"
          query="process_resident_memory_bytes"
          duration="1h"
        />
      </div>
    </div>
  );
};

// E2 Interface section
const E2InterfaceMetrics = () => {
  const { data, loading, error } = useMetrics('e2');

  const getValue = (metricData) => {
    if (!metricData?.result?.[0]) return 0;
    return parseFloat(metricData.result[0].value[1]);
  };

  return (
    <div className="e2-metrics">
      <h2>E2 Interface</h2>
      <div className="metrics-grid">
        <MetricCard
          title="Connected Nodes"
          value={getValue(data?.connectedNodes)}
          status={getValue(data?.connectedNodes) > 0 ? 'healthy' : 'warning'}
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Active Subscriptions"
          value={getValue(data?.activeSubscriptions)}
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Message Rate"
          value={getValue(data?.messageRate)}
          unit="msg/s"
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Indication Latency"
          value={getValue(data?.indicationLatency)}
          unit="seconds"
          status={getValue(data?.indicationLatency) < 0.01 ? 'healthy' : 'warning'}
          loading={loading}
          error={error}
        />
      </div>
      
      <div className="charts-grid">
        <TimeSeriesChart
          title="E2AP Message Rate"
          query="rate(e2ap_messages_total[5m])"
          duration="1h"
        />
        <TimeSeriesChart
          title="Indication Processing Latency"
          query="histogram_quantile(0.95, rate(e2_indication_processing_duration_seconds_bucket[5m]))"
          duration="1h"
        />
      </div>
    </div>
  );
};

// A1 Interface section
const A1InterfaceMetrics = () => {
  const { data, loading, error } = useMetrics('a1');

  const getValue = (metricData) => {
    if (!metricData?.result?.[0]) return 0;
    return parseFloat(metricData.result[0].value[1]);
  };

  return (
    <div className="a1-metrics">
      <h2>A1 Interface</h2>
      <div className="metrics-grid">
        <MetricCard
          title="Policy Types"
          value={getValue(data?.policyTypes)}
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Policy Instances"
          value={getValue(data?.policyInstances)}
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Request Rate"
          value={getValue(data?.requestRate)}
          unit="req/s"
          loading={loading}
          error={error}
        />
        <MetricCard
          title="Processing Latency"
          value={getValue(data?.processingLatency)}
          unit="seconds"
          loading={loading}
          error={error}
        />
      </div>
      
      <div className="charts-grid">
        <TimeSeriesChart
          title="A1 Policy Request Rate"
          query="rate(a1_policy_requests_total[5m])"
          duration="1h"
        />
        <TimeSeriesChart
          title="Policy Processing Latency"
          query="histogram_quantile(0.95, rate(a1_policy_processing_duration_seconds_bucket[5m]))"
          duration="1h"
        />
      </div>
    </div>
  );
};

// Main dashboard component
const MetricsDashboard = () => {
  const [activeTab, setActiveTab] = useState('platform');
  const [autoRefresh, setAutoRefresh] = useState(true);

  useEffect(() => {
    // Connect to real-time metrics when component mounts
    return () => {
      // Cleanup will be handled by the useRealTimeMetrics hook
    };
  }, []);

  const tabs = [
    { id: 'platform', label: 'Platform', component: PlatformOverview },
    { id: 'e2', label: 'E2 Interface', component: E2InterfaceMetrics },
    { id: 'a1', label: 'A1 Interface', component: A1InterfaceMetrics },
  ];

  const ActiveComponent = tabs.find(tab => tab.id === activeTab)?.component || PlatformOverview;

  return (
    <div className="metrics-dashboard">
      <div className="dashboard-header">
        <h1>O-RAN Platform Metrics</h1>
        <div className="dashboard-controls">
          <label className="auto-refresh-toggle">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            Auto Refresh
          </label>
        </div>
      </div>

      <div className="dashboard-tabs">
        {tabs.map(tab => (
          <button
            key={tab.id}
            className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="dashboard-content">
        <div className="main-content">
          <ActiveComponent />
        </div>
        <div className="sidebar">
          <AlertPanel />
        </div>
      </div>
    </div>
  );
};

export default MetricsDashboard;