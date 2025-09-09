//go:build xapp_config_example

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
	// Create xApp configuration manager
	configManager := dashboard.NewXAppConfigManager()
	
	// Start the configuration manager
	ctx := context.Background()
	if err := configManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start config manager: %v", err)
	}
	defer configManager.Stop()
	
	fmt.Println("=== xApp Configuration Manager Example ===")
	
	// Example 1: Set configuration with schema validation
	fmt.Println("\n1. Setting up configuration schema...")
	
	schema := &dashboard.ConfigSchema{
		Type: "object",
		Properties: map[string]*dashboard.PropertySpec{
			"threshold": {
				Type:        "number",
				Description: "Alert threshold value",
				Minimum:     &[]float64{0.0}[0],
				Maximum:     &[]float64{100.0}[0],
			},
			"mode": {
				Type:        "string",
				Description: "Operation mode",
				Enum:        []string{"active", "passive", "standby"},
			},
			"enabled": {
				Type:        "boolean",
				Description: "Enable/disable the xApp",
			},
			"interval": {
				Type:        "integer",
				Description: "Polling interval in seconds",
				Minimum:     &[]float64{1.0}[0],
				Maximum:     &[]float64{3600.0}[0],
			},
		},
		Required: []string{"threshold", "mode"},
	}
	
	if err := configManager.SetConfigSchema("monitoring-xapp", schema); err != nil {
		log.Fatalf("Failed to set schema: %v", err)
	}
	fmt.Println("✓ Schema set for monitoring-xapp")
	
	// Example 2: Set valid configuration
	fmt.Println("\n2. Setting valid configuration...")
	
	config := map[string]interface{}{
		"threshold": 75.5,
		"mode":      "active",
		"enabled":   true,
		"interval":  30,
		"custom":    "additional property",
	}
	
	environment := map[string]string{
		"LOG_LEVEL":    "INFO",
		"METRICS_PORT": "9090",
		"DEBUG":        "false",
	}
	
	if err := configManager.SetConfig("monitoring-xapp", config, environment); err != nil {
		log.Fatalf("Failed to set config: %v", err)
	}
	fmt.Println("✓ Configuration set successfully")
	
	// Example 3: Retrieve and display configuration
	fmt.Println("\n3. Retrieving configuration...")
	
	retrievedConfig, err := configManager.GetConfig("monitoring-xapp")
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}
	
	fmt.Printf("xApp Name: %s\n", retrievedConfig.Name)
	fmt.Printf("Last Updated: %s\n", retrievedConfig.LastUpdated.Format(time.RFC3339))
	fmt.Printf("Checksum: %s\n", retrievedConfig.Checksum)
	fmt.Println("Configuration:")
	for key, value := range retrievedConfig.Config {
		fmt.Printf("  %s: %v\n", key, value)
	}
	fmt.Println("Environment:")
	for key, value := range retrievedConfig.Environment {
		fmt.Printf("  %s: %s\n", key, value)
	}
	
	// Example 4: Configuration watcher
	fmt.Println("\n4. Setting up configuration watcher...")
	
	watcherCalled := make(chan bool, 1)
	watcher := func(config *dashboard.XAppConfig) error {
		fmt.Printf("🔔 Configuration changed for %s at %s\n", 
			config.Name, config.LastUpdated.Format(time.RFC3339))
		watcherCalled <- true
		return nil
	}
	
	configManager.WatchConfig("monitoring-xapp", watcher)
	fmt.Println("✓ Watcher registered")
	
	// Example 5: Update configuration
	fmt.Println("\n5. Updating configuration...")
	
	updates := map[string]interface{}{
		"threshold": 85.0,
		"timeout":   60,
	}
	
	if err := configManager.UpdateConfig("monitoring-xapp", updates); err != nil {
		log.Fatalf("Failed to update config: %v", err)
	}
	fmt.Println("✓ Configuration updated")
	
	// Wait for watcher notification
	select {
	case <-watcherCalled:
		fmt.Println("✓ Watcher was notified of the change")
	case <-time.After(1 * time.Second):
		fmt.Println("⚠ Watcher was not called within timeout")
	}
	
	// Example 6: Try invalid configuration
	fmt.Println("\n6. Testing configuration validation...")
	
	invalidConfig := map[string]interface{}{
		"threshold": 150.0, // Exceeds maximum
		"mode":      "invalid", // Not in enum
	}
	
	if err := configManager.SetConfig("monitoring-xapp", invalidConfig, nil); err != nil {
		fmt.Printf("✓ Validation correctly rejected invalid config: %v\n", err)
	} else {
		fmt.Println("⚠ Validation should have rejected invalid config")
	}
	
	// Example 7: List all configurations
	fmt.Println("\n7. Listing all configurations...")
	
	// Add another xApp config for demonstration
	simpleConfig := map[string]interface{}{
		"name": "simple-xapp",
		"port": 8080,
	}
	configManager.SetConfig("simple-xapp", simpleConfig, nil)
	
	allConfigs, err := configManager.ListConfigs()
	if err != nil {
		log.Fatalf("Failed to list configs: %v", err)
	}
	
	fmt.Printf("Found %d configurations:\n", len(allConfigs))
	for _, cfg := range allConfigs {
		fmt.Printf("  - %s (updated: %s)\n", cfg.Name, cfg.LastUpdated.Format(time.RFC3339))
	}
	
	// Example 8: Delete configuration
	fmt.Println("\n8. Deleting configuration...")
	
	if err := configManager.DeleteConfig("simple-xapp"); err != nil {
		log.Fatalf("Failed to delete config: %v", err)
	}
	fmt.Println("✓ Configuration deleted")
	
	// Verify deletion
	if _, err := configManager.GetConfig("simple-xapp"); err != nil {
		fmt.Println("✓ Configuration no longer exists")
	} else {
		fmt.Println("⚠ Configuration should have been deleted")
	}
	
	fmt.Println("\n=== Example completed successfully ===")
}