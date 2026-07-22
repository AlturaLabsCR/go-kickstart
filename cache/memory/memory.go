// Package memory implements an in-memory key-value cache.
package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"app/cache"
)

const defaultTTL = 20 * time.Second

type OptFunc func(*memoryCache)

type Store struct {
	mu sync.RWMutex
	memoryCache
}

func New(opts ...OptFunc) cache.Store {
	c := &Store{
		memoryCache: memoryCache{
			items: make(map[string]entry),
			ttl:   defaultTTL.Nanoseconds(),
		},
	}

	for _, opt := range opts {
		opt(&c.memoryCache)
	}

	return c
}

func WithTTL(ttl time.Duration) OptFunc {
	return func(c *memoryCache) {
		if ttl > 0 {
			c.ttl = ttl.Nanoseconds()
		}
	}
}

func (c *Store) Set(ctx context.Context, key string, value []byte) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return cache.ErrInvalidKey
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = entry{
		value:     append([]byte(nil), value...),
		createdAt: time.Now().UnixNano(),
	}

	return nil
}

func (c *Store) Get(ctx context.Context, key string) ([]byte, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(key) == "" {
		return nil, cache.ErrInvalidKey
	}

	c.mu.RLock()
	item, ok := c.items[key]
	ttl := c.ttl
	now := time.Now().UnixNano()
	c.mu.RUnlock()

	if !ok {
		return nil, cache.ErrNotFound
	}
	if expired(item.createdAt, ttl, now) {
		c.deleteExpired(now)
		return nil, cache.ErrNotFound
	}

	return append([]byte(nil), item.value...), nil
}

func (c *Store) Delete(ctx context.Context, key string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return cache.ErrInvalidKey
	}

	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()

	return nil
}
