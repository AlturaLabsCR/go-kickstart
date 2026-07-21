package cache

import (
	"context"
	"sync"
	"time"
)

const DefaultMemoryMaxEntries = 1000

type MemoryOptions struct {
	MaxEntries int
}

type memoryCache struct {
	namespace string

	maxEntries int

	mu     sync.RWMutex
	values map[string]memoryValue
}

type memoryValue struct {
	value     string
	expiresAt time.Time
}

func NewMemoryCache(namespace string, options ...MemoryOptions) Cache {
	opts := MemoryOptions{MaxEntries: DefaultMemoryMaxEntries}
	if len(options) > 0 {
		if options[0].MaxEntries > 0 {
			opts.MaxEntries = options[0].MaxEntries
		}
	}

	return &memoryCache{
		namespace:  namespacePrefix(namespace),
		maxEntries: opts.MaxEntries,
		values:     map[string]memoryValue{},
	}
}

func (c *memoryCache) Get(_ context.Context, key string) (string, error) {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return "", err
	}

	now := time.Now()

	c.mu.RLock()
	item, ok := c.values[key]
	c.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}

	if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
		c.mu.Lock()
		if current, ok := c.values[key]; ok && current == item {
			c.deleteLocked(key)
		}
		c.mu.Unlock()
		return "", ErrNotFound
	}

	return item.value, nil
}

func (c *memoryCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return err
	}

	item := memoryValue{value: value}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.values[key] = item
	c.evictLocked(time.Now())
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Delete(_ context.Context, key string) error {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.deleteLocked(key)
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Close() error {
	c.mu.Lock()
	clear(c.values)
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) evictLocked(now time.Time) {
	for key, item := range c.values {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			c.deleteLocked(key)
		}
	}

	for len(c.values) > c.maxEntries {
		for key := range c.values {
			c.deleteLocked(key)
			break
		}
	}
}

func (c *memoryCache) deleteLocked(key string) {
	delete(c.values, key)
}
