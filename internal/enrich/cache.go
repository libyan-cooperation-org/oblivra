package enrich

import (
	"sync"
	"time"
)

// cacheEntry holds a cached enrichment result with a TTL.
type cacheEntry struct {
	location string
	asn      string
	country  string
	cachedAt time.Time
}

// EnrichmentCache is a bounded LRU-style cache for enrichment lookups.
// It eliminates repeated GeoIP/ASN lookups for the same IPs, which can
// account for millions of redundant disk reads per hour under high EPS.
//
// Design:
//   - RWMutex: concurrent reads don't block each other
//   - Max capacity evicts oldest entries to stay bounded
//   - TTL: entries expire after ttl duration (default 10 min)
//   - Zero allocation on cache hit — pointer returned directly
type EnrichmentCache struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	order    []string // insertion order for LRU eviction
	maxSize  int
	ttl      time.Duration
	hits     uint64
	misses   uint64
}

// NewEnrichmentCache creates a bounded enrichment cache.
// maxSize: max number of IPs to cache (default 50_000)
// ttl: how long each entry lives (default 10 min)
func NewEnrichmentCache(maxSize int, ttl time.Duration) *EnrichmentCache {
	if maxSize <= 0 {
		maxSize = 50_000
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &EnrichmentCache{
		entries: make(map[string]*cacheEntry, maxSize),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get returns cached enrichment for an IP, or nil if not found / expired.
func (c *EnrichmentCache) Get(ip string) (location, asn, country string, ok bool) {
	c.mu.RLock()
	entry, exists := c.entries[ip]
	c.mu.RUnlock()

	if !exists {
		c.misses++
		return "", "", "", false
	}
	if time.Since(entry.cachedAt) > c.ttl {
		// Expired — treat as miss; background eviction will clean it
		c.misses++
		return "", "", "", false
	}
	c.hits++
	return entry.location, entry.asn, entry.country, true
}

// Set stores enrichment data for an IP.
func (c *EnrichmentCache) Set(ip, location, asn, country string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity
	if len(c.entries) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[ip] = &cacheEntry{
		location: location,
		asn:      asn,
		country:  country,
		cachedAt: time.Now(),
	}
	c.order = append(c.order, ip)
}

// Stats returns cache hit/miss counters for the diagnostics panel.
func (c *EnrichmentCache) Stats() (hits, misses uint64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.entries)
}

// Purge removes all expired entries. Safe to call from a background goroutine.
func (c *EnrichmentCache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	newOrder := c.order[:0]
	for _, ip := range c.order {
		if e, ok := c.entries[ip]; ok && now.Sub(e.cachedAt) <= c.ttl {
			newOrder = append(newOrder, ip)
		} else {
			delete(c.entries, ip)
		}
	}
	c.order = newOrder
}
