package storage

import (
	"context"
	"fmt"

	"github.com/sealock/core-storage/model"
	"gorm.io/gorm"
)

// fileRepository implements FileRepository interface
type fileRepository struct {
	db *gorm.DB
}

// NewFileRepository creates a new GORM-based file repository implementing the FileRepository interface
func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

// CreateFile creates a file record
func (r *fileRepository) CreateFile(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// GetFileByHash retrieves a file by its hash
func (r *fileRepository) GetFileByHash(ctx context.Context, hash string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("file not found: %s", hash)
		}
		return nil, fmt.Errorf("failed to query file: %w", err)
	}
	return &file, nil
}

// UpdateFile updates a file record
func (r *fileRepository) UpdateFile(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Save(file).Error; err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	return nil
}

// DeleteFile deletes a file by ID
func (r *fileRepository) DeleteFile(ctx context.Context, fileID uint) error {
	err := r.db.WithContext(ctx).Delete(&model.File{}, fileID).Error
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetAllFiles 获取所有文件
func (r *fileRepository) GetAllFiles(ctx context.Context) ([]model.File, error) {
	var files []model.File
	err := r.db.WithContext(ctx).Find(&files).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}
	return files, nil
}