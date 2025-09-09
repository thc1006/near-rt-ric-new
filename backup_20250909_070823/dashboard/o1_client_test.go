/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestO1MediatorClient_GetHealth(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "healthy",
				"version": "1.0.0",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	client := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test GetHealth
	ctx := context.Background()
	health, err := client.GetHealth(ctx)
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if !health.IsHealthy {
		t.Error("Expected health to be healthy")
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", health.Version)
	}

	if health.StatusMessage != "Healthy" {
		t.Errorf("Expected status message 'Healthy', got %s", health.StatusMessage)
	}
}

func TestO1MediatorClient_IsConnected(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	client := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test IsConnected
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	// Test with invalid endpoint
	clientInvalid := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, "")
	if clientInvalid.IsConnected() {
		t.Error("Expected client with empty endpoint to not be connected")
	}
}

func TestO1MediatorClient_GetManagedObjects(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/managed-objects" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := O1ManagedObjectListResponse{
				ManagedObjects: []O1ManagedObject{
					{
						ID:          "obj1",
						Name:        "Test Object",
						Type:        "RIC",
						Description: "Test managed object",
						State:       "ACTIVE",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				},
				Total: 1,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	client := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test GetManagedObjects
	ctx := context.Background()
	objects, err := client.GetManagedObjects(ctx, nil)
	if err != nil {
		t.Fatalf("GetManagedObjects failed: %v", err)
	}

	if objects.Total != 1 {
		t.Errorf("Expected 1 managed object, got %d", objects.Total)
	}

	if len(objects.ManagedObjects) != 1 {
		t.Errorf("Expected 1 managed object in list, got %d", len(objects.ManagedObjects))
	}

	obj := objects.ManagedObjects[0]
	if obj.ID != "obj1" {
		t.Errorf("Expected object ID 'obj1', got %s", obj.ID)
	}

	if obj.Name != "Test Object" {
		t.Errorf("Expected object name 'Test Object', got %s", obj.Name)
	}
}

func TestO1MediatorClient_GetStats(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/stats" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			stats := O1Stats{
				ManagedObjectsByType:   map[string]uint32{"RIC": 1, "E2_NODE": 2},
				ConfigurationsByStatus: map[string]uint32{"ACTIVE": 3},
				AlarmsBySeverity:       map[string]uint32{"CRITICAL": 1, "WARNING": 2},
				KPIsByType:             map[string]uint32{"COUNTER": 5},
				TotalManagedObjects:    3,
				TotalConfigurations:    3,
				TotalActiveAlarms:      3,
				TotalKPIs:              5,
				LastUpdated:            time.Now(),
			}
			json.NewEncoder(w).Encode(stats)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	client := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test GetStats
	ctx := context.Background()
	stats, err := client.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalManagedObjects != 3 {
		t.Errorf("Expected 3 total managed objects, got %d", stats.TotalManagedObjects)
	}

	if stats.ManagedObjectsByType["RIC"] != 1 {
		t.Errorf("Expected 1 RIC object, got %d", stats.ManagedObjectsByType["RIC"])
	}

	if stats.AlarmsBySeverity["CRITICAL"] != 1 {
		t.Errorf("Expected 1 critical alarm, got %d", stats.AlarmsBySeverity["CRITICAL"])
	}
}

func TestO1MediatorClient_ValidateConfiguration(t *testing.T) {
	// Create client
	client := NewO1MediatorClient(&http.Client{Timeout: 5 * time.Second}, "http://localhost:8080")

	// Test ValidateConfiguration with valid JSON
	ctx := context.Background()
	validConfig := json.RawMessage(`{"key": "value"}`)
	result, err := client.ValidateConfiguration(ctx, validConfig)
	if err != nil {
		t.Fatalf("ValidateConfiguration failed: %v", err)
	}

	if !result.IsValid {
		t.Error("Expected configuration to be valid")
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no validation errors, got %d", len(result.Errors))
	}

	// Test ValidateConfiguration with invalid JSON
	invalidConfig := json.RawMessage(`{invalid json}`)
	result, err = client.ValidateConfiguration(ctx, invalidConfig)
	if err != nil {
		t.Fatalf("ValidateConfiguration failed: %v", err)
	}

	if result.IsValid {
		t.Error("Expected configuration to be invalid")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for invalid JSON")
	}
}

func TestExtractHostFromEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"http://localhost:8080", "localhost"},
		{"https://example.com:443/api", "example.com"},
		{"http://192.168.1.1:8080/path", "192.168.1.1"},
		{"localhost", "localhost"},
		{"example.com", "example.com"},
	}

	for _, test := range tests {
		result := extractHostFromEndpoint(test.endpoint)
		if result != test.expected {
			t.Errorf("extractHostFromEndpoint(%s) = %s, expected %s", test.endpoint, result, test.expected)
		}
	}
}

