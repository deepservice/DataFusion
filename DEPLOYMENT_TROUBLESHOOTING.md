# DataFusion 部署问题排查指南

本文档记录了常见的部署问题及其解决方案。

## 问题 1: PostgreSQL Pod 启动超时 (ImagePullBackOff)

### 症状
```bash
error: timed out waiting for the condition on pods/postgresql-95fd48958-xxx
```

查看 Pod 状态显示：
```bash
kubectl get pods -n datafusion
NAME                         READY   STATUS             RESTARTS   AGE
postgresql-95fd48958-xxx     0/1     ImagePullBackOff   0          5m
```

### 原因
- kind 集群无法从 Docker Hub 拉取 `postgres:14-alpine` 镜像
- 网络问题导致连接 Docker Hub 超时

### 解决方案

**方案 1：使用最新版本的 deploy.sh（已自动处理）**
```bash
./deploy.sh --clean all
```

**方案 2：手动解决**
```bash
# 1. 在宿主机拉取镜像
docker pull postgres:14-alpine

# 2. 加载到 kind 集群
kind load docker-image postgres:14-alpine --name dev

# 3. 删除失败的 pod
kubectl delete pod -n datafusion -l app=postgresql

# 4. 等待新 pod 启动
kubectl wait --for=condition=ready pod -l app=postgresql -n datafusion --timeout=60s
```

---

## 问题 2: Worker Pod 启动超时 (Readiness probe failed: 503)

### 症状
```bash
error: timed out waiting for the condition on pods/datafusion-worker-xxx
```

查看日志显示：
```bash
2026/02/23 18:47:52 查询待执行任务失败: pq: relation "collection_tasks" does not exist
```

### 原因
- PostgreSQL 数据库初始化脚本未执行
- 控制面数据库 `datafusion_control` 缺少表结构

### 解决方案

**方案 1：使用最新版本的 deploy.sh（已自动处理）**
```bash
./deploy.sh --clean all
```

**方案 2：手动修复**
```bash
POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 执行初始化脚本
kubectl exec -n datafusion "$POSTGRES_POD" -- \
  psql -U datafusion -d datafusion_control \
  -f /docker-entrypoint-initdb.d/01-init-tables.sql

kubectl exec -n datafusion "$POSTGRES_POD" -- \
  psql -U datafusion -d datafusion_control \
  -f /docker-entrypoint-initdb.d/02-insert-test-data.sql
```

---

## 问题 3: Worker 警告 "database datafusion_data does not exist"

### 症状
Worker pod 运行正常，但日志显示警告：
```bash
kubectl logs -n datafusion -l app=datafusion-worker
2026/02/24 05:22:11 警告: 创建 PostgreSQL 存储失败: 连接数据库失败: pq: database "datafusion_data" does not exist
```

### 原因
- Worker 需要两个数据库：
  - `datafusion_control` - 控制面数据库（存储任务配置）
  - `datafusion_data` - 数据面数据库（存储采集的数据）
- 初始化脚本中使用了错误的 SQL 语法（`CREATE DATABASE IF NOT EXISTS` 是 MySQL 语法，PostgreSQL 不支持）

### 解决方案

**方案 1：手动创建数据库**
```bash
POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 创建 datafusion_data 数据库
kubectl exec -n datafusion "$POSTGRES_POD" -- \
  psql -U datafusion -d postgres -c "CREATE DATABASE datafusion_data;"

# 验证
kubectl exec -n datafusion "$POSTGRES_POD" -- \
  psql -U datafusion -d postgres -c "\l"
```

**方案 2：使用最新版本的 deploy.sh（已自动处理）**
最新的 `deploy.sh` 会自动检查并创建 `datafusion_data` 数据库。

---

## 问题 4: 无法进入 Worker Pod (bash not found)

### 症状
```bash
kubectl exec -it datafusion-worker-xxx -n datafusion -- /bin/bash
# 错误: exec: "/bin/bash": stat /bin/bash: no such file or directory

kubectl exec -it datafusion-worker-xxx -n datafusion -- bash
# 错误: exec: "bash": executable file not found in $PATH
```

### 原因
- Worker 镜像基于 Alpine Linux，默认只有 `sh`，没有 `bash`

### 解决方案

**使用 `sh` 代替 `bash`：**
```bash
# 正确的方式
kubectl exec -it -n datafusion <pod-name> -- sh

# 或者获取 pod 名称后进入
POD_NAME=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it -n datafusion $POD_NAME -- sh
```

