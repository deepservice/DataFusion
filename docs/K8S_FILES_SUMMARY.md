# Kubernetes 部署文件清单

## 📦 创建的文件

### Kubernetes 配置文件（k8s/）

1. **namespace.yaml** - 命名空间定义
   - 创建 `datafusion` 命名空间

2. **postgresql.yaml** - PostgreSQL 部署
   - Deployment: 1 副本
   - Service: ClusterIP
   - ConfigMap: 数据库配置
   - 资源: 256Mi 内存，250m CPU

3. **postgres-init-scripts.yaml** - 数据库初始化
   - 创建数据库（datafusion_control, datafusion_data）
   - 创建表结构（collection_tasks, task_executions, test_posts）
   - 插入测试任务

4. **worker-config.yaml** - Worker 配置
   - 数据库连接配置
   - 采集器配置
   - 存储配置

5. **worker.yaml** - Worker 部署
   - Deployment: 1 副本
   - 资源: 256Mi 内存，250m CPU
   - 挂载配置和数据目录

### 部署脚本

1. **deploy-k8s.sh** - 一键部署脚本
   - 构建 Docker 镜像
   - 部署所有 K8S 资源
   - 等待服务就绪
   - 显示部署状态

2. **verify-k8s.sh** - 验证脚本
   - 检查 Pods 状态
   - 验证数据库连接
   - 查看任务执行记录
   - 检查采集的数据
   - 显示统计信息

### 文档

1. **K8S_DEPLOYMENT_GUIDE.md** - 完整部署指南
   - 详细部署步骤
   - 配置说明
   - 故障排查
   - 性能调优

2. **K8S_QUICK_START.md** - 快速开始指南
   - 2 步快速部署
   - 常用命令
   - 快速故障排查

3. **K8S_FILES_SUMMARY.md** - 本文档

## 📊 部署架构

```
datafusion namespace
├── PostgreSQL
│   ├── Deployment (1 replica)
│   ├── Service (ClusterIP)
│   ├── ConfigMap (postgres-config)
│   └── ConfigMap (postgres-init-scripts)
│       ├── 01-init-databases.sql
│       ├── 02-init-tables.sql
│       ├── 03-insert-test-task.sql
│       └── 04-create-data-tables.sql
│
└── Worker
    ├── Deployment (1 replica)
    ├── ConfigMap (worker-config)
    └── Volumes
        ├── config (ConfigMap)
        └── data (emptyDir)
```

## 🔄 数据流

```
1. Worker 启动
   ↓
2. 连接 PostgreSQL (postgresql.datafusion.svc.cluster.local:5432)
   ↓
3. 轮询任务表 (collection_tasks)
   ↓
4. 获取任务锁
   ↓
5. 执行数据采集
   ├─ API: https://jsonplaceholder.typicode.com/posts?_limit=5
   ├─ 解析 JSON 数据
   └─ 应用清洗规则
   ↓
6. 保存到 PostgreSQL
   └─ 数据库: datafusion_data
       └─ 表: test_posts
   ↓
7. 更新执行记录 (task_executions)
   ↓
8. 释放任务锁
   ↓
9. 等待下次执行（2 分钟后）
```

## 📈 资源配置

### PostgreSQL

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Worker

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 总资源需求

- **最小**: 512Mi 内存，500m CPU
- **最大**: 1Gi 内存，1 CPU

## 🎯 测试任务配置

```json
{
  "data_source": {
    "type": "api",
    "url": "https://jsonplaceholder.typicode.com/posts?_limit=5",
    "method": "GET",
    "selectors": {
      "id": "id",
      "title": "title",
      "body": "body",
      "userId": "userId"
    }
  },
  "processor": {
    "cleaning_rules": [
      {"field": "title", "type": "trim"},
      {"field": "body", "type": "trim"}
    ]
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
}
```

## ✅ 验证清单

- [ ] Docker 镜像构建成功
- [ ] Namespace 创建成功
- [ ] PostgreSQL Pod 运行正常
- [ ] Worker Pod 运行正常
- [ ] 数据库连接成功
- [ ] 任务配置正确
- [ ] 任务执行成功
- [ ] 数据保存到 PostgreSQL
- [ ] 可以查询到采集的数据

## 🚀 快速使用

```bash
# 1. 部署
./deploy-k8s.sh

# 2. 等待 2 分钟

# 3. 验证
./verify-k8s.sh

# 4. 查看数据
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "SELECT * FROM test_posts;"

# 5. 清理
kubectl delete namespace datafusion
```

## 📝 文件大小统计

| 文件类型 | 数量 | 说明 |
|---------|------|------|
| YAML 配置 | 5 | K8S 资源定义 |
| Shell 脚本 | 2 | 部署和验证脚本 |
| Markdown 文档 | 3 | 使用指南 |
| **总计** | **10** | |

## 🎉 总结

所有 Kubernetes 部署文件已创建完成，包括：

1. ✅ 完整的 K8S 配置文件
2. ✅ 自动化部署脚本
3. ✅ 自动化验证脚本
4. ✅ 详细的使用文档

可以立即开始在 Kubernetes 中部署和验证！

---

**创建日期**: 2025-12-04  
**文件总数**: 10  
**状态**: ✅ 就绪
