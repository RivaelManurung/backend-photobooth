package services

import (
	"backendphotobooth/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisService(cfg *config.Config) *RedisService {
	addr := cfg.Redis.Addr
	if addr == "" {
		addr = fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis connection failed: %v\n", err)
		return nil
	}

	fmt.Println("✅ Redis connected successfully")

	return &RedisService{
		client: client,
		ctx:    ctx,
	}
}

// Client exposes the underlying Redis client for infrastructure services such as queues.
func (r *RedisService) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

func (r *RedisService) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis is not configured")
	}
	return r.client.Ping(ctx).Err()
}

// Set stores a value with expiration
func (r *RedisService) Set(key string, value interface{}, expiration time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, jsonValue, expiration).Err()
}

// Get retrieves a value
func (r *RedisService) Get(key string, dest interface{}) error {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Delete removes a key
func (r *RedisService) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Exists checks if key exists
func (r *RedisService) Exists(key string) bool {
	result, err := r.client.Exists(r.ctx, key).Result()
	return err == nil && result > 0
}

// Increment increments a counter
func (r *RedisService) Increment(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

// Decrement decrements a counter
func (r *RedisService) Decrement(key string) (int64, error) {
	return r.client.Decr(r.ctx, key).Result()
}

// SetWithTTL sets a value with TTL in seconds
func (r *RedisService) SetWithTTL(key string, value interface{}, ttlSeconds int) error {
	return r.Set(key, value, time.Duration(ttlSeconds)*time.Second)
}

// GetOrSet gets value or sets it if not exists
func (r *RedisService) GetOrSet(key string, dest interface{}, setter func() (interface{}, error), expiration time.Duration) error {
	// Try to get from cache
	err := r.Get(key, dest)
	if err == nil {
		return nil // Cache hit
	}

	// Cache miss, get from setter
	value, err := setter()
	if err != nil {
		return err
	}

	// Store in cache
	if err := r.Set(key, value, expiration); err != nil {
		return err
	}

	// Marshal to dest
	jsonValue, _ := json.Marshal(value)
	return json.Unmarshal(jsonValue, dest)
}

// InvalidatePattern deletes all keys matching pattern
func (r *RedisService) InvalidatePattern(pattern string) error {
	iter := r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
	for iter.Next(r.ctx) {
		if err := r.client.Del(r.ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// SetHash stores a hash
func (r *RedisService) SetHash(key string, field string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.HSet(r.ctx, key, field, jsonValue).Err()
}

// GetHash retrieves a hash field
func (r *RedisService) GetHash(key string, field string, dest interface{}) error {
	val, err := r.client.HGet(r.ctx, key, field).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// GetAllHash retrieves all hash fields
func (r *RedisService) GetAllHash(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

// AddToSet adds member to set
func (r *RedisService) AddToSet(key string, members ...interface{}) error {
	return r.client.SAdd(r.ctx, key, members...).Err()
}

// GetSet retrieves all set members
func (r *RedisService) GetSet(key string) ([]string, error) {
	return r.client.SMembers(r.ctx, key).Result()
}

// RemoveFromSet removes member from set
func (r *RedisService) RemoveFromSet(key string, members ...interface{}) error {
	return r.client.SRem(r.ctx, key, members...).Err()
}

// IsInSet checks if member is in set
func (r *RedisService) IsInSet(key string, member interface{}) (bool, error) {
	return r.client.SIsMember(r.ctx, key, member).Result()
}

// PushToList pushes value to list
func (r *RedisService) PushToList(key string, values ...interface{}) error {
	return r.client.RPush(r.ctx, key, values...).Err()
}

// PopFromList pops value from list
func (r *RedisService) PopFromList(key string) (string, error) {
	return r.client.LPop(r.ctx, key).Result()
}

// GetList retrieves list range
func (r *RedisService) GetList(key string, start, stop int64) ([]string, error) {
	return r.client.LRange(r.ctx, key, start, stop).Result()
}

// SetExpire sets expiration on existing key
func (r *RedisService) SetExpire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// GetTTL gets remaining TTL
func (r *RedisService) GetTTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

// FlushAll clears all cache (use with caution!)
func (r *RedisService) FlushAll() error {
	return r.client.FlushAll(r.ctx).Err()
}

// Close closes redis connection
func (r *RedisService) Close() error {
	return r.client.Close()
}

// CacheService provides high-level caching operations
type CacheService struct {
	redis *RedisService
}

func NewCacheService(redis *RedisService) *CacheService {
	return &CacheService{redis: redis}
}

// CacheTemplates caches templates list
func (c *CacheService) CacheTemplates(templates interface{}) error {
	return c.redis.Set("templates:all", templates, 10*time.Minute)
}

// GetCachedTemplates retrieves cached templates
func (c *CacheService) GetCachedTemplates(dest interface{}) error {
	return c.redis.Get("templates:all", dest)
}

// InvalidateTemplatesCache invalidates templates cache
func (c *CacheService) InvalidateTemplatesCache() error {
	return c.redis.InvalidatePattern("templates:*")
}

// CacheUser caches user data
func (c *CacheService) CacheUser(userID uint, user interface{}) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.redis.Set(key, user, 30*time.Minute)
}

// GetCachedUser retrieves cached user
func (c *CacheService) GetCachedUser(userID uint, dest interface{}) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.redis.Get(key, dest)
}

// InvalidateUserCache invalidates user cache
func (c *CacheService) InvalidateUserCache(userID uint) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.redis.Delete(key)
}

// RateLimitCheck checks rate limit for user
func (c *CacheService) RateLimitCheck(userID uint, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:user:%d", userID)

	count, err := c.redis.Increment(key)
	if err != nil {
		return false, err
	}

	if count == 1 {
		// First request, set expiration
		c.redis.SetExpire(key, window)
	}

	return count <= int64(limit), nil
}

// SessionStore stores session data
func (c *CacheService) SessionStore(sessionID string, data interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Set(key, data, expiration)
}

// SessionGet retrieves session data
func (c *CacheService) SessionGet(sessionID string, dest interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Get(key, dest)
}

// SessionDelete deletes session
func (c *CacheService) SessionDelete(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.redis.Delete(key)
}

// CacheStats caches statistics
func (c *CacheService) CacheStats(statsType string, data interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("stats:%s", statsType)
	return c.redis.Set(key, data, expiration)
}

// GetCachedStats retrieves cached statistics
func (c *CacheService) GetCachedStats(statsType string, dest interface{}) error {
	key := fmt.Sprintf("stats:%s", statsType)
	return c.redis.Get(key, dest)
}
