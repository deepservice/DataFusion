# DataFusion 快速开始指南

## 🎯 两种启动方式

### 方式 1: 使用 Docker（推荐）

适合没有安装 PostgreSQL 的用户，使用 Docker 容器运行数据库。

#### 一键启动（推荐）

```bash
# 运行 Docker 快速启动脚本
./scripts/docker_quick_start.sh

# 启动 API Server
./bin/api-server

# 测试服务
curl http://localhost:8081/healthz
```

#### 手动步骤

#### 步骤 1: 启动 PostgreSQL 容器

```bash
# 启动 PostgreSQL 容器
docker run -d \
  --name datafusion-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_DB=postgres \
  -p 5432:5432 \
  postgres:14

# 等待容器启动（约 30 秒）
sleep 30
```

#### 步骤 2: 初始化数据库

```bash
# 创建数据库
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_control;"
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_data;"

# 初始化控制面数据库
docker exec -i datafusion-postgres psql -U postgres -d datafusion_control < scripts/init_control_db.sql

# 初始化数据面数据库
docker exec -i datafusion-postgres psql -U postgres -d datafusion_data < scripts/init_db.sql
```

#### 步骤 3: 启动 API Server

```bash
# 编译 API Server
go build -o bin/api-server ./cmd/api-server

# 启动 API Server（使用端口 8081，避免冲突）
./bin/api-server
```

#### 步骤 4: 验证服务

```bash
# 健康检查
curl http://localhost:8081/healthz
# 输出: {"status":"ok"}

# 查看数据源
curl http://localhost:8081/api/v1/datasources

# 运行完整 API 测试
./tests/test_api_server.sh
```

### 方式 2: 使用本地 PostgreSQL

适合已经安装了 PostgreSQL 的用户。

#### 步骤 1: 初始化数据库

```bash
# 创建数据库
createdb datafusion_control
createdb datafusion_data

# 初始化控制面数据库
psql -U postgres -d datafusion_control -f scripts/init_control_db.sql

# 初始化数据面数据库
psql -U postgres -d datafusion_data -f scripts/init_db.sql
```

#### 步骤 2: 修改配置

编辑 `config/api-server.yaml`，确保数据库配置正确：

```yaml
database:
  postgresql:
    host: localhost
    port: 5432
    user: postgres  # 或你的 PostgreSQL 用户名
    password: postgres  # 或你的 PostgreSQL 密码
    database: datafusion_control
    sslmode: disable
```

#### 步骤 3: 启动和测试

```bash
# 编译并启动 API Server
go build -o bin/api-server ./cmd/api-server
./bin/api-server

# 验证服务
curl http://localhost:8081/healthz

# 运行完整测试
./tests/test_api_server.sh
```

## 🧪 运行测试

### 1. API Server 测试

```bash
# 运行完整的 API 测试套件
./tests/test_api_server.sh
```

### 2. Worker 功能测试

```bash
# 简单功能测试（无需数据库）
go run tests/test_simple.go

# 完整流程测试（包含文件存储）
go run tests/test_with_storage.go

# 数据库采集器测试（需要数据库）
go run tests/test_database_collector.go

# MongoDB 和去重测试（需要 MongoDB）
go run tests/test_mongodb_and_dedup.go
```

### 3. 单元测试

```bash
# 运行所有单元测试
go test ./tests/unit/... -v

# 查看测试覆盖率
go test ./tests/unit/... -cover
```

## 📝 配置说明

### API Server 配置

文件：`config/api-server.yaml`

```yaml
server:
  port: 8081  # API Server 端口
  mode: debug

database:
  postgresql:
    host: localhost
    port: 5432
    user: postgres
    password: postgres
    database: datafusion_control
    sslmode: disable
```

### Worker 配置

文件：`config/worker.yaml`

```yaml
worker_type: "web-rpa"
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

## 🔧 常见问题

### 1. 端口冲突

如果端口 8081 被占用，修改 `config/api-server.yaml` 中的端口号：

```yaml
server:
  port: 8082  # 使用其他端口
```

### 2. PostgreSQL 连接失败

检查配置文件中的数据库连接信息：

```bash
# 测试 Docker 容器连接
docker exec -it datafusion-postgres psql -U postgres -d datafusion_control -c "SELECT 1;"

