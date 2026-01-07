package storage

import (
	"context"
	"fmt"

	"github.com/sealock/core-storage/model"
	"gorm.io/gorm"
)

// blockRepository implements BlockRepository interface
type blockRepository struct {
	db *gorm.DB
}

// NewBlockRepository creates a new GORM-based block repository implementing the BlockRepository interface
func NewBlockRepository(db *gorm.DB) BlockRepository {
	return &blockRepository{db: db}
}

// SaveBlockMetadata 保存 Block 的元数据
func (r *blockRepository) SaveBlockMetadata(ctx context.Context, block *model.Block) error {
	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		return fmt.Errorf("failed to save block metadata: %w", err)
	}
	return nil
}

// GetBlockMetadata 获取 Block 元数据
func (r *blockRepository) GetBlockMetadata(ctx context.Context, hash string) (*model.Block, error) {
	var block model.Block
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&block).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("block not found: %s", hash)
		}
		return nil, fmt.Errorf("failed to query block: %w", err)
	}
	return &block, nil
}

// IncrementRefCount 增加引用计数
func (r *blockRepository) IncrementRefCount(ctx context.Context, hash string, delta int) error {
	var block model.Block
	err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&block).Error
	if err != nil {
		return fmt.Errorf("failed to find block: %w", err)
	}

	block.RefCount += delta
	err = r.db.WithContext(ctx).Save(&block).Error
	if err != nil {
		return fmt.Errorf("failed to increment ref count: %w", err)
	}

	return nil
}

// ListOrphanBlocks 列出引用计数为 0 的 Block
func (r *blockRepository) ListOrphanBlocks(ctx context.Context) ([]string, error) {
	var blocks []model.Block
	err := r.db.WithContext(ctx).Where("ref_count = 0").Find(&blocks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list orphan blocks: %w", err)
	}

	hashes := make([]string, len(blocks))
	for i, block := range blocks {
		hashes[i] = block.Hash
	}

	return hashes, nil
}

// DecrementBlockRefCount 减少块的引用计数
func (r *blockRepository) DecrementBlockRefCount(ctx context.Context, hash string) error {
	var block model.Block
	err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&block).Error
	if err != nil {
		return fmt.Errorf("failed to find block: %w", err)
	}

	if block.RefCount > 0 {
		block.RefCount--
		err = r.db.WithContext(ctx).Save(&block).Error
		if err != nil {
			return fmt.Errorf("failed to decrement ref count: %w", err)
		}
	}
	return nil
}