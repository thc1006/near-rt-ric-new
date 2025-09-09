/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// GracefulDegradationManager manages graceful degradation of services
type GracefulDegradationManager struct {
	mu                sync.RWMutex
	services          map[string]*ServiceHealth
	degradationRules  map[string]*DegradationRule
	fallbackHandlers  map[string]FallbackHandler
	healthCheckers    map[string]HealthChecker
	isRunning         bool
	ctx               context.Context
	cancel            context.CancelFunc
	checkInterval     time.Duration
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name              string                 `json:"name"`
	Status            ServiceStatus          `json:"status"`
	LastHealthCheck   time.Time              `json:"lastHealthCheck"`
	ConsecutiveFails  int                    `json:"consecutiveFails"`
	ResponseTime      time.Duration          `json:"responseTime"`
	ErrorRate         float64                `json:"errorRate"`
	Availability      float64                `json:"availability"`
	DegradationLevel  DegradationLevel       `json:"degradationLevel"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	ServiceStatusHealthy   ServiceStatus = "HEALTHY"
	ServiceStatusDegraded  ServiceStatus = "DEGRADED"
	ServiceStatusUnhealthy ServiceStatus = "UNHEALTHY"
	ServiceStatusUnknown   ServiceStatus = "UNKNOWN"
)

// DegradationLevel represents the level of service degradation
type DegradationLevel int

const (
	DegradationNone DegradationLevel = iota
	DegradationMinor
	DegradationModerate
	DegradationSevere
	DegradationCritical
)

// String returns the string representation of degradation level
func (dl DegradationLevel) String() string {
	switch dl {
	case DegradationNone:
		return "NONE"
	case DegradationMinor:
		return "MINOR"
	case DegradationModerate:
		return "MODERATE"
	case DegradationSevere:
		return "SEVERE"
	case DegradationCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// DegradationRule defines rules for service degradation
type DegradationRule struct {
	ServiceName       string                 `json:"serviceName"`
	Conditions        []DegradationCondition `json:"conditions"`
	Actions           []DegradationAction    `json:"actions"`
	Priority          int                    `json:"priority"`
	Enabled           bool                   `json:"enabled"`
}

// DegradationCondition defines a condition for triggering degradation
type DegradationCondition struct {
	Type      ConditionType `json:"type"`
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
}

// ConditionType represents the type of degradation condition
type ConditionType string

const (
	ConditionErrorRate     ConditionType = "ERROR_RATE"
	ConditionResponseTime  ConditionType = "RESPONSE_TIME"
	ConditionAvailability  ConditionType = "AVAILABILITY"
	ConditionConsecutiveFails ConditionType = "CONSECUTIVE_FAILS"
)

// DegradationAction defines an action to take when degradation is triggered
type DegradationAction struct {
	Type       ActionType             `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ActionType type is now defined in types.go to avoid redeclaration

// FallbackHandler defines the interface for fallback handlers
type FallbackHandler interface {
	HandleFallback(ctx context.Context, serviceName string, originalError error) (interface{}, error)
	GetFallbackType() string
}



// NewGracefulDegradationManager creates a new graceful degradation manager
func NewGracefulDegradationManager() *GracefulDegradationManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &GracefulDegradationManager{
		services:         make(map[string]*ServiceHealth),
		degradationRules: make(map[string]*DegradationRule),
		fallbackHandlers: make(map[string]FallbackHandler),
		healthCheckers:   make(map[string]HealthChecker),
		ctx:              ctx,
		cancel:           cancel,
		checkInterval:    30 * time.Second,
	}
}

// Start starts the graceful degradation manager
func (gdm *GracefulDegradationManager) Start() error {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	if gdm.isRunning {
		return fmt.Errorf("graceful degradation manager is already running")
	}
	
	// Start health monitoring routine
	go gdm.healthMonitoringRoutine()
	
	// Start degradation evaluation routine
	go gdm.degradationEvaluationRoutine()
	
	gdm.isRunning = true
	log.Println("Graceful degradation manager started")
	return nil
}

// Stop stops the graceful degradation manager
func (gdm *GracefulDegradationManager) Stop() error {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	if !gdm.isRunning {
		return nil
	}
	
	gdm.cancel()
	gdm.isRunning = false
	log.Println("Graceful degradation manager stopped")
	return nil
}

