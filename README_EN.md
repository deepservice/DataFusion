# DataFusion

<div align="center">

**Enterprise-Grade Cloud-Native Data Collection and Processing Platform**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.20+-blue.svg)](https://kubernetes.io/)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/python-3.9+-3776AB.svg)](https://www.python.org/)

English | [简体中文](README.md)

</div>

---

## 📖 Introduction

**DataFusion** is a universal enterprise-grade data acquisition and processing platform that focuses on efficiently and reliably collecting data from various heterogeneous data sources (websites, APIs, databases, etc.) and transferring it to local storage or target databases. Built on **Kubernetes + Operator** pattern, it provides stable, high-quality, and standardized data foundation for enterprise data analysis, business decision-making, and artificial intelligence applications.

### Core Values

- 🎯 **Unified Data Acquisition**: Shields technical differences of various data sources, provides unified data collection capabilities
- 🔄 **Flexible Data Processing**: Supports data parsing, cleaning, and transformation to ensure data quality
- 🤖 **AI-Friendly**: Native support for MCP (Model Context Protocol), AI applications can directly consume data
- 🚀 **Enterprise-Grade**: Supports high concurrency, high availability, and observable production deployments
- ☁️ **Cloud-Native Architecture**: Based on Kubernetes Operator, declarative management, automated operations

---

## ✨ Key Features

### Data Collection

- **Multi-Source Support**: Web pages (static/dynamic), REST APIs, databases (MySQL/PostgreSQL/MongoDB)
- **RPA Collection**: Based on Puppeteer/Playwright, supports JavaScript rendering, custom scripts, proxy configuration
- **API Collection**: Supports multiple authentication methods (API Key, OAuth2.0, Basic Auth), automatic pagination
- **Database Collection**: Supports SQL queries, incremental sync, field mapping

### Data Processing

- **Intelligent Parsing**: Supports HTML, JSON, XML, CSV and other formats
- **Field Extraction**: CSS selectors, XPath, regular expressions, JSONPath
- **Data Cleaning**: Remove tags, format conversion, regex replacement, custom rules
- **Quality Assurance**: Data validation, deduplication, incremental updates

### Task Scheduling

- **Flexible Scheduling**: Timed (Cron expressions), periodic, manual trigger
- **Concurrency Control**: Task-level concurrency limits, resource isolation
- **Fault Tolerance**: Automatic retry, failure alerts, timeout control
- **Priority Management**: Support task priority settings

### Cloud-Native Architecture

- **Kubernetes Operator**: Declarative API, automated operations
- **Shared Worker Pool**: High resource utilization (70-85%)
- **Horizontal Scaling**: Supports HPA auto-scaling
- **High Availability**: No single point of failure, automatic failure recovery (30s-2min)

### AI Integration

- **MCP Protocol**: Native support for Model Context Protocol
- **Resource Discovery**: AI applications can query all available data sources
- **Data Query**: Supports filtering, pagination, field selection
- **Data Subscription**: Real-time push of newly collected data (WebSocket/HTTP Callback)

### Observability

- **Monitoring**: Prometheus metrics exposure, Grafana visualization
- **Logging**: Centralized log collection (ELK Stack)
- **Alerting**: Multiple notification methods including email, SMS, DingTalk
- **Tracing**: Full-chain tracing of task execution

---

## 🏗️ System Architecture

DataFusion adopts a cloud-native layered architecture with clear separation of user interface, API services, task orchestration, and task execution:

```
┌─────────────────────────────────────────────────────────┐
│                      User Layer                          │
│  Web Browser | Mobile | 3rd-party Apps | AI Apps(MCP)   │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                  Ingress Layer                           │
│      HTTPS Encryption | Load Balancing | Static Assets   │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│         Application Layer (Task Orchestration)           │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │  Operator        │  │  MCP Server      │            │
│  │  Manager         │  │  (AI Integration)│            │
│  │  (2 replicas)    │  │  (2 replicas)    │            │
│  └──────────────────┘  └──────────────────┘            │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│         Execution Layer (Data Collection & Processing)   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │ RPA Worker │  │ API Worker │  │ DB Worker  │        │
│  │ (3 replicas)│  │ (3 replicas)│  │ (3 replicas)│      │
│  └────────────┘  └────────────┘  └────────────┘        │
│         Shared Worker Pool (Auto-scaling)                │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                 Infrastructure Layer                     │
│  PostgreSQL | Prometheus | Grafana | ELK Stack          │
└─────────────────────────────────────────────────────────┘
```

### Core Design

- **Operator Pattern**: Declarative management using Kubernetes CRDs (CollectionTask, DataSource, CleaningRule)
- **Shared Worker Pool**: All tasks share Worker Pod resource pool for improved resource utilization
- **Task Scheduling**: Worker autonomous polling of PostgreSQL + distributed lock contention
- **Data Storage**: Single instance multi-database (control DB + data DB)

For detailed architecture design, please refer to: [Technical Design Document](design/DataFusion技术方案设计.md)

---

## 🚀 Quick Start

### Prerequisites

- Kubernetes 1.20+
- Helm 3.0+
- kubectl
- PostgreSQL 12+ (optional, cloud database can be used)

### Installation

#### 1. Install CRDs

```bash
kubectl apply -f deploy/crds/
```

#### 2. Install Operator

```bash
helm install datafusion-operator deploy/helm/datafusion-operator \
  --namespace datafusion-system \
  --create-namespace
```

#### 3. Deploy Worker Pool

```bash
helm install datafusion-worker deploy/helm/datafusion-worker \
  --namespace datafusion \
  --create-namespace
```

#### 4. Deploy MCP Server (Optional)

```bash
helm install datafusion-mcp deploy/helm/datafusion-mcp \
  --namespace datafusion
```

#### 5. Initialize Database

```bash
kubectl exec -it postgresql-0 -n datafusion -- psql -U datafusion -f /scripts/init-control-db.sql
kubectl exec -it postgresql-0 -n datafusion -- psql -U datafusion -f /scripts/init-data-db.sql
```

### Create Your First Collection Task

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
    cron: "0 2 * * *"  # Execute at 2 AM daily
    timezone: "Asia/Shanghai"
  collector:
    type: web-rpa
    replicas: 1
  storage:
    target: postgresql
    database: datafusion_data_default
    table: collected_data
```

Apply the configuration:

```bash
kubectl apply -f my-first-task.yaml
```

Check task status:

```bash
kubectl get collectiontask -n datafusion
kubectl describe collectiontask my-first-task -n datafusion
```

---

## 📚 Documentation

### Design Documents

- [Product Requirements Document (PRD)](design/DataFusion产品需求分析文档.md) (Chinese)
- [Technical Design Document](design/DataFusion技术方案设计.md) (Chinese)
- [Technical Design Modification Summary](design/技术设计文档修改总结.md) (Chinese)

### Architecture Diagrams

All architecture and sequence diagrams are located in the `design/diagrams/` directory:

- System Architecture
- Kubernetes Operator Deployment Architecture
- Task Scheduling Flow
- Data Collection Sequence Diagrams
- MCP Service Architecture
- And more...

### API Documentation

- Kubernetes CRD API (Declarative)
- RESTful API (Optional, for legacy system integration)
- MCP Protocol Interface

---

## 🎯 Use Cases

### Use Case 1: Web Data Collection

Collect latest articles from medical industry news websites for sentiment analysis:

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

### Use Case 2: Database Synchronization

Sync sales data from partner's MySQL database:

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

### Use Case 3: AI Application Integration (MCP)

AI applications query and subscribe to data via MCP protocol:

```python
from mcp import Client

# Create MCP client
client = Client("http://datafusion-mcp-server")

# Query medical news data
data = client.read_resource(
    uri="datafusion://tasks/medical-news",
    filters={"title": {"contains": "drug development"}},
    limit=10
)

# Subscribe to data updates
subscription = client.subscribe(
    uri="datafusion://tasks/medical-news",
    filters={"title": {"contains": "new drug"}},
    callback=lambda event: print(f"New data received: {event.data}")
)
```

---

## 🔧 Configuration

### Worker Pool Configuration

Configure Worker Pool in `values.yaml`:

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
  
  # HPA auto-scaling
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
```

### Database Configuration

```yaml
postgresql:
  enabled: true
  databases:
    - datafusion_control      # System metadata database
    - datafusion_data_default # Default collection database
  auth:
    username: datafusion
    password: "your-secure-password"
  persistence:
    size: 100Gi
```

### MCP Server Configuration

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

## 📊 Monitoring & Operations

### Prometheus Metrics

DataFusion exposes the following key metrics:

- `datafusion_task_total`: Total number of tasks
- `datafusion_task_success_total`: Number of successful tasks
- `datafusion_task_failed_total`: Number of failed tasks
- `datafusion_task_duration_seconds`: Task execution duration
- `datafusion_records_collected_total`: Number of collected records
- `datafusion_worker_pool_size`: Worker Pool size
- `datafusion_worker_utilization`: Worker resource utilization

### Grafana Dashboard

Import the pre-built Grafana Dashboard:

```bash
kubectl apply -f deploy/monitoring/grafana-dashboard.yaml
```

### Log Queries

View Operator logs:

```bash
kubectl logs -f deployment/datafusion-operator-manager -n datafusion-system
```

View Worker logs:

```bash
kubectl logs -f deployment/rpa-collector-worker -n datafusion
```

View task execution logs:

```bash
kubectl logs -f <worker-pod-name> -n datafusion | grep "task_id=<your-task-id>"
```

---

## 🤝 Contributing

We welcome all forms of contributions!

### How to Contribute

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Environment Setup

```bash
# Clone repository
git clone https://github.com/your-org/datafusion.git
cd datafusion

# Install development dependencies
make dev-setup

# Run tests
make test

# Build images
make build
```

### Code Standards

- Go code follows [Effective Go](https://golang.org/doc/effective_go.html)
- Python code follows [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

---

## 🙏 Acknowledgments

Thanks to the following open source projects:

- [Kubernetes](https://kubernetes.io/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [Puppeteer](https://pptr.dev/)
- [Playwright](https://playwright.dev/)
- [PostgreSQL](https://www.postgresql.org/)
- [Prometheus](https://prometheus.io/)
- [Grafana](https://grafana.com/)

---

## 📞 Contact Us

- Project Homepage: [https://github.com/your-org/datafusion](https://github.com/your-org/datafusion)
- Issue Tracker: [GitHub Issues](https://github.com/your-org/datafusion/issues)
- Email: datafusion@example.com

---

<div align="center">

**⭐ If this project helps you, please give us a Star! ⭐**

Made with ❤️ by DataFusion Team

</div>
