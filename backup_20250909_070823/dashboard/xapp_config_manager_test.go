/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestXAppConfigManager(t *testing.T) {
	// Create temporary directory for test configs
	tempDir, err := os.MkdirTemp("", "xapp-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set config path
	os.Setenv("XAPP_CONFIG_PATH", tempDir)
	defer os.Unsetenv("XAPP_CONFIG_PATH")

	// Create config manager
	cm := NewXAppConfigManager()
	
	// Start config manager
	ctx := context.Background()
	if err := cm.Start(ctx); err != nil {
		t.Fatalf("Failed to start config manager: %v", err)
	}
	defer cm.Stop()

	t.Run("SetAndGetConfig", func(t *testing.T) {
		config := map[string]interface{}{
			"threshold": 10.5,
			"enabled":   true,
			"mode":      "active",
		}
		environment := map[string]string{
			"LOG_LEVEL": "DEBUG",
			"PORT":      "8080",
		}

		// Set configuration
		err := cm.SetConfig("test-xapp", config, environment)
		if err != nil {
			t.Fatalf("Failed to set config: %v", err)
		}

		// Get configuration
		retrievedConfig, err := cm.GetConfig("test-xapp")
		if err != nil {
			t.Fatalf("Failed to get config: %v", err)
		}

		// Verify configuration
		if retrievedConfig.Name != "test-xapp" {
			t.Errorf("Expected name 'test-xapp', got '%s'", retrievedConfig.Name)
		}
		if retrievedConfig.Config["threshold"] != 10.5 {
			t.Errorf("Expected threshold 10.5, got %v", retrievedConfig.Config["threshold"])
		}
		if retrievedConfig.Environment["LOG_LEVEL"] != "DEBUG" {
			t.Errorf("Expected LOG_LEVEL 'DEBUG', got '%s'", retrievedConfig.Environment["LOG_LEVEL"])
		}
	})

	t.Run("ConfigSchema", func(t *testing.T) {
		schema := &ConfigSchema{
			Type: "object",
			Properties: map[string]*PropertySpec{
				"threshold": {
					Type:        "number",
					Description: "Threshold value",
					Minimum:     &[]float64{0.0}[0],
					Maximum:     &[]float64{100.0}[0],
				},
				"mode": {
					Type:        "string",
					Description: "Operation mode",
					Enum:        []string{"active", "passive", "standby"},
				},
			},
			Required: []string{"threshold", "mode"},
		}

		// Set schema
		err := cm.SetConfigSchema("schema-test-xapp", schema)
		if err != nil {
			t.Fatalf("Failed to set config schema: %v", err)
		}

		// Get schema
		retrievedSchema, err := cm.GetConfigSchema("schema-test-xapp")
		if err != nil {
			t.Fatalf("Failed to get config schema: %v", err)
		}

		// Verify schema
		if len(retrievedSchema.Properties) != 2 {
			t.Errorf("Expected 2 properties, got %d", len(retrievedSchema.Properties))
		}
		if len(retrievedSchema.Required) != 2 {
			t.Errorf("Expected 2 required fields, got %d", len(retrievedSchema.Required))
		}
	})

	t.Run("ConfigValidation", func(t *testing.T) {
		schema := &ConfigSchema{
			Type: "object",
			Properties: map[string]*PropertySpec{
				"threshold": {
					Type:    "number",
					Minimum: &[]float64{0.0}[0],
					Maximum: &[]float64{100.0}[0],
				},
				"mode": {
					Type: "string",
					Enum: []string{"active", "passive"},
				},
			},
			Required: []string{"threshold"},
		}

		// Set schema first
		err := cm.SetConfigSchema("validation-test-xapp", schema)
		if err != nil {
			t.Fatalf("Failed to set config schema: %v", err)
		}

		// Test valid configuration
		validConfig := map[string]interface{}{
			"threshold": 50.0,
			"mode":      "active",
		}
		err = cm.SetConfig("validation-test-xapp", validConfig, nil)
		if err != nil {
			t.Errorf("Valid configuration should not fail: %v", err)
		}

		// Test invalid configuration - missing required field
		invalidConfig1 := map[string]interface{}{
			"mode": "active",
		}
		err = cm.SetConfig("validation-test-xapp", invalidConfig1, nil)
		if err == nil {
			t.Error("Configuration missing required field should fail")
		}

		// Test invalid configuration - out of range
		invalidConfig2 := map[string]interface{}{
			"threshold": 150.0,
			"mode":      "active",
		}
		err = cm.SetConfig("validation-test-xapp", invalidConfig2, nil)
		if err == nil {
			t.Error("Configuration with out-of-range value should fail")
		}

		// Test invalid configuration - invalid enum
		invalidConfig3 := map[string]interface{}{
			"threshold": 50.0,
			"mode":      "invalid",
		}
		err = cm.SetConfig("validation-test-xapp", invalidConfig3, nil)
		if err == nil {
			t.Error("Configuration with invalid enum value should fail")
		}
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		// Set initial configuration
		initialConfig := map[string]interface{}{
			"threshold": 10.0,
			"enabled":   true,
		}
		err := cm.SetConfig("update-test-xapp", initialConfig, nil)
		if err != nil {
			t.Fatalf("Failed to set initial config: %v", err)
		}

		// Update configuration
		updates := map[string]interface{}{
			"threshold": 20.0,
			"timeout":   30,
		}
		err = cm.UpdateConfig("update-test-xapp", updates)
		if err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		// Verify updates
		config, err := cm.GetConfig("update-test-xapp")
		if err != nil {
			t.Fatalf("Failed to get updated config: %v", err)
		}

		if config.Config["threshold"] != 20.0 {
			t.Errorf("Expected threshold 20.0, got %v", config.Config["threshold"])
		}
		if config.Config["timeout"] != 30 {
			t.Errorf("Expected timeout 30, got %v", config.Config["timeout"])
		}
		if config.Config["enabled"] != true {
			t.Errorf("Expected enabled true, got %v", config.Config["enabled"])
		}
	})

	t.Run("ListConfigs", func(t *testing.T) {
		// Set multiple configurations
		configs := []string{"list-test-1", "list-test-2", "list-test-3"}
		for _, name := range configs {
			config := map[string]interface{}{
				"name": name,
			}
			err := cm.SetConfig(name, config, nil)
			if err != nil {
				t.Fatalf("Failed to set config for %s: %v", name, err)
			}
		}

		// List configurations
		allConfigs, err := cm.ListConfigs()
		if err != nil {
			t.Fatalf("Failed to list configs: %v", err)
		}

		// Verify we have at least the test configs
		found := 0
		for _, config := range allConfigs {
			for _, testName := range configs {
				if config.Name == testName {
					found++
					break
				}
			}
		}

		if found != len(configs) {
			t.Errorf("Expected to find %d test configs, found %d", len(configs), found)
		}
	})

	t.Run("DeleteConfig", func(t *testing.T) {
		// Set configuration
		config := map[string]interface{}{
			"test": "value",
		}
		err := cm.SetConfig("delete-test-xapp", config, nil)
		if err != nil {
			t.Fatalf("Failed to set config: %v", err)
		}

		// Verify it exists
		_, err = cm.GetConfig("delete-test-xapp")
		if err != nil {
			t.Fatalf("Config should exist before deletion: %v", err)
		}

		// Delete configuration
		err = cm.DeleteConfig("delete-test-xapp")
		if err != nil {
			t.Fatalf("Failed to delete config: %v", err)
		}

		// Verify it's gone
		_, err = cm.GetConfig("delete-test-xapp")
		if err == nil {
			t.Error("Config should not exist after deletion")
		}

		// Verify file is gone
		configFile := filepath.Join(tempDir, "delete-test-xapp.json")
		if _, err := os.Stat(configFile); !os.IsNotExist(err) {
			t.Error("Config file should be deleted")
		}
	})

	t.Run("ConfigWatcher", func(t *testing.T) {
		watcherCalled := false
		var watchedConfig *XAppConfig

		// Register watcher
		watcher := func(config *XAppConfig) error {
			watcherCalled = true
			watchedConfig = config
			return nil
		}
		cm.WatchConfig("watcher-test-xapp", watcher)

		// Set configuration (should trigger watcher)
		config := map[string]interface{}{
			"watched": true,
		}
		err := cm.SetConfig("watcher-test-xapp", config, nil)
		if err != nil {
			t.Fatalf("Failed to set config: %v", err)
		}

		// Give watcher time to be called
		time.Sleep(100 * time.Millisecond)

		// Verify watcher was called
		if !watcherCalled {
			t.Error("Config watcher should have been called")
		}
		if watchedConfig == nil || watchedConfig.Name != "watcher-test-xapp" {
			t.Error("Watcher should have received correct config")
		}
	})

	t.Run("PersistenceAndReload", func(t *testing.T) {
		// Set configuration
		config := map[string]interface{}{
			"persistent": true,
			"value":      42,
		}
		environment := map[string]string{
			"ENV_VAR": "test",
		}
		err := cm.SetConfig("persistence-test-xapp", config, environment)
		if err != nil {
			t.Fatalf("Failed to set config: %v", err)
		}

		// Stop and create new config manager
		cm.Stop()
		
		newCM := NewXAppConfigManager()
		if err := newCM.Start(ctx); err != nil {
			t.Fatalf("Failed to start new config manager: %v", err)
		}
		defer newCM.Stop()

		// Verify configuration was loaded
		loadedConfig, err := newCM.GetConfig("persistence-test-xapp")
		if err != nil {
			t.Fatalf("Failed to get persisted config: %v", err)
		}

		if loadedConfig.Config["persistent"] != true {
			t.Errorf("Expected persistent true, got %v", loadedConfig.Config["persistent"])
		}
		if loadedConfig.Environment["ENV_VAR"] != "test" {
			t.Errorf("Expected ENV_VAR 'test', got '%s'", loadedConfig.Environment["ENV_VAR"])
		}
	})
}

func TestConfigSchemaValidation(t *testing.T) {
	cm := NewXAppConfigManager()

	tests := []struct {
		name        string
		config      map[string]interface{}
		schema      *ConfigSchema
		shouldError bool
	}{
		{
			name: "valid string property",
			config: map[string]interface{}{
				"name": "test",
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"name": {Type: "string"},
				},
			},
			shouldError: false,
		},
		{
			name: "invalid string property",
			config: map[string]interface{}{
				"name": 123,
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"name": {Type: "string"},
				},
			},
			shouldError: true,
		},
		{
			name: "valid number with range",
			config: map[string]interface{}{
				"value": 50.0,
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"value": {
						Type:    "number",
						Minimum: &[]float64{0.0}[0],
						Maximum: &[]float64{100.0}[0],
					},
				},
			},
			shouldError: false,
		},
		{
			name: "number below minimum",
			config: map[string]interface{}{
				"value": -10.0,
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"value": {
						Type:    "number",
						Minimum: &[]float64{0.0}[0],
					},
				},
			},
			shouldError: true,
		},
		{
			name: "valid enum value",
			config: map[string]interface{}{
				"mode": "active",
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"mode": {
						Type: "string",
						Enum: []string{"active", "passive"},
					},
				},
			},
			shouldError: false,
		},
		{
			name: "invalid enum value",
			config: map[string]interface{}{
				"mode": "invalid",
			},
			schema: &ConfigSchema{
				Type: "object",
				Properties: map[string]*PropertySpec{
					"mode": {
						Type: "string",
						Enum: []string{"active", "passive"},
					},
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.validateConfig(tt.config, tt.schema)
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}