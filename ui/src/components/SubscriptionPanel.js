/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './SubscriptionPanel.css';

/**
 * Subscription Panel component
 * Displays and manages E2 subscriptions through Subscription Manager
 */
const SubscriptionPanel = () => {
  const [subscriptions, setSubscriptions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedSubscription, setSelectedSubscription] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [filter, setFilter] = useState({
    e2NodeId: '',
    xappId: '',
    status: '',
  });

  // Fetch subscriptions from the API
  const fetchSubscriptions = async () => {
    try {
      setLoading(true);
      const queryParams = new URLSearchParams();
      
      if (filter.e2NodeId) queryParams.append('e2NodeId', filter.e2NodeId);
      if (filter.xappId) queryParams.append('xappId', filter.xappId);
      if (filter.status) queryParams.append('status', filter.status);
      
      const url = `/api/v1/subscriptions${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
      const response = await fetch(url);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const data = await response.json();
      setSubscriptions(data.subscriptions || []);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch subscriptions:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Fetch detailed subscription information
  const fetchSubscriptionDetails = async (subscriptionId) => {
    try {
      const response = await fetch(`/api/v1/subscriptions/${subscriptionId}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const subscriptionData = await response.json();
      setSelectedSubscription(subscriptionData);
    } catch (err) {
      console.error('Failed to fetch subscription details:', err);
      setError(err.message);
    }
  };

  // Delete subscription
  const deleteSubscription = async (subscriptionId) => {
    if (!window.confirm('Are you sure you want to delete this subscription?')) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/subscriptions/${subscriptionId}`, {
        method: 'DELETE',
      });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      // Refresh subscriptions list
      await fetchSubscriptions();
      
      // Clear selected subscription if it was deleted
      if (selectedSubscription?.id === subscriptionId) {
        setSelectedSubscription(null);
      }
      
      alert('Subscription deleted successfully');
    } catch (err) {
      console.error('Failed to delete subscription:', err);
      alert(`Failed to delete subscription: ${err.message}`);
    }
  };

  useEffect(() => {
    fetchSubscriptions();
  }, [filter]);

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'active':
        return 'status-active';
      case 'pending':
        return 'status-pending';
      case 'failed':
        return 'status-failed';
      case 'deleted':
        return 'status-deleted';
      case 'completed':
        return 'status-completed';
      default:
        return 'status-unknown';
    }
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleString();
  };

  const getActionTypeDisplay = (type) => {
    switch (type) {
      case 'REPORT':
        return 'Report';
      case 'INSERT':
        return 'Insert';
      case 'POLICY':
        return 'Policy';
      default:
        return type || 'Unknown';
    }
  };

  const getTriggerTypeDisplay = (type) => {
    switch (type) {
      case 'PERIODIC':
        return 'Periodic';
      case 'ON_CHANGE':
        return 'On Change';
      case 'ON_DEMAND':
        return 'On Demand';
      default:
        return type || 'Unknown';
    }
  };

  if (loading && subscriptions.length === 0) {
    return (
      <div className="subscription-panel">
        <div className="panel-header">
          <h3>Subscriptions</h3>
        </div>
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Loading subscriptions...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="subscription-panel">
      <div className="panel-header">
        <h3>Subscriptions ({subscriptions.length})</h3>
        <div className="header-controls">
          <button 
            onClick={() => setShowCreateForm(true)} 
            className="create-btn"
          >
            Create Subscription
          </button>
          <button 
            onClick={fetchSubscriptions} 
            className="refresh-btn"
            disabled={loading}
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="filter-section">
        <div className="filter-controls">
          <input
            type="text"
            placeholder="Filter by E2 Node ID"
            value={filter.e2NodeId}
            onChange={(e) => setFilter({...filter, e2NodeId: e.target.value})}
            className="filter-input"
          />
          <input
            type="text"
            placeholder="Filter by xApp ID"
            value={filter.xappId}
            onChange={(e) => setFilter({...filter, xappId: e.target.value})}
            className="filter-input"
          />
          <select
            value={filter.status}
            onChange={(e) => setFilter({...filter, status: e.target.value})}
            className="filter-select"
          >
            <option value="">All Statuses</option>
            <option value="ACTIVE">Active</option>
            <option value="PENDING">Pending</option>
            <option value="FAILED">Failed</option>
            <option value="DELETED">Deleted</option>
            <option value="COMPLETED">Completed</option>
          </select>
          <button 
            onClick={() => setFilter({e2NodeId: '', xappId: '', status: ''})}
            className="clear-filters-btn"
          >
            Clear Filters
          </button>
        </div>
      </div>

      {error && (
        <div className="error-banner">
          <p>Error: {error}</p>
          <button onClick={fetchSubscriptions} className="retry-btn">
            Retry
          </button>
        </div>
      )}

      {subscriptions.length === 0 && !loading ? (
        <div className="empty-state">
          <p>No subscriptions found</p>
          <p className="empty-subtitle">
            Create a subscription to start receiving E2 indications from RAN nodes
          </p>
        </div>
      ) : (
        <div className="subscriptions-container">
          <div className="subscriptions-list">
            {subscriptions.map((subscription) => (
              <div 
                key={subscription.id} 
                className={`subscription-card ${selectedSubscription?.id === subscription.id ? 'selected' : ''}`}
                onClick={() => fetchSubscriptionDetails(subscription.id)}
              >
                <div className="subscription-header">
                  <div className="subscription-id">{subscription.id}</div>
                  <div className={`subscription-status ${getStatusColor(subscription.status)}`}>
                    {subscription.status}
                  </div>
                </div>
                <div className="subscription-details">
                  <div className="detail-row">
                    <span className="label">E2 Node:</span>
                    <span className="value">{subscription.e2NodeId}</span>
                  </div>
                  <div className="detail-row">
                    <span className="label">xApp:</span>
                    <span className="value">{subscription.xappId}</span>
                  </div>
                  <div className="detail-row">
                    <span className="label">RAN Function:</span>
                    <span className="value">{subscription.ranFunctionId}</span>
                  </div>
                  <div className="detail-row">
                    <span className="label">Actions:</span>
                    <span className="value">{subscription.actions?.length || 0}</span>
                  </div>
                </div>
                <div className="subscription-footer">
                  <div className="created-at">
                    Created: {formatTimestamp(subscription.createdAt)}
                  </div>
                  <div className="subscription-actions">
                    <button 
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteSubscription(subscription.id);
                      }}
                      className="delete-btn"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {selectedSubscription && (
            <div className="subscription-details-panel">
              <div className="details-header">
                <h4>Subscription Details: {selectedSubscription.id}</h4>
                <button 
                  onClick={() => setSelectedSubscription(null)}
                  className="close-btn"
                >
                  ×
                </button>
              </div>
              
              <div className="details-content">
                <div className="detail-section">
                  <h5>Basic Information</h5>
                  <div className="detail-grid">
                    <div className="detail-item">
                      <span className="label">Subscription ID:</span>
                      <span className="value">{selectedSubscription.id}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Status:</span>
                      <span className={`value ${getStatusColor(selectedSubscription.status)}`}>
                        {selectedSubscription.status}
                      </span>
                    </div>
                    <div className="detail-item">
                      <span className="label">E2 Node ID:</span>
                      <span className="value">{selectedSubscription.e2NodeId}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">xApp ID:</span>
                      <span className="value">{selectedSubscription.xappId}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">RAN Function ID:</span>
                      <span className="value">{selectedSubscription.ranFunctionId}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Created:</span>
                      <span className="value">{formatTimestamp(selectedSubscription.createdAt)}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Updated:</span>
                      <span className="value">{formatTimestamp(selectedSubscription.updatedAt)}</span>
                    </div>
                  </div>
                  {selectedSubscription.errorMessage && (
                    <div className="error-message">
                      <strong>Error:</strong> {selectedSubscription.errorMessage}
                    </div>
                  )}
                </div>

                <div className="detail-section">
                  <h5>Event Trigger</h5>
                  <div className="detail-grid">
                    <div className="detail-item">
                      <span className="label">Type:</span>
                      <span className="value">
                        {getTriggerTypeDisplay(selectedSubscription.eventTrigger?.type)}
                      </span>
                    </div>
                    {selectedSubscription.eventTrigger?.period && (
                      <div className="detail-item">
                        <span className="label">Period:</span>
                        <span className="value">{selectedSubscription.eventTrigger.period}ms</span>
                      </div>
                    )}
                  </div>
                  {selectedSubscription.eventTrigger?.definition && (
                    <div className="definition-section">
                      <h6>Definition:</h6>
                      <pre className="definition-content">
                        {atob(selectedSubscription.eventTrigger.definition)}
                      </pre>
                    </div>
                  )}
                </div>

                {selectedSubscription.actions && selectedSubscription.actions.length > 0 && (
                  <div className="detail-section">
                    <h5>Actions ({selectedSubscription.actions.length})</h5>
                    <div className="actions-list">
                      {selectedSubscription.actions.map((action) => (
                        <div key={action.id} className="action-item">
                          <div className="action-header">
                            <span className="action-id">Action ID: {action.id}</span>
                            <span className="action-type">
                              {getActionTypeDisplay(action.type)}
                            </span>
                          </div>
                          {action.definition && (
                            <div className="action-definition">
                              <h6>Definition:</h6>
                              <pre className="definition-content">
                                {atob(action.definition)}
                              </pre>
                            </div>
                          )}
                          {action.subsequentAction && (
                            <div className="subsequent-action">
                              <h6>Subsequent Action:</h6>
                              <div className="subsequent-details">
                                <span>Type: {getActionTypeDisplay(action.subsequentAction.type)}</span>
                                <span>Time to Wait: {action.subsequentAction.timeToWait}ms</span>
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {showCreateForm && (
        <CreateSubscriptionModal 
          onClose={() => setShowCreateForm(false)}
          onSuccess={() => {
            setShowCreateForm(false);
            fetchSubscriptions();
          }}
        />
      )}
    </div>
  );
};

// Simple Create Subscription Modal (placeholder)
const CreateSubscriptionModal = ({ onClose, onSuccess }) => {
  const [formData, setFormData] = useState({
    e2NodeId: '',
    xappId: '',
    ranFunctionId: '',
    eventTriggerType: 'PERIODIC',
    period: '1000',
    actionType: 'REPORT',
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    const subscriptionRequest = {
      e2NodeId: formData.e2NodeId,
      xappId: formData.xappId,
      ranFunctionId: parseInt(formData.ranFunctionId),
      eventTrigger: {
        type: formData.eventTriggerType,
        definition: btoa('{}'), // Base64 encoded empty JSON
        ...(formData.eventTriggerType === 'PERIODIC' && {
          period: parseInt(formData.period)
        })
      },
      actions: [{
        id: 1,
        type: formData.actionType,
        definition: btoa('{}') // Base64 encoded empty JSON
      }]
    };

    try {
      const response = await fetch('/api/v1/subscriptions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(subscriptionRequest),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      onSuccess();
      alert('Subscription created successfully');
    } catch (err) {
      console.error('Failed to create subscription:', err);
      alert(`Failed to create subscription: ${err.message}`);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h4>Create Subscription</h4>
          <button onClick={onClose} className="close-btn">×</button>
        </div>
        <form onSubmit={handleSubmit} className="subscription-form">
          <div className="form-group">
            <label>E2 Node ID:</label>
            <input
              type="text"
              value={formData.e2NodeId}
              onChange={(e) => setFormData({...formData, e2NodeId: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>xApp ID:</label>
            <input
              type="text"
              value={formData.xappId}
              onChange={(e) => setFormData({...formData, xappId: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>RAN Function ID:</label>
            <input
              type="number"
              value={formData.ranFunctionId}
              onChange={(e) => setFormData({...formData, ranFunctionId: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>Event Trigger Type:</label>
            <select
              value={formData.eventTriggerType}
              onChange={(e) => setFormData({...formData, eventTriggerType: e.target.value})}
            >
              <option value="PERIODIC">Periodic</option>
              <option value="ON_CHANGE">On Change</option>
              <option value="ON_DEMAND">On Demand</option>
            </select>
          </div>
          {formData.eventTriggerType === 'PERIODIC' && (
            <div className="form-group">
              <label>Period (ms):</label>
              <input
                type="number"
                value={formData.period}
                onChange={(e) => setFormData({...formData, period: e.target.value})}
                min="100"
              />
            </div>
          )}
          <div className="form-group">
            <label>Action Type:</label>
            <select
              value={formData.actionType}
              onChange={(e) => setFormData({...formData, actionType: e.target.value})}
            >
              <option value="REPORT">Report</option>
              <option value="INSERT">Insert</option>
              <option value="POLICY">Policy</option>
            </select>
          </div>
          <div className="form-actions">
            <button type="button" onClick={onClose} className="cancel-btn">
              Cancel
            </button>
            <button type="submit" className="submit-btn">
              Create Subscription
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default SubscriptionPanel;