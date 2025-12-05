# 开始使用 DataFusion Worker

## 🎯 目标

通过本指南，你将在 10 分钟内完成 Worker 的部署和第一个数据采集任务的验证。

## 📋 前置要求

确保你的系统已安装：

- ✅ Go 1.21 或更高版本
- ✅ PostgreSQL 12 或更高版本
- ✅ 基本的命令行操作能力

可选（用于 RPA 采集）：
- Chromium 浏览器

## 🚀 第一步：环境初始化

### 自动初始化（推荐）

```bash
# 运行快速启动脚本
chmod +x scripts/quick_start.sh
./scripts/quick_start.sh
```

这个脚本会自动完成：
- ✅ 检查 Go 和 PostgreSQL 环境
- ✅ 下载 Go 依赖
- ✅ 创建数据库
- ✅ 初始化表结构
- ✅ 插入测试任务
- ✅ 编译 Worker

### 手动初始化

如果自动脚本失败，可以手动执行：

```bash
# 1. 下载依赖
go mod download

# 2. 创建数据库
psql -U postgres -c "CREATE DATABASE datafusion_control;"
psql -U postgres -c "CREATE DATABASE datafusion_data;"

# 3. 初始化表结构
psql -U postgres -d datafusion_control -f scripts/init_db.sql

# 4. 插入测试任务
psql -U postgres -d datafusion_control -f scripts/insert_test_task.sql

# 5. 编译 Worker
go build -o bin/worker cmd/worker/main.go
```

## ⚙️ 第二步：配置 Worker

编辑 `config/worker.yaml`：

```yaml
# Worker 类型（根据你要测试的功能选择）
worker_type: "api"  # 推荐先测试 api，因为不需要浏览器

# 轮询间隔
poll_interval: 30s

# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "postgres"      # 修改为你的用户名
  password: "postgres"  # 修改为你的密码
  database: "datafusion_control"
  ssl_mode: "disable"

# 存储配置
storage:
  type: "file"  # 先使用文件存储，更容易验证
  database:
    host: "localhost"
    port: 5432
    user: "postgres"
    password: "postgres"
    database: "datafusion_data"
    ssl_mode: "disable"
```

## 🎬 第三步：启动 Worker

```bash
# 方式 1: 使用 Makefile
make run

# 方式 2: 直接运行
./bin/worker -config config/worker.yaml

# 方式 3: 使用 go run
go run cmd/worker/main.go -config config/worker.yaml
```

你应该看到类似的输出：

```
2025-12-04 10:00:00 Worker 启动成功，轮询间隔: 30s
2025-12-04 10:00:00 Worker 启动: worker-1234, 类型: api
2025-12-04 10:00:00 没有待执行任务
2025-12-04 10:00:30 发现 1 个待执行任务
2025-12-04 10:00:30 成功锁定任务 测试-产品数据API采集 (ID: 2)，开始执行
...
```

## ✅ 第四步：验证功能

### 方法 1: 查看文件输出

```bash
# 查看生成的数据文件
ls -lh data/

# 查看文件内容
cat data/*/products_*.json | jq .
```

### 方法 2: 查看数据库记录

```bash
# 查看任务执行历史
psql -U postgres -d datafusion_control -c "
SELECT 
    id, 
    task_id, 
    status, 
    records_collected, 
    start_time,
    end_time - start_time as duration
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 5;
"
```

### 方法 3: 查看 Worker 日志

Worker 会输出详细的执行日志，包括：
- 任务发现和锁定
- 数据采集进度
- 数据处理结果
- 存储完成状态

## 🧪 第五步：创建你的第一个任务

### 简单的 API 采集任务

```sql
-- 连接到数据库
psql -U postgres -d datafusion_control

-- 创建任务
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '我的第一个任务',
    'api',
    'enabled',
    '*/1 * * * *',  -- 每分钟执行一次
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://jsonplaceholder.typicode.com/users?_limit=3",
            "method": "GET",
            "headers": {},
            "selectors": {
                "_data_path": "",
                "id": "id",
                "name": "name",
                "email": "email",
                "city": "address.city"
            }
        },
        "processor": {
            "cleaning_rules": [
                {"field": "name", "type": "trim"},
                {"field": "email", "type": "lowercase"}
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "file",
            "database": "my_data",
            "table": "users",
            "mapping": {}
        }
    }'
);

-- 查看任务
SELECT id, name, type, status, next_run_time FROM collection_tasks WHERE name = '我的第一个任务';
```

