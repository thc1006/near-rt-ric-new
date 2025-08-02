/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

func main() {
	fmt.Println("=== xApp Framework Integration Example ===")
	
	// Create client manager (mock for this example)
	clientManager := dashboard.NewClientManager()
	
	// Create and start the xApp framework
	framework := dashboard.NewXAppFramework(clientManager)
	
	if err := framework.Start(); err != nil {
		log.Fatalf("Failed to start xApp framework: %v", err)
	}
	defer framework.Stop()
	
	fmt.Println("✓ xApp Framework started successfully")
	
	// Example 1: Register an xApp
	fmt.Println("\n1. Registering xApp...")
	
	descriptor := &dashboard.XAppDescriptor{
		Name:        "kpi-monitor",
		Version:     "1.0.0",
		Description: "KPI monitoring xApp for O-RAN",
		ServiceModels: []string{
			"E2SM-KPM",
			"E2SM-RC",
		},
		Capabilities: []string{
			"monitoring",
			"alerting",
			"reporting",
		},
		ResourceLimits: dashboard.XAppResourceLimits{
			CPU:              "500m",
			Memory:           "512Mi",
			Storage:          "1Gi",
			MaxSubscriptions: 100,
			MaxConnections:   50,
		},
		Endpoints: dashboard.XAppEndpoints{
			HTTP:     "http://0.0.0.0:8080",
			RMRData:  4560,
			RMRRoute: 4561,
			Metrics:  "http://0.0.0.0:9090/metrics",
		},
		HealthCheck: dashboard.XAppHealthCheck{
			Enabled:             true,
			Path:                "/health",
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			FailureThreshold:    3,
		},
		Configuration: map[string]interface{}{
			"threshold":       75.0,
			"reportInterval":  30,
			"alertEnabled":    true,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    dashboard.XAppStatusPending,
	}
	
	if err := framework.RegisterXApp(descriptor); err != nil {
		log.Fatalf("Failed to register xApp: %v", err)
	}
	fmt.Printf("✓ xApp %s v%s registered successfully\n", descriptor.Name, descriptor.Version)
	
	// Example 2: Set up configuration schema and management
	fmt.Println("\n2. Setting up configuration management...")
	
	configManager := framework.GetConfigManager()
	
	schema := &dashboard.ConfigSchema{
		Type: "object",
		Properties: map[string]*dashboard.PropertySpec{
			"threshold": {
				Type:        "number",
				Description: "KPI threshold for alerts",
				Minimum:     &[]float64{0.0}[0],
				Maximum:     &[]float64{100.0}[0],
			},
			"reportInterval": {
				Type:        "integer",
				Description: "Report interval in seconds",
				Minimum:     &[]float64{1.0}[0],
				Maximum:     &[]float64{3600.0}[0],
			},
			"alertEnabled": {
				Type:        "boolean",
				Description: "Enable/disable alerting",
			},
		},
		Required: []string{"threshold", "reportInterval"},
	}
	
	if err := configManager.SetConfigSchema("kpi-monitor", schema); err != nil {
		log.Fatalf("Failed to set config schema: %v", err)
	}
	fmt.Println("✓ Configuration schema set")
	
	// Set up configuration watcher
	configManager.WatchConfig("kpi-monitor", func(config *dashboard.XAppConfig) error {
		fmt.Printf("🔔 Configuration updated for %s: threshold=%.1f, interval=%v\n",
			config.Name, config.Config["threshold"], config.Config["reportInterval"])
		return nil
	})
	
	// Example 3: Deploy xApp instance
	fmt.Println("\n3. Deploying xApp instance...")
	
	deployConfig := map[string]interface{}{
		"threshold":      80.0,
		"reportInterval": 60,
		"alertEnabled":   true,
		"logLevel":       "INFO",
	}
	
	instance, err := framework.DeployXApp("kpi-monitor", "1.0.0", deployConfig)
	if err != nil {
		log.Fatalf("Failed to deploy xApp: %v", err)
	}
	fmt.Printf("✓ xApp instance deployed with ID: %s\n", instance.ID)
	
	// Wait for deployment to complete
	time.Sleep(3 * time.Second)
	
	// Example 4: Check instance status
	fmt.Println("\n4. Checking instance status...")
	
	updatedInstance, err := framework.GetXAppInstance(instance.ID)
	if err != nil {
		log.Fatalf("Failed to get instance: %v", err)
	}
	
	fmt.Printf("Instance Status: %s\n", updatedInstance.Status)
	fmt.Printf("Started At: %s\n", updatedInstance.StartedAt.Format(time.RFC3339))
	fmt.Printf("Pod Name: %s\n", updatedInstance.PodName)
	fmt.Printf("Service Name: %s\n", updatedInstance.ServiceName)
	
	// Example 5: Resource management
	fmt.Println("\n5. Managing resources...")
	
	resourceManager := framework.GetResourceManager()
	
	// Allocate resources for the instance
	resourceReq := &dashboard.ResourceRequest{
		XAppName:    "kpi-monitor",
		InstanceID:  instance.ID,
		CPU:         "500m",
		Memory:      "512Mi",
		Storage:     "1Gi",
		NetworkPorts: []int{8080, 9090},
	}
	
	allocation, err := resourceManager.AllocateResources(resourceReq)
	if err != nil {
		log.Printf("Warning: Failed to allocate resources: %v", err)
	} else {
		fmt.Printf("✓ Resources allocated: %s\n", allocation.ID)
	}
	
	// Example 6: Communication API usage
	fmt.Println("\n6. Setting up communication...")
	
	commAPI := framework.GetCommunicationAPI()
	
	// Register message handler
	handler := func(msg *dashboard.RMRMessage) error {
		fmt.Printf("📨 Received message: Type=%d, Payload=%s\n", msg.MessageType, string(msg.Payload))
		return nil
	}
	
	if err := commAPI.RegisterMessageHandler(4560, handler); err != nil {
		log.Printf("Warning: Failed to register message handler: %v", err)
	} else {
		fmt.Println("✓ Message handler registered")
	}
	
	// Example 7: Simulate subscription management
	fmt.Println("\n7. Managing subscriptions...")
	
	lifecycleManager := framework.GetLifecycleManager()
	
	// Simulate adding a subscription
	subscription := &dashboard.Subscription{
		ID:           "sub-001",
		XAppName:     "kpi-monitor",
		E2NodeID:     "gnb-001",
		ServiceModel: "E2SM-KPM",
		ActionType:   "report",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}
	
	if err := lifecycleManager.AddSubscription(instance.ID, subscription); err != nil {
		log.Printf("Warning: Failed to add subscription: %v", err)
	} else {
		fmt.Printf("✓ Subscription %s added to instance\n", subscription.ID)
	}
	
	// Example 8: Update configuration dynamically
	fmt.Println("\n8. Updating configuration dynamically...")
	
	configUpdates := map[string]interface{}{
		"threshold":      85.0,
		"reportInterval": 45,
		"newParameter":   "dynamic",
	}
	
	if err := configManager.UpdateConfig("kpi-monitor", configUpdates); err != nil {
		log.Printf("Warning: Failed to update config: %v", err)
	} else {
		fmt.Println("✓ Configuration updated")
	}
	
	// Wait for watcher notification
	time.Sleep(500 * time.Millisecond)
	
	// Example 9: Monitor framework status
	fmt.Println("\n9. Framework status...")
	
	status := framework.GetFrameworkStatus()
	fmt.Printf("Framework Status: %v\n", status["status"])
	fmt.Printf("Registered xApps: %v\n", status["registeredApps"])
	fmt.Printf("Running Instances: %v\n", status["runningInstances"])
	fmt.Printf("Total Instances: %v\n", status["totalInstances"])
	
	// Example 10: List all instances
	fmt.Println("\n10. Listing all instances...")
	
	allInstances, err := framework.ListXAppInstances()
	if err != nil {
		log.Printf("Warning: Failed to list instances: %v", err)
	} else {
		fmt.Printf("Found %d instances:\n", len(allInstances))
		for _, inst := range allInstances {
			fmt.Printf("  - %s (%s) - Status: %s\n", 
				inst.Descriptor.Name, inst.ID[:8], inst.Status)
		}
	}
	
	// Example 11: Simulate metrics update
	fmt.Println("\n11. Updating metrics...")
	
	metrics := dashboard.XAppMetrics{
		CPUUsage:            45.2,
		MemoryUsage:         256 * 1024 * 1024, // 256MB
		NetworkBytesIn:      1024 * 1024,       // 1MB
		NetworkBytesOut:     512 * 1024,        // 512KB
		ActiveSubscriptions: 1,
		MessagesProcessed:   1500,
		ErrorCount:          2,
		LastUpdated:         time.Now().UTC(),
	}
	
	if err := lifecycleManager.UpdateMetrics(instance.ID, metrics); err != nil {
		log.Printf("Warning: Failed to update metrics: %v", err)
	} else {
		fmt.Println("✓ Metrics updated")
	}
	
	// Example 12: Cleanup - undeploy instance
	fmt.Println("\n12. Cleaning up...")
	
	if err := framework.UndeployXApp(instance.ID); err != nil {
		log.Printf("Warning: Failed to undeploy xApp: %v", err)
	} else {
		fmt.Printf("✓ xApp instance %s undeployed\n", instance.ID[:8])
	}
	
	// Final status check
	finalStatus := framework.GetFrameworkStatus()
	fmt.Printf("\nFinal Status - Running Instances: %v\n", finalStatus["runningInstances"])
	
	fmt.Println("\n=== Integration example completed successfully ===")
	fmt.Println("\nThe xApp Framework provides:")
	fmt.Println("• Complete xApp lifecycle management")
	fmt.Println("• Dynamic configuration with schema validation")
	fmt.Println("• Resource allocation and management")
	fmt.Println("• RMR-based communication APIs")
	fmt.Println("• Subscription and service model support")
	fmt.Println("• Real-time monitoring and metrics")
	fmt.Println("• Hot deployment and updates")
}