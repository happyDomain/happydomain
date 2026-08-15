// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package favicon

import (
	"sync"
	"time"
)

const (
	// maxCacheEntries and maxCacheBytes bound memCache. The domain endpoint is
	// unauthenticated and its key is whatever the caller asked for, so both the
	// number of entries and their weight have to be capped: entries alone would
	// leave the footprint at maxCacheEntries times maxIconSize.
	maxCacheEntries = 1024
	maxCacheBytes   = 32 << 20 // 32MB
)

type cacheEntry struct {
	bytes       []byte
	contentType string
	err         error
	expiresAt   time.Time
}

// cache stores fetch results, keyed by the URL fetched. It is its own
// interface, rather than a map field on FaviconService, because the bounds
// and eviction policy below are a service-level decision, not part of what
// fetching an icon means: a caller embedding this package with its own
// storage (a shared cache across instances, say) can supply one instead of
// memCache.
type cache interface {
	lookup(key string) (*cacheEntry, bool)
	store(key string, entry *cacheEntry)
}

// memCache is the default cache: an in-process map bounded in both entries
// and bytes, evicted wholesale rather than by LRU when it fills. An LRU is
// more bookkeeping than a favicon cache is worth.
type memCache struct {
	mu    sync.Mutex
	byKey map[string]*cacheEntry
	bytes int
}

func newMemCache() *memCache {
	return &memCache{byKey: map[string]*cacheEntry{}}
}

func (c *memCache) lookup(key string) (*cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.byKey[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.dropLocked(key)
		return nil, false
	}

	return entry, true
}

func (c *memCache) store(key string, entry *cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dropLocked(key)

	if c.overLimitsLocked(len(entry.bytes)) {
		c.evictLocked(len(entry.bytes))
	}

	c.byKey[key] = entry
	c.bytes += len(entry.bytes)
}

func (c *memCache) dropLocked(key string) {
	if entry, ok := c.byKey[key]; ok {
		delete(c.byKey, key)
		c.bytes -= len(entry.bytes)
	}
}

// overLimitsLocked reports whether adding extraBytes would push the cache past
// either the entry or byte limit.
func (c *memCache) overLimitsLocked(extraBytes int) bool {
	return len(c.byKey) >= maxCacheEntries || c.bytes+extraBytes > maxCacheBytes
}

// evictLocked drops the expired entries and, if that didn't bring the cache
// back under its limits for the incomingBytes about to be stored, the whole
// map.
func (c *memCache) evictLocked(incomingBytes int) {
	now := time.Now()

	for key, entry := range c.byKey {
		if now.After(entry.expiresAt) {
			c.dropLocked(key)
		}
	}

	if c.overLimitsLocked(incomingBytes) {
		clear(c.byKey)
		c.bytes = 0
	}
}