// RegisterService registers a service for monitoring
func (gdm *GracefulDegradationManager) RegisterService(serviceName string, healthChecker HealthChecker) error {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	gdm.services[serviceName] = &ServiceHealth{
		Name:             serviceName,
		Status:           ServiceStatusUnknown,
		LastHealthCheck:  time.Now(),
		DegradationLevel: DegradationNone,
		Metadata:         make(map[string]interface{}),
	}
	
	gdm.healthCheckers[serviceName] = healthChecker
	
	log.Printf("Registered service %s for graceful degradation monitoring", serviceName)
	return nil
}

// RegisterDegradationRule registers a degradation rule
func (gdm *GracefulDegradationManager) RegisterDegradationRule(rule *DegradationRule) error {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	gdm.degradationRules[rule.ServiceName] = rule
	
	log.Printf("Registered degradation rule for service %s", rule.ServiceName)
	return nil
}

// RegisterFallbackHandler registers a fallback handler for a service
func (gdm *GracefulDegradationManager) RegisterFallbackHandler(serviceName string, handler FallbackHandler) error {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	gdm.fallbackHandlers[serviceName] = handler
	
	log.Printf("Registered fallback handler for service %s", serviceName)
	return nil
}

// ExecuteWithDegradation executes a function with graceful degradation support
func (gdm *GracefulDegradationManager) ExecuteWithDegradation(ctx context.Context, serviceName string, fn func() (interface{}, error)) (interface{}, error) {
	gdm.mu.RLock()
	serviceHealth, exists := gdm.services[serviceName]
	gdm.mu.RUnlock()
	
	if !exists {
		// Service not registered, execute normally
		return fn()
	}
	
	// Check if service is healthy enough to execute
	if serviceHealth.Status == ServiceStatusUnhealthy {
		// Use fallback if available
		if handler, exists := gdm.fallbackHandlers[serviceName]; exists {
			log.Printf("Service %s is unhealthy, using fallback", serviceName)
			return handler.HandleFallback(ctx, serviceName, fmt.Errorf("service unhealthy"))
		}
		return nil, fmt.Errorf("service %s is unhealthy and no fallback available", serviceName)
	}
	
	// Apply degradation actions if needed
	if serviceHealth.Status == ServiceStatusDegraded {
		if err := gdm.applyDegradationActions(ctx, serviceName); err != nil {
			log.Printf("Failed to apply degradation actions for service %s: %v", serviceName, err)
		}
	}
	
	// Execute the function
	startTime := time.Now()
	result, err := fn()
	duration := time.Since(startTime)
	
	// Update service health based on execution result
	gdm.updateServiceHealth(serviceName, err == nil, duration)
	
	if err != nil && serviceHealth.DegradationLevel >= DegradationModerate {
		// Use fallback for moderate or higher degradation
		if handler, exists := gdm.fallbackHandlers[serviceName]; exists {
			log.Printf("Service %s execution failed with degradation level %s, using fallback", 
				serviceName, serviceHealth.DegradationLevel.String())
			return handler.HandleFallback(ctx, serviceName, err)
		}
	}
	
	return result, err
}

// GetServiceHealth returns the health status of a service
func (gdm *GracefulDegradationManager) GetServiceHealth(serviceName string) (*ServiceHealth, error) {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	health, exists := gdm.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	
	return health, nil
}

// GetAllServiceHealth returns health status for all services
func (gdm *GracefulDegradationManager) GetAllServiceHealth() map[string]*ServiceHealth {
	gdm.mu.RLock()
	defer gdm.mu.RUnlock()
	
	result := make(map[string]*ServiceHealth)
	for name, health := range gdm.services {
		result[name] = &ServiceHealth{
			Name:             health.Name,
			Status:           health.Status,
			LastHealthCheck:  health.LastHealthCheck,
			ConsecutiveFails: health.ConsecutiveFails,
			ResponseTime:     health.ResponseTime,
			ErrorRate:        health.ErrorRate,
			Availability:     health.Availability,
			DegradationLevel: health.DegradationLevel,
			Metadata:         make(map[string]interface{}),
		}
		// Copy metadata
		for k, v := range health.Metadata {
			result[name].Metadata[k] = v
		}
	}
	
	return result
}

// Private methods

func (gdm *GracefulDegradationManager) healthMonitoringRoutine() {
	ticker := time.NewTicker(gdm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-gdm.ctx.Done():
			return
		case <-ticker.C:
			gdm.performHealthChecks()
		}
	}
}

func (gdm *GracefulDegradationManager) degradationEvaluationRoutine() {
	ticker := time.NewTicker(10 * time.Second) // Evaluate more frequently
	defer ticker.Stop()
	
	for {
		select {
		case <-gdm.ctx.Done():
			return
		case <-ticker.C:
			gdm.evaluateDegradationRules()
		}
	}
}

