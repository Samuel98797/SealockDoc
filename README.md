# SealockDoc

SealockDoc 是一个高性能的网盘与文档协作系统，旨在通过内容寻址存储（CAS）和类似 Git 的版本控制机制，实现高效的数据去重、增量同步与协作编辑。

## 项目架构

### 整体架构
- **后端**: Go 语言实现，采用 Clean Architecture 和分层架构
- **前端**: React (Tailwind CSS) 或 Vue 3 (Vite)
- **数据库**: 
  - PostgreSQL（元数据存储）
  - Redis（缓存）
- **存储**: 本地存储 + 模拟 S3 接口
- **核心算法**: SHA-256 哈希 + 固定大小/可变大小块切分

### 后端架构

#### 目录结构
```
core-storage/                    # 核心服务模块
├── config/                      # 配置文件目录
│   ├── config.yaml              # 主配置文件
│   └── config.test.yaml         # 测试配置文件
├── model/                       # 数据模型定义
│   ├── cas_models.go            # 内容寻址存储相关模型
│   ├── models.go                # 主要数据模型定义
│   └── snapshot.go              # 快照相关模型
├── service/                     # 业务逻辑层
│   ├── file_service.go          # 文件服务主逻辑
│   ├── snapshot_service.go      # 快照服务
│   ├── sync_service.go          # 同步服务
│   └── upload_service.go        # 上传服务
├── storage/                     # 存储接口与实现
│   ├── interfaces.go            # 存储接口定义
│   ├── block_repository.go      # 块仓库接口
│   ├── file_repository.go       # 文件仓库接口
│   ├── gorm_repositories.go     # GORM仓库实现
│   ├── local_block_store.go     # 本地块存储实现
│   ├── cached_block_store.go    # 带缓存的块存储实现
│   ├── redis_cache.go           # Redis缓存实现
│   ├── factory.go               # 存储工厂
│   ├── mock_repositories.go     # Mock仓库实现（用于测试）
│   └── snapshot_repository.go   # 快照仓库实现
├── handler/                     # API处理器
│   └── upload_handler.go        # 上传相关API处理器
├── chunker/                     # 文件分块处理
│   └── chunker.go               # 分块器实现
├── test/                        # 测试文件目录
│   ├── main_test.go             # 集成测试
│   └── main_simple_test.go      # 简单测试
├── docs/                        # 文档目录
│   └── database/                # 数据库相关文档
├── main.go                      # 服务启动入口
├── go.mod                       # Go模块定义
├── go.sum                       # Go模块校验和
└── README.md                    # 项目说明文档
```

#### 架构模式
1. **Clean Architecture（清洁架构）**:
   - 依赖倒置原则：核心逻辑独立于框架和外部依赖
   - 分层结构：Model → Service → Handler

2. **分层架构**:
   - **Model层**: 定义数据结构和业务实体
   - **Service层**: 实现业务逻辑和数据处理
   - **Handler层**: 处理API请求和响应
   - **Storage层**: 管理数据持久化和存储

3. **设计模式**:
   - 策略模式：用于不同存储后端（如 Local vs S3 模拟）
   - 单例模式：数据库连接池（PostgreSQL, Redis）
   - 工厂模式：Block 存储实例化

#### 核心组件交互流程
1. 用户请求通过 Gin 路由进入 handler
2. 调用 file_service 处理业务逻辑
3. service 使用 GORM 操作 PostgreSQL 元数据
4. chunker 将文件切块，通过 local_block_store 写入本地块存储
5. Redis 缓存热点数据（如频繁访问的 Block Hash）

### 关键技术特性

1. **内容寻址存储（CAS）**:
   - 使用 SHA-256 哈希作为内容地址
   - 实现数据去重，节省存储空间
   - 确保数据完整性

2. **文件分块处理**:
   - 文件按块切分，支持大文件处理
   - 支持增量更新，降低带宽消耗
   - 块独立寻址，提高存储效率

3. **版本控制**:
   - 类似 Git 的提交历史
   - 支持版本回滚和对比
   - 提供清晰的变更追踪

4. **高性能存储**:
   - Redis 缓存热点数据
   - 支持本地存储和 S3 接口
   - 优化的块读写性能

## 快速开始

### 环境要求
- Go 1.19+
- Node.js 16+（前端）
- PostgreSQL 12+
- Redis 6+

### 运行项目

#### 后端
```bash
# 进入后端目录
cd core-storage

# 安装依赖
go mod download

# 构建项目
go build -o sealockdoc main.go

# 运行项目
go run main.go
```

> **注意**: 在Windows PowerShell中，请使用分号 `;` 连接命令：
> ```powershell
cd "core-storage"; go mod download; go build -o sealockdoc main.go; go run main.go
> ```

#### 配置说明
- **配置文件位置**: 数据库连接等配置信息位于 `core-storage/config/config.yaml`
- **环境变量覆盖**: 所有配置项支持通过环境变量覆盖，格式为 `CONFIG_SECTION_KEY`（例如 `DATABASE_URL`）
- **敏感信息处理**: 密码等敏感信息禁止硬编码，应通过以下方式注入：
  ```yaml
database:
  password: ${DB_PASSWORD}  # 从环境变量读取
```
- **配置示例**:
  ```yaml
database:
  host: localhost
  port: 5432
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  dbname: sealockdoc
  sslmode: disable
redis:
  addr: ${REDIS_ADDR:localhost:6379}  # 冒号后为默认值
```
- **推荐工具**: 使用 [Viper](https://github.com/spf13/viper) 库管理配置，支持自动绑定环境变量

## 核心功能

1. **文件上传与下载**:
   - 基于块的切分与重组
   - 支持大文件处理
   - 上传进度跟踪

2. **内容寻址存储与去重**:
   - 基于 SHA-256 的内容哈希寻址
   - 自动检测和去重相同内容
   - 节省存储空间

3. **版本提交与历史追踪**:
   - 类似 Git 的提交机制
   - 支持版本对比和回滚
   - 提供清晰的历史记录

4. **多人协作与变更同步**:
   - 实时协作编辑
   - 变更冲突检测与解决
   - 增量同步机制