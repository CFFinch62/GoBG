package engine

import (
	"sync"

	"github.com/yourusername/bgengine/internal/positionid"
)

// Cache constants
const (
	DefaultCacheSize = 1 << 20 // 1M entries (~48MB with 6 floats per entry)
	CacheHit         = ^uint32(0)
)

// CacheEntry stores a cached evaluation result
type CacheEntry struct {
	Key         positionid.PositionKey // Position key (7 uint32s = 28 bytes)
	EvalContext int32                  // Evaluation context (plies, cube info, etc.)
	Output      [6]float32             // 5 probabilities + 1 cubeful equity
}

// EvalCache is a thread-safe position evaluation cache
// Uses a two-way associative cache with MurmurHash3-based indexing
type EvalCache struct {
	entries  []cacheNode
	size     uint32
	hashMask uint32

	// Statistics
	lookups uint64
	hits    uint64
	adds    uint64

	mu sync.RWMutex
}

// cacheNode holds primary and secondary entries for two-way associative cache
type cacheNode struct {
	primary   CacheEntry
	secondary CacheEntry
}

// NewEvalCache creates a new evaluation cache with the given size
// Size will be adjusted to the nearest power of 2
func NewEvalCache(size uint32) *EvalCache {
	// Adjust size to power of 2
	if size > 1<<31 {
		size = 1 << 31
	}

	// Find smallest power of 2 >= size
	p := uint32(1)
	for p < size {
		p <<= 1
	}
	size = p

	cache := &EvalCache{
		entries:  make([]cacheNode, size/2),
		size:     size,
		hashMask: (size / 2) - 1,
	}

	cache.Flush()
	return cache
}

// Flush clears all entries from the cache
func (c *EvalCache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	invalidKey := positionid.PositionKey{Data: [7]uint32{^uint32(0), 0, 0, 0, 0, 0, 0}}
	for i := range c.entries {
		c.entries[i].primary.Key = invalidKey
		c.entries[i].secondary.Key = invalidKey
	}
	c.lookups = 0
	c.hits = 0
	c.adds = 0
}

// hash computes the hash key for a cache entry using MurmurHash3-style mixing
func (c *EvalCache) hash(key positionid.PositionKey, evalContext int32) uint32 {
	// MurmurHash3 constants
	const c1 = 0xcc9e2d51
	const c2 = 0x1b873593

	h := uint32(0)

	// Mix in position key data
	for _, k := range key.Data {
		k *= c1
		k = (k << 15) | (k >> 17)
		k *= c2

		h ^= k
		h = (h << 13) | (h >> 19)
		h = h*5 + 0xe6546b64
	}

	// Mix in evaluation context
	k := uint32(evalContext)
	k *= c1
	k = (k << 15) | (k >> 17)
	k *= c2
	h ^= k

	// Finalization
	h ^= 32
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16

	return h & c.hashMask
}

// keysEqual compares two position keys for equality
func keysEqual(a, b positionid.PositionKey) bool {
	return a.Data == b.Data
}

// Lookup checks if a position is in the cache
// Returns CacheHit if found (outputs filled), otherwise returns hash slot for Add
func (c *EvalCache) Lookup(key positionid.PositionKey, evalContext int32, output []float32) uint32 {
	slot := c.hash(key, evalContext)

	c.mu.RLock()
	defer c.mu.RUnlock()

	c.lookups++

	node := &c.entries[slot]

	// Check primary slot
	if keysEqual(node.primary.Key, key) && node.primary.EvalContext == evalContext {
		copy(output, node.primary.Output[:5])
		c.hits++
		return CacheHit
	}

	// Check secondary slot
	if keysEqual(node.secondary.Key, key) && node.secondary.EvalContext == evalContext {
		// Promote to primary (will be done in Add if we miss)
		copy(output, node.secondary.Output[:5])
		c.hits++
		return CacheHit
	}

	return slot
}

// Add adds an evaluation result to the cache
// slot should be the value returned by a previous Lookup miss
func (c *EvalCache) Add(key positionid.PositionKey, evalContext int32, output []float32, slot uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node := &c.entries[slot]

	// Move primary to secondary, add new as primary
	node.secondary = node.primary
	node.primary = CacheEntry{
		Key:         key,
		EvalContext: evalContext,
	}
	copy(node.primary.Output[:], output[:5])

	c.adds++
}

// Stats returns cache statistics
func (c *EvalCache) Stats() (lookups, hits, adds uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lookups, c.hits, c.adds
}

// HitRate returns the cache hit rate as a percentage
func (c *EvalCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lookups == 0 {
		return 0
	}
	return float64(c.hits) / float64(c.lookups) * 100
}

// MakeEvalContext creates an evaluation context key from evaluation parameters
// This encodes plies, cube info, etc. into a single int32 for cache keying
func MakeEvalContext(plies int, cubeful bool, cubeOwner int, cubeValue int) int32 {
	// Bit layout:
	// Bits 0-3: plies (0-15)
	// Bit 4: cubeful
	// Bits 5-6: cube owner (-1=centered, 0=player0, 1=player1) + 1
	// Bits 7-10: log2(cubeValue)

	ctx := int32(plies & 0xF)
	if cubeful {
		ctx |= 1 << 4
	}
	ctx |= int32((cubeOwner+1)&0x3) << 5

	// log2 of cube value
	logCube := 0
	for v := cubeValue; v > 1; v >>= 1 {
		logCube++
	}
	ctx |= int32(logCube&0xF) << 7

	return ctx
}
