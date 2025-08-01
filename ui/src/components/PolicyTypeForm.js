/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState } from 'react';
import './PolicyTypeForm.css';
import { ErrorDisplay } from './ErrorDisplay';

const PolicyTypeForm = ({ onSubmit, onCancel, loading, error }) => {
  const [formData, setFormData] = useState({
    policy_type_id: '',
    name: '',
    description: '',
    schema: ''
  });
  const [schemaError, setSchemaError] = useState(null);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSchemaChange = (e) => {
    const value = e.target.value;
    setFormData(prev => ({
      ...prev,
      schema: value
    }));

    // Validate JSON schema
    if (value.trim()) {
      try {
        JSON.parse(value);
        setSchemaError(null);
      } catch (err) {
        setSchemaError('Invalid JSON format');
      }
    } else {
      setSchemaError(null);
    }
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (!formData.policy_type_id.trim()) {
      setSchemaError('Policy Type ID is required');
      return;
    }

    if (!formData.schema.trim()) {
      setSchemaError('Policy schema is required');
      return;
    }

    try {
      const parsedSchema = JSON.parse(formData.schema);
      onSubmit({
        ...formData,
        schema: parsedSchema
      });
    } catch (err) {
      setSchemaError('Invalid JSON schema format');
    }
  };

  const insertSampleSchema = () => {
    const sampleSchema = {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "title": "Sample Policy Schema",
      "description": "A sample policy schema for demonstration",
      "properties": {
        "scope": {
          "type": "object",
          "properties": {
            "ueId": {
              "type": "string",
              "description": "User Equipment Identifier"
            },
            "cellId": {
              "type": "string", 
              "description": "Cell Identifier"
            }
          },
          "required": ["ueId"]
        },
        "statement": {
          "type": "object",
          "properties": {
            "priorityLevel": {
              "type": "integer",
              "minimum": 1,
              "maximum": 15,
              "description": "QoS priority level"
            },
            "qosParameters": {
              "type": "object",
              "properties": {
                "maxBitrate": {
                  "type": "integer",
                  "description": "Maximum bitrate in kbps"
                },
                "guaranteedBitrate": {
                  "type": "integer", 
                  "description": "Guaranteed bitrate in kbps"
                }
              }
            }
          },
          "required": ["priorityLevel"]
        }
      },
      "required": ["scope", "statement"]
    };

    setFormData(prev => ({
      ...prev,
      schema: JSON.stringify(sampleSchema, null, 2)
    }));
    setSchemaError(null);
  };

  const formatSchema = () => {
    if (formData.schema.trim()) {
      try {
        const parsed = JSON.parse(formData.schema);
        setFormData(prev => ({
          ...prev,
          schema: JSON.stringify(parsed, null, 2)
        }));
        setSchemaError(null);
      } catch (err) {
        setSchemaError('Cannot format invalid JSON');
      }
    }
  };

  return (
    <div className="policy-type-form-overlay">
      <div className="policy-type-form-modal">
        <div className="form-header">
          <h3>Create Policy Type</h3>
          <button onClick={onCancel} className="close-btn">×</button>
        </div>

        {error && <ErrorDisplay error={error} />}

        <form onSubmit={handleSubmit} className="policy-type-form">
          <div className="form-group">
            <label htmlFor="policy_type_id">Policy Type ID *</label>
            <input
              type="text"
              id="policy_type_id"
              name="policy_type_id"
              value={formData.policy_type_id}
              onChange={handleInputChange}
              placeholder="e.g., qos-policy-type-1"
              required
              disabled={loading}
            />
            <small>Unique identifier for the policy type</small>
          </div>

          <div className="form-group">
            <label htmlFor="name">Name</label>
            <input
              type="text"
              id="name"
              name="name"
              value={formData.name}
              onChange={handleInputChange}
              placeholder="e.g., QoS Management Policy"
              disabled={loading}
            />
            <small>Human-readable name for the policy type</small>
          </div>

          <div className="form-group">
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              name="description"
              value={formData.description}
              onChange={handleInputChange}
              placeholder="Describe what this policy type is used for..."
              rows="3"
              disabled={loading}
            />
            <small>Optional description of the policy type's purpose</small>
          </div>

          <div className="form-group">
            <div className="schema-header">
              <label htmlFor="schema">Policy Schema (JSON) *</label>
              <div className="schema-actions">
                <button
                  type="button"
                  onClick={insertSampleSchema}
                  className="sample-btn"
                  disabled={loading}
                >
                  Insert Sample
                </button>
                <button
                  type="button"
                  onClick={formatSchema}
                  className="format-btn"
                  disabled={loading}
                >
                  Format JSON
                </button>
              </div>
            </div>
            <textarea
              id="schema"
              name="schema"
              value={formData.schema}
              onChange={handleSchemaChange}
              placeholder="Enter JSON schema definition..."
              rows="15"
              className={`schema-textarea ${schemaError ? 'error' : ''}`}
              required
              disabled={loading}
            />
            {schemaError && (
              <div className="error-message">{schemaError}</div>
            )}
            <small>JSON Schema that defines the structure and validation rules for policy instances</small>
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
              disabled={loading || !!schemaError || !formData.policy_type_id.trim() || !formData.schema.trim()}
            >
              {loading ? 'Creating...' : 'Create Policy Type'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default PolicyTypeForm;