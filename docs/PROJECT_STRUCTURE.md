# DataFusion Worker 项目结构说明

## 📁 目录结构

```
datafusion-worker/
├── cmd/                         # 命令行程序
│   └── worker/                  # Worker 主程序
│       └── main.go              # 程序入口
│
├── internal/                    # 内部包（不对外暴露）
│   ├── collector/              # 数据采集器模块
│   │   ├── collector.go        # 采集器接口定义
│   │   ├── rpa_collector.go    # RPA 采集器实现
│   │   └── api_collector.go    # API 采集器实现
│   ├── processor/              # 数据处理器模块
│   │   └── processor.go        # 数据清洗和转换
│   ├── storage/                # 数据存储模块
│   │   ├── storage.go          # 存储接口定义
│   │   ├── postgres_storage.go # PostgreSQL 存储实现
│   │   └── file_storage.go     # 文件存储实现
│   ├── database/               # 数据库操作模块
│   │   └── postgres.go         # PostgreSQL 客户端
│   ├── models/                 # 数据模型
│   │   └── task.go             # 任务相关模型定义
│   ├── config/                 # 配置管理
│   │   └── config.go           # 配置加载和解析
│   └── worker/                 # Worker 核心逻辑
│       └── worker.go           # 任务调度和执行
│
├── config/                      # 配置文件目录
│   └── worker.yaml             # Worker 配置文件
│
├── k8s/                        # Kubernetes 部署文件
│   ├── namespace.yaml          # 命名空间定义
│   ├── postgresql.yaml         # PostgreSQL 部署配置
│   ├── postgres-init-scripts.yaml # 数据库初始化脚本
│   ├── worker-config.yaml      # Worker 配置 ConfigMap
│   └── worker.yaml             # Worker 部署配置
│
├── scripts/                     # 脚本工具
│   ├── init_db.sql             # 数据库初始化 SQL
│   ├── insert_test_task.sql    # 测试任务插入 SQL
│   ├── quick_start.sh          # 快速启动脚本
│   ├── deploy-k8s.sh           # K8S 部署脚本
│   ├── verify-k8s.sh           # K8S 验证脚本
│   ├── update-k8s.sh           # K8S 更新脚本
│   └── debug-k8s.sh            # K8S 调试脚本
│
├── tests/                       # 测试文件
│   ├── test_simple.go          # 简单功能测试
│   ├── test_with_storage.go    # 完整流程测试
│   ├── unit/                   # 单元测试（待添加）
│   ├── integration/            # 集成测试（待添加）
│   ├── e2e/                    # 端到端测试（待添加）
│   └── README.md               # 测试说明文档
│
├── docs/                        # 文档中心
│   ├── README.md               # 文档索引
│   ├── QUICKSTART.md           # 快速开始指南
│   ├── GETTING_STARTED.md      # 详细入门指南
│   ├── WORKER_IMPLEMENTATION.md # Worker 实现说明
│   ├── K8S_QUICK_START.md      # K8S 快速部署
│   ├── K8S_DEPLOYMENT_GUIDE.md # K8S 完整部署指南
│   ├── QUICK_FIX.md            # 快速修复指南
│   ├── FIX_DUPLICATE_KEY_ISSUE.md # 主键冲突修复
│   └── ...                     # 其他文档
│
├── examples/                    # 示例代码
│   └── simple_test.md          # 测试示例说明
│
├── design/                      # 设计文档
│   ├── DataFusion技术方案设计.md
│   ├── DataFusion产品需求分析文档.md
│   ├── deploy/                 # 部署相关设计
│   └── diagrams/               # 架构图和流程图
│
├── data/                        # 数据目录（运行时生成）
│   └── test_output/            # 测试输出数据
│
├── bin/                         # 编译产物（运行时生成）
│   └── worker                  # Worker 可执行文件
│
├── go.mod                       # Go 模块定义
├── go.sum                       # Go 依赖锁定
├── Makefile                     # 构建脚本
├── Dockerfile                   # Docker 镜像构建文件
├── .gitignore                   # Git 忽略文件
├── README.md                    # 项目主文档
└── TODO.md                      # 待办事项
```

## 📦 核心模块说明

### 1. cmd/worker

**作用**: 程序入口

**职责**:
- 加载配置文件
- 创建 Worker 实例
- 启动 Worker
- 处理退出信号

