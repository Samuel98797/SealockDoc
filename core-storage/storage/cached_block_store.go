package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// cachedBlockStore implements BlockStore with Redis caching layer
type cachedBlockStore struct {
	local        BlockStore
	redisClient  *redis.Client
	expiry       time.Duration
	cacheKeyPrefix string
}

// NewCachedBlockStore creates a new cached block store
func NewCachedBlockStore(local BlockStore, redisClient *redis.Client, expiry time.Duration) BlockStore {
	return &cachedBlockStore{
		local:        local,
		redisClient:  redisClient,
		expiry:       expiry,
		cacheKeyPrefix: "block:",
	}
}

// Put stores a block and caches it
func (c *cachedBlockStore) Put(ctx context.Context, data []byte) (string, error) {
	hash, err := c.local.Put(ctx, data)
	if err != nil {
		return "", err
	}
	
	// Cache the data in Redis
	cacheKey := c.cacheKeyPrefix + hash
	err = c.redisClient.SetEx(ctx, cacheKey, data, c.expiry).Err()
	if err != nil {
		// Log error but continue since this is just a cache
	}
	
	return hash, nil
}

// Get retrieves a block, checking cache first
func (c *cachedBlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
	// Check cache first
	cacheKey := c.cacheKeyPrefix + hash
	cachedData, err := c.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		return cachedData, nil
	}
	
	// Not in cache, get from local store
	data, err := c.local.Get(ctx, hash)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	err = c.redisClient.SetEx(ctx, cacheKey, data, c.expiry).Err()
	if err != nil {
		// Log error but continue since this is just a cache
	}
	
	return data, nil
}

// Exists checks if a block exists
func (c *cachedBlockStore) Exists(ctx context.Context, hash string) (bool, error) {
	// Check cache first
	cacheKey := c.cacheKeyPrefix + hash
	exists, err := c.redisClient.Exists(ctx, cacheKey).Result()
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return true, nil
	}
	
	// Check local store
	return c.local.Exists(ctx, hash)
}

// Delete removes a block from both cache and store
func (c *cachedBlockStore) Delete(ctx context.Context, hash string) error {
	// Remove from cache
	cacheKey := c.cacheKeyPrefix + hash
	err := c.redisClient.Del(ctx, cacheKey).Err()
	if err != nil {
		// Log error but continue
	}
	
	// Remove from local store
	return c.local.Delete(ctx, hash)
}

// GetSize gets the size of a block
func (c *cachedBlockStore) GetSize(ctx context.Context, hash string) (int64, error) {
	// Get from local store
	return c.local.GetSize(ctx, hash)
}