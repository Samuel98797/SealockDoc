package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBlockCache 使用 Redis 缓存热块以加快访问速度
type RedisBlockCache struct {
	client         *redis.Client
	defaultExpiry  time.Duration
	blockStore     BlockStore // 底层存储
	cacheKeyPrefix string
}

// NewRedisBlockCache 创建 Redis 缓存层
// blockStore: 底层 BlockStore 实现（本地存储等）
// redisAddr: Redis 服务器地址，例如 "localhost:6379"
// defaultExpiry: 缓存过期时间，0 表示永不过期
func NewRedisBlockCache(blockStore BlockStore, redisAddr string, defaultExpiry time.Duration) (*RedisBlockCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // 默认无密码，可配置化
		DB:       0,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	if defaultExpiry == 0 {
		defaultExpiry = 24 * time.Hour // 默认 24 小时
	}

	return &RedisBlockCache{
		client:         client,
		defaultExpiry:  defaultExpiry,
		blockStore:     blockStore,
		cacheKeyPrefix: "block:",
	}, nil
}

// getCacheKey 生成缓存键
func (c *RedisBlockCache) getCacheKey(hash string) string {
	return c.cacheKeyPrefix + hash
}

// Put 存储数据块（缓存 + 底层存储）
func (c *RedisBlockCache) Put(ctx context.Context, data []byte) (hash string, err error) {
	// 委托给底层存储
	hash, err = c.blockStore.Put(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to put block in underlying store: %w", err)
	}

	// 写入 Redis 缓存
	cacheKey := c.getCacheKey(hash)
	if err := c.client.Set(ctx, cacheKey, data, c.defaultExpiry).Err(); err != nil {
		// 缓存失败不应该导致操作失败，记录但继续
		fmt.Printf("failed to cache block %s: %v\n", hash, err)
	}

	return hash, nil
}

// Get 获取数据块（先查缓存，再查底层存储）
func (c *RedisBlockCache) Get(ctx context.Context, hash string) ([]byte, error) {
	cacheKey := c.getCacheKey(hash)

	// 先从 Redis 查询
	val, err := c.client.Get(ctx, cacheKey).Bytes()
	if err == nil {
		return val, nil
	}

	// 缓存未命中，从底层存储获取
	data, err := c.blockStore.Get(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}

	// 写入 Redis 缓存
	if err := c.client.Set(ctx, cacheKey, data, c.defaultExpiry).Err(); err != nil {
		// 缓存失败不应该影响返回
		fmt.Printf("failed to cache block %s: %v\n", hash, err)
	}

	return data, nil
}

// Exists 检查数据块是否存在
func (c *RedisBlockCache) Exists(ctx context.Context, hash string) (bool, error) {
	cacheKey := c.getCacheKey(hash)

	// 先检查缓存中是否存在
	exists, err := c.client.Exists(ctx, cacheKey).Result()
	if err == nil && exists > 0 {
		return true, nil
	}

	// 检查底层存储
	return c.blockStore.Exists(ctx, hash)
}

// Delete 删除数据块（同时删除缓存和底层存储）
func (c *RedisBlockCache) Delete(ctx context.Context, hash string) error {
	// 删除底层存储
	if err := c.blockStore.Delete(ctx, hash); err != nil {
		return err
	}

	// 删除缓存
	cacheKey := c.getCacheKey(hash)
	if err := c.client.Del(ctx, cacheKey).Err(); err != nil {
		// 缓存删除失败不应该导致操作失败
		fmt.Printf("failed to delete cache for block %s: %v\n", hash, err)
	}

	return nil
}

// GetSize 获取数据块大小
func (c *RedisBlockCache) GetSize(ctx context.Context, hash string) (int64, error) {
	return c.blockStore.GetSize(ctx, hash)
}

// InvalidateCache 清除指定块的缓存
func (c *RedisBlockCache) InvalidateCache(ctx context.Context, hash string) error {
	cacheKey := c.getCacheKey(hash)
	if err := c.client.Del(ctx, cacheKey).Err(); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}
	return nil
}

// ClearCache 清除所有块缓存
func (c *RedisBlockCache) ClearCache(ctx context.Context) error {
	iter := c.client.Scan(ctx, 0, c.cacheKeyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
	}
	return iter.Err()
}

// GetCacheStats 获取缓存统计信息
func (c *RedisBlockCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info := c.client.Info(ctx, "stats")
	return map[string]interface{}{
		"info": info.String(),
	}, nil
}

// Close 关闭 Redis 连接
func (c *RedisBlockCache) Close() error {
	return c.client.Close()
}