func (gdm *GracefulDegradationManager) performHealthChecks() {
	gdm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range gdm.healthCheckers {
		checkers[name] = checker
	}
	gdm.mu.RUnlock()
	
	for serviceName, checker := range checkers {
		go gdm.performHealthCheck(serviceName, checker)
	}
}

func (gdm *GracefulDegradationManager) performHealthCheck(serviceName string, checker HealthChecker) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	startTime := time.Now()
	result, err := checker.CheckHealth(ctx)
	duration := time.Since(startTime)
	
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	service, exists := gdm.services[serviceName]
	if !exists {
		return
	}
	
	service.LastHealthCheck = time.Now()
	service.ResponseTime = duration
	
	if err != nil || (result != nil && !result.Healthy) {
		service.ConsecutiveFails++
		service.Status = ServiceStatusUnhealthy
		log.Printf("Health check failed for service %s: %v", serviceName, err)
	} else {
		service.ConsecutiveFails = 0
		service.Status = ServiceStatusHealthy
		
		// Update metadata if available
		if result != nil && result.Metadata != nil {
			for k, v := range result.Metadata {
				service.Metadata[k] = v
			}
		}
	}
	
	// Calculate availability (simplified)
	if service.ConsecutiveFails == 0 {
		service.Availability = 100.0
	} else {
		service.Availability = 100.0 - (float64(service.ConsecutiveFails) * 10.0)
		if service.Availability < 0 {
			service.Availability = 0
		}
	}
}

func (gdm *GracefulDegradationManager) evaluateDegradationRules() {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	for serviceName, rule := range gdm.degradationRules {
		if !rule.Enabled {
			continue
		}
		
		service, exists := gdm.services[serviceName]
		if !exists {
			continue
		}
		
		// Evaluate conditions
		degradationTriggered := false
		for _, condition := range rule.Conditions {
			if gdm.evaluateCondition(service, condition) {
				degradationTriggered = true
				break
			}
		}
		
		// Update degradation level and status
		if degradationTriggered {
			if service.Status == ServiceStatusHealthy {
				service.Status = ServiceStatusDegraded
				service.DegradationLevel = gdm.calculateDegradationLevel(service, rule)
				log.Printf("Service %s degraded to level %s", serviceName, service.DegradationLevel.String())
			}
		} else {
			if service.Status == ServiceStatusDegraded && service.ConsecutiveFails == 0 {
				service.Status = ServiceStatusHealthy
				service.DegradationLevel = DegradationNone
				log.Printf("Service %s recovered from degradation", serviceName)
			}
		}
	}
}

func (gdm *GracefulDegradationManager) evaluateCondition(service *ServiceHealth, condition DegradationCondition) bool {
	switch condition.Type {
	case ConditionErrorRate:
		return service.ErrorRate > condition.Threshold
	case ConditionResponseTime:
		return service.ResponseTime.Seconds()*1000 > condition.Threshold // Convert to milliseconds
	case ConditionAvailability:
		return service.Availability < condition.Threshold
	case ConditionConsecutiveFails:
		return float64(service.ConsecutiveFails) >= condition.Threshold
	default:
		return false
	}
}

func (gdm *GracefulDegradationManager) calculateDegradationLevel(service *ServiceHealth, rule *DegradationRule) DegradationLevel {
	// Simple degradation level calculation based on multiple factors
	score := 0
	
	if service.ErrorRate > 10 {
		score += 2
	} else if service.ErrorRate > 5 {
		score += 1
	}
	
	if service.ResponseTime > 5*time.Second {
		score += 2
	} else if service.ResponseTime > 2*time.Second {
		score += 1
	}
	
	if service.Availability < 50 {
		score += 3
	} else if service.Availability < 80 {
		score += 2
	} else if service.Availability < 95 {
		score += 1
	}
	
	if service.ConsecutiveFails > 10 {
		score += 3
	} else if service.ConsecutiveFails > 5 {
		score += 2
	} else if service.ConsecutiveFails > 2 {
		score += 1
	}
	
	switch {
	case score >= 8:
		return DegradationCritical
	case score >= 6:
		return DegradationSevere
	case score >= 4:
		return DegradationModerate
	case score >= 2:
		return DegradationMinor
	default:
		return DegradationNone
	}
}

