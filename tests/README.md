# DataFusion Tests

本目录包含 DataFusion Worker 的各类测试。

## 📁 目录结构

```
tests/
├── README.md                    # 本文档
├── test_simple.go              # 简单功能测试
├── test_with_storage.go        # 完整流程测试
├── unit/                       # 单元测试（待添加）
├── integration/                # 集成测试（待添加）
└── e2e/                        # 端到端测试（待添加）
```

## 🧪 现有测试

### 1. 简单功能测试 (test_simple.go)

测试 API 采集器和数据处理器的基本功能。

**运行方式**：
```bash
go run tests/test_simple.go
```

**测试内容**：
- ✅ API 数据采集
- ✅ JSON 数据解析
- ✅ 数据清洗规则应用

### 2. 完整流程测试 (test_with_storage.go)

测试完整的数据采集、处理、存储流程。

**运行方式**：
```bash
go run tests/test_with_storage.go
```

**测试内容**：
- ✅ API 数据采集
- ✅ 数据处理
- ✅ 文件存储

## 📝 计划添加的测试

### 单元测试 (unit/)

针对各个模块的单元测试：

```
unit/
├── collector/
│   ├── api_collector_test.go
│   └── rpa_collector_test.go
├── processor/
│   └── processor_test.go
├── storage/
│   ├── postgres_storage_test.go
│   └── file_storage_test.go
└── worker/
    └── worker_test.go
```

**运行方式**：
```bash
go test ./tests/unit/...
```

### 集成测试 (integration/)

测试多个模块的集成：

```
integration/
├── collector_processor_test.go
├── processor_storage_test.go
└── full_pipeline_test.go
```

**运行方式**：
```bash
go test ./tests/integration/...
```

### 端到端测试 (e2e/)

测试完整的业务场景：

```
e2e/
├── api_collection_test.go
├── rpa_collection_test.go
└── database_collection_test.go
```

**运行方式**：
```bash
go test ./tests/e2e/...
```

## 🚀 快速开始

### 运行所有测试

```bash
# 运行简单测试
go run tests/test_simple.go

# 运行完整流程测试
go run tests/test_with_storage.go

# 运行单元测试（待实现）
go test ./tests/unit/... -v

# 运行所有测试（待实现）
go test ./tests/... -v
```

### 测试覆盖率

```bash
# 生成覆盖率报告（待实现）
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📊 测试数据

测试使用的数据源：

1. **JSONPlaceholder API**
   - URL: https://jsonplaceholder.typicode.com
   - 用途: API 采集测试
   - 优点: 免费、稳定、无需认证

2. **本地测试数据**
   - 位置: tests/testdata/
   - 用途: 单元测试
   - 格式: JSON, HTML, CSV

## 🔧 测试配置

测试配置文件：

```
tests/
├── config/
│   ├── test_config.yaml      # 测试配置
│   └── mock_data.json        # 模拟数据
└── fixtures/
    ├── sample_html.html      # HTML 样本
    └── sample_json.json      # JSON 样本
```

## 📝 编写测试指南

### 单元测试示例

```go
package collector_test

import (
    "context"
    "testing"
    
    "github.com/datafusion/worker/internal/collector"
    "github.com/datafusion/worker/internal/models"
)

func TestAPICollector_Collect(t *testing.T) {
    // 创建采集器
    c := collector.NewAPICollector(30)
    
    // 准备配置
    config := &models.DataSourceConfig{
        Type: "api",
        URL:  "https://jsonplaceholder.typicode.com/posts/1",
        // ...
    }
    
    // 执行采集
    ctx := context.Background()
    data, err := c.Collect(ctx, config)
    
    // 断言
    if err != nil {
        t.Fatalf("采集失败: %v", err)
    }
    
    if len(data) == 0 {
        t.Error("未采集到数据")
    }
}
```

### 集成测试示例

```go
package integration_test

import (
    "context"
    "testing"
    
    "github.com/datafusion/worker/internal/collector"
    "github.com/datafusion/worker/internal/processor"
)

func TestCollectorAndProcessor(t *testing.T) {
    // 1. 采集数据
    c := collector.NewAPICollector(30)
    data, err := c.Collect(ctx, config)
    if err != nil {
        t.Fatal(err)
    }
    
    // 2. 处理数据
    p := processor.NewProcessor(processorConfig)
    processed, err := p.Process(data)
    if err != nil {
        t.Fatal(err)
    }
    
    // 3. 验证结果
    if len(processed) != len(data) {
        t.Error("处理后数据量不匹配")
    }
}
```

## 🎯 测试目标

- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试覆盖核心流程
- [ ] 端到端测试覆盖主要场景
- [ ] 所有测试可自动化运行
- [ ] CI/CD 集成

## 📚 相关文档

- [README.md](../README.md) - 项目主文档
- [docs/WORKER_IMPLEMENTATION.md](../docs/WORKER_IMPLEMENTATION.md) - Worker 实现说明

---

**测试是保证代码质量的关键！** 🧪
