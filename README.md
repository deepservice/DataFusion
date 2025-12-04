# DataFusion

<div align="center">

**企业级云原生数据采集与处理平台**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.20+-blue.svg)](https://kubernetes.io/)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/python-3.9+-3776AB.svg)](https://www.python.org/)

[English](README_EN.md) | 简体中文

</div>

---

## 📖 项目简介

**DataFusion** 是一个通用的企业级数据获取与处理平台，专注于从多种异构数据源（网站、API、数据库等）高效、可靠地采集数据，并将其转储到本地存储或目标数据库中。该平台基于 **Kubernetes + Operator** 模式构建，为企业的数据分析、业务决策和人工智能应用提供稳定、高质量、标准化的数据基础。

### 核心价值

- 🎯 **统一数据获取**: 屏蔽不同数据源的技术差异，提供统一的数据采集能力
- 🔄 **灵活数据处理**: 支持数据解析、清洗、转换，确保数据质量
- 🤖 **AI友好**: 原生支持MCP（Model Context Protocol）协议，AI应用可直接消费数据
- 🚀 **企业级能力**: 支持高并发、高可用、可观测的生产环境部署
- ☁️ **云原生架构**: 基于Kubernetes Operator，声明式管理，自动化运维

---

## ✨ 主要特性

### 数据采集

- **多源支持**: 网页（静态/动态）、REST API、数据库（MySQL/PostgreSQL/MongoDB）
- **RPA采集**: 基于Puppeteer/Playwright，支持JavaScript渲染、自定义脚本、代理配置
- **API采集**: 支持多种认证方式（API Key、OAuth2.0、Basic Auth）、自动分页
- **数据库采集**: 支持SQL查询、增量同步、字段映射

### 数据处理

- **智能解析**: 支持HTML、JSON、XML、CSV等多种格式
- **字段提取**: CSS选择器、XPath、正则表达式、JSONPath
- **数据清洗**: 去除标签、格式转换、正则替换、自定义规则
- **质量保证**: 数据校验、去重、增量更新

### 任务调度

- **灵活调度**: 定时（Cron表达式）、周期性、手动触发
- **并发控制**: 任务级并发限制、资源隔离
- **容错机制**: 自动重试、失败告警、超时控制
- **优先级管理**: 支持任务优先级设置

### 云原生架构

- **Kubernetes Operator**: 声明式API，自动化运维
- **共享Worker Pool**: 高资源利用率（70-85%）
- **水平扩展**: 支持HPA自动扩缩容
- **高可用**: 无单点故障，故障自动恢复（30秒-2分钟）

### AI集成

- **MCP协议**: 原生支持Model Context Protocol
- **资源发现**: AI应用可查询所有可用数据源
- **数据查询**: 支持过滤、分页、字段选择
- **数据订阅**: 实时推送新采集的数据（WebSocket/HTTP Callback）

### 可观测性

- **监控**: Prometheus指标暴露，Grafana可视化
- **日志**: 集中式日志收集（ELK Stack）
- **告警**: 邮件、短信、钉钉等多种通知方式
- **追踪**: 任务执行全链路追踪

---

## 🏗️ 系统架构

DataFusion采用云原生分层架构，实现了用户界面、API服务、任务编排、任务执行的清晰分层：

```
┌─────────────────────────────────────────────────────────┐
│                      用户层                              │
│  Web浏览器 | 移动端 | 第三方应用 | AI应用(MCP Client)    │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    接入层 (Ingress)                      │
│         HTTPS加密 | 负载均衡 | 静态资源托管             │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              应用服务层 (任务编排与管理)                  │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │  Operator        │  │  MCP Server      │            │
│  │  Manager         │  │  (AI集成)        │            │
│  │  (2副本)         │  │  (2副本)         │            │
│  └──────────────────┘  └──────────────────┘            │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              任务执行层 (数据采集与处理)                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │ RPA Worker │  │ API Worker │  │ DB Worker  │        │
│  │ (3副本)    │  │ (3副本)    │  │ (3副本)    │        │
│  └────────────┘  └────────────┘  └────────────┘        │
│         共享Worker Pool (自动扩缩容)                     │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   基础设施层                             │
│  PostgreSQL | Prometheus | Grafana | ELK Stack          │
└─────────────────────────────────────────────────────────┘
```

### 核心设计

- **Operator模式**: 使用Kubernetes CRD（CollectionTask、DataSource、CleaningRule）声明式管理
- **共享Worker Pool**: 所有任务共享Worker Pod资源池，提高资源利用率
- **任务调度**: Worker自主轮询PostgreSQL + 分布式锁争抢
- **数据存储**: 单实例多Database（control DB + data DB）

详细架构设计请参考：[技术方案设计文档](design/DataFusion技术方案设计.md)

---

## 🚀 快速开始

### 前置要求

- Kubernetes 1.20+
- Helm 3.0+
- kubectl
- PostgreSQL 12+（可选，也可使用云数据库）

### 安装部署

#### 1. 安装CRD

```bash
kubectl apply -f deploy/crds/
```

#### 2. 安装Operator

```bash
helm install datafusion-operator deploy/helm/datafusion-operator \
  --namespace datafusion-system \
  --create-namespace
```

#### 3. 部署Worker Pool

```bash
helm install datafusion-worker deploy/helm/datafusion-worker \
  --namespace datafusion \
  --create-namespace
```

#### 4. 部署MCP Server（可选）

```bash
helm install datafusion-mcp deploy/helm/datafusion-mcp \
  --namespace datafusion
```

#### 5. 初始化数据库

```bash
kubectl exec -it postgresql-0 -n datafusion -- psql -U datafusion -f /scripts/init-control-db.sql
kubectl exec -it postgresql-0 -n datafusion -- psql -U datafusion -f /scripts/init-data-db.sql
```

### 创建第一个采集任务

```yaml
apiVersion: datafusion.io/v1
kind: CollectionTask
metadata:
  name: my-first-task
  namespace: datafusion
spec:
  dataSourceRef:
    name: example-website
  schedule:
    cron: "0 2 * * *"  # 每天凌晨2点执行
    timezone: "Asia/Shanghai"
  collector:
    type: web-rpa
    replicas: 1
  storage:
    target: postgresql
    database: datafusion_data_default
    table: collected_data
```

应用配置：

```bash
kubectl apply -f my-first-task.yaml
```

查看任务状态：

```bash
kubectl get collectiontask -n datafusion
kubectl describe collectiontask my-first-task -n datafusion
```

---

## 📚 文档

### 设计文档

- [产品需求文档 (PRD)](design/DataFusion产品需求分析文档.md)
- [技术方案设计](design/DataFusion技术方案设计.md)
- [技术设计文档修改总结](design/技术设计文档修改总结.md)

### 架构图

所有架构图和时序图位于 `design/diagrams/` 目录：

- 系统架构图
- Kubernetes Operator部署架构
- 任务调度流程
- 数据采集时序图
- MCP服务架构
- 更多...

### API文档

- Kubernetes CRD API（声明式）
- RESTful API（可选，用于传统系统集成）
- MCP协议接口

---

## 🎯 使用场景

### 场景一：网页数据采集

从医药行业资讯网站采集最新文章，用于舆情分析：

```yaml
apiVersion: datafusion.io/v1
kind: DataSource
metadata:
  name: medical-news
spec:
  type: web-rpa
  connection:
    url: "https://example.com/medical-news"
  rpaConfig:
    browserType: chromium
    headless: true
  selectors:
    title: ".article-title"
    publishTime: ".publish-time"
    content: ".article-content"
```

### 场景二：数据库同步

从合作方MySQL数据库同步销售数据：

```yaml
apiVersion: datafusion.io/v1
kind: DataSource
metadata:
  name: partner-sales-db
spec:
  type: database
  connection:
    host: "partner-db.example.com"
    port: 3306
    database: "sales_db"
    username: "readonly_user"
    passwordSecretRef:
      name: partner-db-secret
      key: password
  query: |
    SELECT product_id, product_name, sales_amount, sales_date
    FROM sales_records
    WHERE sales_date >= '{start_date}'
```

### 场景三：AI应用集成（MCP）

AI应用通过MCP协议查询和订阅数据：

```python
from mcp import Client

# 创建MCP客户端
client = Client("http://datafusion-mcp-server")

# 查询医药资讯数据
data = client.read_resource(
    uri="datafusion://tasks/medical-news",
    filters={"title": {"contains": "新药研发"}},
    limit=10
)

# 订阅数据更新
subscription = client.subscribe(
    uri="datafusion://tasks/medical-news",
    filters={"title": {"contains": "新药"}},
    callback=lambda event: print(f"收到新数据: {event.data}")
)
```

---

## 🔧 配置说明

### Worker Pool配置

在 `values.yaml` 中配置Worker Pool：

```yaml
worker:
  rpa:
    replicas: 3
    resources:
      requests:
        memory: "1Gi"
        cpu: "1"
      limits:
        memory: "2Gi"
        cpu: "2"
  api:
    replicas: 3
  db:
    replicas: 3
  pollInterval: "30s"
  
  # HPA自动扩缩容
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
```

### 数据库配置

```yaml
postgresql:
  enabled: true
  databases:
    - datafusion_control      # 系统元数据库
    - datafusion_data_default # 默认采集数据库
  auth:
    username: datafusion
    password: "your-secure-password"
  persistence:
    size: 100Gi
```

### MCP Server配置

```yaml
mcp:
  enabled: true
  replicas: 2
  service:
    type: ClusterIP
    httpPort: 80
    websocketPort: 8081
```

---

## 📊 监控与运维

### Prometheus指标

DataFusion暴露以下关键指标：

- `datafusion_task_total`: 任务总数
- `datafusion_task_success_total`: 成功任务数
- `datafusion_task_failed_total`: 失败任务数
- `datafusion_task_duration_seconds`: 任务执行时长
- `datafusion_records_collected_total`: 采集数据条数
- `datafusion_worker_pool_size`: Worker Pool大小
- `datafusion_worker_utilization`: Worker资源利用率

### Grafana Dashboard

导入预置的Grafana Dashboard：

```bash
kubectl apply -f deploy/monitoring/grafana-dashboard.yaml
```

### 日志查询

查看Operator日志：

```bash
kubectl logs -f deployment/datafusion-operator-manager -n datafusion-system
```

查看Worker日志：

```bash
kubectl logs -f deployment/rpa-collector-worker -n datafusion
```

查看任务执行日志：

```bash
kubectl logs -f <worker-pod-name> -n datafusion | grep "task_id=<your-task-id>"
```

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 如何贡献

1. Fork本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启Pull Request

### 开发环境搭建

```bash
# 克隆仓库
git clone https://github.com/your-org/datafusion.git
cd datafusion

# 安装开发依赖
make dev-setup

# 运行测试
make test

# 构建镜像
make build
```

### 代码规范

- Go代码遵循 [Effective Go](https://golang.org/doc/effective_go.html)
- Python代码遵循 [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

感谢以下开源项目：

- [Kubernetes](https://kubernetes.io/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [Puppeteer](https://pptr.dev/)
- [Playwright](https://playwright.dev/)
- [PostgreSQL](https://www.postgresql.org/)
- [Prometheus](https://prometheus.io/)
- [Grafana](https://grafana.com/)

---

## 📞 联系我们

- 项目主页: [https://github.com/your-org/datafusion](https://github.com/your-org/datafusion)
- 问题反馈: [GitHub Issues](https://github.com/your-org/datafusion/issues)
- 邮件: datafusion@example.com

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个Star！⭐**

Made with ❤️ by DataFusion Team

</div>
