/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState } from 'react';
import './PolicyTypeBrowser.css';
import { ErrorDisplay } from './ErrorDisplay';
import PolicyTypeForm from './PolicyTypeForm';
import PolicySchemaViewer from './PolicySchemaViewer';
import dashboardAPI from '../services/api';

const PolicyTypeBrowser = ({ 
  policyTypes, 
  onPolicyTypeSelect, 
  onPolicyTypeCreated, 
  onPolicyTypeDeleted, 
  onRefresh 
}) => {
  const [selectedPolicyType, setSelectedPolicyType] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [showSchemaViewer, setShowSchemaViewer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');

  const filteredPolicyTypes = policyTypes.filter(pt => 
    pt.name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    pt.policy_type_id?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    pt.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handlePolicyTypeClick = (policyType) => {
    setSelectedPolicyType(policyType);
    setShowSchemaViewer(true);
  };

  const handleSelectForInstances = (policyType) => {
    onPolicyTypeSelect(policyType);
  };

  const handleDeletePolicyType = async (policyTypeId) => {
    if (!window.confirm(`Are you sure you want to delete policy type "${policyTypeId}"? This will also delete all associated policy instances.`)) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await dashboardAPI.deletePolicyType(policyTypeId);
      onPolicyTypeDeleted(policyTypeId);
      
      if (selectedPolicyType?.policy_type_id === policyTypeId) {
        setSelectedPolicyType(null);
        setShowSchemaViewer(false);
      }
    } catch (err) {
      console.error('Failed to delete policy type:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePolicyType = async (policyTypeData) => {
    setLoading(true);
    setError(null);

    try {
      await dashboardAPI.createPolicyType(policyTypeData.policy_type_id, {
        name: policyTypeData.name,
        description: policyTypeData.description,
        policy_type_schema: policyTypeData.schema
      });

      // Fetch the created policy type to get complete data
      const createdPolicyType = await dashboardAPI.getPolicyType(policyTypeData.policy_type_id);
      onPolicyTypeCreated(createdPolicyType);
      setShowCreateForm(false);
    } catch (err) {
      console.error('Failed to create policy type:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="policy-type-browser">
      <div className="browser-header">
        <div className="search-section">
          <input
            type="text"
            placeholder="Search policy types..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="search-input"
          />
          <button 
            onClick={() => setShowCreateForm(true)}
            className="create-btn"
            disabled={loading}
          >
            Create Policy Type
          </button>
        </div>
      </div>

      {error && (
        <ErrorDisplay 
          error={error} 
          onRetry={() => {
            setError(null);
            onRefresh();
          }} 
        />
      )}

      <div className="policy-types-grid">
        {filteredPolicyTypes.length === 0 ? (
          <div className="no-policy-types">
            <p>No policy types found</p>
            {searchTerm && (
              <p>Try adjusting your search criteria</p>
            )}
          </div>
        ) : (
          filteredPolicyTypes.map(policyType => (
            <div key={policyType.policy_type_id} className="policy-type-card">
              <div className="card-header">
                <h3 className="policy-type-id">{policyType.policy_type_id}</h3>
                <div className="card-actions">
                  <button
                    onClick={() => handlePolicyTypeClick(policyType)}
                    className="view-schema-btn"
                    title="View Schema"
                  >
                    📋
                  </button>
                  <button
                    onClick={() => handleSelectForInstances(policyType)}
                    className="manage-instances-btn"
                    title="Manage Instances"
                  >
                    ⚙️
                  </button>
                  <button
                    onClick={() => handleDeletePolicyType(policyType.policy_type_id)}
                    className="delete-btn"
                    title="Delete Policy Type"
                    disabled={loading}
                  >
                    🗑️
                  </button>
                </div>
              </div>
              
              <div className="card-content">
                <h4 className="policy-name">{policyType.name || 'Unnamed Policy Type'}</h4>
                <p className="policy-description">
                  {policyType.description || 'No description available'}
                </p>
                <div className="policy-meta">
                  <span className="created-date">
                    Created: {new Date(policyType.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {showCreateForm && (
        <PolicyTypeForm
          onSubmit={handleCreatePolicyType}
          onCancel={() => {
            setShowCreateForm(false);
            setError(null);
          }}
          loading={loading}
          error={error}
        />
      )}

      {showSchemaViewer && selectedPolicyType && (
        <PolicySchemaViewer
          policyType={selectedPolicyType}
          onClose={() => {
            setShowSchemaViewer(false);
            setSelectedPolicyType(null);
          }}
        />
      )}
    </div>
  );
};

export default PolicyTypeBrowser;