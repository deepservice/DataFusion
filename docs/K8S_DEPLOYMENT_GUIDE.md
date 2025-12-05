# DataFusion Worker Kubernetes 部署指南

## 🎯 目标

在 Kubernetes 集群中部署 DataFusion Worker 和 PostgreSQL，验证完整的数据采集流程。

## 📋 前置条件

- ✅ Kubernetes 集群（本地 Minikube/Kind 或云端集群）
- ✅ kubectl 已配置
- ✅ Docker 已安装
- ✅ 集群有足够资源（至少 1GB 内存）

## 🚀 快速部署（一键部署）

```bash
# 一键部署所有组件
./deploy-k8s.sh
```

这个脚本会自动完成：
1. ✅ 构建 Worker Docker 镜像
2. ✅ 创建 Kubernetes 命名空间
3. ✅ 部署 PostgreSQL 数据库
4. ✅ 初始化数据库表结构
5. ✅ 插入测试任务
6. ✅ 部署 Worker

## ✅ 验证部署（一键验证）

```bash
# 等待 2-3 分钟后，运行验证脚本
./verify-k8s.sh
```

验证脚本会检查：
1. ✅ Pods 运行状态
2. ✅ 数据库连接
3. ✅ 任务配置
4. ✅ Worker 日志
5. ✅ 任务执行记录
6. ✅ 采集的数据

## 📊 预期结果

### 成功的验证输出

```
==========================================
验证结果
==========================================
✅ 验证成功！
   - Worker 正常运行
   - 任务执行成功
   - 数据已保存到 PostgreSQL

📝 采集到 5 条数据
```

### 采集的数据示例

```sql
 id |                       title                        | user_id |         created_at         
----+----------------------------------------------------+---------+----------------------------
  1 | sunt aut facere repellat provident occaecati...    |       1 | 2025-12-04 19:50:23.456789
  2 | qui est esse                                       |       1 | 2025-12-04 19:50:23.456789
  3 | ea molestias quasi exercitationem repellat...      |       1 | 2025-12-04 19:50:23.456789
  4 | eum et est occaecati                               |       1 | 2025-12-04 19:50:23.456789
  5 | nesciunt quas odio                                 |       1 | 2025-12-04 19:50:23.456789
```

## 📁 部署文件说明

### 目录结构

```
k8s/
├── namespace.yaml                  # 命名空间定义
├── postgresql.yaml                 # PostgreSQL 部署
├── postgres-init-scripts.yaml      # 数据库初始化脚本
├── worker-config.yaml              # Worker 配置
└── worker.yaml                     # Worker 部署
```

### 核心配置

#### 1. PostgreSQL 配置

- **镜像**: postgres:14-alpine
- **存储**: emptyDir（临时存储，用于测试）
- **资源**: 256Mi 内存，250m CPU
- **数据库**:
  - `datafusion_control`: 存储任务配置和执行记录
  - `datafusion_data`: 存储采集的数据

#### 2. Worker 配置

- **镜像**: datafusion-worker:latest
- **副本数**: 1
- **资源**: 256Mi 内存，250m CPU
- **轮询间隔**: 30 秒
- **任务类型**: API 采集

#### 3. 测试任务

- **名称**: K8S测试-API采集
- **类型**: API
- **数据源**: https://jsonplaceholder.typicode.com/posts?_limit=5
- **执行频率**: 每 2 分钟
- **存储**: PostgreSQL (datafusion_data.test_posts)

## 🔍 手动部署步骤

如果你想手动部署，可以按以下步骤操作：

### 步骤 1: 构建 Docker 镜像

```bash
docker build -t datafusion-worker:latest .
```

### 步骤 2: 创建命名空间

```bash
kubectl apply -f k8s/namespace.yaml
```

### 步骤 3: 部署 PostgreSQL

```bash
kubectl apply -f k8s/postgres-init-scripts.yaml
kubectl apply -f k8s/postgresql.yaml

# 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod -l app=postgresql -n datafusion --timeout=120s
```

### 步骤 4: 部署 Worker

