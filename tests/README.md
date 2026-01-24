# DataFusion Tests

本目录包含 DataFusion 的各类测试。

## 📁 目录结构

```
tests/
├── README.md                    # 本文档
├── test_simple.go              # 简单功能测试
├── test_with_storage.go        # 完整流程测试
├── test_api_server.sh          # API Server 测试脚本
├── test_database_collector.go  # 数据库采集器测试
├── test_mongodb_and_dedup.go   # MongoDB 和去重测试
├── unit/                       # 单元测试
└── data/                       # 测试数据（输出会被忽略）
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

### 3. API Server 测试 (test_api_server.sh)

测试控制面 API Server 的所有端点。

**前置条件**：
- PostgreSQL 运行中（Docker 或本地）
- API Server 运行在端口 8081

**运行方式**：
```bash
# 确保 API Server 正在运行
./bin/api-server &

# 运行测试
./tests/test_api_server.sh
```

**测试内容**：
- ✅ 健康检查（/healthz, /readyz）
- ✅ 数据源管理（CRUD 操作）
- ✅ 清洗规则管理（CRUD 操作）
- ✅ 任务管理（CRUD + 启动/停止）
- ✅ 执行历史查询
- ✅ 统计信息展示

### 4. 数据库采集器测试 (test_database_collector.go)

测试数据库采集器和增强清洗功能。

**运行方式**：
```bash
go run tests/test_database_collector.go
```

**测试内容**：
- ✅ MySQL 采集器（需要数据库）
- ✅ PostgreSQL 采集器（需要数据库）
- ✅ 增强清洗规则

### 5. MongoDB 和去重测试 (test_mongodb_and_dedup.go)

测试 MongoDB 存储和数据去重功能。

**运行方式**：
```bash
go run tests/test_mongodb_and_dedup.go
```

**测试内容**：
- ✅ MongoDB 存储（需要 MongoDB）
- ✅ 内容哈希去重
- ✅ 字段去重
- ✅ 时间窗口去重

## 🚀 快速开始

### 运行所有测试

```bash
# 运行简单测试（无需数据库）
go run tests/test_simple.go

# 运行完整流程测试（无需数据库）
go run tests/test_with_storage.go

# 运行单元测试
go test ./tests/unit/... -v

# 运行 API Server 测试（需要启动 API Server）
./tests/test_api_server.sh
```

### 测试覆盖率

```bash
# 生成覆盖率报告
go test ./tests/unit/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📊 测试数据

测试使用的数据源：

1. **JSONPlaceholder API**
   - URL: https://jsonplaceholder.typicode.com
   - 用途: API 采集测试
   - 优点: 免费、稳定、无需认证

2. **本地测试数据**
   - 位置: tests/data/
   - 用途: 单元测试
   - 格式: JSON, HTML, CSV

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

## 🎯 测试目标

- [x] 单元测试覆盖率 > 70%
- [x] 集成测试覆盖核心流程
- [x] 端到端测试覆盖主要场景
- [x] 所有测试可自动化运行

## 📚 相关文档

- [../README.md](../README.md) - 项目主文档
- [../docs/WORKER_IMPLEMENTATION.md](../docs/WORKER_IMPLEMENTATION.md) - Worker 实现说明

---

**测试是保证代码质量的关键！** 🧪
