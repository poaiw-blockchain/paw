package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_cache_hits_total",
		Help: "Total number of cache hits",
	})

	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_cache_misses_total",
		Help: "Total number of cache misses",
	})

	cacheErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_cache_errors_total",
		Help: "Total number of cache errors",
	})

	cacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_cache_size_bytes",
		Help: "Current cache size in bytes",
	})
)

// RedisCache implements a Redis-based cache
type RedisCache struct {
	client *redis.Client
	prefix string
}

// Config holds Redis cache configuration
type Config struct {
	Address  string
	Password string
	DB       int
	Prefix   string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg Config) (*RedisCache, error) {
	if cfg.Address == "" {
		cfg.Address = "localhost:6379"
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "explorer:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		prefix: cfg.Prefix,
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key

	val, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		cacheMisses.Inc()
		return nil, ErrCacheMiss
	}
	if err != nil {
		cacheErrors.Inc()
		return nil, fmt.Errorf("cache get error: %w", err)
	}

	cacheHits.Inc()
	return val, nil
}

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.prefix + key

	if err := c.client.Set(ctx, fullKey, value, ttl).Err(); err != nil {
		cacheErrors.Inc()
		return fmt.Errorf("cache set error: %w", err)
	}

	cacheSize.Add(float64(len(value)))
	return nil
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		cacheErrors.Inc()
		return fmt.Errorf("cache delete error: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key

	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		cacheErrors.Inc()
		return false, fmt.Errorf("cache exists error: %w", err)
	}

	return count > 0, nil
}

// Increment increments a counter in cache
func (c *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.Incr(ctx, fullKey).Result()
	if err != nil {
		cacheErrors.Inc()
		return 0, fmt.Errorf("cache increment error: %w", err)
	}

	return val, nil
}

// IncrementBy increments a counter by a specific amount
func (c *RedisCache) IncrementBy(ctx context.Context, key string, amount int64) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.IncrBy(ctx, fullKey, amount).Result()
	if err != nil {
		cacheErrors.Inc()
		return 0, fmt.Errorf("cache increment by error: %w", err)
	}

	return val, nil
}

// SetWithExpiry sets a value with expiration
func (c *RedisCache) SetWithExpiry(ctx context.Context, key string, value []byte, expiry time.Duration) error {
	return c.Set(ctx, key, value, expiry)
}

// GetJSON retrieves and unmarshals JSON from cache
func (c *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

// SetJSON marshals and stores JSON in cache
func (c *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return c.Set(ctx, key, data, ttl)
}

// Ping checks cache connection
func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache ping error: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// GetStats returns cache statistics
func (c *RedisCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB size: %w", err)
	}

	stats := map[string]interface{}{
		"info":    info,
		"db_size": dbSize,
	}

	return stats, nil
}

// ClearAll clears all keys with the prefix (use with caution!)
func (c *RedisCache) ClearAll(ctx context.Context) error {
	iter := c.client.Scan(ctx, 0, c.prefix+"*", 0).Iterator()

	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("delete error: %w", err)
		}
	}

	return nil
}

// SetNX sets a value only if the key doesn't exist (SET if Not eXists)
func (c *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	fullKey := c.prefix + key

	success, err := c.client.SetNX(ctx, fullKey, value, ttl).Result()
	if err != nil {
		cacheErrors.Inc()
		return false, fmt.Errorf("cache setnx error: %w", err)
	}

	return success, nil
}

// GetMulti retrieves multiple values from cache
func (c *RedisCache) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	vals, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		cacheErrors.Inc()
		return nil, fmt.Errorf("cache mget error: %w", err)
	}

	result := make(map[string][]byte)
	for i, val := range vals {
		if val != nil {
			if str, ok := val.(string); ok {
				result[keys[i]] = []byte(str)
			}
		}
	}

	return result, nil
}

// SetMulti stores multiple values in cache
func (c *RedisCache) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	pipe := c.client.Pipeline()

	for key, value := range items {
		fullKey := c.prefix + key
		pipe.Set(ctx, fullKey, value, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		cacheErrors.Inc()
		return fmt.Errorf("cache pipeline error: %w", err)
	}

	return nil
}
