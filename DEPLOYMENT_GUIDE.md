# 🚀 DataFusion 部署指南

**版本**: v2.0  
**更新日期**: 2024-12-08  

---

## 📋 目录

1. [部署方式概览](#1-部署方式概览)
2. [使用 deploy.sh 自动部署](#2-使用-deploysh-自动部署)
3. [手动部署到 Kubernetes](#3-手动部署到-kubernetes)
4. [本地开发部署](#4-本地开发部署)
5. [验证部署](#5-验证部署)
6. [常见问题](#6-常见问题)

---

## 1. 部署方式概览

DataFusion 提供三种部署方式：

| 部署方式 | 适用场景 | 难度 | 推荐度 |
|---------|---------|------|--------|
| **deploy.sh 自动部署** | Kubernetes 生产环境 | ⭐ 简单 | ⭐⭐⭐⭐⭐ |
| **手动 K8S 部署** | 需要自定义配置 | ⭐⭐ 中等 | ⭐⭐⭐⭐ |
| **本地开发部署** | 开发和测试 | ⭐ 简单 | ⭐⭐⭐ |

---

## 2. 使用 deploy.sh 自动部署

### 2.1 脚本功能

`deploy.sh` 是一个统一的自动化部署脚本，可以：

✅ **自动检查依赖** (kubectl, docker)  
✅ **自动构建 Docker 镜像**  
✅ **自动部署到 Kubernetes**  
✅ **自动等待服务就绪**  
✅ **自动健康检查**  
✅ **显示访问信息**  

### 2.2 前置要求

```bash
# 1. 确保 kubectl 已安装并配置
kubectl version --client

# 2. 确保 Docker 已安装
docker --version

# 3. 确保可以访问 Kubernetes 集群
kubectl cluster-info

# 4. 确保有足够的权限
kubectl auth can-i create deployments --namespace=datafusion
```

### 2.3 快速部署

#### 方式 1: 部署完整系统（推荐）

```bash
# 部署 API Server + Worker + PostgreSQL
./deploy.sh all
```

**执行流程**:
1. ✅ 检查依赖 (kubectl, docker)
2. ✅ 创建 datafusion 命名空间
3. ✅ 部署 PostgreSQL 数据库
4. ✅ 构建 API Server 镜像
5. ✅ 部署 API Server
6. ✅ 构建 Worker 镜像
7. ✅ 部署 Worker
8. ✅ 等待所有 Pod 就绪
9. ✅ 执行健康检查
10. ✅ 显示访问信息

**预计时间**: 5-10 分钟

#### 方式 2: 只部署 API Server

```bash
# 只部署 API Server（不包含 Worker）
./deploy.sh api-server
```

适用场景：
- 只需要 API 服务
- Worker 单独部署
- 测试 API 功能

#### 方式 3: 只部署 Worker

```bash
# 只部署 Worker（包含 PostgreSQL）
./deploy.sh worker
```

适用场景：
- API Server 已部署
- 扩展 Worker 实例
- 测试 Worker 功能

#### 方式 4: 清理后重新部署

```bash
# 清理现有资源后重新部署
./deploy.sh --clean all
```

⚠️ **警告**: 这会删除 datafusion 命名空间下的所有资源！

### 2.4 部署输出示例

```bash
$ ./deploy.sh all

==========================================
DataFusion Kubernetes 部署
==========================================

检查依赖...
✅ 依赖检查通过

创建命名空间...
✅ 命名空间已就绪

部署 PostgreSQL...
configmap/postgres-init-scripts created
deployment.apps/postgresql created
service/postgresql created
等待 PostgreSQL 启动...
pod/postgresql-xxx condition met
✅ PostgreSQL 部署成功

构建 API Server 镜像...
[+] Building 45.2s (15/15) FINISHED
✅ API Server 镜像构建完成

部署 API Server...
configmap/api-server-config created
deployment.apps/api-server created
service/api-server-service created
ingress.networking.k8s.io/api-server-ingress created
等待 API Server 启动...
pod/api-server-xxx condition met
✅ API Server 部署成功

构建 Worker 镜像...
[+] Building 42.1s (14/14) FINISHED
✅ Worker 镜像构建完成

部署 Worker...
configmap/worker-config created
deployment.apps/datafusion-worker created
等待 Worker 启动...
pod/datafusion-worker-xxx condition met
✅ Worker 部署成功

==========================================
部署状态
==========================================

📦 Pods:
NAME                                 READY   STATUS    RESTARTS   AGE
api-server-xxx                       1/1     Running   0          2m
datafusion-worker-xxx                1/1     Running   0          1m
postgresql-xxx                       1/1     Running   0          3m

🔧 Services:
NAME                  TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
api-server-service    ClusterIP   10.96.xxx.xxx   <none>        8080/TCP   2m
postgresql            ClusterIP   10.96.xxx.xxx   <none>        5432/TCP   3m

测试 API Server 健康检查...
✅ API Server 健康检查通过

==========================================
访问信息
==========================================

🔗 API Server:
  内部访问: http://api-server-service.datafusion.svc.cluster.local:8080
  端口转发: kubectl port-forward -n datafusion svc/api-server-service 8081:8080
  然后访问: http://localhost:8081

📝 常用命令:
  查看 Worker 日志: kubectl logs -f -l app=datafusion-worker -n datafusion
  查看 API Server 日志: kubectl logs -f -l app=api-server -n datafusion
  查看 PostgreSQL 日志: kubectl logs -f -l app=postgresql -n datafusion

🗑️  清理部署:
  kubectl delete namespace datafusion

==========================================
✅ 部署完成！
==========================================
```

### 2.5 验证部署

```bash
# 1. 查看所有 Pod 状态
kubectl get pods -n datafusion

# 预期输出: 所有 Pod 状态为 Running

# 2. 端口转发
kubectl port-forward -n datafusion svc/api-server-service 8081:8080 &

# 3. 测试 API
curl http://localhost:8081/healthz

# 预期输出: {"status":"ok"}

# 4. 查看 Worker 日志
kubectl logs -f -l app=datafusion-worker -n datafusion

# 预期看到: Worker 启动日志和任务轮询日志
```

---

## 3. 手动部署到 Kubernetes

如果你需要自定义配置或了解部署细节，可以手动部署。

### 3.1 准备工作

```bash
# 1. 克隆项目
git clone https://github.com/your-org/datafusion.git
cd datafusion

# 2. 检查 Kubernetes 配置文件
ls -la k8s/

# 应该看到:
# - namespace.yaml
# - postgresql.yaml
# - postgres-init-scripts.yaml
# - api-server-deployment.yaml
# - worker-config.yaml
# - worker.yaml
```

### 3.2 步骤 1: 创建命名空间

```bash
kubectl apply -f k8s/namespace.yaml

# 验证
kubectl get namespace datafusion
```

### 3.3 步骤 2: 部署 PostgreSQL

```bash
# 1. 创建初始化脚本 ConfigMap
kubectl apply -f k8s/postgres-init-scripts.yaml

# 2. 部署 PostgreSQL
kubectl apply -f k8s/postgresql.yaml

# 3. 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod \
  -l app=postgresql \
  -n datafusion \
  --timeout=120s

# 4. 验证
kubectl get pods -n datafusion -l app=postgresql
```

### 3.4 步骤 3: 构建 Docker 镜像

#### 方式 A: 本地构建（推荐用于开发）

```bash
# 1. 构建 API Server 镜像
docker build -f Dockerfile.api-server -t datafusion/api-server:latest .

# 2. 构建 Worker 镜像
docker build -t datafusion-worker:latest .

# 3. 如果使用 Minikube，加载镜像
minikube image load datafusion/api-server:latest
minikube image load datafusion-worker:latest
```

#### 方式 B: 推送到镜像仓库（推荐用于生产）

```bash
# 1. 登录镜像仓库
docker login your-registry.com

# 2. 构建并推送 API Server 镜像
docker build -f Dockerfile.api-server -t your-registry.com/datafusion/api-server:v2.0 .
docker push your-registry.com/datafusion/api-server:v2.0

# 3. 构建并推送 Worker 镜像
docker build -t your-registry.com/datafusion/worker:v2.0 .
docker push your-registry.com/datafusion/worker:v2.0

# 4. 更新 K8S 配置文件中的镜像地址
# 编辑 k8s/api-server-deployment.yaml 和 k8s/worker.yaml
# 将 image 字段改为你的镜像地址
```

### 3.5 步骤 4: 部署 API Server

```bash
# 1. 部署 API Server
kubectl apply -f k8s/api-server-deployment.yaml

# 2. 等待 API Server 就绪
kubectl wait --for=condition=ready pod \
  -l app=api-server \
  -n datafusion \
  --timeout=120s

# 3. 验证
kubectl get pods -n datafusion -l app=api-server
kubectl get svc -n datafusion api-server-service
```

### 3.6 步骤 5: 部署 Worker

```bash
# 1. 创建 Worker 配置
kubectl apply -f k8s/worker-config.yaml

# 2. 部署 Worker
kubectl apply -f k8s/worker.yaml

# 3. 等待 Worker 就绪
kubectl wait --for=condition=ready pod \
  -l app=datafusion-worker \
  -n datafusion \
  --timeout=120s

# 4. 验证
kubectl get pods -n datafusion -l app=datafusion-worker
```

### 3.7 步骤 6: 配置访问

#### 方式 A: 端口转发（开发/测试）

```bash
# 转发 API Server 端口
kubectl port-forward -n datafusion svc/api-server-service 8081:8080

# 访问 API
curl http://localhost:8081/healthz
```

#### 方式 B: Ingress（生产）

```bash
# 1. 确保 Ingress Controller 已安装
kubectl get pods -n ingress-nginx

# 2. 查看 Ingress
kubectl get ingress -n datafusion

# 3. 配置 DNS 或 /etc/hosts
# 将 Ingress 地址指向你的域名

# 4. 访问
curl http://your-domain.com/healthz
```

---

## 4. 本地开发部署

适合开发和测试，不需要 Kubernetes。

### 4.1 使用 Docker Compose（最简单）

```bash
# 1. 启动 PostgreSQL
docker run -d --name datafusion-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:14

# 2. 初始化数据库
docker exec -i datafusion-postgres psql -U postgres <<EOF
CREATE DATABASE datafusion_control;
CREATE DATABASE datafusion_data;
EOF

docker exec -i datafusion-postgres psql -U postgres -d datafusion_control < scripts/init_control_db.sql

# 3. 启动 API Server
go build -o bin/api-server ./cmd/api-server
./bin/api-server

# 4. 启动 Worker（可选）
go build -o bin/worker ./cmd/worker
./bin/worker -config config/worker.yaml
```

### 4.2 使用本地 PostgreSQL

```bash
# 1. 创建数据库
createdb datafusion_control
createdb datafusion_data

# 2. 初始化数据库
psql -U postgres -d datafusion_control -f scripts/init_control_db.sql
psql -U postgres -d datafusion_data -f scripts/init_db.sql

# 3. 启动服务
./bin/api-server
./bin/worker -config config/worker.yaml
```

---

## 5. 验证部署

### 5.1 健康检查

```bash
# API Server 健康检查
curl http://localhost:8081/healthz

# 预期输出
{"status":"ok"}
```

### 5.2 功能测试

```bash
# 1. 登录获取 Token
TOKEN=$(curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.token')

# 2. 获取任务列表
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v1/tasks

# 3. 查看系统统计
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v1/stats
```

### 5.3 Worker 验证

```bash
# 1. 插入测试任务
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') \
  -- psql -U postgres -d datafusion_control -f /scripts/insert_test_task.sql

# 2. 查看 Worker 日志
kubectl logs -f -n datafusion -l app=datafusion-worker

# 预期看到:
# - 发现待执行任务
# - 获取任务锁
# - 开始数据采集
# - 任务执行完成
```

---

## 6. 常见问题

### 6.1 deploy.sh 执行失败

**问题**: `kubectl: command not found`

**解决**:
```bash
# 安装 kubectl
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

**问题**: `docker: command not found`

**解决**:
```bash
# 安装 Docker
# 参考: https://docs.docker.com/get-docker/
```

**问题**: `Error from server (Forbidden): ...`

**解决**:
```bash
# 检查 Kubernetes 权限
kubectl auth can-i create deployments --namespace=datafusion

# 如果返回 no，需要联系集群管理员授权
```

### 6.2 镜像构建失败

**问题**: `failed to solve with frontend dockerfile.v0`

**解决**:
```bash
# 1. 检查 Dockerfile 是否存在
ls -la Dockerfile Dockerfile.api-server

# 2. 检查 Docker 版本
docker --version

# 3. 清理 Docker 缓存
docker system prune -a
```

### 6.3 Pod 无法启动

**问题**: `ImagePullBackOff`

**解决**:
```bash
# 1. 检查镜像是否存在
docker images | grep datafusion

# 2. 如果使用 Minikube，加载镜像
minikube image load datafusion/api-server:latest
minikube image load datafusion-worker:latest

# 3. 或者修改 imagePullPolicy
kubectl patch deployment api-server -n datafusion \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"api-server","imagePullPolicy":"Never"}]}}}}'
```

**问题**: `CrashLoopBackOff`

**解决**:
```bash
# 1. 查看 Pod 日志
kubectl logs -n datafusion <pod-name>

