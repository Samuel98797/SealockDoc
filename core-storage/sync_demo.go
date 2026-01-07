package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sealock/core-storage/model"
	"github.com/sealock/core-storage/service"
)

func ExampleSyncService() {
	// Create a simple file repository mock
	fileRepo := &mockFileRepository{}
	blockStore := &mockBlockStore{}

	// Create sync service
	syncSvc := service.NewSyncService(fileRepo, blockStore)

	// Create test files
	files := []model.File{
		{
			Name: "dir1/file1.txt",
			Hash: "hash1",
		},
		{
			Name: "dir1/file2.txt",
			Hash: "hash2",
		},
		{
			Name: "dir2/file3.txt",
			Hash: "hash3",
		},
	}

	// Convert files to directory entries using our own implementation
	entries := convertFilesToDirectoryEntries(files)
	
	// Build Merkle tree
	rootHash := syncSvc.BuildDirectoryMerkleTree(entries)
	fmt.Printf("Root Hash: %s\n", rootHash)
	
	// Print directory structure
	printDirectory(entries, "")
}

// convertFilesToDirectoryEntries 将文件列表转换为目录树结构
func convertFilesToDirectoryEntries(files []model.File) []model.DirectoryEntry {
	root := make(map[string]*model.DirectoryEntry)

	// 创建所有目录条目
	for _, file := range files {
		parts := strings.Split(file.Name, "/")
		currentMap := root
		
		// 遍历路径的每个部分
		for i, part := range parts {
			if i == len(parts)-1 {
				// 最后一部分是文件名
				if currentEntry, exists := currentMap[part]; exists {
					// 如果已存在，更新为文件
					currentEntry.IsDir = false
					currentEntry.Hash = file.Hash
					currentEntry.Size = file.Size
				} else {
					// 创建新的文件条目
					currentMap[part] = &model.DirectoryEntry{
						Name:  part,
						IsDir: false,
						Hash:  file.Hash,
						Size:  file.Size,
					}
				}
			} else {
				// 中间部分是目录
				if currentEntry, exists := currentMap[part]; exists {
					// 目录已存在，继续到下一级
					if currentEntry.Children == nil {
						currentEntry.Children = make([]*model.DirectoryEntry, 0)
					}
					// 更新currentMap到子目录
					currentMap = createMapFromChildren(currentEntry.Children)
				} else {
					// 创建新的目录条目
					newDir := &model.DirectoryEntry{
						Name:     part,
						IsDir:    true,
						Hash:     "", // 将在构建Merkle树时计算
						Children: make([]*model.DirectoryEntry, 0),
					}
					currentMap[part] = newDir
					// 更新currentMap到新创建的目录
					currentMap = make(map[string]*model.DirectoryEntry)
				}
			}
		}
	}

	// 将map转换为切片
	var entries []model.DirectoryEntry
	for _, entry := range root {
		entries = append(entries, *entry)
	}

	return entries
}

// createMapFromChildren 从 []*DirectoryEntry 创建 map[string]*model.DirectoryEntry
func createMapFromChildren(children []*model.DirectoryEntry) map[string]*model.DirectoryEntry {
	result := make(map[string]*model.DirectoryEntry)
	for _, child := range children {
		result[child.Name] = child
	}
	return result
}

// mockFileRepository is a mock implementation of FileRepository for testing
type mockFileRepository struct{}

func (m *mockFileRepository) GetFileByHash(ctx context.Context, hash string) (*model.File, error) {
	return nil, nil
}

func (m *mockFileRepository) GetAllFiles(ctx context.Context) ([]model.File, error) {
	return []model.File{}, nil
}

func (m *mockFileRepository) SaveFile(ctx context.Context, file *model.File) error {
	return nil
}

func (m *mockFileRepository) DeleteFile(ctx context.Context, id uint) error {
	return nil
}

// Add missing CreateFile method to satisfy FileRepository interface
func (m *mockFileRepository) CreateFile(ctx context.Context, file *model.File) error {
	return nil
}

// Add missing UpdateFile method to satisfy FileRepository interface
func (m *mockFileRepository) UpdateFile(ctx context.Context, file *model.File) error {
	return nil
}

// mockBlockStore is a mock implementation of BlockStore for testing
type mockBlockStore struct{}

func (m *mockBlockStore) Put(ctx context.Context, data []byte) (string, error) {
	return "", nil
}

func (m *mockBlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
	return nil, nil
}

func (m *mockBlockStore) Exists(ctx context.Context, hash string) (bool, error) {
	return false, nil
}

func (m *mockBlockStore) Delete(ctx context.Context, hash string) error {
	return nil
}

func (m *mockBlockStore) GetSize(ctx context.Context, hash string) (int64, error) {
	return 0, nil
}

func printDirectory(entries []model.DirectoryEntry, prefix string) {
	for i, entry := range entries {
		isLast := i == len(entries)-1
		if entry.IsDir {
			if isLast {
				fmt.Printf("%s└── 📁 %s/\n", prefix, entry.Name)
				if entry.Children != nil {
					// Convert []*model.DirectoryEntry to []model.DirectoryEntry
					subEntries := make([]model.DirectoryEntry, len(entry.Children))
					for j, child := range entry.Children {
						subEntries[j] = *child
					}
					printDirectory(subEntries, prefix+"    ")
				}
			} else {
				fmt.Printf("%s├── 📁 %s/\n", prefix, entry.Name)
				if entry.Children != nil {
					// Convert []*model.DirectoryEntry to []model.DirectoryEntry
					subEntries := make([]model.DirectoryEntry, len(entry.Children))
					for j, child := range entry.Children {
						subEntries[j] = *child
					}
					printDirectory(subEntries, prefix+"│   ")
				}
			}
		} else {
			if isLast {
				fmt.Printf("%s└── 📄 %s (%s)\n", prefix, entry.Name, entry.Hash)
			} else {
				fmt.Printf("%s├── 📄 %s (%s)\n", prefix, entry.Name, entry.Hash)
			}
		}
	}
}