/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState } from 'react';
import './PolicyInstanceForm.css';
import { ErrorDisplay } from './ErrorDisplay';
import dashboardAPI from '../services/api';

const PolicyInstanceForm = ({ policyType, onSubmit, onCancel, loading, error }) => {
  const [formData, setFormData] = useState({
    policy_instance_id: '',
    policy: ''
  });
  const [policyError, setPolicyError] = useState(null);
  const [validationResult, setValidationResult] = useState(null);
  const [validating, setValidating] = useState(false);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handlePolicyChange = (e) => {
    const value = e.target.value;
    setFormData(prev => ({
      ...prev,
      policy: value
    }));

    // Clear previous validation
    setValidationResult(null);
    
    // Validate JSON format
    if (value.trim()) {
      try {
        JSON.parse(value);
        setPolicyError(null);
      } catch (err) {
        setPolicyError('Invalid JSON format');
      }
    } else {
      setPolicyError(null);
    }
  };

  const validatePolicy = async () => {
    if (!formData.policy.trim()) {
      setPolicyError('Policy is required');
      return;
    }

    try {
      const parsedPolicy = JSON.parse(formData.policy);
      setValidating(true);
      setPolicyError(null);

      const result = await dashboardAPI.validatePolicy(policyType.policy_type_id, {
        policy: parsedPolicy
      });

      setValidationResult(result);
      
      if (!result.is_valid) {
        setPolicyError('Policy validation failed');
      }
    } catch (err) {
      console.error('Policy validation error:', err);
      setPolicyError('Failed to validate policy');
      setValidationResult(null);
    } finally {
      setValidating(false);
    }
  };

  const generateSamplePolicy = () => {
    try {
      const schema = typeof policyType.schema === 'string' 
        ? JSON.parse(policyType.schema) 
        : policyType.schema;

      const samplePolicy = generateFromSchema(schema);
      
      setFormData(prev => ({
        ...prev,
        policy: JSON.stringify(samplePolicy, null, 2)
      }));
      setPolicyError(null);
      setValidationResult(null);
    } catch (err) {
      console.error('Failed to generate sample policy:', err);
      setPolicyError('Failed to generate sample policy from schema');
    }
  };

  const generateFromSchema = (schema) => {
    if (!schema || typeof schema !== 'object') {
      return {};
    }

    const result = {};

    if (schema.properties) {
      Object.entries(schema.properties).forEach(([key, propSchema]) => {
        if (schema.required && schema.required.includes(key)) {
          result[key] = generateValueFromProperty(propSchema);
        }
      });
    }

    return result;
  };

  const generateValueFromProperty = (propSchema) => {
    if (!propSchema || typeof propSchema !== 'object') {
      return null;
    }

    switch (propSchema.type) {
      case 'string':
        return propSchema.default || 'sample-value';
      case 'number':
      case 'integer':
        return propSchema.default || propSchema.minimum || 1;
      case 'boolean':
        return propSchema.default !== undefined ? propSchema.default : true;
      case 'array':
        return [];
      case 'object':
        return generateFromSchema(propSchema);
      default:
        return null;
    }
  };

  const formatPolicy = () => {
    if (formData.policy.trim()) {
      try {
        const parsed = JSON.parse(formData.policy);
        setFormData(prev => ({
          ...prev,
          policy: JSON.stringify(parsed, null, 2)
        }));
        setPolicyError(null);
      } catch (err) {
        setPolicyError('Cannot format invalid JSON');
      }
    }
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (!formData.policy_instance_id.trim()) {
      setPolicyError('Policy Instance ID is required');
      return;
    }

    if (!formData.policy.trim()) {
      setPolicyError('Policy is required');
      return;
    }

    try {
      const parsedPolicy = JSON.parse(formData.policy);
      onSubmit({
        ...formData,
        policy: parsedPolicy
      });
    } catch (err) {
      setPolicyError('Invalid JSON policy format');
    }
  };

  return (
    <div className="policy-instance-form-overlay">
      <div className="policy-instance-form-modal">
        <div className="form-header">
          <div className="header-info">
            <h3>Create Policy Instance</h3>
            <p className="policy-type-name">
              Type: {policyType.name || policyType.policy_type_id}
            </p>
          </div>
          <button onClick={onCancel} className="close-btn">×</button>
        </div>

        {error && <ErrorDisplay error={error} />}

        <form onSubmit={handleSubmit} className="policy-instance-form">
          <div className="form-group">
            <label htmlFor="policy_instance_id">Policy Instance ID *</label>
            <input
              type="text"
              id="policy_instance_id"
              name="policy_instance_id"
              value={formData.policy_instance_id}
              onChange={handleInputChange}
              placeholder="e.g., qos-policy-instance-1"
              required
              disabled={loading}
            />
            <small>Unique identifier for this policy instance</small>
          </div>

          <div className="form-group">
            <div className="policy-header">
              <label htmlFor="policy">Policy Configuration (JSON) *</label>
              <div className="policy-actions">
                <button
                  type="button"
                  onClick={generateSamplePolicy}
                  className="sample-btn"
                  disabled={loading}
                >
                  Generate Sample
                </button>
                <button
                  type="button"
                  onClick={formatPolicy}
                  className="format-btn"
                  disabled={loading}
                >
                  Format JSON
                </button>
                <button
                  type="button"
                  onClick={validatePolicy}
                  className="validate-btn"
                  disabled={loading || validating || !formData.policy.trim()}
                >
                  {validating ? 'Validating...' : 'Validate'}
                </button>
              </div>
            </div>
            <textarea
              id="policy"
              name="policy"
              value={formData.policy}
              onChange={handlePolicyChange}
              placeholder="Enter policy configuration as JSON..."
              rows="15"
              className={`policy-textarea ${policyError ? 'error' : ''} ${validationResult?.is_valid ? 'valid' : ''}`}
              required
              disabled={loading}
            />
            
            {policyError && (
              <div className="error-message">{policyError}</div>
            )}
            
            {validationResult && (
              <div className={`validation-result ${validationResult.is_valid ? 'valid' : 'invalid'}`}>
                {validationResult.is_valid ? (
                  <div className="validation-success">
                    ✅ Policy is valid
                  </div>
                ) : (
                  <div className="validation-errors">
                    <div className="validation-header">❌ Policy validation failed:</div>
                    <ul>
                      {validationResult.errors?.map((error, index) => (
                        <li key={index}>
                          <strong>{error.field}:</strong> {error.message}
                          {error.value && <span className="error-value"> (value: {error.value})</span>}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
            
            <small>JSON configuration that conforms to the policy type schema</small>
          </div>

          <div className="form-actions">
            <button
              type="button"
              onClick={onCancel}
              className="cancel-btn"
              disabled={loading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="submit-btn"
              disabled={
                loading || 
                validating ||
                !!policyError || 
                !formData.policy_instance_id.trim() || 
                !formData.policy.trim() ||
                (validationResult && !validationResult.is_valid)
              }
            >
              {loading ? 'Creating...' : 'Create Policy Instance'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default PolicyInstanceForm;