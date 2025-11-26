# DataFusion Deployment常驻Worker架构 - 部署指南

本目录包含DataFusion Deployment常驻Worker架构的部署配置文件。

## 📁 文件清单

| 文件名 | 说明 |
|--------|------|
| `postgresql-schema.sql` | PostgreSQL数据库Schema定义（任务配置表、执行记录表、分布式锁表） |
| `rpa-collector-deployment.yaml` | RPA Collector Deployment配置（包含浏览器池） |
| `api-collector-deployment.yaml` | API Collector Deployment配置（轻量级HTTP采集） |
| `db-collector-deployment.yaml` | Database Collector Deployment配置（数据库采集） |
| `collector-service.yaml` | Service、ServiceMonitor、Secret、RBAC配置 |

## 🚀 快速开始

### 前置条件

1. Kubernetes集群（v1.20+）
2. PostgreSQL数据库（v12+）
3. Prometheus Operator（可选，用于Metrics采集）
4. kubectl命令行工具

### 部署步骤

#### 1. 创建PostgreSQL数据库

```bash
# 方式1：直接在PostgreSQL中执行SQL脚本
psql -h <postgres-host> -U <postgres-user> -d postgres -f postgresql-schema.sql

# 方式2：创建数据库后执行
createdb datafusion_control
psql -h <postgres-host> -U <postgres-user> -d datafusion_control -f postgresql-schema.sql
```

#### 2. 创建Kubernetes命名空间

```bash
kubectl create namespace datafusion
```

#### 3. 创建Secret（修改密码）

**重要**：修改`collector-service.yaml`中的`POSTGRES_PASSWORD`为实际密码！

```bash
# 编辑Secret
vim collector-service.yaml  # 修改POSTGRES_PASSWORD

# 应用Secret和相关配置
kubectl apply -f collector-service.yaml
```

#### 4. 部署Collector Deployments（replicas=0，暂不启动）

```bash
# 部署RPA Collector
kubectl apply -f rpa-collector-deployment.yaml

# 部署API Collector
kubectl apply -f api-collector-deployment.yaml

# 部署Database Collector
kubectl apply -f db-collector-deployment.yaml

# 验证Deployment创建成功
kubectl get deployments -n datafusion
```

#### 5. 验证部署

```bash
# 查看Pod状态（应该没有Pod，因为replicas=0）
kubectl get pods -n datafusion

# 查看Service
kubectl get svc -n datafusion

# 查看ServiceAccount和RBAC
kubectl get sa,role,rolebinding -n datafusion
```

#### 6. 启动Worker（测试）

```bash
# 启动1个API Collector副本
kubectl scale deployment api-collector --replicas=1 -n datafusion

# 查看Pod启动状态
kubectl get pods -n datafusion -w

# 查看Pod日志
kubectl logs -f deployment/api-collector -n datafusion

# 检查健康状态
kubectl exec -it deployment/api-collector -n datafusion -- wget -O- http://localhost:8080/healthz
```

## 📊 监控与观测

### Prometheus Metrics

Worker Pod暴露Prometheus Metrics在9090端口：

```bash
# 查看Metrics（示例）
kubectl port-forward deployment/api-collector 9090:9090 -n datafusion

# 访问 http://localhost:9090/metrics
curl http://localhost:9090/metrics
```

### 关键Metrics指标

```promql
# 任务执行总数
datafusion_task_execution_total{collector_type, task_name, status}

# 任务执行耗时
datafusion_task_execution_duration_seconds{collector_type, task_name}

# 数据采集/存储记录数
datafusion_records_fetched_total{collector_type, task_name}
datafusion_records_stored_total{collector_type, task_name}

# 分布式锁指标
datafusion_lock_acquired_total{task_id}
datafusion_lock_contention_total{task_id}

# 浏览器池指标（RPA Collector）
datafusion_browser_pool_size
datafusion_browser_pool_available

# 数据库连接池指标
datafusion_db_pool_open_connections
datafusion_db_pool_idle_connections
```

### 健康检查

```bash
# Liveness Probe
curl http://<pod-ip>:8080/healthz

# Readiness Probe
curl http://<pod-ip>:8080/readyz
```

## ⚙️ 配置调整

### 扩缩容

**手动扩缩容**（推荐）：

```bash
# 扩容到3副本
kubectl scale deployment rpa-collector --replicas=3 -n datafusion

# 缩容到1副本
kubectl scale deployment rpa-collector --replicas=1 -n datafusion

# 缩容到0（暂停）
kubectl scale deployment rpa-collector --replicas=0 -n datafusion
```

**自动扩缩容**（HPA，可选）：

每个Deployment配置文件中已包含HPA定义，基于CPU和内存使用率自动扩缩容。

```bash
# 查看HPA状态
kubectl get hpa -n datafusion

# 查看HPA详情
kubectl describe hpa rpa-collector-hpa -n datafusion
```

### 资源限制调整

编辑Deployment YAML文件，修改`resources`部分：

```yaml
resources:
  requests:
    cpu: "1000m"      # 调整CPU请求
    memory: "2Gi"     # 调整内存请求
  limits:
    cpu: "2000m"      # 调整CPU限制
    memory: "4Gi"     # 调整内存限制
```

