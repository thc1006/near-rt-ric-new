/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import XAppDeploymentForm from './XAppDeploymentForm';
import XAppList from './XAppList';
import XAppDetails from './XAppDetails';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import dashboardAPI from '../services/api';
import './XAppManagement.css';

/**
 * Main xApp management interface component
 * Provides comprehensive xApp lifecycle management including deployment, monitoring, and control
 */
const XAppManagement = ({ 
  xApps = [], 
  loading = false, 
  error = null, 
  onRefresh = null 
}) => {
  const [activeTab, setActiveTab] = useState('list');
  const [selectedXApp, setSelectedXApp] = useState(null);
  const [deploymentLoading, setDeploymentLoading] = useState(false);
  const [deploymentError, setDeploymentError] = useState(null);
  const [operationLoading, setOperationLoading] = useState({});
  const [operationError, setOperationError] = useState(null);
  const [showDeployForm, setShowDeployForm] = useState(false);

  // Reset errors when switching tabs
  useEffect(() => {
    setDeploymentError(null);
    setOperationError(null);
  }, [activeTab]);

  // Handle xApp deployment
  const handleDeploy = async (xappData) => {
    try {
      setDeploymentLoading(true);
      setDeploymentError(null);
      
      await dashboardAPI.deployXApp(xappData);
      
      // Refresh xApp list after successful deployment
      if (onRefresh) {
        await onRefresh();
      }
      
      setShowDeployForm(false);
      setActiveTab('list');
    } catch (err) {
      console.error('Failed to deploy xApp:', err);
      setDeploymentError(err);
    } finally {
      setDeploymentLoading(false);
    }
  };

  // Handle xApp undeployment
  const handleUndeploy = async (xappName) => {
    try {
      setOperationLoading(prev => ({ ...prev, [xappName]: 'undeploying' }));
      setOperationError(null);
      
      await dashboardAPI.undeployXApp(xappName);
      
      // Refresh xApp list after successful undeployment
      if (onRefresh) {
        await onRefresh();
      }
      
      // Clear selection if the undeployed xApp was selected
      if (selectedXApp && selectedXApp.name === xappName) {
        setSelectedXApp(null);
      }
    } catch (err) {
      console.error('Failed to undeploy xApp:', err);
      setOperationError(err);
    } finally {
      setOperationLoading(prev => {
        const newState = { ...prev };
        delete newState[xappName];
        return newState;
      });
    }
  };

  // Handle xApp restart (undeploy + redeploy)
  const handleRestart = async (xapp) => {
    try {
      setOperationLoading(prev => ({ ...prev, [xapp.name]: 'restarting' }));
      setOperationError(null);
      
      // First undeploy
      await dashboardAPI.undeployXApp(xapp.name);
      
      // Wait a moment for cleanup
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Then redeploy with same configuration
      const deploymentData = {
        name: xapp.name,
        version: xapp.version,
        configuration: xapp.configuration || {},
        resources: xapp.resources || {}
      };
      
      await dashboardAPI.deployXApp(deploymentData);
      
      // Refresh xApp list
      if (onRefresh) {
        await onRefresh();
      }
    } catch (err) {
      console.error('Failed to restart xApp:', err);
      setOperationError(err);
    } finally {
      setOperationLoading(prev => {
        const newState = { ...prev };
        delete newState[xapp.name];
        return newState;
      });
    }
  };

  // Handle xApp scaling (mock implementation - would need backend support)
  const handleScale = async (xappName, instances) => {
    try {
      setOperationLoading(prev => ({ ...prev, [xappName]: 'scaling' }));
      setOperationError(null);
      
      // This would need backend API support for scaling
      // For now, we'll simulate the operation
      console.log(`Scaling xApp ${xappName} to ${instances} instances`);
      
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Refresh xApp list
      if (onRefresh) {
        await onRefresh();
      }
    } catch (err) {
      console.error('Failed to scale xApp:', err);
      setOperationError(err);
    } finally {
      setOperationLoading(prev => {
        const newState = { ...prev };
        delete newState[xappName];
        return newState;
      });
    }
  };

  // Handle xApp selection for details view
  const handleSelectXApp = (xapp) => {
    setSelectedXApp(xapp);
    setActiveTab('details');
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'deploy':
        return (
          <div className="tab-content">
            <XAppDeploymentForm
              onDeploy={handleDeploy}
              loading={deploymentLoading}
              error={deploymentError}
              onCancel={() => {
                setShowDeployForm(false);
                setActiveTab('list');
              }}
            />
          </div>
        );
      
      case 'details':
        return (
          <div className="tab-content">
            {selectedXApp ? (
              <XAppDetails
                xapp={selectedXApp}
                onUndeploy={handleUndeploy}
                onRestart={handleRestart}
                onScale={handleScale}
                operationLoading={operationLoading[selectedXApp.name]}
                operationError={operationError}
                onBack={() => {
                  setSelectedXApp(null);
                  setActiveTab('list');
                }}
              />
            ) : (
              <div className="no-selection">
                <p>No xApp selected. Please select an xApp from the list.</p>
                <button 
                  className="btn btn-primary"
                  onClick={() => setActiveTab('list')}
                >
                  Back to List
                </button>
              </div>
            )}
          </div>
        );
      
      case 'list':
      default:
        return (
          <div className="tab-content">
            <XAppList
              xApps={xApps}
              loading={loading}
              error={error}
              onRefresh={onRefresh}
              onSelectXApp={handleSelectXApp}
              onUndeploy={handleUndeploy}
              onRestart={handleRestart}
              onScale={handleScale}
              operationLoading={operationLoading}
              operationError={operationError}
            />
          </div>
        );
    }
  };

  if (loading && xApps.length === 0) {
    return (
      <div className="xapp-management">
        <div className="management-header">
          <h2>xApp Management</h2>
        </div>
        <LoadingDisplay message="Loading xApp management interface..." />
      </div>
    );
  }

  if (error && xApps.length === 0) {
    return (
      <div className="xapp-management">
        <div className="management-header">
          <h2>xApp Management</h2>
        </div>
        <ErrorDisplay error={error} onRetry={onRefresh} />
      </div>
    );
  }

  return (
    <div className="xapp-management">
      <div className="management-header">
        <h2>xApp Management</h2>
        <div className="header-actions">
          <button 
            className="btn btn-primary"
            onClick={() => {
              setShowDeployForm(true);
              setActiveTab('deploy');
            }}
            disabled={deploymentLoading}
          >
            {deploymentLoading ? 'Deploying...' : 'Deploy New xApp'}
          </button>
          {onRefresh && (
            <button 
              className="btn btn-secondary"
              onClick={onRefresh}
              disabled={loading}
            >
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>
          )}
        </div>
      </div>

      <div className="management-tabs">
        <button 
          className={`tab-button ${activeTab === 'list' ? 'active' : ''}`}
          onClick={() => setActiveTab('list')}
        >
          xApp List ({xApps.length})
        </button>
        {selectedXApp && (
          <button 
            className={`tab-button ${activeTab === 'details' ? 'active' : ''}`}
            onClick={() => setActiveTab('details')}
          >
            {selectedXApp.name} Details
          </button>
        )}
        {showDeployForm && (
          <button 
            className={`tab-button ${activeTab === 'deploy' ? 'active' : ''}`}
            onClick={() => setActiveTab('deploy')}
          >
            Deploy xApp
          </button>
        )}
      </div>

      {renderTabContent()}
    </div>
  );
};

export default XAppManagement;