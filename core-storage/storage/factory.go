package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sealock/core-storage/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// StorageFactory 存储工厂，用于创建各种存储后端组合
type StorageFactory struct {
	db *gorm.DB
}

// NewStorageFactory 创建存储工厂
func NewStorageFactory(db *gorm.DB) *StorageFactory {
	return &StorageFactory{db: db}
}

// StorageStack 完整的存储栈配置
type StorageStack struct {
	BlockStore         BlockStore
	FileRepository     FileRepository
	LibraryRepository  LibraryRepository
	LibraryVersionRepo LibraryVersionRepository
	BlockRepository    BlockRepository
	SnapshotRepository SnapshotRepository
	CloseFunc          func() error // 清理函数
}

// CreateLocalStack 创建本地存储栈（开发环境）
// 使用：本地内存块存储 + GORM PostgreSQL 元数据
func (sf *StorageFactory) CreateLocalStack() (*StorageStack, error) {
	blockStore := NewLocalBlockStore()
	fileRepo := NewFileRepository(sf.db)  // 使用接口实现
	libRepo := NewGormLibraryRepository(sf.db)
	libVersionRepo := NewGormLibraryVersionRepository(sf.db)
	blockRepo := NewBlockRepository(sf.db)  // 使用接口实现
	snapshotRepo := NewSnapshotRepository(sf.db)

	return &StorageStack{
		BlockStore:         blockStore,
		FileRepository:     fileRepo,
		LibraryRepository:  libRepo,
		LibraryVersionRepo: libVersionRepo,
		BlockRepository:    blockRepo,
		SnapshotRepository: snapshotRepo,
	}, nil
}

// CreateCachedLocalStack 创建带缓存的本地存储栈（开发环境+缓存测试）
// 使用：本地块存储 + Redis 缓存 + GORM PostgreSQL 元数据
func (sf *StorageFactory) CreateCachedLocalStack(
	redisAddr string,
	cacheExpiry time.Duration,
) (*StorageStack, error) {
	localStore := NewLocalBlockStore()

	// 包装 Redis 缓存层
	cachedStore, err := NewRedisBlockCache(localStore, redisAddr, cacheExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis cache: %w", err)
	}

	fileRepo := NewFileRepository(sf.db)  // 使用接口实现
	libRepo := NewGormLibraryRepository(sf.db)
	libVersionRepo := NewGormLibraryVersionRepository(sf.db)
	blockRepo := NewBlockRepository(sf.db)  // 使用接口实现
	snapshotRepo := NewSnapshotRepository(sf.db)

	return &StorageStack{
		BlockStore:         cachedStore,
		FileRepository:     fileRepo,
		LibraryRepository:  libRepo,
		LibraryVersionRepo: libVersionRepo,
		BlockRepository:    blockRepo,
		SnapshotRepository: snapshotRepo,
		CloseFunc: func() error {
		return cachedStore.Close()
		},
	}, nil
}

// StorageConfig 统一存储配置
type StorageConfig struct {
	// 数据库配置
	DatabaseDSN string

	// 存储类型: "local", "local-cached"
	StorageType string

	// Redis 配置（当 StorageType 为 "local-cached" 时需要）
	RedisAddr   string
	CacheExpiry time.Duration
}

// InitializeStorage 根据配置初始化完整的存储栈
func InitializeStorage(cfg StorageConfig) (*StorageStack, error) {
	// 初始化数据库
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移模式
	err = db.AutoMigrate(&model.File{}, &model.Block{}, &model.Library{}, &model.LibraryVersion{}, &model.Snapshot{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	factory := NewStorageFactory(db)

	// 根据存储类型创建相应的栈
	switch cfg.StorageType {
	case "local":
		return factory.CreateLocalStack()

	case "local-cached":
		if cfg.RedisAddr == "" {
			return nil, fmt.Errorf("Redis address required for local-cached storage type")
		}
		if cfg.CacheExpiry == 0 {
			cfg.CacheExpiry = 24 * time.Hour
		}
		
		// 初始化Redis客户端
		redisClient := redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr,
		})
		
		// 测试Redis连接
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			return nil, fmt.Errorf("failed to connect Redis: %w", err)
		}
		
		localStore := NewLocalBlockStore()
		cachedStore := NewCachedBlockStore(localStore, redisClient, cfg.CacheExpiry)
		
		fileRepo := NewFileRepository(db)  // 使用接口实现
		libRepo := NewGormLibraryRepository(db)
		libVersionRepo := NewGormLibraryVersionRepository(db)
		blockRepo := NewBlockRepository(db)  // 使用接口实现
		snapshotRepo := NewSnapshotRepository(db)

		return &StorageStack{
			BlockStore:         cachedStore,
			FileRepository:     fileRepo,
			LibraryRepository:  libRepo,
			LibraryVersionRepo: libVersionRepo,
			BlockRepository:    blockRepo,
			SnapshotRepository: snapshotRepo,
			CloseFunc: func() error {
				return redisClient.Close()
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}

// Close 优雅关闭存储栈
func (s *StorageStack) Close() error {
	if s.CloseFunc != nil {
		return s.CloseFunc()
	}
	return nil
}