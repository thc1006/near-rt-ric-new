/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

/**
 * Integration test to verify React app can connect to Dashboard API Gateway
 * This test requires the dashboard API to be running on localhost:8080
 */

import dashboardAPI from './services/api';

describe('Dashboard API Integration', () => {
  // Skip these tests if API is not running
  const isAPIRunning = async () => {
    try {
      await Promise.race([
        dashboardAPI.getHealth(),
        new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 2000))
      ]);
      return true;
    } catch (error) {
      return false;
    }
  };

  beforeAll(async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Dashboard API not running on localhost:8080 - skipping integration tests');
    }
  }, 10000);

  it('should connect to health endpoint', async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Skipping test - API not running');
      return;
    }

    const health = await Promise.race([
      dashboardAPI.getHealth(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 5000))
    ]);
    expect(health).toHaveProperty('status');
    expect(health.status).toBe('healthy');
  }, 10000);

  it('should fetch components', async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Skipping test - API not running');
      return;
    }

    const response = await Promise.race([
      dashboardAPI.getComponents(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 5000))
    ]);
    expect(response).toHaveProperty('components');
    expect(response).toHaveProperty('count');
    expect(Array.isArray(response.components)).toBe(true);
  }, 10000);

  it('should fetch E2 nodes', async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Skipping test - API not running');
      return;
    }

    const response = await Promise.race([
      dashboardAPI.getE2Nodes(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 5000))
    ]);
    expect(response).toHaveProperty('e2nodes');
    expect(response).toHaveProperty('count');
    expect(Array.isArray(response.e2nodes)).toBe(true);
  }, 10000);

  it('should fetch subscriptions', async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Skipping test - API not running');
      return;
    }

    const response = await Promise.race([
      dashboardAPI.getSubscriptions(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 5000))
    ]);
    expect(response).toHaveProperty('subscriptions');
    expect(response).toHaveProperty('count');
    expect(Array.isArray(response.subscriptions)).toBe(true);
  }, 10000);

  it('should fetch xApps', async () => {
    const apiRunning = await isAPIRunning();
    if (!apiRunning) {
      console.warn('Skipping test - API not running');
      return;
    }

    const response = await Promise.race([
      dashboardAPI.getXApps(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), 5000))
    ]);
    expect(response).toHaveProperty('xapps');
    expect(response).toHaveProperty('count');
    expect(Array.isArray(response.xapps)).toBe(true);
  }, 10000);
});