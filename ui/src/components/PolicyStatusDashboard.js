/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './PolicyStatusDashboard.css';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';
import dashboardAPI from '../services/api';

const PolicyStatusDashboard = ({ policyTypes, policyInstances, a1Stats, onRefresh }) => {
  const [detailedStats, setDetailedStats] = useState(null);
  const [distributionStatus, setDistributionStatus] = useState([]);
  const [complianceReports, setComplianceReports] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [selectedTimeRange, setSelectedTimeRange] = useState('24h');

  useEffect(() => {
    loadDetailedData();
  }, [policyTypes, policyInstances]);

  const loadDetailedData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Load detailed statistics
      const stats = await dashboardAPI.getA1Stats();
      setDetailedStats(stats);

      // Simulate loading distribution status and compliance data
      // In a real implementation, these would be separate API calls
      const mockDistributionStatus = [];
      const mockComplianceReports = [];

      // Generate mock data based on existing policy instances
      Object.entries(policyInstances).forEach(([policyTypeId, instances]) => {
        instances.forEach(instance => {
          // Mock distribution status
          mockDistributionStatus.push({
            policy_instance_id: instance.policy_instance_id,
            xapp_id: 'hello-world-xapp',
            status: Math.random() > 0.2 ? 'DEPLOYED' : 'FAILED',
            message: Math.random() > 0.2 ? 'Successfully deployed' : 'Deployment failed',
            last_update: new Date(Date.now() - Math.random() * 86400000).toISOString()
          });

          // Mock compliance reports
          mockComplianceReports.push({
            policy_instance_id: instance.policy_instance_id,
            xapp_id: 'hello-world-xapp',
            compliance_status: Math.random() > 0.3 ? 'COMPLIANT' : 'NON_COMPLIANT',
            violations: Math.random() > 0.3 ? [] : ['Resource limit exceeded', 'Invalid parameter value'],
            last_check: new Date(Date.now() - Math.random() * 3600000).toISOString()
          });
        });
      });

      setDistributionStatus(mockDistributionStatus);
      setComplianceReports(mockComplianceReports);

    } catch (err) {
      console.error('Failed to load detailed policy data:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const calculateStatusCounts = () => {
    const counts = {
      total: 0,
      active: 0,
      inactive: 0,
      pending: 0,
      error: 0
    };

    Object.values(policyInstances).forEach(instances => {
      instances.forEach(instance => {
        counts.total++;
        const status = instance.status?.status?.toLowerCase() || 'unknown';
        if (counts[status] !== undefined) {
          counts[status]++;
        }
      });
    });

    return counts;
  };

  const calculateDistributionStats = () => {
    const stats = {
      deployed: 0,
      pending: 0,
      failed: 0,
      withdrawn: 0
    };

    distributionStatus.forEach(status => {
      const statusKey = status.status.toLowerCase();
      if (stats[statusKey] !== undefined) {
        stats[statusKey]++;
      }
    });

    return stats;
  };

  const calculateComplianceStats = () => {
    const stats = {
      compliant: 0,
      nonCompliant: 0,
      unknown: 0
    };

    complianceReports.forEach(report => {
      const statusKey = report.compliance_status.toLowerCase().replace('_', '');
      if (statusKey === 'noncompliant') {
        stats.nonCompliant++;
      } else if (stats[statusKey] !== undefined) {
        stats[statusKey]++;
      }
    });

    return stats;
  };

  const getStatusColor = (status, type = 'instance') => {
    const colorMaps = {
      instance: {
        active: '#28a745',
        inactive: '#6c757d',
        pending: '#ffc107',
        error: '#dc3545'
      },
      distribution: {
        deployed: '#28a745',
        pending: '#ffc107',
        failed: '#dc3545',
        withdrawn: '#6c757d'
      },
      compliance: {
        compliant: '#28a745',
        nonCompliant: '#dc3545',
        unknown: '#6c757d'
      }
    };

    return colorMaps[type][status.toLowerCase()] || '#6c757d';
  };

  const statusCounts = calculateStatusCounts();
  const distributionStats = calculateDistributionStats();
  const complianceStats = calculateComplianceStats();

  if (loading) {
    return <LoadingDisplay message="Loading policy status dashboard..." />;
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={loadDetailedData} />;
  }

  return (
    <div className="policy-status-dashboard">
      <div className="dashboard-header">
        <h3>Policy Status & Compliance Dashboard</h3>
        <div className="dashboard-controls">
          <select
            value={selectedTimeRange}
            onChange={(e) => setSelectedTimeRange(e.target.value)}
            className="time-range-select"
          >
            <option value="1h">Last Hour</option>
            <option value="24h">Last 24 Hours</option>
            <option value="7d">Last 7 Days</option>
            <option value="30d">Last 30 Days</option>
          </select>
          <button onClick={onRefresh} className="refresh-btn">
            Refresh
          </button>
        </div>
      </div>

      <div className="stats-grid">
        {/* Policy Instance Status */}
        <div className="stats-card">
          <h4>Policy Instance Status</h4>
          <div className="stats-content">
            <div className="stat-item large">
              <span className="stat-value">{statusCounts.total}</span>
              <span className="stat-label">Total Instances</span>
            </div>
            <div className="status-breakdown">
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('active') }}
                ></span>
                <span className="status-count">{statusCounts.active}</span>
                <span className="status-label">Active</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('pending') }}
                ></span>
                <span className="status-count">{statusCounts.pending}</span>
                <span className="status-label">Pending</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('error') }}
                ></span>
                <span className="status-count">{statusCounts.error}</span>
                <span className="status-label">Error</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('inactive') }}
                ></span>
                <span className="status-count">{statusCounts.inactive}</span>
                <span className="status-label">Inactive</span>
              </div>
            </div>
          </div>
        </div>

        {/* Distribution Status */}
        <div className="stats-card">
          <h4>Distribution Status</h4>
          <div className="stats-content">
            <div className="stat-item large">
              <span className="stat-value">{distributionStatus.length}</span>
              <span className="stat-label">Total Distributions</span>
            </div>
            <div className="status-breakdown">
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('deployed', 'distribution') }}
                ></span>
                <span className="status-count">{distributionStats.deployed}</span>
                <span className="status-label">Deployed</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('pending', 'distribution') }}
                ></span>
                <span className="status-count">{distributionStats.pending}</span>
                <span className="status-label">Pending</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('failed', 'distribution') }}
                ></span>
                <span className="status-count">{distributionStats.failed}</span>
                <span className="status-label">Failed</span>
              </div>
            </div>
          </div>
        </div>

        {/* Compliance Status */}
        <div className="stats-card">
          <h4>Compliance Status</h4>
          <div className="stats-content">
            <div className="stat-item large">
              <span className="stat-value">{complianceReports.length}</span>
              <span className="stat-label">Total Reports</span>
            </div>
            <div className="status-breakdown">
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('compliant', 'compliance') }}
                ></span>
                <span className="status-count">{complianceStats.compliant}</span>
                <span className="status-label">Compliant</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('nonCompliant', 'compliance') }}
                ></span>
                <span className="status-count">{complianceStats.nonCompliant}</span>
                <span className="status-label">Non-Compliant</span>
              </div>
              <div className="status-item">
                <span 
                  className="status-indicator"
                  style={{ backgroundColor: getStatusColor('unknown', 'compliance') }}
                ></span>
                <span className="status-count">{complianceStats.unknown}</span>
                <span className="status-label">Unknown</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Detailed Tables */}
      <div className="details-section">
        <div className="details-card">
          <h4>Distribution Status Details</h4>
          <div className="table-container">
            <table className="status-table">
              <thead>
                <tr>
                  <th>Policy Instance</th>
                  <th>xApp</th>
                  <th>Status</th>
                  <th>Message</th>
                  <th>Last Update</th>
                </tr>
              </thead>
              <tbody>
                {distributionStatus.slice(0, 10).map((status, index) => (
                  <tr key={index}>
                    <td className="policy-id">{status.policy_instance_id}</td>
                    <td>{status.xapp_id}</td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(status.status, 'distribution') }}
                      >
                        {status.status}
                      </span>
                    </td>
                    <td className="message-cell">{status.message}</td>
                    <td>{new Date(status.last_update).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="details-card">
          <h4>Compliance Reports</h4>
          <div className="table-container">
            <table className="status-table">
              <thead>
                <tr>
                  <th>Policy Instance</th>
                  <th>xApp</th>
                  <th>Compliance</th>
                  <th>Violations</th>
                  <th>Last Check</th>
                </tr>
              </thead>
              <tbody>
                {complianceReports.slice(0, 10).map((report, index) => (
                  <tr key={index}>
                    <td className="policy-id">{report.policy_instance_id}</td>
                    <td>{report.xapp_id}</td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(report.compliance_status, 'compliance') }}
                      >
                        {report.compliance_status.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="violations-cell">
                      {report.violations.length > 0 ? (
                        <ul>
                          {report.violations.map((violation, vIndex) => (
                            <li key={vIndex}>{violation}</li>
                          ))}
                        </ul>
                      ) : (
                        <span className="no-violations">None</span>
                      )}
                    </td>
                    <td>{new Date(report.last_check).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PolicyStatusDashboard;