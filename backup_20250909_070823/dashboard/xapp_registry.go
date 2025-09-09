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
	"sync"
	"time"
)

// XAppRegistry manages xApp registration and discovery
type XAppRegistry struct {
	xapps      map[string]map[string]*XAppDescriptor // name -> version -> descriptor
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	watchers   []chan XAppRegistryEvent
	watchersMu sync.RWMutex
}

// XAppRegistryEvent represents an event in the xApp registry
type XAppRegistryEvent struct {
	Type       XAppRegistryEventType `json:"type"`
	XAppName   string                `json:"xappName"`
	Version    string                `json:"version"`
	Descriptor *XAppDescriptor       `json:"descriptor,omitempty"`
	Timestamp  time.Time             `json:"timestamp"`
}

// XAppRegistryEventType represents the type of registry event
type XAppRegistryEventType string

const (
	XAppRegistryEventRegister   XAppRegistryEventType = "REGISTER"
	XAppRegistryEventUnregister XAppRegistryEventType = "UNREGISTER"
	XAppRegistryEventUpdate     XAppRegistryEventType = "UPDATE"
)

// XAppSearchCriteria defines search criteria for xApps
type XAppSearchCriteria struct {
	Name          string   `json:"name,omitempty"`
	Version       string   `json:"version,omitempty"`
	ServiceModels []string `json:"serviceModels,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
}

// NewXAppRegistry creates a new xApp registry
func NewXAppRegistry() *XAppRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &XAppRegistry{
		xapps:    make(map[string]map[string]*XAppDescriptor),
		ctx:      ctx,
		cancel:   cancel,
		watchers: make([]chan XAppRegistryEvent, 0),
	}
}

// Start starts the xApp registry
func (r *XAppRegistry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	log.Println("Starting xApp Registry...")
	
	// Start background tasks
	go r.cleanupTask()
	go r.discoveryTask()
	
	log.Println("xApp Registry started successfully")
	return nil
}

// Stop stops the xApp registry
func (r *XAppRegistry) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	log.Println("Stopping xApp Registry...")
	
	// Cancel context to stop background tasks
	r.cancel()
	
	// Close all watchers
	r.watchersMu.Lock()
	for _, watcher := range r.watchers {
		close(watcher)
	}
	r.watchers = nil
	r.watchersMu.Unlock()
	
	log.Println("xApp Registry stopped")
}

// Register registers a new xApp with the registry
func (r *XAppRegistry) Register(descriptor *XAppDescriptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Initialize version map if it doesn't exist
	if r.xapps[descriptor.Name] == nil {
		r.xapps[descriptor.Name] = make(map[string]*XAppDescriptor)
	}
	
	// Check if this version already exists
	if existing, exists := r.xapps[descriptor.Name][descriptor.Version]; exists {
		// Update existing descriptor
		descriptor.CreatedAt = existing.CreatedAt
		descriptor.UpdatedAt = time.Now()
		r.xapps[descriptor.Name][descriptor.Version] = descriptor
		
		// Notify watchers
		r.notifyWatchers(XAppRegistryEvent{
			Type:       XAppRegistryEventUpdate,
			XAppName:   descriptor.Name,
			Version:    descriptor.Version,
			Descriptor: descriptor,
			Timestamp:  time.Now(),
		})
		
		log.Printf("Updated xApp %s v%s in registry", descriptor.Name, descriptor.Version)
	} else {
		// Register new descriptor
		descriptor.CreatedAt = time.Now()
		descriptor.UpdatedAt = time.Now()
		r.xapps[descriptor.Name][descriptor.Version] = descriptor
		
		// Notify watchers
		r.notifyWatchers(XAppRegistryEvent{
			Type:       XAppRegistryEventRegister,
			XAppName:   descriptor.Name,
			Version:    descriptor.Version,
			Descriptor: descriptor,
			Timestamp:  time.Now(),
		})
		
		log.Printf("Registered xApp %s v%s in registry", descriptor.Name, descriptor.Version)
	}
	
	return nil
}

// Unregister removes an xApp from the registry
func (r *XAppRegistry) Unregister(name, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if versions, exists := r.xapps[name]; exists {
		if descriptor, versionExists := versions[version]; versionExists {
			delete(versions, version)
			
			// Remove the name entry if no versions remain
			if len(versions) == 0 {
				delete(r.xapps, name)
			}
			
			// Notify watchers
			r.notifyWatchers(XAppRegistryEvent{
				Type:       XAppRegistryEventUnregister,
				XAppName:   name,
				Version:    version,
				Descriptor: descriptor,
				Timestamp:  time.Now(),
			})
			
			log.Printf("Unregistered xApp %s v%s from registry", name, version)
			return nil
		}
	}
	
	return fmt.Errorf("xApp %s v%s not found in registry", name, version)
}

// GetXApp retrieves an xApp descriptor by name and version
func (r *XAppRegistry) GetXApp(name, version string) (*XAppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if versions, exists := r.xapps[name]; exists {
		if descriptor, versionExists := versions[version]; versionExists {
			// Return a copy to prevent external modification
			descriptorCopy := *descriptor
			return &descriptorCopy, nil
		}
	}
	
	return nil, fmt.Errorf("xApp %s v%s not found", name, version)
}

// ListXApps returns all registered xApps
func (r *XAppRegistry) ListXApps() ([]*XAppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var descriptors []*XAppDescriptor
	
	for _, versions := range r.xapps {
		for _, descriptor := range versions {
			// Return a copy to prevent external modification
			descriptorCopy := *descriptor
			descriptors = append(descriptors, &descriptorCopy)
		}
	}
	
	return descriptors, nil
}

// ListXAppVersions returns all versions of a specific xApp
func (r *XAppRegistry) ListXAppVersions(name string) ([]*XAppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	versions, exists := r.xapps[name]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", name)
	}
	
	var descriptors []*XAppDescriptor
	for _, descriptor := range versions {
		// Return a copy to prevent external modification
		descriptorCopy := *descriptor
		descriptors = append(descriptors, &descriptorCopy)
	}
	
	return descriptors, nil
}

// SearchXApps searches for xApps based on criteria
func (r *XAppRegistry) SearchXApps(criteria XAppSearchCriteria) ([]*XAppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var results []*XAppDescriptor
	
	for _, versions := range r.xapps {
		for _, descriptor := range versions {
			if r.matchesCriteria(descriptor, criteria) {
				// Return a copy to prevent external modification
				descriptorCopy := *descriptor
				results = append(results, &descriptorCopy)
			}
		}
	}
	
	return results, nil
}

// GetLatestVersion returns the latest version of an xApp
func (r *XAppRegistry) GetLatestVersion(name string) (*XAppDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	versions, exists := r.xapps[name]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", name)
	}
	
	var latest *XAppDescriptor
	var latestTime time.Time
	
	for _, descriptor := range versions {
		if descriptor.CreatedAt.After(latestTime) {
			latest = descriptor
			latestTime = descriptor.CreatedAt
		}
	}
	
	if latest == nil {
		return nil, fmt.Errorf("no versions found for xApp %s", name)
	}
	
	// Return a copy to prevent external modification
	latestCopy := *latest
	return &latestCopy, nil
}

// Watch returns a channel that receives registry events
func (r *XAppRegistry) Watch() <-chan XAppRegistryEvent {
	r.watchersMu.Lock()
	defer r.watchersMu.Unlock()
	
	watcher := make(chan XAppRegistryEvent, 100)
	r.watchers = append(r.watchers, watcher)
	
	return watcher
}

// GetRegistryStats returns statistics about the registry
func (r *XAppRegistry) GetRegistryStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	totalApps := len(r.xapps)
	totalVersions := 0
	serviceModelCount := make(map[string]int)
	capabilityCount := make(map[string]int)
	
	for _, versions := range r.xapps {
		totalVersions += len(versions)
		
		for _, descriptor := range versions {
			// Count service models
			for _, sm := range descriptor.ServiceModels {
				serviceModelCount[sm]++
			}
			
			// Count capabilities
			for _, cap := range descriptor.Capabilities {
				capabilityCount[cap]++
			}
		}
	}
	
	return map[string]interface{}{
		"totalApps":         totalApps,
		"totalVersions":     totalVersions,
		"serviceModels":     serviceModelCount,
		"capabilities":      capabilityCount,
		"activeWatchers":    len(r.watchers),
		"timestamp":         time.Now().UTC(),
	}
}

// matchesCriteria checks if a descriptor matches the search criteria
func (r *XAppRegistry) matchesCriteria(descriptor *XAppDescriptor, criteria XAppSearchCriteria) bool {
	// Check name
	if criteria.Name != "" && descriptor.Name != criteria.Name {
		return false
	}
	
	// Check version
	if criteria.Version != "" && descriptor.Version != criteria.Version {
		return false
	}
	
	// Check service models
	if len(criteria.ServiceModels) > 0 {
		for _, requiredSM := range criteria.ServiceModels {
			found := false
			for _, descriptorSM := range descriptor.ServiceModels {
				if descriptorSM == requiredSM {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	
	// Check capabilities
	if len(criteria.Capabilities) > 0 {
		for _, requiredCap := range criteria.Capabilities {
			found := false
			for _, descriptorCap := range descriptor.Capabilities {
				if descriptorCap == requiredCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	
	return true
}

// notifyWatchers sends an event to all watchers
func (r *XAppRegistry) notifyWatchers(event XAppRegistryEvent) {
	r.watchersMu.RLock()
	defer r.watchersMu.RUnlock()
	
	for _, watcher := range r.watchers {
		select {
		case watcher <- event:
		default:
			// Channel is full, skip this watcher
			log.Printf("Warning: xApp registry watcher channel is full, skipping event")
		}
	}
}

// cleanupTask performs periodic cleanup of the registry
func (r *XAppRegistry) cleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.performCleanup()
		}
	}
}

// performCleanup removes stale entries from the registry
func (r *XAppRegistry) performCleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// For now, just log the cleanup operation
	// In the future, this could remove old versions or inactive xApps
	log.Printf("Performing xApp registry cleanup - %d apps registered", len(r.xapps))
}

// discoveryTask performs periodic discovery of new xApps
func (r *XAppRegistry) discoveryTask() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.performDiscovery()
		}
	}
}

// performDiscovery discovers new xApps from various sources
func (r *XAppRegistry) performDiscovery() {
	// For now, just log the discovery operation
	// In the future, this could discover xApps from:
	// - Kubernetes deployments
	// - Helm releases
	// - External registries
	log.Printf("Performing xApp discovery - %d apps currently registered", len(r.xapps))
}

// ExportRegistry exports the registry to JSON
func (r *XAppRegistry) ExportRegistry() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return json.MarshalIndent(r.xapps, "", "  ")
}

// ImportRegistry imports the registry from JSON
func (r *XAppRegistry) ImportRegistry(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var importedXApps map[string]map[string]*XAppDescriptor
	if err := json.Unmarshal(data, &importedXApps); err != nil {
		return fmt.Errorf("failed to unmarshal registry data: %w", err)
	}
	
	// Merge imported xApps with existing ones
	for name, versions := range importedXApps {
		if r.xapps[name] == nil {
			r.xapps[name] = make(map[string]*XAppDescriptor)
		}
		
		for version, descriptor := range versions {
			r.xapps[name][version] = descriptor
		}
	}
	
	log.Printf("Imported %d xApps into registry", len(importedXApps))
	return nil
}