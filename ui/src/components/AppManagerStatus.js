/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import ComponentStatusPanel from './ComponentStatusPanel';
import './AppManagerStatus.css';

/**
 * App Manager status display component
 * Shows xApp deployment status, resource usage, and lifecycle management
 */
const AppManagerStatus = ({ 
  component, 
  xApps = [], 
  loading = false, 
  error = null, 
  onRefresh = null 
}) => {
  const getXAppStats = () => {
    if (!xApps || xApps.length === 0) {
      return {
        total: 0,
        running: 0,
        stopped: 0,
        deploying: 0,
        failed: 0,
        totalInstances: 0
      };
    }

    return xApps.reduce((stats, xapp) => {
      stats.total++;
      stats.totalInstances += xapp.instances || 0;
      
      switch (xapp.status?.toLowerCase()) {
        case 'running':
        case 'deployed':
        case 'active':
          stats.running++;
          break;
        case 'stopped':
        case 'undeployed':
        case 'inactive':
          stats.stopped++;
          break;
        case 'deploying':
        case 'starting':
          stats.deploying++;
          break;
        case 'failed':
        case 'error':
          stats.failed++;
          break;
        default:
          break;
      }
      return stats;
    }, { total: 0, running: 0, stopped: 0, deploying: 0, failed: 0, totalInstances: 0 });
  };

  const xappStats = getXAppStats();

  const getResourceUsage = () => {
    if (!xApps || xApps.length === 0) return null;
    
    return xApps.reduce((usage, xapp) => {
      if (xapp.resources) {
        usage.cpu += parseFloat(xapp.resources.cpu || 0);
        usage.memory += parseFloat(xapp.resources.memory || 0);
      }
      return usage;
    }, { cpu: 0, memory: 0 });
  };

  const resourceUsage = getResourceUsage();

  return (
    <ComponentStatusPanel
      component={component}
      loading={loading}
      error={error}
      onRefresh={onRefresh}
    >
      <div className="app-manager-details">
        <div className="metrics-section">
          <h4>xApp Deployment Metrics</h4>
          <div className="metrics-grid">
            <div className="metric-item">
              <span className="metric-label">Total xApps:</span>
              <span className="metric-value">{xappStats.total}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Running:</span>
              <span className="metric-value running">{xappStats.running}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Deploying:</span>
              <span className="metric-value deploying">{xappStats.deploying}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Failed:</span>
              <span className="metric-value failed">{xappStats.failed}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Total Instances:</span>
              <span className="metric-value">{xappStats.totalInstances}</span>
            </div>
          </div>
        </div>

        {resourceUsage && (resourceUsage.cpu > 0 || resourceUsage.memory > 0) && (
          <div className="resource-section">
            <h4>Resource Usage</h4>
            <div className="resource-grid">
              <div className="resource-item">
                <span className="resource-label">Total CPU:</span>
                <span className="resource-value">{resourceUsage.cpu.toFixed(2)} cores</span>
              </div>
              <div className="resource-item">
                <span className="resource-label">Total Memory:</span>
                <span className="resource-value">{resourceUsage.memory.toFixed(2)} GB</span>
              </div>
            </div>
          </div>
        )}

        {component?.metrics && (
          <div className="performance-metrics">
            <h4>Performance Metrics</h4>
            <div className="metrics-grid">
              {component.metrics.deploymentsPerHour && (
                <div className="metric-item">
                  <span className="metric-label">Deployments/hour:</span>
                  <span className="metric-value">{component.metrics.deploymentsPerHour.toFixed(1)}</span>
                </div>
              )}
              {component.metrics.averageDeployTime && (
                <div className="metric-item">
                  <span className="metric-label">Avg Deploy Time:</span>
                  <span className="metric-value">{component.metrics.averageDeployTime}s</span>
                </div>
              )}
              {component.metrics.successRate && (
                <div className="metric-item">
                  <span className="metric-label">Success Rate:</span>
                  <span className="metric-value">{(component.metrics.successRate * 100).toFixed(1)}%</span>
                </div>
              )}
            </div>
          </div>
        )}

        {xApps && xApps.length > 0 && (
          <div className="xapps-section">
            <h4>Deployed xApps</h4>
            <div className="xapps-list">
              {xApps.slice(0, 5).map((xapp, index) => (
                <div key={xapp.name || index} className="xapp-item">
                  <div className="xapp-header">
                    <div className="xapp-info">
                      <span className="xapp-name">{xapp.name || `xApp-${index + 1}`}</span>
                      <span className="xapp-version">{xapp.version || 'Unknown'}</span>
                    </div>
                    <span className={`xapp-status ${xapp.status?.toLowerCase() || 'unknown'}`}>
                      {xapp.status || 'Unknown'}
                    </span>
                  </div>
                  
                  <div className="xapp-details">
                    {xapp.instances && (
                      <div className="detail-item">
                        <span className="detail-label">Instances:</span>
                        <span className="detail-value">{xapp.instances}</span>
                      </div>
                    )}
                    {xapp.resources && (
                      <div className="detail-item">
                        <span className="detail-label">Resources:</span>
                        <span className="detail-value">
                          {xapp.resources.cpu && `${xapp.resources.cpu} CPU`}
                          {xapp.resources.cpu && xapp.resources.memory && ', '}
                          {xapp.resources.memory && `${xapp.resources.memory} Memory`}
                        </span>
                      </div>
                    )}
                    {xapp.namespace && (
                      <div className="detail-item">
                        <span className="detail-label">Namespace:</span>
                        <span className="detail-value">{xapp.namespace}</span>
                      </div>
                    )}
                  </div>
                  
                  {xapp.deployedAt && (
                    <div className="xapp-timestamp">
                      Deployed: {new Date(xapp.deployedAt).toLocaleString()}
                    </div>
                  )}
                </div>
              ))}
              {xApps.length > 5 && (
                <div className="more-xapps">
                  +{xApps.length - 5} more xApps
                </div>
              )}
            </div>
          </div>
        )}

        {component?.kubernetesConnection && (
          <div className="k8s-section">
            <h4>Kubernetes Integration</h4>
            <div className="k8s-info">
              <div className="k8s-item">
                <span className="k8s-label">Connection Status:</span>
                <span className={`k8s-value ${component.kubernetesConnection.status?.toLowerCase() || 'unknown'}`}>
                  {component.kubernetesConnection.status || 'Unknown'}
                </span>
              </div>
              {component.kubernetesConnection.cluster && (
                <div className="k8s-item">
                  <span className="k8s-label">Cluster:</span>
                  <span className="k8s-value">{component.kubernetesConnection.cluster}</span>
                </div>
              )}
              {component.kubernetesConnection.namespace && (
                <div className="k8s-item">
                  <span className="k8s-label">Default Namespace:</span>
                  <span className="k8s-value">{component.kubernetesConnection.namespace}</span>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </ComponentStatusPanel>
  );
};

export default AppManagerStatus;