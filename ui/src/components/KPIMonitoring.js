/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState, useEffect } from 'react';
import './KPIMonitoring.css';
import dashboardAPI from '../services/api';

const KPIMonitoring = () => {
  const [kpis, setKpis] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState({ type: '' });
  const [selectedKPI, setSelectedKPI] = useState(null);
  const [showKPIForm, setShowKPIForm] = useState(false);
  const [showCollectionForm, setShowCollectionForm] = useState(false);
  const [kpiData, setKpiData] = useState([]);

  useEffect(() => {
    loadKPIs();
    const interval = setInterval(loadKPIs, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [filter]);

  const loadKPIs = async () => {
    try {
      setLoading(true);
      const data = await dashboardAPI.getKPIs(filter);
      setKpis(data.kpis || []);
      setError(null);
    } catch (err) {
      console.error('Failed to load KPIs:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateKPI = async (kpiData) => {
    try {
      await dashboardAPI.createKPI(kpiData);
      setShowKPIForm(false);
      loadKPIs();
    } catch (err) {
      console.error('Failed to create KPI:', err);
      setError(err.message);
    }
  };

  const handleUpdateKPI = async (kpiId, updateData) => {
    try {
      await dashboardAPI.updateKPI(kpiId, updateData);
      setSelectedKPI(null);
      loadKPIs();
    } catch (err) {
      console.error('Failed to update KPI:', err);
      setError(err.message);
    }
  };

  const handleCollectKPIData = async (collectionRequest) => {
    try {
      const data = await dashboardAPI.collectKPIData(collectionRequest);
      setKpiData(data.collected_kpis || []);
      setShowCollectionForm(false);
    } catch (err) {
      console.error('Failed to collect KPI data:', err);
      setError(err.message);
    }
  };

  const getKPIStatus = (kpi) => {
    if (!kpi.threshold) return 'normal';
    
    const value = kpi.value;
    const threshold = kpi.threshold;
    
    if ((threshold.critical_min && value < threshold.critical_min) ||
        (threshold.critical_max && value > threshold.critical_max)) {
      return 'critical';
    }
    
    if ((threshold.warning_min && value < threshold.warning_min) ||
        (threshold.warning_max && value > threshold.warning_max)) {
      return 'warning';
    }
    
    return 'normal';
  };

  const kpiStats = {
    total: kpis.length,
    critical: kpis.filter(kpi => getKPIStatus(kpi) === 'critical').length,
    warning: kpis.filter(kpi => getKPIStatus(kpi) === 'warning').length,
    normal: kpis.filter(kpi => getKPIStatus(kpi) === 'normal').length,
    counter: kpis.filter(kpi => kpi.measurement_type === 'COUNTER').length,
    gauge: kpis.filter(kpi => kpi.measurement_type === 'GAUGE').length,
    histogram: kpis.filter(kpi => kpi.measurement_type === 'HISTOGRAM').length
  };

  if (loading && kpis.length === 0) {
    return (
      <div className="kpi-monitoring">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading KPIs...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="kpi-monitoring">
      <div className="kpi-header">
        <h3>Performance Monitoring</h3>
        <div className="kpi-actions">
          <button 
            className="btn btn-primary"
            onClick={() => setShowKPIForm(true)}
          >
            Create KPI
          </button>
          <button 
            className="btn btn-secondary"
            onClick={() => setShowCollectionForm(true)}
          >
            Collect Data
          </button>
        </div>
      </div>

      {error && (
        <div className="error-message">
          <span className="error-icon">⚠</span>
          {error}
        </div>
      )}

      <div className="kpi-stats">
        <div className="stat-card">
          <span className="stat-value">{kpiStats.total}</span>
          <span className="stat-label">Total KPIs</span>
        </div>
        <div className="stat-card critical">
          <span className="stat-value">{kpiStats.critical}</span>
          <span className="stat-label">Critical</span>
        </div>
        <div className="stat-card warning">
          <span className="stat-value">{kpiStats.warning}</span>
          <span className="stat-label">Warning</span>
        </div>
        <div className="stat-card normal">
          <span className="stat-value">{kpiStats.normal}</span>
          <span className="stat-label">Normal</span>
        </div>
        <div className="stat-card counter">
          <span className="stat-value">{kpiStats.counter}</span>
          <span className="stat-label">Counters</span>
        </div>
        <div className="stat-card gauge">
          <span className="stat-value">{kpiStats.gauge}</span>
          <span className="stat-label">Gauges</span>
        </div>
      </div>

      <div className="kpi-filters">
        <select
          value={filter.type}
          onChange={(e) => setFilter({ ...filter, type: e.target.value })}
          className="filter-select"
        >
          <option value="">All Types</option>
          <option value="COUNTER">Counter</option>
          <option value="GAUGE">Gauge</option>
          <option value="HISTOGRAM">Histogram</option>
        </select>
        
        <button onClick={loadKPIs} className="btn btn-outline btn-sm">
          Refresh
        </button>
      </div>

      <div className="kpi-grid">
        {kpis.map(kpi => (
          <div key={kpi.id} className={`kpi-card ${getKPIStatus(kpi)}`}>
            <div className="kpi-header-info">
              <h5>{kpi.name}</h5>
              <span className={`kpi-status ${getKPIStatus(kpi)}`}>
                {getKPIStatus(kpi).toUpperCase()}
              </span>
            </div>
            
            <div className="kpi-value">
              <span className="value">{kpi.value.toFixed(2)}</span>
              <span className="unit">{kpi.unit}</span>
            </div>
            
            <div className="kpi-details">
              <p>{kpi.description}</p>
              <div className="kpi-meta">
                <span className="measurement-type">{kpi.measurement_type}</span>
                <span className="object-id">{kpi.managed_object_id}</span>
                <span className="timestamp">
                  {new Date(kpi.timestamp).toLocaleString()}
                </span>
              </div>
            </div>
            
            {kpi.threshold && (
              <div className="kpi-thresholds">
                <div className="threshold-bar">
                  <div className="threshold-indicator" style={{
                    left: `${Math.min(100, Math.max(0, (kpi.value / (kpi.threshold.critical_max || 100)) * 100))}%`
                  }}></div>
                  {kpi.threshold.warning_max && (
                    <div className="threshold-line warning" style={{
                      left: `${(kpi.threshold.warning_max / (kpi.threshold.critical_max || 100)) * 100}%`
                    }}></div>
                  )}
                  {kpi.threshold.critical_max && (
                    <div className="threshold-line critical" style={{
                      left: `${(kpi.threshold.critical_max / (kpi.threshold.critical_max || 100)) * 100}%`
                    }}></div>
                  )}
                </div>
                <div className="threshold-labels">
                  {kpi.threshold.warning_max && (
                    <span className="threshold-label warning">
                      Warning: {kpi.threshold.warning_max}
                    </span>
                  )}
                  {kpi.threshold.critical_max && (
                    <span className="threshold-label critical">
                      Critical: {kpi.threshold.critical_max}
                    </span>
                  )}
                </div>
              </div>
            )}
            
            <div className="kpi-actions">
              <button
                className="btn btn-sm btn-outline"
                onClick={() => setSelectedKPI(kpi)}
              >
                Edit
              </button>
              <button
                className="btn btn-sm btn-outline"
                onClick={() => handleCollectKPIData({
                  kpi_ids: [kpi.id],
                  start_time: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                  end_time: new Date().toISOString(),
                  interval: '1h'
                })}
              >
                View History
              </button>
            </div>
          </div>
        ))}
        
        {kpis.length === 0 && !loading && (
          <div className="no-kpis">
            <p>No KPIs found. Create your first KPI to start monitoring performance.</p>
          </div>
        )}
      </div>

      {kpiData.length > 0 && (
        <div className="kpi-data-section">
          <h4>Collected KPI Data</h4>
          <div className="kpi-data-chart">
            <KPIChart data={kpiData} />
          </div>
        </div>
      )}

      {showKPIForm && (
        <KPIForm
          onSubmit={handleCreateKPI}
          onCancel={() => setShowKPIForm(false)}
        />
      )}

      {showCollectionForm && (
        <KPICollectionForm
          kpis={kpis}
          onSubmit={handleCollectKPIData}
          onCancel={() => setShowCollectionForm(false)}
        />
      )}

      {selectedKPI && (
        <KPIEditor
          kpi={selectedKPI}
          onSave={handleUpdateKPI}
          onCancel={() => setSelectedKPI(null)}
        />
      )}
    </div>
  );
};

const KPIForm = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    measurement_type: 'GAUGE',
    unit: '',
    managed_object_id: '',
    threshold: {
      warning_min: '',
      warning_max: '',
      critical_min: '',
      critical_max: ''
    }
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    
    // Clean up threshold values
    const threshold = {};
    Object.keys(formData.threshold).forEach(key => {
      const value = parseFloat(formData.threshold[key]);
      if (!isNaN(value)) {
        threshold[key] = value;
      }
    });

    onSubmit({
      ...formData,
      threshold: Object.keys(threshold).length > 0 ? threshold : null
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Create KPI</h4>
        <form onSubmit={handleSubmit}>
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
            <label>Measurement Type</label>
            <select
              value={formData.measurement_type}
              onChange={(e) => setFormData({ ...formData, measurement_type: e.target.value })}
              required
            >
              <option value="COUNTER">Counter</option>
              <option value="GAUGE">Gauge</option>
              <option value="HISTOGRAM">Histogram</option>
            </select>
          </div>
          <div className="form-group">
            <label>Unit</label>
            <input
              type="text"
              value={formData.unit}
              onChange={(e) => setFormData({ ...formData, unit: e.target.value })}
              placeholder="e.g., %, MB/s, count"
              required
            />
          </div>
          <div className="form-group">
            <label>Managed Object ID</label>
            <input
              type="text"
              value={formData.managed_object_id}
              onChange={(e) => setFormData({ ...formData, managed_object_id: e.target.value })}
              required
            />
          </div>
          
          <h5>Thresholds (Optional)</h5>
          <div className="threshold-grid">
            <div className="form-group">
              <label>Warning Min</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.warning_min}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, warning_min: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Warning Max</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.warning_max}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, warning_max: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Critical Min</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.critical_min}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, critical_min: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Critical Max</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.critical_max}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, critical_max: e.target.value }
                })}
              />
            </div>
          </div>
          
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Create KPI
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const KPICollectionForm = ({ kpis, onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    kpi_ids: [],
    start_time: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 16),
    end_time: new Date().toISOString().slice(0, 16),
    interval: '1h'
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit({
      ...formData,
      start_time: new Date(formData.start_time).toISOString(),
      end_time: new Date(formData.end_time).toISOString()
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Collect KPI Data</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Select KPIs</label>
            <div className="kpi-selection">
              {kpis.map(kpi => (
                <label key={kpi.id} className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.kpi_ids.includes(kpi.id)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setFormData({
                          ...formData,
                          kpi_ids: [...formData.kpi_ids, kpi.id]
                        });
                      } else {
                        setFormData({
                          ...formData,
                          kpi_ids: formData.kpi_ids.filter(id => id !== kpi.id)
                        });
                      }
                    }}
                  />
                  {kpi.name}
                </label>
              ))}
            </div>
          </div>
          <div className="form-group">
            <label>Start Time</label>
            <input
              type="datetime-local"
              value={formData.start_time}
              onChange={(e) => setFormData({ ...formData, start_time: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>End Time</label>
            <input
              type="datetime-local"
              value={formData.end_time}
              onChange={(e) => setFormData({ ...formData, end_time: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Interval</label>
            <select
              value={formData.interval}
              onChange={(e) => setFormData({ ...formData, interval: e.target.value })}
            >
              <option value="1m">1 minute</option>
              <option value="5m">5 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="1h">1 hour</option>
              <option value="1d">1 day</option>
            </select>
          </div>
          <div className="form-actions">
            <button type="button" onClick={onCancel} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Collect Data
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const KPIEditor = ({ kpi, onSave, onCancel }) => {
  const [formData, setFormData] = useState({
    value: kpi.value,
    threshold: kpi.threshold || {
      warning_min: '',
      warning_max: '',
      critical_min: '',
      critical_max: ''
    }
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    
    // Clean up threshold values
    const threshold = {};
    Object.keys(formData.threshold).forEach(key => {
      const value = parseFloat(formData.threshold[key]);
      if (!isNaN(value)) {
        threshold[key] = value;
      }
    });

    onSave(kpi.id, {
      ...formData,
      threshold: Object.keys(threshold).length > 0 ? threshold : null,
      timestamp: new Date().toISOString()
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h4>Edit KPI: {kpi.name}</h4>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Current Value</label>
            <input
              type="number"
              step="any"
              value={formData.value}
              onChange={(e) => setFormData({ ...formData, value: parseFloat(e.target.value) })}
              required
            />
          </div>
          
          <h5>Thresholds</h5>
          <div className="threshold-grid">
            <div className="form-group">
              <label>Warning Min</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.warning_min}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, warning_min: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Warning Max</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.warning_max}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, warning_max: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Critical Min</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.critical_min}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, critical_min: e.target.value }
                })}
              />
            </div>
            <div className="form-group">
              <label>Critical Max</label>
              <input
                type="number"
                step="any"
                value={formData.threshold.critical_max}
                onChange={(e) => setFormData({
                  ...formData,
                  threshold: { ...formData.threshold, critical_max: e.target.value }
                })}
              />
            </div>
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

const KPIChart = ({ data }) => {
  if (!data || data.length === 0) {
    return <div className="no-data">No data available</div>;
  }

  // Simple line chart representation using CSS
  const maxValue = Math.max(...data.map(d => d.value));
  const minValue = Math.min(...data.map(d => d.value));
  const range = maxValue - minValue || 1;

  return (
    <div className="simple-chart">
      <div className="chart-header">
        <span>Value Range: {minValue.toFixed(2)} - {maxValue.toFixed(2)}</span>
        <span>Data Points: {data.length}</span>
      </div>
      <div className="chart-container">
        {data.map((point, index) => (
          <div
            key={index}
            className="chart-point"
            style={{
              left: `${(index / (data.length - 1)) * 100}%`,
              bottom: `${((point.value - minValue) / range) * 100}%`
            }}
            title={`${new Date(point.timestamp).toLocaleString()}: ${point.value}`}
          />
        ))}
      </div>
      <div className="chart-axis">
        <span className="axis-label">Time →</span>
      </div>
    </div>
  );
};

export default KPIMonitoring;