**进入后可以执行的操作：**
```bash
# 查看工作目录
ls -la /app

# 查看配置
cat /app/config/worker.yaml

# 查看进程
ps aux

# 查看网络连接
netstat -an

# 测试数据库连接（需要安装 postgresql-client）
# 注意：默认镜像没有 psql，需要单独安装或使用 PostgreSQL pod
```

---

## 验证部署状态

使用验证脚本：
```bash
./scripts/verify_deployment.sh
```

---

## 常用排查命令

### 进入各个 Pod

```bash
# Worker (使用 sh)
kubectl exec -it -n datafusion $(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}') -- sh

# API Server (使用 sh)
kubectl exec -it -n datafusion $(kubectl get pod -n datafusion -l app=api-server -o jsonpath='{.items[0].metadata.name}') -- sh

# PostgreSQL (使用 bash，因为 postgres 镜像有 bash)
kubectl exec -it -n datafusion $(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- bash

# 或者直接进入 psql
kubectl exec -it -n datafusion $(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U datafusion -d datafusion_control
```

### 查看日志
```bash
# 实时查看 Worker 日志
kubectl logs -f -l app=datafusion-worker -n datafusion

# 查看最近 50 行
kubectl logs -n datafusion <pod-name> --tail=50

# 查看之前的容器日志（如果 pod 重启过）
kubectl logs -n datafusion <pod-name> --previous
```

### 检查数据库
```bash
POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 列出所有数据库
kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d postgres -c "\l"

# 列出 datafusion_control 数据库的所有表
kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -c "\dt"

# 查询用户
kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -c "SELECT * FROM users;"

# 查询数据源
kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -c "SELECT * FROM data_sources;"

# 查询任务
kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -c "SELECT * FROM collection_tasks;"
```

### 重启组件
```bash
# 重启 Worker
kubectl rollout restart deployment/datafusion-worker -n datafusion

# 删除 Pod（会自动重建）
kubectl delete pod -n datafusion -l app=datafusion-worker
```

---

## 清理部署

```bash
# 完全清理
./deploy.sh --clean

# 仅删除命名空间
kubectl delete namespace datafusion
```

---

## 已知问题和限制

1. **PostgreSQL 初始化脚本语法问题**
   - 旧版本使用了 MySQL 语法 `CREATE DATABASE IF NOT EXISTS`
   - PostgreSQL 不支持此语法，已在新版本修复

2. **PostgreSQL 使用临时存储**
   - 当前使用 `emptyDir`，Pod 重启后数据丢失
   - 生产环境需要配置 PVC

3. **Worker 镜像基于 Alpine**
   - 只有 `sh`，没有 `bash`
   - 需要使用 `kubectl exec ... -- sh` 进入

4. **镜像拉取依赖网络**
   - 如果网络不稳定，建议提前拉取并加载镜像

---

## 最佳实践

### 1. 部署前检查
```bash
# 检查 Kubernetes 集群状态
kubectl cluster-info

# 检查节点状态
kubectl get nodes

# 检查是否有足够的资源
kubectl top nodes
```

### 2. 使用验证脚本
每次部署后运行验证脚本确保所有组件正常：
```bash
./scripts/verify_deployment.sh
```

### 3. 查看完整的部署日志
部署时保存日志方便排查：
```bash
./deploy.sh --clean all 2>&1 | tee deploy.log
```

### 4. 定期备份数据库
```bash
POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 备份控制面数据库
kubectl exec -n datafusion "$POSTGRES_POD" -- \
  pg_dump -U datafusion datafusion_control > backup_control.sql

# 备份数据面数据库
kubectl exec -n datafusion "$POSTGRES_POD" -- \
  pg_dump -U datafusion datafusion_data > backup_data.sql
```

---

## 获取帮助

如果遇到问题：

1. 运行验证脚本：`./scripts/verify_deployment.sh`
2. 收集日志：
   ```bash
   kubectl get pods -n datafusion -o wide > pods-status.txt
   kubectl logs -n datafusion -l app=datafusion-worker --tail=100 > worker-logs.txt
   kubectl logs -n datafusion -l app=postgresql --tail=100 > postgres-logs.txt
   ```
3. 检查事件：
   ```bash
   kubectl get events -n datafusion --sort-by='.lastTimestamp'
   ```
