package service

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"

	"github.com/sealock/core-storage/model"
	"github.com/sealock/core-storage/storage"
)

// SyncService provides synchronization functionality using Merkle Tree comparison
type SyncService struct {
	fileRepository storage.FileRepository
	blockStore     storage.BlockStore
}

// NewSyncService creates a new synchronization service
func NewSyncService(fileRepo storage.FileRepository, blockStore storage.BlockStore) *SyncService {
	return &SyncService{
		fileRepository: fileRepo,
		blockStore:     blockStore,
	}
}

// BuildMerkleTree constructs a Merkle Tree for a given file list
func (s *SyncService) BuildMerkleTree(files []model.File) string {
	if len(files) == 0 {
		return ""
	}

	// Sort files by name to ensure consistent ordering
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Hash all file hashes to create root
	hasher := sha256.New()
	for _, file := range files {
		hasher.Write([]byte(file.Hash))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// BuildDirectoryMerkleTree 构建目录树的Merkle树，支持目录层次结构
func (s *SyncService) BuildDirectoryMerkleTree(entries []model.DirectoryEntry) string {
	if len(entries) == 0 {
		emptyHash := sha256.Sum256([]byte{})
		return hex.EncodeToString(emptyHash[:])
	}

	// 按名称排序确保一致性
	sortedEntries := make([]model.DirectoryEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Name < sortedEntries[j].Name
	})

	// 计算每个条目的哈希值
	entryHashes := make([]string, len(sortedEntries))
	for i, entry := range sortedEntries {
		var contentHash string
		if entry.IsDir && entry.Children != nil {
			// 转换指针切片为值切片
			children := make([]model.DirectoryEntry, len(entry.Children))
			for j, child := range entry.Children {
				children[j] = *child
			}
			contentHash = s.BuildDirectoryMerkleTree(children)
		} else {
			contentHash = entry.Hash
		}

		// 组合名称、类型和内容哈希
		combined := entry.Name + strconv.FormatBool(entry.IsDir) + contentHash
		h := sha256.Sum256([]byte(combined))
		entryHashes[i] = hex.EncodeToString(h[:])
	}

	// 递归构建Merkle树
	for len(entryHashes) > 1 {
		if len(entryHashes)%2 == 1 {
			entryHashes = append(entryHashes, entryHashes[len(entryHashes)-1])
		}

		var newLevel []string
		for i := 0; i < len(entryHashes); i += 2 {
			pairHash := sha256.Sum256([]byte(entryHashes[i] + entryHashes[i+1]))
			newLevel = append(newLevel, hex.EncodeToString(pairHash[:]))
		}
		entryHashes = newLevel
	}

	return entryHashes[0]
}

// CompareMerkleTrees compares two Merkle roots and returns the differences
func (s *SyncService) CompareMerkleTrees(oldRoot, newRoot string, oldFiles, newFiles []model.File) (added, removed, updated []model.File) {
	// If roots are identical, no changes
	if oldRoot == newRoot {
		return nil, nil, nil
	}

	// Sort both file lists by name
	sort.Slice(oldFiles, func(i, j int) bool { return oldFiles[i].Name < oldFiles[j].Name })
	sort.Slice(newFiles, func(i, j int) bool { return newFiles[i].Name < newFiles[j].Name })

	i, j := 0, 0
	for i < len(oldFiles) && j < len(newFiles) {
		switch {
		case oldFiles[i].Name == newFiles[j].Name:
			// File exists in both snapshots
			if oldFiles[i].Hash != newFiles[j].Hash {
				updated = append(updated, newFiles[j])
			}
			i++
			j++
		case oldFiles[i].Name < newFiles[j].Name:
			// File removed
			removed = append(removed, oldFiles[i])
			i++
		default:
			// File added
			added = append(added, newFiles[j])
			j++
		}
	}

	// Add remaining files
	for ; i < len(oldFiles); i++ {
		removed = append(removed, oldFiles[i])
	}
	for ; j < len(newFiles); j++ {
		added = append(added, newFiles[j])
	}

	return added, removed, updated
}

// CompareDirectoryTrees 比较两个目录树的差异
func (s *SyncService) CompareDirectoryTrees(oldRoot, newRoot string, oldEntries, newEntries []model.DirectoryEntry) (added, removed, modified []model.DirectoryEntry) {
	// 根哈希相同表示目录树完全一致
	if oldRoot == newRoot {
		return nil, nil, nil
	}

	// 按名称排序确保一致性
	sort.Slice(oldEntries, func(i, j int) bool { return oldEntries[i].Name < oldEntries[j].Name })
	sort.Slice(newEntries, func(i, j int) bool { return newEntries[i].Name < newEntries[j].Name })

	i, j := 0, 0
	for i < len(oldEntries) && j < len(newEntries) {
		switch {
		case oldEntries[i].Name == newEntries[j].Name:
			// 条目存在于两个快照中
			if oldEntries[i].Hash != newEntries[j].Hash {
				modified = append(modified, newEntries[j])
			}
			i++
			j++
		case oldEntries[i].Name < newEntries[j].Name:
			// 条目已删除
			removed = append(removed, oldEntries[i])
			i++
		default:
			// 条目已添加
			added = append(added, newEntries[j])
			j++
		}
	}

	// 添加剩余条目
	for ; i < len(oldEntries); i++ {
		removed = append(removed, oldEntries[i])
	}
	for ; j < len(newEntries); j++ {
		added = append(added, newEntries[j])
	}

	return added, removed, modified
}
