/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import './XAppDetails.css';

/**
 * Detailed xApp view component
 * Shows comprehensive xApp information including status, logs, and resource usage
 */
const XAppDetails = ({ 
  xapp,
  onUndeploy = null,
  onRestart = null,
  onScale = null,
  operationLoading = null,
  operationError = null,
  onBack = null
}) => {
  const [activeTab, setActiveTab] = useState('overview');
  const [logs, setLogs] = useState([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsError, setLogsError] = useState(null);
  const [scaleInstances, setScaleInstances] = useState(xapp.instances || 1);
  const [showScaleDialog, setShowScaleDialog] = useState(false);

  // Fetch xApp logs (mock implementation)
  const fetchLogs = async () => {
    try {
      setLogsLoading(true);
      setLogsError(null);
      
      // Mock logs - in real implementation, this would call the API
      const mockLogs = [
        {
          timestamp: new Date(Date.now() - 300000).toISOString(),
          level: 'INFO',
          message: 'xApp started successfully',
          source: 'main'
        },
        {
          timestamp: new Date(Date.now() - 240000).toISOString(),
          level: 'INFO',
          message: 'Connected to E2 Manager',
          source: 'e2-client'
        },
        {
          timestamp: new Date(Date.now() - 180000).toISOString(),
          level: 'INFO',
          message: 'Subscription created for E2 node: gnb-001',
          source: 'subscription-manager'
        },
        {
          timestamp: new Date(Date.now() - 120000).toISOString(),
          level: 'DEBUG',
          message: 'Processing RIC indication message',
          source: 'message-handler'
        },
        {
          timestamp: new Date(Date.now() - 60000).toISOString(),
          level: 'INFO',
          message: 'Health check passed',
          source: 'health-monitor'
        }
      ];
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setLogs(mockLogs);
    } catch (err) {
      console.error('Failed to fetch logs:', err);
      setLogsError(err);
    } finally {
      setLogsLoading(false);
    }
  };

  // Load logs when component mounts or tab changes to logs
  useEffect(() => {
    if (activeTab === 'logs') {
      fetchLogs();
    }
  }, [activeTab]);

  // Handle scale confirmation
  const handleScaleConfirm = () => {
    if (onScale) {
      onScale(xapp.name, scaleInstances);
      setShowScaleDialog(false);
    }
  };

  // Get status color class
  const getStatusClass = (status) => {
    switch (status?.toLowerCase()) {
      case 'running':
      case 'deployed':
      case 'active':
        return 'status-running';
      case 'stopped':
      case 'undeployed':
      case 'inactive':
        return 'status-stopped';
      case 'deploying':
      case 'starting':
        return 'status-deploying';
      case 'failed':
      case 'error':
        return 'status-failed';
      default:
        return 'status-unknown';
    }
  };

  // Get log level class
  const getLogLevelClass = (level) => {
    switch (level?.toLowerCase()) {
      case 'error':
        return 'log-error';
      case 'warn':
      case 'warning':
        return 'log-warning';
      case 'info':
        return 'log-info';
      case 'debug':
        return 'log-debug';
      default:
        return 'log-default';
    }
  };

  const renderOverviewTab = () => (
    <div className="tab-content overview-tab">
      <div className="overview-sections">
        {/* Basic Information */}
        <div className="info-section">
          <h4>Basic Information</h4>
          <div className="info-grid">
            <div className="info-item">
              <span className="info-label">Name:</span>
              <span className="info-value">{xapp.name}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Version:</span>
              <span className="info-value">{xapp.version || 'Unknown'}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Status:</span>
              <span className={`info-value ${getStatusClass(xapp.status)}`}>
                {xapp.status || 'Unknown'}
              </span>
            </div>
            <div className="info-item">
              <span className="info-label">Namespace:</span>
              <span className="info-value">{xapp.namespace || 'ricxapp'}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Instances:</span>
              <span className="info-value">{xapp.instances || 1}</span>
            </div>
            {xapp.deployedAt && (
              <div className="info-item">
                <span className="info-label">Deployed:</span>
                <span className="info-value">
                  {new Date(xapp.deployedAt).toLocaleString()}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Resource Usage */}
        {xapp.resources && (
          <div className="info-section">
            <h4>Resource Allocation</h4>
            <div className="resource-grid">
              {xapp.resources.cpu && (
                <div className="resource-item">
                  <span className="resource-label">CPU:</span>
                  <span className="resource-value">{xapp.resources.cpu}</span>
                </div>
              )}
              {xapp.resources.memory && (
                <div className="resource-item">
                  <span className="resource-label">Memory:</span>
                  <span className="resource-value">{xapp.resources.memory}</span>
                </div>
              )}
              {xapp.resources.storage && (
                <div className="resource-item">
                  <span className="resource-label">Storage:</span>
                  <span className="resource-value">{xapp.resources.storage}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Configuration */}
        {xapp.configuration && Object.keys(xapp.configuration).length > 0 && (
          <div className="info-section">
            <h4>Configuration</h4>
            <div className="config-display">
              <pre>{JSON.stringify(xapp.configuration, null, 2)}</pre>
            </div>
          </div>
        )}

        {/* Environment Variables */}
        {xapp.environment && Object.keys(xapp.environment).length > 0 && (
          <div className="info-section">
            <h4>Environment Variables</h4>
            <div className="env-grid">
              {Object.entries(xapp.environment).map(([key, value]) => (
                <div key={key} className="env-item">
                  <span className="env-key">{key}:</span>
                  <span className="env-value">{value}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Subscriptions */}
        {xapp.subscriptions && xapp.subscriptions.length > 0 && (
          <div className="info-section">
            <h4>E2 Subscriptions</h4>
            <div className="subscriptions-list">
              {xapp.subscriptions.map((sub, index) => (
                <div key={sub.id || index} className="subscription-item">
                  <div className="subscription-header">
                    <span className="subscription-id">ID: {sub.id || `sub-${index + 1}`}</span>
                    <span className={`subscription-status ${sub.status?.toLowerCase() || 'unknown'}`}>
                      {sub.status || 'Unknown'}
                    </span>
                  </div>
                  {sub.nodeId && (
                    <div className="subscription-detail">
                      <span className="detail-label">E2 Node:</span>
                      <span className="detail-value">{sub.nodeId}</span>
                    </div>
                  )}
                  {sub.eventType && (
                    <div className="subscription-detail">
                      <span className="detail-label">Event Type:</span>
                      <span className="detail-value">{sub.eventType}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );

  const renderLogsTab = () => (
    <div className="tab-content logs-tab">
      <div className="logs-header">
        <h4>Application Logs</h4>
        <button 
          className="btn btn-sm btn-secondary"
          onClick={fetchLogs}
          disabled={logsLoading}
        >
          {logsLoading ? 'Refreshing...' : 'Refresh Logs'}
        </button>
      </div>

      {logsLoading && logs.length === 0 ? (
        <LoadingDisplay message="Loading logs..." />
      ) : logsError ? (
        <ErrorDisplay error={logsError} onRetry={fetchLogs} />
      ) : logs.length === 0 ? (
        <div className="no-logs">
          <p>No logs available for this xApp.</p>
        </div>
      ) : (
        <div className="logs-container">
          {logs.map((log, index) => (
            <div key={index} className={`log-entry ${getLogLevelClass(log.level)}`}>
              <span className="log-timestamp">
                {new Date(log.timestamp).toLocaleTimeString()}
              </span>
              <span className="log-level">{log.level}</span>
              <span className="log-source">[{log.source}]</span>
              <span className="log-message">{log.message}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  const renderMetricsTab = () => (
    <div className="tab-content metrics-tab">
      <div className="metrics-header">
        <h4>Performance Metrics</h4>
        <p>Real-time performance metrics and resource usage</p>
      </div>

      <div className="metrics-sections">
        {/* Mock metrics - in real implementation, these would come from Prometheus */}
        <div className="metrics-section">
          <h5>Resource Usage</h5>
          <div className="metrics-grid">
            <div className="metric-card">
              <div className="metric-title">CPU Usage</div>
              <div className="metric-value">45%</div>
              <div className="metric-subtitle">of {xapp.resources?.cpu || '100m'}</div>
            </div>
            <div className="metric-card">
              <div className="metric-title">Memory Usage</div>
              <div className="metric-value">78MB</div>
              <div className="metric-subtitle">of {xapp.resources?.memory || '128Mi'}</div>
            </div>
            <div className="metric-card">
              <div className="metric-title">Network I/O</div>
              <div className="metric-value">1.2MB/s</div>
              <div className="metric-subtitle">in/out</div>
            </div>
          </div>
        </div>

        <div className="metrics-section">
          <h5>Application Metrics</h5>
          <div className="metrics-grid">
            <div className="metric-card">
              <div className="metric-title">Messages Processed</div>
              <div className="metric-value">1,247</div>
              <div className="metric-subtitle">last hour</div>
            </div>
            <div className="metric-card">
              <div className="metric-title">Error Rate</div>
              <div className="metric-value">0.1%</div>
              <div className="metric-subtitle">last hour</div>
            </div>
            <div className="metric-card">
              <div className="metric-title">Avg Response Time</div>
              <div className="metric-value">12ms</div>
              <div className="metric-subtitle">last hour</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div className="xapp-details">
      <div className="details-header">
        <div className="header-info">
          <button className="back-button" onClick={onBack}>
            ← Back to List
          </button>
          <div className="xapp-title">
            <h3>{xapp.name}</h3>
            <span className={`xapp-status ${getStatusClass(xapp.status)}`}>
              {xapp.status || 'Unknown'}
            </span>
          </div>
        </div>

        <div className="header-actions">
          {xapp.status?.toLowerCase() === 'running' && (
            <>
              <button
                className="btn btn-warning"
                onClick={() => onRestart && onRestart(xapp)}
                disabled={operationLoading === 'restarting'}
              >
                {operationLoading === 'restarting' ? 'Restarting...' : 'Restart'}
              </button>

              <button
                className="btn btn-secondary"
                onClick={() => setShowScaleDialog(true)}
                disabled={operationLoading === 'scaling'}
              >
                {operationLoading === 'scaling' ? 'Scaling...' : 'Scale'}
              </button>
            </>
          )}

          <button
            className="btn btn-danger"
            onClick={() => onUndeploy && onUndeploy(xapp.name)}
            disabled={operationLoading === 'undeploying'}
          >
            {operationLoading === 'undeploying' ? 'Undeploying...' : 'Undeploy'}
          </button>
        </div>
      </div>

      {operationError && (
        <div className="operation-error">
          <ErrorDisplay error={operationError} />
        </div>
      )}

      <div className="details-tabs">
        <button 
          className={`tab-button ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button 
          className={`tab-button ${activeTab === 'logs' ? 'active' : ''}`}
          onClick={() => setActiveTab('logs')}
        >
          Logs
        </button>
        <button 
          className={`tab-button ${activeTab === 'metrics' ? 'active' : ''}`}
          onClick={() => setActiveTab('metrics')}
        >
          Metrics
        </button>
      </div>

      <div className="details-content">
        {activeTab === 'overview' && renderOverviewTab()}
        {activeTab === 'logs' && renderLogsTab()}
        {activeTab === 'metrics' && renderMetricsTab()}
      </div>

      {/* Scale Dialog */}
      {showScaleDialog && (
        <div className="modal-overlay">
          <div className="modal-content">
            <div className="modal-header">
              <h4>Scale {xapp.name}</h4>
              <button 
                className="modal-close"
                onClick={() => setShowScaleDialog(false)}
              >
                ×
              </button>
            </div>
            <div className="modal-body">
              <div className="scale-form">
                <label htmlFor="scale-instances">Number of Instances:</label>
                <input
                  type="number"
                  id="scale-instances"
                  min="1"
                  max="10"
                  value={scaleInstances}
                  onChange={(e) => setScaleInstances(parseInt(e.target.value))}
                />
                <p className="scale-note">
                  Current instances: {xapp.instances || 1}
                </p>
              </div>
            </div>
            <div className="modal-actions">
              <button
                className="btn btn-secondary"
                onClick={() => setShowScaleDialog(false)}
              >
                Cancel
              </button>
              <button
                className="btn btn-primary"
                onClick={handleScaleConfirm}
                disabled={operationLoading === 'scaling'}
              >
                {operationLoading === 'scaling' ? 'Scaling...' : 'Scale'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default XAppDetails;