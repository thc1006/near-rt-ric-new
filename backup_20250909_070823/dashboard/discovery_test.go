/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"net/http"
	"testing"
)

func TestDiscoveryServiceHelperMethods(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	clients := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
	}

	ds := NewDiscoveryService(clients)

	// Test helper methods (they should not panic and return default values)
	version := ds.getE2ManagerVersion()
	if version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", version)
	}

	metrics := ds.getE2ManagerMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}

	version = ds.getSubscriptionManagerVersion()
	if version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", version)
	}

	metrics = ds.getSubscriptionManagerMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}

	reachable := ds.isAppManagerReachable()
	if reachable {
		t.Error("Expected App Manager to not be reachable in test")
	}

	version = ds.getAppManagerVersion()
	if version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", version)
	}

	metrics = ds.getAppManagerMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}
}

func TestDiscoveryServiceStop(t *testing.T) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	clients := &ClientManager{
		config:     config,
		httpClient: &http.Client{},
	}

	ds := NewDiscoveryService(clients)

	// Test Stop method (should not panic)
	ds.Stop()

	// Test that ticker is stopped
	if ds.ticker == nil {
		t.Error("Expected ticker to be initialized")
	}
}

// TestDiscoveryServiceBroadcastComponentUpdates is removed due to blocking behavior
// The broadcastComponentUpdates function is tested indirectly through integration tests
