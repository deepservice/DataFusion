# DataFusion v2.0

DataFusion 是一个完整的数据采集和处理系统，包含控制面（API Server）和数据面（Worker）两大组件。

**🎉 控制面 + 数据面全部完成！系统生产就绪！**

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                   控制面 (Control Plane)                  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  API Server - RESTful API 管理服务                │  │
│  │  - 任务管理  - 数据源管理  - 清洗规则管理         │  │
│  │  - 执行历史  - 统计信息    - 系统配置             │  │
│  └──────────────────────────────────────────────────┘  │
│                          ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  PostgreSQL - 控制面数据库                         │  │
│  │  - 任务配置  - 数据源  - 规则  - 执行记录         │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   数据面 (Data Plane)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ Worker 1 │  │ Worker 2 │  │ Worker N │             │
│  │ RPA采集  │  │ API采集  │  │ DB采集   │             │
│  └──────────┘  └──────────┘  └──────────┘             │
│                          ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  数据存储 - PostgreSQL / MongoDB / File           │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## ✨ 功能特性

### 控制面 (API Server)
- ✅ **任务管理** - 创建、更新、删除、启动、停止采集任务
- ✅ **数据源管理** - 管理 Web、API、Database 数据源
- ✅ **清洗规则管理** - 配置和复用数据清洗规则
- ✅ **执行历史** - 查看任务执行记录和统计
- ✅ **系统监控** - 实时监控系统状态和性能
- ✅ **RESTful API** - 完整的 REST API 接口
- ✅ **健康检查** - /healthz 和 /readyz 端点

### 数据面 (Worker)

#### 数据采集 (3 种)
- ✅ **Web RPA 采集器** - 基于 Chromium 的网页数据抓取
- ✅ **API 采集器** - REST API 数据采集
- ✅ **数据库采集器** - MySQL + PostgreSQL 数据采集

### 数据处理 (18 种)
- ✅ **基础清洗** (5 种) - trim, remove_html, regex, lowercase, uppercase
- ✅ **增强清洗** (10 种) - date_format, number_format, email_validate, phone_format, url_normalize, etc.
- ✅ **数据去重** (3 种) - content_hash, field_based, time_window

### 数据存储 (3 种)
- ✅ **PostgreSQL** - 关系型数据库存储
- ✅ **MongoDB** - 文档数据库存储
- ✅ **File** - 文件存储（JSON/CSV）

### 运维功能 (7 项)
- ✅ **错误重试** - 指数退避，最大 3 次重试
- ✅ **超时控制** - 任务级别超时，默认 5 分钟
- ✅ **健康检查** - /healthz, /readyz 端点
- ✅ **优雅关闭** - 等待任务完成，30 秒超时
- ✅ **监控指标** - 28 个 Prometheus 指标
- ✅ **结构化日志** - JSON 格式，上下文追踪
- ✅ **单元测试** - 19 个测试，~70% 覆盖率

### 监控和告警
- ✅ **Prometheus 指标** - 28 个业务指标
- ✅ **Grafana Dashboard** - 14 个可视化面板
- ✅ **告警规则** - 20+ 条智能告警规则

## 快速开始

### 方式 1: 使用 Docker（推荐）

```bash
# 1. 启动 PostgreSQL 容器
docker run -d --name datafusion-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 postgres:14

# 2. 初始化数据库
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_control;"
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_data;"
docker exec -i datafusion-postgres psql -U postgres -d datafusion_control < scripts/init_control_db.sql

# 3. 启动 API Server
go build -o bin/api-server ./cmd/api-server
./bin/api-server

# 4. 测试 API
curl http://localhost:8081/healthz
curl http://localhost:8081/api/v1/tasks

# 5. 运行完整测试
./tests/test_api_server.sh
```

### 方式 2: 使用本地 PostgreSQL

```bash
# 1. 初始化数据库
createdb datafusion_control
psql -U postgres -d datafusion_control -f scripts/init_control_db.sql

# 2. 启动 API Server
go build -o bin/api-server ./cmd/api-server
./bin/api-server

# 3. 测试 API
curl http://localhost:8081/healthz

# 4. 运行完整测试
./tests/test_api_server.sh
```

