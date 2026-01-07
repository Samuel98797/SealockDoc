package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sealock/core-storage/chunker"
	"github.com/sealock/core-storage/model"
	"github.com/sealock/core-storage/storage"
)

// FileService 文件业务服务层
// 负责处理文件上传、下载、完整性校验、增量同步和快照管理等核心功能
type FileService struct {
	blockStore         storage.BlockStore        // 块存储接口，用于实际的数据块读写
	fileRepo           storage.FileRepository    // 文件仓库接口，用于管理文件元数据
	blockRepo          storage.BlockRepository   // 块仓库接口，用于管理块的引用计数等元数据
	chunker            chunker.Chunker           // 分块器，用于将文件流切分成固定或动态大小的数据块
	snapshotService    *SnapshotService          // 快照服务，用于创建和管理系统在某一时刻的状态快照
	snapshotRepo       storage.SnapshotRepository // 快照仓库接口，用于持久化快照元数据
	autoUpdateRefCount bool                      // 标志位，指示是否自动管理块的引用计数
	redisClient        *redis.Client             // Redis客户端，用于跟踪上传会话等临时状态
}

// NewFileService 创建并初始化一个新的文件服务实例
// 参数:
// - bs: 底层块存储实现
// - fr: 文件元数据仓库
// - br: 块元数据仓库
// - c: 文件分块策略
// - sr: 快照元数据仓库
// - redisClient: 用于会话管理的Redis客户端
// - autoUpdateRefCount: 是否开启引用计数自动增减
// 返回一个配置好的*FileService指针
func NewFileService(
	bs storage.BlockStore,
	fr storage.FileRepository,
	br storage.BlockRepository,
	c chunker.Chunker,
	sr storage.SnapshotRepository,
	redisClient *redis.Client,
	autoUpdateRefCount bool,
) *FileService {
	snapshotService := NewSnapshotService(sr, fr)
	return &FileService{
		blockStore:         bs,
		fileRepo:           fr,
		blockRepo:          br,
		chunker:            c,
		snapshotService:    snapshotService,
		snapshotRepo:       sr,
		autoUpdateRefCount: autoUpdateRefCount,
		redisClient:        redisClient,
	}
}

