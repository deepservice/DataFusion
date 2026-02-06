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
7. [kind 环境特别说明](#7-kind-环境特别说明)

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

**支持的 Kubernetes 环境**:
- ✅ **kind** - 自动检测并使用 `kind load docker-image` 加载镜像
- ✅ **minikube** - 自动检测并使用 `minikube image load` 加载镜像
- ✅ **其他 K8S 集群** - 需要手动推送镜像到镜像仓库

**kind 环境说明**:
- deploy.sh 会自动检测 kind 环境
- 构建镜像后会自动使用 `kind load docker-image` 加载到集群
- 无需手动导入镜像或配置镜像仓库

### 2.3 快速部署

#### 方式 1: 部署完整系统（推荐）

```bash
# 部署 API Server + Worker + PostgreSQL + Web 前端
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
8. ✅ 构建 Web 前端镜像
9. ✅ 部署 Web 前端
10. ✅ 等待所有 Pod 就绪
11. ✅ 执行健康检查
12. ✅ 显示访问信息

**预计时间**: 5-10 分钟

#### 方式 2: 只部署 API Server

```bash
# 只部署 API Server（不包含 Worker 和 Web）
./deploy.sh api-server
```

适用场景：
- 只需要 API 服务
- Worker 和 Web 单独部署
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

#### 方式 4: 只部署 Web 前端

```bash
# 只部署 Web 前端
./deploy.sh web
```

适用场景：
- API Server 已部署
- 只更新前端
- 测试前端功能

#### 方式 5: 清理后重新部署

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
# - api-server-xxx
# - datafusion-worker-xxx
# - datafusion-web-xxx (如果部署了 Web)
# - postgresql-xxx

# 2. 端口转发 API Server
kubectl port-forward -n datafusion svc/api-server-service 8081:8080 &

# 3. 测试 API
curl http://localhost:8081/healthz

# 预期输出: {"status":"ok"}

# 4. 端口转发 Web 前端（如果部署了）
kubectl port-forward -n datafusion svc/datafusion-web-service 3000:80 &

# 5. 访问 Web 界面
# 打开浏览器访问: http://localhost:3000
# 默认账户: admin / admin123

# 6. 查看 Worker 日志
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

#### 方式 A: 本地构建（kind/minikube 环境）

```bash
# 1. 构建 API Server 镜像
docker build -f Dockerfile.api-server -t datafusion/api-server:latest .

# 2. 构建 Worker 镜像
docker build -t datafusion-worker:latest .

# 3. 构建 Web 前端镜像
cd web
docker build -t datafusion/web:latest .
cd ..

# 4. 加载镜像到集群
# 如果使用 kind:
kind load docker-image datafusion/api-server:latest
kind load docker-image datafusion-worker:latest
kind load docker-image datafusion/web:latest

# 如果使用 minikube:
minikube image load datafusion/api-server:latest
minikube image load datafusion-worker:latest
minikube image load datafusion/web:latest
```

**kind 环境说明**:
- kind 使用 containerd 作为容器运行时
- 需要使用 `kind load docker-image` 将 Docker 镜像导入到 kind 集群
- 镜像导入后，Pod 的 `imagePullPolicy` 应设置为 `IfNotPresent` 或 `Never`

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

### 3.7 步骤 6: 部署 Web 前端（可选）

#### 构建 Web 前端镜像

```bash
# 1. 进入 web 目录
cd web

# 2. 构建镜像
docker build -t datafusion/web:latest .

# 3. 加载镜像到集群
# 如果使用 kind:
kind load docker-image datafusion/web:latest

# 如果使用 minikube:
minikube image load datafusion/web:latest

# 4. 返回项目根目录
cd ..
```

#### 部署 Web 前端

```bash
# 1. 部署 Web 前端
kubectl apply -f k8s/web-deployment.yaml

# 2. 等待 Web 前端就绪
kubectl wait --for=condition=ready pod \
  -l app=datafusion-web \
  -n datafusion \
  --timeout=120s

# 3. 验证
kubectl get pods -n datafusion -l app=datafusion-web
kubectl get svc -n datafusion datafusion-web-service
```

### 3.8 步骤 7: 配置访问

#### 方式 A: 端口转发（开发/测试）

```bash
# 转发 API Server 端口
kubectl port-forward -n datafusion svc/api-server-service 8081:8080 &

# 访问 API
curl http://localhost:8081/healthz

# 转发 Web 前端端口（如果部署了）
kubectl port-forward -n datafusion svc/datafusion-web-service 3000:80 &

# 访问 Web 界面
# 打开浏览器: http://localhost:3000
# 默认账户: admin / admin123
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

### 4.3 启动 Web 前端（本地开发）

```bash
# 1. 进入 web 目录
cd web

# 2. 安装依赖（首次运行）
npm install

# 3. 启动开发服务器
npm start

# 4. 访问 Web 界面
# 自动打开浏览器: http://localhost:3000
# 默认账户: admin / admin123

# 注意: 确保 API Server 已在 8080 端口运行
# Web 前端会自动代理 API 请求到 http://localhost:8080
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

### 5.3 Web 界面测试

```bash
# 1. 访问 Web 界面
# 浏览器打开: http://localhost:3000

# 2. 登录
# 用户名: admin
# 密码: admin123

# 3. 验证功能
# - 仪表板显示正常
# - 任务管理功能可用
# - 数据源管理功能可用
# - 用户管理功能可用
```

### 5.4 Worker 验证

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

### 6.6 Web 前端无法访问

**问题**: 无法访问 Web 界面或页面显示错误

**解决**:
```bash
# 1. 检查 Web Pod 状态
kubectl get pods -n datafusion -l app=datafusion-web

# 2. 查看 Web Pod 日志
kubectl logs -n datafusion -l app=datafusion-web --tail=100

# 3. 检查 Service
kubectl get svc -n datafusion datafusion-web-service

# 4. 测试 Service 连接
kubectl run -n datafusion test-web --image=curlimages/curl --rm -it --restart=Never -- \
  curl http://datafusion-web-service:80

# 5. 重新端口转发
kubectl port-forward -n datafusion svc/datafusion-web-service 3000:80
```

**问题**: Web 界面无法连接到 API

**解决**:
```bash
# 1. 检查 Nginx 配置中的 API 代理设置
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=datafusion-web -o jsonpath='{.items[0].metadata.name}') \
  -- cat /etc/nginx/nginx.conf

