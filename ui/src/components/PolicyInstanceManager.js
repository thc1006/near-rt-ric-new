/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './PolicyInstanceManager.css';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import PolicyInstanceForm from './PolicyInstanceForm';
import PolicyInstanceViewer from './PolicyInstanceViewer';
import dashboardAPI from '../services/api';

const PolicyInstanceManager = ({ 
  policyTypes, 
  policyInstances, 
  selectedPolicyType, 
  onPolicyInstanceCreated, 
  onPolicyInstanceDeleted, 
  onRefresh 
}) => {
  const [currentPolicyType, setCurrentPolicyType] = useState(selectedPolicyType);
  const [selectedInstance, setSelectedInstance] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [showInstanceViewer, setShowInstanceViewer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');

  useEffect(() => {
    if (selectedPolicyType) {
      setCurrentPolicyType(selectedPolicyType);
    }
  }, [selectedPolicyType]);

  const currentInstances = currentPolicyType 
    ? (policyInstances[currentPolicyType.policy_type_id] || [])
    : [];

  const filteredInstances = currentInstances.filter(instance => {
    const matchesSearch = 
      instance.policy_instance_id?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      JSON.stringify(instance.policy).toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesStatus = statusFilter === 'all' || 
      instance.status?.status?.toLowerCase() === statusFilter.toLowerCase();

    return matchesSearch && matchesStatus;
  });

  const handlePolicyTypeChange = (e) => {
    const policyTypeId = e.target.value;
    const policyType = policyTypes.find(pt => pt.policy_type_id === policyTypeId);
    setCurrentPolicyType(policyType);
    setSelectedInstance(null);
    setShowInstanceViewer(false);
  };

  const handleInstanceClick = (instance) => {
    setSelectedInstance(instance);
    setShowInstanceViewer(true);
  };

  const handleCreateInstance = async (instanceData) => {
    if (!currentPolicyType) return;

    setLoading(true);
    setError(null);

    try {
      await dashboardAPI.createPolicyInstance(
        currentPolicyType.policy_type_id,
        instanceData.policy_instance_id,
        { policy: instanceData.policy }
      );

      // Fetch the created instance to get complete data
      const createdInstance = await dashboardAPI.getPolicyInstance(
        currentPolicyType.policy_type_id,
        instanceData.policy_instance_id
      );

      onPolicyInstanceCreated(currentPolicyType.policy_type_id, createdInstance);
      setShowCreateForm(false);
    } catch (err) {
      console.error('Failed to create policy instance:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteInstance = async (policyInstanceId) => {
    if (!currentPolicyType) return;

    if (!window.confirm(`Are you sure you want to delete policy instance "${policyInstanceId}"?`)) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await dashboardAPI.deletePolicyInstance(
        currentPolicyType.policy_type_id,
        policyInstanceId
      );

      onPolicyInstanceDeleted(currentPolicyType.policy_type_id, policyInstanceId);
      
      if (selectedInstance?.policy_instance_id === policyInstanceId) {
        setSelectedInstance(null);
        setShowInstanceViewer(false);
      }
    } catch (err) {
      console.error('Failed to delete policy instance:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
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

  if (!policyTypes.length) {
    return (
      <div className="policy-instance-manager">
        <div className="no-policy-types">
          <p>No policy types available</p>
          <p>Create a policy type first to manage policy instances</p>
        </div>
      </div>
    );
  }

  return (
    <div className="policy-instance-manager">
      <div className="manager-header">
        <div className="policy-type-selector">
          <label htmlFor="policyTypeSelect">Policy Type:</label>
          <select
            id="policyTypeSelect"
            value={currentPolicyType?.policy_type_id || ''}
            onChange={handlePolicyTypeChange}
            className="policy-type-select"
          >
            <option value="">Select a policy type...</option>
            {policyTypes.map(pt => (
              <option key={pt.policy_type_id} value={pt.policy_type_id}>
                {pt.name || pt.policy_type_id}
              </option>
            ))}
          </select>
        </div>

        {currentPolicyType && (
          <button
            onClick={() => setShowCreateForm(true)}
            className="create-instance-btn"
            disabled={loading}
          >
            Create Instance
          </button>
        )}
      </div>

      {currentPolicyType && (
        <div className="policy-type-info">
          <h3>{currentPolicyType.name || currentPolicyType.policy_type_id}</h3>
          <p>{currentPolicyType.description || 'No description available'}</p>
          <div className="instance-count">
            <span>Instances: {currentInstances.length}</span>
          </div>
        </div>
      )}

      {currentPolicyType && (
        <div className="filters-section">
          <div className="search-filter">
            <input
              type="text"
              placeholder="Search instances..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>
          <div className="status-filter">
            <label htmlFor="statusFilter">Status:</label>
            <select
              id="statusFilter"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="status-select"
            >
              <option value="all">All</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
              <option value="pending">Pending</option>
              <option value="error">Error</option>
            </select>
          </div>
        </div>
      )}

      {error && (
        <ErrorDisplay 
          error={error} 
          onRetry={() => {
            setError(null);
            onRefresh();
          }} 
        />
      )}

      {currentPolicyType && (
        <div className="instances-grid">
          {loading && <LoadingDisplay message="Loading instances..." />}
          
          {!loading && filteredInstances.length === 0 ? (
            <div className="no-instances">
              <p>No policy instances found</p>
              {searchTerm || statusFilter !== 'all' ? (
                <p>Try adjusting your search or filter criteria</p>
              ) : (
                <p>Create your first policy instance to get started</p>
              )}
            </div>
          ) : (
            filteredInstances.map(instance => (
              <div key={instance.policy_instance_id} className="instance-card">
                <div className="card-header">
                  <h4 className="instance-id">{instance.policy_instance_id}</h4>
                  <div className="card-actions">
                    <button
                      onClick={() => handleInstanceClick(instance)}
                      className="view-btn"
                      title="View Instance"
                    >
                      👁️
                    </button>
                    <button
                      onClick={() => handleDeleteInstance(instance.policy_instance_id)}
                      className="delete-btn"
                      title="Delete Instance"
                      disabled={loading}
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                
                <div className="card-content">
                  <div className="status-indicator">
                    <span 
                      className="status-dot"
                      style={{ backgroundColor: getStatusColor(instance.status?.status) }}
                    ></span>
                    <span className="status-text">
                      {instance.status?.status || 'Unknown'}
                    </span>
                  </div>
                  
                  <div className="instance-meta">
                    <div className="meta-item">
                      <span className="meta-label">Created:</span>
                      <span className="meta-value">
                        {new Date(instance.created_at).toLocaleDateString()}
                      </span>
                    </div>
                    <div className="meta-item">
                      <span className="meta-label">Updated:</span>
                      <span className="meta-value">
                        {new Date(instance.updated_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                  
                  <div className="policy-preview">
                    <pre>{JSON.stringify(instance.policy, null, 2).substring(0, 200)}...</pre>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {showCreateForm && currentPolicyType && (
        <PolicyInstanceForm
          policyType={currentPolicyType}
          onSubmit={handleCreateInstance}
          onCancel={() => {
            setShowCreateForm(false);
            setError(null);
          }}
          loading={loading}
          error={error}
        />
      )}

      {showInstanceViewer && selectedInstance && currentPolicyType && (
        <PolicyInstanceViewer
          policyType={currentPolicyType}
          policyInstance={selectedInstance}
          onClose={() => {
            setShowInstanceViewer(false);
            setSelectedInstance(null);
          }}
          onDelete={() => handleDeleteInstance(selectedInstance.policy_instance_id)}
        />
      )}
    </div>
  );
};

export default PolicyInstanceManager;