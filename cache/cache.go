// Package cache defines a small key-value cache interface.
package cache

import "context"

var (
	ErrInvalidKey = errStr("invalid cache key")
	ErrNotFound   = errStr("cache key not found")
)

type Store interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
