package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Block 代表存储中的最小单位
// 每个 Block 由其内容的 SHA-256 哈希命名（内容寻址存储原理）
type Block struct {
	ID        uint      `gorm:"primaryKey"`
	Hash      string    `gorm:"uniqueIndex;type:varchar(64)"` // SHA-256 hex string
	Size      int64     `gorm:"type:bigint"`                  // 字节大小
	Data      []byte    `gorm:"type:bytea"`                   // 实际数据（开发环境）
	RefCount  int       `gorm:"default:0"`                    // 引用计数（垃圾回收）
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// File 代表一个文件，由多个 Block 组成
// 类似 Git 中的 Blob，存储的是 Block ID 的列表及元数据
type File struct {
	ID        uint           `gorm:"primaryKey"`
	UUID      string         `gorm:"uniqueIndex;type:varchar(36)"` // 文件唯一标识
	Name      string         `gorm:"type:varchar(255)"`
	Size      int64          `gorm:"type:bigint"`      // 文件总大小
	Hash      string         `gorm:"type:varchar(64)"` // 文件内容的 Merkle hash
	BlockIDs  datatypes.JSON `gorm:"type:jsonb"`       // Block ID 列表（JSON 数组）
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	LibraryID uint           `gorm:"index"`
}

// LibraryVersion 代表 Library 的一次提交（类似 Git Commit）
type LibraryVersion struct {
	ID            uint           `gorm:"primaryKey"`
	CommitID      string         `gorm:"uniqueIndex;type:varchar(64)"` // Commit hash
	LibraryID     uint           `gorm:"index"`
	RootHash      string         `gorm:"type:varchar(64)"` // 根目录树的 hash
	Message       string         `gorm:"type:text"`
	Author        string         `gorm:"type:varchar(255)"`
	ParentCommits datatypes.JSON `gorm:"type:jsonb"` // 父 commit 列表（支持合并）
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
}

// Library 代表一个库/仓库（类似 Git Repository）
// 包含版本历史和当前状态
type Library struct {
	ID          uint   `gorm:"primaryKey"`
	UUID        string `gorm:"uniqueIndex;type:varchar(36)"`
	Name        string `gorm:"type:varchar(255)"`
	Description string `gorm:"type:text"`
	OwnerID     uint   `gorm:"index"` // 用户 ID
	// 当前版本（HEAD）
	CurrentVersionID uint
	// 统计信息
	TotalSize    int64     `gorm:"type:bigint;default:0"`
	FileCount    int       `gorm:"default:0"`
	VersionCount int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// DirectoryEntry 代表目录树中的一个条目（文件或目录）
// 用于重建目录结构和计算 Merkle hash
type DirectoryEntry struct {
	Name     string            // 文件/目录名
	IsDir    bool              // 是否为目录
	Hash     string            // 目录条目的 hash（文件为 Block hash，目录为树 hash）
	Size     int64             // 大小
	Children []*DirectoryEntry // 子项（仅当 IsDir=true 时）
	Metadata map[string]string // 额外元数据（权限、修改时间等）
}

// ============ 辅助函数 ============

// NewBlock 创建新的 Block，自动计算 SHA-256 hash
func NewBlock(data []byte) *Block {
	hash := sha256.Sum256(data)
	return &Block{
		Hash: hex.EncodeToString(hash[:]),
		Size: int64(len(data)),
		Data: data,
	}
}

// NewFile 创建新文件
func NewFile(name string, blockIDs []string, fileHash string) *File {
	blockIDsJSON, err := json.Marshal(blockIDs)
	if err != nil {
		blockIDsJSON = []byte("[]")
	}

	return &File{
		UUID:     uuid.New().String(),
		Name:     name,
		Size:     calculateTotalSize(), // 需要实现计算文件总大小的函数
		Hash:     fileHash,
		BlockIDs: datatypes.JSON(blockIDsJSON),
	}
}

// NewLibrary 创建新库
func NewLibrary(name, description string, ownerID uint) *Library {
	return &Library{
		UUID:        uuid.New().String(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
	}
}

// NewLibraryVersion 创建版本提交
func NewLibraryVersion(libraryID uint, rootHash, message, author string, parentCommits []string) *LibraryVersion {
	hash := sha256.Sum256([]byte(rootHash + message + author + time.Now().String()))

	// 使用 JSON 序列化处理 datatypes.JSON 类型
	parentCommitsJSON, err := json.Marshal(parentCommits)
	if err != nil {
		// 这里应该处理错误，但为了保持与原代码相同的函数签名，我们使用空数组
		parentCommitsJSON = []byte("[]")
	}

	return &LibraryVersion{
		LibraryID:     libraryID,
		CommitID:      hex.EncodeToString(hash[:]),
		RootHash:      rootHash,
		Message:       message,
		Author:        author,
		ParentCommits: datatypes.JSON(parentCommitsJSON),
	}
}

// calculateTotalSize 计算文件总大小的辅助函数
func calculateTotalSize() int64 {
	// 这里需要根据实际需求实现大小计算逻辑
	// 可以查询数据库中的Block表获取每个block的大小并求和
	return 0
}
