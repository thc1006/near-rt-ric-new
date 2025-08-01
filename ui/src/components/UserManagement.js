/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './UserManagement.css';
import dashboardAPI from '../services/api';

const UserManagement = () => {
  const [policies, setPolicies] = useState([]);
  const [certificates, setCertificates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedPolicy, setSelectedPolicy] = useState(null);
  const [showPolicyForm, setShowPolicyForm] = useState(false);
  const [showCertForm, setShowCertForm] = useState(false);
  const [activeTab, setActiveTab] = useState('policies');

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [policiesData, certsData] = await Promise.all([
        dashboardAPI.getAccessControlPolicies(),
        dashboardAPI.getCertificates()
      ]);
      
      setPolicies(policiesData.policies || []);
      setCertificates(certsData.certificates || []);
      setError(null);
    } catch (err) {
      console.error('Failed to load user management data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePolicy = async (policyData) => {
    try {
      await dashboardAPI.createAccessControlPolicy(policyData);
      setShowPolicyForm(false);
      loadData();
    } catch (err) {
      console.error('Failed to create policy:', err);
      setError(err.message);
    }
  };

  const handleUpdatePolicy = async (policyId, policyData) => {
    try {
      await dashboardAPI.updateAccessControlPolicy(policyId, policyData);
      setSelectedPolicy(null);
      loadData();
    } catch (err) {
      console.error('Failed to update policy:', err);
      setError(err.message);
    }
  };

  const handleDeletePolicy = async (policyId) => {
    if (window.confirm('Are you sure you want to delete this policy?')) {
      try {
        await dashboardAPI.deleteAccessControlPolicy(policyId);
        loadData();
      } catch (err) {
        console.error('Failed to delete policy:', err);
        setError(err.message);
      }
    }
  };

  const handleCreateCertificate = async (certData) => {
    try {
      await dashboardAPI.createCertificate(certData);
      setShowCertForm(false);
      loadData();
    } catch (err) {
      console.error('Failed to create certificate:', err);
      setError(err.message);
    }
  };

  const policyStats = {
    total: policies.length,
    active: policies.filter(p => p.status === 'ACTIVE').length,
    inactive: policies.filter(p => p.status === 'INACTIVE').length,
    rbac: policies.filter(p => p.policy_type === 'RBAC').length,
    abac: policies.filter(p => p.policy_type === 'ABAC').length
  };

  const certStats = {
    total: certificates.length,
    active: certificates.filter(c => c.status === 'ACTIVE').length,
    expired: certificates.filter(c => c.status === 'EXPIRED').length,
    expiringSoon: certificates.filter(c => {
      const expiryDate = new Date(c.not_after);
      const thirtyDaysFromNow = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000);
      return expiryDate <= thirtyDaysFromNow && c.status === 'ACTIVE';
    }).length
  };

  if (loading) {
    return (
      <div className="user-management">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading user management data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="user-management">
      <div className="user-header">
        <h3>User Management & RBAC</h3>
        <div className="user-actions">
          {activeTab === 'policies' && (
            <button 
              className="btn btn-primary"
              onClick={() => setShowPolicyForm(true)}
            >
              Create Policy
            </button>
          )}
          {activeTab === 'certificates' && (
            <button 
              className="btn btn-primary"
              onClick={() => setShowCertForm(true)}
            >
              Create Certificate
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="error-message">
          <span className="error-icon">⚠</span>
          {error}
        </div>
      )}

      <div className="user-tabs">
        <button
          className={`tab-button ${activeTab === 'policies' ? 'active' : ''}`}
          onClick={() => setActiveTab('policies')}
        >
          Access Control Policies
        </button>
        <button
          className={`tab-button ${activeTab === 'certificates' ? 'active' : ''}`}
          onClick={() => setActiveTab('certificates')}
        >
          Certificates
        </button>
      </div>

      {activeTab === 'policies' && (
        <div className="policies-section">
          <div className="policy-stats">
            <div className="stat-card">
              <span className="stat-value">{policyStats.total}</span>
              <span className="stat-label">Total Policies</span>
            </div>
            <div className="stat-card active">
              <span className="stat-value">{policyStats.active}</span>
              <span className="stat-label">Active</span>
            </div>
            <div className="stat-card inactive">
              <span className="stat-value">{policyStats.inactive}</span>
              <span className="stat-label">Inactive</span>
            </div>
            <div className="stat-card rbac">
              <span className="stat-value">{policyStats.rbac}</span>
              <span className="stat-label">RBAC</span>
            </div>
            <div className="stat-card abac">
              <span className="stat-value">{policyStats.abac}</span>
              <span className="stat-label">ABAC</span>
            </div>
          </div>

          <div className="policy-list">
            {policies.map(policy => (
              <div key={policy.id} className="policy-item">
                <div className="policy-info">
                  <div className="policy-header">
                    <h5>{policy.name}</h5>
                    <div className="policy-badges">
                      <span className={`status ${policy.status.toLowerCase()}`}>
                        {policy.status}
                      </span>
                      <span className={`policy-type ${policy.policy_type.toLowerCase()}`}>
                        {policy.policy_type}
                      </span>
                    </div>
                  </div>
                  <p>{policy.description}</p>
                  <div className="policy-meta">
                    <span className="rule-count">
                      {policy.rules.length} rules
                    </span>
                    <span className="date">
                      Updated: {new Date(policy.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                  <div className="policy-rules-preview">
                    {policy.rules.slice(0, 3).map((rule, index) => (
                      <div key={index} className="rule-preview">
                        <span className="rule-subject">{rule.subject.type}:{rule.subject.identifier}</span>
                        <span className="rule-action">{rule.action}</span>
                        <span className="rule-resource">{rule.resource}</span>
                        <span className={`rule-effect ${rule.effect.toLowerCase()}`}>
                          {rule.effect}
                        </span>
                      </div>
                    ))}
                    {policy.rules.length > 3 && (
                      <div className="rule-preview more">
                        +{policy.rules.length - 3} more rules
                      </div>
                    )}
                  </div>
                </div>
                <div className="policy-actions">
                  <button
                    className="btn btn-sm btn-outline"
                    onClick={() => setSelectedPolicy(policy)}
                  >
                    Edit
                  </button>
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleDeletePolicy(policy.id)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
            
            {policies.length === 0 && (
              <div className="no-policies">
                <p>No access control policies found. Create your first policy to manage user permissions.</p>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'certificates' && (
        <div className="certificates-section">
          <div className="cert-stats">
            <div className="stat-card">
              <span className="stat-value">{certStats.total}</span>
              <span className="stat-label">Total Certificates</span>
            </div>
            <div className="stat-card active">
              <span className="stat-value">{certStats.active}</span>
              <span className="stat-label">Active</span>
            </div>
            <div className="stat-card expired">
              <span className="stat-value">{certStats.expired}</span>
              <span className="stat-label">Expired</span>
            </div>
            <div className="stat-card expiring">
              <span className="stat-value">{certStats.expiringSoon}</span>
              <span className="stat-label">Expiring Soon</span>
            </div>
          </div>

          <div className="cert-list">
            {certificates.map(cert => (
              <div key={cert.id} className="cert-item">
                <div className="cert-info">
                  <div className="cert-header">
                    <h5>{cert.name}</h5>
                    <span className={`cert-status ${cert.status.toLowerCase()}`}>
                      {cert.status}
                    </span>
                  </div>
                  <div className="cert-details">
                    <div className="cert-detail">
                      <label>Type:</label>
                      <span>{cert.type}</span>
                    </div>
                    <div className="cert-detail">
                      <label>Subject:</label>
                      <span className="cert-subject">{cert.subject}</span>
                    </div>
                    <div className="cert-detail">
                      <label>Issuer:</label>
                      <span className="cert-issuer">{cert.issuer}</span>
                    </div>
                    <div className="cert-detail">
                      <label>Valid From:</label>
                      <span>{new Date(cert.not_before).toLocaleDateString()}</span>
                    </div>
                    <div className="cert-detail">
                      <label>Valid Until:</label>
                      <span className={new Date(cert.not_after) <= new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) ? 'expiring' : ''}>
                        {new Date(cert.not_after).toLocaleDateString()}
                      </span>
                    </div>
                    <div className="cert-detail">
                      <label>Fingerprint:</label>
                      <span className="cert-fingerprint">{cert.fingerprint}</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
            
            {certificates.length === 0 && (
              <div className="no-certificates">
                <p>No certificates found. Create certificates to enable secure communications.</p>
              </div>
            )}
          </div>
        </div>
      )}

      {showPolicyForm && (
        <PolicyForm
          onSubmit={handleCreatePolicy}
          onCancel={() => setShowPolicyForm(false)}
        />
      )}

      {showCertForm && (
        <CertificateForm
          onSubmit={handleCreateCertificate}
          onCancel={() => setShowCertForm(false)}
        />
      )}

      {selectedPolicy && (
        <PolicyEditor
          policy={selectedPolicy}
          onSave={handleUpdatePolicy}
          onCancel={() => setSelectedPolicy(null)}
        />
      )}
    </div>
  );
};

const PolicyForm = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    policy_type: 'RBAC',
    rules: [
      {
        subject: { type: 'USER', identifier: '', attributes: [] },
        action: 'READ',
        resource: '*',
        effect: 'ALLOW',
        conditions: {}
      }
    ]
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const addRule = () => {
    setFormData({
      ...formData,
      rules: [
        ...formData.rules,
        {
          subject: { type: 'USER', identifier: '', attributes: [] },
          action: 'READ',
          resource: '*',
          effect: 'ALLOW',
          conditions: {}
        }
      ]
    });
  };

  const updateRule = (index, field, value) => {
    const newRules = [...formData.rules];
    if (field.startsWith('subject.')) {
      const subjectField = field.split('.')[1];
      newRules[index].subject[subjectField] = value;
    } else {
      newRules[index][field] = value;
    }
    setFormData({ ...formData, rules: newRules });
  };

  const removeRule = (index) => {
    setFormData({
      ...formData,
      rules: formData.rules.filter((_, i) => i !== index)
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content large">
        <h4>Create Access Control Policy</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Policy Name</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows="3"
            />
          </div>
          <div className="form-group">
            <label>Policy Type</label>
            <select
              value={formData.policy_type}
              onChange={(e) => setFormData({ ...formData, policy_type: e.target.value })}
              required
            >
              <option value="RBAC">Role-Based Access Control (RBAC)</option>
              <option value="ABAC">Attribute-Based Access Control (ABAC)</option>
            </select>
          </div>
          
          <div className="rules-section">
            <div className="rules-header">
              <h5>Access Control Rules</h5>
              <button type="button" onClick={addRule} className="btn btn-sm btn-outline">
                Add Rule
              </button>
            </div>
            
            {formData.rules.map((rule, index) => (
              <div key={index} className="rule-form">
                <div className="rule-header">
                  <span>Rule {index + 1}</span>
                  {formData.rules.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeRule(index)}
                      className="btn btn-sm btn-danger"
                    >
                      Remove
                    </button>
                  )}
                </div>
                
                <div className="rule-fields">
                  <div className="form-group">
                    <label>Subject Type</label>
                    <select
                      value={rule.subject.type}
                      onChange={(e) => updateRule(index, 'subject.type', e.target.value)}
                      required
                    >
                      <option value="USER">User</option>
                      <option value="ROLE">Role</option>
                      <option value="GROUP">Group</option>
                      <option value="SERVICE">Service</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Subject Identifier</label>
                    <input
                      type="text"
                      value={rule.subject.identifier}
                      onChange={(e) => updateRule(index, 'subject.identifier', e.target.value)}
                      placeholder="e.g., admin, user123, service-account"
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label>Action</label>
                    <select
                      value={rule.action}
                      onChange={(e) => updateRule(index, 'action', e.target.value)}
                      required
                    >
                      <option value="READ">Read</option>
                      <option value="write">Write</option>
                      <option value="delete">Delete</option>
                      <option value="execute">Execute</option>
                      <option value="*">All Actions</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Resource</label>
                    <input
                      type="text"
                      value={rule.resource}
                      onChange={(e) => updateRule(index, 'resource', e.target.value)}
                      placeholder="e.g., /api/v1/*, configurations, alarms"
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label>Effect</label>
                    <select
                      value={rule.effect}
                      onChange={(e) => updateRule(index, 'effect', e.target.value)}
                      required
                    >
                      <option value="ALLOW">Allow</option>
                      <option value="DENY">Deny</option>
                    </select>
                  </div>
                </div>
              </div>
            ))}
          </div>
          
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create Policy
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const CertificateForm = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    name: '',
    type: 'SERVER',
    subject: '',
    key_size: 2048,
    validity_days: 365
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Create Certificate</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Certificate Name</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Certificate Type</label>
            <select
              value={formData.type}
              onChange={(e) => setFormData({ ...formData, type: e.target.value })}
              required
            >
              <option value="SERVER">Server Certificate</option>
              <option value="CLIENT">Client Certificate</option>
              <option value="CA">Certificate Authority</option>
            </select>
          </div>
          <div className="form-group">
            <label>Subject (Distinguished Name)</label>
            <input
              type="text"
              value={formData.subject}
              onChange={(e) => setFormData({ ...formData, subject: e.target.value })}
              placeholder="CN=example.com,O=Organization,C=US"
              required
            />
          </div>
          <div className="form-group">
            <label>Key Size</label>
            <select
              value={formData.key_size}
              onChange={(e) => setFormData({ ...formData, key_size: parseInt(e.target.value) })}
            >
              <option value={2048}>2048 bits</option>
              <option value={3072}>3072 bits</option>
              <option value={4096}>4096 bits</option>
            </select>
          </div>
          <div className="form-group">
            <label>Validity Period (days)</label>
            <input
              type="number"
              value={formData.validity_days}
              onChange={(e) => setFormData({ ...formData, validity_days: parseInt(e.target.value) })}
              min="1"
              max="3650"
              required
            />
          </div>
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create Certificate
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const PolicyEditor = ({ policy, onSave, onCancel }) => {
  const [formData, setFormData] = useState({
    name: policy.name,
    description: policy.description,
    policy_type: policy.policy_type,
    rules: [...policy.rules]
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSave(policy.id, formData);
  };

  const addRule = () => {
    setFormData({
      ...formData,
      rules: [
        ...formData.rules,
        {
          subject: { type: 'USER', identifier: '', attributes: [] },
          action: 'READ',
          resource: '*',
          effect: 'ALLOW',
          conditions: {}
        }
      ]
    });
  };

  const updateRule = (index, field, value) => {
    const newRules = [...formData.rules];
    if (field.startsWith('subject.')) {
      const subjectField = field.split('.')[1];
      newRules[index].subject[subjectField] = value;
    } else {
      newRules[index][field] = value;
    }
    setFormData({ ...formData, rules: newRules });
  };

  const removeRule = (index) => {
    setFormData({
      ...formData,
      rules: formData.rules.filter((_, i) => i !== index)
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content large">
        <h4>Edit Policy: {policy.name}</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Policy Name</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows="3"
            />
          </div>
          
          <div className="rules-section">
            <div className="rules-header">
              <h5>Access Control Rules</h5>
              <button type="button" onClick={addRule} className="btn btn-sm btn-outline">
                Add Rule
              </button>
            </div>
            
            {formData.rules.map((rule, index) => (
              <div key={index} className="rule-form">
                <div className="rule-header">
                  <span>Rule {index + 1}</span>
                  {formData.rules.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeRule(index)}
                      className="btn btn-sm btn-danger"
                    >
                      Remove
                    </button>
                  )}
                </div>
                
                <div className="rule-fields">
                  <div className="form-group">
                    <label>Subject Type</label>
                    <select
                      value={rule.subject.type}
                      onChange={(e) => updateRule(index, 'subject.type', e.target.value)}
                      required
                    >
                      <option value="USER">User</option>
                      <option value="ROLE">Role</option>
                      <option value="GROUP">Group</option>
                      <option value="SERVICE">Service</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Subject Identifier</label>
                    <input
                      type="text"
                      value={rule.subject.identifier}
                      onChange={(e) => updateRule(index, 'subject.identifier', e.target.value)}
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label>Action</label>
                    <select
                      value={rule.action}
                      onChange={(e) => updateRule(index, 'action', e.target.value)}
                      required
                    >
                      <option value="read">Read</option>
                      <option value="write">Write</option>
                      <option value="delete">Delete</option>
                      <option value="execute">Execute</option>
                      <option value="*">All Actions</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Resource</label>
                    <input
                      type="text"
                      value={rule.resource}
                      onChange={(e) => updateRule(index, 'resource', e.target.value)}
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label>Effect</label>
                    <select
                      value={rule.effect}
                      onChange={(e) => updateRule(index, 'effect', e.target.value)}
                      required
                    >
                      <option value="ALLOW">Allow</option>
                      <option value="DENY">Deny</option>
                    </select>
                  </div>
                </div>
              </div>
            ))}
          </div>
          
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Save Changes
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default UserManagement;