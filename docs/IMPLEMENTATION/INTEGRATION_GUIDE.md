# Sealock 核心存储集成指南

## 📋 概述

本模块实现了完整的存储层集成，支持：
- **GORM + PostgreSQL**: 元数据持久化
- **Redis 缓存**: 热块加速
- **S3 存储**: 生产级云存储
- **灵活的存储栈工厂**: 快速切换存储后端

## 🏗️ 架构

```
┌─────────────────────────────────────┐
│     应用层 (API Service)              │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   FileService (业务逻辑层)            │
├───────────────────────────────────┤
│ • 上传文件（分块、存储、记录元数据）  │
│ • 下载文件（验证、拼接、完整性检查）  │
│ • 版本管理                          │
└──────────────┬──────────────────────┘
               │
        ┌──────┴────────┐
        │               │
┌───────▼──────┐  ┌────▼──────────┐
│ BlockStore   │  │ Repositories  │
│ (块存储)      │  │ (元数据)      │
├──────────────┤  └───────────────┘
│ • Local      │
│ • S3         │  ┌────────────────┐
│ • Cached     │  │ PostgreSQL DB  │
│   (Redis)    │  └────────────────┘
└──────────────┘
```

## 🚀 快速开始

### 1. 初始化数据库

```go
import "github.com/sealock/core-storage/storage"

// 初始化 PostgreSQL
db, err := storage.InitDB("host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable")
if err != nil {
    log.Fatal(err)
}
```

### 2. 创建存储栈

#### 开发环境（本地存储）
```go
cfg := storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "local",
}
stack, err := storage.InitializeStorage(cfg)
defer stack.Close()
```

#### 生产环境（S3 + Redis）
```go
cfg := storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "s3-cached",
    S3Config: &storage.S3Config{
        Region:    "us-east-1",
        Bucket:    "sealock-blocks",
        Prefix:    "blocks/",
        AccessKey: os.Getenv("AWS_ACCESS_KEY"),
        SecretKey: os.Getenv("AWS_SECRET_KEY"),
    },
    RedisAddr:   "localhost:6379",
    CacheExpiry: 24 * time.Hour,
}
stack, err := storage.InitializeStorage(cfg)
```

### 3. 使用文件服务

```go
import "github.com/sealock/core-storage/service"
import "github.com/sealock/core-storage/chunker"

fsChunker := chunker.NewFixedSizeChunker(8192)
fileSvc := service.NewFileService(
    stack.BlockStore,
    stack.FileRepository,
    stack.BlockRepository,
    fsChunker,
)

// 上传
file, err := fileSvc.UploadFile(ctx, "document.pdf", fileData)

// 下载
data, err := fileSvc.DownloadFile(ctx, file.Hash)
```

## 📚 主要组件

### BlockStore 接口
```go
type BlockStore interface {
    Put(ctx context.Context, data []byte) (hash string, err error)
    Get(ctx context.Context, hash string) (data []byte, err error)
    Exists(ctx context.Context, hash string) (bool, error)
    Delete(ctx context.Context, hash string) error
    GetSize(ctx context.Context, hash string) (int64, error)
}
```

### 存储实现

#### LocalBlockStore
- **使用场景**: 开发、测试
- **优点**: 零依赖、快速启动
- **缺点**: 单机、无持久化

#### RedisBlockCache
- **使用场景**: 加速热块访问
- **工作原理**: 
  1. 检查 Redis 缓存
  2. 未命中则从底层存储获取
  3. 自动写入缓存
- **特性**:
  - 透明缓存层
  - 可配置过期时间
  - 缓存失败不影响操作

```go
cachedStore, err := storage.NewRedisBlockCache(
    s3Store,
    "localhost:6379",
    24 * time.Hour,
)
```

### GORM 仓储实现

#### GormFileRepository
元数据存储：文件名、大小、块列表、hash

#### GormLibraryRepository  
库管理：创建、更新、列表、权限

#### GormLibraryVersionRepository
版本控制：提交、历史、回溯

#### GormBlockRepository
块元数据：引用计数、垃圾回收

## 🔄 存储栈类型

### 1. local
```
文件数据 → LocalBlockStore (内存)
         ↓
       PostgreSQL (元数据)
```
**用途**: 开发环境、单机测试

### 2. s3
```
文件数据 → S3 (持久化)
         ↓
       PostgreSQL (元数据)
```
**用途**: 生产环境（无缓存）

### 3. s3-cached ⭐ 推荐
```
文件数据 → Redis (热块) → S3 (冷块)
         ↓
       PostgreSQL (元数据)
```
**用途**: 生产环境（最优性能）

