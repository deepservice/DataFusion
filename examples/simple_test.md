# 简单测试示例

本文档提供一个最简单的测试流程，帮助你快速验证 Worker 的核心功能。

## 测试场景

我们将创建一个简单的 API 采集任务，从 JSONPlaceholder（一个免费的测试 API）获取数据。

## 步骤 1: 准备环境

```bash
# 1. 初始化数据库
./scripts/quick_start.sh

# 2. 确认配置文件
cat config/worker.yaml
```

## 步骤 2: 创建测试任务

```sql
-- 连接到数据库
psql -U postgres -d datafusion_control

-- 插入一个简单的 API 采集任务
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '测试-JSONPlaceholder',
    'api',
    'enabled',
    '*/1 * * * *',  -- 每分钟执行一次
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://jsonplaceholder.typicode.com/posts?_limit=5",
            "method": "GET",
            "headers": {},
            "selectors": {
                "_data_path": "",
                "id": "id",
                "title": "title",
                "body": "body",
                "userId": "userId"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "field": "title",
                    "type": "trim"
                }
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "file",
            "database": "test_output",
            "table": "posts",
            "mapping": {}
        }
    }'
);

-- 查看任务
SELECT id, name, type, status, next_run_time FROM collection_tasks;
```

## 步骤 3: 启动 Worker

```bash
# 方式 1: 直接运行
make run

# 方式 2: 编译后运行
make build
./bin/worker -config config/worker.yaml
```

## 步骤 4: 观察执行结果

Worker 启动后，你应该看到类似的日志输出：

```
2025-12-04 10:00:00 Worker 启动: worker-1234, 类型: api
2025-12-04 10:00:00 发现 1 个待执行任务
2025-12-04 10:00:00 成功锁定任务 测试-JSONPlaceholder (ID: 1)，开始执行
2025-12-04 10:00:01 开始 API 采集: https://jsonplaceholder.typicode.com/posts?_limit=5
2025-12-04 10:00:02 API 请求成功，状态码: 200，响应大小: 1234 bytes
2025-12-04 10:00:02 解析完成，提取到 5 条数据
2025-12-04 10:00:02 开始数据处理，共 5 条数据
2025-12-04 10:00:02 数据处理完成，有效数据 5 条
2025-12-04 10:00:02 开始存储数据到文件，数据量: 5
2025-12-04 10:00:02 数据存储完成，文件: ./data/test_output/posts_20251204_100002.json
2025-12-04 10:00:02 任务执行完成: 测试-JSONPlaceholder, 耗时: 2s, 数据量: 5
```

## 步骤 5: 查看采集结果

### 查看文件输出

```bash
# 查看生成的文件
ls -lh data/test_output/

# 查看文件内容
cat data/test_output/posts_*.json | jq .
```

### 查看执行记录

```sql
-- 查看任务执行历史
psql -U postgres -d datafusion_control -c "
SELECT 
    id, 
    task_id, 
    worker_pod, 
    status, 
    records_collected, 
    start_time, 
    end_time,
    end_time - start_time as duration
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 10;
"
```

## 步骤 6: 测试数据库存储

修改任务配置，将数据存储到 PostgreSQL：

```sql
-- 首先创建测试表
\c datafusion_data

CREATE TABLE IF NOT EXISTS test_posts (
    id INT PRIMARY KEY,
    title VARCHAR(500),
    body TEXT,
    user_id INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 更新任务配置
\c datafusion_control

UPDATE collection_tasks 
SET config = '{
    "data_source": {
        "type": "api",
        "url": "https://jsonplaceholder.typicode.com/posts?_limit=5",
        "method": "GET",
        "headers": {},
        "selectors": {
            "_data_path": "",
            "id": "id",
            "title": "title",
            "body": "body",
            "userId": "userId"
        }
    },
    "processor": {
        "cleaning_rules": [
            {"field": "title", "type": "trim"}
        ],
        "transform_rules": []
    },
    "storage": {
        "target": "postgresql",
        "database": "datafusion_data",
        "table": "test_posts",
        "mapping": {
            "id": "id",
            "title": "title",
            "body": "body",
            "userId": "user_id"
        }
    }
}'
WHERE name = '测试-JSONPlaceholder';

-- 更新下次执行时间（立即执行）
UPDATE collection_tasks 
SET next_run_time = NOW() 
WHERE name = '测试-JSONPlaceholder';
```

等待 Worker 执行后，查看数据：

```sql
\c datafusion_data

SELECT id, title, user_id, created_at 
FROM test_posts 
ORDER BY created_at DESC;
```

## 步骤 7: 测试 RPA 采集（可选）

如果你想测试 RPA 采集功能，需要确保安装了 Chromium：

```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium
```

然后创建一个 RPA 任务：

```sql
\c datafusion_control

INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '测试-RPA采集',
    'web-rpa',
    'enabled',
    '*/2 * * * *',
    NOW(),
    1,
    '{
        "data_source": {
            "type": "web-rpa",
            "url": "https://example.com",
            "selectors": {
                "title": "h1",
                "content": "p"
            }
        },
        "processor": {
            "cleaning_rules": [
                {"field": "title", "type": "trim"},
                {"field": "content", "type": "remove_html"}
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "file",
            "database": "rpa_output",
            "table": "example_data",
            "mapping": {}
        }
    }'
);
```

## 故障排查

### Worker 无法连接数据库

检查 `config/worker.yaml` 中的数据库配置：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"  # 修改为你的用户名
  password: "your_password"  # 修改为你的密码
  database: "datafusion_control"
  ssl_mode: "disable"
```

### 任务不执行

```sql
-- 检查任务状态
SELECT id, name, status, next_run_time, type 
FROM collection_tasks;

-- 确保 next_run_time 已到期
UPDATE collection_tasks 
SET next_run_time = NOW() 
WHERE id = 1;
```

### API 请求失败

检查网络连接和 API 地址是否正确：

```bash
# 测试 API 是否可访问
curl https://jsonplaceholder.typicode.com/posts?_limit=5
```

## 下一步

恭喜！你已经成功验证了 Worker 的核心功能。接下来你可以：

1. 创建更复杂的采集任务
2. 配置数据清洗规则
3. 集成到你的业务系统
4. 部署到生产环境

更多信息请参考主 README.md 文档。
