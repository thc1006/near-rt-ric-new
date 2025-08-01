/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './PolicyConflictResolver.css';
import { ErrorDisplay, LoadingDisplay } from './ErrorDisplay';

const PolicyConflictResolver = ({ policyTypes, policyInstances, onRefresh }) => {
  const [conflicts, setConflicts] = useState([]);
  const [selectedConflict, setSelectedConflict] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [resolutionStrategy, setResolutionStrategy] = useState('priority');

  useEffect(() => {
    detectConflicts();
  }, [policyTypes, policyInstances]);

  const detectConflicts = () => {
    setLoading(true);
    setError(null);

    try {
      const detectedConflicts = [];
      const allInstances = [];

      // Collect all policy instances
      Object.entries(policyInstances).forEach(([policyTypeId, instances]) => {
        instances.forEach(instance => {
          allInstances.push({
            ...instance,
            policy_type_id: policyTypeId
          });
        });
      });

      // Simulate conflict detection logic
      for (let i = 0; i < allInstances.length; i++) {
        for (let j = i + 1; j < allInstances.length; j++) {
          const instance1 = allInstances[i];
          const instance2 = allInstances[j];

          // Check for resource conflicts
          const conflict = checkForConflict(instance1, instance2);
          if (conflict) {
            detectedConflicts.push({
              conflict_id: `conflict-${Date.now()}-${i}-${j}`,
              policy_instance_id: instance1.policy_instance_id,
              conflicting_policy_id: instance2.policy_instance_id,
              conflict_type: conflict.type,
              description: conflict.description,
              severity: conflict.severity,
              detected_at: new Date().toISOString(),
              resolution: null,
              status: 'UNRESOLVED'
            });
          }
        }
      }

      setConflicts(detectedConflicts);
    } catch (err) {
      console.error('Failed to detect conflicts:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const checkForConflict = (instance1, instance2) => {
    try {
      const policy1 = typeof instance1.policy === 'string' 
        ? JSON.parse(instance1.policy) 
        : instance1.policy;
      const policy2 = typeof instance2.policy === 'string' 
        ? JSON.parse(instance2.policy) 
        : instance2.policy;

      // Check for resource conflicts
      if (policy1.scope && policy2.scope) {
        // Same UE ID conflict
        if (policy1.scope.ueId === policy2.scope.ueId) {
          // Check for conflicting QoS parameters
          if (policy1.statement?.priorityLevel && policy2.statement?.priorityLevel) {
            if (policy1.statement.priorityLevel === policy2.statement.priorityLevel) {
              return {
                type: 'RESOURCE',
                description: `Both policies target the same UE (${policy1.scope.ueId}) with the same priority level (${policy1.statement.priorityLevel})`,
                severity: 'HIGH'
              };
            }
          }

          // Check for bandwidth conflicts
          if (policy1.statement?.qosParameters?.maxBitrate && 
              policy2.statement?.qosParameters?.maxBitrate) {
            const total = policy1.statement.qosParameters.maxBitrate + 
                         policy2.statement.qosParameters.maxBitrate;
            if (total > 100000) { // Assume 100Mbps limit
              return {
                type: 'RESOURCE',
                description: `Combined bandwidth requirements (${total} kbps) exceed available capacity for UE ${policy1.scope.ueId}`,
                severity: 'MEDIUM'
              };
            }
          }
        }

        // Cell capacity conflicts
        if (policy1.scope.cellId === policy2.scope.cellId) {
          if (policy1.statement?.qosParameters?.guaranteedBitrate && 
              policy2.statement?.qosParameters?.guaranteedBitrate) {
            const total = policy1.statement.qosParameters.guaranteedBitrate + 
                         policy2.statement.qosParameters.guaranteedBitrate;
            if (total > 500000) { // Assume 500Mbps cell capacity
              return {
                type: 'RESOURCE',
                description: `Combined guaranteed bandwidth (${total} kbps) exceeds cell capacity for Cell ${policy1.scope.cellId}`,
                severity: 'HIGH'
              };
            }
          }
        }
      }

      // Check for parameter conflicts
      if (policy1.statement && policy2.statement) {
        // Conflicting priority levels for overlapping scopes
        if (policy1.statement.priorityLevel && policy2.statement.priorityLevel) {
          if (Math.abs(policy1.statement.priorityLevel - policy2.statement.priorityLevel) > 10) {
            return {
              type: 'PARAMETER',
              description: `Large priority level difference (${policy1.statement.priorityLevel} vs ${policy2.statement.priorityLevel}) may cause service disruption`,
              severity: 'LOW'
            };
          }
        }
      }

      return null;
    } catch (err) {
      console.error('Error checking for conflicts:', err);
      return null;
    }
  };

  const resolveConflict = (conflictId, resolution) => {
    setConflicts(prev => prev.map(conflict => 
      conflict.conflict_id === conflictId 
        ? { ...conflict, resolution, status: 'RESOLVED' }
        : conflict
    ));
    setSelectedConflict(null);
  };

  const getConflictSeverityColor = (severity) => {
    switch (severity?.toLowerCase()) {
      case 'high':
        return '#dc3545';
      case 'medium':
        return '#ffc107';
      case 'low':
        return '#28a745';
      default:
        return '#6c757d';
    }
  };

  const getConflictTypeIcon = (type) => {
    switch (type?.toLowerCase()) {
      case 'resource':
        return '⚠️';
      case 'parameter':
        return '🔧';
      case 'priority':
        return '📊';
      case 'exclusive':
        return '🚫';
      default:
        return '❓';
    }
  };

  const generateResolutionSuggestions = (conflict) => {
    const suggestions = [];

    switch (conflict.conflict_type) {
      case 'RESOURCE':
        suggestions.push({
          strategy: 'priority',
          title: 'Priority-based Resolution',
          description: 'Keep the policy with higher priority, modify or remove the lower priority policy',
          action: 'Automatically resolve based on priority levels'
        });
        suggestions.push({
          strategy: 'modify',
          title: 'Modify Parameters',
          description: 'Adjust resource allocation parameters to eliminate the conflict',
          action: 'Reduce bandwidth requirements or change target resources'
        });
        suggestions.push({
          strategy: 'schedule',
          title: 'Time-based Scheduling',
          description: 'Apply policies at different times to avoid resource contention',
          action: 'Implement temporal separation of policy enforcement'
        });
        break;

      case 'PARAMETER':
        suggestions.push({
          strategy: 'merge',
          title: 'Parameter Merging',
          description: 'Combine compatible parameters from both policies',
          action: 'Create a unified policy with merged parameters'
        });
        suggestions.push({
          strategy: 'override',
          title: 'Override Conflicting Parameters',
          description: 'Use the most recent or highest priority parameter values',
          action: 'Apply parameter override rules'
        });
        break;

      default:
        suggestions.push({
          strategy: 'manual',
          title: 'Manual Resolution',
          description: 'Requires manual intervention to resolve the conflict',
          action: 'Review and manually resolve the conflict'
        });
    }

    return suggestions;
  };

  const unresolvedConflicts = conflicts.filter(c => c.status === 'UNRESOLVED');
  const resolvedConflicts = conflicts.filter(c => c.status === 'RESOLVED');

  if (loading) {
    return <LoadingDisplay message="Detecting policy conflicts..." />;
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={detectConflicts} />;
  }

  return (
    <div className="policy-conflict-resolver">
      <div className="resolver-header">
        <h3>Policy Conflict Resolution</h3>
        <div className="resolver-controls">
          <button onClick={detectConflicts} className="detect-btn">
            Re-scan for Conflicts
          </button>
        </div>
      </div>

      <div className="conflict-summary">
        <div className="summary-card">
          <div className="summary-stat">
            <span className="stat-value">{conflicts.length}</span>
            <span className="stat-label">Total Conflicts</span>
          </div>
        </div>
        <div className="summary-card">
          <div className="summary-stat">
            <span className="stat-value unresolved">{unresolvedConflicts.length}</span>
            <span className="stat-label">Unresolved</span>
          </div>
        </div>
        <div className="summary-card">
          <div className="summary-stat">
            <span className="stat-value resolved">{resolvedConflicts.length}</span>
            <span className="stat-label">Resolved</span>
          </div>
        </div>
      </div>

      {conflicts.length === 0 ? (
        <div className="no-conflicts">
          <div className="no-conflicts-icon">✅</div>
          <h4>No Policy Conflicts Detected</h4>
          <p>All policy instances are compatible with each other.</p>
        </div>
      ) : (
        <div className="conflicts-section">
          <div className="conflicts-list">
            <h4>Detected Conflicts</h4>
            {conflicts.map(conflict => (
              <div 
                key={conflict.conflict_id} 
                className={`conflict-item ${conflict.status.toLowerCase()}`}
                onClick={() => setSelectedConflict(conflict)}
              >
                <div className="conflict-header">
                  <div className="conflict-info">
                    <span className="conflict-icon">
                      {getConflictTypeIcon(conflict.conflict_type)}
                    </span>
                    <span className="conflict-type">{conflict.conflict_type}</span>
                    <span 
                      className="conflict-severity"
                      style={{ color: getConflictSeverityColor(conflict.severity) }}
                    >
                      {conflict.severity}
                    </span>
                  </div>
                  <div className="conflict-status">
                    <span className={`status-badge ${conflict.status.toLowerCase()}`}>
                      {conflict.status}
                    </span>
                  </div>
                </div>
                <div className="conflict-description">
                  {conflict.description}
                </div>
                <div className="conflict-policies">
                  <span className="policy-ref">{conflict.policy_instance_id}</span>
                  <span className="vs">vs</span>
                  <span className="policy-ref">{conflict.conflicting_policy_id}</span>
                </div>
              </div>
            ))}
          </div>

          {selectedConflict && (
            <div className="conflict-resolver-modal">
              <div className="resolver-modal-content">
                <div className="modal-header">
                  <h4>Resolve Conflict</h4>
                  <button 
                    onClick={() => setSelectedConflict(null)}
                    className="close-btn"
                  >
                    ×
                  </button>
                </div>

                <div className="conflict-details">
                  <div className="detail-item">
                    <span className="detail-label">Conflict Type:</span>
                    <span className="detail-value">{selectedConflict.conflict_type}</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Severity:</span>
                    <span 
                      className="detail-value"
                      style={{ color: getConflictSeverityColor(selectedConflict.severity) }}
                    >
                      {selectedConflict.severity}
                    </span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Description:</span>
                    <span className="detail-value">{selectedConflict.description}</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Detected:</span>
                    <span className="detail-value">
                      {new Date(selectedConflict.detected_at).toLocaleString()}
                    </span>
                  </div>
                </div>

                <div className="resolution-suggestions">
                  <h5>Resolution Suggestions</h5>
                  {generateResolutionSuggestions(selectedConflict).map((suggestion, index) => (
                    <div key={index} className="suggestion-item">
                      <div className="suggestion-header">
                        <h6>{suggestion.title}</h6>
                      </div>
                      <p className="suggestion-description">
                        {suggestion.description}
                      </p>
                      <button
                        onClick={() => resolveConflict(selectedConflict.conflict_id, suggestion.action)}
                        className="apply-resolution-btn"
                      >
                        Apply Resolution
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default PolicyConflictResolver;