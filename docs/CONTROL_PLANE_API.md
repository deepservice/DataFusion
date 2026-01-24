# DataFusion 控制面 API 文档

## 概述

DataFusion 控制面提供 RESTful API 用于管理数据采集任务、数据源、清洗规则等。

## 服务启动

```bash
# 1. 初始化数据库
psql -U datafusion -d datafusion_control -f scripts/init_control_db.sql

# 2. 启动 API 服务器
./bin/api-server

# 服务默认监听在 http://localhost:8080
```

## API 端点

### 健康检查

#### GET /healthz
检查服务是否运行

**响应示例：**
```json
{
  "status": "ok"
}
```

#### GET /readyz
检查服务是否就绪（包括数据库连接）

**响应示例：**
```json
{
  "status": "ready",
  "database": "connected"
}
```

---

### 任务管理 (Tasks)

#### GET /api/v1/tasks
获取任务列表

**查询参数：**
- `page` - 页码（默认：1）
- `page_size` - 每页数量（默认：20）
- `status` - 状态过滤（enabled/disabled）

**响应示例：**
```json
{
  "data": [
    {
      "id": 1,
      "name": "采集示例网站",
      "description": "每小时采集一次",
      "type": "web-rpa",
      "data_source_id": 1,
      "cron": "0 * * * *",
      "status": "enabled",
      "replicas": 1,
      "execution_timeout": 3600,
      "max_retries": 3,
      "config": "{\"url\":\"https://example.com\"}",
      "created_at": "2024-12-08T10:00:00Z",
      "updated_at": "2024-12-08T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### GET /api/v1/tasks/:id
获取单个任务详情

**响应示例：**
```json
{
  "id": 1,
  "name": "采集示例网站",
  "description": "每小时采集一次",
  "type": "web-rpa",
  "data_source_id": 1,
  "cron": "0 * * * *",
  "status": "enabled",
  "replicas": 1,
  "execution_timeout": 3600,
  "max_retries": 3,
  "config": "{\"url\":\"https://example.com\"}",
  "created_at": "2024-12-08T10:00:00Z",
  "updated_at": "2024-12-08T10:00:00Z"
}
```

#### POST /api/v1/tasks
创建新任务

**请求体：**
```json
{
  "name": "采集新闻网站",
  "description": "每天采集一次新闻",
  "type": "web-rpa",
  "data_source_id": 1,
  "cron": "0 0 * * *",
  "status": "enabled",
  "replicas": 1,
  "execution_timeout": 3600,
  "max_retries": 3,
  "config": "{\"url\":\"https://news.example.com\",\"selectors\":{\"title\":\".title\",\"content\":\".content\"}}"
}
```

**响应：** 201 Created，返回创建的任务对象

#### PUT /api/v1/tasks/:id
更新任务

**请求体：** 同创建任务

**响应：** 200 OK

#### DELETE /api/v1/tasks/:id
删除任务

**响应：** 200 OK

#### POST /api/v1/tasks/:id/run
手动触发任务执行

**响应示例：**
```json
{
  "message": "任务已触发"
}
```

#### POST /api/v1/tasks/:id/stop
停止任务

**响应示例：**
```json
{
  "message": "任务已停止"
}
```

---

### 数据源管理 (DataSources)

#### GET /api/v1/datasources
获取数据源列表

**查询参数：**
- `page` - 页码
- `page_size` - 每页数量
- `type` - 类型过滤（web/api/database）

**响应示例：**
```json
{
  "data": [
    {
      "id": 1,
      "name": "示例网页数据源",
      "type": "web",
      "config": "{\"url\":\"https://example.com\",\"method\":\"GET\"}",
      "description": "示例网页数据源",
      "status": "active",
      "created_at": "2024-12-08T10:00:00Z",
      "updated_at": "2024-12-08T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### GET /api/v1/datasources/:id
获取单个数据源

#### POST /api/v1/datasources
创建数据源

**请求体示例（Web）：**
```json
{
  "name": "新闻网站",
  "type": "web",
  "config": "{\"url\":\"https://news.example.com\",\"method\":\"GET\"}",
  "description": "新闻网站数据源",
  "status": "active"
}
```

**请求体示例（Database）：**
```json
{
  "name": "MySQL数据库",
  "type": "database",
  "config": "{\"type\":\"mysql\",\"host\":\"localhost\",\"port\":3306,\"user\":\"root\",\"password\":\"password\",\"database\":\"mydb\"}",
  "description": "MySQL数据源",
  "status": "active"
}
```

#### PUT /api/v1/datasources/:id
更新数据源

#### DELETE /api/v1/datasources/:id
删除数据源

#### POST /api/v1/datasources/:id/test
测试数据源连接

**响应示例：**
```json
{
  "status": "success",
  "message": "连接成功"
}
```

---

### 清洗规则管理 (Cleaning Rules)

#### GET /api/v1/cleaning-rules
获取清洗规则列表

**响应示例：**
```json
{
  "data": [
    {
      "id": 1,
      "name": "去除空白",
      "description": "去除字符串首尾空白",
      "rule_type": "trim",
      "config": "{}",
      "created_at": "2024-12-08T10:00:00Z",
      "updated_at": "2024-12-08T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### GET /api/v1/cleaning-rules/:id
获取单个清洗规则

#### POST /api/v1/cleaning-rules
创建清洗规则

**请求体示例：**
```json
{
  "name": "邮箱验证",
  "description": "验证邮箱格式",
  "rule_type": "email_validate",
  "config": "{}"
}
```

#### PUT /api/v1/cleaning-rules/:id
更新清洗规则

#### DELETE /api/v1/cleaning-rules/:id
删除清洗规则

---

### 执行历史 (Executions)

#### GET /api/v1/executions
获取执行历史列表

**查询参数：**
- `page` - 页码
- `page_size` - 每页数量
- `status` - 状态过滤（running/success/failed）

**响应示例：**
```json
{
  "data": [
    {
      "id": 1,
      "task_id": 1,
      "worker_pod": "worker-abc123",
      "status": "success",
      "start_time": "2024-12-08T10:00:00Z",
      "end_time": "2024-12-08T10:05:00Z",
      "records_collected": 100,
      "error_message": "",
      "retry_count": 0,
      "created_at": "2024-12-08T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### GET /api/v1/executions/:id
获取单个执行记录

#### GET /api/v1/executions/task/:task_id
获取指定任务的执行历史

---

### 统计信息 (Stats)

#### GET /api/v1/stats/overview
获取系统概览统计

**响应示例：**
```json
{
  "total_tasks": 10,
  "enabled_tasks": 8,
  "disabled_tasks": 2,
  "total_datasources": 5,
  "total_executions": 1000,
  "success_executions": 950,
  "failed_executions": 50,
  "success_rate": 95.0,
  "total_records_collected": 50000
}
```

#### GET /api/v1/stats/tasks
获取任务统计

**响应示例：**
```json
{
  "tasks": [
    {
      "task_id": 1,
      "task_name": "采集示例网站",
      "total_executions": 100,
      "success_executions": 95,
      "failed_executions": 5,
      "success_rate": 95.0,
      "total_records": 5000,
      "avg_duration": 300
    }
  ]
}
```

---

## 错误响应

所有错误响应遵循统一格式：

```json
{
  "error": "错误描述信息"
}
```

常见HTTP状态码：
- `200` - 成功
- `201` - 创建成功
- `400` - 请求参数错误
- `404` - 资源不存在
- `500` - 服务器内部错误

---

## 使用示例

### 创建一个Web采集任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "采集技术博客",
    "description": "每天采集技术文章",
    "type": "web-rpa",
    "data_source_id": 1,
    "cron": "0 2 * * *",
    "config": "{\"url\":\"https://blog.example.com\",\"selectors\":{\"title\":\".post-title\",\"content\":\".post-content\",\"author\":\".author-name\"}}"
  }'
```

### 手动触发任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/run
```

### 查看执行历史

```bash
curl http://localhost:8080/api/v1/executions/task/1
```

### 获取系统统计

```bash
curl http://localhost:8080/api/v1/stats/overview
```

---

## 配置文件

配置文件位于 `config/api-server.yaml`：

```yaml
server:
  port: 8080
  mode: debug  # debug, release
  read_timeout: 30
  write_timeout: 30

database:
  postgresql:
    host: localhost
    port: 5432
    user: datafusion
    password: datafusion123
    database: datafusion_control
    sslmode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300

log:
  level: info  # debug, info, warn, error
  format: json  # json, console
```

---

## 下一步

1. 实现前端管理界面
2. 添加用户认证和授权
3. 实现API密钥管理
4. 添加Webhook通知
5. 实现任务依赖关系
