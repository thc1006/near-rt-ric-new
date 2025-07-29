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
 * Component for displaying connection status
 */
function ConnectionStatus({ connected, error }) {
  if (connected) {
    return (
      <div className="connection-status connected">
        <span className="status-indicator">●</span>
        Connected
      </div>
    );
  }

  return (
    <div className="connection-status disconnected">
      <span className="status-indicator">●</span>
      {error ? `Connection Error: ${formatAPIError(error)}` : 'Disconnected'}
    </div>
  );
}

export { ErrorDisplay, LoadingDisplay, ConnectionStatus };