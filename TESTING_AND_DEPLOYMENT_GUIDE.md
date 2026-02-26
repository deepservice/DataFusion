# 📋 DataFusion 测试和部署完整指南

**文档版本**: v1.0  
**更新日期**: 2024-12-08  
**适用版本**: DataFusion v2.0+  

---

## 📑 目录

1. [开发环境搭建](#1-开发环境搭建)
2. [本地开发测试](#2-本地开发测试)
3. [功能验证测试](#3-功能验证测试)
4. [生产环境部署](#4-生产环境部署)
5. [部署后验证](#5-部署后验证)
6. [常见问题排查](#6-常见问题排查)

---

## 1. 开发环境搭建

### 1.1 系统要求

**硬件要求**:
- CPU: 4核心以上
- 内存: 8GB以上
- 磁盘: 50GB可用空间

**软件要求**:
- 操作系统: Linux / macOS / Windows (WSL2)
- Go: 1.21+
- PostgreSQL: 14+
- Node.js: 16+ (用于Web界面开发)
- Docker: 20.10+ (可选)
- Kubernetes: 1.24+ (生产部署)

### 1.2 安装依赖

#### 1.2.1 安装 Go

```bash
# Linux/macOS
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 验证安装
go version
```

#### 1.2.2 安装 PostgreSQL

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install postgresql-14 postgresql-client-14

# macOS
brew install postgresql@14

# 启动服务
sudo systemctl start postgresql  # Linux
brew services start postgresql@14  # macOS

# 验证安装
psql --version
```

#### 1.2.3 安装 Node.js (Web界面开发)

```bash
# 使用 nvm 安装
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
nvm use 18

# 验证安装
node --version
npm --version
```

#### 1.2.4 安装 Chromium (RPA采集器)

```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium

# 验证安装
chromium --version
```

### 1.3 克隆项目

```bash
# 克隆代码仓库
git clone https://github.com/your-org/datafusion.git
cd datafusion

# 下载 Go 依赖
go mod download

# 验证依赖
go mod verify
```

### 1.4 配置数据库

```bash
# 创建数据库用户
sudo -u postgres psql
CREATE USER datafusion WITH PASSWORD 'datafusion123';
ALTER USER datafusion CREATEDB;
\q

# 创建数据库
createdb -U datafusion datafusion_control
createdb -U datafusion datafusion_data

# 初始化数据库表结构
psql -U datafusion -d datafusion_control -f scripts/init_control_db.sql
psql -U datafusion -d datafusion_data -f scripts/init_db.sql

# 验证数据库
psql -U datafusion -d datafusion_control -c "\dt"
```

### 1.5 配置文件

```bash
# 复制配置文件模板
cp config/api-server.yaml.example config/api-server.yaml
cp config/worker.yaml.example config/worker.yaml
cp .env.example .env

# 编辑配置文件，修改数据库连接信息
vim config/api-server.yaml
```

**配置示例** (`config/api-server.yaml`):
```yaml
server:
  port: 8080
  mode: debug

database:
  postgresql:
    host: localhost
    port: 5432
    user: datafusion
    password: datafusion123
    database: datafusion_control
    sslmode: disable

auth:
  jwt:
    secret_key: "your-secret-key-change-in-production"
    token_duration: "24h"

log:
  level: info
  format: console
```

---

## 2. 本地开发测试

### 2.1 编译项目

```bash
# 编译 API Server
go build -o bin/api-server ./cmd/api-server

# 编译 Worker
go build -o bin/worker ./cmd/worker

# 验证编译
./bin/api-server --version
./bin/worker --version
```

### 2.2 启动服务

#### 2.2.1 启动 API Server

```bash
# 方式1: 直接运行
./bin/api-server

# 方式2: 使用配置文件
./bin/api-server -config config/api-server.yaml

# 方式3: 使用环境变量
export DATAFUSION_SERVER_PORT=8081
./bin/api-server

# 验证服务启动
curl http://localhost:8080/healthz
```

**预期输出**:
```json
{
  "status": "healthy",
  "timestamp": "2024-12-08T10:00:00Z",
  "version": "v2.0.0"
}
```

#### 2.2.2 启动 Worker

```bash
# 启动 Worker
./bin/worker -config config/worker.yaml

# 查看日志
tail -f logs/worker.log
```

**预期日志**:
```
2024-12-08 10:00:00 INFO Worker 启动成功
2024-12-08 10:00:00 INFO Worker ID: worker-12345
2024-12-08 10:00:00 INFO Worker 类型: web-rpa
2024-12-08 10:00:00 INFO 开始轮询任务...
```

### 2.3 运行单元测试

```bash
# 运行所有单元测试
go test ./... -v

# 运行特定模块测试
go test ./internal/collector/... -v
go test ./internal/processor/... -v
go test ./internal/storage/... -v

# 运行测试并生成覆盖率报告
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率
open coverage.html
```

### 2.4 运行集成测试

```bash
# 简单功能测试
go run tests/test_simple.go

# 完整流程测试
go run tests/test_with_storage.go

# 数据库采集器测试
go run tests/test_database_collector.go

# MongoDB和去重测试
go run tests/test_mongodb_and_dedup.go
```

---

## 3. 功能验证测试

### 3.1 API Server 功能测试

#### 3.1.1 健康检查

```bash
# 健康检查
curl http://localhost:8080/healthz

# 就绪检查
curl http://localhost:8080/readyz
```

#### 3.1.2 用户认证测试

```bash
# 1. 用户登录
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123"}' \
  | jq -r '.token')

echo "Token: $TOKEN"

# 2. 获取用户信息
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/auth/profile

# 3. 获取用户列表
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/users
```

#### 3.1.3 任务管理测试

```bash
# 1. 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试任务",
    "type": "web-rpa",
    "status": "enabled",
    "cron": "0 */1 * * *",
    "config": {
      "data_source": {
        "type": "web-rpa",
        "url": "https://example.com"
      }
    }
  }'

# 2. 获取任务列表
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tasks

# 3. 获取任务详情
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tasks/1

# 4. 更新任务
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'

# 5. 删除任务
curl -X DELETE http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN"
```

#### 3.1.4 数据源管理测试

```bash
# 1. 创建数据源
curl -X POST http://localhost:8080/api/v1/datasources \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试数据源",
    "type": "web-rpa",
    "config": {
      "url": "https://example.com",
      "selectors": {
        "title": ".title",
        "content": ".content"
      }
    }
  }'

# 2. 测试数据源连接
curl -X POST http://localhost:8080/api/v1/datasources/1/test \
  -H "Authorization: Bearer $TOKEN"

# 3. 获取数据源列表
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/datasources
```

#### 3.1.5 执行历史查询

```bash
# 1. 获取执行历史
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/executions?task_id=1&limit=10"

# 2. 获取执行详情
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/executions/1

# 3. 获取统计信息
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/stats
```

### 3.2 Worker 功能测试

#### 3.2.1 插入测试任务

```bash
# 插入测试任务到数据库
psql -U datafusion -d datafusion_control -f scripts/insert_test_task.sql

# 验证任务已插入
psql -U datafusion -d datafusion_control -c "SELECT id, name, type, status FROM collection_tasks;"
```

#### 3.2.2 观察 Worker 执行

```bash
# 查看 Worker 日志
tail -f logs/worker.log

# 预期看到以下日志:
# - 发现待执行任务
# - 获取任务锁
# - 开始数据采集
# - 数据处理
# - 数据存储
# - 任务完成
```

#### 3.2.3 验证采集结果

```bash
# 查询采集的数据
psql -U datafusion -d datafusion_data -c "SELECT * FROM collected_data LIMIT 10;"

# 查询执行记录
psql -U datafusion -d datafusion_control -c "SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 5;"
```

### 3.3 Web 界面测试

#### 3.3.1 启动 Web 开发服务器

```bash
# 进入 web 目录
cd web

# 安装依赖
npm install

# 启动开发服务器
npm start

# 或使用脚本
cd ..
./scripts/start_web_dev.sh
```

#### 3.3.2 访问 Web 界面

```bash
# 打开浏览器访问
open http://localhost:3000

# 使用默认账户登录
# 用户名: admin
# 密码: Admin@123
```

#### 3.3.3 功能验证清单

- [ ] **登录功能**: 能够成功登录
- [ ] **仪表板**: 显示系统统计信息
- [ ] **任务管理**: 能够创建、编辑、删除任务
- [ ] **数据源管理**: 能够配置数据源
- [ ] **执行历史**: 能够查看任务执行记录
- [ ] **用户管理**: 能够管理用户和权限
- [ ] **系统配置**: 能够修改系统配置
- [ ] **备份管理**: 能够执行备份操作

### 3.4 性能测试

#### 3.4.1 运行性能测试脚本

```bash
# 运行完整性能测试
./scripts/performance_test.sh

# 自定义测试参数
./scripts/performance_test.sh --users 100 --duration 120s
```

#### 3.4.2 查看性能报告

```bash
# 查看测试报告
cat performance_test_report.txt

# 关键指标:
# - API响应时间 (P95, P99)
# - 吞吐量 (QPS)
# - 错误率
# - 资源使用 (CPU, 内存)
```

---

## 4. 生产环境部署

### 4.1 部署前准备

#### 4.1.1 环境检查清单

- [ ] Kubernetes 集群已就绪 (v1.24+)
- [ ] kubectl 已配置并能访问集群
- [ ] Helm 已安装 (v3.0+)
- [ ] PostgreSQL 数据库已准备
- [ ] Redis 服务已准备 (可选)
- [ ] 域名和 SSL 证书已准备
- [ ] 监控系统已部署 (Prometheus + Grafana)

#### 4.1.2 创建命名空间

```bash
# 创建命名空间
kubectl create namespace datafusion

# 验证命名空间
kubectl get namespaces
```

#### 4.1.3 配置 Secret

```bash
# 创建数据库密码 Secret
kubectl create secret generic datafusion-db-secret \
  --from-literal=password='your-secure-password' \
  -n datafusion

# 创建 JWT Secret
kubectl create secret generic datafusion-jwt-secret \
  --from-literal=secret-key='your-jwt-secret-key-min-32-chars' \
  -n datafusion

# 验证 Secret
kubectl get secrets -n datafusion
```

### 4.2 部署 PostgreSQL

#### 4.2.1 使用 Helm 部署

```bash
# 添加 Bitnami Helm 仓库
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# 部署 PostgreSQL
helm install datafusion-postgres bitnami/postgresql \
  --namespace datafusion \
  --set auth.username=datafusion \
  --set auth.password=datafusion123 \
  --set auth.database=datafusion_control \
  --set primary.persistence.size=50Gi

# 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=postgresql \
  -n datafusion \
  --timeout=300s
```

#### 4.2.2 初始化数据库

```bash
# 获取 PostgreSQL Pod 名称
POSTGRES_POD=$(kubectl get pods -n datafusion -l app.kubernetes.io/name=postgresql -o jsonpath='{.items[0].metadata.name}')

# 复制初始化脚本到 Pod
kubectl cp scripts/init_control_db.sql datafusion/$POSTGRES_POD:/tmp/

# 执行初始化脚本
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control -f /tmp/init_control_db.sql

# 验证表已创建
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control -c "\dt"
```

### 4.3 部署 API Server

#### 4.3.1 构建 Docker 镜像

```bash
# 构建 API Server 镜像
docker build -f Dockerfile.api-server -t datafusion-api-server:v2.0.0 .

# 推送到镜像仓库
docker tag datafusion-api-server:v2.0.0 your-registry/datafusion-api-server:v2.0.0
docker push your-registry/datafusion-api-server:v2.0.0
```

#### 4.3.2 部署到 Kubernetes

```bash
# 应用部署配置
kubectl apply -f k8s/api-server-deployment.yaml

# 等待 Pod 就绪
kubectl wait --for=condition=ready pod \
  -l app=datafusion-api-server \
  -n datafusion \
  --timeout=300s

# 查看 Pod 状态
kubectl get pods -n datafusion -l app=datafusion-api-server

# 查看日志
kubectl logs -n datafusion -l app=datafusion-api-server --tail=50
```

#### 4.3.3 配置 Service 和 Ingress

```bash
# 创建 Service
kubectl apply -f k8s/api-server-service.yaml

# 创建 Ingress
kubectl apply -f k8s/api-server-ingress.yaml

# 验证 Service
kubectl get svc -n datafusion

# 验证 Ingress
kubectl get ingress -n datafusion
```

### 4.4 部署 Worker

#### 4.4.1 构建 Worker 镜像

```bash
# 构建 Worker 镜像
docker build -f Dockerfile -t datafusion-worker:v2.0.0 .

# 推送到镜像仓库
docker tag datafusion-worker:v2.0.0 your-registry/datafusion-worker:v2.0.0
docker push your-registry/datafusion-worker:v2.0.0
```

#### 4.4.2 部署 Worker

```bash
# 创建 ConfigMap
kubectl apply -f k8s/worker-config.yaml

# 部署 Worker
kubectl apply -f k8s/worker.yaml

# 查看 Worker 状态
kubectl get pods -n datafusion -l app=datafusion-worker

# 查看 Worker 日志
kubectl logs -n datafusion -l app=datafusion-worker --tail=50 -f
```

### 4.5 部署 Web 前端

#### 4.5.1 构建 Web 镜像

```bash
# 构建 Web 前端镜像
cd web
docker build -t datafusion-web:v2.0.0 .

# 推送到镜像仓库
docker tag datafusion-web:v2.0.0 your-registry/datafusion-web:v2.0.0
docker push your-registry/datafusion-web:v2.0.0
cd ..
```

#### 4.5.2 部署 Web 前端

```bash
# 部署 Web 前端
kubectl apply -f k8s/web-deployment.yaml

# 查看 Web 前端状态
kubectl get pods -n datafusion -l app=datafusion-web

# 查看 Web 前端日志
kubectl logs -n datafusion -l app=datafusion-web --tail=50
```

#### 4.5.3 配置 Web 访问

```bash
# 方式1: 端口转发（开发/测试）
kubectl port-forward -n datafusion svc/web-service 3000:80

# 方式2: NodePort（测试环境）
kubectl patch svc web-service -n datafusion -p '{"spec":{"type":"NodePort"}}'
kubectl get svc web-service -n datafusion

# 方式3: Ingress（生产环境）
# 创建 Ingress 配置
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: datafusion-web-ingress
  namespace: datafusion
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: datafusion.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80
EOF

# 访问 Web 界面
# 浏览器访问 http://localhost:3000 (端口转发)
# 或 http://datafusion.example.com (Ingress)
```

### 4.6 部署监控系统

#### 4.5.1 部署 Prometheus

```bash
# 使用 Helm 部署 Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace datafusion-monitor \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# 应用 Prometheus 规则
kubectl apply -f k8s/monitoring/prometheus-rules.yaml
```

#### 4.5.2 部署 Grafana

```bash
# Grafana 已包含在 kube-prometheus-stack 中
# 获取 Grafana 密码
kubectl get secret -n datafusion-monitor prometheus-grafana \
  -o jsonpath="{.data.admin-password}" | base64 --decode

# 端口转发访问 Grafana
kubectl port-forward -n datafusion-monitor \
  svc/prometheus-grafana 3000:80

# 访问 http://localhost:3000
# 用户名: admin
# 密码: (上面获取的密码)
```

#### 4.6.3 导入 Grafana Dashboard

```bash
# 导入预定义的 Dashboard
kubectl apply -f k8s/monitoring/grafana-dashboard.json
```

---

## 5. 部署后验证

### 5.1 健康检查

```bash
# 检查所有 Pod 状态
kubectl get pods -n datafusion

# 预期输出: 所有 Pod 状态为 Running

# 检查 API Server 健康
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=datafusion-api-server -o jsonpath='{.items[0].metadata.name}') \
  -- curl http://localhost:8080/healthz
```

### 5.2 功能验证

#### 5.2.1 API 访问测试

```bash
# 获取 API Server 地址
API_URL=$(kubectl get ingress -n datafusion datafusion-api-ingress \
  -o jsonpath='{.spec.rules[0].host}')

# 测试健康检查
curl https://$API_URL/healthz

# 测试登录
curl -X POST https://$API_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123"}'
```

#### 5.2.2 Web 界面测试

```bash
# 端口转发 Web 服务
kubectl port-forward -n datafusion svc/web-service 3000:80 &

# 浏览器访问 http://localhost:3000
# 使用默认账户登录: admin / Admin@123

# 功能验证清单:
# - [ ] 能够成功登录
# - [ ] 仪表板显示正常
# - [ ] 任务管理功能正常
# - [ ] 数据源管理功能正常
# - [ ] 用户管理功能正常
# - [ ] 系统配置功能正常
```

#### 5.2.3 Worker 功能测试

```bash
# 插入测试任务
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control -f /tmp/insert_test_task.sql

# 观察 Worker 日志
kubectl logs -n datafusion -l app=datafusion-worker --tail=100 -f

# 验证任务执行
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 5;"
```

### 5.3 性能验证

```bash
# 运行压力测试
kubectl run -n datafusion load-test --image=grafana/k6 --rm -it --restart=Never -- \
  run - <scripts/performance_test.js

# 查看资源使用
kubectl top pods -n datafusion
kubectl top nodes
```

### 5.4 监控验证

```bash
# 访问 Prometheus
kubectl port-forward -n datafusion-monitor svc/prometheus-kube-prometheus-prometheus 9090:9090

# 访问 Grafana
kubectl port-forward -n datafusion-monitor svc/prometheus-grafana 3000:80

# 验证指标采集
# 在 Prometheus 中查询: datafusion_tasks_total
```

---

## 6. 常见问题排查

### 6.1 API Server 无法启动

**症状**: API Server Pod 一直处于 CrashLoopBackOff 状态

**排查步骤**:
```bash
# 1. 查看 Pod 日志
kubectl logs -n datafusion -l app=datafusion-api-server --tail=100

# 2. 查看 Pod 事件
kubectl describe pod -n datafusion -l app=datafusion-api-server

# 3. 检查配置
kubectl get configmap -n datafusion datafusion-config -o yaml

# 4. 检查 Secret
kubectl get secret -n datafusion datafusion-db-secret -o yaml
```

**常见原因**:
- 数据库连接失败: 检查数据库地址和密码
- 配置文件错误: 检查 ConfigMap 配置
- 端口冲突: 检查端口配置
- 资源不足: 检查节点资源

### 6.2 Worker 无法连接数据库

**症状**: Worker 日志显示数据库连接错误

**排查步骤**:
```bash
# 1. 检查数据库服务
kubectl get svc -n datafusion

# 2. 测试数据库连接
kubectl run -n datafusion db-test --image=postgres:14 --rm -it --restart=Never -- \
  psql -h datafusion-postgres-postgresql -U datafusion -d datafusion_control

# 3. 检查 Worker 配置
kubectl get configmap -n datafusion worker-config -o yaml

# 4. 检查网络策略
kubectl get networkpolicies -n datafusion
```

**解决方案**:
- 确认数据库服务名称正确
- 检查数据库用户权限
- 验证网络策略配置
- 检查 DNS 解析

### 6.3 任务不执行

**症状**: 任务已创建但 Worker 不执行

**排查步骤**:
```bash
# 1. 检查任务状态
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT id, name, status, next_run_time FROM collection_tasks;"

# 2. 检查 Worker 日志
kubectl logs -n datafusion -l app=datafusion-worker --tail=100

# 3. 检查任务锁
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT * FROM pg_locks WHERE locktype = 'advisory';"

# 4. 手动触发任务
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "UPDATE collection_tasks SET next_run_time = NOW() WHERE id = 1;"
```

**常见原因**:
- 任务状态为 disabled
- next_run_time 未到期
- Worker 类型不匹配
- 任务锁未释放

### 6.4 性能问题

**症状**: API 响应慢，任务执行缓慢

**排查步骤**:
```bash
# 1. 检查资源使用
kubectl top pods -n datafusion
kubectl top nodes

# 2. 检查数据库性能
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT * FROM pg_stat_activity;"

# 3. 检查慢查询
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT query, calls, total_time, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# 4. 查看 Prometheus 指标
# 访问 Grafana 查看性能图表
```

**优化建议**:
- 增加 Pod 副本数
- 优化数据库查询
- 启用 Redis 缓存
- 调整资源限制

### 6.5 Web 界面无法访问

**症状**: 无法访问 Web 界面或页面加载失败

**排查步骤**:
```bash
# 1. 检查 Web Pod 状态
kubectl get pods -n datafusion -l app=datafusion-web

# 2. 查看 Web Pod 日志
kubectl logs -n datafusion -l app=datafusion-web --tail=100

# 3. 检查 Service
kubectl get svc -n datafusion web-service

# 4. 测试 Service 连接
kubectl run -n datafusion test-web --image=curlimages/curl --rm -it --restart=Never -- \
  curl http://web-service:80

# 5. 检查 Nginx 配置
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=datafusion-web -o jsonpath='{.items[0].metadata.name}') \
  -- cat /etc/nginx/nginx.conf
```

**常见原因**:
- Web Pod 未就绪
- Service 配置错误
- Nginx 配置问题
- API Server 连接失败

**解决方案**:
- 检查 Web Pod 日志
- 验证 Service 端口配置
- 确认 API Server 地址正确
- 检查网络策略

### 6.6 数据丢失

**症状**: 采集的数据未保存或丢失

**排查步骤**:
```bash
# 1. 检查存储配置
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT id, name, config FROM collection_tasks WHERE id = 1;"

# 2. 检查执行记录
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_control \
  -c "SELECT * FROM task_executions WHERE task_id = 1 ORDER BY start_time DESC LIMIT 5;"

# 3. 检查数据表
kubectl exec -n datafusion $POSTGRES_POD -- \
  psql -U datafusion -d datafusion_data \
  -c "SELECT COUNT(*) FROM collected_data;"

# 4. 检查 Worker 日志
kubectl logs -n datafusion -l app=datafusion-worker --tail=200 | grep -i error
```

**常见原因**:
- 存储配置错误
- 数据库权限不足
- 数据处理失败
- 网络问题

---

## 附录

### A. 配置文件模板

#### A.1 API Server 配置

```yaml
# config/api-server.yaml
server:
  port: 8080
  mode: release

database:
  postgresql:
    host: datafusion-postgres-postgresql
    port: 5432
    user: datafusion
    password: ${DB_PASSWORD}
    database: datafusion_control
    sslmode: require
    max_open_conns: 25
    max_idle_conns: 5

auth:
  jwt:
    secret_key: ${JWT_SECRET_KEY}
    token_duration: "24h"
  password:
    min_length: 8
    require_upper: true
    require_lower: true
    require_digit: true

cache:
  type: redis
  redis:
    host: datafusion-redis
    port: 6379
    password: ${REDIS_PASSWORD}
    db: 0

log:
  level: info
  format: json
```

#### A.2 Worker 配置

```yaml
# config/worker.yaml
worker_type: "web-rpa"
poll_interval: 30s

database:
  host: datafusion-postgres-postgresql
  port: 5432
  user: datafusion
  password: ${DB_PASSWORD}
  database: datafusion_control
  sslmode: require

storage:
  type: "postgresql"
  database:
    host: datafusion-postgres-postgresql
    port: 5432
    user: datafusion
    password: ${DB_PASSWORD}
    database: datafusion_data
    sslmode: require

log:
  level: info
  format: json
```

### B. 有用的命令

```bash
# 查看所有资源
kubectl get all -n datafusion

# 查看 Pod 详情
kubectl describe pod -n datafusion <pod-name>

# 进入 Pod
kubectl exec -n datafusion -it <pod-name> -- /bin/sh

# 查看日志
kubectl logs -n datafusion <pod-name> --tail=100 -f

# 端口转发
kubectl port-forward -n datafusion <pod-name> 8080:8080

# 删除 Pod (重启)
kubectl delete pod -n datafusion <pod-name>

# 扩容
kubectl scale deployment -n datafusion datafusion-api-server --replicas=3

# 更新镜像
kubectl set image deployment/datafusion-api-server \
  api-server=your-registry/datafusion-api-server:v2.0.1 \
  -n datafusion

# 回滚
kubectl rollout undo deployment/datafusion-api-server -n datafusion

# 查看回滚历史
kubectl rollout history deployment/datafusion-api-server -n datafusion
```

### C. 监控指标说明

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| datafusion_tasks_total | Counter | 任务总数 |
| datafusion_tasks_running | Gauge | 正在运行的任务数 |
| datafusion_tasks_success_total | Counter | 成功执行的任务数 |
| datafusion_tasks_failed_total | Counter | 失败的任务数 |
| datafusion_task_duration_seconds | Histogram | 任务执行时间 |
| datafusion_api_requests_total | Counter | API 请求总数 |
| datafusion_api_request_duration_seconds | Histogram | API 响应时间 |
| datafusion_worker_active | Gauge | 活跃的 Worker 数量 |
| datafusion_data_collected_total | Counter | 采集的数据总量 |

---

**文档结束**

如有问题，请参考项目文档或联系技术支持。
