package memory

import (
	"container/list"
	"context"
	"time"
)

type entry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

type memoryCache struct {
	ttl        time.Duration
	maxEntries int
	maxBytes   int
}

// All helpers below run with mu held. Fixed TTLs keep write order and expiry
// order identical, so cleanup only visits expired entries at the front.
func (c *Store) remove(element *list.Element) {
	item := element.Value.(*entry)
	delete(c.items, item.key)
	c.bytes -= len(item.key) + len(item.value)
	c.order.Remove(element)
}

func (c *Store) deleteExpired(now time.Time) {
	for element := c.order.Front(); element != nil; element = c.order.Front() {
		if now.Before(element.Value.(*entry).expiresAt) {
			return
		}
		c.remove(element)
	}
}

func (c *Store) clean(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(c.done)
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			c.deleteExpired(time.Now())
			c.mu.Unlock()
		}
	}
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
