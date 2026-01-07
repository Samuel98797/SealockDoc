package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// LocalBlockStore 实现 BlockStore 接口
// 用于开发环境的本地磁盘存储
type LocalBlockStore struct {
	blocks map[string][]byte
	mu     sync.RWMutex
}

// NewLocalBlockStore 创建本地存储实例
func NewLocalBlockStore() *LocalBlockStore {
	return &LocalBlockStore{
		blocks: make(map[string][]byte),
	}
}

// Put 存储数据块
func (s *LocalBlockStore) Put(ctx context.Context, data []byte) (hash string, err error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}

	// 计算 SHA-256 哈希
	hashSum := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hashSum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	// 存储数据
	s.blocks[hashHex] = make([]byte, len(data))
	copy(s.blocks[hashHex], data)

	return hashHex, nil
}

// Get 获取数据块
func (s *LocalBlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.blocks[hash]
	if !exists {
		return nil, fmt.Errorf("block not found: %s", hash)
	}

	// 返回副本（避免外部修改）
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Exists 检查数据块是否存在
func (s *LocalBlockStore) Exists(ctx context.Context, hash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.blocks[hash]
	return exists, nil
}

// Delete 删除数据块
func (s *LocalBlockStore) Delete(ctx context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.blocks[hash]; !exists {
		return fmt.Errorf("block not found: %s", hash)
	}

	delete(s.blocks, hash)
	return nil
}

// GetSize 获取数据块大小
func (s *LocalBlockStore) GetSize(ctx context.Context, hash string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.blocks[hash]
	if !exists {
		return 0, fmt.Errorf("block not found: %s", hash)
	}

	return int64(len(data)), nil
}

// Stats 返回存储统计信息（开发辅助）
func (s *LocalBlockStore) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalSize := int64(0)
	for _, data := range s.blocks {
		totalSize += int64(len(data))
	}

	return map[string]interface{}{
		"block_count": len(s.blocks),
		"total_size":  totalSize,
	}
}