# 2. 查看 Pod 事件
kubectl describe pod -n datafusion <pod-name>

# 3. 常见原因:
# - 数据库连接失败
# - 配置文件错误
# - 端口冲突
```

### 6.4 无法访问 API

**问题**: 端口转发后无法访问

**解决**:
```bash
# 1. 检查端口转发是否成功
ps aux | grep "port-forward"

# 2. 检查 Service
kubectl get svc -n datafusion api-server-service

# 3. 检查 Pod 状态
kubectl get pods -n datafusion -l app=api-server

# 4. 重新端口转发
kubectl port-forward -n datafusion svc/api-server-service 8081:8080
```

### 6.5 Worker 不执行任务

**问题**: Worker 日志显示"未发现待执行任务"

**解决**:
```bash
# 1. 检查任务是否已创建
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') \
  -- psql -U postgres -d datafusion_control -c "SELECT id, name, status, next_run_time FROM collection_tasks;"

# 2. 检查任务状态
# status 应该是 'enabled'
# next_run_time 应该 <= NOW()

# 3. 手动更新 next_run_time
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') \
  -- psql -U postgres -d datafusion_control -c "UPDATE collection_tasks SET next_run_time = NOW() WHERE id = 1;"
```

---

## 7. 清理部署

### 7.1 清理 Kubernetes 部署

```bash
# 方式 1: 删除命名空间（推荐）
kubectl delete namespace datafusion

# 方式 2: 逐个删除资源
kubectl delete -f k8s/worker.yaml
kubectl delete -f k8s/worker-config.yaml
kubectl delete -f k8s/api-server-deployment.yaml
kubectl delete -f k8s/postgresql.yaml
kubectl delete -f k8s/postgres-init-scripts.yaml
kubectl delete -f k8s/namespace.yaml
```

### 7.2 清理本地部署

```bash
# 停止进程
pkill -f api-server
pkill -f worker

# 删除 Docker 容器
docker stop datafusion-postgres
docker rm datafusion-postgres

# 删除数据库（可选）
dropdb datafusion_control
dropdb datafusion_data
```

---

## 8. 下一步

部署成功后，你可以：

1. **查看 API 文档**: [docs/CONTROL_PLANE_API.md](docs/CONTROL_PLANE_API.md)
2. **运行测试**: [TESTING_AND_DEPLOYMENT_GUIDE.md](TESTING_AND_DEPLOYMENT_GUIDE.md)
3. **配置监控**: [k8s/monitoring/](k8s/monitoring/)
4. **启动 Web 界面**: [web/README.md](web/README.md)

---

**部署愉快！** 🚀
