package storage

import (
	"context"
	"fmt"

	"github.com/sealock/core-storage/model"
	"gorm.io/gorm"
)

// GormFileRepository 基于 GORM 的文件仓储实现
type GormFileRepository struct {
	db *gorm.DB
}

// GormBlockRepository 基于 GORM 的块仓储实现
type GormBlockRepository struct {
	db *gorm.DB
}

// GormLibraryRepository 基于 GORM 的库仓储实现
type GormLibraryRepository struct {
	db *gorm.DB
}

// GormLibraryVersionRepository 基于 GORM 的版本仓储实现
type GormLibraryVersionRepository struct {
	db *gorm.DB
}

// NewGormFileRepository 创建 GORM 文件仓储
func NewGormFileRepository(db *gorm.DB) *GormFileRepository {
	return &GormFileRepository{db: db}
}

// NewGormBlockRepository 创建 GORM 块仓储
func NewGormBlockRepository(db *gorm.DB) *GormBlockRepository {
	return &GormBlockRepository{db: db}
}

// NewGormLibraryRepository 创建 GORM 库仓储
func NewGormLibraryRepository(db *gorm.DB) *GormLibraryRepository {
	return &GormLibraryRepository{db: db}
}

// NewGormLibraryVersionRepository 创建 GORM 版本仓储
func NewGormLibraryVersionRepository(db *gorm.DB) *GormLibraryVersionRepository {
	return &GormLibraryVersionRepository{db: db}
}

// CreateFile 创建文件记录
func (r *GormFileRepository) CreateFile(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// GetFileByHash 通过文件 hash 获取文件
func (r *GormFileRepository) GetFileByHash(ctx context.Context, hash string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("file not found: %s", hash)
		}
		return nil, fmt.Errorf("failed to query file: %w", err)
	}
	return &file, nil
}

// UpdateFile 更新文件
func (r *GormFileRepository) UpdateFile(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Save(file).Error; err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	return nil
}

// DeleteFile 删除文件
func (r *GormFileRepository) DeleteFile(ctx context.Context, fileID uint) error {
	err := r.db.WithContext(ctx).Delete(&model.File{}, fileID).Error
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetAllFiles 获取所有文件
func (r *GormFileRepository) GetAllFiles(ctx context.Context) ([]model.File, error) {
	var files []model.File
	err := r.db.WithContext(ctx).Find(&files).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}
	return files, nil
}

// SaveBlockMetadata 保存 Block 的元数据
func (r *GormBlockRepository) SaveBlockMetadata(ctx context.Context, block *model.Block) error {
	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		return fmt.Errorf("failed to save block metadata: %w", err)
	}
	return nil
}

// GetBlockMetadata 获取 Block 元数据
func (r *GormBlockRepository) GetBlockMetadata(ctx context.Context, hash string) (*model.Block, error) {
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
func (r *GormBlockRepository) IncrementRefCount(ctx context.Context, hash string, delta int) error {
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
func (r *GormBlockRepository) ListOrphanBlocks(ctx context.Context) ([]string, error) {
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
func (r *GormBlockRepository) DecrementBlockRefCount(ctx context.Context, hash string) error {
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

// CreateLibrary 创建库
func (r *GormLibraryRepository) CreateLibrary(ctx context.Context, lib *model.Library) error {
	if err := r.db.WithContext(ctx).Create(lib).Error; err != nil {
		return fmt.Errorf("failed to create library: %w", err)
	}
	return nil
}

// GetLibraryByID 获取库
func (r *GormLibraryRepository) GetLibraryByID(ctx context.Context, id uint) (*model.Library, error) {
	var lib model.Library
	if err := r.db.WithContext(ctx).First(&lib, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("library not found: %d", id)
		}
		return nil, fmt.Errorf("failed to query library: %w", err)
	}
	return &lib, nil
}

// ListLibrariesByOwner 列出用户的所有库
func (r *GormLibraryRepository) ListLibrariesByOwner(ctx context.Context, ownerID uint) ([]*model.Library, error) {
	var libs []*model.Library
	if err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&libs).Error; err != nil {
		return nil, fmt.Errorf("failed to list libraries: %w", err)
	}
	return libs, nil
}

// UpdateLibrary 更新库信息
func (r *GormLibraryRepository) UpdateLibrary(ctx context.Context, lib *model.Library) error {
	if err := r.db.WithContext(ctx).Save(lib).Error; err != nil {
		return fmt.Errorf("failed to update library: %w", err)
	}
	return nil
}

// DeleteLibrary 删除库
func (r *GormLibraryRepository) DeleteLibrary(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Library{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete library: %w", err)
	}
	return nil
}

// CreateVersion 创建版本
func (r *GormLibraryVersionRepository) CreateVersion(ctx context.Context, version *model.LibraryVersion) error {
	if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}
	return nil
}

// GetVersionByCommitID 通过 commit ID 获取版本
func (r *GormLibraryVersionRepository) GetVersionByCommitID(ctx context.Context, commitID string) (*model.LibraryVersion, error) {
	var version model.LibraryVersion
	if err := r.db.WithContext(ctx).Where("commit_id = ?", commitID).First(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("version not found: %s", commitID)
		}
		return nil, fmt.Errorf("failed to query version: %w", err)
	}
	return &version, nil
}

// ListVersionsByLibrary 列出库的所有版本
func (r *GormLibraryVersionRepository) ListVersionsByLibrary(ctx context.Context, libraryID uint) ([]*model.LibraryVersion, error) {
	var versions []*model.LibraryVersion
	if err := r.db.WithContext(ctx).
		Where("library_id = ?", libraryID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	return versions, nil
}

// GetLatestVersion 获取库的最新版本
func (r *GormLibraryVersionRepository) GetLatestVersion(ctx context.Context, libraryID uint) (*model.LibraryVersion, error) {
	var version model.LibraryVersion
	if err := r.db.WithContext(ctx).
		Where("library_id = ?", libraryID).
		Order("created_at DESC").
		First(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no version found for library: %d", libraryID)
		}
		return nil, fmt.Errorf("failed to query latest version: %w", err)
	}
	return &version, nil
}