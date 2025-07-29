/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import ComponentStatusPanel from './ComponentStatusPanel';
import './SubscriptionManagerStatus.css';

/**
 * Subscription Manager status display component
 * Shows subscription statistics, active subscriptions, and performance metrics
 */
const SubscriptionManagerStatus = ({ 
  component, 
  subscriptions = [], 
  loading = false, 
  error = null, 
  onRefresh = null 
}) => {
  const getSubscriptionStats = () => {
    if (!subscriptions || subscriptions.length === 0) {
      return {
        total: 0,
        active: 0,
        pending: 0,
        failed: 0,
        completed: 0
      };
    }

    return subscriptions.reduce((stats, sub) => {
      stats.total++;
      switch (sub.status?.toLowerCase()) {
        case 'active':
        case 'running':
          stats.active++;
          break;
        case 'pending':
        case 'creating':
          stats.pending++;
          break;
        case 'failed':
        case 'error':
          stats.failed++;
          break;
        case 'completed':
        case 'finished':
          stats.completed++;
          break;
        default:
          break;
      }
      return stats;
    }, { total: 0, active: 0, pending: 0, failed: 0, completed: 0 });
  };

  const subStats = getSubscriptionStats();

  const getRecentSubscriptions = () => {
    if (!subscriptions || subscriptions.length === 0) return [];
    
    return subscriptions
      .sort((a, b) => new Date(b.createdAt || b.timestamp || 0) - new Date(a.createdAt || a.timestamp || 0))
      .slice(0, 5);
  };

  const recentSubscriptions = getRecentSubscriptions();

  return (
    <ComponentStatusPanel
      component={component}
      loading={loading}
      error={error}
      onRefresh={onRefresh}
    >
      <div className="subscription-manager-details">
        <div className="metrics-section">
          <h4>Subscription Metrics</h4>
          <div className="metrics-grid">
            <div className="metric-item">
              <span className="metric-label">Total Subscriptions:</span>
              <span className="metric-value">{subStats.total}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Active:</span>
              <span className="metric-value active">{subStats.active}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Pending:</span>
              <span className="metric-value pending">{subStats.pending}</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Failed:</span>
              <span className="metric-value failed">{subStats.failed}</span>
            </div>
          </div>
        </div>

        {component?.metrics && (
          <div className="performance-metrics">
            <h4>Performance Metrics</h4>
            <div className="metrics-grid">
              {component.metrics.subscriptionsPerSecond && (
                <div className="metric-item">
                  <span className="metric-label">Subscriptions/sec:</span>
                  <span className="metric-value">{component.metrics.subscriptionsPerSecond.toFixed(2)}</span>
                </div>
              )}
              {component.metrics.averageProcessingTime && (
                <div className="metric-item">
                  <span className="metric-label">Avg Processing Time:</span>
                  <span className="metric-value">{component.metrics.averageProcessingTime}ms</span>
                </div>
              )}
              {component.metrics.indicationsReceived && (
                <div className="metric-item">
                  <span className="metric-label">Indications Received:</span>
                  <span className="metric-value">{component.metrics.indicationsReceived.toLocaleString()}</span>
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

        {recentSubscriptions.length > 0 && (
          <div className="subscriptions-section">
            <h4>Recent Subscriptions</h4>
            <div className="subscriptions-list">
              {recentSubscriptions.map((subscription, index) => (
                <div key={subscription.id || subscription.subscriptionId || index} className="subscription-item">
                  <div className="subscription-info">
                    <div className="subscription-header">
                      <span className="subscription-id">
                        {subscription.id || subscription.subscriptionId || `Sub-${index + 1}`}
                      </span>
                      <span className={`subscription-status ${subscription.status?.toLowerCase() || 'unknown'}`}>
                        {subscription.status || 'Unknown'}
                      </span>
                    </div>
                    <div className="subscription-details">
                      {subscription.nodeId && (
                        <span className="detail-item">
                          <span className="detail-label">Node:</span>
                          <span className="detail-value">{subscription.nodeId}</span>
                        </span>
                      )}
                      {subscription.functionId && (
                        <span className="detail-item">
                          <span className="detail-label">Function:</span>
                          <span className="detail-value">{subscription.functionId}</span>
                        </span>
                      )}
                      {subscription.eventTrigger && (
                        <span className="detail-item">
                          <span className="detail-label">Trigger:</span>
                          <span className="detail-value">{subscription.eventTrigger}</span>
                        </span>
                      )}
                    </div>
                    {(subscription.createdAt || subscription.timestamp) && (
                      <div className="subscription-timestamp">
                        Created: {new Date(subscription.createdAt || subscription.timestamp).toLocaleString()}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {subscriptions.length > 5 && (
                <div className="more-subscriptions">
                  +{subscriptions.length - 5} more subscriptions
                </div>
              )}
            </div>
          </div>
        )}

        {component?.sdlConnection && (
          <div className="sdl-section">
            <h4>Shared Data Layer (SDL)</h4>
            <div className="sdl-info">
              <div className="sdl-item">
                <span className="sdl-label">Connection Status:</span>
                <span className={`sdl-value ${component.sdlConnection.status?.toLowerCase() || 'unknown'}`}>
                  {component.sdlConnection.status || 'Unknown'}
                </span>
              </div>
              {component.sdlConnection.endpoint && (
                <div className="sdl-item">
                  <span className="sdl-label">Endpoint:</span>
                  <span className="sdl-value">{component.sdlConnection.endpoint}</span>
                </div>
              )}
              {component.sdlConnection.keysStored && (
                <div className="sdl-item">
                  <span className="sdl-label">Keys Stored:</span>
                  <span className="sdl-value">{component.sdlConnection.keysStored.toLocaleString()}</span>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </ComponentStatusPanel>
  );
};

export default SubscriptionManagerStatus;