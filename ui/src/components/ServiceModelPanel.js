/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './ServiceModelPanel.css';

/**
 * Service Model Panel component
 * Displays O-RAN service models and their capabilities
 */
const ServiceModelPanel = () => {
  const [serviceModels, setServiceModels] = useState([]);
  const [capabilities, setCapabilities] = useState([]);
  const [stats, setStats] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedModel, setSelectedModel] = useState(null);
  const [activeTab, setActiveTab] = useState('models');

  // Fetch service models from the API
  const fetchServiceModels = async () => {
    try {
      const response = await fetch('/api/v1/servicemodels');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setServiceModels(data.serviceModels || []);
    } catch (err) {
      console.error('Failed to fetch service models:', err);
      setError(err.message);
    }
  };

  // Fetch service model capabilities
  const fetchCapabilities = async () => {
    try {
      const response = await fetch('/api/v1/servicemodels/capabilities');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setCapabilities(data.capabilities || []);
    } catch (err) {
      console.error('Failed to fetch capabilities:', err);
      setError(err.message);
    }
  };

  // Fetch service model statistics
  const fetchStats = async () => {
    try {
      const response = await fetch('/api/v1/servicemodels/stats');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setStats(data.stats || {});
    } catch (err) {
      console.error('Failed to fetch stats:', err);
      setError(err.message);
    }
  };

  // Fetch detailed service model information
  const fetchServiceModelDetails = async (oid) => {
    try {
      const response = await fetch(`/api/v1/servicemodels/${encodeURIComponent(oid)}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const modelData = await response.json();
      setSelectedModel(modelData);
    } catch (err) {
      console.error('Failed to fetch service model details:', err);
      setError(err.message);
    }
  };

  // Load all data
  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      await Promise.all([
        fetchServiceModels(),
        fetchCapabilities(),
        fetchStats()
      ]);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const getServiceModelTypeColor = (type) => {
    switch (type) {
      case 'E2SM-KPM':
        return 'type-kpm';
      case 'E2SM-RC':
        return 'type-rc';
      case 'E2SM-NI':
        return 'type-ni';
      default:
        return 'type-unknown';
    }
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleString();
  };

  if (loading) {
    return (
      <div className="service-model-panel">
        <div className="panel-header">
          <h3>Service Models</h3>
        </div>
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Loading service models...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="service-model-panel">
        <div className="panel-header">
          <h3>Service Models</h3>
          <button onClick={loadData} className="refresh-btn">
            Refresh
          </button>
        </div>
        <div className="error-state">
          <p>Error loading service models: {error}</p>
          <button onClick={loadData} className="retry-btn">
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="service-model-panel">
      <div className="panel-header">
        <h3>O-RAN Service Models</h3>
        <div className="header-controls">
          <button onClick={loadData} className="refresh-btn">
            Refresh
          </button>
        </div>
      </div>

      <div className="tab-navigation">
        <button 
          className={`tab-btn ${activeTab === 'models' ? 'active' : ''}`}
          onClick={() => setActiveTab('models')}
        >
          Service Models ({serviceModels.length})
        </button>
        <button 
          className={`tab-btn ${activeTab === 'capabilities' ? 'active' : ''}`}
          onClick={() => setActiveTab('capabilities')}
        >
          Capabilities ({capabilities.length})
        </button>
        <button 
          className={`tab-btn ${activeTab === 'stats' ? 'active' : ''}`}
          onClick={() => setActiveTab('stats')}
        >
          Statistics
        </button>
      </div>

      <div className="tab-content">
        {activeTab === 'models' && (
          <div className="models-tab">
            {serviceModels.length === 0 ? (
              <div className="empty-state">
                <p>No service models available</p>
              </div>
            ) : (
              <div className="models-container">
                <div className="models-list">
                  {serviceModels.map((model) => (
                    <div 
                      key={model.oid} 
                      className={`model-card ${selectedModel?.oid === model.oid ? 'selected' : ''}`}
                      onClick={() => fetchServiceModelDetails(model.oid)}
                    >
                      <div className="model-header">
                        <div className="model-name">{model.name}</div>
                        <div className={`model-type ${getServiceModelTypeColor(model.type)}`}>
                          {model.type}
                        </div>
                      </div>
                      <div className="model-details">
                        <div className="model-version">Version: {model.version}</div>
                        <div className="model-oid">{model.oid}</div>
                        <div className="model-description">{model.description}</div>
                        <div className="model-stats">
                          <span>{model.ranFunctions?.length || 0} RAN Functions</span>
                          <span>{model.capabilities?.length || 0} Capabilities</span>
                        </div>
                      </div>
                      <div className="model-footer">
                        <div className="last-updated">
                          Updated: {formatTimestamp(model.lastUpdated)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                {selectedModel && (
                  <div className="model-details-panel">
                    <div className="details-header">
                      <h4>{selectedModel.name} Details</h4>
                      <button 
                        onClick={() => setSelectedModel(null)}
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
                            <span className="label">Name:</span>
                            <span className="value">{selectedModel.name}</span>
                          </div>
                          <div className="detail-item">
                            <span className="label">Type:</span>
                            <span className={`value ${getServiceModelTypeColor(selectedModel.type)}`}>
                              {selectedModel.type}
                            </span>
                          </div>
                          <div className="detail-item">
                            <span className="label">Version:</span>
                            <span className="value">{selectedModel.version}</span>
                          </div>
                          <div className="detail-item">
                            <span className="label">OID:</span>
                            <span className="value oid">{selectedModel.oid}</span>
                          </div>
                        </div>
                        <div className="description">
                          <h6>Description:</h6>
                          <p>{selectedModel.description}</p>
                        </div>
                      </div>

                      {selectedModel.capabilities && selectedModel.capabilities.length > 0 && (
                        <div className="detail-section">
                          <h5>Capabilities ({selectedModel.capabilities.length})</h5>
                          <div className="capabilities-list">
                            {selectedModel.capabilities.map((capability, index) => (
                              <div key={index} className="capability-item">
                                <div className="capability-header">
                                  <span className="capability-name">{capability.name}</span>
                                  <span className={`capability-status ${capability.supported ? 'supported' : 'not-supported'}`}>
                                    {capability.supported ? 'Supported' : 'Not Supported'}
                                  </span>
                                </div>
                                <div className="capability-description">{capability.description}</div>
                                <div className="capability-version">Version: {capability.version}</div>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {selectedModel.ranFunctions && selectedModel.ranFunctions.length > 0 && (
                        <div className="detail-section">
                          <h5>RAN Functions ({selectedModel.ranFunctions.length})</h5>
                          <div className="functions-list">
                            {selectedModel.ranFunctions.map((func) => (
                              <div key={func.id} className="function-item">
                                <div className="function-header">
                                  <span className="function-id">ID: {func.id}</span>
                                  <span className="function-revision">Rev: {func.revision}</span>
                                </div>
                                <div className="function-oid">{func.oid}</div>
                                <div className="function-description">{func.description}</div>
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
          </div>
        )}

        {activeTab === 'capabilities' && (
          <div className="capabilities-tab">
            {capabilities.length === 0 ? (
              <div className="empty-state">
                <p>No capabilities available</p>
              </div>
            ) : (
              <div className="capabilities-grid">
                {capabilities.map((capability, index) => (
                  <div key={index} className="capability-card">
                    <div className="capability-header">
                      <h4>{capability.name}</h4>
                      <span className={`status ${capability.supported ? 'supported' : 'not-supported'}`}>
                        {capability.supported ? 'Supported' : 'Not Supported'}
                      </span>
                    </div>
                    <div className="capability-content">
                      <p>{capability.description}</p>
                      <div className="capability-meta">
                        <span>Version: {capability.version}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'stats' && (
          <div className="stats-tab">
            <div className="stats-grid">
              <div className="stat-card">
                <h4>Total Models</h4>
                <div className="stat-value">{stats.total_models || 0}</div>
              </div>
              <div className="stat-card">
                <h4>Total Functions</h4>
                <div className="stat-value">{stats.total_functions || 0}</div>
              </div>
              <div className="stat-card">
                <h4>Total Capabilities</h4>
                <div className="stat-value">{stats.total_capabilities || 0}</div>
              </div>
            </div>

            {stats.models_by_type && (
              <div className="models-by-type">
                <h4>Models by Type</h4>
                <div className="type-stats">
                  {Object.entries(stats.models_by_type).map(([type, count]) => (
                    <div key={type} className="type-stat">
                      <span className={`type-label ${getServiceModelTypeColor(type)}`}>
                        {type}
                      </span>
                      <span className="type-count">{count}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ServiceModelPanel;