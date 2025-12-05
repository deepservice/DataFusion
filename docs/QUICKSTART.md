# DataFusion Worker 快速开始指南

本指南将帮助你在 5 分钟内启动并验证 DataFusion Worker 的核心功能。

## 前置条件

- ✅ Go 1.21+
- ✅ PostgreSQL 12+
- ✅ 基本的命令行操作能力

## 快速启动（3 步）

### 第 1 步：一键初始化

```bash
# 克隆或进入项目目录
cd datafusion-worker

# 运行快速启动脚本（会自动完成环境检查、依赖下载、数据库初始化）
./scripts/quick_start.sh
```

### 第 2 步：修改配置（可选）

如果你的 PostgreSQL 不是默认配置，请编辑 `config/worker.yaml`：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"        # 修改为你的用户名
  password: "your_pass"   # 修改为你的密码
  database: "datafusion_control"
  ssl_mode: "disable"
```

### 第 3 步：启动 Worker

```bash
# 方式 1: 使用 Makefile
make run

# 方式 2: 直接运行
./bin/worker -config config/worker.yaml
```

## 验证功能

Worker 启动后，会自动执行测试任务。你应该看到类似的日志：

```
2025-12-04 10:00:00 Worker 启动: worker-1234, 类型: web-rpa
2025-12-04 10:00:30 发现 3 个待执行任务
2025-12-04 10:00:30 成功锁定任务 测试-新闻文章采集 (ID: 1)，开始执行
2025-12-04 10:00:35 开始 RPA 采集: https://example.com/news
...
2025-12-04 10:00:45 任务执行完成, 耗时: 15s, 数据量: 50
```

### 查看执行结果

```bash
# 查看生成的数据文件
ls -lh data/

# 查看数据库中的执行记录
psql -U postgres -d datafusion_control -c "
SELECT id, task_id, status, records_collected, start_time 
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 5;
"
```

## 测试不同的采集方式

### 1. 测试 API 采集（推荐先测试这个）

```sql
-- 连接数据库
psql -U postgres -d datafusion_control

-- 插入 API 测试任务
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    'API测试-JSONPlaceholder',
    'api',
    'enabled',
    '*/1 * * * *',
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://jsonplaceholder.typicode.com/posts?_limit=3",
            "method": "GET",
            "selectors": {
                "_data_path": "",
                "id": "id",
                "title": "title"
            }
        },
        "processor": {"cleaning_rules": [], "transform_rules": []},
        "storage": {
            "target": "file",
            "database": "api_test",
            "table": "posts"
        }
    }'
);
```

等待 1 分钟后，查看结果：

```bash
cat data/api_test/posts_*.json | jq .
```

### 2. 测试数据库存储

```sql
-- 创建测试表
\c datafusion_data

CREATE TABLE test_data (
    id INT PRIMARY KEY,
    title VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 更新任务配置为数据库存储
\c datafusion_control

UPDATE collection_tasks 
SET config = jsonb_set(
    config::jsonb, 
    '{storage}', 
    '{"target": "postgresql", "database": "datafusion_data", "table": "test_data", "mapping": {"id": "id", "title": "title"}}'::jsonb
)
WHERE name = 'API测试-JSONPlaceholder';

-- 立即执行
UPDATE collection_tasks SET next_run_time = NOW() WHERE name = 'API测试-JSONPlaceholder';
```

查看数据：

```sql
\c datafusion_data
SELECT * FROM test_data;
```

### 3. 测试数据清洗

```sql
\c datafusion_control

-- 添加清洗规则
UPDATE collection_tasks 
SET config = jsonb_set(
    config::jsonb,
    '{processor,cleaning_rules}',
    '[
        {"field": "title", "type": "trim"},
        {"field": "title", "type": "uppercase"}
    ]'::jsonb
)
WHERE name = 'API测试-JSONPlaceholder';

-- 立即执行
UPDATE collection_tasks SET next_run_time = NOW() WHERE name = 'API测试-JSONPlaceholder';
```

## 常用命令

```bash
# 编译
make build

# 运行
make run

# 清理
make clean

# 查看帮助
make help

# 查看任务列表
psql -U postgres -d datafusion_control -c "SELECT id, name, type, status, next_run_time FROM collection_tasks;"

# 查看执行历史
psql -U postgres -d datafusion_control -c "SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 10;"

# 手动触发任务
psql -U postgres -d datafusion_control -c "UPDATE collection_tasks SET next_run_time = NOW() WHERE id = 1;"
```

## 项目结构速览

```
datafusion-worker/
├── cmd/worker/main.go          # 入口文件
├── internal/
│   ├── collector/              # 采集器（RPA、API）
│   ├── processor/              # 数据处理（清洗、转换）
│   ├── storage/                # 存储（PostgreSQL、文件）
│   ├── database/               # 数据库操作
│   └── worker/                 # Worker 核心逻辑
├── config/worker.yaml          # 配置文件
├── scripts/
│   ├── init_db.sql            # 数据库初始化
│   └── quick_start.sh         # 快速启动脚本
└── examples/simple_test.md    # 详细测试示例
```

## 下一步

✅ **基础功能验证完成后**，你可以：

1. 📖 阅读 [详细测试示例](examples/simple_test.md)
2. 🔧 创建自己的采集任务
3. 📊 配置复杂的数据清洗规则
4. 🚀 部署到生产环境

## 故障排查

### Worker 无法启动

```bash
# 检查 Go 版本
go version

# 检查依赖
go mod download

# 查看详细错误
./bin/worker -config config/worker.yaml
```

### 数据库连接失败

```bash
# 测试数据库连接
psql -U postgres -d datafusion_control -c "SELECT 1;"

# 检查配置文件
cat config/worker.yaml
```

### 任务不执行

```sql
-- 检查任务状态
SELECT id, name, status, next_run_time, type FROM collection_tasks;

-- 确保 Worker 类型匹配
-- Worker 配置的 worker_type 必须与任务的 type 字段一致
```

## 获取帮助

- 📖 查看 [README.md](README.md) 了解完整功能
- 📝 查看 [examples/simple_test.md](examples/simple_test.md) 了解详细测试流程
- 🐛 遇到问题？检查日志输出或提交 Issue

---

**祝你使用愉快！** 🎉
