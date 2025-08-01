/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './AlarmManager.css';
import dashboardAPI from '../services/api';

const AlarmManager = () => {
  const [alarms, setAlarms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState({ severity: '', search: '' });
  const [selectedAlarm, setSelectedAlarm] = useState(null);
  const [showAlarmForm, setShowAlarmForm] = useState(false);
  const [showCorrelationForm, setShowCorrelationForm] = useState(false);
  const [selectedAlarms, setSelectedAlarms] = useState([]);

  useEffect(() => {
    loadAlarms();
    const interval = setInterval(loadAlarms, 10000); // Refresh every 10 seconds
    return () => clearInterval(interval);
  }, [filter]);

  const loadAlarms = async () => {
    try {
      setLoading(true);
      const data = await dashboardAPI.getAlarms(filter);
      setAlarms(data.alarms || []);
      setError(null);
    } catch (err) {
      console.error('Failed to load alarms:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleAcknowledgeAlarm = async (alarmId) => {
    try {
      await dashboardAPI.acknowledgeAlarm(alarmId, {
        user: 'current_user', // In real app, get from auth context
        comment: 'Acknowledged via dashboard'
      });
      loadAlarms();
    } catch (err) {
      console.error('Failed to acknowledge alarm:', err);
      setError(err.message);
    }
  };

  const handleClearAlarm = async (alarmId) => {
    try {
      await dashboardAPI.clearAlarm(alarmId, {
        user: 'current_user',
        reason: 'Cleared via dashboard',
        clear_time: new Date().toISOString()
      });
      loadAlarms();
    } catch (err) {
      console.error('Failed to clear alarm:', err);
      setError(err.message);
    }
  };

  const handleGenerateAlarm = async (alarmData) => {
    try {
      await dashboardAPI.generateAlarm(alarmData);
      setShowAlarmForm(false);
      loadAlarms();
    } catch (err) {
      console.error('Failed to generate alarm:', err);
      setError(err.message);
    }
  };

  const handleCorrelateAlarms = async (correlationData) => {
    try {
      await dashboardAPI.correlateAlarms({
        alarm_ids: selectedAlarms,
        ...correlationData
      });
      setShowCorrelationForm(false);
      setSelectedAlarms([]);
      loadAlarms();
    } catch (err) {
      console.error('Failed to correlate alarms:', err);
      setError(err.message);
    }
  };

  const filteredAlarms = alarms.filter(alarm => {
    const matchesSearch = !filter.search || 
      alarm.specific_problem.toLowerCase().includes(filter.search.toLowerCase()) ||
      alarm.managed_object_id.toLowerCase().includes(filter.search.toLowerCase()) ||
      alarm.probable_cause.toLowerCase().includes(filter.search.toLowerCase());
    
    return matchesSearch;
  });

  const alarmStats = {
    total: filteredAlarms.length,
    critical: filteredAlarms.filter(a => a.severity === 'CRITICAL').length,
    major: filteredAlarms.filter(a => a.severity === 'MAJOR').length,
    minor: filteredAlarms.filter(a => a.severity === 'MINOR').length,
    warning: filteredAlarms.filter(a => a.severity === 'WARNING').length,
    active: filteredAlarms.filter(a => a.alarm_state === 'ACTIVE').length,
    acknowledged: filteredAlarms.filter(a => a.ack_state === 'ACKNOWLEDGED').length
  };

  if (loading && alarms.length === 0) {
    return (
      <div className="alarm-manager">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading alarms...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="alarm-manager">
      <div className="alarm-header">
        <h3>Alarm & Event Management</h3>
        <div className="alarm-actions">
          <button 
            className="btn btn-primary"
            onClick={() => setShowAlarmForm(true)}
          >
            Generate Test Alarm
          </button>
          <button 
            className="btn btn-secondary"
            onClick={() => setShowCorrelationForm(true)}
            disabled={selectedAlarms.length < 2}
          >
            Correlate Selected ({selectedAlarms.length})
          </button>
        </div>
      </div>

      {error && (
        <div className="error-message">
          <span className="error-icon">⚠</span>
          {error}
        </div>
      )}

      <div className="alarm-stats">
        <div className="stat-card">
          <span className="stat-value">{alarmStats.total}</span>
          <span className="stat-label">Total Alarms</span>
        </div>
        <div className="stat-card critical">
          <span className="stat-value">{alarmStats.critical}</span>
          <span className="stat-label">Critical</span>
        </div>
        <div className="stat-card major">
          <span className="stat-value">{alarmStats.major}</span>
          <span className="stat-label">Major</span>
        </div>
        <div className="stat-card minor">
          <span className="stat-value">{alarmStats.minor}</span>
          <span className="stat-label">Minor</span>
        </div>
        <div className="stat-card warning">
          <span className="stat-value">{alarmStats.warning}</span>
          <span className="stat-label">Warning</span>
        </div>
        <div className="stat-card active">
          <span className="stat-value">{alarmStats.active}</span>
          <span className="stat-label">Active</span>
        </div>
      </div>

      <div className="alarm-filters">
        <select
          value={filter.severity}
          onChange={(e) => setFilter({ ...filter, severity: e.target.value })}
          className="filter-select"
        >
          <option value="">All Severities</option>
          <option value="CRITICAL">Critical</option>
          <option value="MAJOR">Major</option>
          <option value="MINOR">Minor</option>
          <option value="WARNING">Warning</option>
        </select>
        
        <input
          type="text"
          placeholder="Search alarms..."
          value={filter.search}
          onChange={(e) => setFilter({ ...filter, search: e.target.value })}
          className="search-input"
        />
        
        <button onClick={loadAlarms} className="btn btn-outline btn-sm">
          Refresh
        </button>
      </div>

      <div className="alarm-list">
        {filteredAlarms.map(alarm => (
          <div key={alarm.id} className={`alarm-item ${alarm.severity.toLowerCase()}`}>
            <div className="alarm-checkbox">
              <input
                type="checkbox"
                checked={selectedAlarms.includes(alarm.id)}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedAlarms([...selectedAlarms, alarm.id]);
                  } else {
                    setSelectedAlarms(selectedAlarms.filter(id => id !== alarm.id));
                  }
                }}
              />
            </div>
            
            <div className="alarm-content" onClick={() => setSelectedAlarm(alarm)}>
              <div className="alarm-header-info">
                <div className="alarm-severity">
                  <span className={`severity-badge ${alarm.severity.toLowerCase()}`}>
                    {alarm.severity}
                  </span>
                  <span className={`state-badge ${alarm.alarm_state.toLowerCase()}`}>
                    {alarm.alarm_state}
                  </span>
                  {alarm.ack_state === 'ACKNOWLEDGED' && (
                    <span className="ack-badge">ACK</span>
                  )}
                </div>
                <div className="alarm-time">
                  {new Date(alarm.event_time).toLocaleString()}
                </div>
              </div>
              
              <div className="alarm-details">
                <h5>{alarm.specific_problem}</h5>
                <p><strong>Object:</strong> {alarm.managed_object_id}</p>
                <p><strong>Cause:</strong> {alarm.probable_cause}</p>
                {alarm.additional_text && (
                  <p><strong>Details:</strong> {alarm.additional_text}</p>
                )}
                {alarm.correlated_alarms && alarm.correlated_alarms.length > 0 && (
                  <p><strong>Correlated:</strong> {alarm.correlated_alarms.length} related alarms</p>
                )}
              </div>
            </div>
            
            <div className="alarm-actions">
              {alarm.ack_state === 'UNACKNOWLEDGED' && (
                <button
                  className="btn btn-sm btn-warning"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleAcknowledgeAlarm(alarm.id);
                  }}
                >
                  Acknowledge
                </button>
              )}
              {alarm.alarm_state === 'ACTIVE' && (
                <button
                  className="btn btn-sm btn-success"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClearAlarm(alarm.id);
                  }}
                >
                  Clear
                </button>
              )}
            </div>
          </div>
        ))}
        
        {filteredAlarms.length === 0 && !loading && (
          <div className="no-alarms">
            <p>No alarms found matching the current filters.</p>
          </div>
        )}
      </div>

      {showAlarmForm && (
        <AlarmForm
          onSubmit={handleGenerateAlarm}
          onCancel={() => setShowAlarmForm(false)}
        />
      )}

      {showCorrelationForm && (
        <CorrelationForm
          selectedAlarms={selectedAlarms}
          onSubmit={handleCorrelateAlarms}
          onCancel={() => setShowCorrelationForm(false)}
        />
      )}

      {selectedAlarm && (
        <AlarmDetails
          alarm={selectedAlarm}
          onClose={() => setSelectedAlarm(null)}
          onAcknowledge={handleAcknowledgeAlarm}
          onClear={handleClearAlarm}
        />
      )}
    </div>
  );
};