func (gdm *GracefulDegradationManager) applyDegradationActions(ctx context.Context, serviceName string) error {
	rule, exists := gdm.degradationRules[serviceName]
	if !exists {
		return nil
	}
	
	for _, action := range rule.Actions {
		if err := gdm.executeAction(ctx, serviceName, action); err != nil {
			log.Printf("Failed to execute degradation action %s for service %s: %v", 
				action.Type, serviceName, err)
		}
	}
	
	return nil
}

func (gdm *GracefulDegradationManager) executeAction(ctx context.Context, serviceName string, action DegradationAction) error {
	switch action.Type {
	case ActionFallback:
		// Fallback is handled in ExecuteWithDegradation
		return nil
	case ActionRateLimit:
		// Implement rate limiting logic
		log.Printf("Applying rate limiting to service %s", serviceName)
		return nil
	case ActionCircuitBreaker:
		// Circuit breaker logic would be integrated here
		log.Printf("Activating circuit breaker for service %s", serviceName)
		return nil
	case ActionLoadShedding:
		// Implement load shedding logic
		log.Printf("Applying load shedding to service %s", serviceName)
		return nil
	case ActionCaching:
		// Implement caching logic
		log.Printf("Enabling aggressive caching for service %s", serviceName)
		return nil
	case ActionRetry:
		// Retry logic would be implemented here
		log.Printf("Adjusting retry parameters for service %s", serviceName)
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (gdm *GracefulDegradationManager) updateServiceHealth(serviceName string, success bool, duration time.Duration) {
	gdm.mu.Lock()
	defer gdm.mu.Unlock()
	
	service, exists := gdm.services[serviceName]
	if !exists {
		return
	}
	
	service.ResponseTime = duration
	
	// Update error rate (simplified exponential moving average)
	if success {
		service.ErrorRate = service.ErrorRate * 0.9 // Decay error rate
	} else {
		service.ErrorRate = service.ErrorRate*0.9 + 10.0 // Increase error rate
	}
	
	if service.ErrorRate > 100 {
		service.ErrorRate = 100
	}
}

// Default fallback handlers

// CachedFallbackHandler provides cached responses as fallback
type CachedFallbackHandler struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

// NewCachedFallbackHandler creates a new cached fallback handler
func NewCachedFallbackHandler() *CachedFallbackHandler {
	return &CachedFallbackHandler{
		cache: make(map[string]interface{}),
	}
}

// HandleFallback returns cached data as fallback
func (cfh *CachedFallbackHandler) HandleFallback(ctx context.Context, serviceName string, originalError error) (interface{}, error) {
	cfh.mu.RLock()
	defer cfh.mu.RUnlock()
	
	if cachedData, exists := cfh.cache[serviceName]; exists {
		log.Printf("Returning cached data for service %s", serviceName)
		return cachedData, nil
	}
	
	return nil, fmt.Errorf("no cached data available for service %s", serviceName)
}

// GetFallbackType returns the fallback type
func (cfh *CachedFallbackHandler) GetFallbackType() string {
	return "CACHED"
}

// SetCachedData sets cached data for a service
func (cfh *CachedFallbackHandler) SetCachedData(serviceName string, data interface{}) {
	cfh.mu.Lock()
	defer cfh.mu.Unlock()
	
	cfh.cache[serviceName] = data
}

// DefaultFallbackHandler provides default responses as fallback
type DefaultFallbackHandler struct {
	defaultResponses map[string]interface{}
}

// NewDefaultFallbackHandler creates a new default fallback handler
func NewDefaultFallbackHandler() *DefaultFallbackHandler {
	return &DefaultFallbackHandler{
		defaultResponses: make(map[string]interface{}),
	}
}

// HandleFallback returns default data as fallback
func (dfh *DefaultFallbackHandler) HandleFallback(ctx context.Context, serviceName string, originalError error) (interface{}, error) {
	if defaultData, exists := dfh.defaultResponses[serviceName]; exists {
		log.Printf("Returning default data for service %s", serviceName)
		return defaultData, nil
	}
	
	// Return a generic default response
	return map[string]interface{}{
		"status":  "degraded",
		"message": "Service temporarily unavailable",
		"service": serviceName,
	}, nil
}

// GetFallbackType returns the fallback type
func (dfh *DefaultFallbackHandler) GetFallbackType() string {
	return "DEFAULT"
}

// SetDefaultResponse sets a default response for a service
func (dfh *DefaultFallbackHandler) SetDefaultResponse(serviceName string, response interface{}) {
	dfh.defaultResponses[serviceName] = response
}