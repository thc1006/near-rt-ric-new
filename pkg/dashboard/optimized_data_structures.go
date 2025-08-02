/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"hash/fnv"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// FastHashMap provides high-performance concurrent hash map
type FastHashMap struct {
	buckets    []bucket
	bucketMask uint64
	size       int64
	loadFactor float64
}

// bucket represents a hash map bucket with lock-free operations
type bucket struct {
	entries []entry
	mu      sync.RWMutex
}

// entry represents a key-value pair in the hash map
type entry struct {
	key       string
	value     unsafe.Pointer
	hash      uint64
	timestamp time.Time
	deleted   int32 // atomic flag for soft deletion
}

// LockFreeQueue provides lock-free FIFO queue for high-throughput scenarios
type LockFreeQueue struct {
	head unsafe.Pointer // *node
	tail unsafe.Pointer // *node
	size int64
}

// node represents a queue node
type node struct {
	data unsafe.Pointer
	next unsafe.Pointer // *node
}

// RingBuffer provides high-performance circular buffer
type RingBuffer struct {
	buffer   []unsafe.Pointer
	capacity int64
	head     int64
	tail     int64
	mask     int64
	mu       sync.RWMutex
}

// TrieIndex provides fast prefix-based lookups for E2 node IDs and subscription IDs
type TrieIndex struct {
	root *trieNode
	mu   sync.RWMutex
}

// trieNode represents a node in the trie
type trieNode struct {
	children map[rune]*trieNode
	value    unsafe.Pointer
	isEnd    bool
}

// BloomFilter provides fast membership testing with low memory footprint
type BloomFilter struct {
	bitArray []uint64
	size     uint64
	hashFuncs int
	mu       sync.RWMutex
}

// LRUCache provides fast LRU cache with O(1) operations
type LRUCache struct {
	capacity int
	items    map[string]*lruNode
	head     *lruNode
	tail     *lruNode
	mu       sync.RWMutex
}

// lruNode represents a node in the LRU cache
type lruNode struct {
	key   string
	value unsafe.Pointer
	prev  *lruNode
	next  *lruNode
}

// SkipList provides fast ordered operations with probabilistic balancing
type SkipList struct {
	header   *skipNode
	level    int
	size     int64
	maxLevel int
	mu       sync.RWMutex
}

// skipNode represents a node in the skip list
type skipNode struct {
	key     string
	value   unsafe.Pointer
	forward []*skipNode
}

// ConcurrentBitSet provides thread-safe bit operations
type ConcurrentBitSet struct {
	bits []uint64
	size uint64
	mu   sync.RWMutex
}

// NewFastHashMap creates a new high-performance hash map
func NewFastHashMap(initialCapacity int) *FastHashMap {
	capacity := nextPowerOfTwo(initialCapacity)
	return &FastHashMap{
		buckets:    make([]bucket, capacity),
		bucketMask: uint64(capacity - 1),
		loadFactor: 0.75,
	}
}

// nextPowerOfTwo returns the next power of two greater than or equal to n
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}

// hash computes hash for a string key
func (fhm *FastHashMap) hash(key string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return h.Sum64()
}

// Put inserts or updates a key-value pair
func (fhm *FastHashMap) Put(key string, value unsafe.Pointer) {
	hash := fhm.hash(key)
	bucketIndex := hash & fhm.bucketMask
	bucket := &fhm.buckets[bucketIndex]
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	// Look for existing entry
	for i := range bucket.entries {
		if bucket.entries[i].key == key && atomic.LoadInt32(&bucket.entries[i].deleted) == 0 {
			bucket.entries[i].value = value
			bucket.entries[i].timestamp = time.Now()
			return
		}
	}
	
	// Add new entry
	bucket.entries = append(bucket.entries, entry{
		key:       key,
		value:     value,
		hash:      hash,
		timestamp: time.Now(),
		deleted:   0,
	})
	
	atomic.AddInt64(&fhm.size, 1)
	
	// Check if resize is needed
	if float64(atomic.LoadInt64(&fhm.size)) > float64(len(fhm.buckets))*fhm.loadFactor {
		go fhm.resize() // Resize in background
	}
}

// Get retrieves a value by key
func (fhm *FastHashMap) Get(key string) (unsafe.Pointer, bool) {
	hash := fhm.hash(key)
	bucketIndex := hash & fhm.bucketMask
	bucket := &fhm.buckets[bucketIndex]
	
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()
	
	for i := range bucket.entries {
		if bucket.entries[i].key == key && atomic.LoadInt32(&bucket.entries[i].deleted) == 0 {
			return bucket.entries[i].value, true
		}
	}
	
	return nil, false
}

