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

// CircuitBreaker and CircuitState types moved to types.go to avoid redeclaration

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig represents configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name             string        `json:"name"`
	MaxFailures      int           `json:"maxFailures"`
	Timeout          time.Duration `json:"timeout"`
	ResetTimeout     time.Duration `json:"resetTimeout"`
	HalfOpenMaxCalls int           `json:"halfOpenMaxCalls"`
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	mu             sync.RWMutex
	circuitBreakers map[string]*CircuitBreaker
	defaultConfig   *CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			Name:             "default",
			MaxFailures:      5,
			Timeout:          30 * time.Second,
			ResetTimeout:     60 * time.Second,
			HalfOpenMaxCalls: 3,
		}
	}
	
	return &CircuitBreaker{
		name:             config.Name,
		state:            StateClosed,
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		resetTimeout:     config.ResetTimeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	cb.totalCalls++
	
	// Check if circuit breaker allows the call
	if !cb.allowRequest() {
		cb.mu.Unlock()
		return fmt.Errorf("circuit breaker %s is OPEN", cb.name)
	}
	
	// For half-open state, limit concurrent calls
	if cb.state == StateHalfOpen {
		if cb.successCount+cb.failureCount >= cb.halfOpenMaxCalls {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker %s half-open call limit reached", cb.name)
		}
	}
	
	cb.mu.Unlock()
	
	// Execute the function with timeout
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			cb.onFailure()
			return err
		}
		cb.onSuccess()
		return nil
		
	case <-time.After(cb.timeout):
		cb.onTimeout()
		return fmt.Errorf("circuit breaker %s timeout after %v", cb.name, cb.timeout)
		
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetOnStateChange sets the callback for state changes
func (cb *CircuitBreaker) SetOnStateChange(callback func(name string, from, to CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.onStateChange = callback
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return cb.state
}

// GetStatistics returns statistics for the circuit breaker
func (cb *CircuitBreaker) GetStatistics() *CircuitBreakerStatistics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return &CircuitBreakerStatistics{
		Name:           cb.name,
		State:          cb.state.String(),
		TotalCalls:     cb.totalCalls,
		TotalFailures:  cb.totalFailures,
		TotalSuccesses: cb.totalSuccesses,
		TotalTimeouts:  cb.totalTimeouts,
		FailureRate:    cb.calculateFailureRate(),
		LastFailure:    cb.lastFailureTime,
		LastSuccess:    cb.lastSuccessTime,
		NextAttempt:    cb.nextAttempt,
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.nextAttempt = time.Time{}
	
	if cb.onStateChange != nil && oldState != StateClosed {
		go cb.onStateChange(cb.name, oldState, StateClosed)
	}
	
	log.Printf("Circuit breaker %s reset to CLOSED state", cb.name)
}

// Private methods

func (cb *CircuitBreaker) allowRequest() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Now().After(cb.nextAttempt)
	case StateHalfOpen:
		return cb.successCount+cb.failureCount < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.totalSuccesses++
	cb.lastSuccessTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
		
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxCalls {
			cb.setState(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.totalFailures++
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.maxFailures {
			cb.setState(StateOpen)
		}
		
	case StateHalfOpen:
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) onTimeout() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.totalTimeouts++
	cb.onFailure() // Treat timeout as failure
}

func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	
	switch newState {
	case StateOpen:
		cb.nextAttempt = time.Now().Add(cb.resetTimeout)
		log.Printf("Circuit breaker %s opened due to %d failures", cb.name, cb.failureCount)
		
	case StateHalfOpen:
		cb.successCount = 0
		cb.failureCount = 0
		log.Printf("Circuit breaker %s transitioned to half-open", cb.name)
		
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
		cb.nextAttempt = time.Time{}
		log.Printf("Circuit breaker %s closed", cb.name)
	}
	
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

func (cb *CircuitBreaker) calculateFailureRate() float64 {
	if cb.totalCalls == 0 {
		return 0.0
	}
	return float64(cb.totalFailures) / float64(cb.totalCalls) * 100
}

// CircuitBreakerManager implementation

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		defaultConfig: &CircuitBreakerConfig{
			MaxFailures:      5,
			Timeout:          30 * time.Second,
			ResetTimeout:     60 * time.Second,
			HalfOpenMaxCalls: 3,
		},
	}
}

