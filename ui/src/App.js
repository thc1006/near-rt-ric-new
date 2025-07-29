import React, { useState, useEffect } from 'react';
import './App.css';
import ErrorBoundary from './components/ErrorBoundary';
import { ErrorDisplay, LoadingDisplay, ConnectionStatus } from './components/ErrorDisplay';
import { 
  useComponents, 
  useE2Nodes, 
  useSubscriptions, 
  useXApps, 
  useHealth,
  useWebSocket 
} from './hooks/useAPI';

function App() {
  // API hooks for data fetching
  const { data: componentsData, loading: componentsLoading, error: componentsError, refetch: refetchComponents } = useComponents();
  const { data: e2NodesData, loading: e2NodesLoading, error: e2NodesError, refetch: refetchE2Nodes } = useE2Nodes();
  const { data: subscriptionsData, loading: subscriptionsLoading, error: subscriptionsError, refetch: refetchSubscriptions } = useSubscriptions();
  const { data: xAppsData, loading: xAppsLoading, error: xAppsError, refetch: refetchXApps } = useXApps();
  const { error: healthError } = useHealth();
  
  // WebSocket for real-time updates
  const { connected: wsConnected, messages: wsMessages, error: wsError } = useWebSocket();

  // Local state for processed data
  const [networkFunctions, setNetworkFunctions] = useState([]);
  const [kpis, setKpis] = useState({});
  const [alarms, setAlarms] = useState([]);
  const [logs, setLogs] = useState([]);

  // Process components data into network functions
  useEffect(() => {
    if (componentsData?.components) {
      const functions = componentsData.components.map(component => ({
        id: component.name || component.id,
        name: component.name || component.id,
        type: component.type || 'Unknown',
        status: component.status || 'unknown'
      }));
      setNetworkFunctions(functions);
    }
  }, [componentsData]);

  // Process E2 nodes and xApps data into KPIs
  useEffect(() => {
    const newKpis = {};
    
    if (e2NodesData?.e2nodes) {
      e2NodesData.e2nodes.forEach(node => {
        newKpis[node.id] = {
          status: node.status,
          type: node.type,
          plmnId: node.plmnId
        };
      });
    }
    
    if (xAppsData?.xapps) {
      xAppsData.xapps.forEach(xapp => {
        newKpis[xapp.name] = {
          status: xapp.status,
          instances: xapp.instances,
          version: xapp.version
        };
      });
    }
    
    if (subscriptionsData?.subscriptions) {
      newKpis.subscriptions = {
        total: subscriptionsData.count || subscriptionsData.subscriptions.length,
        active: subscriptionsData.subscriptions.filter(sub => sub.status === 'active').length
      };
    }
    
    setKpis(newKpis);
  }, [e2NodesData, xAppsData, subscriptionsData]);

  // Process health data into alarms
  useEffect(() => {
    const newAlarms = [];
    
    if (healthError) {
      newAlarms.push({
        id: 'health-error',
        severity: 'critical',
        message: 'Dashboard API health check failed'
      });
    }
    
    if (componentsError) {
      newAlarms.push({
        id: 'components-error',
        severity: 'warning',
        message: 'Failed to discover components'
      });
    }
    
    if (e2NodesError) {
      newAlarms.push({
        id: 'e2nodes-error',
        severity: 'warning',
        message: 'Failed to fetch E2 nodes'
      });
    }
    
    if (!wsConnected) {
      newAlarms.push({
        id: 'websocket-error',
        severity: 'warning',
        message: 'Real-time updates unavailable'
      });
    }
    
    setAlarms(newAlarms);
  }, [healthError, componentsError, e2NodesError, wsConnected]);

  // Process WebSocket messages into logs
  useEffect(() => {
    if (wsMessages.length > 0) {
      const newLogs = wsMessages.map((msg, index) => ({
        id: `ws-${index}`,
        timestamp: new Date().toISOString(),
        service: 'dashboard-api',
        message: `${msg.type}: ${JSON.stringify(msg.data)}`
      }));
      setLogs(prev => [...prev, ...newLogs].slice(-50)); // Keep last 50 logs
    }
  }, [wsMessages]);

  // Auto-refresh data every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      refetchComponents();
      refetchE2Nodes();
      refetchSubscriptions();
      refetchXApps();
    }, 30000);

    return () => clearInterval(interval);
  }, [refetchComponents, refetchE2Nodes, refetchSubscriptions, refetchXApps]);

  return (
    <ErrorBoundary>
      <div className="App">
        <header className="App-header">
          <h1>O-RAN Interactive Operations Console</h1>
          <ConnectionStatus connected={wsConnected} error={wsError} />
        </header>
        <main>
          <div className="dashboard-container">
            <div className="panel network-functions">
              <h2>Network Functions</h2>
              {componentsLoading ? (
                <LoadingDisplay message="Discovering components..." />
              ) : componentsError ? (
                <ErrorDisplay error={componentsError} onRetry={refetchComponents} />
              ) : (
                <ul>
                  {networkFunctions.map(nf => (
                    <li key={nf.id}>
                      <strong>{nf.name}</strong> ({nf.type})
                      {nf.status && <span className={`status-${nf.status}`}> - {nf.status}</span>}
                    </li>
                  ))}
                  {networkFunctions.length === 0 && (
                    <li>No network functions discovered</li>
                  )}
                </ul>
              )}
            </div>
            
            <div className="panel kpis">
              <h2>Real-Time KPIs</h2>
              {(e2NodesLoading || xAppsLoading || subscriptionsLoading) ? (
                <LoadingDisplay message="Loading KPIs..." />
              ) : (e2NodesError || xAppsError || subscriptionsError) ? (
                <div>
                  {e2NodesError && <ErrorDisplay error={e2NodesError} onRetry={refetchE2Nodes} />}
                  {xAppsError && <ErrorDisplay error={xAppsError} onRetry={refetchXApps} />}
                  {subscriptionsError && <ErrorDisplay error={subscriptionsError} onRetry={refetchSubscriptions} />}
                </div>
              ) : (
                <div>
                  {Object.entries(kpis).map(([nf, data]) => (
                    <div key={nf}>
                      <h3>{nf}</h3>
                      <ul>
                        {Object.entries(data).map(([key, value]) => (
                          <li key={key}>
                            {key}: {typeof value === 'number' ? value.toFixed(2) : value}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                  {Object.keys(kpis).length === 0 && (
                    <p>No KPI data available</p>
                  )}
                </div>
              )}
            </div>
            
            <div className="panel alarms">
              <h2>Alarms</h2>
              <ul>
                {alarms.map(alarm => (
                  <li key={alarm.id} className={`alarm-${alarm.severity}`}>
                    <strong>{alarm.severity.toUpperCase()}:</strong> {alarm.message}
                  </li>
                ))}
                {alarms.length === 0 && (
                  <li className="alarm-info">No active alarms</li>
                )}
              </ul>
            </div>
            
            <div className="panel logs">
              <h2>Real-Time Logs</h2>
              <div className="logs-container">
                {logs.length > 0 ? (
                  <pre>
                    {logs.map(log => 
                      `[${new Date(log.timestamp).toLocaleTimeString()}] [${log.service}] ${log.message}\n`
                    ).join('')}
                  </pre>
                ) : (
                  <p>No real-time logs available</p>
                )}
              </div>
            </div>
          </div>
        </main>
      </div>
    </ErrorBoundary>
  );
}

export default App;