**关键文件**:
- `main.go`: 程序入口，约 50 行代码

### 2. internal/collector

**作用**: 数据采集器模块

**职责**:
- 定义采集器接口
- 实现不同类型的采集器
- 从数据源获取原始数据

**关键文件**:
- `collector.go`: 采集器接口和工厂模式
- `rpa_collector.go`: RPA 采集器（基于 Chromedp）
- `api_collector.go`: API 采集器（基于 Resty）

**设计模式**: 工厂模式 + 策略模式

### 3. internal/processor

**作用**: 数据处理器模块

**职责**:
- 数据清洗（去除空格、HTML 标签等）
- 数据转换（字段映射、类型转换等）
- 数据验证

**关键文件**:
- `processor.go`: 数据处理逻辑

**支持的清洗规则**:
- trim: 去除空格
- remove_html: 移除 HTML 标签
- regex: 正则表达式替换
- lowercase/uppercase: 大小写转换

### 4. internal/storage

**作用**: 数据存储模块

**职责**:
- 定义存储接口
- 实现不同类型的存储
- 将处理后的数据持久化

**关键文件**:
- `storage.go`: 存储接口和工厂模式
- `postgres_storage.go`: PostgreSQL 存储
- `file_storage.go`: 文件存储（JSON 格式）

**设计模式**: 工厂模式 + 策略模式

### 5. internal/database

**作用**: 数据库操作模块

**职责**:
- 管理数据库连接
- 查询待执行任务
- 管理分布式锁
- 记录任务执行历史

**关键文件**:
- `postgres.go`: PostgreSQL 客户端

**核心功能**:
- 任务查询
- 分布式锁（PostgreSQL Advisory Lock）
- 执行记录管理

### 6. internal/models

**作用**: 数据模型定义

**职责**:
- 定义任务相关的数据结构
- 定义配置相关的数据结构

**关键文件**:
- `task.go`: 任务模型、配置模型、执行记录模型

### 7. internal/config

**作用**: 配置管理模块

**职责**:
- 加载 YAML 配置文件
- 解析配置
- 设置默认值

**关键文件**:
- `config.go`: 配置加载和解析

### 8. internal/worker

**作用**: Worker 核心逻辑

**职责**:
- 任务轮询
- 任务调度
- 任务执行
- 状态管理

**关键文件**:
- `worker.go`: Worker 主逻辑

**核心流程**:
1. 轮询数据库获取待执行任务
2. 尝试获取任务锁
3. 执行任务（采集 → 处理 → 存储）
4. 更新执行记录
5. 释放任务锁

## 🔄 数据流

```
1. Worker 启动
   ↓
2. 轮询数据库（每 30 秒）
   ↓
3. 获取待执行任务
   ↓
4. 尝试获取分布式锁
   ↓
5. 执行任务
   ├─ Collector: 采集数据
   ├─ Processor: 处理数据
   └─ Storage: 存储数据
   ↓
6. 更新执行记录
   ↓
7. 释放锁
   ↓
8. 返回步骤 2
```

## 📊 代码统计

| 模块 | 文件数 | 代码行数 | 说明 |
|------|--------|---------|------|
| cmd | 1 | ~50 | 程序入口 |
| collector | 3 | ~220 | 数据采集 |
| processor | 1 | ~120 | 数据处理 |
| storage | 3 | ~200 | 数据存储 |
| database | 1 | ~150 | 数据库操作 |
| models | 1 | ~120 | 数据模型 |
| config | 1 | ~70 | 配置管理 |
| worker | 1 | ~200 | Worker 核心 |
| **总计** | **12** | **~1,130** | |

## 🎯 设计原则

### 1. 模块化

每个模块职责单一，相互独立，易于测试和维护。

### 2. 接口抽象

使用接口定义核心功能，便于扩展和替换实现。

### 3. 工厂模式

使用工厂模式管理采集器和存储器，支持动态选择。

### 4. 配置驱动

通过配置文件控制行为，无需修改代码。

### 5. 错误处理

完善的错误处理和日志记录，便于问题排查。

## 📚 相关文档

- [README.md](../README.md) - 项目主文档
- [WORKER_IMPLEMENTATION.md](WORKER_IMPLEMENTATION.md) - 实现细节
- [tests/README.md](../tests/README.md) - 测试说明

---

**清晰的结构是项目成功的基础！** 📁