// UploadFile 上传一个新文件到存储系统
// 实现步骤:
// 1. 使用分块器将文件数据切分成多个块
// 2. 将每个块独立存储，利用内容寻址(CAS)实现自动去重
// 3. 更新每个块的引用计数
// 4. 将所有块的哈希值序列化后与文件名、大小等信息一起作为元数据保存
// 5. 成功后触发创建一个自动快照
// 参数:
// - ctx: 上下文，用于控制超时和取消
// - fileName: 文件的原始名称
// - data: 文件的完整二进制数据
// 返回上传成功后的文件对象和错误信息
func (s *FileService) UploadFile(ctx context.Context, fileName string, data []byte) (*model.File, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	// 步骤1: 分块
	chunks, err := s.chunker.(*chunker.FixedSizeChunker).Chunk(data)
	if err != nil {
		return nil, fmt.Errorf("chunk failed: %w", err)
	}

	// 步骤1.5: 重新计算原始块数据
	var blockHashes []string
	var currentPos int
	blockSize := s.chunker.(*chunker.FixedSizeChunker).ChunkSize()
	
	for i := 0; i < len(chunks); i++ {
		// 计算当前块的数据
		endPos := currentPos + blockSize
		if endPos > len(data) {
			endPos = len(data)
		}
		
		currentChunkData := data[currentPos:endPos]
		
		// 存储块并获取其哈希
		hash, err := s.blockStore.Put(ctx, currentChunkData)
		if err != nil {
			return nil, fmt.Errorf("failed to store block: %w", err)
		}
		blockHashes = append(blockHashes, hash)

		// 增加块的引用计数
		if err := s.blockRepo.IncrementRefCount(ctx, hash, 1); err != nil {
			return nil, fmt.Errorf("failed to increment block ref count: %w", err)
		}
		
		currentPos = endPos
	}

	// 步骤3: 记录文件元数据
	file := &model.File{
		Name: fileName,
		Size: int64(len(data)),
		Hash: calculateFileHash(data), // Calculate file hash from content
	}

	// 将块ID序列化为JSON并存储到BlockIDs字段
	blockIDsJSON, err := json.Marshal(blockHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes: %w", err)
	}
	file.BlockIDs = blockIDsJSON

	// 保存文件元数据
	if err := s.fileRepo.CreateFile(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// 创建自动快照（异步）
	go func() {
		_, _ = s.snapshotService.CreateCommit(ctx, "", "")
	}()

	// 步骤4: 返回文件
	return file, nil
}

// DownloadFile 根据文件哈希下载文件
// 实现步骤:
// 1. 通过文件哈希查询文件元数据
// 2. 反序列化出构成该文件的所有数据块哈希列表
// 3. 按顺序从块存储中读取每一个块的数据
// 4. 将所有块的数据拼接成完整的原始文件数据
// 参数:
// - ctx: 上下文
// - fileHash: 文件的内容哈希
// 返回完整的文件数据和错误信息
func (s *FileService) DownloadFile(ctx context.Context, fileHash string) ([]byte, error) {
	// 1. 获取文件元数据
	file, err := s.fileRepo.GetFileByHash(ctx, fileHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// 2. 解析块ID列表
	var blockHashes []string
	if err := json.Unmarshal(file.BlockIDs, &blockHashes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block IDs: %w", err)
	}

	// 3. 获取所有块数据
	var fileData []byte
	for _, blockHash := range blockHashes {
		// 从块存储中获取块数据
		blockData, err := s.blockStore.Get(ctx, blockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block %s: %w", blockHash, err)
		}

		// 拼接块数据
		fileData = append(fileData, blockData...)
	}

	// 4. 返回完整文件数据
	return fileData, nil
}

// CheckIntegrity 检查指定文件的完整性
// 通过验证文件所依赖的每一个数据块是否都存在于块存储中来判断文件是否完整
// 这是确保数据可靠性的关键检查
// 参数:
// - ctx: 上下文
// - fileHash: 待检查的文件哈希
// 返回文件是否完整和潜在的错误
func (s *FileService) CheckIntegrity(ctx context.Context, fileHash string) (bool, error) {
	file, err := s.fileRepo.GetFileByHash(ctx, fileHash)
	if err != nil {
		return false, fmt.Errorf("file not found: %w", err)
	}

	// 解析块ID列表
	var blockHashes []string
	if err := json.Unmarshal(file.BlockIDs, &blockHashes); err != nil {
		return false, fmt.Errorf("failed to unmarshal block IDs: %w", err)
	}

	for _, blockHash := range blockHashes {
		exists, err := s.blockStore.Exists(ctx, blockHash)
		if err != nil || !exists {
			return false, nil
		}
	}

	return true, nil
}

// ============ 高级功能：增量同步 ============

// FileChangeSet 表示文件变化集合
type FileChangeSet struct {
	Added    []*model.File
	Modified []*model.File
	Deleted  []uint
}

// DetectChanges 检测两组文件集合之间的差异
// 利用内容哈希进行比对，可以高效地识别出新增、修改和删除的文件
// 是实现增量备份和同步的核心方法
// 参数:
// - ctx: 上下文
// - oldFileHashes: 旧版本的文件哈希映射表
// - newFileHashes: 新版本的文件哈希映射表
// 返回一个描述所有变化的FileChangeSet对象
func (s *FileService) DetectChanges(ctx context.Context, oldFileHashes, newFileHashes map[string]*model.File) *FileChangeSet {
	changes := &FileChangeSet{
		Added:    make([]*model.File, 0),
		Modified: make([]*model.File, 0),
		Deleted:  make([]uint, 0),
	}

	// 检测新增和修改
	for hash, newFile := range newFileHashes {
		if _, exists := oldFileHashes[hash]; exists {
			// 如果哈希相同，内容相同（CAS 的优势）
			continue
		} else {
			// 新增文件
			changes.Added = append(changes.Added, newFile)
		}
	}

	// 检测删除
	for hash, oldFile := range oldFileHashes {
		if _, exists := newFileHashes[hash]; !exists {
			changes.Deleted = append(changes.Deleted, oldFile.ID)
		}
	}

	return changes
}

// GetFileByHash 根据文件的内容哈希获取其元数据
// 提供了一种基于内容寻址的方式来检索文件信息
// 参数:
// - ctx: 上下文
// - hash: 文件的内容哈希
// 返回查询到的文件对象和错误信息
func (s *FileService) GetFileByHash(ctx context.Context, hash string) (*model.File, error) {
	// 从元数据存储中获取文件记录
	file, err := s.fileRepo.GetFileByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get file by hash: %w", err)
	}

	return file, nil
}

// GetAllFiles 获取系统中存储的所有文件的元数据
// 返回一个包含所有文件对象的切片
// 注意：此操作可能在文件数量巨大时消耗较多资源
// 参数:
// - ctx: 上下文
// 返回文件指针切片和错误信息
func (s *FileService) GetAllFiles(ctx context.Context) ([]*model.File, error) {
	// 从元数据存储中获取所有文件记录
	files, err := s.fileRepo.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}

	// 转换为指针切片
	result := make([]*model.File, len(files))
	for i := range files {
		result[i] = &files[i]
	}

	return result, nil
}

