/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"testing"
	"time"
	"unsafe"
)

func TestPerformanceOptimizer(t *testing.T) {
	optimizer := NewPerformanceOptimizer()
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	// Start the optimizer
	err := optimizer.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start performance optimizer: %v", err)
	}
	defer optimizer.Stop()
	
	// Test message processing
	testData := []byte("test message data")
	err = optimizer.ProcessMessage(ctx, WorkTypeE2APMessage, testData, PriorityHigh)
	if err != nil {
		t.Errorf("Failed to process message: %v", err)
	}
	
	// Wait a bit for processing
	time.Sleep(time.Millisecond * 100)
	
	// Check metrics
	metrics := optimizer.GetMetrics()
	if metrics.ProcessedMessages == 0 {
		t.Error("Expected processed messages to be greater than 0")
	}
}

func TestFastHashMap(t *testing.T) {
	hashMap := NewFastHashMap(16)
	
	// Test basic operations
	key := "test-key"
	value := "test-value"
	valuePtr := unsafe.Pointer(&value)
	
	// Put
	hashMap.Put(key, valuePtr)
	
	// Get
	retrievedPtr, exists := hashMap.Get(key)
	if !exists {
		t.Error("Expected key to exist")
	}
	
	retrievedValue := *(*string)(retrievedPtr)
	if retrievedValue != value {
		t.Errorf("Expected %s, got %s", value, retrievedValue)
	}
	
	// Size
	if hashMap.Size() != 1 {
		t.Errorf("Expected size 1, got %d", hashMap.Size())
	}
	
	// Delete
	deleted := hashMap.Delete(key)
	if !deleted {
		t.Error("Expected key to be deleted")
	}
	
	if hashMap.Size() != 0 {
		t.Errorf("Expected size 0 after deletion, got %d", hashMap.Size())
	}
}

func TestLockFreeQueue(t *testing.T) {
	queue := NewLockFreeQueue()
	
	// Test enqueue/dequeue
	testData := "test data"
	dataPtr := unsafe.Pointer(&testData)
	
	queue.Enqueue(dataPtr)
	
	if queue.Size() != 1 {
		t.Errorf("Expected size 1, got %d", queue.Size())
	}
	
	retrievedPtr, ok := queue.Dequeue()
	if !ok {
		t.Error("Expected successful dequeue")
	}
	
	retrievedData := *(*string)(retrievedPtr)
	if retrievedData != testData {
		t.Errorf("Expected %s, got %s", testData, retrievedData)
	}
	
	if queue.Size() != 0 {
		t.Errorf("Expected size 0 after dequeue, got %d", queue.Size())
	}
}

func TestRingBuffer(t *testing.T) {
	buffer := NewRingBuffer(4)
	
	// Test put/get
	testData := []string{"data1", "data2", "data3"}
	
	for _, data := range testData {
		dataPtr := unsafe.Pointer(&data)
		success := buffer.Put(dataPtr)
		if !success {
			t.Error("Expected successful put")
		}
	}
	
	if buffer.Size() != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), buffer.Size())
	}
	
	for i, expectedData := range testData {
		retrievedPtr, ok := buffer.Get()
		if !ok {
			t.Errorf("Expected successful get at index %d", i)
		}
		
		retrievedData := *(*string)(retrievedPtr)
		if retrievedData != expectedData {
			t.Errorf("Expected %s, got %s", expectedData, retrievedData)
		}
	}
	
	if buffer.Size() != 0 {
		t.Errorf("Expected size 0 after getting all items, got %d", buffer.Size())
	}
}

