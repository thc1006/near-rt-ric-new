/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"net/http"
	"testing"
)

func TestClientManagerGetters(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	cm := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
	}

	// Test GetHTTPClient
	httpClient := cm.GetHTTPClient()
	if httpClient == nil {
		t.Error("Expected HTTP client to be returned")
	}

	// Test GetAppManagerEndpoint
	endpoint := cm.GetAppManagerEndpoint()
	if endpoint != config.AppmgrEndpoint {
		t.Errorf("Expected endpoint %s, got %s", config.AppmgrEndpoint, endpoint)
	}

	// Test GetE2ManagerConnection (should be nil for test)
	e2conn := cm.GetE2ManagerConnection()
	if e2conn != nil {
		t.Error("Expected E2 Manager connection to be nil in test")
	}

	// Test GetSubscriptionManagerConnection (should be nil for test)
	submgrConn := cm.GetSubscriptionManagerConnection()
	if submgrConn != nil {
		t.Error("Expected Subscription Manager connection to be nil in test")
	}

	// Test IsE2ManagerConnected (should be false for test)
	if cm.IsE2ManagerConnected() {
		t.Error("Expected E2 Manager to not be connected in test")
	}

	// Test IsSubscriptionManagerConnected (should be false for test)
	if cm.IsSubscriptionManagerConnected() {
		t.Error("Expected Subscription Manager to not be connected in test")
	}

	// Test Close (should not panic)
	cm.Close()
}

func TestClientManagerReconnect(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "invalid:endpoint",
		SubmgrEndpoint: "invalid:endpoint",
		AppmgrEndpoint: "invalid:endpoint",
	}

	cm := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
	}

	// Test Reconnect with invalid endpoints (should not panic)
	err := cm.Reconnect()
	if err != nil {
		t.Logf("Expected error for invalid endpoints: %v", err)
	}
}