应用修改：

```bash
kubectl apply -f rpa-collector-deployment.yaml
```

### 浏览器池配置调整

修改`rpa-collector-deployment.yaml`中的环境变量：

```yaml
env:
- name: BROWSER_POOL_SIZE
  value: "10"  # 增加到10个浏览器实例
- name: BROWSER_MAX_LIFETIME
  value: "60m"  # 延长到60分钟
```

### PostgreSQL连接配置

修改`collector-service.yaml`中的Secret：

```yaml
stringData:
  POSTGRES_HOST: "your-postgres-host"
  POSTGRES_PORT: "5432"
  POSTGRES_DB: "datafusion_control"
  POSTGRES_USER: "datafusion_worker"
  POSTGRES_PASSWORD: "your-strong-password"
  POSTGRES_MAX_CONNS: "50"  # 增加连接数
```

## 🔧 故障排查

### Pod无法启动

```bash
# 查看Pod状态
kubectl get pods -n datafusion

# 查看Pod事件
kubectl describe pod <pod-name> -n datafusion

# 查看Pod日志
kubectl logs <pod-name> -n datafusion
```

### 数据库连接失败

```bash
# 检查Secret配置
kubectl get secret postgresql-credentials -n datafusion -o yaml

# 测试数据库连接（从Pod内）
kubectl exec -it deployment/api-collector -n datafusion -- sh
# 在Pod内执行
psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB
```

### Metrics未被Prometheus抓取

```bash
# 检查ServiceMonitor
kubectl get servicemonitor -n datafusion

# 检查Service annotations
kubectl get svc rpa-collector -n datafusion -o yaml | grep prometheus

# 查看Prometheus targets（Prometheus UI）
# Targets -> datafusion-collectors
```

### 任务未执行

```bash
# 查看Worker日志
kubectl logs deployment/api-collector -n datafusion

# 检查任务配置表
psql -h <postgres-host> -U <postgres-user> -d datafusion_control
SELECT * FROM collection_tasks WHERE enabled = true;

# 检查任务锁表
SELECT * FROM task_locks;

# 检查任务执行记录
SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 10;
```

## 📈 性能优化

### 1. 调整并发数

修改`SCHEDULER_MAX_CONCURRENT_TASKS`环境变量：

```yaml
env:
- name: SCHEDULER_MAX_CONCURRENT_TASKS
  value: "10"  # 增加并发任务数
```

### 2. 调整轮询间隔

修改`SCHEDULER_POLL_INTERVAL`环境变量：

```yaml
env:
- name: SCHEDULER_POLL_INTERVAL
  value: "15s"  # 缩短轮询间隔（更快响应）
```

### 3. 数据库索引优化

```sql
-- 为常用查询创建索引
CREATE INDEX IF NOT EXISTS idx_collection_tasks_next_run_collector
    ON collection_tasks (next_run_time, collector_type)
    WHERE enabled = true;

-- 分析表（更新统计信息）
ANALYZE collection_tasks;
ANALYZE task_executions;
ANALYZE task_locks;
```

### 4. 连接池优化

根据实际负载调整连接池大小：

```yaml
stringData:
  POSTGRES_MAX_CONNS: "50"  # 增加最大连接数
  POSTGRES_MIN_CONNS: "10"  # 增加最小连接数
```

## 🔄 迁移从Job模式

详细迁移步骤请参考主设计文档`DataFusion技术方案设计.md`的3.2.7.7节。

简要步骤：

1. **准备阶段**：创建PostgreSQL表、部署Deployment(replicas=0)
2. **双写阶段**：Controller同时支持Job和PostgreSQL
3. **Canary测试**：选择少量任务测试Deployment模式
4. **灰度迁移**：分批迁移任务
5. **完全切换**：删除Job代码，清理CronJob资源

## 🔐 安全建议

### 生产环境必做

1. **修改默认密码**：修改`postgresql-credentials` Secret中的密码
2. **使用密钥管理系统**：集成Vault、AWS Secrets Manager等
3. **启用网络策略**：限制Pod间通信
4. **定期更新镜像**：修复安全漏洞
5. **配置Pod Security Policy**：限制Pod权限

### Secret管理（推荐）

使用External Secrets Operator从Vault获取密码：

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: postgresql-credentials
  namespace: datafusion
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: postgresql-credentials
  data:
  - secretKey: POSTGRES_PASSWORD
    remoteRef:
      key: secret/datafusion/postgres
      property: password
```

## 📚 相关文档

- [DataFusion技术方案设计.md](../DataFusion技术方案设计.md) - 完整技术方案设计文档
- [PostgreSQL官方文档](https://www.postgresql.org/docs/)
- [Kubernetes官方文档](https://kubernetes.io/docs/)
- [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)

## 🆘 获取帮助

遇到问题？

1. 查看[故障排查](#故障排查)章节
2. 查看Pod日志和事件
3. 查看PostgreSQL日志
4. 联系DataFusion团队

---

**注意**：这是架构优化方案的配置文件，当前系统仍使用Job模式。待资源充足时再进行迁移。