// Delete removes a key-value pair (soft delete)
func (fhm *FastHashMap) Delete(key string) bool {
	hash := fhm.hash(key)
	bucketIndex := hash & fhm.bucketMask
	bucket := &fhm.buckets[bucketIndex]
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	for i := range bucket.entries {
		if bucket.entries[i].key == key && atomic.LoadInt32(&bucket.entries[i].deleted) == 0 {
			atomic.StoreInt32(&bucket.entries[i].deleted, 1)
			atomic.AddInt64(&fhm.size, -1)
			return true
		}
	}
	
	return false
}

// Size returns the number of elements
func (fhm *FastHashMap) Size() int64 {
	return atomic.LoadInt64(&fhm.size)
}

// resize doubles the hash map capacity
func (fhm *FastHashMap) resize() {
	oldBuckets := fhm.buckets
	newCapacity := len(oldBuckets) * 2
	newBuckets := make([]bucket, newCapacity)
	newMask := uint64(newCapacity - 1)
	
	// Rehash all entries
	for i := range oldBuckets {
		oldBuckets[i].mu.Lock()
		for j := range oldBuckets[i].entries {
			if atomic.LoadInt32(&oldBuckets[i].entries[j].deleted) == 0 {
				entry := oldBuckets[i].entries[j]
				newBucketIndex := entry.hash & newMask
				newBuckets[newBucketIndex].entries = append(newBuckets[newBucketIndex].entries, entry)
			}
		}
		oldBuckets[i].mu.Unlock()
	}
	
	fhm.buckets = newBuckets
	fhm.bucketMask = newMask
}

// NewLockFreeQueue creates a new lock-free queue
func NewLockFreeQueue() *LockFreeQueue {
	n := &node{}
	return &LockFreeQueue{
		head: unsafe.Pointer(n),
		tail: unsafe.Pointer(n),
	}
}

// Enqueue adds an item to the queue
func (q *LockFreeQueue) Enqueue(data unsafe.Pointer) {
	n := &node{data: data}
	
	for {
		tail := (*node)(atomic.LoadPointer(&q.tail))
		next := (*node)(atomic.LoadPointer(&tail.next))
		
		if tail == (*node)(atomic.LoadPointer(&q.tail)) {
			if next == nil {
				if atomic.CompareAndSwapPointer(&tail.next, unsafe.Pointer(next), unsafe.Pointer(n)) {
					break
				}
			} else {
				atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
			}
		}
	}
	
	atomic.CompareAndSwapPointer(&q.tail, atomic.LoadPointer(&q.tail), unsafe.Pointer(n))
	atomic.AddInt64(&q.size, 1)
}

// Dequeue removes and returns an item from the queue
func (q *LockFreeQueue) Dequeue() (unsafe.Pointer, bool) {
	for {
		head := (*node)(atomic.LoadPointer(&q.head))
		tail := (*node)(atomic.LoadPointer(&q.tail))
		next := (*node)(atomic.LoadPointer(&head.next))
		
		if head == (*node)(atomic.LoadPointer(&q.head)) {
			if head == tail {
				if next == nil {
					return nil, false // Queue is empty
				}
				atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
			} else {
				if next == nil {
					continue
				}
				data := atomic.LoadPointer(&next.data)
				if atomic.CompareAndSwapPointer(&q.head, unsafe.Pointer(head), unsafe.Pointer(next)) {
					atomic.AddInt64(&q.size, -1)
					return data, true
				}
			}
		}
	}
}

// Size returns the queue size
func (q *LockFreeQueue) Size() int64 {
	return atomic.LoadInt64(&q.size)
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(capacity int) *RingBuffer {
	cap := int64(nextPowerOfTwo(capacity))
	return &RingBuffer{
		buffer:   make([]unsafe.Pointer, cap),
		capacity: cap,
		mask:     cap - 1,
	}
}

// Put adds an item to the ring buffer
func (rb *RingBuffer) Put(data unsafe.Pointer) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	nextTail := (rb.tail + 1) & rb.mask
	if nextTail == rb.head {
		return false // Buffer is full
	}
	
	rb.buffer[rb.tail] = data
	rb.tail = nextTail
	return true
}

