// Package cache defines a small key-value cache interface.
package cache

import (
	"context"
	"errors"
)

var (
	ErrInvalidKey = errors.New("invalid cache key")
	ErrNotFound   = errors.New("cache key not found")
)

type Store interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
