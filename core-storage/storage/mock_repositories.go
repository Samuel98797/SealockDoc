package storage

import (
	"context"
	"sync"

	"github.com/sealock/core-storage/model"
)

// MockFileRepository 内存中的文件仓库实现，用于测试
type MockFileRepository struct {
	files map[string]*model.File
	mutex sync.RWMutex
}

// NewMockFileRepository 创建新的 Mock 文件仓库
func NewMockFileRepository() FileRepository {
	return &MockFileRepository{
		files: make(map[string]*model.File),
	}
}

func (m *MockFileRepository) CreateFile(ctx context.Context, file *model.File) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.files[file.Hash] = file
	return nil
}

func (m *MockFileRepository) GetFileByHash(ctx context.Context, hash string) (*model.File, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	file, exists := m.files[hash]
	if !exists {
		return nil, nil // 模拟 GORM 的行为，找不到返回 nil
	}
	return file, nil
}

func (m *MockFileRepository) UpdateFile(ctx context.Context, file *model.File) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.files[file.Hash] = file
	return nil
}

func (m *MockFileRepository) DeleteFile(ctx context.Context, fileID uint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// 简化实现：根据 ID 查找并删除
	for hash, file := range m.files {
		if file.ID == fileID {
			delete(m.files, hash)
			break
		}
	}
	return nil
}

func (m *MockFileRepository) GetAllFiles(ctx context.Context) ([]model.File, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	files := make([]model.File, 0, len(m.files))
	for _, file := range m.files {
		files = append(files, *file)
	}
	return files, nil
}

// MockBlockRepository 内存中的块仓库实现，用于测试
type MockBlockRepository struct {
	blocks map[string]*model.Block
	mutex  sync.RWMutex
}

// NewMockBlockRepository 创建新的 Mock 块仓库
func NewMockBlockRepository() BlockRepository {
	return &MockBlockRepository{
		blocks: make(map[string]*model.Block),
	}
}

func (m *MockBlockRepository) SaveBlockMetadata(ctx context.Context, block *model.Block) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.blocks[block.Hash] = block
	return nil
}

func (m *MockBlockRepository) GetBlockMetadata(ctx context.Context, hash string) (*model.Block, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	block, exists := m.blocks[hash]
	if !exists {
		return nil, nil
	}
	return block, nil
}

func (m *MockBlockRepository) IncrementRefCount(ctx context.Context, hash string, delta int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if block, exists := m.blocks[hash]; exists {
		block.RefCount += delta
	}
	return nil
}

func (m *MockBlockRepository) DecrementBlockRefCount(ctx context.Context, hash string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if block, exists := m.blocks[hash]; exists {
		block.RefCount--
	}
	return nil
}

func (m *MockBlockRepository) ListOrphanBlocks(ctx context.Context) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var orphans []string
	for hash, block := range m.blocks {
		if block.RefCount <= 0 {
			orphans = append(orphans, hash)
		}
	}
	return orphans, nil
}

// MockSnapshotRepository 内存中的快照仓库实现，用于测试
type MockSnapshotRepository struct {
	snapshots map[uint]*model.Snapshot
	nextID    uint
	mutex     sync.RWMutex
}

// NewMockSnapshotRepository 创建新的 Mock 快照仓库
func NewMockSnapshotRepository() SnapshotRepository {
	return &MockSnapshotRepository{
		snapshots: make(map[uint]*model.Snapshot),
		nextID:    1,
	}
}

func (m *MockSnapshotRepository) CreateSnapshot(ctx context.Context, snapshot *model.Snapshot) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	id := m.nextID
	m.nextID++
	snapshot.ID = id
	m.snapshots[id] = snapshot
	return nil
}

func (m *MockSnapshotRepository) GetSnapshotByID(ctx context.Context, id uint) (*model.Snapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	snapshot, exists := m.snapshots[id]
	if !exists {
		return nil, nil
	}
	return snapshot, nil
}

func (m *MockSnapshotRepository) GetSnapshotByUUID(ctx context.Context, uuid string) (*model.Snapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, snapshot := range m.snapshots {
		if snapshot.UUID == uuid {
			return snapshot, nil
		}
	}
	return nil, nil
}

func (m *MockSnapshotRepository) ListSnapshots(ctx context.Context, limit, offset int) ([]model.Snapshot, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	snapshots := make([]model.Snapshot, 0, len(m.snapshots))
	i := 0
	for _, snapshot := range m.snapshots {
		if i >= offset {
			if limit <= 0 || len(snapshots) < limit {
				snapshots = append(snapshots, *snapshot)
			} else {
				break
			}
		}
		i++
	}
	return snapshots, nil
}

func (m *MockSnapshotRepository) ListSnapshotFiles(ctx context.Context, snapshotID uint, limit, offset int) ([]model.SnapshotFile, error) {
	// 简化实现：返回空列表
	return []model.SnapshotFile{}, nil
}

func (m *MockSnapshotRepository) CreateSnapshotFile(ctx context.Context, snapshotFile *model.SnapshotFile) error {
	// 简化实现：不实际存储
	return nil
}