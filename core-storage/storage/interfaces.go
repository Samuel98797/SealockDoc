package storage

import (
	"context"

	"github.com/sealock/core-storage/model"
)

// BlockStore 定义 Block 存储接口（内容寻址存储的核心）
// 所有 Block 操作都通过其 SHA-256 hash 进行寻址
type BlockStore interface {
	// Put 将数据块存储，返回其哈希值
	Put(ctx context.Context, data []byte) (hash string, err error)

	// Get 通过哈希值获取数据块
	Get(ctx context.Context, hash string) (data []byte, err error)

	// Exists 检查数据块是否存在
	Exists(ctx context.Context, hash string) (bool, error)

	// Delete 删除数据块
	Delete(ctx context.Context, hash string) error

	// GetSize 获取数据块大小
	GetSize(ctx context.Context, hash string) (int64, error)
}

// FileRepository 文件数据访问层
type FileRepository interface {
	// CreateFile 创建文件记录
	CreateFile(ctx context.Context, file *model.File) error

	// GetFileByHash 通过文件 hash 获取文件
	GetFileByHash(ctx context.Context, hash string) (*model.File, error)

	// UpdateFile 更新文件
	UpdateFile(ctx context.Context, file *model.File) error

	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, fileID uint) error

	// GetAllFiles 获取所有文件
	GetAllFiles(ctx context.Context) ([]model.File, error)
}

// LibraryRepository 库的数据访问层
type LibraryRepository interface {
	// CreateLibrary 创建库
	CreateLibrary(ctx context.Context, lib *model.Library) error

	// GetLibraryByID 获取库
	GetLibraryByID(ctx context.Context, id uint) (*model.Library, error)

	// ListLibrariesByOwner 列出用户的所有库
	ListLibrariesByOwner(ctx context.Context, ownerID uint) ([]*model.Library, error)

	// UpdateLibrary 更新库信息
	UpdateLibrary(ctx context.Context, lib *model.Library) error

	// DeleteLibrary 删除库
	DeleteLibrary(ctx context.Context, id uint) error
}

// LibraryVersionRepository 版本控制数据访问层
type LibraryVersionRepository interface {
	// CreateVersion 创建版本
	CreateVersion(ctx context.Context, version *model.LibraryVersion) error

	// GetVersionByCommitID 通过 commit ID 获取版本
	GetVersionByCommitID(ctx context.Context, commitID string) (*model.LibraryVersion, error)

	// ListVersionsByLibrary 列出库的所有版本
	ListVersionsByLibrary(ctx context.Context, libraryID uint) ([]*model.LibraryVersion, error)

	// GetLatestVersion 获取库的最新版本
	GetLatestVersion(ctx context.Context, libraryID uint) (*model.LibraryVersion, error)
}

// BlockRepository Block 数据访问层（元数据存储）
type BlockRepository interface {
	// SaveBlockMetadata 保存 Block 的元数据（hash, size, ref_count）
	SaveBlockMetadata(ctx context.Context, block *model.Block) error

	// GetBlockMetadata 获取 Block 元数据
	GetBlockMetadata(ctx context.Context, hash string) (*model.Block, error)

	// IncrementRefCount 增加引用计数（GC 用）
	IncrementRefCount(ctx context.Context, hash string, delta int) error

	// DecrementBlockRefCount 减少引用计数（GC 用）
	DecrementBlockRefCount(ctx context.Context, hash string) error

	// ListOrphanBlocks 列出引用计数为 0 的 Block（可被删除）
	ListOrphanBlocks(ctx context.Context) ([]string, error)
}

// SnapshotRepository manages snapshot persistence
type SnapshotRepository interface {
	// CreateSnapshot creates a new snapshot
	CreateSnapshot(ctx context.Context, snapshot *model.Snapshot) error
	
	// GetSnapshotByID retrieves a snapshot by ID
	GetSnapshotByID(ctx context.Context, id uint) (*model.Snapshot, error)
	
	// GetSnapshotByUUID retrieves a snapshot by UUID
	GetSnapshotByUUID(ctx context.Context, uuid string) (*model.Snapshot, error)
	
	// ListSnapshots lists snapshots with pagination
	ListSnapshots(ctx context.Context, limit, offset int) ([]model.Snapshot, error)
	
	// ListSnapshotFiles lists files in a snapshot
	ListSnapshotFiles(ctx context.Context, snapshotID uint, limit, offset int) ([]model.SnapshotFile, error)
	
	// CreateSnapshotFile creates a new snapshot file entry
	CreateSnapshotFile(ctx context.Context, snapshotFile *model.SnapshotFile) error
}
