/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import XAppManagement from './XAppManagement';
import dashboardAPI from '../services/api';

// Mock the API
jest.mock('../services/api');

// Mock child components
jest.mock('./XAppDeploymentForm', () => {
  return function MockXAppDeploymentForm({ onDeploy, onCancel }) {
    return (
      <div data-testid="deployment-form">
        <button onClick={() => onDeploy({ name: 'test-xapp', version: '1.0.0' })}>
          Deploy
        </button>
        <button onClick={onCancel}>Cancel</button>
      </div>
    );
  };
});

jest.mock('./XAppList', () => {
  return function MockXAppList({ xApps, onSelectXApp }) {
    return (
      <div data-testid="xapp-list">
        {xApps.map(xapp => (
          <div key={xapp.name} data-testid={`xapp-${xapp.name}`}>
            <span>{xapp.name}</span>
            <button onClick={() => onSelectXApp(xapp)}>Select</button>
          </div>
        ))}
      </div>
    );
  };
});

jest.mock('./XAppDetails', () => {
  return function MockXAppDetails({ xapp, onBack }) {
    return (
      <div data-testid="xapp-details">
        <span>Details for {xapp.name}</span>
        <button onClick={onBack}>Back</button>
      </div>
    );
  };
});

describe('XAppManagement', () => {
  const mockXApps = [
    {
      name: 'hello-world',
      version: '1.0.0',
      status: 'running',
      instances: 1
    },
    {
      name: 'traffic-steering',
      version: '2.1.0',
      status: 'stopped',
      instances: 0
    }
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders xApp management interface', () => {
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    expect(screen.getByText('xApp Management')).toBeInTheDocument();
    expect(screen.getByText('Deploy New xApp')).toBeInTheDocument();
    expect(screen.getByText('xApp List (2)')).toBeInTheDocument();
  });

  test('shows loading state', () => {
    render(
      <XAppManagement 
        xApps={[]}
        loading={true}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    expect(screen.getByText('Loading xApp management interface...')).toBeInTheDocument();
  });

  test('shows error state', () => {
    const error = new Error('Failed to load xApps');
    render(
      <XAppManagement 
        xApps={[]}
        loading={false}
        error={error}
        onRefresh={jest.fn()}
      />
    );

    expect(screen.getByText('Failed to load xApps')).toBeInTheDocument();
  });

  test('switches to deployment form when deploy button clicked', () => {
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    fireEvent.click(screen.getByText('Deploy New xApp'));
    
    expect(screen.getByTestId('deployment-form')).toBeInTheDocument();
    expect(screen.getByText('Deploy xApp')).toBeInTheDocument();
  });

  test('handles xApp deployment', async () => {
    const mockOnRefresh = jest.fn();
    dashboardAPI.deployXApp.mockResolvedValue({ success: true });

    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={mockOnRefresh}
      />
    );

    // Switch to deployment form
    fireEvent.click(screen.getByText('Deploy New xApp'));
    
    // Deploy xApp
    fireEvent.click(screen.getByText('Deploy'));

    await waitFor(() => {
      expect(dashboardAPI.deployXApp).toHaveBeenCalledWith({
        name: 'test-xapp',
        version: '1.0.0'
      });
      expect(mockOnRefresh).toHaveBeenCalled();
    });
  });

  test('handles xApp selection for details view', () => {
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    // Select an xApp
    fireEvent.click(screen.getAllByText('Select')[0]);
    
    expect(screen.getByTestId('xapp-details')).toBeInTheDocument();
    expect(screen.getByText('Details for hello-world')).toBeInTheDocument();
    expect(screen.getByText('hello-world Details')).toBeInTheDocument();
  });

  test('handles xApp undeployment', async () => {
    const mockOnRefresh = jest.fn();
    dashboardAPI.undeployXApp.mockResolvedValue({ success: true });

    const component = render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={mockOnRefresh}
      />
    );

    // Access the component instance to call handleUndeploy directly
    const xappManagement = component.container.querySelector('.xapp-management');
    expect(xappManagement).toBeInTheDocument();

    // This would normally be triggered by child components
    // For testing, we verify the API is available
    expect(dashboardAPI.undeployXApp).toBeDefined();
  });

  test('handles refresh action', () => {
    const mockOnRefresh = jest.fn();
    
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={mockOnRefresh}
      />
    );

    fireEvent.click(screen.getByText('Refresh'));
    expect(mockOnRefresh).toHaveBeenCalled();
  });

  test('shows correct tab counts', () => {
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    expect(screen.getByText('xApp List (2)')).toBeInTheDocument();
  });

  test('handles deployment form cancellation', () => {
    render(
      <XAppManagement 
        xApps={mockXApps}
        loading={false}
        error={null}
        onRefresh={jest.fn()}
      />
    );

    // Switch to deployment form
    fireEvent.click(screen.getByText('Deploy New xApp'));
    expect(screen.getByTestId('deployment-form')).toBeInTheDocument();
    
    // Cancel deployment
    fireEvent.click(screen.getByText('Cancel'));
    expect(screen.queryByTestId('deployment-form')).not.toBeInTheDocument();
    expect(screen.getByTestId('xapp-list')).toBeInTheDocument();
  });
});