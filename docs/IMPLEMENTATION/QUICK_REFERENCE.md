# Sealock 存储快速参考卡

## 🚀 30 秒快速开始

```go
// 1. 初始化数据库
db, _ := storage.InitDB("host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable")

// 2. 选择存储栈（开发）
stack, _ := storage.InitializeStorage(storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "local",  // 或 "s3-cached"、"s3" 等
})
defer stack.Close()

// 3. 创建文件服务
fileSvc := service.NewFileService(
    stack.BlockStore,
    stack.FileRepository,
    stack.BlockRepository,
    chunker.NewFixedSizeChunker(8192),
)

// 4. 上传文件
file, _ := fileSvc.UploadFile(ctx, "document.pdf", fileData)

// 5. 下载文件
data, _ := fileSvc.DownloadFile(ctx, file.Hash)
```

## 📦 四种存储栈

```go
// 开发环境（最简单）
storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "local",
}

// 生产环境（推荐）⭐
storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "s3-cached",
    S3Config: &storage.S3Config{
        Region: "us-east-1",
        Bucket: "sealock-blocks",
        Prefix: "blocks/",
    },
    RedisAddr: "localhost:6379",
    CacheExpiry: 24 * time.Hour,
}

// 其他选项
// "s3" - S3 无缓存
// "local-cached" - 本地 + Redis 缓存
```

## 🔑 主要 API

### BlockStore 接口
```go
// 所有实现都支持这些方法
blockStore.Put(ctx, data)          // 上传块，返回 hash
blockStore.Get(ctx, hash)          // 下载块
blockStore.Exists(ctx, hash)       // 检查块是否存在
blockStore.Delete(ctx, hash)       // 删除块
blockStore.GetSize(ctx, hash)      // 获取块大小
```

### FileRepository 接口
```go
repo.CreateFile(ctx, file)         // 创建文件记录
repo.GetFileByHash(ctx, hash)      // 获取文件
repo.UpdateFile(ctx, file)         // 更新文件
repo.DeleteFile(ctx, fileID)       // 删除文件
```

### BlockRepository 接口
```go
repo.SaveBlockMetadata(ctx, block) // 保存块元数据
repo.GetBlockMetadata(ctx, hash)   // 获取块元数据
repo.IncrementRefCount(ctx, hash, delta)  // 更新引用计数
repo.ListOrphanBlocks(ctx)         // 列出孤立块（GC）
```

## 🗄️ PostgreSQL 快速启动

```bash
# Docker
docker run -d \
  -e POSTGRES_DB=sealock \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15-alpine

# 连接字符串
host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable
```

## 📚 Redis 快速启动

```bash
# Docker
docker run -d -p 6379:6379 redis:7-alpine

# 验证
redis-cli ping
# 返回: PONG
```

## ☁️ S3 配置

```go
storage.S3Config{
    Region:    "us-east-1",
    Bucket:    "sealock-blocks",
    Prefix:    "blocks/",
    AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
}

// MinIO（兼容存储）
storage.S3Config{
    Region:    "us-east-1",
    Bucket:    "sealock-blocks",
    Prefix:    "blocks/",
    Endpoint:  "http://localhost:9000",  // MinIO 地址
    UsePathStyle: true,                   // 重要！
}
```

## 🔍 常见操作

### 检查缓存命中率
```bash
redis-cli KEYS "block:*" | wc -l   # 缓存块数
redis-cli INFO stats               # Redis 统计信息
```

### 查看块引用计数
```sql
-- 所有块的引用计数分布
SELECT ref_count, COUNT(*) as count FROM blocks GROUP BY ref_count;

-- 找出孤立块（可以删除）
SELECT hash FROM blocks WHERE ref_count = 0;
```

### 检查存储大小
```sql
-- 块总大小
SELECT SUM(size) as total_size FROM blocks;

-- 各文件的大小
SELECT name, size FROM files ORDER BY size DESC LIMIT 10;
```

### 删除孤立块
```go
// 自动化 GC
orphans, _ := blockRepo.ListOrphanBlocks(ctx)
for _, hash := range orphans {
    blockStore.Delete(ctx, hash)
}
```

## 🚨 错误处理

```go
// 典型的错误处理模式
file, err := fileSvc.UploadFile(ctx, "test.txt", data)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // 块未找到
    } else if strings.Contains(err.Error(), "connection") {
        // 数据库连接问题
    }
    // 处理错误
}
```

## 💾 环境变量

```bash
# 基础
STORAGE_TYPE=s3-cached                    # local | s3 | s3-cached | local-cached
DATABASE_DSN=postgresql://...

# 缓存
REDIS_ADDR=localhost:6379
CACHE_EXPIRY=24h

# S3
S3_REGION=us-east-1
S3_BUCKET=sealock-blocks
S3_PREFIX=blocks/
AWS_ACCESS_KEY_ID=***
AWS_SECRET_ACCESS_KEY=***
```

## 📊 性能目标

| 操作 | 延迟 | 吞吐 | 缓存命中 |
|------|------|------|---------|
| 块上传 | 50-500ms | 1-10 MB/s | - |
| 块下载（缓存） | 1-10ms | 100+ MB/s | 100% |
| 块下载（S3） | 50-200ms | 10-50 MB/s | 0% |
| 元数据查询 | 10-50ms | - | - |

## 🐛 故障排查

| 问题 | 解决方案 |
|------|---------|
| Redis 连接失败 | `redis-cli ping` 检查；检查地址和密码 |
| S3 上传超时 | 增加超时时间；检查网络和 IAM 权限 |
| PostgreSQL 连接失败 | 检查 DSN 格式；验证数据库是否运行 |
| 缓存不工作 | 检查 Redis 连接；查看日志中的警告 |

## 📖 文档速查

| 主题 | 文件 |
|------|------|
| 完整集成指南 | [INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) |
| 环境配置 | [ENVIRONMENT_CONFIG.md](./ENVIRONMENT_CONFIG.md) |
| 架构设计 | [docs/项目架构图与逻辑描述.md](../docs/项目架构图与逻辑描述.md) |
| API 接口 | [docs/核心接口简述.md](../docs/核心接口简述.md) |
| 源代码 | [storage/](./storage/) |

## 🔗 有用的链接

- Redis Go: https://pkg.go.dev/github.com/redis/go-redis/v9
- GORM: https://gorm.io
- Redis: https://redis.io
- PostgreSQL: https://www.postgresql.org

---

**提示**: 保存本页面书签以快速参考！
