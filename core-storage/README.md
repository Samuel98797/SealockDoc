# Sealock 核心存储层 (Phase 1: Core Storage Layer)

## 项目概述

本目录实现了 Sealock 文档协作系统的**第一阶段：核心存储层设计**，包括：

- **内容寻址存储 (CAS)** - 通过文件内容 SHA-256 哈希命名数据块
- **文件分块机制** - 支持固定大小和内容定义分块 (CDC)
- **全局去重** - 自动检测和复用相同内容的块
- **增量同步** - 只需传输变化的块，大幅降低带宽
- **完整性验证** - 块级别的哈希验证和重构

## 架构设计

### 核心概念

#### Block（数据块）
- 最小存储单位，由内容 SHA-256 哈希命名
- 大小：通常 4-64KB（可配置）
- 特点：**内容相同 = 哈希相同 = 存储唯一**（自动去重）

#### File（文件）
- 由多个 Block 哈希组成的列表
- 包含文件元数据（名称、大小、时间戳）
- 文件 Merkle 哈希 = 所有块哈希的聚合哈希

#### Library（库）
- 顶层容器，包含版本提交历史
- 类似 Git Repository，支持版本回溯
- 当前版本 HEAD 指向最新提交

#### LibraryVersion（版本提交）
- 代表某一时刻库的完整快照
- 包含：Commit ID, Root Hash, 时间戳, 父提交等
- 支持合并（Merge）和分支操作

### 分层架构（Clean Architecture）

```
┌─────────────────────────────────────────┐
│  Service Layer (业务逻辑)                │
│  - FileService: 文件上传/下载/同步      │
│  - LibraryService: 版本管理              │
└─────────────────────────────────────────┘
           ↓ 依赖
┌─────────────────────────────────────────┐
│  Storage & Repository Interfaces         │
│  - BlockStore: Block 存储抽象            │
│  - FileRepository: 文件元数据持久化      │
│  - LibraryRepository: 库版本持久化       │
└─────────────────────────────────────────┘
           ↓ 实现
┌─────────────────────────────────────────┐
│  Implementation Layer                    │
│  - LocalBlockStore: 本地磁盘存储         │
│  - LocalBlockStore: 本地块存储    │
│  - PostgreSQL Repositories: GORM 实现    │
└─────────────────────────────────────────┘
           ↓ 驱动
┌─────────────────────────────────────────┐
│  Data Layer                              │
│  - PostgreSQL (元数据)                   │
│  - Redis (缓存)                          │
│  - 对象存储 (块数据)                     │
└─────────────────────────────────────────┘
```

## 代码结构

```
core-storage/
├── model/
│   └── models.go          # 数据模型定义 (Block, File, Library, Version)
├── storage/
│   ├── interfaces.go      # 存储接口定义 (BlockStore, Repositories)
│   └── local_block_store.go # 本地 Block 存储实现
├── chunker/
│   └── chunker.go         # 文件分块器 (FixedSize, CDC)
├── service/
│   └── file_service.go    # 文件业务逻辑
├── main.go                # 演示代码
├── go.mod                 # Go 依赖配置
└── README.md              # 本文件
```

## 核心功能详解

### 1. 内容寻址存储 (CAS)

**原理**: 每个数据块由其 SHA-256 哈希命名，而不是顺序编号。

```go
hash := sha256.Sum256(blockData)
hashHex := hex.EncodeToString(hash[:])
blockStore.Put(ctx, blockData) // 返回 hash
```

**优势**:
- **自动去重**: 相同内容的块只存储一次
- **完整性校验**: 哈希本身就是校验值
- **秒传**: 检测到块已存在则跳过

### 2. 文件分块机制

#### 固定大小分块 (Fixed-Size Chunking)
```go
chunker := chunker.NewFixedSizeChunker(8192) // 8KB
blockHashes, _ := chunker.Chunk(fileData)
```

**特点**:
- 实现简单、速度快
- 块大小均匀，便于计划分配
- **缺点**: 文件中间修改会导致后续块全部重排

#### 内容定义分块 (CDC)
```go
chunker := chunker.NewCDCChunker(2048, 8192, 65536)
blockHashes, _ := chunker.Chunk(fileData)
```

**特点**:
- 基于内容特征点分块，而非固定位置
- 文件中间修改仅影响相邻块
- 大幅提升增量同步效率（推荐用于大文件）

### 3. 全局去重原理

