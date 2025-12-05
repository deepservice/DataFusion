# Kubernetes 快速部署指南

## 🎯 目标

在 Kubernetes 中部署 DataFusion Worker，验证数据采集并保存到 PostgreSQL。

## ⚡ 超快速部署（2 步）

### 第 1 步：一键部署

```bash
./deploy-k8s.sh
```

**这个命令会自动完成**：
- ✅ 构建 Docker 镜像
- ✅ 创建 K8S 命名空间
- ✅ 部署 PostgreSQL（包含初始化脚本）
- ✅ 部署 Worker
- ✅ 插入测试任务

**预计时间**: 3-5 分钟

### 第 2 步：验证结果

```bash
# 等待 2 分钟后运行
./verify-k8s.sh
```

**验证内容**：
- ✅ Pods 运行状态
- ✅ 数据库连接
- ✅ 任务执行记录
- ✅ 采集的数据

**预计时间**: 2-3 分钟

## ✅ 成功标志

看到以下输出表示成功：

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

## 📊 查看采集的数据

```bash
# 获取 PostgreSQL Pod 名称
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

# 查看采集的数据
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "SELECT * FROM test_posts;"
```

**预期输出**：

```
 id |                       title                        | user_id |         created_at         
----+----------------------------------------------------+---------+----------------------------
  1 | sunt aut facere repellat provident occaecati...    |       1 | 2025-12-04 19:50:23
  2 | qui est esse                                       |       1 | 2025-12-04 19:50:23
  3 | ea molestias quasi exercitationem repellat...      |       1 | 2025-12-04 19:50:23
  4 | eum et est occaecati                               |       1 | 2025-12-04 19:50:23
  5 | nesciunt quas odio                                 |       1 | 2025-12-04 19:50:23
(5 rows)
```

## 📝 常用命令

### 查看 Worker 日志

```bash
kubectl logs -f -l app=datafusion-worker -n datafusion
```

### 查看任务执行记录

```bash
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT id, task_id, status, records_collected, start_time 
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 5;
"
```

### 手动触发任务

```bash
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
UPDATE collection_tasks SET next_run_time = NOW();
"
```

## 🗑️ 清理部署

```bash
kubectl delete namespace datafusion
```

## 🔧 故障排查

### 问题 1: Worker Pod 无法启动

```bash
# 查看 Pod 状态
kubectl describe pod -l app=datafusion-worker -n datafusion

# 查看日志
kubectl logs -l app=datafusion-worker -n datafusion
```

### 问题 2: 没有采集到数据

```bash
# 1. 检查 Worker 日志
kubectl logs -l app=datafusion-worker -n datafusion | tail -50

# 2. 检查任务状态
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT id, name, status, next_run_time FROM collection_tasks;
"

# 3. 手动触发任务
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
UPDATE collection_tasks SET next_run_time = NOW();
"

# 4. 等待 1 分钟后再次查看数据
```

### 问题 3: PostgreSQL 连接失败

```bash
# 测试连接
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT 1;"

# 检查 Service
kubectl get svc -n datafusion
```

## 📚 详细文档

- [K8S_DEPLOYMENT_GUIDE.md](K8S_DEPLOYMENT_GUIDE.md) - 完整部署指南
- [README.md](README.md) - 项目文档

## 🎉 总结

通过这两个命令，你可以：
1. ✅ 在 K8S 中部署完整系统
2. ✅ 验证数据采集功能
3. ✅ 确认数据保存到 PostgreSQL

**总耗时**: 约 5-8 分钟

---

**快速开始，立即验证！** 🚀
