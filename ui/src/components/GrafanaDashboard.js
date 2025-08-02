import React, { useState, useEffect, useRef } from 'react';
import './GrafanaDashboard.css';

const GrafanaDashboard = ({ 
  dashboardId, 
  panelId = null, 
  height = '400px',
  timeRange = 'now-1h',
  refreshInterval = '30s',
  theme = 'light',
  showControls = true,
  autoRefresh = true 
}) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [grafanaUrl, setGrafanaUrl] = useState('');
  const iframeRef = useRef(null);

  useEffect(() => {
    // Get Grafana URL from environment or API
    const baseUrl = process.env.REACT_APP_GRAFANA_URL || 'http://localhost:3000';
    setGrafanaUrl(baseUrl);
  }, []);

  useEffect(() => {
    if (grafanaUrl && dashboardId) {
      setLoading(false);
    }
  }, [grafanaUrl, dashboardId]);

  const buildGrafanaUrl = () => {
    if (!grafanaUrl || !dashboardId) return '';

    let url = `${grafanaUrl}/d/${dashboardId}`;
    
    // Add panel-specific parameters if panelId is provided
    if (panelId) {
      url += `?viewPanel=${panelId}`;
    } else {
      url += '?';
    }

    // Add common parameters
    const params = new URLSearchParams();
    params.append('orgId', '1');
    params.append('from', timeRange.startsWith('now-') ? timeRange : 'now-1h');
    params.append('to', 'now');
    params.append('theme', theme);
    params.append('kiosk', showControls ? 'false' : 'true');
    
    if (autoRefresh) {
      params.append('refresh', refreshInterval);
    }

    return url + (url.includes('?') ? '&' : '?') + params.toString();
  };

  const handleIframeLoad = () => {
    setLoading(false);
    setError(null);
  };

  const handleIframeError = () => {
    setLoading(false);
    setError('Failed to load Grafana dashboard');
  };

  const refreshDashboard = () => {
    if (iframeRef.current) {
      setLoading(true);
      iframeRef.current.src = buildGrafanaUrl();
    }
  };

  if (!grafanaUrl || !dashboardId) {
    return (
      <div className="grafana-dashboard error">
        <div className="error-message">
          Grafana configuration missing. Please check your environment settings.
        </div>
      </div>
    );
  }

  return (
    <div className="grafana-dashboard" style={{ height }}>
      {showControls && (
        <div className="grafana-controls">
          <button 
            className="refresh-button"
            onClick={refreshDashboard}
            disabled={loading}
          >
            🔄 Refresh
          </button>
          <span className="dashboard-info">
            Dashboard: {dashboardId} {panelId && `| Panel: ${panelId}`}
          </span>
        </div>
      )}
      
      {loading && (
        <div className="loading-overlay">
          <div className="loading-spinner"></div>
          <span>Loading Grafana dashboard...</span>
        </div>
      )}
      
      {error && (
        <div className="error-overlay">
          <div className="error-message">{error}</div>
          <button onClick={refreshDashboard}>Retry</button>
        </div>
      )}
      
      <iframe
        ref={iframeRef}
        src={buildGrafanaUrl()}
        width="100%"
        height="100%"
        frameBorder="0"
        onLoad={handleIframeLoad}
        onError={handleIframeError}
        style={{ display: loading || error ? 'none' : 'block' }}
        title={`Grafana Dashboard ${dashboardId}`}
      />
    </div>
  );
};

// Pre-configured dashboard components
export const PlatformOverviewDashboard = (props) => (
  <GrafanaDashboard
    dashboardId="oran-sc-platform-overview"
    {...props}
  />
);

export const E2InterfaceDashboard = (props) => (
  <GrafanaDashboard
    dashboardId="e2-interface-monitoring"
    {...props}
  />
);

export const A1InterfaceDashboard = (props) => (
  <GrafanaDashboard
    dashboardId="a1-interface-monitoring"
    {...props}
  />
);

export const O1InterfaceDashboard = (props) => (
  <GrafanaDashboard
    dashboardId="o1-interface-monitoring"
    {...props}
  />
);

export const SecurityDashboard = (props) => (
  <GrafanaDashboard
    dashboardId="security-monitoring"
    {...props}
  />
);

// Dashboard grid component for multiple dashboards
export const GrafanaDashboardGrid = ({ dashboards, columns = 2, height = '400px' }) => {
  return (
    <div 
      className="grafana-dashboard-grid"
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, 1fr)`,
        gap: '20px',
        height
      }}
    >
      {dashboards.map((dashboard, index) => (
        <GrafanaDashboard
          key={index}
          {...dashboard}
          height="100%"
          showControls={false}
        />
      ))}
    </div>
  );
};

export default GrafanaDashboard;