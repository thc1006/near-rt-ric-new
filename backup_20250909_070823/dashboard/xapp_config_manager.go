/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// XAppConfigManager manages configuration for xApps
type XAppConfigManager struct {
	configs       map[string]*XAppConfig
	configPath    string
	watchers      map[string][]ConfigWatcher
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	updateChannel chan ConfigUpdate
}

// XAppConfig represents the configuration for an xApp
type XAppConfig struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Config       map[string]interface{} `json:"config"`
	Environment  map[string]string      `json:"environment"`
	Schema       *ConfigSchema          `json:"schema,omitempty"`
	LastUpdated  time.Time              `json:"lastUpdated"`
	Checksum     string                 `json:"checksum"`
}

// ConfigSchema defines the schema for configuration validation
type ConfigSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]*PropertySpec  `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySpec defines a configuration property specification
type PropertySpec struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}

// ConfigWatcher defines a callback for configuration changes
type ConfigWatcher func(config *XAppConfig) error

// ConfigUpdate represents a configuration update event
type ConfigUpdate struct {
	XAppName string
	Config   *XAppConfig
	Action   ConfigAction
}

// ConfigAction represents the type of configuration action
type ConfigAction string

const (
	ConfigActionCreate ConfigAction = "CREATE"
	ConfigActionUpdate ConfigAction = "UPDATE"
	ConfigActionDelete ConfigAction = "DELETE"
)

// NewXAppConfigManager creates a new xApp configuration manager
func NewXAppConfigManager() *XAppConfigManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	configPath := os.Getenv("XAPP_CONFIG_PATH")
	if configPath == "" {
		configPath = "/opt/ric/config"
	}
	
	return &XAppConfigManager{
		configs:       make(map[string]*XAppConfig),
		configPath:    configPath,
		watchers:      make(map[string][]ConfigWatcher),
		ctx:           ctx,
		cancel:        cancel,
		updateChannel: make(chan ConfigUpdate, 100),
	}
}

// Start starts the configuration manager
func (cm *XAppConfigManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	log.Println("Starting xApp Configuration Manager...")
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(cm.configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Load existing configurations
	if err := cm.loadConfigurations(); err != nil {
		log.Printf("Warning: failed to load existing configurations: %v", err)
	}
	
	// Start configuration update processor
	go cm.processConfigUpdates()
	
	log.Println("xApp Configuration Manager started")
	return nil
}

// Stop stops the configuration manager
func (cm *XAppConfigManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	log.Println("Stopping xApp Configuration Manager...")
	
	if cm.cancel != nil {
		cm.cancel()
	}
	
	close(cm.updateChannel)
	
	log.Println("xApp Configuration Manager stopped")
}

// SetConfig sets the configuration for an xApp
func (cm *XAppConfigManager) SetConfig(name string, config map[string]interface{}, environment map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Create or update configuration
	xappConfig := &XAppConfig{
		Name:        name,
		Config:      config,
		Environment: environment,
		LastUpdated: time.Now().UTC(),
	}
	
	// Calculate checksum
	checksum, err := cm.calculateChecksum(xappConfig)
	if err != nil {
		return fmt.Errorf("failed to calculate config checksum: %w", err)
	}
	xappConfig.Checksum = checksum
	
	// Validate configuration if schema exists
	if existingConfig, exists := cm.configs[name]; exists && existingConfig.Schema != nil {
		if err := cm.validateConfig(config, existingConfig.Schema); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		xappConfig.Schema = existingConfig.Schema
		xappConfig.Version = existingConfig.Version
	}
	
	// Store configuration
	cm.configs[name] = xappConfig
	
	// Save to file
	if err := cm.saveConfiguration(xappConfig); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	// Notify watchers
	action := ConfigActionCreate
	if _, exists := cm.configs[name]; exists {
		action = ConfigActionUpdate
	}
	
	cm.notifyWatchers(name, xappConfig, action)
	
	log.Printf("Configuration set for xApp %s", name)
	return nil
}

// GetConfig gets the configuration for an xApp
func (cm *XAppConfigManager) GetConfig(name string) (*XAppConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	config, exists := cm.configs[name]
	if !exists {
		return nil, fmt.Errorf("configuration not found for xApp %s", name)
	}
	
	// Return a copy to prevent external modifications
	configCopy := *config
	configCopy.Config = make(map[string]interface{})
	for k, v := range config.Config {
		configCopy.Config[k] = v
	}
	configCopy.Environment = make(map[string]string)
	for k, v := range config.Environment {
		configCopy.Environment[k] = v
	}
	
	return &configCopy, nil
}

// DeleteConfig deletes the configuration for an xApp
func (cm *XAppConfigManager) DeleteConfig(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("configuration not found for xApp %s", name)
	}
	
	// Remove from memory
	delete(cm.configs, name)
	
	// Remove file
	configFile := filepath.Join(cm.configPath, fmt.Sprintf("%s.json", name))
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}
	
	// Notify watchers
	cm.notifyWatchers(name, config, ConfigActionDelete)
	
	log.Printf("Configuration deleted for xApp %s", name)
	return nil
}

// ListConfigs lists all xApp configurations
func (cm *XAppConfigManager) ListConfigs() ([]*XAppConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	configs := make([]*XAppConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		// Return copies to prevent external modifications
		configCopy := *config
		configCopy.Config = make(map[string]interface{})
		for k, v := range config.Config {
			configCopy.Config[k] = v
		}
		configCopy.Environment = make(map[string]string)
		for k, v := range config.Environment {
			configCopy.Environment[k] = v
		}
		configs = append(configs, &configCopy)
	}
	
	return configs, nil
}

// SetConfigSchema sets the configuration schema for an xApp
func (cm *XAppConfigManager) SetConfigSchema(name string, schema *ConfigSchema) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	config, exists := cm.configs[name]
	if !exists {
		// Create empty config with schema
		config = &XAppConfig{
			Name:        name,
			Config:      make(map[string]interface{}),
			Environment: make(map[string]string),
			LastUpdated: time.Now().UTC(),
		}
		cm.configs[name] = config
	}
	
	config.Schema = schema
	
	// Validate existing configuration against new schema
	if len(config.Config) > 0 {
		if err := cm.validateConfig(config.Config, schema); err != nil {
			return fmt.Errorf("existing configuration is invalid against new schema: %w", err)
		}
	}
	
	// Save configuration with schema
	if err := cm.saveConfiguration(config); err != nil {
		return fmt.Errorf("failed to save configuration with schema: %w", err)
	}
	
	log.Printf("Configuration schema set for xApp %s", name)
	return nil
}

// GetConfigSchema gets the configuration schema for an xApp
func (cm *XAppConfigManager) GetConfigSchema(name string) (*ConfigSchema, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	config, exists := cm.configs[name]
	if !exists {
		return nil, fmt.Errorf("configuration not found for xApp %s", name)
	}
	
	if config.Schema == nil {
		return nil, fmt.Errorf("no schema defined for xApp %s", name)
	}
	
	return config.Schema, nil
}

// WatchConfig registers a watcher for configuration changes
func (cm *XAppConfigManager) WatchConfig(name string, watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.watchers[name] == nil {
		cm.watchers[name] = make([]ConfigWatcher, 0)
	}
	
	cm.watchers[name] = append(cm.watchers[name], watcher)
	log.Printf("Configuration watcher registered for xApp %s", name)
}

// UpdateConfig updates specific configuration values for an xApp
func (cm *XAppConfigManager) UpdateConfig(name string, updates map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("configuration not found for xApp %s", name)
	}
	
	// Create updated configuration
	updatedConfig := make(map[string]interface{})
	for k, v := range config.Config {
		updatedConfig[k] = v
	}
	for k, v := range updates {
		updatedConfig[k] = v
	}
	
	// Validate updated configuration
	if config.Schema != nil {
		if err := cm.validateConfig(updatedConfig, config.Schema); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}
	
	// Update configuration
	config.Config = updatedConfig
	config.LastUpdated = time.Now().UTC()
	
	// Calculate new checksum
	checksum, err := cm.calculateChecksum(config)
	if err != nil {
		return fmt.Errorf("failed to calculate config checksum: %w", err)
	}
	config.Checksum = checksum
	
	// Save configuration
	if err := cm.saveConfiguration(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	// Notify watchers
	cm.notifyWatchers(name, config, ConfigActionUpdate)
	
	log.Printf("Configuration updated for xApp %s", name)
	return nil
}

// loadConfigurations loads existing configurations from disk
func (cm *XAppConfigManager) loadConfigurations() error {
	files, err := filepath.Glob(filepath.Join(cm.configPath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list config files: %w", err)
	}
	
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: failed to read config file %s: %v", file, err)
			continue
		}
		
		var config XAppConfig
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("Warning: failed to parse config file %s: %v", file, err)
			continue
		}
		
		cm.configs[config.Name] = &config
		log.Printf("Loaded configuration for xApp %s", config.Name)
	}
	
	return nil
}

// saveConfiguration saves a configuration to disk
func (cm *XAppConfigManager) saveConfiguration(config *XAppConfig) error {
	configFile := filepath.Join(cm.configPath, fmt.Sprintf("%s.json", config.Name))
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// validateConfig validates configuration against schema
func (cm *XAppConfigManager) validateConfig(config map[string]interface{}, schema *ConfigSchema) error {
	// Check required properties
	for _, required := range schema.Required {
		if _, exists := config[required]; !exists {
			return fmt.Errorf("required property '%s' is missing", required)
		}
	}
	
	// Validate each property
	for key, value := range config {
		spec, exists := schema.Properties[key]
		if !exists {
			continue // Allow additional properties
		}
		
		if err := cm.validateProperty(key, value, spec); err != nil {
			return err
		}
	}
	
	return nil
}

// validateProperty validates a single property against its specification
func (cm *XAppConfigManager) validateProperty(key string, value interface{}, spec *PropertySpec) error {
	// Type validation
	switch spec.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("property '%s' must be a string", key)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("property '%s' must be a number", key)
		}
	case "integer":
		if _, ok := value.(int); !ok {
			return fmt.Errorf("property '%s' must be an integer", key)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("property '%s' must be a boolean", key)
		}
	}
	
	// Enum validation
	if len(spec.Enum) > 0 {
		valueStr := fmt.Sprintf("%v", value)
		found := false
		for _, enumValue := range spec.Enum {
			if enumValue == valueStr {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("property '%s' must be one of %v", key, spec.Enum)
		}
	}
	
	// Range validation for numbers
	if spec.Type == "number" || spec.Type == "integer" {
		if numValue, ok := value.(float64); ok {
			if spec.Minimum != nil && numValue < *spec.Minimum {
				return fmt.Errorf("property '%s' must be >= %f", key, *spec.Minimum)
			}
			if spec.Maximum != nil && numValue > *spec.Maximum {
				return fmt.Errorf("property '%s' must be <= %f", key, *spec.Maximum)
			}
		}
	}
	
	return nil
}

// calculateChecksum calculates a checksum for the configuration
func (cm *XAppConfigManager) calculateChecksum(config *XAppConfig) (string, error) {
	// Create a normalized representation for checksum calculation
	normalized := map[string]interface{}{
		"config":      config.Config,
		"environment": config.Environment,
	}
	
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	
	// Simple checksum using string length and content hash
	// In production, use a proper hash function like SHA-256
	checksum := fmt.Sprintf("%x", len(data))
	for _, b := range data {
		checksum += fmt.Sprintf("%02x", b)
	}
	
	return checksum[:16], nil // Return first 16 characters
}

// notifyWatchers notifies all watchers of configuration changes
func (cm *XAppConfigManager) notifyWatchers(name string, config *XAppConfig, action ConfigAction) {
	watchers := cm.watchers[name]
	if len(watchers) == 0 {
		return
	}
	
	// Send update to channel for async processing
	select {
	case cm.updateChannel <- ConfigUpdate{
		XAppName: name,
		Config:   config,
		Action:   action,
	}:
	default:
		log.Printf("Warning: config update channel full, dropping update for %s", name)
	}
}

// processConfigUpdates processes configuration updates asynchronously
func (cm *XAppConfigManager) processConfigUpdates() {
	for {
		select {
		case update := <-cm.updateChannel:
			cm.processConfigUpdate(update)
		case <-cm.ctx.Done():
			return
		}
	}
}

// processConfigUpdate processes a single configuration update
func (cm *XAppConfigManager) processConfigUpdate(update ConfigUpdate) {
	watchers := cm.watchers[update.XAppName]
	
	for _, watcher := range watchers {
		go func(w ConfigWatcher) {
			if err := w(update.Config); err != nil {
				log.Printf("Configuration watcher error for %s: %v", update.XAppName, err)
			}
		}(watcher)
	}
}