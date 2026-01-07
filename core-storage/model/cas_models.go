package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Repository represents a storage repository with end-to-end encryption support
type Repository struct {
	gorm.Model
	Name             string         `gorm:"not null;unique"`
	OwnerID          uint           `gorm:"index"`
	EncryptionConfig datatypes.JSON `gorm:"type:jsonb"` // Stores encryption metadata like algorithm, key ID, etc.
}

// Commit represents a version snapshot of the repository
// Similar to Git commits, but with CAS-specific optimizations
type Commit struct {
	gorm.Model
	RepoID           uint      `gorm:"index"`
	CommitHash       string    `gorm:"uniqueIndex;not null"`
	ParentCommitHash *string   `gorm:"index"`
	RootTreeHash     string    `gorm:"index;not null"` // Root node hash of the file system tree
	Author           string    `gorm:"not null"`
	Message          string    
	CreatedAt        time.Time `gorm:"index"`

	Repository Repository `gorm:"foreignKey:RepoID"`
}

// Node represents a file system node (file or directory)
// Similar to Git tree objects, but optimized for CAS
type Node struct {
	gorm.Model
	RepoID        uint           `gorm:"index"`
	CommitHash    string         `gorm:"index;not null"` // References Commit.CommitHash
	ParentID      *uint          `gorm:"index"`          // Self-referential for tree structure
	Name          string         `gorm:"not null"`
	Size          int64          // File size (0 for directories)
	Type          string         `gorm:"not null;check:type IN ('file', 'dir')"` // Node type
	ContentHash   *string        `gorm:"index"`          // For files: points to Block.Hash; for dirs: may be nil
	BlockHashes   []string       `gorm:"type:varchar(64)[]"` // Array of block hashes for file content
	Extra         datatypes.JSON `gorm:"type:jsonb"`         // Extended attributes in JSONB

	Commit Commit `gorm:"foreignKey:CommitHash;references:CommitHash"`
}