// GetCircuitBreaker returns a circuit breaker by name, creating it if it doesn't exist
func (cbm *CircuitBreakerManager) GetCircuitBreaker(name string) *CircuitBreaker {
	cbm.mu.RLock()
	cb, exists := cbm.circuitBreakers[name]
	cbm.mu.RUnlock()
	
	if exists {
		return cb
	}
	
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	// Double-check after acquiring write lock
	if cb, exists := cbm.circuitBreakers[name]; exists {
		return cb
	}
	
	// Create new circuit breaker
	config := *cbm.defaultConfig
	config.Name = name
	cb = NewCircuitBreaker(&config)
	cbm.circuitBreakers[name] = cb
	
	log.Printf("Created new circuit breaker: %s", name)
	return cb
}

// CreateCircuitBreaker creates a circuit breaker with custom configuration
func (cbm *CircuitBreakerManager) CreateCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	cb := NewCircuitBreaker(config)
	cbm.circuitBreakers[config.Name] = cb
	
	log.Printf("Created circuit breaker %s with custom config", config.Name)
	return cb
}

// GetAllCircuitBreakers returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAllCircuitBreakers() map[string]*CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.circuitBreakers {
		result[name] = cb
	}
	
	return result
}

// GetStatistics returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStatistics() map[string]*CircuitBreakerStatistics {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	stats := make(map[string]*CircuitBreakerStatistics)
	for name, cb := range cbm.circuitBreakers {
		stats[name] = cb.GetStatistics()
	}
	
	return stats
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	for _, cb := range cbm.circuitBreakers {
		cb.Reset()
	}
	
	log.Printf("Reset all %d circuit breakers", len(cbm.circuitBreakers))
}

// SetDefaultConfig sets the default configuration for new circuit breakers
func (cbm *CircuitBreakerManager) SetDefaultConfig(config *CircuitBreakerConfig) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	cbm.defaultConfig = config
}

// Data structures

// CircuitBreakerStatistics represents statistics for a circuit breaker
type CircuitBreakerStatistics struct {
	Name           string        `json:"name"`
	State          string        `json:"state"`
	TotalCalls     int64         `json:"totalCalls"`
	TotalFailures  int64         `json:"totalFailures"`
	TotalSuccesses int64         `json:"totalSuccesses"`
	TotalTimeouts  int64         `json:"totalTimeouts"`
	FailureRate    float64       `json:"failureRate"`
	LastFailure    time.Time     `json:"lastFailure"`
	LastSuccess    time.Time     `json:"lastSuccess"`
	NextAttempt    time.Time     `json:"nextAttempt"`
}

// Utility functions for common circuit breaker patterns

// WithCircuitBreaker wraps a function with circuit breaker protection
func WithCircuitBreaker(cbm *CircuitBreakerManager, name string, fn func() error) func(context.Context) error {
	return func(ctx context.Context) error {
		cb := cbm.GetCircuitBreaker(name)
		return cb.Execute(ctx, fn)
	}
}

// WithCircuitBreakerAndFallback wraps a function with circuit breaker and fallback
func WithCircuitBreakerAndFallback(cbm *CircuitBreakerManager, name string, fn func() error, fallback func() error) func(context.Context) error {
	return func(ctx context.Context) error {
		cb := cbm.GetCircuitBreaker(name)
		err := cb.Execute(ctx, fn)
		if err != nil && cb.GetState() == StateOpen {
			log.Printf("Circuit breaker %s is open, executing fallback", name)
			return fallback()
		}
		return err
	}
}

// CircuitBreakerMiddleware provides HTTP middleware with circuit breaker protection
type CircuitBreakerMiddleware struct {
	manager *CircuitBreakerManager
}

// NewCircuitBreakerMiddleware creates a new circuit breaker middleware
func NewCircuitBreakerMiddleware(manager *CircuitBreakerManager) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		manager: manager,
	}
}

// Wrap wraps a function with circuit breaker middleware
func (cbm *CircuitBreakerMiddleware) Wrap(name string, fn func() error) func(context.Context) error {
	return WithCircuitBreaker(cbm.manager, name, fn)
}

// WrapWithFallback wraps a function with circuit breaker and fallback middleware
func (cbm *CircuitBreakerMiddleware) WrapWithFallback(name string, fn func() error, fallback func() error) func(context.Context) error {
	return WithCircuitBreakerAndFallback(cbm.manager, name, fn, fallback)
}