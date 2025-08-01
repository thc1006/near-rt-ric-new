/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './E2NodesPanel.css';

/**
 * E2 Nodes Panel component
 * Displays real-time E2 node information from E2 Manager
 */
const E2NodesPanel = () => {
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedNode, setSelectedNode] = useState(null);
  const [refreshInterval, setRefreshInterval] = useState(null);

  // Fetch E2 nodes from the API
  const fetchE2Nodes = async () => {
    try {
      const response = await fetch('/api/v1/e2nodes');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setNodes(data.nodes || []);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch E2 nodes:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Fetch detailed node information
  const fetchNodeDetails = async (nodeId) => {
    try {
      const response = await fetch(`/api/v1/e2nodes/${nodeId}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const nodeData = await response.json();
      setSelectedNode(nodeData);
    } catch (err) {
      console.error('Failed to fetch node details:', err);
      setError(err.message);
    }
  };

  // Start auto-refresh
  const startAutoRefresh = () => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
    const interval = setInterval(fetchE2Nodes, 10000); // Refresh every 10 seconds
    setRefreshInterval(interval);
  };

  // Stop auto-refresh
  const stopAutoRefresh = () => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
      setRefreshInterval(null);
    }
  };

  useEffect(() => {
    fetchE2Nodes();
    startAutoRefresh();

    return () => {
      stopAutoRefresh();
    };
  }, []);

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'connected':
      case 'associated':
        return 'status-connected';
      case 'disconnected':
        return 'status-disconnected';
      case 'connecting':
        return 'status-connecting';
      case 'setup_failed':
        return 'status-failed';
      default:
        return 'status-unknown';
    }
  };

  const getNodeTypeDisplay = (type) => {
    switch (type) {
      case 'GNB':
        return 'gNodeB';
      case 'ENB':
        return 'eNodeB';
      case 'O_CU':
        return 'O-CU';
      case 'O_DU':
        return 'O-DU';
      case 'O_CU_CP':
        return 'O-CU-CP';
      case 'O_CU_UP':
        return 'O-CU-UP';
      default:
        return type || 'Unknown';
    }
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleString();
  };

  if (loading) {
    return (
      <div className="e2-nodes-panel">
        <div className="panel-header">
          <h3>E2 Nodes</h3>
        </div>
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Loading E2 nodes...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="e2-nodes-panel">
        <div className="panel-header">
          <h3>E2 Nodes</h3>
          <button onClick={fetchE2Nodes} className="refresh-btn">
            Refresh
          </button>
        </div>
        <div className="error-state">
          <p>Error loading E2 nodes: {error}</p>
          <button onClick={fetchE2Nodes} className="retry-btn">
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="e2-nodes-panel">
      <div className="panel-header">
        <h3>E2 Nodes ({nodes.length})</h3>
        <div className="header-controls">
          <button 
            onClick={fetchE2Nodes} 
            className="refresh-btn"
            disabled={loading}
          >
            Refresh
          </button>
          <button 
            onClick={refreshInterval ? stopAutoRefresh : startAutoRefresh}
            className={`auto-refresh-btn ${refreshInterval ? 'active' : ''}`}
          >
            {refreshInterval ? 'Stop Auto-Refresh' : 'Start Auto-Refresh'}
          </button>
        </div>
      </div>

      {nodes.length === 0 ? (
        <div className="empty-state">
          <p>No E2 nodes discovered</p>
          <p className="empty-subtitle">
            E2 nodes will appear here when they connect to the E2 Termination component
          </p>
        </div>
      ) : (
        <div className="nodes-container">
          <div className="nodes-list">
            {nodes.map((node) => (
              <div 
                key={node.id} 
                className={`node-card ${selectedNode?.id === node.id ? 'selected' : ''}`}
                onClick={() => fetchNodeDetails(node.id)}
              >
                <div className="node-header">
                  <div className="node-id">{node.id}</div>
                  <div className={`node-status ${getStatusColor(node.connectionStatus)}`}>
                    {node.connectionStatus || 'Unknown'}
                  </div>
                </div>
                <div className="node-details">
                  <div className="node-type">
                    {getNodeTypeDisplay(node.globalE2NodeId?.type)}
                  </div>
                  <div className="node-address">
                    {node.ipAddress}:{node.port}
                  </div>
                  <div className="node-functions">
                    {node.ranFunctions?.length || 0} RAN Functions
                  </div>
                  <div className="node-subscriptions">
                    {node.subscriptions?.length || 0} Subscriptions
                  </div>
                </div>
                <div className="node-footer">
                  <div className="last-update">
                    Updated: {formatTimestamp(node.lastUpdate)}
                  </div>
                </div>
              </div>
            ))}
          </div>

          {selectedNode && (
            <div className="node-details-panel">
              <div className="details-header">
                <h4>Node Details: {selectedNode.id}</h4>
                <button 
                  onClick={() => setSelectedNode(null)}
                  className="close-btn"
                >
                  ×
                </button>
              </div>
              
              <div className="details-content">
                <div className="detail-section">
                  <h5>Connection Information</h5>
                  <div className="detail-grid">
                    <div className="detail-item">
                      <span className="label">Status:</span>
                      <span className={`value ${getStatusColor(selectedNode.connectionStatus)}`}>
                        {selectedNode.connectionStatus}
                      </span>
                    </div>
                    <div className="detail-item">
                      <span className="label">IP Address:</span>
                      <span className="value">{selectedNode.ipAddress}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Port:</span>
                      <span className="value">{selectedNode.port}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">SCTP Streams:</span>
                      <span className="value">{selectedNode.sctpStreams || 'N/A'}</span>
                    </div>
                  </div>
                </div>

                <div className="detail-section">
                  <h5>Global E2 Node ID</h5>
                  <div className="detail-grid">
                    <div className="detail-item">
                      <span className="label">PLMN ID:</span>
                      <span className="value">{selectedNode.globalE2NodeId?.plmnId || 'N/A'}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Node ID:</span>
                      <span className="value">{selectedNode.globalE2NodeId?.nodeId || 'N/A'}</span>
                    </div>
                    <div className="detail-item">
                      <span className="label">Type:</span>
                      <span className="value">
                        {getNodeTypeDisplay(selectedNode.globalE2NodeId?.type)}
                      </span>
                    </div>
                  </div>
                </div>

                {selectedNode.ranFunctions && selectedNode.ranFunctions.length > 0 && (
                  <div className="detail-section">
                    <h5>RAN Functions ({selectedNode.ranFunctions.length})</h5>
                    <div className="functions-list">
                      {selectedNode.ranFunctions.map((func) => (
                        <div key={func.id} className="function-item">
                          <div className="function-header">
                            <span className="function-id">ID: {func.id}</span>
                            <span className="function-revision">Rev: {func.revision}</span>
                          </div>
                          <div className="function-oid">{func.oid}</div>
                          {func.description && (
                            <div className="function-description">{func.description}</div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {selectedNode.serviceModels && selectedNode.serviceModels.length > 0 && (
                  <div className="detail-section">
                    <h5>Service Models ({selectedNode.serviceModels.length})</h5>
                    <div className="service-models-list">
                      {selectedNode.serviceModels.map((model, index) => (
                        <div key={index} className="service-model-item">
                          <div className="model-name">{model.name}</div>
                          <div className="model-version">Version: {model.version}</div>
                          <div className="model-oid">{model.oid}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {selectedNode.subscriptions && selectedNode.subscriptions.length > 0 && (
                  <div className="detail-section">
                    <h5>Active Subscriptions ({selectedNode.subscriptions.length})</h5>
                    <div className="subscriptions-list">
                      {selectedNode.subscriptions.map((sub) => (
                        <div key={sub.id} className="subscription-item">
                          <div className="subscription-header">
                            <span className="subscription-id">{sub.id}</span>
                            <span className="subscription-status">{sub.status}</span>
                          </div>
                          <div className="subscription-details">
                            <span>xApp: {sub.xappId}</span>
                            <span>RAN Function: {sub.ranFunctionId}</span>
                            <span>Created: {formatTimestamp(sub.createdAt)}</span>
                          </div>
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
  );
};

export default E2NodesPanel;