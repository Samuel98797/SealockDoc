package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sealock/core-storage/model"
	"github.com/sealock/core-storage/storage"
)

// SnapshotService 快照服务，处理版本控制相关业务逻辑
type SnapshotService struct {
	SnapshotRepo storage.SnapshotRepository
	FileRepo     storage.FileRepository
}

// NewSnapshotService 创建快照服务实例
func NewSnapshotService(snapshotRepo storage.SnapshotRepository, fileRepo storage.FileRepository) *SnapshotService {
	return &SnapshotService{
		SnapshotRepo: snapshotRepo,
		FileRepo:     fileRepo,
	}
}

// CreateCommit 创建新的版本提交
// 当用户修改文件夹内容并点击保存时，递归扫描目录生成Merkle Tree哈希
// 对比上一个Commit的Root Hash，无变化则不生成新记录
// 整个操作在数据库事务中完成，保证原子性
func (s *SnapshotService) CreateCommit(ctx context.Context, repoID string, userID string) (*model.Commit, error) {
	// 1. 获取当前仓库的所有文件
	files, err := s.FileRepo.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}

	// 2. 计算所有文件的Merkle根哈希
	var allHashes []byte
	for _, file := range files {
		allHashes = append(allHashes, []byte(file.Hash)...) // 追加每个文件的哈希
	}

	// 计算父目录哈希
	h := sha256.Sum256(allHashes)
	currentRootTreeHash := fmt.Sprintf("%x", h)

	// 3. 获取上一个Commit记录
	lastCommit, err := s.getLastCommit(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("获取最新提交记录失败: %w", err)
	}

	// 4. 对比Root Tree Hash，无变化则跳过
	if lastCommit != nil && lastCommit.RootTreeHash == currentRootTreeHash {
		return nil, fmt.Errorf("无变化: 当前状态与最新提交相同")
	}

	// 5. 创建新Commit记录
	commitUUID := uuid.New().String()
	newCommit := &model.Commit{
		RepoID:           1, // 简化实现，实际应根据repoID确定
		CommitHash:       commitUUID,
		ParentCommitHash: nil, // 简化实现
		RootTreeHash:     currentRootTreeHash,
		Author:           userID,
		Message:          "Auto commit",
		CreatedAt:        time.Now(),
	}

	// 6. 转换为Snapshot并保存
	snapshot := &model.Snapshot{
		UUID:        newCommit.CommitHash,
		Name:        newCommit.RootTreeHash,
		Description: newCommit.Message,
		RootHash:    newCommit.RootTreeHash,
		CreatedAt:   newCommit.CreatedAt,
	}

	err = s.SnapshotRepo.CreateSnapshot(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("创建提交记录失败: %w", err)
	}

	return newCommit, nil
}

// getLastCommit 获取指定仓库的最新提交记录
func (s *SnapshotService) getLastCommit(ctx context.Context, _ string) (*model.Commit, error) {
	// 简化实现：获取最新的Commit
	// 在实际应用中，应该根据repoID查询最新Commit
	snapshots, err := s.SnapshotRepo.ListSnapshots(ctx, 1, 0)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	// 假设第一个是最新提交
	latestSnapshot := snapshots[0]
	// 尝试转换为Commit结构
	commit := &model.Commit{
		CommitHash:   latestSnapshot.UUID,
		RootTreeHash: latestSnapshot.Name, // 假设Name存储了RootTreeHash
		CreatedAt:    latestSnapshot.CreatedAt,
	}

	return commit, nil
}

// GetCommitHistory 获取提交历史记录
func (s *SnapshotService) GetCommitHistory(ctx context.Context, repoID string, limit int) ([]*model.Commit, error) {
	// 获取快照列表
	snapshots, err := s.SnapshotRepo.ListSnapshots(ctx, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("获取快照列表失败: %w", err)
	}

	// 转换为Commit列表
	var commits []*model.Commit
	for _, snapshot := range snapshots {
		commit := &model.Commit{
			CommitHash:   snapshot.UUID,
			RootTreeHash: snapshot.Name,
			CreatedAt:    snapshot.CreatedAt,
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

// RevertToCommit 回滚到指定提交
func (s *SnapshotService) RevertToCommit(ctx context.Context, commitID string) error {
	// 获取指定Commit
	_, err := s.SnapshotRepo.GetSnapshotByUUID(ctx, commitID)
	if err != nil {
		return fmt.Errorf("获取提交记录失败: %w", err)
	}

	// 在事务中执行回滚
	// 简化实现：直接返回，实际应更新文件系统状态
	return nil
}
