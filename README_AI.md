# Sealock Doc 项目定义 (Project Context)

## 项目简介
Sealock Doc 是一个基于 Go 和 React/Vue 的高性能网盘与文档协作系统。
核心理念：内容寻址存储 (CAS)、高效增量同步、类似 Git 的版本控制。

## 技术栈要求
- **Backend**: Go (使用 Gin 框架), GORM (数据库 ORM)
- **Frontend**: React (使用 Tailwind CSS) 或 Vue 3 (Vite)
- **Database**: PostgreSQL (元数据), Redis (缓存)
- **Storage**: 本地存储 + AWS S3 或 MinIO 兼容存储
- **Core Algorithm**: 使用 SHA-256 进行文件切块 (Chunking) 与去重。

## ✨ 最新进展 (v1.0.0 - 2026-01-07)

### 核心存储层已完成生产级集成！🎉

#### 已实现功能
- ✅ **GORM + PostgreSQL** - 元数据持久化
- ✅ **Redis 缓存层** - 热块加速访问
- ✅ **AWS S3 存储** - 生产级云存储
- ✅ **灵活的存储栈** - 支持 4 种预定义配置

#### 快速开始
```go
// 初始化存储栈（推荐生产配置）
stack, _ := storage.InitializeStorage(storage.StorageConfig{
    DatabaseDSN: "...",
    StorageType: "s3-cached",  // 或 "local", "s3", "local-cached"
    S3Config: &storage.S3Config{...},
    RedisAddr: "localhost:6379",
})

// 使用文件服务
fileSvc := service.NewFileService(...)
file, _ := fileSvc.UploadFile(ctx, "document.pdf", data)
```

#### 相关文档
📖 **快速参考**: [QUICK_REFERENCE.md](./core-storage/QUICK_REFERENCE.md)  
📖 **集成指南**: [INTEGRATION_GUIDE.md](./core-storage/INTEGRATION_GUIDE.md)  
📖 **环境配置**: [ENVIRONMENT_CONFIG.md](./core-storage/ENVIRONMENT_CONFIG.md)  
📖 **实现总结**: [IMPLEMENTATION_SUMMARY.md](./core-storage/IMPLEMENTATION_SUMMARY.md)  
📖 **文件清单**: [FILE_MANIFEST.md](./FILE_MANIFEST.md)  
📖 **变更日志**: [CHANGELOG.md](./CHANGELOG.md)

## 核心数据模型 (Data Structure)
1. **Block**: 最小存储单位，由 SHA-256 Hash 命名（内容寻址）。
2. **File**: 由一系列 Block ID 组成的列表，支持 Merkle 树验证。
3. **Library/Repo**: 顶层容器，包含版本提交记录 (Commits)。

## AI 开发规范
- 后端代码需遵循 Clean Architecture，逻辑层与数据访问层分离。
  - ✅ **BlockStore** - 块存储接口（支持多种实现）
  - ✅ **Repository** - 数据访问层（GORM 实现）
  - ✅ **Service** - 业务逻辑层（FileService 等）
- 前端组件需保持原子化，使用 Hooks 管理状态。
- 所有接口必须符合 RESTful 规范，并包含详细的错误处理。

## 📂 项目结构

```
SealockDoc/
├── README_AI.md (本文件)
├── CHANGELOG.md ✨ v1.0.0 变更日志
├── FILE_MANIFEST.md ✨ 文件清单
│
├── core-storage/ ← 核心存储模块（已完成 v1.0.0）
│   ├── storage/
│   │   ├── interfaces.go (已有)
│   │   ├── local_block_store.go (已有)
│   │   ├── gorm_repositories.go ✨ 新增
│   │   ├── redis_cache.go ✨ 新增
│   │   └── factory.go ✨ 新增
│   │
│   ├── main.go (已重写为生产级演示)
│   ├── go.mod (已更新依赖)
│   ├── QUICK_REFERENCE.md ✨ 快速参考
│   ├── INTEGRATION_GUIDE.md ✨ 集成指南
│   ├── ENVIRONMENT_CONFIG.md ✨ 环境配置
│   └── IMPLEMENTATION_SUMMARY.md ✨ 实现总结
│
├── docs/
│   ├── 核心接口简述.md (已更新)
│   ├── 项目架构图与逻辑描述.md (已更新)
│   ├── 产品功能清单.md
│   └── ...
```

## 🚀 快速开始（基于 v1.0.0）

### 1. 启动完整的开发环境
```bash
cd core-storage
docker-compose up -d  # 启动 PostgreSQL + Redis + MinIO

# 查看服务是否就绪
docker-compose logs -f
```

### 2. 运行演示代码
```bash
go run main.go
```

### 3. 开始开发
- 参考 [QUICK_REFERENCE.md](./core-storage/QUICK_REFERENCE.md) 了解 API
- 参考 [INTEGRATION_GUIDE.md](./core-storage/INTEGRATION_GUIDE.md) 学习集成
- 查看 main.go 中的完整示例

## 📚 文档导航

### 核心概念
- [项目架构图与逻辑描述](./docs/项目架构图与逻辑描述.md) - 整体设计
- [核心接口简述](./docs/核心接口简述.md) - API 定义

### 存储实现
- [QUICK_REFERENCE.md](./core-storage/QUICK_REFERENCE.md) - 30 秒速查表
- [INTEGRATION_GUIDE.md](./core-storage/INTEGRATION_GUIDE.md) - 完整集成指南
- [ENVIRONMENT_CONFIG.md](./core-storage/ENVIRONMENT_CONFIG.md) - Docker + 配置

### 项目信息
- [CHANGELOG.md](./CHANGELOG.md) - v1.0.0 变更说明
- [FILE_MANIFEST.md](./FILE_MANIFEST.md) - 文件清单
- [IMPLEMENTATION_SUMMARY.md](./core-storage/IMPLEMENTATION_SUMMARY.md) - 实现总结

## 🎯 存储栈选择指南

### 开发环境
```go
StorageType: "local"  // 最简单，零依赖
```

### 测试环境
```go
StorageType: "local-cached"  // 本地 + Redis，测试缓存
```

### 生产环境 ⭐ 推荐
```go
StorageType: "s3-cached"  // S3 + Redis，最优性能
```

详见 [INTEGRATION_GUIDE.md 的存储栈类型部分](./core-storage/INTEGRATION_GUIDE.md#-存储栈类型)

## 🔧 主要技术

### 已实现
- ✅ GORM ORM (PostgreSQL)
- ✅ Redis 缓存
- ✅ AWS S3 / MinIO
- ✅ 内容寻址存储 (CAS)
- ✅ 块引用计数和垃圾回收

### 开发中
- 🔄 API 服务层 (Gin)
- 🔄 前端界面 (React/Vue)
- 🔄 用户认证
- 🔄 权限管理

### 后续计划
- ⏳ 块压缩和加密
- ⏳ P2P 传输
- ⏳ 多云支持
- ⏳ WebDAV 协议