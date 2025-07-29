/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import XAppDeploymentForm from './XAppDeploymentForm';

describe('XAppDeploymentForm', () => {
  const mockOnDeploy = jest.fn();
  const mockOnCancel = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders deployment form', () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    expect(screen.getByText('Deploy New xApp')).toBeInTheDocument();
    expect(screen.getByLabelText('xApp Name *')).toBeInTheDocument();
    expect(screen.getByLabelText('Version *')).toBeInTheDocument();
    expect(screen.getByLabelText('Container Image *')).toBeInTheDocument();
    expect(screen.getByText('Deploy xApp')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  test('validates required fields', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    // Try to submit without filling required fields
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('xApp name is required')).toBeInTheDocument();
      expect(screen.getByText('Version is required')).toBeInTheDocument();
      expect(screen.getByText('Container image is required')).toBeInTheDocument();
    });

    expect(mockOnDeploy).not.toHaveBeenCalled();
  });

  test('validates xApp name format', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    const nameInput = screen.getByLabelText('xApp Name *');
    
    // Test invalid characters
    fireEvent.change(nameInput, { target: { value: 'Invalid_Name!' } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('xApp name must contain only lowercase letters, numbers, and hyphens')).toBeInTheDocument();
    });

    // Test too long name
    fireEvent.change(nameInput, { target: { value: 'a'.repeat(64) } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('xApp name must be 63 characters or less')).toBeInTheDocument();
    });
  });

  test('validates version format', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    const versionInput = screen.getByLabelText('Version *');
    
    fireEvent.change(versionInput, { target: { value: 'invalid-version' } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('Version must follow semantic versioning (e.g., 1.0.0)')).toBeInTheDocument();
    });
  });

  test('validates container image format', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    const imageInput = screen.getByLabelText('Container Image *');
    
    fireEvent.change(imageInput, { target: { value: 'invalid-image' } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('Image must be in format registry/repository:tag')).toBeInTheDocument();
    });
  });

  test('validates resource formats', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    const cpuInput = screen.getByLabelText('CPU *');
    const memoryInput = screen.getByLabelText('Memory *');
    
    fireEvent.change(cpuInput, { target: { value: 'invalid' } });
    fireEvent.change(memoryInput, { target: { value: 'invalid' } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('CPU must be in format like 100m or 1')).toBeInTheDocument();
      expect(screen.getByText('Memory must be in format like 128Mi or 1Gi')).toBeInTheDocument();
    });
  });

  test('validates JSON fields', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    const configTextarea = screen.getByLabelText('xApp Configuration (JSON)');
    
    fireEvent.change(configTextarea, { target: { value: 'invalid json' } });
    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(screen.getByText('Configuration must be valid JSON')).toBeInTheDocument();
    });
  });

  test('submits valid form data', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    // Fill in valid form data
    fireEvent.change(screen.getByLabelText('xApp Name *'), { 
      target: { value: 'test-xapp' } 
    });
    fireEvent.change(screen.getByLabelText('Version *'), { 
      target: { value: '1.0.0' } 
    });
    fireEvent.change(screen.getByLabelText('Container Image *'), { 
      target: { value: 'registry.example.com/test-xapp:1.0.0' } 
    });

    fireEvent.click(screen.getByText('Deploy xApp'));

    await waitFor(() => {
      expect(mockOnDeploy).toHaveBeenCalledWith({
        name: 'test-xapp',
        version: '1.0.0',
        image: 'registry.example.com/test-xapp:1.0.0',
        namespace: 'ricxapp',
        instances: 1,
        resources: {
          cpu: '100m',
          memory: '128Mi',
          storage: '1Gi'
        },
        configuration: {},
        environment: {},
        ports: [],
        volumes: []
      });
    });
  });

  test('handles cancel action', () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    fireEvent.click(screen.getByText('Cancel'));
    expect(mockOnCancel).toHaveBeenCalled();
  });

  test('shows loading state', () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={true}
        error={null}
      />
    );

    expect(screen.getByText('Deploying...')).toBeInTheDocument();
    expect(screen.getByText('Deploying...')).toBeDisabled();
    expect(screen.getByText('Cancel')).toBeDisabled();
  });

  test('displays error message', () => {
    const error = new Error('Deployment failed');
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={error}
      />
    );

    expect(screen.getByText('Deployment failed')).toBeInTheDocument();
  });

  test('clears validation errors when input changes', async () => {
    render(
      <XAppDeploymentForm 
        onDeploy={mockOnDeploy}
        onCancel={mockOnCancel}
        loading={false}
        error={null}
      />
    );

    // Trigger validation error
    fireEvent.click(screen.getByText('Deploy xApp'));
    
    await waitFor(() => {
      expect(screen.getByText('xApp name is required')).toBeInTheDocument();
    });

    // Fix the error
    fireEvent.change(screen.getByLabelText('xApp Name *'), { 
      target: { value: 'test-xapp' } 
    });

    // Error should be cleared
    expect(screen.queryByText('xApp name is required')).not.toBeInTheDocument();
  });
});