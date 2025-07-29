/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import './ComponentDiscovery.css';

/**
 * Component discovery visualization
 * Shows auto-discovered O-RAN SC components with topology view
 */
const ComponentDiscovery = ({ 
  components = [], 
  loading = false, 
  error = null, 
  onRefresh = null 
}) => {
  const getComponentsByType = () => {
    const grouped = {
      'e2manager': [],
      'submgr': [],
      'appmgr': [],
      'rtmgr': [],
      'xapp': [],
      'other': []
    };

    components.forEach(component => {
      const type = component.type?.toLowerCase() || 'other';
      if (grouped[type]) {
        grouped[type].push(component);
      } else {
        grouped.other.push(component);
      }
    });

    return grouped;
  };

  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'running':
      case 'active':
      case 'connected':
      case 'healthy':
        return '●';
      case 'stopped':
      case 'inactive':
      case 'disconnected':
      case 'unhealthy':
        return '○';
      case 'starting':
      case 'deploying':
      case 'connecting':
        return '◐';
      case 'error':
      case 'failed':
        return '⚠';
      default:
        return '?';
    }
  };

  const getStatusClass = (status) => {
    switch (status?.toLowerCase()) {
      case 'running':
      case 'active':
      case 'connected':
      case 'healthy':
        return 'running';
      case 'stopped':
      case 'inactive':
      case 'disconnected':
      case 'unhealthy':
        return 'stopped';
      case 'starting':
      case 'deploying':
      case 'connecting':
        return 'starting';
      case 'error':
      case 'failed':
        return 'error';
      default:
        return 'unknown';
    }
  };

  const getComponentTypeLabel = (type) => {
    const labels = {
      'e2manager': 'E2 Manager',
      'submgr': 'Subscription Manager',
      'appmgr': 'App Manager',
      'rtmgr': 'Routing Manager',
      'xapp': 'xApps',
      'other': 'Other Components'
    };
    return labels[type] || type;
  };

  const getComponentTypeIcon = (type) => {
    const icons = {
      'e2manager': '📡',
      'submgr': '📋',
      'appmgr': '📦',
      'rtmgr': '🔀',
      'xapp': '⚡',
      'other': '🔧'
    };
    return icons[type] || '🔧';
  };

  if (loading) {
    return (
      <div className="component-discovery loading">
        <div className="discovery-header">
          <h3>Component Discovery</h3>
          <div className="loading-spinner">⟳</div>
        </div>
        <div className="discovery-body">
          <p>Discovering O-RAN SC components...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="component-discovery error">
        <div className="discovery-header">
          <h3>Component Discovery</h3>
          <div className="error-indicator">⚠</div>
        </div>
        <div className="discovery-body">
          <div className="error-message">
            <p>Failed to discover components</p>
            <p className="error-details">{error.message}</p>
            {onRefresh && (
              <button onClick={onRefresh} className="refresh-button">
                Retry Discovery
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  const componentsByType = getComponentsByType();
  const totalComponents = components.length;
  const runningComponents = components.filter(c => 
    ['running', 'active', 'connected', 'healthy'].includes(c.status?.toLowerCase())
  ).length;

  return (
    <div className="component-discovery">
      <div className="discovery-header">
        <h3>Component Discovery</h3>
        <div className="discovery-stats">
          <span className="stat-item">
            <span className="stat-value">{totalComponents}</span>
            <span className="stat-label">Total</span>
          </span>
          <span className="stat-item">
            <span className="stat-value running">{runningComponents}</span>
            <span className="stat-label">Running</span>
          </span>
        </div>
      </div>

      <div className="discovery-body">
        {totalComponents === 0 ? (
          <div className="no-components">
            <p>No O-RAN SC components discovered</p>
            <p className="help-text">
              Make sure O-RAN SC components are deployed and accessible
            </p>
          </div>
        ) : (
          <div className="topology-view">
            {Object.entries(componentsByType).map(([type, typeComponents]) => {
              if (typeComponents.length === 0) return null;
              
              return (
                <div key={type} className="component-group">
                  <div className="group-header">
                    <span className="group-icon">{getComponentTypeIcon(type)}</span>
                    <span className="group-title">{getComponentTypeLabel(type)}</span>
                    <span className="group-count">({typeComponents.length})</span>
                  </div>
                  
                  <div className="components-grid">
                    {typeComponents.map((component, index) => (
                      <div 
                        key={component.id || component.name || index} 
                        className={`component-node ${getStatusClass(component.status)}`}
                      >
                        <div className="node-header">
                          <span className={`status-indicator ${getStatusClass(component.status)}`}>
                            {getStatusIcon(component.status)}
                          </span>
                          <span className="node-name">
                            {component.name || component.id || `${type}-${index + 1}`}
                          </span>
                        </div>
                        
                        <div className="node-details">
                          <div className="detail-row">
                            <span className="detail-label">Status:</span>
                            <span className={`detail-value ${getStatusClass(component.status)}`}>
                              {component.status || 'Unknown'}
                            </span>
                          </div>
                          
                          {component.version && (
                            <div className="detail-row">
                              <span className="detail-label">Version:</span>
                              <span className="detail-value">{component.version}</span>
                            </div>
                          )}
                          
                          {component.endpoints && component.endpoints.length > 0 && (
                            <div className="detail-row">
                              <span className="detail-label">Endpoints:</span>
                              <span className="detail-value">{component.endpoints.length}</span>
                            </div>
                          )}
                        </div>
                        
                        {component.lastUpdated && (
                          <div className="node-timestamp">
                            Updated: {new Date(component.lastUpdated).toLocaleTimeString()}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {onRefresh && (
        <div className="discovery-footer">
          <button onClick={onRefresh} className="refresh-button">
            Refresh Discovery
          </button>
          <span className="last-discovery">
            Last discovery: {new Date().toLocaleTimeString()}
          </span>
        </div>
      )}
    </div>
  );
};

export default ComponentDiscovery;