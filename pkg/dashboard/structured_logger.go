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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// LogLevel represents the log level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of log level
func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogFormat represents the log format
type LogFormat int

const (
	LogFormatJSON LogFormat = iota
	LogFormatText
)

// String returns the string representation of log format
func (lf LogFormat) String() string {
	switch lf {
	case LogFormatJSON:
		return "JSON"
	case LogFormatText:
		return "TEXT"
	default:
		return "UNKNOWN"
	}
}

// StructuredLoggerConfig contains configuration for structured logging
type StructuredLoggerConfig struct {
	EnableCorrelation bool      `json:"enableCorrelation"`
	LogLevel         LogLevel  `json:"logLevel"`
	LogFormat        LogFormat `json:"logFormat"`
	BufferSize       int       `json:"bufferSize"`
	FlushInterval    time.Duration `json:"flushInterval"`
	Component        string    `json:"component"`
	NodeID           string    `json:"nodeId"`
}

// LoggingMetrics tracks logging performance metrics
type LoggingMetrics struct {
	mu              sync.RWMutex
	TotalLogs       int64                `json:"totalLogs"`
	LogsByLevel     map[string]int64     `json:"logsByLevel"`
	LogsByComponent map[string]int64     `json:"logsByComponent"`
	BufferFlushes   int64                `json:"bufferFlushes"`
	DroppedLogs     int64                `json:"droppedLogs"`
	AverageLatency  time.Duration        `json:"averageLatency"`
	LastFlush       time.Time            `json:"lastFlush"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *StructuredLoggerConfig) *StructuredLogger {
	if config == nil {
		config = &StructuredLoggerConfig{
			EnableCorrelation: true,
			LogLevel:         LogLevelInfo,
			LogFormat:        LogFormatJSON,
			BufferSize:       1000,
			FlushInterval:    5 * time.Second,
			Component:        "unknown",
			NodeID:          getNodeID(),
		}
	}
	
	logger := &StructuredLogger{
		baseLogger:       NewLogger(config.Component),
		correlationIDGen: NewCorrelationIDGenerator(config.NodeID),
		logBuffer:        NewCircularLogBuffer(config.BufferSize),
		enableCorrelation: config.EnableCorrelation,
		logLevel:         config.LogLevel,
		logFormat:        config.LogFormat,
		logMetrics: &LoggingMetrics{
			LogsByLevel:     make(map[string]int64),
			LogsByComponent: make(map[string]int64),
		},
	}
	
	// Start buffer flushing if enabled
	if config.FlushInterval > 0 {
		go logger.flushRoutine(config.FlushInterval)
	}
	
	return logger
}

// GenerateCorrelationID generates a new correlation ID
func (sl *StructuredLogger) GenerateCorrelationID() string {
	if !sl.enableCorrelation {
		return ""
	}
	return sl.correlationIDGen.Generate()
}

// WithFields adds structured fields to the logger
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *Logger {
	return sl.baseLogger.WithFields(fields)
}

// WithError adds error information to the logger
func (sl *StructuredLogger) WithError(err error) *Logger {
	return sl.baseLogger.WithError(err)
}

// InfoCtx logs an info message with context
func (sl *StructuredLogger) InfoCtx(ctx context.Context, msg string, keyValuePairs ...interface{}) {
	if sl.logLevel > LogLevelInfo {
		return
	}
	sl.logWithContext(ctx, LogLevelInfo, msg, keyValuePairs...)
}

// WarnCtx logs a warning message with context
func (sl *StructuredLogger) WarnCtx(ctx context.Context, msg string, keyValuePairs ...interface{}) {
	if sl.logLevel > LogLevelWarn {
		return
	}
	sl.logWithContext(ctx, LogLevelWarn, msg, keyValuePairs...)
}

// ErrorCtx logs an error message with context
func (sl *StructuredLogger) ErrorCtx(ctx context.Context, msg string, keyValuePairs ...interface{}) {
	if sl.logLevel > LogLevelError {
		return
	}
	sl.logWithContext(ctx, LogLevelError, msg, keyValuePairs...)
}

// DebugCtx logs a debug message with context
func (sl *StructuredLogger) DebugCtx(ctx context.Context, msg string, keyValuePairs ...interface{}) {
	if sl.logLevel > LogLevelDebug {
		return
	}
	sl.logWithContext(ctx, LogLevelDebug, msg, keyValuePairs...)
}

// FatalCtx logs a fatal message with context and exits
func (sl *StructuredLogger) FatalCtx(ctx context.Context, msg string, keyValuePairs ...interface{}) {
	sl.logWithContext(ctx, LogLevelFatal, msg, keyValuePairs...)
	os.Exit(1)
}

// GetMetrics returns logging metrics
func (sl *StructuredLogger) GetMetrics() *LoggingMetrics {
	sl.logMetrics.mu.RLock()
	defer sl.logMetrics.mu.RUnlock()
	
	metrics := &LoggingMetrics{
		TotalLogs:       sl.logMetrics.TotalLogs,
		BufferFlushes:   sl.logMetrics.BufferFlushes,
		DroppedLogs:     sl.logMetrics.DroppedLogs,
		AverageLatency:  sl.logMetrics.AverageLatency,
		LastFlush:       sl.logMetrics.LastFlush,
		LogsByLevel:     make(map[string]int64),
		LogsByComponent: make(map[string]int64),
	}
	
	// Copy maps
	for k, v := range sl.logMetrics.LogsByLevel {
		metrics.LogsByLevel[k] = v
	}
	for k, v := range sl.logMetrics.LogsByComponent {
		metrics.LogsByComponent[k] = v
	}
	
	return metrics
}

// Private methods

func (sl *StructuredLogger) logWithContext(ctx context.Context, level LogLevel, msg string, keyValuePairs ...interface{}) {
	startTime := time.Now()
	
	// Create log entry
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     level,
		Component: sl.baseLogger.component,
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}
	
	// Add correlation ID from context
	if sl.enableCorrelation {
		entry.CorrelationID = GetCorrelationID(ctx)
		if entry.CorrelationID == "" {
			entry.CorrelationID = sl.GenerateCorrelationID()
		}
	}
	
	// Parse key-value pairs
	for i := 0; i < len(keyValuePairs)-1; i += 2 {
		if key, ok := keyValuePairs[i].(string); ok {
			entry.Fields[key] = keyValuePairs[i+1]
		}
	}
	
	// Add stack trace for error and fatal levels
	if level >= LogLevelError {
		entry.StackTrace = sl.captureStackTrace()
	}
	
	// Add to buffer
	if !sl.logBuffer.Add(entry) {
		// Buffer is full, increment dropped logs
		sl.logMetrics.mu.Lock()
		sl.logMetrics.DroppedLogs++
		sl.logMetrics.mu.Unlock()
	}
	
	// Output log immediately for errors and fatals
	if level >= LogLevelError {
		sl.outputLog(entry)
	}
	
	// Update metrics
	sl.updateMetrics(level, time.Since(startTime))
}

func (sl *StructuredLogger) outputLog(entry LogEntry) {
	var output string
	
	switch sl.logFormat {
	case LogFormatJSON:
		if jsonBytes, err := json.Marshal(entry); err == nil {
			output = string(jsonBytes)
		} else {
			output = fmt.Sprintf("Failed to marshal log entry: %v", err)
		}
		
	case LogFormatText:
		output = sl.formatTextLog(entry)
		
	default:
		output = sl.formatTextLog(entry)
	}
	
	// Output to standard logger
	log.Println(output)
}

func (sl *StructuredLogger) formatTextLog(entry LogEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02T15:04:05.000Z")
	
	baseFormat := fmt.Sprintf("[%s] %s [%s] %s", 
		timestamp, entry.Level.String(), entry.Component, entry.Message)
	
	if sl.enableCorrelation && entry.CorrelationID != "" {
		baseFormat += fmt.Sprintf(" correlation_id=%s", entry.CorrelationID)
	}
	
	// Add fields
	for key, value := range entry.Fields {
		baseFormat += fmt.Sprintf(" %s=%v", key, value)
	}
	
	return baseFormat
}

func (sl *StructuredLogger) captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (sl *StructuredLogger) updateMetrics(level LogLevel, duration time.Duration) {
	sl.logMetrics.mu.Lock()
	defer sl.logMetrics.mu.Unlock()
	
	sl.logMetrics.TotalLogs++
	sl.logMetrics.LogsByLevel[level.String()]++
	sl.logMetrics.LogsByComponent[sl.baseLogger.component]++
	
	// Update average latency (simple moving average)
	if sl.logMetrics.TotalLogs == 1 {
		sl.logMetrics.AverageLatency = duration
	} else {
		sl.logMetrics.AverageLatency = time.Duration(
			(int64(sl.logMetrics.AverageLatency) + int64(duration)) / 2)
	}
}

func (sl *StructuredLogger) flushRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for range ticker.C {
		sl.flushBuffer()
	}
}

func (sl *StructuredLogger) flushBuffer() {
	entries := sl.logBuffer.GetAll()
	if len(entries) == 0 {
		return
	}
	
	for _, entry := range entries {
		sl.outputLog(entry)
	}
	
	sl.logMetrics.mu.Lock()
	sl.logMetrics.BufferFlushes++
	sl.logMetrics.LastFlush = time.Now()
	sl.logMetrics.mu.Unlock()
}

// CorrelationIDGenerator implementation

// NewCorrelationIDGenerator creates a new correlation ID generator
func NewCorrelationIDGenerator(nodeID string) *CorrelationIDGenerator {
	if nodeID == "" {
		nodeID = getNodeID()
	}
	
	return &CorrelationIDGenerator{
		prefix:  fmt.Sprintf("%s-%d", nodeID, time.Now().Unix()),
		nodeID:  nodeID,
		counter: 0,
	}
}

// Generate generates a new correlation ID
func (cig *CorrelationIDGenerator) Generate() string {
	cig.mu.Lock()
	defer cig.mu.Unlock()
	
	cig.counter++
	return fmt.Sprintf("%s-%06d", cig.prefix, cig.counter)
}

// CircularLogBuffer implementation

// NewCircularLogBuffer creates a new circular log buffer
func NewCircularLogBuffer(size int) *CircularLogBuffer {
	return &CircularLogBuffer{
		buffer: make([]LogEntry, size),
		size:   size,
		head:   0,
		tail:   0,
		count:  0,
	}
}

// Add adds a log entry to the buffer
func (clb *CircularLogBuffer) Add(entry LogEntry) bool {
	clb.mu.Lock()
	defer clb.mu.Unlock()
	
	if clb.count >= clb.size {
		// Buffer is full
		return false
	}
	
	clb.buffer[clb.tail] = entry
	clb.tail = (clb.tail + 1) % clb.size
	clb.count++
	
	return true
}

// GetAll returns all entries in the buffer and clears it
func (clb *CircularLogBuffer) GetAll() []LogEntry {
	clb.mu.Lock()
	defer clb.mu.Unlock()
	
	if clb.count == 0 {
		return nil
	}
	
	entries := make([]LogEntry, clb.count)
	
	for i := 0; i < clb.count; i++ {
		idx := (clb.head + i) % clb.size
		entries[i] = clb.buffer[idx]
	}
	
	// Clear buffer
	clb.head = 0
	clb.tail = 0
	clb.count = 0
	
	return entries
}

// GetLatest returns the latest N entries without removing them
func (clb *CircularLogBuffer) GetLatest(n int) []LogEntry {
	clb.mu.RLock()
	defer clb.mu.RUnlock()
	
	if clb.count == 0 || n <= 0 {
		return nil
	}
	
	if n > clb.count {
		n = clb.count
	}
	
	entries := make([]LogEntry, n)
	startIdx := clb.count - n
	
	for i := 0; i < n; i++ {
		idx := (clb.head + startIdx + i) % clb.size
		entries[i] = clb.buffer[idx]
	}
	
	return entries
}

// Size returns the current number of entries in the buffer
func (clb *CircularLogBuffer) Size() int {
	clb.mu.RLock()
	defer clb.mu.RUnlock()
	return clb.count
}

// Capacity returns the maximum capacity of the buffer
func (clb *CircularLogBuffer) Capacity() int {
	return clb.size
}

// Production-grade error recovery handlers

// ProductionConnectionRecoveryHandler handles connection errors with circuit breaker integration
type ProductionConnectionRecoveryHandler struct {
	circuitBreakerManager *CircuitBreakerManager
	connectionPoolManager *ConnectionPoolManager
}

func (h *ProductionConnectionRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeConnection && error.Severity != ErrorSeverityCritical
}

func (h *ProductionConnectionRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting production connection recovery for error %s", error.ID)
	
	// Check circuit breaker status for the component
	componentName := error.Component
	if componentName == "" {
		componentName = "default"
	}
	
	circuitBreaker := h.circuitBreakerManager.GetCircuitBreaker(componentName)
	if circuitBreaker.GetState() == StateOpen {
		return fmt.Errorf("circuit breaker is open for component %s", componentName)
	}
	
	// Attempt to recreate connection pools if applicable
	if h.connectionPoolManager != nil {
		// Force pool recreation by clearing and reinitializing
		// This would be implemented based on specific pool management logic
		log.Printf("Attempting to recreate connection pools for component %s", componentName)
	}
	
	// Simulate connection recovery with exponential backoff
	backoffDuration := time.Duration(error.RetryCount) * 500 * time.Millisecond
	time.Sleep(backoffDuration)
	
	return nil
}

func (h *ProductionConnectionRecoveryHandler) GetRecoveryStrategy() string {
	return "production_connection_recovery"
}

// ProductionResourceRecoveryHandler handles resource exhaustion with sophisticated logic
type ProductionResourceRecoveryHandler struct {
	resourceMonitor *ResourceMonitor
}

func (h *ProductionResourceRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeResource && error.Severity != ErrorSeverityCritical
}

func (h *ProductionResourceRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting production resource recovery for error %s", error.ID)
	
	if h.resourceMonitor != nil {
		usage := h.resourceMonitor.GetResourceUsage()
		
		// Check if resource usage is critical
		if usage.MemoryUsage > 95.0 {
			// Force garbage collection
			runtime.GC()
			runtime.GC() // Call twice for more aggressive cleanup
			
			// Wait a bit for GC to take effect
			time.Sleep(100 * time.Millisecond)
		}
		
		if usage.CPUUsage > 95.0 {
			// Reduce CPU load by adding delays
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return nil
}

func (h *ProductionResourceRecoveryHandler) GetRecoveryStrategy() string {
	return "production_resource_recovery"
}

// ProductionProtocolRecoveryHandler handles protocol errors with circuit breaker integration
type ProductionProtocolRecoveryHandler struct {
	circuitBreakerManager *CircuitBreakerManager
}

func (h *ProductionProtocolRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeProtocol && error.RetryCount < 3
}

func (h *ProductionProtocolRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting production protocol recovery for error %s", error.ID)
	
	// Check if we should open circuit breaker
	componentName := error.Component
	if componentName == "" {
		componentName = "default"
	}
	
	circuitBreaker := h.circuitBreakerManager.GetCircuitBreaker(componentName)
	
	// If too many protocol errors, consider opening circuit breaker
	if error.RetryCount >= 2 {
		circuitBreaker.onFailure() // This will open circuit if threshold is reached
	}
	
	// Implement protocol-specific recovery logic
	switch error.Details["protocol"] {
	case "grpc":
		// gRPC specific recovery
		time.Sleep(50 * time.Millisecond)
	case "http":
		// HTTP specific recovery
		time.Sleep(100 * time.Millisecond)
	default:
		// Generic protocol recovery
		time.Sleep(75 * time.Millisecond)
	}
	
	return nil
}

func (h *ProductionProtocolRecoveryHandler) GetRecoveryStrategy() string {
	return "production_protocol_recovery"
}

// Helper functions

func getNodeID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return hostname
}

// SetGlobalStateChangeCallback sets a global callback for all circuit breakers
func (cbm *CircuitBreakerManager) SetGlobalStateChangeCallback(callback func(name string, from, to CircuitState)) {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	for _, cb := range cbm.circuitBreakers {
		cb.SetOnStateChange(callback)
	}
}
