package memory

import "context"

type entry struct {
	value     []byte
	createdAt int64
}

type memoryCache struct {
	items map[string]entry
	ttl   int64
}

func (c *Store) deleteExpired(now int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if expired(item.createdAt, c.ttl, now) {
			delete(c.items, key)
		}
	}
}

func expired(createdAt int64, ttl int64, now int64) bool {
	return now-createdAt >= ttl
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	return ctx.Err()
}