// Get removes and returns an item from the ring buffer
func (rb *RingBuffer) Get() (unsafe.Pointer, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if rb.head == rb.tail {
		return nil, false // Buffer is empty
	}
	
	data := rb.buffer[rb.head]
	rb.buffer[rb.head] = nil // Clear reference
	rb.head = (rb.head + 1) & rb.mask
	return data, true
}

// Size returns the number of items in the buffer
func (rb *RingBuffer) Size() int64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.tail >= rb.head {
		return rb.tail - rb.head
	}
	return rb.capacity - rb.head + rb.tail
}

// IsFull returns true if the buffer is full
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	return ((rb.tail + 1) & rb.mask) == rb.head
}

// NewTrieIndex creates a new trie index
func NewTrieIndex() *TrieIndex {
	return &TrieIndex{
		root: &trieNode{
			children: make(map[rune]*trieNode),
		},
	}
}

// Insert adds a key-value pair to the trie
func (ti *TrieIndex) Insert(key string, value unsafe.Pointer) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	
	node := ti.root
	for _, char := range key {
		if _, exists := node.children[char]; !exists {
			node.children[char] = &trieNode{
				children: make(map[rune]*trieNode),
			}
		}
		node = node.children[char]
	}
	
	node.value = value
	node.isEnd = true
}

// Search finds a value by exact key match
func (ti *TrieIndex) Search(key string) (unsafe.Pointer, bool) {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	
	node := ti.root
	for _, char := range key {
		if child, exists := node.children[char]; exists {
			node = child
		} else {
			return nil, false
		}
	}
	
	if node.isEnd {
		return node.value, true
	}
	return nil, false
}

// PrefixSearch finds all values with the given prefix
func (ti *TrieIndex) PrefixSearch(prefix string) []unsafe.Pointer {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	
	node := ti.root
	for _, char := range prefix {
		if child, exists := node.children[char]; exists {
			node = child
		} else {
			return nil // No matches
		}
	}
	
	var results []unsafe.Pointer
	ti.collectValues(node, &results)
	return results
}

// collectValues recursively collects all values from a subtree
func (ti *TrieIndex) collectValues(node *trieNode, results *[]unsafe.Pointer) {
	if node.isEnd {
		*results = append(*results, node.value)
	}
	
	for _, child := range node.children {
		ti.collectValues(child, results)
	}
}

// NewBloomFilter creates a new Bloom filter
func NewBloomFilter(expectedItems int, falsePositiveRate float64) *BloomFilter {
	size := uint64(-float64(expectedItems) * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2)))
	hashFuncs := int(float64(size) / float64(expectedItems) * math.Log(2))
	
	return &BloomFilter{
		bitArray:  make([]uint64, (size+63)/64), // Round up to nearest 64-bit word
		size:      size,
		hashFuncs: hashFuncs,
	}
}

// Add adds an item to the Bloom filter
func (bf *BloomFilter) Add(item string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	hashes := bf.getHashes(item)
	for i := 0; i < bf.hashFuncs; i++ {
		bitIndex := hashes[i] % bf.size
		wordIndex := bitIndex / 64
		bitOffset := bitIndex % 64
		bf.bitArray[wordIndex] |= 1 << bitOffset
	}
}

// Contains checks if an item might be in the set
func (bf *BloomFilter) Contains(item string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	hashes := bf.getHashes(item)
	for i := 0; i < bf.hashFuncs; i++ {
		bitIndex := hashes[i] % bf.size
		wordIndex := bitIndex / 64
		bitOffset := bitIndex % 64
		if (bf.bitArray[wordIndex] & (1 << bitOffset)) == 0 {
			return false
		}
	}
	return true
}

// getHashes generates multiple hash values for an item
func (bf *BloomFilter) getHashes(item string) []uint64 {
	h1 := fnv.New64a()
	h1.Write([]byte(item))
	hash1 := h1.Sum64()
	
	h2 := fnv.New64()
	h2.Write([]byte(item))
	hash2 := h2.Sum64()
	
	hashes := make([]uint64, bf.hashFuncs)
	for i := 0; i < bf.hashFuncs; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}
	
	return hashes
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,
		items:    make(map[string]*lruNode),
	}
	
	// Create dummy head and tail nodes
	cache.head = &lruNode{}
	cache.tail = &lruNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

// Get retrieves a value and marks it as recently used
func (lru *LRUCache) Get(key string) (unsafe.Pointer, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	
	if node, exists := lru.items[key]; exists {
		lru.moveToHead(node)
		return node.value, true
	}
	
	return nil, false
}

