/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React, { useState } from 'react';
import './PolicySchemaViewer.css';

const PolicySchemaViewer = ({ policyType, onClose }) => {
  const [viewMode, setViewMode] = useState('formatted'); // 'formatted' or 'raw'

  const formatSchema = (schema) => {
    try {
      if (typeof schema === 'string') {
        return JSON.stringify(JSON.parse(schema), null, 2);
      }
      return JSON.stringify(schema, null, 2);
    } catch (err) {
      return typeof schema === 'string' ? schema : JSON.stringify(schema);
    }
  };

  const renderSchemaTree = (schema, path = '') => {
    try {
      const schemaObj = typeof schema === 'string' ? JSON.parse(schema) : schema;
      return renderObjectTree(schemaObj, path);
    } catch (err) {
      return (
        <div className="schema-error">
          <p>Unable to parse schema structure</p>
          <pre>{typeof schema === 'string' ? schema : JSON.stringify(schema)}</pre>
        </div>
      );
    }
  };

  const renderObjectTree = (obj, path = '', level = 0) => {
    if (typeof obj !== 'object' || obj === null) {
      return <span className="schema-value">{String(obj)}</span>;
    }

    return (
      <div className={`schema-object level-${level}`}>
        {Object.entries(obj).map(([key, value]) => {
          const currentPath = path ? `${path}.${key}` : key;
          const isObject = typeof value === 'object' && value !== null;
          const isArray = Array.isArray(value);

          return (
            <div key={key} className="schema-property">
              <div className="property-header">
                <span className="property-key">{key}</span>
                {isArray && <span className="property-type">array</span>}
                {isObject && !isArray && <span className="property-type">object</span>}
                {!isObject && <span className="property-type">{typeof value}</span>}
              </div>
              
              <div className="property-value">
                {isObject ? (
                  renderObjectTree(value, currentPath, level + 1)
                ) : (
                  <span className={`schema-value ${typeof value}`}>
                    {String(value)}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  const extractSchemaInfo = (schema) => {
    try {
      const schemaObj = typeof schema === 'string' ? JSON.parse(schema) : schema;
      
      const info = {
        title: schemaObj.title || 'Untitled Schema',
        description: schemaObj.description || 'No description available',
        type: schemaObj.type || 'unknown',
        version: schemaObj.$schema || 'Unknown version',
        required: schemaObj.required || [],
        properties: schemaObj.properties ? Object.keys(schemaObj.properties) : []
      };

      return info;
    } catch (err) {
      return {
        title: 'Invalid Schema',
        description: 'Unable to parse schema information',
        type: 'unknown',
        version: 'Unknown',
        required: [],
        properties: []
      };
    }
  };

  const schemaInfo = extractSchemaInfo(policyType.schema);

  return (
    <div className="schema-viewer-overlay">
      <div className="schema-viewer-modal">
        <div className="viewer-header">
          <div className="header-info">
            <h3>Policy Schema: {policyType.policy_type_id}</h3>
            <p className="schema-title">{schemaInfo.title}</p>
          </div>
          <button onClick={onClose} className="close-btn">×</button>
        </div>

        <div className="schema-meta">
          <div className="meta-item">
            <span className="meta-label">Type:</span>
            <span className="meta-value">{schemaInfo.type}</span>
          </div>
          <div className="meta-item">
            <span className="meta-label">Schema Version:</span>
            <span className="meta-value">{schemaInfo.version}</span>
          </div>
          <div className="meta-item">
            <span className="meta-label">Properties:</span>
            <span className="meta-value">{schemaInfo.properties.length}</span>
          </div>
          <div className="meta-item">
            <span className="meta-label">Required Fields:</span>
            <span className="meta-value">{schemaInfo.required.length}</span>
          </div>
        </div>

        {schemaInfo.description !== 'No description available' && (
          <div className="schema-description">
            <h4>Description</h4>
            <p>{schemaInfo.description}</p>
          </div>
        )}

        {schemaInfo.required.length > 0 && (
          <div className="required-fields">
            <h4>Required Fields</h4>
            <div className="required-list">
              {schemaInfo.required.map(field => (
                <span key={field} className="required-field">{field}</span>
              ))}
            </div>
          </div>
        )}

        <div className="view-controls">
          <button
            className={`view-btn ${viewMode === 'formatted' ? 'active' : ''}`}
            onClick={() => setViewMode('formatted')}
          >
            Tree View
          </button>
          <button
            className={`view-btn ${viewMode === 'raw' ? 'active' : ''}`}
            onClick={() => setViewMode('raw')}
          >
            Raw JSON
          </button>
        </div>

        <div className="schema-content">
          {viewMode === 'formatted' ? (
            <div className="schema-tree">
              {renderSchemaTree(policyType.schema)}
            </div>
          ) : (
            <pre className="schema-raw">
              {formatSchema(policyType.schema)}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
};

export default PolicySchemaViewer;