/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState } from 'react';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import './XAppList.css';

/**
 * xApp list component with lifecycle controls
 * Displays all deployed xApps with status and provides quick action buttons
 */
const XAppList = ({ 
  xApps = [], 
  loading = false, 
  error = null, 
  onRefresh = null,
  onSelectXApp = null,
  onUndeploy = null,
  onRestart = null,
  onScale = null,
  operationLoading = {},
  operationError = null
}) => {
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState('asc');
  const [filterStatus, setFilterStatus] = useState('all');
  const [scaleDialogs, setScaleDialogs] = useState({});

  // Sort xApps
  const sortedXApps = [...xApps].sort((a, b) => {
    let aValue = a[sortBy] || '';
    let bValue = b[sortBy] || '';
    
    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase();
      bValue = bValue.toLowerCase();
    }
    
    if (sortOrder === 'asc') {
      return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
    } else {
      return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
    }
  });

  // Filter xApps by status
  const filteredXApps = sortedXApps.filter(xapp => {
    if (filterStatus === 'all') return true;
    return xapp.status?.toLowerCase() === filterStatus.toLowerCase();
  });

  // Handle sort change
  const handleSort = (field) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('asc');
    }
  };

  // Handle scale dialog
  const handleScaleDialog = (xappName, show, instances = 1) => {
    if (show) {
      setScaleDialogs(prev => ({ ...prev, [xappName]: instances }));
    } else {
      setScaleDialogs(prev => {
        const newDialogs = { ...prev };
        delete newDialogs[xappName];
        return newDialogs;
      });
    }
  };

  // Handle scale confirmation
  const handleScaleConfirm = (xappName) => {
    const instances = scaleDialogs[xappName];
    if (instances && onScale) {
      onScale(xappName, instances);
      handleScaleDialog(xappName, false);
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

  // Get unique statuses for filter
  const uniqueStatuses = [...new Set(xApps.map(xapp => xapp.status?.toLowerCase()).filter(Boolean))];

  if (loading && xApps.length === 0) {
    return <LoadingDisplay message="Loading xApps..." />;
  }

  if (error && xApps.length === 0) {
    return <ErrorDisplay error={error} onRetry={onRefresh} />;
  }

  return (
    <div className="xapp-list">
      <div className="list-header">
        <div className="list-controls">
          <div className="filter-controls">
            <label htmlFor="status-filter">Filter by Status:</label>
            <select
              id="status-filter"
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <option value="all">All ({xApps.length})</option>
              {uniqueStatuses.map(status => (
                <option key={status} value={status}>
                  {status.charAt(0).toUpperCase() + status.slice(1)} 
                  ({xApps.filter(x => x.status?.toLowerCase() === status).length})
                </option>
              ))}
            </select>
          </div>
          
          <div className="sort-controls">
            <label>Sort by:</label>
            <button
              className={`sort-btn ${sortBy === 'name' ? 'active' : ''}`}
              onClick={() => handleSort('name')}
            >
              Name {sortBy === 'name' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              className={`sort-btn ${sortBy === 'status' ? 'active' : ''}`}
              onClick={() => handleSort('status')}
            >
              Status {sortBy === 'status' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              className={`sort-btn ${sortBy === 'version' ? 'active' : ''}`}
              onClick={() => handleSort('version')}
            >
              Version {sortBy === 'version' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
          </div>
        </div>

        {operationError && (
          <div className="operation-error">
            <ErrorDisplay error={operationError} />
          </div>
        )}
      </div>

      {filteredXApps.length === 0 ? (
        <div className="no-xapps">
          {xApps.length === 0 ? (
            <div>
              <h3>No xApps Deployed</h3>
              <p>Deploy your first xApp to get started with RAN intelligent control.</p>
            </div>
          ) : (
            <div>
              <h3>No xApps Match Filter</h3>
              <p>Try adjusting your filter criteria to see more xApps.</p>
            </div>
          )}
        </div>
      ) : (
        <div className="xapp-grid">
          {filteredXApps.map((xapp, index) => (
            <div key={xapp.name || index} className="xapp-card">
              <div className="xapp-card-header">
                <div className="xapp-info">
                  <h4 className="xapp-name">{xapp.name || `xApp-${index + 1}`}</h4>
                  <span className="xapp-version">{xapp.version || 'Unknown'}</span>
                </div>
                <span className={`xapp-status ${getStatusClass(xapp.status)}`}>
                  {xapp.status || 'Unknown'}
                </span>
              </div>

              <div className="xapp-card-body">
                <div className="xapp-metrics">
                  {xapp.instances && (
                    <div className="metric">
                      <span className="metric-label">Instances:</span>
                      <span className="metric-value">{xapp.instances}</span>
                    </div>
                  )}
                  {xapp.resources && (
                    <div className="metric">
                      <span className="metric-label">Resources:</span>
                      <span className="metric-value">
                        {xapp.resources.cpu && `${xapp.resources.cpu} CPU`}
                        {xapp.resources.cpu && xapp.resources.memory && ', '}
                        {xapp.resources.memory && `${xapp.resources.memory} Memory`}
                      </span>
                    </div>
                  )}
                  {xapp.namespace && (
                    <div className="metric">
                      <span className="metric-label">Namespace:</span>
                      <span className="metric-value">{xapp.namespace}</span>
                    </div>
                  )}
                  {xapp.deployedAt && (
                    <div className="metric">
                      <span className="metric-label">Deployed:</span>
                      <span className="metric-value">
                        {new Date(xapp.deployedAt).toLocaleDateString()}
                      </span>
                    </div>
                  )}
                </div>

                {xapp.subscriptions && xapp.subscriptions.length > 0 && (
                  <div className="xapp-subscriptions">
                    <span className="subscriptions-label">
                      Subscriptions: {xapp.subscriptions.length}
                    </span>
                  </div>
                )}
              </div>

              <div className="xapp-card-actions">
                <button
                  className="btn btn-sm btn-primary"
                  onClick={() => onSelectXApp && onSelectXApp(xapp)}
                >
                  Details
                </button>

                {xapp.status?.toLowerCase() === 'running' && (
                  <>
                    <button
                      className="btn btn-sm btn-warning"
                      onClick={() => onRestart && onRestart(xapp)}
                      disabled={operationLoading[xapp.name] === 'restarting'}
                    >
                      {operationLoading[xapp.name] === 'restarting' ? 'Restarting...' : 'Restart'}
                    </button>

                    <button
                      className="btn btn-sm btn-secondary"
                      onClick={() => handleScaleDialog(xapp.name, true, xapp.instances || 1)}
                      disabled={operationLoading[xapp.name] === 'scaling'}
                    >
                      {operationLoading[xapp.name] === 'scaling' ? 'Scaling...' : 'Scale'}
                    </button>
                  </>
                )}

                <button
                  className="btn btn-sm btn-danger"
                  onClick={() => onUndeploy && onUndeploy(xapp.name)}
                  disabled={operationLoading[xapp.name] === 'undeploying'}
                >
                  {operationLoading[xapp.name] === 'undeploying' ? 'Undeploying...' : 'Undeploy'}
                </button>
              </div>

              {/* Scale Dialog */}
              {scaleDialogs[xapp.name] !== undefined && (
                <div className="scale-dialog">
                  <div className="scale-dialog-content">
                    <h5>Scale {xapp.name}</h5>
                    <div className="scale-input">
                      <label htmlFor={`instances-${xapp.name}`}>Instances:</label>
                      <input
                        type="number"
                        id={`instances-${xapp.name}`}
                        min="1"
                        max="10"
                        value={scaleDialogs[xapp.name]}
                        onChange={(e) => handleScaleDialog(xapp.name, true, parseInt(e.target.value))}
                      />
                    </div>
                    <div className="scale-actions">
                      <button
                        className="btn btn-sm btn-secondary"
                        onClick={() => handleScaleDialog(xapp.name, false)}
                      >
                        Cancel
                      </button>
                      <button
                        className="btn btn-sm btn-primary"
                        onClick={() => handleScaleConfirm(xapp.name)}
                      >
                        Scale
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default XAppList;