/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import { ErrorDisplay } from './ErrorDisplay';
import './XAppDeploymentForm.css';

/**
 * xApp deployment form component with validation
 * Provides form interface for deploying new xApps with comprehensive validation
 */
const XAppDeploymentForm = ({ 
  onDeploy, 
  loading = false, 
  error = null, 
  onCancel = null 
}) => {
  const [formData, setFormData] = useState({
    name: '',
    version: '',
    image: '',
    namespace: 'ricxapp',
    instances: 1,
    resources: {
      cpu: '100m',
      memory: '128Mi',
      storage: '1Gi'
    },
    configuration: {},
    environment: {},
    ports: [],
    volumes: []
  });

  const [validationErrors, setValidationErrors] = useState({});
  const [configText, setConfigText] = useState('{}');
  const [envText, setEnvText] = useState('{}');
  const [portsText, setPortsText] = useState('[]');
  const [volumesText, setVolumesText] = useState('[]');

  // Reset form when component mounts or error changes
  useEffect(() => {
    if (!error) {
      setValidationErrors({});
    }
  }, [error]);

  // Validate form data
  const validateForm = () => {
    const errors = {};

    // Name validation
    if (!formData.name.trim()) {
      errors.name = 'xApp name is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.name)) {
      errors.name = 'xApp name must contain only lowercase letters, numbers, and hyphens';
    } else if (formData.name.length > 63) {
      errors.name = 'xApp name must be 63 characters or less';
    }

    // Version validation
    if (!formData.version.trim()) {
      errors.version = 'Version is required';
    } else if (!/^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9-]+)?$/.test(formData.version)) {
      errors.version = 'Version must follow semantic versioning (e.g., 1.0.0)';
    }

    // Image validation
    if (!formData.image.trim()) {
      errors.image = 'Container image is required';
    } else if (!/^[a-z0-9.-]+\/[a-z0-9.-]+:[a-z0-9.-]+$/i.test(formData.image)) {
      errors.image = 'Image must be in format registry/repository:tag';
    }

    // Namespace validation
    if (!formData.namespace.trim()) {
      errors.namespace = 'Namespace is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.namespace)) {
      errors.namespace = 'Namespace must contain only lowercase letters, numbers, and hyphens';
    }

    // Instances validation
    if (formData.instances < 1 || formData.instances > 10) {
      errors.instances = 'Instances must be between 1 and 10';
    }

    // Resource validation
    if (!formData.resources.cpu.trim()) {
      errors.cpu = 'CPU resource is required';
    } else if (!/^[0-9]+m?$/.test(formData.resources.cpu)) {
      errors.cpu = 'CPU must be in format like 100m or 1';
    }

    if (!formData.resources.memory.trim()) {
      errors.memory = 'Memory resource is required';
    } else if (!/^[0-9]+[KMGT]?i?$/.test(formData.resources.memory)) {
      errors.memory = 'Memory must be in format like 128Mi or 1Gi';
    }

    if (!formData.resources.storage.trim()) {
      errors.storage = 'Storage resource is required';
    } else if (!/^[0-9]+[KMGT]?i?$/.test(formData.resources.storage)) {
      errors.storage = 'Storage must be in format like 1Gi or 10Gi';
    }

    // JSON validation for configuration
    try {
      JSON.parse(configText);
    } catch (e) {
      errors.configuration = 'Configuration must be valid JSON';
    }

    // JSON validation for environment
    try {
      JSON.parse(envText);
    } catch (e) {
      errors.environment = 'Environment variables must be valid JSON';
    }

    // JSON validation for ports
    try {
      const ports = JSON.parse(portsText);
      if (!Array.isArray(ports)) {
        errors.ports = 'Ports must be a JSON array';
      } else {
        ports.forEach((port, index) => {
          if (typeof port !== 'object' || !port.name || !port.port) {
            errors.ports = `Port ${index + 1} must have name and port fields`;
          }
        });
      }
    } catch (e) {
      errors.ports = 'Ports must be valid JSON array';
    }

    // JSON validation for volumes
    try {
      const volumes = JSON.parse(volumesText);
      if (!Array.isArray(volumes)) {
        errors.volumes = 'Volumes must be a JSON array';
      }
    } catch (e) {
      errors.volumes = 'Volumes must be valid JSON array';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    try {
      // Parse JSON fields
      const deploymentData = {
        ...formData,
        configuration: JSON.parse(configText),
        environment: JSON.parse(envText),
        ports: JSON.parse(portsText),
        volumes: JSON.parse(volumesText)
      };

      await onDeploy(deploymentData);
    } catch (err) {
      console.error('Form submission error:', err);
    }
  };

  // Handle input changes
  const handleInputChange = (field, value) => {
    if (field.includes('.')) {
      const [parent, child] = field.split('.');
      setFormData(prev => ({
        ...prev,
        [parent]: {
          ...prev[parent],
          [child]: value
        }
      }));
    } else {
      setFormData(prev => ({
        ...prev,
        [field]: value
      }));
    }

    // Clear validation error for this field
    if (validationErrors[field]) {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  return (
    <div className="xapp-deployment-form">
      <div className="form-header">
        <h3>Deploy New xApp</h3>
        <p>Fill in the details below to deploy a new xApp to the O-RAN SC platform</p>
      </div>

      {error && (
        <div className="form-error">
          <ErrorDisplay error={error} />
        </div>
      )}

      <form onSubmit={handleSubmit} className="deployment-form">
        {/* Basic Information */}
        <div className="form-section">
          <h4>Basic Information</h4>
          
          <div className="form-group">
            <label htmlFor="name">xApp Name *</label>
            <input
              type="text"
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="my-xapp"
              className={validationErrors.name ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.name && (
              <span className="error-message">{validationErrors.name}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="version">Version *</label>
            <input
              type="text"
              id="version"
              value={formData.version}
              onChange={(e) => handleInputChange('version', e.target.value)}
              placeholder="1.0.0"
              className={validationErrors.version ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.version && (
              <span className="error-message">{validationErrors.version}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="image">Container Image *</label>
            <input
              type="text"
              id="image"
              value={formData.image}
              onChange={(e) => handleInputChange('image', e.target.value)}
              placeholder="registry.example.com/my-xapp:1.0.0"
              className={validationErrors.image ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.image && (
              <span className="error-message">{validationErrors.image}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="namespace">Namespace</label>
            <input
              type="text"
              id="namespace"
              value={formData.namespace}
              onChange={(e) => handleInputChange('namespace', e.target.value)}
              placeholder="ricxapp"
              className={validationErrors.namespace ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.namespace && (
              <span className="error-message">{validationErrors.namespace}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="instances">Instances</label>
            <input
              type="number"
              id="instances"
              min="1"
              max="10"
              value={formData.instances}
              onChange={(e) => handleInputChange('instances', parseInt(e.target.value))}
              className={validationErrors.instances ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.instances && (
              <span className="error-message">{validationErrors.instances}</span>
            )}
          </div>
        </div>

        {/* Resource Requirements */}
        <div className="form-section">
          <h4>Resource Requirements</h4>
          
          <div className="resource-grid">
            <div className="form-group">
              <label htmlFor="cpu">CPU *</label>
              <input
                type="text"
                id="cpu"
                value={formData.resources.cpu}
                onChange={(e) => handleInputChange('resources.cpu', e.target.value)}
                placeholder="100m"
                className={validationErrors.cpu ? 'error' : ''}
                disabled={loading}
              />
              {validationErrors.cpu && (
                <span className="error-message">{validationErrors.cpu}</span>
              )}
            </div>

            <div className="form-group">
              <label htmlFor="memory">Memory *</label>
              <input
                type="text"
                id="memory"
                value={formData.resources.memory}
                onChange={(e) => handleInputChange('resources.memory', e.target.value)}
                placeholder="128Mi"
                className={validationErrors.memory ? 'error' : ''}
                disabled={loading}
              />
              {validationErrors.memory && (
                <span className="error-message">{validationErrors.memory}</span>
              )}
            </div>

            <div className="form-group">
              <label htmlFor="storage">Storage *</label>
              <input
                type="text"
                id="storage"
                value={formData.resources.storage}
                onChange={(e) => handleInputChange('resources.storage', e.target.value)}
                placeholder="1Gi"
                className={validationErrors.storage ? 'error' : ''}
                disabled={loading}
              />
              {validationErrors.storage && (
                <span className="error-message">{validationErrors.storage}</span>
              )}
            </div>
          </div>
        </div>

        {/* Configuration */}
        <div className="form-section">
          <h4>Configuration</h4>
          
          <div className="form-group">
            <label htmlFor="configuration">xApp Configuration (JSON)</label>
            <textarea
              id="configuration"
              value={configText}
              onChange={(e) => setConfigText(e.target.value)}
              placeholder='{"key": "value"}'
              rows="4"
              className={validationErrors.configuration ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.configuration && (
              <span className="error-message">{validationErrors.configuration}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="environment">Environment Variables (JSON)</label>
            <textarea
              id="environment"
              value={envText}
              onChange={(e) => setEnvText(e.target.value)}
              placeholder='{"ENV_VAR": "value"}'
              rows="3"
              className={validationErrors.environment ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.environment && (
              <span className="error-message">{validationErrors.environment}</span>
            )}
          </div>
        </div>

        {/* Advanced Configuration */}
        <div className="form-section">
          <h4>Advanced Configuration</h4>
          
          <div className="form-group">
            <label htmlFor="ports">Ports (JSON Array)</label>
            <textarea
              id="ports"
              value={portsText}
              onChange={(e) => setPortsText(e.target.value)}
              placeholder='[{"name": "http", "port": 8080, "protocol": "TCP"}]'
              rows="3"
              className={validationErrors.ports ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.ports && (
              <span className="error-message">{validationErrors.ports}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="volumes">Volumes (JSON Array)</label>
            <textarea
              id="volumes"
              value={volumesText}
              onChange={(e) => setVolumesText(e.target.value)}
              placeholder='[{"name": "data", "mountPath": "/data", "size": "1Gi"}]'
              rows="3"
              className={validationErrors.volumes ? 'error' : ''}
              disabled={loading}
            />
            {validationErrors.volumes && (
              <span className="error-message">{validationErrors.volumes}</span>
            )}
          </div>
        </div>

        {/* Form Actions */}
        <div className="form-actions">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={onCancel}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={loading}
          >
            {loading ? 'Deploying...' : 'Deploy xApp'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default XAppDeploymentForm;