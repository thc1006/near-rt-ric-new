/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './PolicyInstanceViewer.css';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import dashboardAPI from '../services/api';

const PolicyInstanceViewer = ({ policyType, policyInstance, onClose, onDelete }) => {
  const [instanceStatus, setInstanceStatus] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [viewMode, setViewMode] = useState('formatted'); // 'formatted' or 'raw'

  useEffect(() => {
    loadInstanceStatus();
  }, [policyType, policyInstance]);

  const loadInstanceStatus = async () => {
    setLoading(true);
    setError(null);

    try {
      const status = await dashboardAPI.getPolicyInstanceStatus(
        policyType.policy_type_id,
        policyInstance.policy_instance_id
      );
      setInstanceStatus(status);
    } catch (err) {
      console.error('Failed to load policy instance status:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const formatPolicy = (policy) => {
    try {
      if (typeof policy === 'string') {
        return JSON.stringify(JSON.parse(policy), null, 2);
      }
      return JSON.stringify(policy, null, 2);
    } catch (err) {
      return typeof policy === 'string' ? policy : JSON.stringify(policy);
    }
  };

  const renderPolicyTree = (policy, path = '') => {
    try {
      const policyObj = typeof policy === 'string' ? JSON.parse(policy) : policy;
      return renderObjectTree(policyObj, path);
    } catch (err) {
      return (
        <div className="policy-error">
          <p>Unable to parse policy structure</p>
          <pre>{typeof policy === 'string' ? policy : JSON.stringify(policy)}</pre>
        </div>
      );
    }
  };

  const renderObjectTree = (obj, path = '', level = 0) => {
    if (typeof obj !== 'object' || obj === null) {
      return <span className="policy-value">{String(obj)}</span>;
    }

    return (
      <div className={`policy-object level-${level}`}>
        {Object.entries(obj).map(([key, value]) => {
          const currentPath = path ? `${path}.${key}` : key;
          const isObject = typeof value === 'object' && value !== null;
          const isArray = Array.isArray(value);

          return (
            <div key={key} className="policy-property">
              <div className="property-header">
                <span className="property-key">{key}</span>
                {isArray && <span className="property-type">array</span>}
                {isObject && !isArray && <span className="property-type">object</span>}
                {!isObject && <span className="property-type">{typeof value}</span>}
              </div>
              
              <div className="property-value">
                {isObject ? (
                  renderObjectTree(value, currentPath, level + 1)
                ) : (
                  <span className={`policy-value ${typeof value}`}>
                    {String(value)}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'active':
        return '#28a745';
      case 'inactive':
        return '#6c757d';
      case 'error':
        return '#dc3545';
      case 'pending':
        return '#ffc107';
      default:
        return '#6c757d';
    }
  };

  const handleDelete = () => {
    if (window.confirm(`Are you sure you want to delete policy instance "${policyInstance.policy_instance_id}"?`)) {
      onDelete();
      onClose();
    }
  };

  return (
    <div className="policy-instance-viewer-overlay">
      <div className="policy-instance-viewer-modal">
        <div className="viewer-header">
          <div className="header-info">
            <h3>Policy Instance: {policyInstance.policy_instance_id}</h3>
            <p className="policy-type-info">
              Type: {policyType.name || policyType.policy_type_id}
            </p>
          </div>
          <div className="header-actions">
            <button onClick={handleDelete} className="delete-btn" title="Delete Instance">
              🗑️ Delete
            </button>
            <button onClick={onClose} className="close-btn">×</button>
          </div>
        </div>

        <div className="instance-meta">
          <div className="meta-section">
            <h4>Status Information</h4>
            <div className="meta-grid">
              <div className="meta-item">
                <span className="meta-label">Status:</span>
                <div className="status-display">
                  <span 
                    className="status-dot"
                    style={{ backgroundColor: getStatusColor(instanceStatus?.status || policyInstance.status?.status) }}
                  ></span>
                  <span className="status-text">
                    {instanceStatus?.status || policyInstance.status?.status || 'Unknown'}
                  </span>
                </div>
              </div>
              <div className="meta-item">
                <span className="meta-label">Created:</span>
                <span className="meta-value">
                  {new Date(policyInstance.created_at).toLocaleString()}
                </span>
              </div>
              <div className="meta-item">
                <span className="meta-label">Updated:</span>
                <span className="meta-value">
                  {new Date(policyInstance.updated_at).toLocaleString()}
                </span>
              </div>
              <div className="meta-item">
                <span className="meta-label">Last Status Check:</span>
                <span className="meta-value">
                  {instanceStatus?.last_update 
                    ? new Date(instanceStatus.last_update).toLocaleString()
                    : 'Not available'
                  }
                </span>
              </div>
            </div>
          </div>

          {(instanceStatus?.reason || policyInstance.status?.reason) && (
            <div className="meta-section">
              <h4>Status Details</h4>
              <p className="status-reason">
                {instanceStatus?.reason || policyInstance.status?.reason}
              </p>
            </div>
          )}
        </div>

        {error && (
          <div className="error-section">
            <ErrorDisplay error={error} onRetry={loadInstanceStatus} />
          </div>
        )}

        {loading && (
          <div className="loading-section">
            <LoadingDisplay message="Loading instance status..." />
          </div>
        )}

        <div className="view-controls">
          <button
            className={`view-btn ${viewMode === 'formatted' ? 'active' : ''}`}
            onClick={() => setViewMode('formatted')}
          >
            Tree View
          </button>
          <button
            className={`view-btn ${viewMode === 'raw' ? 'active' : ''}`}
            onClick={() => setViewMode('raw')}
          >
            Raw JSON
          </button>
        </div>

        <div className="policy-content">
          <h4>Policy Configuration</h4>
          {viewMode === 'formatted' ? (
            <div className="policy-tree">
              {renderPolicyTree(policyInstance.policy)}
            </div>
          ) : (
            <pre className="policy-raw">
              {formatPolicy(policyInstance.policy)}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
};

export default PolicyInstanceViewer;