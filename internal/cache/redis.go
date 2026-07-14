package cache

import (
	"context"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func newRedisCache(cfg config.CacheTierConfig) (Cache, error) {
	opt := &redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       0,
	}
	if opt.Addr == "" {
		opt.Addr = "localhost:6379"
	}
	client := redis.NewClient(opt)
	return &redisCache{client: client}, nil
}

func (r *redisCache) Get(ctx context.Context, key string) (*Item, bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	item, err := DeserializeItem(data)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

func (r *redisCache) Set(ctx context.Context, key string, item *Item, ttl time.Duration) error {
	data, err := SerializeItem(item)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *redisCache) Close() error {
	return r.client.Close()
}