# 2. 确认 API Server Service 地址
kubectl get svc -n datafusion api-server-service

# 3. 测试从 Web Pod 到 API Server 的连接
kubectl exec -n datafusion -it \
  $(kubectl get pod -n datafusion -l app=datafusion-web -o jsonpath='{.items[0].metadata.name}') \
  -- wget -O- http://api-server-service:8080/healthz
```

**问题**: 本地开发时 Web 前端无法启动

**解决**:
```bash
# 1. 检查 Node.js 版本
node --version  # 应该 >= 16

# 2. 清理并重新安装依赖
cd web
rm -rf node_modules package-lock.json
npm install

# 3. 检查端口占用
lsof -i :3000

# 4. 使用其他端口
PORT=3001 npm start
```

---

## 7. 清理部署

### 7.1 清理 Kubernetes 部署

```bash
# 方式 1: 删除命名空间（推荐，会删除所有资源）
kubectl delete namespace datafusion

# 方式 2: 逐个删除资源
kubectl delete -f k8s/web-deployment.yaml
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

# 停止 Web 开发服务器（如果在运行）
pkill -f "npm start"

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

1. **使用 Web 界面**: 访问 http://localhost:3000 进行可视化管理
2. **查看 API 文档**: [docs/CONTROL_PLANE_API.md](docs/CONTROL_PLANE_API.md)
3. **运行测试**: [TESTING_AND_DEPLOYMENT_GUIDE.md](TESTING_AND_DEPLOYMENT_GUIDE.md)
4. **配置监控**: [k8s/monitoring/](k8s/monitoring/)
5. **创建采集任务**: 通过 Web 界面或 API 创建数据采集任务

---

## 9. kind 环境特别说明

### 9.1 kind 简介

kind (Kubernetes IN Docker) 是一个使用 Docker 容器运行本地 Kubernetes 集群的工具，非常适合本地开发和测试。

**kind 的特点**:
- ✅ 使用 containerd 作为容器运行时
- ✅ 轻量级，启动快速
- ✅ 完全兼容 Kubernetes API
- ✅ 支持多节点集群

### 9.2 kind 环境下的镜像管理

#### 问题说明

