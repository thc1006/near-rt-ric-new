import React, { useState, useEffect } from 'react';
import './App.css';
import ErrorBoundary from './components/ErrorBoundary';
import { ErrorDisplay, LoadingDisplay, ConnectionStatus } from './components/ErrorDisplay';
import ComponentDiscovery from './components/ComponentDiscovery';
import E2ManagerStatus from './components/E2ManagerStatus';
import SubscriptionManagerStatus from './components/SubscriptionManagerStatus';
import AppManagerStatus from './components/AppManagerStatus';
import XAppManagement from './components/XAppManagement';
import E2NodesPanel from './components/E2NodesPanel';
import SubscriptionPanel from './components/SubscriptionPanel';
import ServiceModelPanel from './components/ServiceModelPanel';
import A1PolicyManagement from './components/A1PolicyManagement';
import O1Management from './components/O1Management';
import MetricsDashboard from './components/MetricsDashboard';
import { 
  useComponents, 
  useE2Nodes, 
  useSubscriptions, 
  useXApps, 
  useHealth,
  useWebSocket 
} from './hooks/useAPI';

function App() {
  // Navigation state
  const [activeView, setActiveView] = useState('dashboard');
  
  // API hooks for data fetching
  const { data: componentsData, loading: componentsLoading, error: componentsError, refetch: refetchComponents } = useComponents();
  const { data: e2NodesData, loading: e2NodesLoading, error: e2NodesError, refetch: refetchE2Nodes } = useE2Nodes();
  const { data: subscriptionsData, loading: subscriptionsLoading, error: subscriptionsError, refetch: refetchSubscriptions } = useSubscriptions();
  const { data: xAppsData, loading: xAppsLoading, error: xAppsError, refetch: refetchXApps } = useXApps();
  const { error: healthError } = useHealth();
  
  // WebSocket for real-time updates
  const { 
    connected: wsConnected, 
    connecting: wsConnecting,
    error: wsError,
    connectionState: wsConnectionState,
    lastMessage: wsLastMessage,
    reconnectAttempts: wsReconnectAttempts,
    connect: wsConnect
  } = useWebSocket();

  // Local state for processed data
  const [kpis, setKpis] = useState({});
  const [alarms, setAlarms] = useState([]);
  const [logs, setLogs] = useState([]);



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

  // Process WebSocket messages for real-time updates
  useEffect(() => {
    if (wsLastMessage) {
      const message = wsLastMessage;
      const timestamp = new Date().toISOString();
      
      // Process different types of real-time messages
      switch (message.type) {
        case 'component_status_update':
        case 'component_discovered':
        case 'component_removed':
          // Trigger component data refresh
          refetchComponents();
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'component-discovery',
            message: `${message.type}: ${message.data?.name || 'unknown'} - ${message.data?.status || 'unknown'}`
          }].slice(-50));
          break;
          
        case 'e2node_connected':
        case 'e2node_disconnected':
          // Trigger E2 nodes data refresh
          refetchE2Nodes();
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'e2-manager',
            message: `${message.type}: ${message.data?.nodeId || 'unknown'}`
          }].slice(-50));
          break;
          
        case 'subscription_created':
        case 'subscription_deleted':
        case 'subscription_failed':
          // Trigger subscriptions data refresh
          refetchSubscriptions();
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'subscription-manager',
            message: `${message.type}: ${message.data?.subscriptionId || message.data?.id || 'unknown'}`
          }].slice(-50));
          break;
          
        case 'xapp_deployed':
        case 'xapp_undeployed':
        case 'xapp_status_changed':
          // Trigger xApps data refresh
          refetchXApps();
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'app-manager',
            message: `${message.type}: ${message.data?.name || 'unknown'} - ${message.data?.status || ''}`
          }].slice(-50));
          break;
          
        case 'alarm_raised':
        case 'alarm_cleared':
          // Add alarm to the alarms list
          const alarmSeverity = message.data?.severity || 'warning';
          const alarmMessage = message.data?.message || 'Unknown alarm';
          const alarmId = message.data?.id || `alarm-${Date.now()}`;
          
          if (message.type === 'alarm_raised') {
            setAlarms(prev => [...prev, {
              id: alarmId,
              severity: alarmSeverity,
              message: alarmMessage,
              timestamp
            }]);
          } else {
            setAlarms(prev => prev.filter(alarm => alarm.id !== alarmId));
          }
          
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'alarm-manager',
            message: `${message.type}: ${alarmMessage}`
          }].slice(-50));
          break;
          
        case 'system_event':
          // Log system events
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'system',
            message: `${message.type}: ${message.data?.message || JSON.stringify(message.data)}`
          }].slice(-50));
          break;
          
        default:
          // Log unknown message types
          setLogs(prev => [...prev, {
            id: `ws-${Date.now()}`,
            timestamp,
            service: 'websocket',
            message: `${message.type}: ${JSON.stringify(message.data)}`
          }].slice(-50));
      }
    }
  }, [wsLastMessage, refetchComponents, refetchE2Nodes, refetchSubscriptions, refetchXApps]);

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

  const renderActiveView = () => {
    switch (activeView) {
      case 'metrics':
        return <MetricsDashboard />;
      case 'components':
        return (
          <div className="panel full-width">
            <ComponentDiscovery
              components={componentsData?.components || []}
              loading={componentsLoading}
              error={componentsError}
              onRefresh={refetchComponents}
            />
          </div>
        );
      case 'e2nodes':
        return (
          <div className="panel full-width">
            <E2NodesPanel />
          </div>
        );
      case 'subscriptions':
        return (
          <div className="panel full-width">
            <SubscriptionPanel />
          </div>
        );
      case 'servicemodels':
        return (
          <div className="panel full-width">
            <ServiceModelPanel />
          </div>
        );
      case 'a1policies':
        return (
          <div className="panel full-width">
            <A1PolicyManagement />
          </div>
        );
      case 'o1management':
        return (
          <div className="panel full-width">
            <O1Management />
          </div>
        );
      case 'xapps':
        return (
          <div className="panel full-width">
            <XAppManagement
              xApps={xAppsData?.xapps || []}
              loading={xAppsLoading}
              error={xAppsError}
              onRefresh={refetchXApps}
            />
          </div>
        );
      default:
        return (
          <div className="dashboard-container">
            {/* Component Discovery Panel */}
            <div className="panel full-width">
              <ComponentDiscovery
                components={componentsData?.components || []}
                loading={componentsLoading}
                error={componentsError}
                onRefresh={refetchComponents}
              />
            </div>

            {/* O-RAN SC Component Status Panels */}
            <div className="component-status-grid">
              <E2ManagerStatus
                component={componentsData?.components?.find(c => c.type?.toLowerCase() === 'e2manager')}
                e2Nodes={e2NodesData?.e2nodes || []}
                loading={componentsLoading || e2NodesLoading}
                error={componentsError || e2NodesError}
                onRefresh={() => {
                  refetchComponents();
                  refetchE2Nodes();
                }}
              />

              <SubscriptionManagerStatus
                component={componentsData?.components?.find(c => c.type?.toLowerCase() === 'submgr')}
                subscriptions={subscriptionsData?.subscriptions || []}
                loading={componentsLoading || subscriptionsLoading}
                error={componentsError || subscriptionsError}
                onRefresh={() => {
                  refetchComponents();
                  refetchSubscriptions();
                }}
              />

              <AppManagerStatus
                component={componentsData?.components?.find(c => c.type?.toLowerCase() === 'appmgr')}
                xApps={xAppsData?.xapps || []}
                loading={componentsLoading || xAppsLoading}
                error={componentsError || xAppsError}
                onRefresh={() => {
                  refetchComponents();
                  refetchXApps();
                }}
              />
            </div>
            
            {/* Legacy Panels for backward compatibility */}
            <div className="legacy-panels">
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
          </div>
        );
    }
  };

  return (
    <ErrorBoundary>
      <div className="App">
        <header className="App-header">
          <h1>O-RAN Interactive Operations Console</h1>
          <nav className="main-navigation">
            <button 
              className={activeView === 'dashboard' ? 'active' : ''}
              onClick={() => setActiveView('dashboard')}
            >
              Dashboard
            </button>
            <button 
              className={activeView === 'metrics' ? 'active' : ''}
              onClick={() => setActiveView('metrics')}
            >
              Metrics
            </button>
            <button 
              className={activeView === 'components' ? 'active' : ''}
              onClick={() => setActiveView('components')}
            >
              Components
            </button>
            <button 
              className={activeView === 'e2nodes' ? 'active' : ''}
              onClick={() => setActiveView('e2nodes')}
            >
              E2 Nodes
            </button>
            <button 
              className={activeView === 'subscriptions' ? 'active' : ''}
              onClick={() => setActiveView('subscriptions')}
            >
              Subscriptions
            </button>
            <button 
              className={activeView === 'a1policies' ? 'active' : ''}
              onClick={() => setActiveView('a1policies')}
            >
              A1 Policies
            </button>
            <button 
              className={activeView === 'o1management' ? 'active' : ''}
              onClick={() => setActiveView('o1management')}
            >
              O1 Management
            </button>
            <button 
              className={activeView === 'xapps' ? 'active' : ''}
              onClick={() => setActiveView('xapps')}
            >
              xApps
            </button>
          </nav>
          <ConnectionStatus 
            connected={wsConnected} 
            connecting={wsConnecting}
            error={wsError}
            connectionState={wsConnectionState}
            reconnectAttempts={wsReconnectAttempts}
            onReconnect={wsConnect}
          />
        </header>
        <main>
          {renderActiveView()}
        </main>
      </div>
    </ErrorBoundary>
  );
}

export default App;