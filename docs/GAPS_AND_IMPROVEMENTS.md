# DataFusion Worker 功能缺口分析和改进计划

## 📋 当前状态总结

**代码量**: ~1,254 行 Go 代码  
**核心模块**: 7 个（collector, processor, storage, database, models, config, worker）  
**已验证功能**: API 采集、RPA 采集、数据处理、PostgreSQL 存储、K8S 部署

---

## ⚠️ 发现的功能缺口

### 1. 错误处理和重试机制 🔴 重要

**当前状态**: 
- ❌ 没有自动重试机制
- ❌ 任务失败后需要手动重新触发
- ❌ 没有指数退避策略

**影响**: 
- 临时网络故障会导致任务失败
- 需要人工干预
- 降低系统可靠性

**需要补充**:
```go
// internal/worker/retry.go
type RetryPolicy struct {
    MaxRetries     int
    InitialDelay   time.Duration
    MaxDelay       time.Duration
    BackoffFactor  float64
}

func (w *Worker) executeWithRetry(ctx context.Context, task *models.CollectionTask) error {
    // 实现重试逻辑
}
```

---

### 2. 超时控制 🔴 重要

**当前状态**:
- ❌ 采集器有超时配置但未在 Worker 层面统一控制
- ❌ 没有任务级别的超时控制
- ❌ 可能导致任务长时间占用资源

**影响**:
- 任务可能无限期运行
- 资源无法释放
- 影响其他任务执行

**需要补充**:
```go
// 在 executeTask 中添加超时控制
ctx, cancel := context.WithTimeout(ctx, time.Duration(task.ExecutionTimeout)*time.Second)
defer cancel()
```

---

### 3. 优雅关闭 🟡 中等

**当前状态**:
- ⚠️ 收到退出信号后立即停止
- ❌ 正在执行的任务可能被中断
- ❌ 没有等待任务完成的机制

**影响**:
- 数据可能不完整
- 执行记录可能不准确

**需要补充**:
```go
// 在 Worker 中添加优雅关闭
func (w *Worker) Shutdown(ctx context.Context) error {
    // 1. 停止接收新任务
    // 2. 等待当前任务完成
    // 3. 释放资源
}
```

---

### 4. 健康检查 🟡 中等

**当前状态**:
- ❌ 没有健康检查端点
- ❌ K8S 只能通过进程存在判断健康状态
- ❌ 无法检测 Worker 是否正常工作

**影响**:
- 无法及时发现 Worker 异常
- K8S 无法准确判断 Pod 健康状态

**需要补充**:
```go
// cmd/worker/main.go
// 添加 HTTP 健康检查端点
http.HandleFunc("/healthz", healthCheckHandler)
http.HandleFunc("/readyz", readinessCheckHandler)
go http.ListenAndServe(":8080", nil)
```

---

### 5. 指标采集 🟡 中等

**当前状态**:
- ❌ 没有 Prometheus 指标
- ❌ 无法监控任务执行情况
- ❌ 无法进行性能分析

**影响**:
- 缺乏可观测性
- 无法进行性能优化
- 问题排查困难

**需要补充**:
```go
// internal/metrics/metrics.go
var (
    taskExecutionTotal = prometheus.NewCounterVec(...)
    taskDuration = prometheus.NewHistogramVec(...)
    dataRecordsCollected = prometheus.NewCounterVec(...)
)
```

---

### 6. 结构化日志 🟢 低优先级

**当前状态**:
- ⚠️ 使用标准 log 包
- ❌ 日志格式不统一
- ❌ 缺少结构化字段

**影响**:
- 日志分析困难
- 无法有效过滤和搜索

**需要补充**:
```go
// 使用 zap 或 logrus
logger.Info("task execution started",
    zap.String("task_id", taskID),
    zap.String("worker", workerName),
)
```

---

### 7. 配置验证 🟢 低优先级

**当前状态**:
- ❌ 配置加载后没有验证
- ❌ 错误配置可能导致运行时错误

**需要补充**:
```go
func (c *Config) Validate() error {
    if c.PollInterval < time.Second {
        return errors.New("poll_interval too small")
    }
    // 更多验证...
}
```

---

## 🎯 改进后的开发计划

### Phase 1: 补充核心缺失功能（1 周）⭐ 最高优先级

#### 1.1 错误重试机制（2 天）
- [ ] 创建 `internal/worker/retry.go`
- [ ] 实现指数退避重试
- [ ] 集成到 Worker 执行流程
- [ ] 更新配置支持重试参数

#### 1.2 超时控制（1 天）
- [ ] 在 Worker 层面添加统一超时控制
- [ ] 使用 context.WithTimeout
- [ ] 处理超时后的清理工作

#### 1.3 健康检查（1 天）
- [ ] 添加 HTTP 健康检查端点
- [ ] 实现 /healthz 和 /readyz
- [ ] 更新 K8S 配置使用健康检查

#### 1.4 优雅关闭（1 天）
- [ ] 实现 Shutdown 方法
- [ ] 等待当前任务完成
- [ ] 更新 main.go 的退出逻辑

#### 1.5 基础指标（1 天）
- [ ] 添加 Prometheus 指标
- [ ] 暴露 /metrics 端点
- [ ] 记录关键指标

---

### Phase 2: 扩展采集能力（1 周）

#### 2.1 数据库采集器（3 天）
- [ ] 实现 `internal/collector/db_collector.go`
- [ ] 支持 MySQL 和 PostgreSQL
- [ ] 连接池管理
- [ ] 增量同步支持

#### 2.2 增强清洗规则（2 天）
- [ ] 日期格式转换
- [ ] 数字格式化
- [ ] 数据验证规则

#### 2.3 数据去重（2 天）
- [ ] 实现 `internal/processor/deduplicator.go`
- [ ] 内容哈希去重
- [ ] 时间窗口去重

