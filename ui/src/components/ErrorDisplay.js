/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { formatAPIError } from '../hooks/useAPI';

/**
 * Component for displaying API errors with retry functionality
 */
function ErrorDisplay({ error, onRetry, className = '' }) {
  const errorMessage = formatAPIError(error);

  return (
    <div className={`error-display ${className}`}>
      <div className="error-icon">⚠️</div>
      <div className="error-content">
        <h3>Error</h3>
        <p>{errorMessage}</p>
        {onRetry && (
          <button onClick={onRetry} className="retry-button">
            Retry
          </button>
        )}
      </div>
    </div>
  );
}

/**
 * Component for displaying loading states
 */
function LoadingDisplay({ message = 'Loading...', className = '' }) {
  return (
    <div className={`loading-display ${className}`}>
      <div className="loading-spinner">⟳</div>
      <p>{message}</p>
    </div>
  );
}

/**
 * Component for displaying connection status with detailed state information
 */
function ConnectionStatus({ connected, connecting, error, connectionState, reconnectAttempts, onReconnect }) {
  const getStatusText = () => {
    if (connected) {
      return 'Real-time Updates Active';
    } else if (connecting) {
      return 'Connecting...';
    } else if (reconnectAttempts > 0) {
      return `Reconnecting (${reconnectAttempts}/5)`;
    } else if (error) {
      return `Connection Error: ${formatAPIError(error)}`;
    } else {
      return 'Disconnected';
    }
  };

  const getStatusClass = () => {
    if (connected) return 'connected';
    if (connecting) return 'connecting';
    return 'disconnected';
  };

  const getStatusIndicator = () => {
    if (connected) return '●';
    if (connecting) return '◐';
    return '○';
  };

  return (
    <div className={`connection-status ${getStatusClass()}`}>
      <span className="status-indicator">{getStatusIndicator()}</span>
      <span className="status-text">{getStatusText()}</span>
      {connectionState && (
        <span className="connection-state">({connectionState})</span>
      )}
      {!connected && !connecting && onReconnect && (
        <button 
          className="reconnect-button" 
          onClick={onReconnect}
          title="Manually reconnect WebSocket"
        >
          Reconnect
        </button>
      )}
    </div>
  );
}

export { ErrorDisplay, LoadingDisplay, ConnectionStatus };