```
文件 A (10MB)          文件 B (10MB)
├─ Block-001          ├─ Block-001  ← 相同内容，复用!
├─ Block-002          ├─ Block-003  ← 不同内容，新块
├─ Block-003          ├─ Block-002  ← 相同内容，复用!
└─ Block-004          └─ Block-005  ← 不同内容，新块

实际存储: 6 个块 (去重率 33%)，而非 8 个
```

### 4. 文件上传流程

```
用户选择文件
    ↓
分块 (Chunking)
    ↓
逐块存储到 BlockStore (CAS 自动去重)
    ↓
保存文件元数据 (块列表 + 文件哈希)
    ↓
返回文件指纹（可用于秒传检测）
```

**代码示例**:
```go
fileService := service.NewFileService(blockStore, fileRepo, blockRepo, chunker)
file, err := fileService.UploadFile(ctx, "document.pdf", fileData)
// file.Hash 是文件的 Merkle 指纹，可用于去重检测
```

### 5. 文件下载流程

```
用户请求下载
    ↓
查询文件元数据（获取块哈希列表）
    ↓
依次读取每个块
    ↓
拼接成完整文件
    ↓
可选：验证完整性（重新计算哈希）
```

**代码示例**:
```go
data, err := fileService.DownloadFile(ctx, fileHash)
// 自动从块存储中取回数据并拼接
```

### 6. 增量同步检测

```go
oldFileHashes := map[string]*File{...}  // 版本 A 的文件
newFileHashes := map[string]*File{...}  // 版本 B 的文件

changes := fileService.DetectChanges(ctx, oldFileHashes, newFileHashes)
// changes.Added: 新增文件
// changes.Modified: 修改文件
// changes.Deleted: 删除文件
```

## 使用示例

### 基础流程

```go
package main

import (
    "context"
    "github.com/sealock/core-storage/chunker"
    "github.com/sealock/core-storage/storage"
)

func main() {
    ctx := context.Background()

    // 1. 初始化存储层
    blockStore := storage.NewLocalBlockStore()
    chunker := chunker.NewFixedSizeChunker(8192)

    // 2. 上传文件
    fileData := []byte("Hello, Sealock!")
    hash, err := blockStore.Put(ctx, fileData)
    if err != nil {
        panic(err)
    }
    // hash = "abc123..." (SHA-256)

    // 3. 下载文件
    data, err := blockStore.Get(ctx, hash)
    if err != nil {
        panic(err)
    }
    // data = "Hello, Sealock!"

    // 4. 检查块完整性
    valid := chunker.VerifyChunk(fileData, hash)
    // valid = true
}
```

### 完整上传/下载

详见 `main.go` 中的演示代码，运行:

```bash
go run main.go
```

## 设计决策与权衡

### 为什么选择 SHA-256？

| 特性 | SHA-256 | SHA-1 | MD5 |
|------|---------|-------|-----|
| 安全性 | ✓ (NIST 推荐) | ✗ (已破解) | ✗ (已破解) |
| 碰撞率 | 1 in 2^256 | 可预测 | 可预测 |
| 性能 | 快速 | 更快 | 最快 |
| 采用度 | 广泛 | 遗留系统 | 遗留系统 |

### Block 大小选择

| 大小 | 适用场景 | 优缺点 |
|-----|---------|-------|
| 4KB | 小文件 / 精细去重 | 块数多，索引大 |
| 8KB | **推荐通用** | 平衡存储和索引 |
| 16KB | 大文件 | 块数少，但去重率可能下降 |
| 64KB | 视频流 | 快速传输，但对小文件浪费 |

### 固定 vs. CDC 分块

| 分块方式 | 优点 | 缺点 | 适用场景 |
|---------|------|------|--------|
| 固定大小 | 简单、快速 | 修改敏感 | 首次上传、冷备份 |
| CDC | 修改容错 | 复杂、慢 | 增量同步、版本控制 |

## 下一步：第二阶段（同步协议设计）

核心存储层完成后，下一步将实现：

1. **同步协议设计** (Sync Protocol)
   - 客户端与服务器的差异对比算法
   - Merkle Tree 树形对比（高效）
   - 冲突解决策略

2. **服务接口** (Service API)
   - RESTful API 或 gRPC
   - 用户认证、授权
   - 配额管理

3. **前端集成**
   - Web UI 文件浏览
   - 上传/下载界面
   - 版本历史查看

## 贡献指南

修改核心存储层时，请：

1. 遵循 **Clean Architecture** 原则：逻辑层与数据访问层分离
2. 为新的接口编写模拟实现（用于测试）
3. 更新 `docs/核心接口简述.md` 中的数据模型
4. 在 `docs/项目架构图与逻辑描述.md` 中说明影响范围

## 许可证

MIT License