# 测试本地 PostgreSQL 连接
psql -U postgres -d datafusion_control -c "SELECT 1;"
```

### 3. Docker 容器管理

```bash
# 查看容器状态
docker ps

# 查看容器日志
docker logs datafusion-postgres

# 停止容器
docker stop datafusion-postgres

# 重新启动容器
docker start datafusion-postgres

# 删除容器（注意：会丢失数据）
docker rm -f datafusion-postgres
```

## ✅ 验证成功标志

当你看到以下输出时，说明系统启动成功：

1. **API Server 启动成功**：
   ```json
   {"status":"ok"}
   ```

2. **API 测试通过**：
   ```
   =========================================
   测试完成！
   =========================================
   ```

3. **数据库连接正常**：
   ```
   DataFusion Control Database initialized successfully!
   ```

## 🚀 下一步

系统启动成功后，你可以：

1. **启动 Worker**：处理数据采集任务
2. **创建采集任务**：通过 API 或直接插入数据库
3. **查看监控**：访问 Prometheus 指标端点
4. **部署到 K8S**：使用提供的部署脚本

详细信息请查看：
- [Worker 实现文档](docs/WORKER_IMPLEMENTATION.md)
- [API 文档](docs/CONTROL_PLANE_API.md)
- [K8S 部署指南](docs/K8S_DEPLOYMENT_GUIDE.md)

# 查看统计信息
curl http://localhost:8080/api/v1/stats/overview
```

---

## 10 分钟完整体验

### 场景: 采集技术博客文章

#### 1. 准备环境

```bash
# 安装依赖
go mod download

# 启动 PostgreSQL
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:14

# 等待数据库启动
sleep 5
```

#### 2. 初始化数据库

```bash
# 创建控制面数据库
docker exec -i postgres psql -U postgres << EOF
CREATE DATABASE datafusion_control;
\c datafusion_control
\i /scripts/init_control_db.sql
EOF

# 创建数据面数据库
docker exec -i postgres psql -U postgres << EOF
CREATE DATABASE datafusion_data;
\c datafusion_data
CREATE TABLE articles (
  id SERIAL PRIMARY KEY,
  title TEXT,
  content TEXT,
  author TEXT,
  published_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
EOF
```

#### 3. 启动服务

```bash
# 启动 API Server
./bin/api-server &
API_PID=$!

# 启动 Worker
./bin/worker &
WORKER_PID=$!

# 等待服务启动
sleep 3
```

#### 4. 配置采集任务

```bash
# 创建数据源
DS_ID=$(curl -s -X POST http://localhost:8080/api/v1/datasources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "技术博客",
    "type": "web",
    "config": "{\"url\":\"https://blog.example.com\"}",
    "description": "技术博客数据源",
    "status": "active"
  }' | jq -r '.id')

echo "数据源 ID: $DS_ID"

# 创建采集任务
TASK_ID=$(curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"采集技术文章\",
    \"description\": \"每天采集技术博客文章\",
    \"type\": \"web-rpa\",
    \"data_source_id\": $DS_ID,
    \"cron\": \"0 2 * * *\",
    \"status\": \"enabled\",
    \"config\": \"{\\\"data_source\\\":{\\\"type\\\":\\\"web-rpa\\\",\\\"url\\\":\\\"https://blog.example.com\\\",\\\"selectors\\\":{\\\"_list\\\":\\\".post\\\",\\\"title\\\":\\\".post-title\\\",\\\"content\\\":\\\".post-content\\\",\\\"author\\\":\\\".author-name\\\"}},\\\"processor\\\":{\\\"cleaning_rules\\\":[{\\\"field\\\":\\\"title\\\",\\\"type\\\":\\\"trim\\\"},{\\\"field\\\":\\\"content\\\",\\\"type\\\":\\\"remove_html\\\"}]},\\\"storage\\\":{\\\"target\\\":\\\"postgresql\\\",\\\"table\\\":\\\"articles\\\"}}\"
  }" | jq -r '.id')

echo "任务 ID: $TASK_ID"
```

#### 5. 手动触发任务

```bash
# 立即执行任务
curl -X POST http://localhost:8080/api/v1/tasks/$TASK_ID/run

echo "任务已触发，等待执行..."
sleep 10
```

#### 6. 查看结果