### 4. local-cached
```
文件数据 → LocalBlockStore (内存) + Redis (热块)
         ↓
       PostgreSQL (元数据)
```
**用途**: 开发环境（测试缓存逻辑）

## 🔐 安全性

### S3 认证
```go
// ✅ 推荐: IAM 角色（EC2/ECS）
// ✅ 推荐: 环境变量
os.Getenv("AWS_ACCESS_KEY_ID")
os.Getenv("AWS_SECRET_ACCESS_KEY")

// ⚠️ 避免: 硬编码凭证
```

### Redis 连接
```go
cfg := &redis.Options{
    Addr:     "localhost:6379",
    Password: os.Getenv("REDIS_PASSWORD"),
    TLSConfig: &tls.Config{...}, // 支持 TLS 连接
}
```

### PostgreSQL
```go
// 连接加密
"sslmode=require"

// 权限管理
GRANT SELECT, INSERT, UPDATE ON blocks TO app_user;
```

## 📊 性能优化

### Redis 缓存策略
```go
// 热块保留 24 小时
CacheExpiry: 24 * time.Hour

// 大文件自动分块
BlockSize: 8 * 1024 * 1024 // 8MB

// 并行批量操作
DeleteBatch() // 批量删除
```

### BlockStore 扩展
```go
// 自定义实现 BlockStore 接口
type BlockStore interface {
    Put(ctx context.Context, hash string, data []byte) error
    Get(ctx context.Context, hash string) ([]byte, error)
    Exists(ctx context.Context, hash string) (bool, error)
    Delete(ctx context.Context, hash string) error
    GetSize(ctx context.Context, hash string) (int64, error)
}

// 可集成 MinIO、阿里 OSS、Azure Blob Storage 等
```

### PostgreSQL 优化
```sql
-- 创建索引
CREATE INDEX idx_block_hash ON blocks(hash);
CREATE INDEX idx_file_library_id ON files(library_id);

-- 启用批量插入
INSERT INTO blocks (...) VALUES (...), (...), (...)
  ON CONFLICT (hash) DO UPDATE SET ref_count = blocks.ref_count + 1;
```

## 🔄 扩展方案

### 本地 → 云存储迁移

如需集成云存储（MinIO、阿里 OSS 等），遵循以下步骤：

1️⃣ **实现 BlockStore 接口**
```go
type CustomBlockStore struct {
    // 你的云存储客户端
}

func (s *CustomBlockStore) Put(ctx context.Context, hash string, data []byte) error {
    // 实现上传逻辑
    return nil
}

func (s *CustomBlockStore) Get(ctx context.Context, hash string) ([]byte, error) {
    // 实现下载逻辑
    return nil, nil
}
```

2️⃣ **在工厂中使用**
```go
// 保持旧的 local 栈运行
oldStack, _ := factory.CreateLocalStack()

// 使用自定义实现创建新栈
customStore := NewCustomBlockStore(config)
newStack := &storage.StorageStack{
    BlockStore: customStore,
    // ... 其他组件
}
```

3️⃣ **增量同步**
```go
// 对比块哈希，只上传缺失的块
blockList, _ := oldRepo.ListBlocks(ctx)
for _, block := range blockList {
    data, _ := oldStore.Get(ctx, block.Hash)
    newStore.Put(ctx, block.Hash, data)
}
```

3️⃣ **验证一致性**
```go
oldList, _ := oldStore.ListBlocks(ctx)
newList, _ := newStore.ListBlocks(ctx)
// 对比后切流
```

4️⃣ **灰度切流**
```go
// 10% 流量到新栈
if rand.Intn(100) < 10 {
    useS3Stack = true
}
```

## 🐛 故障排查

### Redis 连接失败
```go
// 检查 Redis 运行
redis-cli ping  // 应返回 PONG

// 检查网络
telnet localhost 6379
```

### S3 上传超时
```go
// 增加超时时间
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// 检查 S3 权限
aws s3api head-bucket --bucket sealock-blocks
```

### PostgreSQL 连接失败
```go
// 检查 DSN 格式
psql "host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable"

// 查看 GORM 日志
db := db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default})
```

## 📖 示例代码

查看 `main.go` 中的完整演示：
```bash
cd core-storage
go run main.go
```

## 🔗 相关文档

- [核心接口简述](../docs/核心接口简述.md)
- [项目架构图](../docs/项目架构图与逻辑描述.md)
- [数据库 Schema](../docs/数据库%20Schema%20提示词.md)
