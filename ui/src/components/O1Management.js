/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './O1Management.css';
import ConfigurationManager from './ConfigurationManager';
import AlarmManager from './AlarmManager';
import KPIMonitoring from './KPIMonitoring';
import UserManagement from './UserManagement';
import dashboardAPI from '../services/api';

const O1Management = () => {
  const [activeTab, setActiveTab] = useState('configuration');
  const [o1Health, setO1Health] = useState(null);
  const [o1Stats, setO1Stats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    loadO1Data();
    const interval = setInterval(loadO1Data, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const loadO1Data = async () => {
    try {
      setLoading(true);
      const [healthData, statsData] = await Promise.all([
        dashboardAPI.getO1Health(),
        dashboardAPI.getO1Stats()
      ]);
      
      setO1Health(healthData);
      setO1Stats(statsData);
      setError(null);
    } catch (err) {
      console.error('Failed to load O1 data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'configuration':
        return <ConfigurationManager />;
      case 'alarms':
        return <AlarmManager />;
      case 'kpis':
        return <KPIMonitoring />;
      case 'users':
        return <UserManagement />;
      default:
        return <ConfigurationManager />;
    }
  };

  if (loading && !o1Health) {
    return (
      <div className="o1-management">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading O1 Management Interface...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="o1-management">
      <div className="o1-header">
        <div className="o1-title">
          <h2>O1 Management Interface</h2>
          <div className="o1-health-indicator">
            <span className={`health-status ${o1Health?.is_healthy ? 'healthy' : 'unhealthy'}`}>
              {o1Health?.is_healthy ? '●' : '●'}
            </span>
            <span className="health-text">
              {o1Health?.is_healthy ? 'Healthy' : 'Unhealthy'}
            </span>
            {o1Health?.version && (
              <span className="version">v{o1Health.version}</span>
            )}
          </div>
        </div>
        
        {o1Stats && (
          <div className="o1-stats-summary">
            <div className="stat-item">
              <span className="stat-value">{o1Stats.total_managed_objects}</span>
              <span className="stat-label">Managed Objects</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">{o1Stats.total_configurations}</span>
              <span className="stat-label">Configurations</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">{o1Stats.total_active_alarms}</span>
              <span className="stat-label">Active Alarms</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">{o1Stats.total_kpis}</span>
              <span className="stat-label">KPIs</span>
            </div>
          </div>
        )}
      </div>

      {error && (
        <div className="error-banner">
          <span className="error-icon">⚠</span>
          <span className="error-message">{error}</span>
          <button className="retry-button" onClick={loadO1Data}>
            Retry
          </button>
        </div>
      )}

      <div className="o1-tabs">
        <button
          className={`tab-button ${activeTab === 'configuration' ? 'active' : ''}`}
          onClick={() => setActiveTab('configuration')}
        >
          Configuration Management
        </button>
        <button
          className={`tab-button ${activeTab === 'alarms' ? 'active' : ''}`}
          onClick={() => setActiveTab('alarms')}
        >
          Alarm & Event Management
        </button>
        <button
          className={`tab-button ${activeTab === 'kpis' ? 'active' : ''}`}
          onClick={() => setActiveTab('kpis')}
        >
          Performance Monitoring
        </button>
        <button
          className={`tab-button ${activeTab === 'users' ? 'active' : ''}`}
          onClick={() => setActiveTab('users')}
        >
          User Management
        </button>
      </div>

      <div className="o1-content">
        {renderTabContent()}
      </div>
    </div>
  );
};

export default O1Management;