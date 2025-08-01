/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './ConfigurationManager.css';
import dashboardAPI from '../services/api';

const ConfigurationManager = () => {
  const [configurations, setConfigurations] = useState([]);
  const [managedObjects, setManagedObjects] = useState([]);
  const [backups, setBackups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [showConfigForm, setShowConfigForm] = useState(false);
  const [showBackupForm, setShowBackupForm] = useState(false);
  const [filter, setFilter] = useState({ status: '' });
  const [validationResult, setValidationResult] = useState(null);

  useEffect(() => {
    loadData();
  }, [filter]);

  const loadData = async () => {
    try {
      setLoading(true);
      const [configsData, objectsData, backupsData] = await Promise.all([
        dashboardAPI.getConfigurations(filter),
        dashboardAPI.getManagedObjects(),
        dashboardAPI.getBackups()
      ]);
      
      setConfigurations(configsData.configurations || []);
      setManagedObjects(objectsData.managed_objects || []);
      setBackups(backupsData.backups || []);
      setError(null);
    } catch (err) {
      console.error('Failed to load configuration data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateConfiguration = async (configData) => {
    try {
      await dashboardAPI.createConfiguration(configData.id, {
        name: configData.name,
        description: configData.description,
        config: configData.config
      });
      
      setShowConfigForm(false);
      loadData();
    } catch (err) {
      console.error('Failed to create configuration:', err);
      setError(err.message);
    }
  };

  const handleUpdateConfiguration = async (configId, configData) => {
    try {
      await dashboardAPI.updateConfiguration(configId, configData);
      setSelectedConfig(null);
      loadData();
    } catch (err) {
      console.error('Failed to update configuration:', err);
      setError(err.message);
    }
  };

  const handleValidateConfiguration = async (configData) => {
    try {
      const result = await dashboardAPI.validateConfiguration(configData);
      setValidationResult(result);
    } catch (err) {
      console.error('Failed to validate configuration:', err);
      setError(err.message);
    }
  };

  const handleCreateBackup = async (backupData) => {
    try {
      await dashboardAPI.createBackup(backupData);
      setShowBackupForm(false);
      loadData();
    } catch (err) {
      console.error('Failed to create backup:', err);
      setError(err.message);
    }
  };

  const handleRestoreBackup = async (backupId) => {
    try {
      await dashboardAPI.restoreConfiguration({
        backup_id: backupId,
        restore_all: true
      });
      loadData();
    } catch (err) {
      console.error('Failed to restore backup:', err);
      setError(err.message);
    }
  };

  const handleDeleteBackup = async (backupId) => {
    if (window.confirm('Are you sure you want to delete this backup?')) {
      try {
        await dashboardAPI.deleteBackup(backupId);
        loadData();
      } catch (err) {
        console.error('Failed to delete backup:', err);
        setError(err.message);
      }
    }
  };

  if (loading) {
    return (
      <div className="configuration-manager">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading configurations...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="configuration-manager">
      <div className="config-header">
        <h3>Configuration Management</h3>
        <div className="config-actions">
          <button 
            className="btn btn-primary"
            onClick={() => setShowConfigForm(true)}
          >
            Create Configuration
          </button>
          <button 
            className="btn btn-secondary"
            onClick={() => setShowBackupForm(true)}
          >
            Create Backup
          </button>
        </div>
      </div>

      {error && (
        <div className="error-message">
          <span className="error-icon">⚠</span>
          {error}
        </div>
      )}

      <div className="config-filters">
        <select
          value={filter.status}
          onChange={(e) => setFilter({ ...filter, status: e.target.value })}
          className="filter-select"
        >
          <option value="">All Statuses</option>
          <option value="ACTIVE">Active</option>
          <option value="INACTIVE">Inactive</option>
          <option value="PENDING">Pending</option>
          <option value="ERROR">Error</option>
        </select>
      </div>

      <div className="config-content">
        <div className="config-section">
          <h4>Configurations ({configurations.length})</h4>
          <div className="config-list">
            {configurations.map(config => (
              <div key={config.id} className="config-item">
                <div className="config-info">
                  <h5>{config.name}</h5>
                  <p>{config.description}</p>
                  <div className="config-meta">
                    <span className={`status ${config.status.toLowerCase()}`}>
                      {config.status}
                    </span>
                    <span className="version">v{config.version}</span>
                    <span className="date">
                      Updated: {new Date(config.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
                <div className="config-actions">
                  <button
                    className="btn btn-sm btn-outline"
                    onClick={() => setSelectedConfig(config)}
                  >
                    Edit
                  </button>
                  <button
                    className="btn btn-sm btn-outline"
                    onClick={() => handleValidateConfiguration({ config: config.config })}
                  >
                    Validate
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="managed-objects-section">
          <h4>Managed Objects ({managedObjects.length})</h4>
          <div className="objects-grid">
            {managedObjects.map(obj => (
              <div key={obj.id} className="object-card">
                <div className="object-header">
                  <h6>{obj.name}</h6>
                  <span className={`object-type ${obj.type.toLowerCase()}`}>
                    {obj.type}
                  </span>
                </div>
                <p>{obj.description}</p>
                <div className="object-state">
                  <span className={`state ${obj.state.toLowerCase()}`}>
                    {obj.state}
                  </span>
                  <span className="date">
                    {new Date(obj.updated_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="backups-section">
          <h4>Configuration Backups ({backups.length})</h4>
          <div className="backup-list">
            {backups.map(backup => (
              <div key={backup.backup_id} className="backup-item">
                <div className="backup-info">
                  <h6>{backup.name}</h6>
                  <p>{backup.description}</p>
                  <div className="backup-meta">
                    <span className={`status ${backup.status.toLowerCase()}`}>
                      {backup.status}
                    </span>
                    <span className="size">{(backup.size / 1024).toFixed(1)} KB</span>
                    <span className="date">
                      {new Date(backup.created_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
                <div className="backup-actions">
                  <button
                    className="btn btn-sm btn-success"
                    onClick={() => handleRestoreBackup(backup.backup_id)}
                    disabled={backup.status !== 'COMPLETED'}
                  >
                    Restore
                  </button>
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleDeleteBackup(backup.backup_id)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {showConfigForm && (
        <ConfigurationForm
          onSubmit={handleCreateConfiguration}
          onCancel={() => setShowConfigForm(false)}
          onValidate={handleValidateConfiguration}
          validationResult={validationResult}
        />
      )}

      {showBackupForm && (
        <BackupForm
          onSubmit={handleCreateBackup}
          onCancel={() => setShowBackupForm(false)}
          managedObjects={managedObjects}
        />
      )}

      {selectedConfig && (
        <ConfigurationEditor
          config={selectedConfig}
          onSave={handleUpdateConfiguration}
          onCancel={() => setSelectedConfig(null)}
          onValidate={handleValidateConfiguration}
          validationResult={validationResult}
        />
      )}
    </div>
  );
};

const ConfigurationForm = ({ onSubmit, onCancel, onValidate, validationResult }) => {
  const [formData, setFormData] = useState({
    id: '',
    name: '',
    description: '',
    config: '{}'
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const handleValidate = () => {
    try {
      const configObj = JSON.parse(formData.config);
      onValidate({ config: configObj });
    } catch (err) {
      alert('Invalid JSON format');
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Create Configuration</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Configuration ID</label>
            <input
              type="text"
              value={formData.id}
              onChange={(e) => setFormData({ ...formData, id: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Name</label>
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
            <label>Configuration (JSON)</label>
            <textarea
              value={formData.config}
              onChange={(e) => setFormData({ ...formData, config: e.target.value })}
              rows="10"
              className="config-editor"
              required
            />
            <button type="button" onClick={handleValidate} className="btn btn-sm btn-outline">
              Validate JSON
            </button>
          </div>
          
          {validationResult && (
            <div className={`validation-result ${validationResult.is_valid ? 'valid' : 'invalid'}`}>
              {validationResult.is_valid ? (
                <span className="success">✓ Configuration is valid</span>
              ) : (
                <div>
                  <span className="error">✗ Configuration has errors:</span>
                  <ul>
                    {validationResult.errors.map((error, index) => (
                      <li key={index}>{error.field}: {error.message}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
          
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const BackupForm = ({ onSubmit, onCancel, managedObjects }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    include_all: true,
    object_types: []
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Create Backup</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Backup Name</label>
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
            <label>
              <input
                type="checkbox"
                checked={formData.include_all}
                onChange={(e) => setFormData({ ...formData, include_all: e.target.checked })}
              />
              Include all managed objects
            </label>
          </div>
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create Backup
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const ConfigurationEditor = ({ config, onSave, onCancel, onValidate, validationResult }) => {
  const [formData, setFormData] = useState({
    config: JSON.stringify(config.config, null, 2),
    description: config.description
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    try {
      const configObj = JSON.parse(formData.config);
      onSave(config.id, {
        config: configObj,
        description: formData.description
      });
    } catch (err) {
      alert('Invalid JSON format');
    }
  };

  const handleValidate = () => {
    try {
      const configObj = JSON.parse(formData.config);
      onValidate({ config: configObj });
    } catch (err) {
      alert('Invalid JSON format');
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content large">
        <h4>Edit Configuration: {config.name}</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows="3"
            />
          </div>
          <div className="form-group">
            <label>Configuration (JSON)</label>
            <textarea
              value={formData.config}
              onChange={(e) => setFormData({ ...formData, config: e.target.value })}
              rows="15"
              className="config-editor"
              required
            />
            <button type="button" onClick={handleValidate} className="btn btn-sm btn-outline">
              Validate JSON
            </button>
          </div>
          
          {validationResult && (
            <div className={`validation-result ${validationResult.is_valid ? 'valid' : 'invalid'}`}>
              {validationResult.is_valid ? (
                <span className="success">✓ Configuration is valid</span>
              ) : (
                <div>
                  <span className="error">✗ Configuration has errors:</span>
                  <ul>
                    {validationResult.errors.map((error, index) => (
                      <li key={index}>{error.field}: {error.message}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
          
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

export default ConfigurationManager;