func TestO1MediatorClient_ManagementOperations(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/backups":
			if r.Method == "GET" {
				response := O1BackupListResponse{
					Backups: []O1BackupInfo{
						{
							BackupID:    "backup-1",
							Name:        "test-backup",
							Description: "Test backup",
							Size:        1024,
							CreatedAt:   time.Now(),
							Status:      "COMPLETED",
							ObjectTypes: []string{"RIC", "E2_NODE"},
						},
					},
					Total: 1,
				}
				json.NewEncoder(w).Encode(response)
			}
		case "/api/v1/backups/backup-1":
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/v1/alarms":
			if r.Method == "POST" {
				alarm := O1Alarm{
					ID:              "alarm-1",
					ManagedObjectID: "ric-001",
					AlarmType:       "CONNECTIVITY_LOSS",
					Severity:        "CRITICAL",
					AlarmState:      "ACTIVE",
					EventTime:       time.Now(),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(alarm)
			}
		case "/api/v1/alarms/alarm-1/clear":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/v1/alarms/correlate":
			if r.Method == "POST" {
				response := O1AlarmCorrelationResponse{
					CorrelationID:   "corr-1",
					AlarmIDs:        []O1AlarmID{"alarm-1", "alarm-2"},
					CorrelationType: "root_cause",
					CreatedAt:       time.Now(),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			}
		case "/api/v1/kpis":
			if r.Method == "POST" {
				kpi := O1KPI{
					ID:              "kpi-1",
					Name:            "cpu-utilization",
					Description:     "CPU utilization",
					MeasurementType: "GAUGE",
					Unit:            "percentage",
					Value:           75.5,
					Timestamp:       time.Now(),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(kpi)
			}
		case "/api/v1/kpis/kpi-1":
			if r.Method == "PUT" {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/v1/kpis/collect":
			if r.Method == "POST" {
				response := O1KPICollectionResponse{
					CollectedKPIs: []O1KPIDataPoint{
						{
							KPIID:     "kpi-1",
							Value:     80.0,
							Timestamp: time.Now(),
							Quality:   "good",
						},
					},
					StartTime:   time.Now().Add(-1 * time.Hour),
					EndTime:     time.Now(),
					TotalPoints: 1,
				}
				json.NewEncoder(w).Encode(response)
			}
		case "/api/v1/certificates":
			if r.Method == "GET" {
				response := O1CertificateListResponse{
					Certificates: []O1Certificate{
						{
							ID:          "cert-1",
							Name:        "test-cert",
							Type:        "TLS",
							Subject:     "CN=test.example.com",
							NotBefore:   time.Now(),
							NotAfter:    time.Now().Add(365 * 24 * time.Hour),
							Status:      "ACTIVE",
						},
					},
					Total: 1,
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == "POST" {
				cert := O1Certificate{
					ID:        "cert-2",
					Name:      "new-cert",
					Type:      "TLS",
					Subject:   "CN=new.example.com",
					NotBefore: time.Now(),
					NotAfter:  time.Now().Add(365 * 24 * time.Hour),
					Status:    "ACTIVE",
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(cert)
			}
		case "/api/v1/certificates/cert-1/revoke":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
			}
		case "/api/v1/resource-usage":
			if r.Method == "GET" {
				response := O1ResourceUsageResponse{
					ResourceUsage: []O1ResourceUsage{
						{
							ID:           "usage-1",
							ResourceType: "CPU",
							ResourceID:   "cpu-001",
							UsageMetrics: map[string]interface{}{"hours": 4.5},
							StartTime:    time.Now().Add(-1 * time.Hour),
							EndTime:      time.Now(),
							Duration:     "1h",
						},
					},
					Total: 1,
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == "POST" {
				usage := O1ResourceUsage{
					ID:           "usage-2",
					ResourceType: "MEMORY",
					ResourceID:   "mem-001",
					UsageMetrics: map[string]interface{}{"gb": 8.0},
					StartTime:    time.Now().Add(-1 * time.Hour),
					EndTime:      time.Now(),
					Duration:     "1h",
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(usage)
			}
		case "/api/v1/access-control/policies":
			if r.Method == "GET" {
				response := O1AccessControlPolicyListResponse{
					Policies: []O1AccessControlPolicy{
						{
							ID:          "policy-1",
							Name:        "test-policy",
							Description: "Test access control policy",
							PolicyType:  "RBAC",
							Rules:       []O1AccessControlRule{},
							Status:      "ACTIVE",
							CreatedAt:   time.Now(),
						},
					},
					Total: 1,
				}
				json.NewEncoder(w).Encode(response)
			} else if r.Method == "POST" {
				policy := O1AccessControlPolicy{
					ID:          "policy-2",
					Name:        "new-policy",
					Description: "New access control policy",
					PolicyType:  "RBAC",
					Rules:       []O1AccessControlRule{},
					Status:      "ACTIVE",
					CreatedAt:   time.Now(),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(policy)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewO1MediatorClient(&http.Client{}, server.URL)
	ctx := context.Background()

	// Test backup operations
	t.Run("GetBackups", func(t *testing.T) {
		backups, err := client.GetBackups(ctx, nil)
		if err != nil {
			t.Fatalf("GetBackups failed: %v", err)
		}
		if backups.Total != 1 {
			t.Errorf("Expected 1 backup, got %d", backups.Total)
		}
		if len(backups.Backups) != 1 {
			t.Errorf("Expected 1 backup in list, got %d", len(backups.Backups))
		}
		if backups.Backups[0].BackupID != "backup-1" {
			t.Errorf("Expected backup ID 'backup-1', got %s", backups.Backups[0].BackupID)
		}
	})

	t.Run("DeleteBackup", func(t *testing.T) {
		err := client.DeleteBackup(ctx, "backup-1")
		if err != nil {
			t.Fatalf("DeleteBackup failed: %v", err)
		}
	})

	// Test alarm operations
	t.Run("GenerateAlarm", func(t *testing.T) {
		request := &O1AlarmRequest{
			ManagedObjectID: "ric-001",
			AlarmType:       "CONNECTIVITY_LOSS",
			Severity:        "CRITICAL",
			ProbableCause:   "Network failure",
			SpecificProblem: "Connection timeout",
			EventTime:       time.Now(),
		}
		alarm, err := client.GenerateAlarm(ctx, request)
		if err != nil {
			t.Fatalf("GenerateAlarm failed: %v", err)
		}
		if string(alarm.ID) != "alarm-1" {
			t.Errorf("Expected alarm ID 'alarm-1', got %s", alarm.ID)
		}
		if alarm.AlarmType != "CONNECTIVITY_LOSS" {
			t.Errorf("Expected alarm type 'CONNECTIVITY_LOSS', got %s", alarm.AlarmType)
		}
	})

	t.Run("ClearAlarm", func(t *testing.T) {
		request := &O1AlarmClearRequest{
			AlarmID:   "alarm-1",
			User:      "admin",
			Reason:    "Issue resolved",
			ClearTime: time.Now(),
		}
		err := client.ClearAlarm(ctx, "alarm-1", request)
		if err != nil {
			t.Fatalf("ClearAlarm failed: %v", err)
		}
	})

	t.Run("CorrelateAlarms", func(t *testing.T) {
		request := &O1AlarmCorrelationRequest{
			AlarmIDs:        []O1AlarmID{"alarm-1", "alarm-2"},
			CorrelationType: "root_cause",
			RootCause:       "Network connectivity issue",
		}
		response, err := client.CorrelateAlarms(ctx, request)
		if err != nil {
			t.Fatalf("CorrelateAlarms failed: %v", err)
		}
		if response.CorrelationID != "corr-1" {
			t.Errorf("Expected correlation ID 'corr-1', got %s", response.CorrelationID)
		}
		if len(response.AlarmIDs) != 2 {
			t.Errorf("Expected 2 alarm IDs, got %d", len(response.AlarmIDs))
		}
	})

	// Test KPI operations
	t.Run("CreateKPI", func(t *testing.T) {
		request := &O1KPIRequest{
			Name:            "cpu-utilization",
			Description:     "CPU utilization percentage",
			MeasurementType: "GAUGE",
			Unit:            "percentage",
			ManagedObjectID: "ric-001",
		}
		kpi, err := client.CreateKPI(ctx, request)
		if err != nil {
			t.Fatalf("CreateKPI failed: %v", err)
		}
		if string(kpi.ID) != "kpi-1" {
			t.Errorf("Expected KPI ID 'kpi-1', got %s", kpi.ID)
		}
		if kpi.Name != "cpu-utilization" {
			t.Errorf("Expected KPI name 'cpu-utilization', got %s", kpi.Name)
		}
	})

	t.Run("UpdateKPI", func(t *testing.T) {
		update := &O1KPIUpdate{
			KPIID:     "kpi-1",
			Value:     85.0,
			Timestamp: time.Now(),
		}
		err := client.UpdateKPI(ctx, "kpi-1", update)
		if err != nil {
			t.Fatalf("UpdateKPI failed: %v", err)
		}
	})

	t.Run("CollectKPIData", func(t *testing.T) {
		request := &O1KPICollectionRequest{
			KPIIDs:    []O1KPIID{"kpi-1"},
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Interval:  "5m",
		}
		response, err := client.CollectKPIData(ctx, request)
		if err != nil {
			t.Fatalf("CollectKPIData failed: %v", err)
		}
		if response.TotalPoints != 1 {
			t.Errorf("Expected 1 data point, got %d", response.TotalPoints)
		}
		if len(response.CollectedKPIs) != 1 {
			t.Errorf("Expected 1 KPI data point, got %d", len(response.CollectedKPIs))
		}
	})

	// Test certificate operations
	t.Run("GetCertificates", func(t *testing.T) {
		certs, err := client.GetCertificates(ctx, nil)
		if err != nil {
			t.Fatalf("GetCertificates failed: %v", err)
		}
		if certs.Total != 1 {
			t.Errorf("Expected 1 certificate, got %d", certs.Total)
		}
		if len(certs.Certificates) != 1 {
			t.Errorf("Expected 1 certificate in list, got %d", len(certs.Certificates))
		}
	})

	t.Run("CreateCertificate", func(t *testing.T) {
		request := &O1CertificateRequest{
			Name:         "new-cert",
			Type:         "TLS",
			Subject:      "CN=new.example.com",
			KeySize:      2048,
			ValidityDays: 365,
		}
		cert, err := client.CreateCertificate(ctx, request)
		if err != nil {
			t.Fatalf("CreateCertificate failed: %v", err)
		}
		if cert.ID != "cert-2" {
			t.Errorf("Expected certificate ID 'cert-2', got %s", cert.ID)
		}
		if cert.Name != "new-cert" {
			t.Errorf("Expected certificate name 'new-cert', got %s", cert.Name)
		}
	})

	t.Run("RevokeCertificate", func(t *testing.T) {
		err := client.RevokeCertificate(ctx, "cert-1", "compromised")
		if err != nil {
			t.Fatalf("RevokeCertificate failed: %v", err)
		}
	})

	// Test resource usage operations
	t.Run("GetResourceUsage", func(t *testing.T) {
		usage, err := client.GetResourceUsage(ctx, nil)
		if err != nil {
			t.Fatalf("GetResourceUsage failed: %v", err)
		}
		if usage.Total != 1 {
			t.Errorf("Expected 1 resource usage record, got %d", usage.Total)
		}
		if len(usage.ResourceUsage) != 1 {
			t.Errorf("Expected 1 resource usage record in list, got %d", len(usage.ResourceUsage))
		}
	})

	t.Run("CreateResourceUsageRecord", func(t *testing.T) {
		request := &O1ResourceUsageRequest{
			ResourceType:    "MEMORY",
			ResourceID:      "mem-001",
			UsageMetrics:    map[string]interface{}{"gb": 8.0},
			StartTime:       time.Now().Add(-1 * time.Hour),
			EndTime:         time.Now(),
			ManagedObjectID: "ric-001",
		}
		usage, err := client.CreateResourceUsageRecord(ctx, request)
		if err != nil {
			t.Fatalf("CreateResourceUsageRecord failed: %v", err)
		}
		if usage.ID != "usage-2" {
			t.Errorf("Expected usage ID 'usage-2', got %s", usage.ID)
		}
		if usage.ResourceType != "MEMORY" {
			t.Errorf("Expected resource type 'MEMORY', got %s", usage.ResourceType)
		}
	})

	// Test access control operations
	t.Run("GetAccessControlPolicies", func(t *testing.T) {
		policies, err := client.GetAccessControlPolicies(ctx, nil)
		if err != nil {
			t.Fatalf("GetAccessControlPolicies failed: %v", err)
		}
		if policies.Total != 1 {
			t.Errorf("Expected 1 access control policy, got %d", policies.Total)
		}
		if len(policies.Policies) != 1 {
			t.Errorf("Expected 1 access control policy in list, got %d", len(policies.Policies))
		}
	})

	t.Run("CreateAccessControlPolicy", func(t *testing.T) {
		request := &O1AccessControlPolicyRequest{
			Name:        "new-policy",
			Description: "New access control policy",
			PolicyType:  "RBAC",
			Rules:       []O1AccessControlRule{},
		}
		policy, err := client.CreateAccessControlPolicy(ctx, request)
		if err != nil {
			t.Fatalf("CreateAccessControlPolicy failed: %v", err)
		}
		if policy.ID != "policy-2" {
			t.Errorf("Expected policy ID 'policy-2', got %s", policy.ID)
		}
		if policy.Name != "new-policy" {
			t.Errorf("Expected policy name 'new-policy', got %s", policy.Name)
		}
	})
}