```bash
# 查看执行历史
echo "=== 执行历史 ==="
curl -s http://localhost:8080/api/v1/executions/task/$TASK_ID | jq '.'

# 查看采集的数据
echo "=== 采集的数据 ==="
docker exec -i postgres psql -U postgres -d datafusion_data -c "SELECT * FROM articles LIMIT 5;"

# 查看统计信息
echo "=== 统计信息 ==="
curl -s http://localhost:8080/api/v1/stats/overview | jq '.'
```

#### 7. 清理

```bash
# 停止服务
kill $API_PID $WORKER_PID

# 停止数据库
docker stop postgres
docker rm postgres
```

---

## Kubernetes 快速部署

### 一键部署

```bash
# 1. 部署控制面
./deploy-api-server.sh

# 2. 部署 Worker
./deploy-k8s-worker.sh

# 3. 查看状态
kubectl get pods -n datafusion
kubectl get svc -n datafusion

# 4. 端口转发
kubectl port-forward -n datafusion svc/api-server-service 8080:8080 &

# 5. 测试 API
curl http://localhost:8080/healthz
```

### 访问服务

```bash
# 方式 1: 端口转发
kubectl port-forward -n datafusion svc/api-server-service 8080:8080

# 方式 2: Ingress (需要配置 DNS)
# 访问 http://api.datafusion.local

# 方式 3: NodePort (修改 Service 类型)
kubectl patch svc api-server-service -n datafusion -p '{"spec":{"type":"NodePort"}}'
kubectl get svc api-server-service -n datafusion
```

---

## 常用操作

### 任务管理

```bash
# 查看所有任务
curl http://localhost:8080/api/v1/tasks

# 查看启用的任务
curl http://localhost:8080/api/v1/tasks?status=enabled

# 启动任务
curl -X POST http://localhost:8080/api/v1/tasks/1/run

# 停止任务
curl -X POST http://localhost:8080/api/v1/tasks/1/stop

# 更新任务
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'

# 删除任务
curl -X DELETE http://localhost:8080/api/v1/tasks/1
```

### 数据源管理

```bash
# 查看所有数据源
curl http://localhost:8080/api/v1/datasources

# 测试连接
curl -X POST http://localhost:8080/api/v1/datasources/1/test

# 更新数据源
curl -X PUT http://localhost:8080/api/v1/datasources/1 \
  -H "Content-Type: application/json" \
  -d '{"status":"inactive"}'
```

### 监控和日志

```bash
# 查看 Prometheus 指标
curl http://localhost:9090/metrics

# 查看健康状态
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz

# 查看 Pod 日志
kubectl logs -n datafusion -l app=api-server
kubectl logs -n datafusion -l app=worker
```

---

## 故障排查

### API Server 无法启动

```bash
# 检查配置文件
cat config/api-server.yaml

# 检查数据库连接
psql -U datafusion -d datafusion_control -c "SELECT 1;"

# 查看日志
./bin/api-server 2>&1 | tee api-server.log
```

### Worker 无法执行任务

```bash
# 检查任务状态
curl http://localhost:8080/api/v1/tasks/1

# 检查执行历史
curl http://localhost:8080/api/v1/executions/task/1

# 查看 Worker 日志
kubectl logs -n datafusion -l app=worker --tail=100
```

### 数据库连接失败

```bash
# 检查数据库状态
kubectl get pods -n datafusion -l app=postgres

# 测试连接
kubectl exec -it -n datafusion postgres-0 -- psql -U datafusion -d datafusion_control -c "SELECT 1;"

# 查看数据库日志
kubectl logs -n datafusion postgres-0
```

---

## 下一步

1. **阅读完整文档**: [docs/README.md](docs/README.md)
2. **查看 API 文档**: [docs/CONTROL_PLANE_API.md](docs/CONTROL_PLANE_API.md)
3. **运行测试**: `./test_api_server.sh`
4. **配置监控**: 查看 [k8s/monitoring/](k8s/monitoring/)
5. **自定义扩展**: 添加自定义采集器和处理器

---

## 获取帮助

- **文档**: [docs/](docs/)
- **示例**: [examples/](examples/)
- **测试**: [tests/](tests/)
- **问题**: 查看 [docs/QUICK_FIX.md](docs/QUICK_FIX.md)

---

**祝您使用愉快！** 🎉
