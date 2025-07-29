/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ComponentStatusPanel from './ComponentStatusPanel';

describe('ComponentStatusPanel', () => {
  const mockComponent = {
    name: 'Test Component',
    type: 'e2manager',
    status: 'running',
    version: '1.0.0',
    endpoints: [
      { name: 'gRPC', url: 'localhost:8080' },
      { name: 'REST', url: 'http://localhost:8081' }
    ],
    lastUpdated: '2023-01-01T12:00:00Z'
  };

  test('renders component information correctly', () => {
    render(<ComponentStatusPanel component={mockComponent} />);
    
    expect(screen.getByText('Test Component')).toBeInTheDocument();
    expect(screen.getByText('e2manager')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
    expect(screen.getByText('1.0.0')).toBeInTheDocument();
  });

  test('displays loading state', () => {
    render(<ComponentStatusPanel component={mockComponent} loading={true} />);
    
    expect(screen.getByText('Loading component status...')).toBeInTheDocument();
    expect(screen.getByText('⟳')).toBeInTheDocument();
  });

  test('displays error state with retry button', () => {
    const mockError = new Error('Failed to load');
    const mockOnRefresh = jest.fn();
    
    render(
      <ComponentStatusPanel 
        component={mockComponent} 
        error={mockError} 
        onRefresh={mockOnRefresh} 
      />
    );
    
    expect(screen.getByText('Failed to load component status')).toBeInTheDocument();
    expect(screen.getByText('Failed to load')).toBeInTheDocument();
    
    const retryButton = screen.getByText('Retry');
    fireEvent.click(retryButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  test('displays not found state when component is null', () => {
    render(<ComponentStatusPanel component={null} />);
    
    expect(screen.getByText('Component Not Found')).toBeInTheDocument();
    expect(screen.getByText('Component not discovered or not deployed')).toBeInTheDocument();
  });

  test('shows endpoints information', () => {
    render(<ComponentStatusPanel component={mockComponent} />);
    
    expect(screen.getByText('gRPC:')).toBeInTheDocument();
    expect(screen.getByText('localhost:8080')).toBeInTheDocument();
    expect(screen.getByText('REST:')).toBeInTheDocument();
    expect(screen.getByText('http://localhost:8081')).toBeInTheDocument();
  });

  test('calls refresh function when refresh button is clicked', () => {
    const mockOnRefresh = jest.fn();
    
    render(<ComponentStatusPanel component={mockComponent} onRefresh={mockOnRefresh} />);
    
    const refreshButton = screen.getByText('Refresh');
    fireEvent.click(refreshButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  test('applies correct status classes', () => {
    const { rerender } = render(<ComponentStatusPanel component={{...mockComponent, status: 'running'}} />);
    expect(screen.getByText('●')).toBeInTheDocument();

    rerender(<ComponentStatusPanel component={{...mockComponent, status: 'stopped'}} />);
    expect(screen.getByText('○')).toBeInTheDocument();

    rerender(<ComponentStatusPanel component={{...mockComponent, status: 'starting'}} />);
    expect(screen.getByText('◐')).toBeInTheDocument();

    rerender(<ComponentStatusPanel component={{...mockComponent, status: 'error'}} />);
    expect(screen.getByText('⚠')).toBeInTheDocument();
  });

  test('renders children content', () => {
    render(
      <ComponentStatusPanel component={mockComponent}>
        <div>Custom child content</div>
      </ComponentStatusPanel>
    );
    
    expect(screen.getByText('Custom child content')).toBeInTheDocument();
  });
});