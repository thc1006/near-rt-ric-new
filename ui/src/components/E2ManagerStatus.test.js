/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { render, screen } from '@testing-library/react';
import E2ManagerStatus from './E2ManagerStatus';

describe('E2ManagerStatus', () => {
  const mockComponent = {
    name: 'E2 Manager',
    type: 'e2manager',
    status: 'running',
    version: '1.0.0',
    metrics: {
      messagesProcessed: 1500,
      averageResponseTime: 25,
      errorRate: 0.02,
      uptime: '2d 4h 30m'
    },
    supportedFunctions: [
      { name: 'RAN Control' },
      { name: 'RAN Monitoring' }
    ]
  };

  const mockE2Nodes = [
    {
      nodeId: 'gnb-001',
      nodeType: 'gnb',
      connectionStatus: 'connected',
      plmnId: '001-01'
    },
    {
      nodeId: 'enb-002',
      nodeType: 'enb',
      connectionStatus: 'disconnected',
      plmnId: '001-01'
    },
    {
      nodeId: 'gnb-003',
      nodeType: 'gnb',
      connectionStatus: 'setup_failed',
      plmnId: '001-01'
    }
  ];

  test('renders E2 Manager component information', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={mockE2Nodes} />);
    
    expect(screen.getByText('E2 Manager')).toBeInTheDocument();
    expect(screen.getByText('e2manager')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
  });

  test('displays E2 node statistics correctly', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={mockE2Nodes} />);
    
    // Check for specific metric labels and values
    expect(screen.getByText('Total E2 Nodes:')).toBeInTheDocument();
    expect(screen.getByText('Connected:')).toBeInTheDocument();
    expect(screen.getByText('Disconnected:')).toBeInTheDocument();
    expect(screen.getByText('Setup Failed:')).toBeInTheDocument();
    
    // Check that we have the right number of metric items
    const metricItems = screen.getAllByText(/^(3|1)$/);
    expect(metricItems).toHaveLength(4); // Total, Connected, Disconnected, Setup Failed
  });

  test('displays performance metrics', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={mockE2Nodes} />);
    
    expect(screen.getByText('1,500')).toBeInTheDocument(); // Messages processed
    expect(screen.getByText('25ms')).toBeInTheDocument(); // Average response time
    expect(screen.getByText('2.00%')).toBeInTheDocument(); // Error rate
    expect(screen.getByText('2d 4h 30m')).toBeInTheDocument(); // Uptime
  });

  test('displays connected E2 nodes', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={mockE2Nodes} />);
    
    expect(screen.getByText('gnb-001')).toBeInTheDocument();
    expect(screen.getByText('enb-002')).toBeInTheDocument();
    expect(screen.getByText('gnb-003')).toBeInTheDocument();
  });

  test('displays supported RAN functions', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={mockE2Nodes} />);
    
    expect(screen.getByText('RAN Control')).toBeInTheDocument();
    expect(screen.getByText('RAN Monitoring')).toBeInTheDocument();
  });

  test('handles empty E2 nodes list', () => {
    render(<E2ManagerStatus component={mockComponent} e2Nodes={[]} />);
    
    // Check that all metrics show 0
    expect(screen.getByText('Total E2 Nodes:')).toBeInTheDocument();
    const zeroValues = screen.getAllByText('0');
    expect(zeroValues.length).toBeGreaterThanOrEqual(4); // At least 4 zeros for the metrics
  });

  test('limits displayed E2 nodes to 5', () => {
    const manyNodes = Array.from({ length: 7 }, (_, i) => ({
      nodeId: `node-${i + 1}`,
      nodeType: 'gnb',
      connectionStatus: 'connected'
    }));

    render(<E2ManagerStatus component={mockComponent} e2Nodes={manyNodes} />);
    
    expect(screen.getByText('+2 more nodes')).toBeInTheDocument();
  });

  test('displays loading state', () => {
    render(<E2ManagerStatus component={mockComponent} loading={true} />);
    
    expect(screen.getByText('Loading component status...')).toBeInTheDocument();
  });

  test('displays error state', () => {
    const mockError = new Error('Failed to load E2 Manager');
    
    render(<E2ManagerStatus component={mockComponent} error={mockError} />);
    
    expect(screen.getByText('Failed to load component status')).toBeInTheDocument();
    expect(screen.getByText('Failed to load E2 Manager')).toBeInTheDocument();
  });
});