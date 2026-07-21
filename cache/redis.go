package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOptions struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type redisCache struct {
	namespace string
	client    *redis.Client
}

func NewRedisCache(namespace string, opts RedisOptions) (Cache, error) {
	if opts.Addr == "" {
		return nil, errors.New("redis address is required")
	}

	return &redisCache{
		namespace: namespacePrefix(namespace),
		client: redis.NewClient(&redis.Options{
			Addr:     opts.Addr,
			Username: opts.Username,
			Password: opts.Password,
			DB:       opts.DB,
		}),
	}, nil
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return "", err
	}

	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

func (c *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	key, err := cacheKey(c.namespace, key)
	if err != nil {
		return err
	}

	return c.client.Del(ctx, key).Err()
}

func (c *redisCache) Close() error {
	return c.client.Close()
}