func TestTrieIndex(t *testing.T) {
	trie := NewTrieIndex()
	
	// Test insert/search
	testData := map[string]string{
		"e2node1": "node1-data",
		"e2node2": "node2-data",
		"e2node10": "node10-data",
	}
	
	for key, value := range testData {
		valuePtr := unsafe.Pointer(&value)
		trie.Insert(key, valuePtr)
	}
	
	// Test exact search
	for key, expectedValue := range testData {
		retrievedPtr, found := trie.Search(key)
		if !found {
			t.Errorf("Expected to find key %s", key)
		}
		
		retrievedValue := *(*string)(retrievedPtr)
		if retrievedValue != expectedValue {
			t.Errorf("Expected %s, got %s", expectedValue, retrievedValue)
		}
	}
	
	// Test prefix search
	results := trie.PrefixSearch("e2node1")
	if len(results) != 2 { // e2node1 and e2node10
		t.Errorf("Expected 2 results for prefix 'e2node1', got %d", len(results))
	}
}

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(2)
	
	// Test put/get
	key1, value1 := "key1", "value1"
	key2, value2 := "key2", "value2"
	key3, value3 := "key3", "value3"
	
	cache.Put(key1, unsafe.Pointer(&value1))
	cache.Put(key2, unsafe.Pointer(&value2))
	
	// Get key1 (should exist)
	retrievedPtr, found := cache.Get(key1)
	if !found {
		t.Error("Expected to find key1")
	}
	retrievedValue := *(*string)(retrievedPtr)
	if retrievedValue != value1 {
		t.Errorf("Expected %s, got %s", value1, retrievedValue)
	}
	
	// Add key3 (should evict key2 since key1 was recently accessed)
	cache.Put(key3, unsafe.Pointer(&value3))
	
	// key2 should be evicted
	_, found = cache.Get(key2)
	if found {
		t.Error("Expected key2 to be evicted")
	}
	
	// key1 and key3 should still exist
	_, found = cache.Get(key1)
	if !found {
		t.Error("Expected key1 to still exist")
	}
	
	_, found = cache.Get(key3)
	if !found {
		t.Error("Expected key3 to exist")
	}
}

func TestLoadBalancer(t *testing.T) {
	lb := NewLoadBalancer(RoundRobin)
	
	// Add backends
	backend1 := &Backend{
		ID:        "backend1",
		Address:   "127.0.0.1",
		Port:      8001,
		Weight:    1,
		IsHealthy: 1,
	}
	
	backend2 := &Backend{
		ID:        "backend2",
		Address:   "127.0.0.1",
		Port:      8002,
		Weight:    1,
		IsHealthy: 1,
	}
	
	lb.AddBackend(backend1)
	lb.AddBackend(backend2)
	
	// Test round-robin selection
	selected1, err := lb.SelectBackend("test-key")
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	
	selected2, err := lb.SelectBackend("test-key")
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	
	// Should alternate between backends
	if selected1.ID == selected2.ID {
		t.Error("Expected different backends to be selected in round-robin")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Millisecond*100)
	
	// Initially should be closed and allow execution
	if !cb.CanExecute() {
		t.Error("Circuit breaker should initially allow execution")
	}
	
	// Record failures
	cb.RecordFailure()
	cb.RecordFailure()
	
	// Should still allow execution (at threshold)
	if !cb.CanExecute() {
		t.Error("Circuit breaker should allow execution at threshold")
	}
	
	// One more failure should open the circuit
	cb.RecordFailure()
	
	if cb.CanExecute() {
		t.Error("Circuit breaker should be open after exceeding threshold")
	}
	
	// Wait for timeout
	time.Sleep(time.Millisecond * 150)
	
	// Should allow execution again (half-open)
	if !cb.CanExecute() {
		t.Error("Circuit breaker should allow execution after timeout")
	}
	
	// Record success should close the circuit
	cb.RecordSuccess()
	
	if !cb.CanExecute() {
		t.Error("Circuit breaker should be closed after successful execution")
	}
}

func BenchmarkFastHashMap(b *testing.B) {
	hashMap := NewFastHashMap(1000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			valuePtr := unsafe.Pointer(&value)
			
			hashMap.Put(key, valuePtr)
			hashMap.Get(key)
			
			i++
		}
	})
}

func BenchmarkLockFreeQueue(b *testing.B) {
	queue := NewLockFreeQueue()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			data := fmt.Sprintf("data-%d", i)
			dataPtr := unsafe.Pointer(&data)
			
			queue.Enqueue(dataPtr)
			queue.Dequeue()
			
			i++
		}
	})
}

func BenchmarkPerformanceOptimizer(b *testing.B) {
	optimizer := NewPerformanceOptimizer()
	ctx := context.Background()
	
	err := optimizer.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start optimizer: %v", err)
	}
	defer optimizer.Stop()
	
	testData := []byte("benchmark test data")
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			optimizer.ProcessMessage(ctx, WorkTypeE2APMessage, testData, PriorityNormal)
		}
	})
}