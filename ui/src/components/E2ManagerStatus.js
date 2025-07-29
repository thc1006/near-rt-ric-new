/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import ComponentStatusPanel from './ComponentStatusPanel';
import './E2ManagerStatus.css';

/**
 * E2 Manager status display component
 * Shows E2 interface status, connected nodes, and metrics
 */
const E2ManagerStatus = ({ 
  component, 
  e2Nodes = [], 
  loading = false, 
  error = null, 
  onRefresh = null 
}) => {
  const getE2NodeStats = () => {
    if (!e2Nodes || e2Nodes.length === 0) {
      return {
        total: 0,
        connected: 0,
        disconnected: 0,
        setupFailed: 0
      };
    }

    return e2Nodes.reduce((stats, node) => {
      stats.total++;
      switch (node.connectionStatus?.toLowerCase()) {
        case 'connected':
          stats.connected++;
          break;
        case 'disconnected':
          stats.disconnected++;
          break;
        case 'setup_failed':
          stats.setupFailed++;
          break;
        default:
          break;
      }
      return stats;
    }, { total: 0, connected: 0, disconnected: 0, setupFailed: 0 });
  };

  const e2Stats = getE2NodeStats();

  return (
    <ComponentStatusPanel
      component={component}
      loading={loading}
      error={error}
      onRefresh={onRefresh}
    >
      <div className="e2-manager-details">
        <div className="metrics-section">
          <h4>E2 Interface Metrics</h4>
          <div className="metrics-grid">
            <div className="metric-item">
              <span className="metric-label">Total E2 Nodes:</span>
              <span className="metric-value">{e2Stats.total}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Connected:</span>
              <span className="metric-value connected">{e2Stats.connected}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Disconnected:</span>
              <span className="metric-value disconnected">{e2Stats.disconnected}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Setup Failed:</span>
              <span className="metric-value failed">{e2Stats.setupFailed}</span>
            </div>
          </div>
        </div>

        {component?.metrics && (
          <div className="performance-metrics">
            <h4>Performance Metrics</h4>
            <div className="metrics-grid">
              {component.metrics.messagesProcessed && (
                <div className="metric-item">
                  <span className="metric-label">Messages Processed:</span>
                  <span className="metric-value">{component.metrics.messagesProcessed.toLocaleString()}</span>
                </div>
              )}
              {component.metrics.averageResponseTime && (
                <div className="metric-item">
                  <span className="metric-label">Avg Response Time:</span>
                  <span className="metric-value">{component.metrics.averageResponseTime}ms</span>
                </div>
              )}
              {component.metrics.errorRate && (
                <div className="metric-item">
                  <span className="metric-label">Error Rate:</span>
                  <span className="metric-value">{(component.metrics.errorRate * 100).toFixed(2)}%</span>
                </div>
              )}
              {component.metrics.uptime && (
                <div className="metric-item">
                  <span className="metric-label">Uptime:</span>
                  <span className="metric-value">{component.metrics.uptime}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {e2Nodes && e2Nodes.length > 0 && (
          <div className="e2-nodes-section">
            <h4>Connected E2 Nodes</h4>
            <div className="e2-nodes-list">
              {e2Nodes.slice(0, 5).map((node, index) => (
                <div key={node.nodeId || index} className="e2-node-item">
                  <div className="node-info">
                    <span className="node-id">{node.nodeId || `Node ${index + 1}`}</span>
                    <span className="node-type">{node.nodeType || 'Unknown'}</span>
                  </div>
                  <div className={`node-status ${node.connectionStatus?.toLowerCase() || 'unknown'}`}>
                    {node.connectionStatus || 'Unknown'}
                  </div>
                </div>
              ))}
              {e2Nodes.length > 5 && (
                <div className="more-nodes">
                  +{e2Nodes.length - 5} more nodes
                </div>
              )}
            </div>
          </div>
        )}

        {component?.supportedFunctions && component.supportedFunctions.length > 0 && (
          <div className="functions-section">
            <h4>Supported RAN Functions</h4>
            <div className="functions-list">
              {component.supportedFunctions.map((func, index) => (
                <span key={index} className="function-tag">
                  {func.name || func}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </ComponentStatusPanel>
  );
};

export default E2ManagerStatus;