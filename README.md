# DataFusion Worker

DataFusion 数据采集系统的 Worker 组件，负责执行数据采集、处理和存储任务。

## 功能特性

- ✅ 支持多种数据源类型：Web RPA、API、数据库
- ✅ 灵活的数据清洗和转换规则
- ✅ 多种存储方式：PostgreSQL、MongoDB、文件
- ✅ 分布式任务调度和锁机制
- ✅ 自动重试和错误处理
- ✅ 完整的任务执行记录

## 快速开始

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
  user: "datafusion"
  password: "datafusion123"
  database: "datafusion_control"
  ssl_mode: "disable"

storage:
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
datafusion-worker/
├── cmd/
│   └── worker/                  # Worker 主程序入口
│       └── main.go
├── internal/                    # 内部包（核心业务逻辑）
│   ├── collector/              # 数据采集器
│   │   ├── collector.go        # 采集器接口
│   │   ├── rpa_collector.go    # RPA 采集器
│   │   └── api_collector.go    # API 采集器
│   ├── processor/              # 数据处理器
│   │   └── processor.go        # 数据清洗和转换
│   ├── storage/                # 数据存储
│   │   ├── storage.go          # 存储接口
│   │   ├── postgres_storage.go # PostgreSQL 存储
│   │   └── file_storage.go     # 文件存储
│   ├── database/               # 数据库操作
│   │   └── postgres.go         # PostgreSQL 客户端
│   ├── models/                 # 数据模型
│   │   └── task.go             # 任务模型
│   ├── config/                 # 配置管理
│   │   └── config.go           # 配置加载
│   └── worker/                 # Worker 核心逻辑
│       └── worker.go           # 任务调度和执行
├── config/                      # 配置文件
│   └── worker.yaml             # Worker 配置
├── k8s/                        # Kubernetes 部署文件
│   ├── namespace.yaml          # 命名空间
│   ├── postgresql.yaml         # PostgreSQL 部署
│   ├── postgres-init-scripts.yaml # 数据库初始化
│   ├── worker-config.yaml      # Worker 配置
│   └── worker.yaml             # Worker 部署
├── scripts/                     # 脚本工具
│   ├── init_db.sql             # 数据库初始化
│   ├── insert_test_task.sql    # 测试任务
│   └── quick_start.sh          # 快速启动
├── tests/                       # 测试文件
│   ├── test_simple.go          # 简单测试
│   ├── test_with_storage.go    # 完整流程测试
│   └── README.md               # 测试说明
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
├── Dockerfile                   # Docker 镜像
├── README.md                    # 项目主文档（本文档）
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

- **快速开始**: [docs/QUICKSTART.md](docs/QUICKSTART.md) - 5 分钟快速上手
- **详细入门**: [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) - 10 分钟详细指南
- **K8S 部署**: [docs/K8S_QUICK_START.md](docs/K8S_QUICK_START.md) - Kubernetes 快速部署
- **实现说明**: [docs/WORKER_IMPLEMENTATION.md](docs/WORKER_IMPLEMENTATION.md) - Worker 实现细节
- **问题修复**: [docs/QUICK_FIX.md](docs/QUICK_FIX.md) - 常见问题快速修复
- **文档索引**: [docs/README.md](docs/README.md) - 完整文档列表

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

## 开发计划

- [ ] 支持数据库采集器
- [ ] 支持 MongoDB 存储
- [ ] 增加更多数据清洗规则
- [ ] 实现数据去重机制
- [ ] 添加 Prometheus 监控指标
- [ ] 支持插件化扩展
- [ ] 完善单元测试和集成测试

## 许可证

MIT License