等待 1 分钟后，查看结果：

```bash
# 查看生成的文件
ls -lh data/my_data/

# 查看内容
cat data/my_data/users_*.json | jq .
```

## 🎓 常用操作

### 查看所有任务

```sql
psql -U postgres -d datafusion_control -c "
SELECT id, name, type, status, next_run_time 
FROM collection_tasks 
ORDER BY id;
"
```

### 手动触发任务

```sql
psql -U postgres -d datafusion_control -c "
UPDATE collection_tasks 
SET next_run_time = NOW() 
WHERE id = 1;
"
```

### 停止任务

```sql
psql -U postgres -d datafusion_control -c "
UPDATE collection_tasks 
SET status = 'disabled' 
WHERE id = 1;
"
```

### 启用任务

```sql
psql -U postgres -d datafusion_control -c "
UPDATE collection_tasks 
SET status = 'enabled', next_run_time = NOW() 
WHERE id = 1;
"
```

### 查看任务执行历史

```sql
psql -U postgres -d datafusion_control -c "
SELECT 
    te.id,
    ct.name as task_name,
    te.status,
    te.records_collected,
    te.start_time,
    te.error_message
FROM task_executions te
JOIN collection_tasks ct ON te.task_id = ct.id
ORDER BY te.start_time DESC
LIMIT 10;
"
```

## 🔧 故障排查

### Worker 无法启动

**问题**: `panic: runtime error` 或编译错误

**解决**:
```bash
# 重新下载依赖
go mod tidy
go mod download

# 重新编译
make clean
make build
```

### 数据库连接失败

**问题**: `connection refused` 或 `authentication failed`

**解决**:
```bash
# 1. 检查 PostgreSQL 是否运行
sudo systemctl status postgresql

# 2. 测试连接
psql -U postgres -c "SELECT 1;"

# 3. 修改配置文件中的用户名和密码
vim config/worker.yaml
```

### 任务不执行

**问题**: Worker 启动了但任务不执行

**检查清单**:
```sql
-- 1. 检查任务状态
SELECT id, name, status, next_run_time, type 
FROM collection_tasks;

-- 2. 确保 next_run_time 已到期
UPDATE collection_tasks 
SET next_run_time = NOW() 
WHERE id = 1;

-- 3. 确保 Worker 类型匹配
-- config/worker.yaml 中的 worker_type 必须与任务的 type 一致
```

### API 请求失败

**问题**: `API 请求失败` 或 `timeout`

**解决**:
```bash
# 1. 测试 API 是否可访问
curl https://jsonplaceholder.typicode.com/users

# 2. 检查网络连接
ping jsonplaceholder.typicode.com

# 3. 增加超时时间
# 在 config/worker.yaml 中修改:
collector:
  api:
    timeout: 60  # 增加到 60 秒
```

## 📚 下一步

恭喜！你已经成功完成了基础验证。接下来可以：

1. 📖 阅读 [README.md](README.md) 了解完整功能
2. 🧪 查看 [examples/simple_test.md](examples/simple_test.md) 学习更多示例
3. 🔧 创建更复杂的采集任务
4. 🚀 部署到生产环境

## 💡 提示

- 建议先使用 API 采集器测试，因为不需要安装浏览器
- 使用文件存储更容易验证结果
- 查看 Worker 日志可以了解详细的执行过程
- 使用 `jq` 工具可以更好地查看 JSON 文件

## 🆘 获取帮助

如果遇到问题：

1. 查看 Worker 日志输出
2. 查看数据库中的错误信息
3. 参考 [WORKER_IMPLEMENTATION.md](WORKER_IMPLEMENTATION.md)
4. 提交 Issue

---

**祝你使用愉快！** 🎉
