# 开发环境配置示例

## PostgreSQL 配置

### Docker 快速启动
```bash
docker run --name sealock-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=sealock \
  -p 5432:5432 \
  postgres:15-alpine
```

### 连接字符串
```
# 本地开发
host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable

# Docker 容器内
host=sealock-postgres user=postgres password=postgres dbname=sealock port=5432 sslmode=disable

# 生产环境（AWS RDS）
host=sealock.xxxxx.rds.amazonaws.com user=postgres password=<strong-password> dbname=sealock port=5432 sslmode=require
```

## Redis 配置

### Docker 快速启动
```bash
docker run --name sealock-redis \
  -p 6379:6379 \
  redis:7-alpine
```

### 连接字符串
```
# 本地开发
localhost:6379

# Docker 容器内
sealock-redis:6379

# 生产环境（带密码）
user:password@redis-cluster.xxxxx.cache.amazonaws.com:6379
```

### Redis 集群配置（高可用）
```go
import "github.com/redis/go-redis/v9"

// 集群模式
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{
        "redis-1:6379",
        "redis-2:6379",
        "redis-3:6379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
})

// 单例 + 哨兵模式
rdb := redis.NewFailoverClient(&redis.FailoverOptions{
    MasterName: "mymaster",
    SentinelAddrs: []string{
        "sentinel-1:26379",
        "sentinel-2:26379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
})
```

## S3 / MinIO 配置

### AWS S3
```bash
# 设置凭证
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_DEFAULT_REGION=us-east-1

# 或使用 ~/.aws/credentials
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1
```

### MinIO（S3 兼容存储）
```bash
# Docker 快速启动
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /minio_data --console-address ":9001"

# 创建 bucket
mc alias set myminio http://localhost:9000 minioadmin minioadmin
mc mb myminio/sealock-blocks
```

### 配置代码
```go
s3Config := storage.S3Config{
    Region:       "us-east-1",
    Bucket:       "sealock-blocks",
    Prefix:       "blocks/",
    AccessKey:    os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
    
    // MinIO 配置
    Endpoint:     "http://localhost:9000", // MinIO 端点
    UsePathStyle: true,                    // MinIO 需要
}
```

## 环境变量配置文件

### .env.development
```bash
# 数据库
DATABASE_DSN=host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
CACHE_EXPIRY=24h

# 存储类型: local | s3 | s3-cached | local-cached
STORAGE_TYPE=local-cached

# S3 配置（可选）
S3_REGION=us-east-1
S3_BUCKET=sealock-blocks-dev
S3_PREFIX=blocks/
S3_USE_PATH_STYLE=false
```

### .env.production
```bash
# 数据库（RDS）
DATABASE_DSN=host=sealock.xxxxx.rds.amazonaws.com user=postgres password=<strong-password> dbname=sealock port=5432 sslmode=require

# Redis（ElastiCache）
REDIS_ADDR=sealock-redis.xxxxx.cache.amazonaws.com:6379
REDIS_PASSWORD=<redis-password>
CACHE_EXPIRY=24h

# 存储类型
STORAGE_TYPE=s3-cached

# S3（生产 bucket）
S3_REGION=us-east-1
S3_BUCKET=sealock-blocks-prod
S3_PREFIX=blocks/
S3_USE_PATH_STYLE=false

# AWS 凭证（从 IAM 角色获取，不需要硬编码）
AWS_ACCESS_KEY_ID=<from-iam-role>
AWS_SECRET_ACCESS_KEY=<from-iam-role>
```

## 初始化脚本

### Go 代码中读取环境变量
```go
import (
    "os"
    "time"
    "github.com/sealock/core-storage/storage"
)

func initStorageFromEnv() (*storage.StorageStack, error) {
    cfg := storage.StorageConfig{
        DatabaseDSN: os.Getenv("DATABASE_DSN"),
        StorageType: os.Getenv("STORAGE_TYPE"),
        RedisAddr:   os.Getenv("REDIS_ADDR"),
        CacheExpiry: parseDuration(os.Getenv("CACHE_EXPIRY")),
    }

    if cfg.StorageType == "s3" || cfg.StorageType == "s3-cached" {
        cfg.S3Config = &storage.S3Config{
            Region:       os.Getenv("S3_REGION"),
            Bucket:       os.Getenv("S3_BUCKET"),
            Prefix:       os.Getenv("S3_PREFIX"),
            AccessKey:    os.Getenv("AWS_ACCESS_KEY_ID"),
            SecretKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
            UsePathStyle: os.Getenv("S3_USE_PATH_STYLE") == "true",
        }
    }

    return storage.InitializeStorage(cfg)
}

func parseDuration(s string) time.Duration {
    d, _ := time.ParseDuration(s)
    return d
}
```

## Docker Compose 配置

### docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: sealock
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/minio_data
    command: server /minio_data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  app:
    build: .
    environment:
      DATABASE_DSN: host=postgres user=postgres password=postgres dbname=sealock port=5432 sslmode=disable
      REDIS_ADDR: redis:6379
      STORAGE_TYPE: s3-cached
      S3_REGION: us-east-1
      S3_BUCKET: sealock-blocks
      S3_PREFIX: blocks/
      S3_USE_PATH_STYLE: "true"
      AWS_ACCESS_KEY_ID: minioadmin
      AWS_SECRET_ACCESS_KEY: minioadmin
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### 启动完整开发环境
```bash
docker-compose up -d

# 创建 MinIO bucket
docker-compose exec minio mc alias set myminio http://localhost:9000 minioadmin minioadmin
docker-compose exec minio mc mb myminio/sealock-blocks

# 查看日志
docker-compose logs -f app
```

## 验证配置

### 检查 PostgreSQL
```bash
psql "host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable"

# 应显示表
\dt

# 检查块表
SELECT COUNT(*) FROM blocks;
SELECT COUNT(*) FROM files;
SELECT COUNT(*) FROM libraries;
```

### 检查 Redis
```bash
redis-cli ping
# 应返回 PONG

redis-cli KEYS "block:*"
# 应显示缓存的块
```

### 检查 S3/MinIO
```bash
# AWS CLI
aws s3 ls s3://sealock-blocks/blocks/ --recursive

# MinIO CLI
mc ls myminio/sealock-blocks/blocks/
```

## 常见问题

### Q: 如何在生产环境中使用 IAM 角色而不是凭证？
**A:** 
```go
// 不设置 AccessKey/SecretKey，AWS SDK 自动从 IAM 角色读取
cfg := storage.S3Config{
    Region: "us-east-1",
    Bucket: "sealock-blocks",
    // AccessKey 和 SecretKey 留空
}
```

### Q: Redis 有密码保护应该怎么配置？
**A:**
```go
redisClient := redis.NewClient(&redis.Options{
    Addr:     "redis.example.com:6379",
    Password: os.Getenv("REDIS_PASSWORD"),
})
```

### Q: 如何监控缓存命中率？
**A:**
```bash
# Redis 命令
redis-cli INFO stats

# 查看缓存的块数
redis-cli KEYS "block:*" | wc -l
```