### 方式 3: 启动 Worker

### 1. 环境准备

**必需：**
- Go 1.21+
- PostgreSQL 12+
- Chromium（用于 RPA 采集）

**可选：**
- Docker & Docker Compose

### 2. 安装依赖

```bash
# 下载 Go 依赖
make deps

# 或者
go mod download
```

### 3. 初始化数据库

```bash
# 创建数据库和表结构
make init-db

# 或者手动执行
psql -U postgres -f scripts/init_db.sql
```

### 4. 配置 Worker

编辑 `config/worker.yaml`：

```yaml
worker_type: "web-rpa"  # 或 "api", "database"
poll_interval: 30s

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  database: "datafusion_control"
  ssl_mode: "disable"

storage:
  type: "postgresql"
  database:
    host: "localhost"
    port: 5432
    user: "postgres"
    password: "postgres"
    database: "datafusion_data"
    ssl_mode: "disable"
```
  type: "postgresql"
  database:
    host: "localhost"
    port: 5432
    user: "datafusion"
    password: "datafusion123"
    database: "datafusion_data"
    ssl_mode: "disable"
```

### 5. 插入测试任务

```bash
# 插入示例采集任务
make insert-test-task

# 或者手动执行
psql -U postgres -f scripts/insert_test_task.sql
```

### 6. 运行 Worker

```bash
# 方式 1: 直接运行
make run

# 方式 2: 编译后运行
make build
./bin/worker -config config/worker.yaml

