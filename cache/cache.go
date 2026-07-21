// Package cache provides a namespaced key-value cache interface.
package cache

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidKey error = errors.New("invalid cache key")
	ErrNotFound   error = errors.New("cache key not found")
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

func cacheKey(namespace string, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrInvalidKey
	}

	if namespace == "" {
		return key, nil
	}

	return namespace + key, nil
}

func namespacePrefix(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return ""
	}

	return strings.TrimRight(namespace, ":") + ":"
}
