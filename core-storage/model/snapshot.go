package model

import (
	"time"
)

// Snapshot represents a point-in-time view of the file system
type Snapshot struct {
	ID          uint      `gorm:"primaryKey"`
	UUID        string    `gorm:"uniqueIndex;type:varchar(36)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	Name        string    `gorm:"type:varchar(255)"`
	Description string    `gorm:"type:text"`
	ParentID    *uint     `gorm:"index"`
	RootHash    string    `gorm:"type:varchar(64)"`
	FileCount   int
	Size        int64
}

// SnapshotFile represents a file in a snapshot
type SnapshotFile struct {
	ID         uint      `gorm:"primaryKey"`
	SnapshotID uint      `gorm:"index"`
	FileID     uint      `gorm:"index"`
	FileName   string    `gorm:"type:varchar(255);index:idx_snapshot_file_name"`
	FileHash   string    `gorm:"type:varchar(64)"`
	Status     string    `gorm:"type:varchar(20)"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// SnapshotDiff represents the differences between two snapshots
type SnapshotDiff struct {
	Added    []SnapshotFile
	Removed  []SnapshotFile
	Modified []SnapshotFile
}