const AlarmForm = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    managed_object_id: '',
    alarm_type: '',
    severity: 'MINOR',
    probable_cause: '',
    specific_problem: '',
    additional_text: '',
    event_time: new Date().toISOString().slice(0, 16)
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit({
      ...formData,
      event_time: new Date(formData.event_time).toISOString()
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Generate Test Alarm</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Managed Object ID</label>
            <input
              type="text"
              value={formData.managed_object_id}
              onChange={(e) => setFormData({ ...formData, managed_object_id: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Alarm Type</label>
            <input
              type="text"
              value={formData.alarm_type}
              onChange={(e) => setFormData({ ...formData, alarm_type: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Severity</label>
            <select
              value={formData.severity}
              onChange={(e) => setFormData({ ...formData, severity: e.target.value })}
              required
            >
              <option value="CRITICAL">Critical</option>
              <option value="MAJOR">Major</option>
              <option value="MINOR">Minor</option>
              <option value="WARNING">Warning</option>
            </select>
          </div>
          <div className="form-group">
            <label>Probable Cause</label>
            <input
              type="text"
              value={formData.probable_cause}
              onChange={(e) => setFormData({ ...formData, probable_cause: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Specific Problem</label>
            <input
              type="text"
              value={formData.specific_problem}
              onChange={(e) => setFormData({ ...formData, specific_problem: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Additional Text</label>
            <textarea
              value={formData.additional_text}
              onChange={(e) => setFormData({ ...formData, additional_text: e.target.value })}
              rows="3"
            />
          </div>
          <div className="form-group">
            <label>Event Time</label>
            <input
              type="datetime-local"
              value={formData.event_time}
              onChange={(e) => setFormData({ ...formData, event_time: e.target.value })}
              required
            />
          </div>
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Generate Alarm
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const CorrelationForm = ({ selectedAlarms, onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    correlation_type: 'RELATED',
    root_cause: '',
    description: ''
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(formData);
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Correlate Alarms</h4>
        <p>Correlating {selectedAlarms.length} selected alarms</p>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Correlation Type</label>
            <select
              value={formData.correlation_type}
              onChange={(e) => setFormData({ ...formData, correlation_type: e.target.value })}
              required
            >
              <option value="RELATED">Related</option>
              <option value="DUPLICATE">Duplicate</option>
              <option value="ROOT_CAUSE">Root Cause</option>
              <option value="SYMPTOM">Symptom</option>
            </select>
          </div>
          <div className="form-group">
            <label>Root Cause</label>
            <input
              type="text"
              value={formData.root_cause}
              onChange={(e) => setFormData({ ...formData, root_cause: e.target.value })}
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
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create Correlation
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const AlarmDetails = ({ alarm, onClose, onAcknowledge, onClear }) => {
  return (
    <div className="modal-overlay">
      <div className="modal-content large">
        <div className="alarm-details-header">
          <h4>Alarm Details</h4>
          <button className="close-button" onClick={onClose}>×</button>
        </div>
        
        <div className="alarm-details-content">
          <div className="detail-section">
            <h5>Basic Information</h5>
            <div className="detail-grid">
              <div className="detail-item">
                <label>Alarm ID:</label>
                <span>{alarm.id}</span>
              </div>
              <div className="detail-item">
                <label>Managed Object:</label>
                <span>{alarm.managed_object_id}</span>
              </div>
              <div className="detail-item">
                <label>Alarm Type:</label>
                <span>{alarm.alarm_type}</span>
              </div>
              <div className="detail-item">
                <label>Severity:</label>
                <span className={`severity-badge ${alarm.severity.toLowerCase()}`}>
                  {alarm.severity}
                </span>
              </div>
              <div className="detail-item">
                <label>State:</label>
                <span className={`state-badge ${alarm.alarm_state.toLowerCase()}`}>
                  {alarm.alarm_state}
                </span>
              </div>
              <div className="detail-item">
                <label>Acknowledgment:</label>
                <span className={`ack-badge ${alarm.ack_state.toLowerCase()}`}>
                  {alarm.ack_state}
                </span>
              </div>
            </div>
          </div>
          
          <div className="detail-section">
            <h5>Problem Description</h5>
            <div className="detail-grid">
              <div className="detail-item full-width">
                <label>Specific Problem:</label>
                <span>{alarm.specific_problem}</span>
              </div>
              <div className="detail-item full-width">
                <label>Probable Cause:</label>
                <span>{alarm.probable_cause}</span>
              </div>
              {alarm.additional_text && (
                <div className="detail-item full-width">
                  <label>Additional Information:</label>
                  <span>{alarm.additional_text}</span>
                </div>
              )}
            </div>
          </div>
          
          <div className="detail-section">
            <h5>Timestamps</h5>
            <div className="detail-grid">
              <div className="detail-item">
                <label>Event Time:</label>
                <span>{new Date(alarm.event_time).toLocaleString()}</span>
              </div>
              {alarm.ack_time && (
                <div className="detail-item">
                  <label>Acknowledged:</label>
                  <span>{new Date(alarm.ack_time).toLocaleString()}</span>
                </div>
              )}
              {alarm.clear_time && (
                <div className="detail-item">
                  <label>Cleared:</label>
                  <span>{new Date(alarm.clear_time).toLocaleString()}</span>
                </div>
              )}
              {alarm.ack_user && (
                <div className="detail-item">
                  <label>Acknowledged By:</label>
                  <span>{alarm.ack_user}</span>
                </div>
              )}
            </div>
          </div>
          
          {alarm.correlated_alarms && alarm.correlated_alarms.length > 0 && (
            <div className="detail-section">
              <h5>Correlated Alarms</h5>
              <div className="correlated-list">
                {alarm.correlated_alarms.map(correlatedId => (
                  <span key={correlatedId} className="correlated-alarm">
                    {correlatedId}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
        
        <div className="alarm-details-actions">
          {alarm.ack_state === 'UNACKNOWLEDGED' && (
            <button
              className="btn btn-warning"
              onClick={() => {
                onAcknowledge(alarm.id);
                onClose();
              }}
            >
              Acknowledge
            </button>
          )}
          {alarm.alarm_state === 'ACTIVE' && (
            <button
              className="btn btn-success"
              onClick={() => {
                onClear(alarm.id);
                onClose();
              }}
            >
              Clear Alarm
            </button>
          )}
          <button className="btn btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default AlarmManager;