---

### Phase 3: 扩展存储能力（1 周）

#### 3.1 MongoDB 存储（3 天）⭐ 解耦设计
- [ ] 创建 `internal/storage/mongodb_storage.go`
- [ ] 实现 Storage 接口
- [ ] 连接池管理
- [ ] 批量操作优化
- [ ] 配置独立，可选加载

**解耦设计要点**:
```go
// 1. 独立的配置结构
type MongoDBConfig struct {
    URI        string
    Database   string
    Collection string
    // MongoDB 特有配置
}

// 2. 可选注册
if cfg.Storage.Type == "mongodb" {
    mongoStorage, err := storage.NewMongoDBStorage(cfg.Storage.MongoDB)
    if err != nil {
        log.Printf("警告: MongoDB 存储初始化失败: %v", err)
    } else {
        storageFactory.Register(mongoStorage)
    }
}

// 3. 独立的依赖
// go.mod 中使用 build tags
// +build mongodb
```

#### 3.2 存储插件化（2 天）
- [ ] 改进存储工厂
- [ ] 支持动态加载
- [ ] 配置驱动的存储选择

---

### Phase 4: 监控和日志（3 天）

#### 4.1 完善 Prometheus 监控（2 天）
- [ ] 添加更多指标
- [ ] 创建 Grafana Dashboard
- [ ] 配置告警规则

#### 4.2 结构化日志（1 天）
- [ ] 集成 zap 日志库
- [ ] 统一日志格式
- [ ] 添加日志级别控制

---

### Phase 5: 测试完善（1 周）

#### 5.1 单元测试（4 天）
- [ ] Collector 测试
- [ ] Processor 测试
- [ ] Storage 测试
- [ ] Worker 测试

#### 5.2 集成测试（2 天）
- [ ] 端到端测试
- [ ] 性能测试

---

## 📊 优先级矩阵

| 功能 | 重要性 | 紧急性 | 优先级 | 工作量 |
|------|--------|--------|--------|--------|
| 错误重试 | 高 | 高 | P0 | 2天 |
| 超时控制 | 高 | 高 | P0 | 1天 |
| 健康检查 | 高 | 中 | P0 | 1天 |
| 优雅关闭 | 中 | 中 | P1 | 1天 |
| 基础指标 | 中 | 中 | P1 | 1天 |
| 数据库采集器 | 高 | 低 | P1 | 3天 |
| 增强清洗规则 | 中 | 低 | P2 | 2天 |
| 数据去重 | 中 | 低 | P2 | 2天 |
| MongoDB 存储 | 中 | 低 | P2 | 3天 |
| 结构化日志 | 低 | 低 | P3 | 1天 |
| 单元测试 | 中 | 低 | P3 | 4天 |

---

## 🎯 推荐的执行顺序

### 第 1 周：补充核心缺失（P0 优先级）
```
Day 1-2: 错误重试机制
Day 3:   超时控制
Day 4:   健康检查
Day 5:   优雅关闭 + 基础指标
```

### 第 2 周：扩展采集能力（P1 优先级）
```
Day 1-3: 数据库采集器
Day 4-5: 增强清洗规则
```

### 第 3 周：扩展存储能力（P2 优先级）
```
Day 1-3: MongoDB 存储（解耦设计）
Day 4-5: 数据去重机制
```

### 第 4 周：监控和测试（P3 优先级）
```
Day 1-2: 完善监控
Day 3:   结构化日志
Day 4-5: 单元测试
```

---

## 💡 关键决策

### 1. 为什么先补充缺失功能？

**理由**:
- 错误重试、超时控制、健康检查是生产环境必需的
- 这些功能影响系统的可靠性和可用性
- 补充后系统才能真正投入生产使用

### 2. MongoDB 的解耦设计

**设计原则**:
```go
// 1. 独立的包结构
internal/storage/
├── storage.go           # 接口定义
├── postgres_storage.go  # PostgreSQL 实现
├── file_storage.go      # 文件实现
└── mongodb/             # MongoDB 独立包
    ├── mongodb_storage.go
    └── config.go

// 2. 可选编译
// Makefile
build-with-mongodb:
    go build -tags mongodb -o bin/worker cmd/worker/main.go

// 3. 配置驱动
storage:
  type: mongodb  # 或 postgresql, file
  mongodb:       # MongoDB 特有配置
    uri: "mongodb://localhost:27017"
    database: "datafusion"
```

**优势**:
- 不使用 MongoDB 时不需要引入依赖
- 可以独立测试和维护
- 易于扩展其他存储类型

---

## 📋 更新后的 TODO

### 立即开始（本周）
- [ ] 错误重试机制
- [ ] 超时控制
- [ ] 健康检查
- [ ] 优雅关闭
- [ ] 基础指标

### 下周开始
- [ ] 数据库采集器
- [ ] 增强清洗规则
- [ ] MongoDB 存储（解耦）
- [ ] 数据去重

### 后续迭代
- [ ] 完善监控
- [ ] 结构化日志
- [ ] 单元测试
- [ ] Kubernetes Operator
- [ ] Web UI

---

## 🎉 总结

**当前完成度**: 约 70%（核心功能已实现）  
**需要补充**: 约 30%（生产就绪功能）  
**预计时间**: 4 周完成所有 P0-P2 功能  

**关键改进**:
1. ✅ 先补充生产必需的功能（重试、超时、健康检查）
2. ✅ 再扩展采集和存储能力
3. ✅ MongoDB 采用解耦设计，可选加载
4. ✅ 最后完善监控和测试

---

**更新日期**: 2025-12-04  
**审视人**: Kiro AI Assistant  
**状态**: ✅ 审视完成，计划已更新