// DeleteFile 删除一个指定的文件
// 实现步骤:
// 1. 查找文件元数据
// 2. 解析出其所依赖的所有数据块
// 3. 对每个块的引用计数进行递减
// 4. 删除文件自身的元数据记录
// 5. 成功后触发创建一个自动快照
// 参数:
// - ctx: 上下文
// - fileHash: 待删除文件的哈希
// 返回操作结果的错误信息
func (s *FileService) DeleteFile(ctx context.Context, fileHash string) error {
	// 1. 获取文件
	file, err := s.fileRepo.GetFileByHash(ctx, fileHash)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 2. 解析块ID列表
	var blockHashes []string
	if err := json.Unmarshal(file.BlockIDs, &blockHashes); err != nil {
		return fmt.Errorf("failed to unmarshal block IDs: %w", err)
	}

	// 3. 逐块减少引用计数
	for _, blockHash := range blockHashes {
		// 从元数据中减少引用计数
		if err := s.blockRepo.DecrementBlockRefCount(ctx, blockHash); err != nil {
			// 记录错误但继续处理其他块
			fmt.Printf("Warning: failed to decrement ref count for block %s: %v\n", blockHash, err)
		}
	}

	// 4. 删除文件记录
	if err := s.fileRepo.DeleteFile(ctx, file.ID); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	// 创建自动快照（异步）
	go func() {
		_, _ = s.snapshotService.CreateCommit(ctx, "", "")
	}()

	return nil
}

// CreateSnapshot 创建一个系统快照
// 快照记录了当前所有文件的状态，可用于版本管理和恢复
// 参数:
// - ctx: 上下文
// - name: 快照名称
// - description: 快照描述
// 返回创建的快照对象和错误信息
func (s *FileService) CreateSnapshot(ctx context.Context, name, description string) (*model.Snapshot, error) {
	// 获取所有文件
	files, err := s.fileRepo.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	// Create snapshot record
	snapshot := &model.Snapshot{
		Name:        name,
		Description: description,
	}

	err = s.snapshotRepo.CreateSnapshot(ctx, snapshot)
	if err != nil {
		return nil, err
	}

	// Link files to the snapshot
	for _, file := range files {
		snapshotFile := &model.SnapshotFile{
			SnapshotID: snapshot.ID,
			FileID:     file.ID,
			FileName:   file.Name,
		}
		if err := s.snapshotRepo.CreateSnapshotFile(ctx, snapshotFile); err != nil {
			return nil, fmt.Errorf("failed to create snapshot file: %w", err)
		}
	}

	return snapshot, nil
}

// CompareSnapshots 比较两个快照之间的差异
// 用于分析两次备份之间文件的变化情况
// 参数:
// - ctx: 上下文
// - oldSnapshotID: 旧快照的ID
// - newSnapshotID: 新快照的ID
// 返回描述差异的SnapshotDiff对象和错误信息
func (s *FileService) CompareSnapshots(ctx context.Context, oldSnapshotID, newSnapshotID uint) (*model.SnapshotDiff, error) {
	// Get snapshots
	oldSnapshot, err := s.snapshotRepo.GetSnapshotByID(ctx, oldSnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get old snapshot: %w", err)
	}

	newSnapshot, err := s.snapshotRepo.GetSnapshotByID(ctx, newSnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new snapshot: %w", err)
	}

	// If root hashes are the same, no changes
	if oldSnapshot.RootHash == newSnapshot.RootHash {
		return &model.SnapshotDiff{
			Added:    []model.SnapshotFile{},
			Removed:  []model.SnapshotFile{},
			Modified: []model.SnapshotFile{},
		}, nil
	}

	// Get files in both snapshots
	oldFiles, err := s.snapshotRepo.ListSnapshotFiles(ctx, oldSnapshotID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get files for old snapshot: %w", err)
	}

	newFiles, err := s.snapshotRepo.ListSnapshotFiles(ctx, newSnapshotID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get files for new snapshot: %w", err)
	}

	// Build file hash maps for comparison
	oldFileMap := make(map[string]model.SnapshotFile)
	for _, file := range oldFiles {
		oldFileMap[file.FileHash] = file
	}

	newFileMap := make(map[string]model.SnapshotFile)
	for _, file := range newFiles {
		newFileMap[file.FileHash] = file
	}

	// Detect changes
	diff := &model.SnapshotDiff{
		Added:    []model.SnapshotFile{},
		Removed:  []model.SnapshotFile{},
		Modified: []model.SnapshotFile{},
	}

	// Files in new but not in old -> Added
	for hash, file := range newFileMap {
		if _, exists := oldFileMap[hash]; !exists {
			diff.Added = append(diff.Added, file)
		}
	}

	// Files in old but not in new -> Removed
	for hash, file := range oldFileMap {
		if _, exists := newFileMap[hash]; !exists {
			diff.Removed = append(diff.Removed, file)
		}
	}

	// Files with same hash but different names or other attributes could be considered modified
	// For now, we only consider hash differences

	return diff, nil
}

// calculateFileHash 根据文件内容计算其唯一哈希值
// 在实际生产环境中，这里应使用真正的哈希算法如SHA-256
// 当前实现仅为占位符，使用数据长度生成伪哈希
// 参数:
// - data: 文件的原始字节数据
// 返回计算出的哈希字符串
func calculateFileHash(data []byte) string {
	// In a real implementation, this would calculate the actual hash
	// For now, we'll return a placeholder
	return fmt.Sprintf("hash_%d", len(data))
}





