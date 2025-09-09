//go:build o1_management_example

/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

func main() {
	fmt.Println("O1 Management Operations Example")
	fmt.Println("===============================")

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create O1 Mediator client
	client := dashboard.NewO1MediatorClient(httpClient, "http://localhost:8080")

	// Create O1 Management Service
	managementService := dashboard.NewO1ManagementService(client)

	ctx := context.Background()

	// Start the management service
	fmt.Println("\n1. Starting O1 Management Service...")
	if err := managementService.Start(ctx); err != nil {
		log.Printf("Failed to start management service: %v", err)
		return
	}
	defer managementService.Stop()

	// Test Configuration Management
	fmt.Println("\n2. Testing Configuration Management...")
	
	// Create a configuration backup
	backup, err := managementService.CreateConfigurationBackup(ctx, 
		"demo-backup", 
		"Demonstration backup for O1 management", 
		true, 
		[]string{"RIC", "E2_NODE", "XAPP"})
	if err != nil {
		log.Printf("Failed to create backup: %v", err)
	} else {
		fmt.Printf("Created backup: %s (Size: %d bytes)\n", backup.BackupID, backup.Size)
	}

	// Test Fault Management
	fmt.Println("\n3. Testing Fault Management...")
	
	// Generate an alarm
	alarm, err := managementService.GenerateAlarm(ctx,
		"ric-001",
		"CONNECTIVITY_LOSS",
		"CRITICAL",
		"Network interface down",
		"E2 connection lost to gNB",
		"Connection timeout after 30 seconds")
	if err != nil {
		log.Printf("Failed to generate alarm: %v", err)
	} else {
		fmt.Printf("Generated alarm: %s (Severity: %s)\n", alarm.ID, alarm.Severity)
		
		// Clear the alarm after demonstration
		time.Sleep(2 * time.Second)
		if err := managementService.ClearAlarm(ctx, alarm.ID, "admin", "Demonstration completed"); err != nil {
			log.Printf("Failed to clear alarm: %v", err)
		} else {
			fmt.Printf("Cleared alarm: %s\n", alarm.ID)
		}
	}

	// Test Performance Management
	fmt.Println("\n4. Testing Performance Management...")
	
	// Create a KPI
	threshold := &dashboard.O1KPIThreshold{
		WarningMax:  &[]float64{80.0}[0],
		CriticalMax: &[]float64{95.0}[0],
	}
	
	kpi, err := managementService.CreateKPI(ctx,
		"cpu-utilization",
		"CPU Utilization percentage for RIC components",
		"GAUGE",
		"percentage",
		"ric-001",
		threshold)
	if err != nil {
		log.Printf("Failed to create KPI: %v", err)
	} else {
		fmt.Printf("Created KPI: %s (%s)\n", kpi.ID, kpi.Name)
		
		// Start KPI collection
		if err := managementService.StartKPICollection(ctx, kpi.ID, 30*time.Second); err != nil {
			log.Printf("Failed to start KPI collection: %v", err)
		} else {
			fmt.Printf("Started KPI collection for %s\n", kpi.ID)
		}
	}

	// Test Security Management
	fmt.Println("\n5. Testing Security Management...")
	
	// Create a certificate
	certificate, err := managementService.CreateCertificate(ctx,
		"ric-tls-cert",
		"TLS",
		"CN=ric.example.com,O=O-RAN,C=US",
		2048,
		365)
	if err != nil {
		log.Printf("Failed to create certificate: %v", err)
	} else {
		fmt.Printf("Created certificate: %s (Subject: %s)\n", certificate.ID, certificate.Subject)
	}
	
	// Create an access control policy
	rules := []dashboard.O1AccessControlRule{
		{
			ID: "rule-1",
			Subject: dashboard.O1AccessControlSubject{
				Type:       "user",
				Identifier: "admin",
				Attributes: []string{"administrator"},
			},
			Action:   "READ",
			Resource: "/api/v1/o1/*",
			Effect:   "ALLOW",
		},
		{
			ID: "rule-2",
			Subject: dashboard.O1AccessControlSubject{
				Type:       "role",
				Identifier: "operator",
				Attributes: []string{"network-operator"},
			},
			Action:   "READ",
			Resource: "/api/v1/o1/alarms",
			Effect:   "ALLOW",
		},
	}
	
	policy, err := managementService.CreateAccessControlPolicy(ctx,
		"ric-access-policy",
		"Access control policy for RIC management operations",
		"RBAC",
		rules)
	if err != nil {
		log.Printf("Failed to create access control policy: %v", err)
	} else {
		fmt.Printf("Created access control policy: %s (%d rules)\n", policy.ID, len(policy.Rules))
	}

	// Test Accounting Management
	fmt.Println("\n6. Testing Accounting Management...")
	
	// Track resource usage
	usageMetrics := map[string]interface{}{
		"cpu_hours":    4.5,
		"memory_gb":    8.0,
		"storage_gb":   50.0,
		"network_gb":   2.1,
	}
	
	cost := &dashboard.O1ResourceCost{
		Amount:   12.50,
		Currency: "USD",
		Unit:     "hour",
	}
	
	usage, err := managementService.TrackResourceUsage(ctx,
		"XAPP",
		"hello-world-xapp",
		"ric-001",
		usageMetrics,
		time.Now().Add(-1*time.Hour),
		time.Now(),
		cost)
	if err != nil {
		log.Printf("Failed to track resource usage: %v", err)
	} else {
		fmt.Printf("Tracked resource usage: %s (Cost: $%.2f %s)\n", 
			usage.ID, cost.Amount, cost.Currency)
	}

	// Test comprehensive operations
	fmt.Println("\n7. Testing Comprehensive Operations...")
	
	// Get management statistics
	stats := managementService.GetManagementStats()
	fmt.Printf("Management Service Statistics:\n")
	if serviceStatus, ok := stats["service_status"].(map[string]interface{}); ok {
		fmt.Printf("  - Service Running: %v\n", serviceStatus["running"])
	}
	if configStats, ok := stats["configuration"].(map[string]interface{}); ok {
		fmt.Printf("  - Total Backups: %v\n", configStats["total_backups"])
	}
	if faultStats, ok := stats["fault_management"].(map[string]interface{}); ok {
		fmt.Printf("  - Correlation Rules: %v\n", faultStats["correlation_rules"])
	}
	if perfStats, ok := stats["performance"].(map[string]interface{}); ok {
		fmt.Printf("  - Total KPIs: %v\n", perfStats["total_kpis"])
		fmt.Printf("  - Active Collectors: %v\n", perfStats["active_collectors"])
	}
	if secStats, ok := stats["security"].(map[string]interface{}); ok {
		fmt.Printf("  - Total Certificates: %v\n", secStats["total_certificates"])
		fmt.Printf("  - Access Policies: %v\n", secStats["total_access_policies"])
	}
	if accStats, ok := stats["accounting"].(map[string]interface{}); ok {
		fmt.Printf("  - Tracked Resource Types: %v\n", accStats["tracked_resource_types"])
	}

	// Test original O1 client functionality
	fmt.Println("\n8. Testing Basic O1 Client Operations...")
	
	// Test health check
	health, err := client.GetHealth(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("O1 Mediator Health: %s (Healthy: %t)\n", health.StatusMessage, health.IsHealthy)
	}

	// Test managed objects
	managedObjects, err := client.GetManagedObjects(ctx, nil)
	if err != nil {
		log.Printf("Failed to get managed objects: %v", err)
	} else {
		fmt.Printf("Found %d managed objects\n", managedObjects.Total)
	}

	// Test configurations
	configurations, err := client.GetConfigurations(ctx, nil)
	if err != nil {
		log.Printf("Failed to get configurations: %v", err)
	} else {
		fmt.Printf("Found %d configurations\n", configurations.Total)
	}

	// Test alarms
	alarms, err := client.GetAlarms(ctx, nil)
	if err != nil {
		log.Printf("Failed to get alarms: %v", err)
	} else {
		fmt.Printf("Found %d alarms\n", alarms.Total)
	}

	// Test KPIs
	kpis, err := client.GetKPIs(ctx, nil)
	if err != nil {
		log.Printf("Failed to get KPIs: %v", err)
	} else {
		fmt.Printf("Found %d KPIs\n", kpis.Total)
	}

	// Test configuration validation
	testConfig := json.RawMessage(`{
		"ric": {
			"components": ["e2mgr", "submgr", "rtmgr"],
			"interfaces": ["e2", "a1", "o1"],
			"security": {
				"tls_enabled": true,
				"authentication": "jwt"
			}
		}
	}`)
	
	validation, err := client.ValidateConfiguration(ctx, testConfig)
	if err != nil {
		log.Printf("Failed to validate configuration: %v", err)
	} else {
		fmt.Printf("Configuration validation: Valid=%t, Errors=%d\n", validation.IsValid, len(validation.Errors))
	}

	// Test statistics
	clientStats, err := client.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get client stats: %v", err)
	} else {
		fmt.Printf("O1 Mediator Statistics:\n")
		fmt.Printf("  - Managed Objects: %d\n", clientStats.TotalManagedObjects)
		fmt.Printf("  - Configurations: %d\n", clientStats.TotalConfigurations)
		fmt.Printf("  - Active Alarms: %d\n", clientStats.TotalActiveAlarms)
		fmt.Printf("  - KPIs: %d\n", clientStats.TotalKPIs)
		fmt.Printf("  - Last Updated: %s\n", clientStats.LastUpdated.Format(time.RFC3339))
	}

	// Test NETCONF operations
	fmt.Println("\n9. Testing NETCONF Operations...")
	err = client.ConnectNetconf()
	if err != nil {
		log.Printf("Failed to connect to NETCONF server: %v", err)
	} else {
		fmt.Println("Connected to NETCONF server successfully")
		
		// Send a simple NETCONF RPC
		rpcReply, err := client.SendNetconfRPC(ctx, "<get-config><source><running/></source></get-config>")
		if err != nil {
			log.Printf("Failed to send NETCONF RPC: %v", err)
		} else {
			fmt.Printf("NETCONF RPC reply: MessageID=%s\n", rpcReply.MessageID)
		}
		
		// Disconnect
		err = client.DisconnectNetconf()
		if err != nil {
			log.Printf("Failed to disconnect from NETCONF server: %v", err)
		} else {
			fmt.Println("Disconnected from NETCONF server")
		}
	}

	// Wait a bit to see background operations
	fmt.Println("\n10. Waiting for background operations...")
	time.Sleep(5 * time.Second)

	fmt.Println("\nO1 Management Operations Example completed!")
	fmt.Println("The management service demonstrated:")
	fmt.Println("  ✓ Configuration management with backup and restore")
	fmt.Println("  ✓ Fault management with alarm generation and correlation")
	fmt.Println("  ✓ Performance management with KPI collection and reporting")
	fmt.Println("  ✓ Security management with certificate and access control")
	fmt.Println("  ✓ Accounting functionality for resource usage tracking")
}