# 方式 3: 使用 Docker
make docker-build
docker run -v $(pwd)/config:/app/config datafusion-worker:latest
```

## 项目结构

```
datafusion/
├── cmd/
│   ├── api-server/              # API Server 主程序入口
│   │   └── main.go
│   └── worker/                  # Worker 主程序入口
│       └── main.go
├── internal/                    # 内部包（核心业务逻辑）
│   ├── api/                    # API Server 处理器
│   │   ├── router.go           # 路由注册
│   │   ├── middleware.go       # 中间件
│   │   ├── health.go           # 健康检查
│   │   ├── task_handler.go     # 任务管理
│   │   ├── datasource_handler.go # 数据源管理
│   │   ├── cleaning_rule_handler.go # 清洗规则管理
│   │   ├── execution_handler.go # 执行历史
│   │   └── stats_handler.go    # 统计信息
│   ├── collector/              # 数据采集器
│   │   ├── collector.go        # 采集器接口
│   │   ├── rpa_collector.go    # RPA 采集器
│   │   ├── api_collector.go    # API 采集器
│   │   └── db_collector.go     # 数据库采集器
│   ├── processor/              # 数据处理器
│   │   ├── processor.go        # 数据清洗和转换
│   │   ├── enhanced_cleaner.go # 增强清洗规则
│   │   └── deduplicator.go     # 数据去重
│   ├── storage/                # 数据存储
│   │   ├── storage.go          # 存储接口
│   │   ├── postgres_storage.go # PostgreSQL 存储
│   │   ├── file_storage.go     # 文件存储
│   │   └── mongodb/            # MongoDB 存储
│   │       ├── config.go
│   │       ├── pool.go
│   │       └── mongodb_storage.go
│   ├── database/               # 数据库操作
│   │   └── postgres.go         # PostgreSQL 客户端
│   ├── models/                 # 数据模型
│   │   └── task.go             # 任务模型
│   ├── config/                 # 配置管理
│   │   ├── config.go           # Worker 配置
│   │   └── api_config.go       # API Server 配置
│   ├── logger/                 # 日志管理
│   │   └── logger.go           # 结构化日志
│   ├── metrics/                # 监控指标
│   │   └── metrics.go          # Prometheus 指标
│   ├── health/                 # 健康检查
│   │   └── health.go           # 健康检查处理
│   └── worker/                 # Worker 核心逻辑
│       ├── worker.go           # 任务调度和执行
│       └── retry.go            # 重试机制
├── config/                      # 配置文件
│   ├── api-server.yaml         # API Server 配置
│   └── worker.yaml             # Worker 配置
├── k8s/                        # Kubernetes 部署文件
│   ├── namespace.yaml          # 命名空间
│   ├── postgresql.yaml         # PostgreSQL 部署
│   ├── postgres-init-scripts.yaml # 数据库初始化
│   ├── api-server-deployment.yaml # API Server 部署
│   ├── worker-config.yaml      # Worker 配置
│   ├── worker.yaml             # Worker 部署
│   └── monitoring/             # 监控配置
│       ├── grafana-dashboard.json
│       └── prometheus-rules.yaml
├── scripts/                     # 脚本工具
│   ├── init_db.sql             # Worker 数据库初始化
│   ├── init_control_db.sql     # 控制面数据库初始化
│   ├── insert_test_task.sql    # 测试任务
│   └── quick_start.sh          # 快速启动
├── tests/                       # 测试文件
│   ├── unit/                   # 单元测试
│   │   ├── collector_test.go
│   │   ├── processor_test.go
│   │   └── storage_test.go
│   ├── test_simple.go          # 简单测试
│   ├── test_with_storage.go    # 完整流程测试
│   └── README.md               # 测试说明
├── test_api_server.sh          # API Server 测试脚本
├── test_database_collector.go  # 数据库采集器测试
├── test_mongodb_and_dedup.go   # MongoDB 和去重测试
├── docs/                        # 文档中心
│   ├── README.md               # 文档索引
│   ├── QUICKSTART.md           # 快速开始
│   ├── K8S_DEPLOYMENT_GUIDE.md # K8S 部署指南
│   └── ...                     # 其他文档
├── examples/                    # 示例代码
│   └── simple_test.md          # 测试示例
├── design/                      # 设计文档
│   ├── DataFusion技术方案设计.md
│   └── DataFusion产品需求分析文档.md
├── go.mod                       # Go 模块定义
├── Makefile                     # 构建脚本
├── Dockerfile                   # Worker Docker 镜像
├── Dockerfile.api-server        # API Server Docker 镜像
├── deploy-api-server.sh         # API Server 部署脚本
├── deploy-k8s-worker.sh         # Worker 部署脚本
├── README.md                    # 项目主文档（本文档）
├── FINAL_CHECKLIST.md           # 最终检查清单
└── TODO.md                      # 待办事项
```

> 📚 **文档说明**：所有详细文档已移至 [docs/](docs/) 目录，请查看 [docs/README.md](docs/README.md) 获取完整文档索引。

## 使用示例

### 创建 RPA 采集任务

```sql
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '新闻文章采集',
    'web-rpa',
    'enabled',
    '0 */1 * * *',  -- 每小时执行
    NOW(),
    1,
    '{
        "data_source": {
            "type": "web-rpa",
            "url": "https://example.com/news",
            "selectors": {
                "_list": ".article-item",
                "title": ".article-title",
                "content": ".article-content"
            }
        },
        "processor": {
            "cleaning_rules": [
                {"field": "title", "type": "trim"},
                {"field": "content", "type": "remove_html"}
            ]
        },
        "storage": {
            "target": "postgresql",
            "table": "articles",
            "mapping": {"title": "title", "content": "content"}
        }
    }'
);
```

### 创建 API 采集任务

```sql
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    'API数据采集',
    'api',
    'enabled',
    '*/30 * * * *',  -- 每30分钟执行
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://api.example.com/data",
            "method": "GET",
            "headers": {"Authorization": "Bearer TOKEN"},
            "selectors": {
                "_data_path": "data.items",
                "id": "id",
                "name": "name"
            }
        },
        "processor": {
            "cleaning_rules": []
        },
        "storage": {
            "target": "file",
            "database": "exports",
            "table": "api_data"
        }
    }'
);
```

## 数据清洗规则

支持的清洗规则类型：

- `trim`: 去除首尾空格
- `remove_html`: 移除 HTML 标签
- `regex`: 正则表达式替换
- `lowercase`: 转换为小写
- `uppercase`: 转换为大写

示例：

```json
{
    "cleaning_rules": [
        {
            "field": "title",
            "type": "trim"
        },
        {
            "field": "content",
            "type": "regex",
            "pattern": "\\s+",
            "replacement": " "
        }
    ]
}
```

## 监控和日志

Worker 会输出详细的执行日志：

```
2025-12-04 10:00:00 Worker 启动: worker-1234, 类型: web-rpa
2025-12-04 10:00:30 发现 2 个待执行任务
2025-12-04 10:00:30 成功锁定任务 新闻文章采集 (ID: 1)，开始执行
2025-12-04 10:00:31 开始 RPA 采集: https://example.com/news
2025-12-04 10:00:35 页面加载成功，开始解析数据
2025-12-04 10:00:36 解析完成，提取到 50 条数据
2025-12-04 10:00:36 开始数据处理，共 50 条数据
2025-12-04 10:00:37 数据处理完成，有效数据 48 条
2025-12-04 10:00:37 开始存储数据到 PostgreSQL，表: articles，数据量: 48
2025-12-04 10:00:38 数据存储完成，成功: 48 条，失败: 0 条
2025-12-04 10:00:38 任务执行完成: 新闻文章采集, 耗时: 8s, 数据量: 48
```

## 常见问题

### 1. Chromium 无法启动

确保安装了 Chromium 及其依赖：

```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Alpine (Docker)
apk add chromium nss freetype harfbuzz
```

### 2. 数据库连接失败

检查配置文件中的数据库连接信息，确保：
- PostgreSQL 服务正在运行
- 用户名和密码正确
- 数据库已创建
- 防火墙允许连接

### 3. 任务不执行

检查：
- 任务的 `status` 是否为 `enabled`
- `next_run_time` 是否已到期
- Worker 类型是否匹配任务类型
- 查看 Worker 日志输出

## 📚 文档

完整文档请查看 [docs/](docs/) 目录：

### 控制面文档
- **API 文档**: [docs/CONTROL_PLANE_API.md](docs/CONTROL_PLANE_API.md) - 完整的 REST API 文档
- **控制面总结**: [docs/CONTROL_PLANE_SUMMARY.md](docs/CONTROL_PLANE_SUMMARY.md) - 控制面实现总结

### Worker 文档
- **快速开始**: [docs/QUICKSTART.md](docs/QUICKSTART.md) - 5 分钟快速上手
- **详细入门**: [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) - 10 分钟详细指南
- **K8S 部署**: [docs/K8S_QUICK_START.md](docs/K8S_QUICK_START.md) - Kubernetes 快速部署
- **实现说明**: [docs/WORKER_IMPLEMENTATION.md](docs/WORKER_IMPLEMENTATION.md) - Worker 实现细节
- **数据库采集器**: [docs/DATABASE_COLLECTOR_GUIDE.md](docs/DATABASE_COLLECTOR_GUIDE.md) - 数据库采集指南

### 项目文档
- **问题修复**: [docs/QUICK_FIX.md](docs/QUICK_FIX.md) - 常见问题快速修复
- **文档索引**: [docs/README.md](docs/README.md) - 完整文档列表
- **项目总结**: [docs/PROJECT_COMPLETION_SUMMARY.md](docs/PROJECT_COMPLETION_SUMMARY.md) - 项目完成总结
- **最终总结**: [docs/FINAL_SUMMARY.md](docs/FINAL_SUMMARY.md) - 最终总结报告

## 🧪 测试

测试文件位于 [tests/](tests/) 目录：

```bash
# 运行简单测试
go run tests/test_simple.go

