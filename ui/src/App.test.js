import { render, screen } from '@testing-library/react';
import App from './App';

// Mock the API service to avoid network calls in tests
jest.mock('./services/api', () => ({
  __esModule: true,
  default: {
    getComponents: jest.fn().mockResolvedValue({ components: [], count: 0 }),
    getE2Nodes: jest.fn().mockResolvedValue({ e2nodes: [], count: 0 }),
    getSubscriptions: jest.fn().mockResolvedValue({ subscriptions: [], count: 0 }),
    getXApps: jest.fn().mockResolvedValue({ xapps: [], count: 0 }),
    getHealth: jest.fn().mockResolvedValue({ status: 'healthy' }),
    connectWebSocket: jest.fn(),
    disconnectWebSocket: jest.fn(),
    isWebSocketConnected: jest.fn().mockReturnValue(true),
    getWebSocketState: jest.fn().mockReturnValue('OPEN'),
    sendWebSocketMessage: jest.fn().mockReturnValue(true),
  },
  APIError: class APIError extends Error {
    constructor(message, status) {
      super(message);
      this.status = status;
    }
  }
}));

test('renders O-RAN Interactive Operations Console', () => {
  render(<App />);
  const headerElement = screen.getByText(/O-RAN Interactive Operations Console/i);
  expect(headerElement).toBeInTheDocument();
});

test('renders dashboard panels', () => {
  render(<App />);
  
  expect(screen.getByText('Network Functions')).toBeInTheDocument();
  expect(screen.getByText('Real-Time KPIs')).toBeInTheDocument();
  expect(screen.getByText('Alarms')).toBeInTheDocument();
  expect(screen.getByText('Real-Time Logs')).toBeInTheDocument();
});

test('renders connection status', () => {
  render(<App />);
  
  // Should show connection status in header
  expect(screen.getByText(/Real-time Updates Active|Connecting|Disconnected/)).toBeInTheDocument();
});
