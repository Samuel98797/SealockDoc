package storage

import (
	"context"
	"github.com/sealock/core-storage/model"
	"gorm.io/gorm"
)

type snapshotRepository struct {
	db *gorm.DB
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(db *gorm.DB) SnapshotRepository {
	return &snapshotRepository{db: db}
}

func (r *snapshotRepository) CreateSnapshot(ctx context.Context, snapshot *model.Snapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

func (r *snapshotRepository) GetSnapshotByID(ctx context.Context, id uint) (*model.Snapshot, error) {
	var snapshot model.Snapshot
	if err := r.db.WithContext(ctx).First(&snapshot, id).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *snapshotRepository) GetSnapshotByUUID(ctx context.Context, uuid string) (*model.Snapshot, error) {
	var snapshot model.Snapshot
	if err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *snapshotRepository) ListSnapshots(ctx context.Context, limit, offset int) ([]model.Snapshot, error) {
	var snapshots []model.Snapshot
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (r *snapshotRepository) ListSnapshotFiles(ctx context.Context, snapshotID uint, limit, offset int) ([]model.SnapshotFile, error) {
	var snapshotFiles []model.SnapshotFile
	err := r.db.WithContext(ctx).
		Where("snapshot_id = ?", snapshotID).
		Limit(limit).
		Offset(offset).
		Find(&snapshotFiles).Error
	if err != nil {
		return nil, err
	}
	return snapshotFiles, nil
}

func (r *snapshotRepository) CreateSnapshotFile(ctx context.Context, snapshotFile *model.SnapshotFile) error {
	return r.db.WithContext(ctx).Create(snapshotFile).Error
}