在 kind 环境中，由于使用 containerd 作为容器运行时，而不是 Docker，因此：

1. **Docker 构建的镜像不会自动在 kind 集群中可用**
2. **需要手动将镜像从 Docker 导入到 kind 集群**
3. **直接拉取远程镜像可能很慢或失败**

#### 解决方案

deploy.sh 脚本已经自动处理了这个问题：

```bash
# deploy.sh 会自动检测 kind 环境
# 构建镜像后自动使用 kind load docker-image 加载
./deploy.sh all
```

**自动化流程**:
1. 检测当前 kubectl context 是否为 kind
2. 使用 Docker 构建镜像
3. 自动执行 `kind load docker-image <镜像名>`
4. 部署到 Kubernetes

### 9.3 手动加载镜像到 kind

如果需要手动操作：

```bash
# 1. 构建镜像
docker build -f Dockerfile.api-server -t datafusion/api-server:latest .
docker build -t datafusion-worker:latest .
docker build -t datafusion/web:latest ./web

# 2. 加载镜像到 kind 集群
kind load docker-image datafusion/api-server:latest
kind load docker-image datafusion-worker:latest
kind load docker-image datafusion/web:latest

# 3. 验证镜像已加载
docker exec -it <kind-node-name> crictl images | grep datafusion

# 获取 kind 节点名称
kubectl get nodes
```

### 9.4 kind 集群创建

如果还没有 kind 集群：

```bash
# 1. 安装 kind
# macOS
brew install kind

# Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# 2. 创建集群
kind create cluster --name datafusion

# 3. 验证集群
kubectl cluster-info --context kind-datafusion

# 4. 设置当前 context
kubectl config use-context kind-datafusion
```

### 9.5 kind 集群配置（高级）

创建支持 Ingress 的 kind 集群：

```bash
# 创建配置文件
cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF

# 使用配置创建集群
kind create cluster --name datafusion --config kind-config.yaml

# 安装 Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

### 9.6 kind 环境常见问题

#### 问题 1: ImagePullBackOff

**原因**: 镜像未加载到 kind 集群

**解决**:
```bash
# 检查镜像是否在 Docker 中
docker images | grep datafusion

# 加载镜像到 kind
kind load docker-image datafusion/api-server:latest

# 或使用 deploy.sh 自动处理
./deploy.sh all
```

#### 问题 2: 镜像拉取策略

**问题**: Pod 尝试从远程仓库拉取镜像

**解决**: 确保 K8S 配置文件中的 `imagePullPolicy` 设置正确

```yaml
# k8s/api-server-deployment.yaml
spec:
  containers:
  - name: api-server
    image: datafusion/api-server:latest
    imagePullPolicy: IfNotPresent  # 或 Never
```

#### 问题 3: 查看 kind 节点中的镜像

```bash
# 1. 获取 kind 节点名称
kubectl get nodes

# 2. 进入 kind 节点
docker exec -it <node-name> bash

# 3. 查看镜像（使用 crictl）
crictl images

# 或直接执行
docker exec -it <node-name> crictl images | grep datafusion
```

#### 问题 4: 清理 kind 集群

```bash
# 删除集群
kind delete cluster --name datafusion

# 重新创建
kind create cluster --name datafusion
```

### 9.7 kind vs minikube vs 生产环境

| 特性 | kind | minikube | 生产环境 |
|-----|------|----------|---------|
| 容器运行时 | containerd | Docker/containerd | containerd/CRI-O |
| 镜像加载 | `kind load docker-image` | `minikube image load` | 镜像仓库 |
| 启动速度 | 快 | 中等 | N/A |
| 资源占用 | 低 | 中等 | 高 |
| 多节点支持 | ✅ | ✅ | ✅ |
| 适用场景 | 本地开发/CI | 本地开发 | 生产部署 |

### 9.8 kind 环境最佳实践

1. **使用 deploy.sh**: 自动处理镜像加载
2. **设置 imagePullPolicy**: 使用 `IfNotPresent` 或 `Never`
3. **定期清理**: 删除不用的镜像和集群
4. **使用标签**: 为镜像打标签便于管理
5. **监控资源**: 注意 Docker Desktop 的资源限制

```bash
# 查看 kind 集群资源使用
kubectl top nodes
kubectl top pods -n datafusion

# 清理未使用的镜像
docker system prune -a
```

---

**部署愉快！** 🚀