```bash
kubectl apply -f k8s/worker-config.yaml
kubectl apply -f k8s/worker.yaml

# 等待 Worker 就绪
kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s
```

### 步骤 5: 查看部署状态

```bash
kubectl get pods -n datafusion
kubectl get svc -n datafusion
```

## 📝 常用操作命令

### 查看日志

```bash
# Worker 日志
kubectl logs -f -l app=datafusion-worker -n datafusion

# PostgreSQL 日志
kubectl logs -f -l app=postgresql -n datafusion
```

### 查看数据

```bash
# 获取 PostgreSQL Pod 名称
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 查看任务配置
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT * FROM collection_tasks;"

# 查看任务执行记录
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 5;"

# 查看采集的数据
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "SELECT * FROM test_posts;"
```

### 手动触发任务

```bash
# 更新任务的下次执行时间为当前时间
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "UPDATE collection_tasks SET next_run_time = NOW() WHERE id = 1;"
```

### 进入容器

```bash
# 进入 Worker 容器
kubectl exec -it -n datafusion $(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

# 进入 PostgreSQL 容器
kubectl exec -it -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control
```

## 🔧 故障排查

### Worker 无法启动

```bash
# 查看 Pod 状态
kubectl describe pod -l app=datafusion-worker -n datafusion

# 查看日志
kubectl logs -l app=datafusion-worker -n datafusion
```

常见问题：
- 镜像拉取失败：确保镜像已构建
- 配置错误：检查 ConfigMap 配置
- 资源不足：增加集群资源

### PostgreSQL 连接失败

```bash
# 测试数据库连接
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT 1;"

# 检查 Service
kubectl get svc -n datafusion
```

### 任务不执行

```bash
# 检查任务配置
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT id, name, status, next_run_time FROM collection_tasks;"

# 检查 Worker 日志
kubectl logs -f -l app=datafusion-worker -n datafusion

# 手动触发任务
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "UPDATE collection_tasks SET next_run_time = NOW();"
```

### 数据未保存

```bash
# 检查表是否存在
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "\dt"

# 检查任务执行状态
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT * FROM task_executions WHERE status='failed';"
```

## 🗑️ 清理部署

### 删除所有资源

```bash
kubectl delete namespace datafusion
```

### 删除 Docker 镜像

```bash
docker rmi datafusion-worker:latest
```

## 📈 性能调优

### 增加 Worker 副本数

```bash
kubectl scale deployment datafusion-worker -n datafusion --replicas=3
```

### 调整资源限制

编辑 `k8s/worker.yaml`：

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1"
```

### 调整轮询间隔

编辑 `k8s/worker-config.yaml`：

```yaml
poll_interval: 15s  # 从 30s 改为 15s
```

## 🔐 生产环境建议

### 1. 使用持久化存储

替换 emptyDir 为 PersistentVolumeClaim：

```yaml
volumes:
- name: postgres-storage
  persistentVolumeClaim:
    claimName: postgres-pvc
```

### 2. 使用 Secret 管理密码

```bash
kubectl create secret generic postgres-secret \
  --from-literal=password=your-secure-password \
  -n datafusion
```

### 3. 配置资源限制

根据实际负载调整 CPU 和内存限制。

### 4. 启用监控

集成 Prometheus 和 Grafana 监控。

### 5. 配置备份

定期备份 PostgreSQL 数据。

## 📚 相关文档

- [README.md](README.md) - 项目主文档
- [QUICKSTART.md](QUICKSTART.md) - 快速开始指南
- [VERIFICATION_SUCCESS.md](VERIFICATION_SUCCESS.md) - 本地验证报告

## 🎉 总结

通过本指南，你可以：
1. ✅ 在 Kubernetes 中部署完整的 DataFusion Worker 系统
2. ✅ 验证数据采集、处理、存储的完整流程
3. ✅ 确认数据已保存到 PostgreSQL 数据库

所有组件都已容器化，可以轻松扩展和管理！

---

**部署时间**: 约 5-10 分钟  
**验证时间**: 约 2-3 分钟  
**状态**: ✅ 生产就绪
