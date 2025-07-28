/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

/**
 * Example client for the Dashboard API Gateway
 * Demonstrates how to interact with the O-RAN SC components via REST API
 */

const API_BASE_URL = 'http://localhost:8080/api/v1';
const WS_URL = 'ws://localhost:8080/ws';

class DashboardAPIClient {
    constructor(baseUrl = API_BASE_URL) {
        this.baseUrl = baseUrl;
        this.ws = null;
    }

    // Generic HTTP request method
    async request(endpoint, options = {}) {
        const url = `${this.baseUrl}${endpoint}`;
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return response.json();
    }

    // Component Discovery
    async getComponents() {
        return this.request('/components');
    }

    async getComponent(id) {
        return this.request(`/components/${id}`);
    }

    // E2 Manager Integration
    async getE2Nodes() {
        return this.request('/e2nodes');
    }

    async getE2Node(id) {
        return this.request(`/e2nodes/${id}`);
    }

    // Subscription Manager Integration
    async getSubscriptions() {
        return this.request('/subscriptions');
    }

    async createSubscription(subscriptionData) {
        return this.request('/subscriptions', {
            method: 'POST',
            body: JSON.stringify(subscriptionData)
        });
    }

    async getSubscription(id) {
        return this.request(`/subscriptions/${id}`);
    }

    async deleteSubscription(id) {
        return this.request(`/subscriptions/${id}`, {
            method: 'DELETE'
        });
    }

    // App Manager Integration
    async getXApps() {
        return this.request('/xapps');
    }

    async deployXApp(xappData) {
        return this.request('/xapps', {
            method: 'POST',
            body: JSON.stringify(xappData)
        });
    }

    async getXApp(name) {
        return this.request(`/xapps/${name}`);
    }

    async undeployXApp(name) {
        return this.request(`/xapps/${name}`, {
            method: 'DELETE'
        });
    }

    // WebSocket for real-time updates
    connectWebSocket(onMessage, onError, onClose) {
        this.ws = new WebSocket(WS_URL);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            // Send subscription message for component updates
            this.ws.send(JSON.stringify({
                type: 'subscribe',
                data: ['component_update', 'subscription_created', 'xapp_deployed']
            }));
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                if (onMessage) onMessage(message);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            if (onError) onError(error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            if (onClose) onClose();
        };
    }

    disconnectWebSocket() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    // Health check
    async getHealth() {
        const response = await fetch(`${this.baseUrl.replace('/api/v1', '')}/health`);
        return response.json();
    }
}

// Example usage
async function example() {
    const client = new DashboardAPIClient();

    try {
        // Check API health
        const health = await client.getHealth();
        console.log('API Health:', health);

        // Get all components
        const components = await client.getComponents();
        console.log('Discovered Components:', components);

        // Get E2 nodes
        const e2nodes = await client.getE2Nodes();
        console.log('E2 Nodes:', e2nodes);

        // Get subscriptions
        const subscriptions = await client.getSubscriptions();
        console.log('Active Subscriptions:', subscriptions);

        // Get deployed xApps
        const xapps = await client.getXApps();
        console.log('Deployed xApps:', xapps);

        // Connect to WebSocket for real-time updates
        client.connectWebSocket(
            (message) => {
                console.log('Real-time update:', message);
                
                switch (message.type) {
                    case 'component_update':
                        console.log('Component status changed:', message.data);
                        break;
                    case 'subscription_created':
                        console.log('New subscription created:', message.data);
                        break;
                    case 'xapp_deployed':
                        console.log('xApp deployed:', message.data);
                        break;
                    default:
                        console.log('Unknown message type:', message.type);
                }
            },
            (error) => console.error('WebSocket error:', error),
            () => console.log('WebSocket connection closed')
        );

        // Example: Create a subscription
        const newSubscription = await client.createSubscription({
            e2nodeId: 'gnb-001',
            ranFunctionId: 1,
            eventTrigger: {
                type: 'periodic',
                interval: 1000
            }
        });
        console.log('Created subscription:', newSubscription);

        // Example: Deploy an xApp
        const deployedXApp = await client.deployXApp({
            name: 'hello-world',
            version: '1.0.0',
            helmChart: 'oran/xapp-hello-world'
        });
        console.log('Deployed xApp:', deployedXApp);

    } catch (error) {
        console.error('API Error:', error);
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DashboardAPIClient;
}

// Run example if this file is executed directly
if (typeof window === 'undefined' && require.main === module) {
    example();
}