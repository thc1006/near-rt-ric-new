// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom';

// Set global test timeout to prevent hanging tests
jest.setTimeout(10000);

// Mock WebSocket for all tests
global.WebSocket = jest.fn(() => ({
  send: jest.fn(),
  close: jest.fn(),
  readyState: 1, // OPEN
  onopen: null,
  onmessage: null,
  onerror: null,
  onclose: null
}));

global.WebSocket.CONNECTING = 0;
global.WebSocket.OPEN = 1;
global.WebSocket.CLOSING = 2;
global.WebSocket.CLOSED = 3;

// Mock fetch for all tests
global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({}),
  })
);

// Clean up after each test
afterEach(() => {
  jest.clearAllTimers();
  jest.clearAllMocks();
});