# 运行完整流程测试
go run tests/test_with_storage.go
```

更多测试信息请查看 [tests/README.md](tests/README.md)。

## 🚀 快速验证

### 本地验证

```bash
# 1. 下载依赖
go mod download

# 2. 运行简单测试
go run tests/test_simple.go

# 3. 运行完整测试
go run tests/test_with_storage.go
```

### Kubernetes 验证

```bash
# 1. 一键部署
./deploy-k8s.sh

# 2. 等待 2 分钟后验证
./verify-k8s.sh
```

详细说明请查看 [docs/K8S_QUICK_START.md](docs/K8S_QUICK_START.md)。

## 📊 项目统计

### 代码统计
- **总代码行数**: ~6000 行
- **Go 文件数**: 40+ 个
- **控制面代码**: ~1000 行
- **数据面代码**: ~4000 行
- **配置和脚本**: ~1000 行

### 功能统计
- **API 端点**: 25+ 个
- **采集器**: 3 个
- **清洗规则**: 15 种
- **去重策略**: 3 种
- **存储类型**: 3 种
- **监控指标**: 28 个
- **单元测试**: 19 个
- **测试覆盖率**: ~70%

### 文档统计
- **技术文档**: 15+ 份
- **API 文档**: 1 份
- **部署脚本**: 5+ 个
- **测试脚本**: 3+ 个

## 🎯 开发完成情况

### 控制面 (Control Plane) ✅
- ✅ RESTful API Server
- ✅ 任务管理 (CRUD + 启动/停止)
- ✅ 数据源管理 (CRUD + 连接测试)
- ✅ 清洗规则管理 (CRUD)
- ✅ 执行历史查询
- ✅ 统计信息展示
- ✅ 健康检查端点
- ✅ 结构化日志
- ✅ K8S 部署配置
- ✅ 完整 API 文档

### 数据面 (Data Plane) ✅

#### Week 1: 生产必需功能 ✅
- ✅ 错误重试机制
- ✅ 超时控制
- ✅ 健康检查
- ✅ 优雅关闭
- ✅ 基础指标

### Week 2: 扩展采集能力 ✅
- ✅ 数据库采集器（MySQL + PostgreSQL）
- ✅ 15 种增强清洗规则
- ✅ 自动类型转换
- ✅ 连接池管理

### Week 3: 扩展存储能力 ✅
- ✅ MongoDB 存储
- ✅ 3 种去重策略
- ✅ 连接池优化
- ✅ 统计分析

### Week 4: 监控和测试 ✅
- ✅ 28 个 Prometheus 指标
- ✅ 14 个 Grafana 面板
- ✅ 20+ 告警规则
- ✅ 结构化日志
- ✅ 19 个单元测试

## 📚 完整文档

### 完成报告
- [Week 1 完成报告](docs/WEEK1_COMPLETION.md)
- [Week 2 完成报告](docs/WEEK2_COMPLETION.md)
- [Week 2 总结](docs/WEEK2_SUMMARY.md)
- [Week 3 完成报告](docs/WEEK3_COMPLETION.md)
- [Week 3 总结](docs/WEEK3_SUMMARY.md)
- [Week 4 完成报告](docs/WEEK4_COMPLETION.md)

### 使用指南
- [数据库采集器指南](docs/DATABASE_COLLECTOR_GUIDE.md)
- [项目完成总结](docs/PROJECT_COMPLETION_SUMMARY.md)
- [最终总结](docs/FINAL_SUMMARY.md)
- [部署总结](DEPLOYMENT_SUMMARY.md)

### 检查清单
- [最终检查清单](FINAL_CHECKLIST.md)

## 🚀 快速部署

### 方式 1: 部署控制面 API Server
```bash
# 1. 初始化数据库
psql -U postgres -f scripts/init_control_db.sql

# 2. 本地运行
go build -o bin/api-server ./cmd/api-server
./bin/api-server

# 3. K8S 部署
./deploy-api-server.sh

# 4. 测试 API
./test_api_server.sh
```

### 方式 2: 部署 Worker
```bash
# 1. 快速更新（推荐）
./quick-update.sh

# 2. K8S 完整部署
./deploy-k8s-worker.sh

# 3. 本地运行
go build -o bin/worker ./cmd/worker
./bin/worker -config config/worker.yaml
```

### 方式 3: 完整系统部署
```bash
# 1. 部署控制面
./deploy-api-server.sh

# 2. 部署 Worker
./deploy-k8s-worker.sh

# 3. 验证部署
kubectl get pods -n datafusion
kubectl get svc -n datafusion
```

## 🔍 监控端点

```bash
# Prometheus 指标
curl http://localhost:9090/metrics

# 健康检查
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## 🧪 运行测试

```bash
# 单元测试
go test ./tests/unit/... -v

# 覆盖率
go test ./tests/unit/... -cover

# 集成测试
go run test_database_collector.go
go run test_mongodb_and_dedup.go
```

## 许可证

MIT License