// Put adds or updates a key-value pair
func (lru *LRUCache) Put(key string, value unsafe.Pointer) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	
	if node, exists := lru.items[key]; exists {
		node.value = value
		lru.moveToHead(node)
		return
	}
	
	newNode := &lruNode{
		key:   key,
		value: value,
	}
	
	lru.items[key] = newNode
	lru.addToHead(newNode)
	
	if len(lru.items) > lru.capacity {
		tail := lru.removeTail()
		delete(lru.items, tail.key)
	}
}

// moveToHead moves a node to the head of the list
func (lru *LRUCache) moveToHead(node *lruNode) {
	lru.removeNode(node)
	lru.addToHead(node)
}

// addToHead adds a node to the head of the list
func (lru *LRUCache) addToHead(node *lruNode) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

// removeNode removes a node from the list
func (lru *LRUCache) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// removeTail removes and returns the tail node
func (lru *LRUCache) removeTail() *lruNode {
	lastNode := lru.tail.prev
	lru.removeNode(lastNode)
	return lastNode
}

// NewSkipList creates a new skip list
func NewSkipList(maxLevel int) *SkipList {
	return &SkipList{
		header:   &skipNode{forward: make([]*skipNode, maxLevel)},
		maxLevel: maxLevel,
	}
}

// Insert adds a key-value pair to the skip list
func (sl *SkipList) Insert(key string, value unsafe.Pointer) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	
	update := make([]*skipNode, sl.maxLevel)
	current := sl.header
	
	// Find insertion point
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}
	
	current = current.forward[0]
	
	if current != nil && current.key == key {
		current.value = value // Update existing
		return
	}
	
	// Generate random level
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level; i < newLevel; i++ {
			update[i] = sl.header
		}
		sl.level = newLevel
	}
	
	// Create new node
	newNode := &skipNode{
		key:     key,
		value:   value,
		forward: make([]*skipNode, newLevel),
	}
	
	// Update pointers
	for i := 0; i < newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}
	
	atomic.AddInt64(&sl.size, 1)
}

// Search finds a value by key
func (sl *SkipList) Search(key string) (unsafe.Pointer, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	current := sl.header
	
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
	}
	
	current = current.forward[0]
	
	if current != nil && current.key == key {
		return current.value, true
	}
	
	return nil, false
}

// randomLevel generates a random level for skip list insertion
func (sl *SkipList) randomLevel() int {
	level := 1
	for level < sl.maxLevel && (rand.Int()&1) == 0 {
		level++
	}
	return level
}

// Size returns the number of elements
func (sl *SkipList) Size() int64 {
	return atomic.LoadInt64(&sl.size)
}

// NewConcurrentBitSet creates a new concurrent bit set
func NewConcurrentBitSet(size uint64) *ConcurrentBitSet {
	return &ConcurrentBitSet{
		bits: make([]uint64, (size+63)/64),
		size: size,
	}
}

// Set sets a bit at the given position
func (cbs *ConcurrentBitSet) Set(pos uint64) {
	if pos >= cbs.size {
		return
	}
	
	wordIndex := pos / 64
	bitOffset := pos % 64
	
	cbs.mu.Lock()
	cbs.bits[wordIndex] |= 1 << bitOffset
	cbs.mu.Unlock()
}

// Clear clears a bit at the given position
func (cbs *ConcurrentBitSet) Clear(pos uint64) {
	if pos >= cbs.size {
		return
	}
	
	wordIndex := pos / 64
	bitOffset := pos % 64
	
	cbs.mu.Lock()
	cbs.bits[wordIndex] &^= 1 << bitOffset
	cbs.mu.Unlock()
}

// Test tests if a bit is set at the given position
func (cbs *ConcurrentBitSet) Test(pos uint64) bool {
	if pos >= cbs.size {
		return false
	}
	
	wordIndex := pos / 64
	bitOffset := pos % 64
	
	cbs.mu.RLock()
	result := (cbs.bits[wordIndex] & (1 << bitOffset)) != 0
	cbs.mu.RUnlock()
	
	return result
}

// Count returns the number of set bits
func (cbs *ConcurrentBitSet) Count() uint64 {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()
	
	var count uint64
	for _, word := range cbs.bits {
		count += uint64(popcount(word))
	}
	
	return count
}

// popcount counts the number of set bits in a 64-bit word
func popcount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1 // Clear the lowest set bit
	}
	return count
}