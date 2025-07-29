/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ComponentDiscovery from './ComponentDiscovery';

describe('ComponentDiscovery', () => {
  const mockComponents = [
    {
      id: 'e2mgr-1',
      name: 'E2 Manager',
      type: 'e2manager',
      status: 'running',
      version: '1.0.0',
      endpoints: [{ name: 'gRPC', url: 'localhost:8080' }],
      lastUpdated: '2023-01-01T12:00:00Z'
    },
    {
      id: 'submgr-1',
      name: 'Subscription Manager',
      type: 'submgr',
      status: 'running',
      version: '1.1.0',
      endpoints: [{ name: 'REST', url: 'http://localhost:8081' }],
      lastUpdated: '2023-01-01T12:05:00Z'
    },
    {
      id: 'appmgr-1',
      name: 'App Manager',
      type: 'appmgr',
      status: 'stopped',
      version: '1.2.0',
      lastUpdated: '2023-01-01T12:10:00Z'
    },
    {
      id: 'xapp-1',
      name: 'Hello World xApp',
      type: 'xapp',
      status: 'running',
      version: '1.0.0',
      lastUpdated: '2023-01-01T12:15:00Z'
    }
  ];

  test('renders component discovery with statistics', () => {
    render(<ComponentDiscovery components={mockComponents} />);
    
    expect(screen.getByText('Component Discovery')).toBeInTheDocument();
    expect(screen.getByText('4')).toBeInTheDocument(); // Total components
    expect(screen.getByText('3')).toBeInTheDocument(); // Running components
  });

  test('displays loading state', () => {
    render(<ComponentDiscovery components={[]} loading={true} />);
    
    expect(screen.getByText('Discovering O-RAN SC components...')).toBeInTheDocument();
    expect(screen.getByText('⟳')).toBeInTheDocument();
  });

  test('displays error state with retry button', () => {
    const mockError = new Error('Discovery failed');
    const mockOnRefresh = jest.fn();
    
    render(
      <ComponentDiscovery 
        components={[]} 
        error={mockError} 
        onRefresh={mockOnRefresh} 
      />
    );
    
    expect(screen.getByText('Failed to discover components')).toBeInTheDocument();
    expect(screen.getByText('Discovery failed')).toBeInTheDocument();
    
    const retryButton = screen.getByText('Retry Discovery');
    fireEvent.click(retryButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  test('displays no components message when empty', () => {
    render(<ComponentDiscovery components={[]} />);
    
    expect(screen.getByText('No O-RAN SC components discovered')).toBeInTheDocument();
    expect(screen.getByText('Make sure O-RAN SC components are deployed and accessible')).toBeInTheDocument();
  });

  test('groups components by type correctly', () => {
    const { container } = render(<ComponentDiscovery components={mockComponents} />);
    
    // Check for component groups by looking for group headers
    const componentGroups = container.querySelectorAll('.component-group');
    expect(componentGroups).toHaveLength(4);
    
    // Check for group titles using class selector
    const groupTitles = container.querySelectorAll('.group-title');
    expect(groupTitles).toHaveLength(4);
    
    // Check for group counts
    const groupCounts = screen.getAllByText('(1)');
    expect(groupCounts).toHaveLength(4);
  });

  test('displays component details correctly', () => {
    render(<ComponentDiscovery components={mockComponents} />);
    
    // Check unique component names
    expect(screen.getByText('Hello World xApp')).toBeInTheDocument();
    
    // Check status indicators
    expect(screen.getAllByText('●')).toHaveLength(3); // Running components
    expect(screen.getByText('○')).toBeInTheDocument(); // Stopped component
    
    // Check that we have the right number of components displayed
    const nodeNames = screen.getAllByText(/Manager|xApp/);
    expect(nodeNames.length).toBeGreaterThanOrEqual(4);
  });

  test('shows component versions and endpoints', () => {
    render(<ComponentDiscovery components={mockComponents} />);
    
    // Check that versions are displayed (using getAllByText since there might be duplicates)
    const versions = screen.getAllByText(/^1\.\d+\.\d+$/);
    expect(versions.length).toBeGreaterThanOrEqual(3);
    
    // Check that version labels are present
    expect(screen.getAllByText('Version:')).toHaveLength(4);
    
    // Check endpoints labels
    expect(screen.getAllByText('Endpoints:')).toHaveLength(2); // Only 2 components have endpoints
  });

  test('calls refresh function when refresh button is clicked', () => {
    const mockOnRefresh = jest.fn();
    
    render(<ComponentDiscovery components={mockComponents} onRefresh={mockOnRefresh} />);
    
    const refreshButton = screen.getByText('Refresh Discovery');
    fireEvent.click(refreshButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  test('displays last discovery time', () => {
    render(<ComponentDiscovery components={mockComponents} onRefresh={() => {}} />);
    
    expect(screen.getByText(/Last discovery:/)).toBeInTheDocument();
  });

  test('applies correct status classes to component nodes', () => {
    const { container } = render(<ComponentDiscovery components={mockComponents} />);
    
    const runningNodes = container.querySelectorAll('.component-node.running');
    const stoppedNodes = container.querySelectorAll('.component-node.stopped');
    
    expect(runningNodes).toHaveLength(3);
    expect(stoppedNodes).toHaveLength(1);
  });

  test('shows component type icons', () => {
    render(<ComponentDiscovery components={mockComponents} />);
    
    expect(screen.getByText('📡')).toBeInTheDocument(); // E2 Manager icon
    expect(screen.getByText('📋')).toBeInTheDocument(); // Subscription Manager icon
    expect(screen.getByText('📦')).toBeInTheDocument(); // App Manager icon
    expect(screen.getByText('⚡')).toBeInTheDocument(); // xApp icon
  });
});