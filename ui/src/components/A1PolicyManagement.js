/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './A1PolicyManagement.css';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import PolicyTypeBrowser from './PolicyTypeBrowser';
import PolicyInstanceManager from './PolicyInstanceManager';
import PolicyStatusDashboard from './PolicyStatusDashboard';
import PolicyConflictResolver from './PolicyConflictResolver';
import dashboardAPI from '../services/api';

const A1PolicyManagement = () => {
  const [activeTab, setActiveTab] = useState('types');
  const [policyTypes, setPolicyTypes] = useState([]);
  const [policyInstances, setPolicyInstances] = useState({});
  const [a1Health, setA1Health] = useState(null);
  const [a1Stats, setA1Stats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedPolicyType, setSelectedPolicyType] = useState(null);

  // Load initial data
  useEffect(() => {
    loadA1Data();
  }, []);

  const loadA1Data = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Load A1 health status
      const healthData = await dashboardAPI.getA1Health();
      setA1Health(healthData);

      // Load policy types
      const typesData = await dashboardAPI.getPolicyTypes();
      setPolicyTypes(typesData.policy_types || []);

      // Load policy instances for each type
      const instancesData = {};
      for (const policyType of typesData.policy_types || []) {
        try {
          const instances = await dashboardAPI.getPolicyInstances(policyType.policy_type_id);
          instancesData[policyType.policy_type_id] = instances.policy_instances || [];
        } catch (err) {
          console.warn(`Failed to load instances for policy type ${policyType.policy_type_id}:`, err);
          instancesData[policyType.policy_type_id] = [];
        }
      }
      setPolicyInstances(instancesData);

      // Load A1 statistics
      const statsData = await dashboardAPI.getA1Stats();
      setA1Stats(statsData);

    } catch (err) {
      console.error('Failed to load A1 data:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = () => {
    loadA1Data();
  };

  const handlePolicyTypeSelect = (policyType) => {
    setSelectedPolicyType(policyType);
    setActiveTab('instances');
  };

  const handlePolicyTypeCreated = (policyType) => {
    setPolicyTypes(prev => [...prev, policyType]);
    setPolicyInstances(prev => ({
      ...prev,
      [policyType.policy_type_id]: []
    }));
  };

  const handlePolicyTypeDeleted = (policyTypeId) => {
    setPolicyTypes(prev => prev.filter(pt => pt.policy_type_id !== policyTypeId));
    setPolicyInstances(prev => {
      const newInstances = { ...prev };
      delete newInstances[policyTypeId];
      return newInstances;
    });
    if (selectedPolicyType?.policy_type_id === policyTypeId) {
      setSelectedPolicyType(null);
    }
  };

  const handlePolicyInstanceCreated = (policyTypeId, policyInstance) => {
    setPolicyInstances(prev => ({
      ...prev,
      [policyTypeId]: [...(prev[policyTypeId] || []), policyInstance]
    }));
  };

  const handlePolicyInstanceDeleted = (policyTypeId, policyInstanceId) => {
    setPolicyInstances(prev => ({
      ...prev,
      [policyTypeId]: (prev[policyTypeId] || []).filter(pi => pi.policy_instance_id !== policyInstanceId)
    }));
  };

  if (loading) {
    return <LoadingDisplay message="Loading A1 Policy Management..." />;
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={handleRefresh} />;
  }

  return (
    <div className="a1-policy-management">
      <div className="a1-header">
        <h2>A1 Policy Management</h2>
        <div className="a1-status">
          <span className={`status-indicator ${a1Health?.is_healthy ? 'healthy' : 'unhealthy'}`}>
            A1 Mediator: {a1Health?.is_healthy ? 'Healthy' : 'Unhealthy'}
          </span>
          <button onClick={handleRefresh} className="refresh-btn">
            Refresh
          </button>
        </div>
      </div>

      {a1Stats && (
        <div className="a1-stats">
          <div className="stat-item">
            <span className="stat-label">Policy Types:</span>
            <span className="stat-value">{a1Stats.total_policy_types}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Policy Instances:</span>
            <span className="stat-value">{a1Stats.total_policy_instances}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Last Updated:</span>
            <span className="stat-value">{new Date(a1Stats.last_updated).toLocaleString()}</span>
          </div>
        </div>
      )}

      <div className="a1-tabs">
        <button 
          className={`tab-btn ${activeTab === 'types' ? 'active' : ''}`}
          onClick={() => setActiveTab('types')}
        >
          Policy Types
        </button>
        <button 
          className={`tab-btn ${activeTab === 'instances' ? 'active' : ''}`}
          onClick={() => setActiveTab('instances')}
        >
          Policy Instances
        </button>
        <button 
          className={`tab-btn ${activeTab === 'status' ? 'active' : ''}`}
          onClick={() => setActiveTab('status')}
        >
          Status & Compliance
        </button>
        <button 
          className={`tab-btn ${activeTab === 'conflicts' ? 'active' : ''}`}
          onClick={() => setActiveTab('conflicts')}
        >
          Conflict Resolution
        </button>
      </div>

      <div className="a1-content">
        {activeTab === 'types' && (
          <PolicyTypeBrowser
            policyTypes={policyTypes}
            onPolicyTypeSelect={handlePolicyTypeSelect}
            onPolicyTypeCreated={handlePolicyTypeCreated}
            onPolicyTypeDeleted={handlePolicyTypeDeleted}
            onRefresh={handleRefresh}
          />
        )}

        {activeTab === 'instances' && (
          <PolicyInstanceManager
            policyTypes={policyTypes}
            policyInstances={policyInstances}
            selectedPolicyType={selectedPolicyType}
            onPolicyInstanceCreated={handlePolicyInstanceCreated}
            onPolicyInstanceDeleted={handlePolicyInstanceDeleted}
            onRefresh={handleRefresh}
          />
        )}

        {activeTab === 'status' && (
          <PolicyStatusDashboard
            policyTypes={policyTypes}
            policyInstances={policyInstances}
            a1Stats={a1Stats}
            onRefresh={handleRefresh}
          />
        )}

        {activeTab === 'conflicts' && (
          <PolicyConflictResolver
            policyTypes={policyTypes}
            policyInstances={policyInstances}
            onRefresh={handleRefresh}
          />
        )}
      </div>
    </div>
  );
};

export default A1PolicyManagement;