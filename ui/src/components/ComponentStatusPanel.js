/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import './ComponentStatusPanel.css';

/**
 * Component status display for O-RAN SC components
 * Shows real-time metrics, connection status, and health information
 */

const ComponentStatusPanel = ({ 
  component, 
  loading = false, 
  error = null, 
  onRefresh = null,
  children 
}) => {
  if (loading) {
    return (
      <div className="component-status-panel loading">
        <div className="component-header">
          <h3>{component?.name || 'Loading...'}</h3>
          <div className="loading-spinner">⟳</div>
        </div>
        <div className="component-body">
          <p>Loading component status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="component-status-panel error">
        <div className="component-header">
          <h3>{component?.name || 'Component'}</h3>
          <div className="status-indicator error">⚠</div>
        </div>
        <div className="component-body">
          <div className="error-message">
            <p>Failed to load component status</p>
            <p className="error-details">{error.message}</p>
            {onRefresh && (
              <button onClick={onRefresh} className="refresh-button">
                Retry
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  if (!component) {
    return (
      <div className="component-status-panel not-found">
        <div className="component-header">
          <h3>Component Not Found</h3>
          <div className="status-indicator unknown">?</div>
        </div>
        <div className="component-body">
          <p>Component not discovered or not deployed</p>
        </div>
      </div>
    );
  }

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

  const getStatusIcon = (status) => {
    switch (getStatusClass(status)) {
      case 'running':
        return '●';
      case 'stopped':
        return '○';
      case 'starting':
        return '◐';
      case 'error':
        return '⚠';
      default:
        return '?';
    }
  };

  return (
    <div className={`component-status-panel ${getStatusClass(component.status)}`}>
      <div className="component-header">
        <div className="component-title">
          <h3>{component.name}</h3>
          <span className="component-type">{component.type}</span>
        </div>
        <div className={`status-indicator ${getStatusClass(component.status)}`}>
          {getStatusIcon(component.status)}
        </div>
      </div>
      
      <div className="component-body">
        <div className="status-info">
          <div className="status-item">
            <span className="label">Status:</span>
            <span className={`value status-${getStatusClass(component.status)}`}>
              {component.status || 'Unknown'}
            </span>
          </div>
          
          {component.version && (
            <div className="status-item">
              <span className="label">Version:</span>
              <span className="value">{component.version}</span>
            </div>
          )}
          
          {component.endpoints && component.endpoints.length > 0 && (
            <div className="status-item">
              <span className="label">Endpoints:</span>
              <div className="endpoints-list">
                {component.endpoints.map((endpoint, index) => (
                  <div key={index} className="endpoint">
                    <span className="endpoint-name">{endpoint.name}:</span>
                    <span className="endpoint-url">{endpoint.url}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          
          {component.lastUpdated && (
            <div className="status-item">
              <span className="label">Last Updated:</span>
              <span className="value">
                {new Date(component.lastUpdated).toLocaleString()}
              </span>
            </div>
          )}
        </div>
        
        {children}
      </div>
      
      {onRefresh && (
        <div className="component-footer">
          <button onClick={onRefresh} className="refresh-button">
            Refresh
          </button>
        </div>
      )}
    </div>
  );
};

export default ComponentStatusPanel;