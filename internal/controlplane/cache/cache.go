package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache is a local in-memory cache wrapper around go-cache
type Cache struct {
	inner *gocache.Cache
}

// New creates a new Cache with the specified default expiration and cleanup interval.
// If defaultExpiration is 0, items never expire by default.
// If cleanupInterval is 0, expired items are not cleaned up automatically.
func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		inner: gocache.New(defaultExpiration, cleanupInterval),
	}
}

// Get retrieves an item from the cache by key.
// Returns the value and true if found, nil and false otherwise.
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.inner.Get(key)
}

// Set adds or updates an item in the cache with the specified key and value.
// The ttl parameter specifies the expiration time. If ttl is 0, the default
// expiration time is used. If ttl is negative, the item never expires.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.inner.Set(key, value, ttl)
}

// Delete removes an item from the cache by key.
func (c *Cache) Delete(key string) {
	c.inner.Delete(key)
}

// Flush removes all items from the cache.
func (c *Cache) Flush() {
	c.inner.Flush()
}

// ItemCount returns the number of items in the cache.
func (c *Cache) ItemCount() int {
	return c.inner.ItemCount()
}
