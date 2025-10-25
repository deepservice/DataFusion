# DataFusion技术方案设计 3.1节扩展任务追踪

## 任务目标
将产品需求分析文档中的11个用户场景转化为14个技术时序图，扩展技术方案设计文档的3.1节。

## 总体进度: 8/14 (57%)

---

## Session 1 进度 (2025-10-25)

### ✅ 已完成 (4个时序图 - 3.1.1数据采集场景全部完成)

1. **3.1节结构重构**
   - ✅ 添加分组标题和总体说明
   - ✅ 重新组织章节结构为4大类
   - 文件位置: 第389-402行

2. **3.1.1.1 网页数据源配置与RPA采集流程** ✅
   - 场景描述: 用户配置网页数据源，使用RPA测试采集
   - 参与组件: 9个（WebUI, Gateway, Master, PostgreSQL, RabbitMQ, Worker, Playwright, Website等）
   - PlantUML代码: 完整的时序图（约140行）
   - 关键技术点: 6个方面（异步执行、RPA引擎、数据提取、错误处理、性能优化、安全性）
   - 文件位置: 第403-602行
   - 图片引用: `diagrams/seq_web_rpa_collection.png`
   - Git提交: 7407a9d

3. **3.1.1.2 数据库数据源配置与同步流程** ✅
   - 场景描述: 用户配置数据库数据源，测试连接和SQL查询
   - 参与组件: 8个（WebUI, Gateway, Master, PostgreSQL, RabbitMQ, Worker, Source Database等）
   - PlantUML代码: 完整的时序图（约210行）
   - 关键技术点: 6个方面（连接池管理、SQL注入防护、增量同步、大数据分批、跨数据库兼容、安全性）
   - 文件位置: 第604-886行
   - 图片引用: `diagrams/seq_db_sync.png`
   - Git提交: 454cdb1

4. **3.1.1.3 API数据源配置与采集流程** ✅
   - 场景描述: 用户配置RESTful API数据源，测试API调用
   - 参与组件: 8个（WebUI, Gateway, Master, PostgreSQL, RabbitMQ, Worker, HTTP Client, Target API）
   - PlantUML代码: 完整的时序图（约270行）
   - 关键技术点: 6个方面（HTTP客户端、OAuth 2.0流程、JSONPath解析、分页策略、Rate Limiting、Schema推断）
   - 文件位置: 第888-1222行
   - 图片引用: `diagrams/seq_api_collection.png`
   - Git提交: 1a5a55f

5. **3.1.1.4 页面DOM解析与智能字段识别流程** ✅
   - 场景描述: 用户输入URL，系统自动解析页面结构，智能识别数据字段
   - 参与组件: 9个（WebUI, Gateway, Master, PostgreSQL, RabbitMQ, Worker, Playwright, AI Recognizer, Website）
   - PlantUML代码: 完整的时序图（约275行）
   - 关键技术点: 6个方面（DOM序列化、重复模式识别、字段类型推断、选择器生成、置信度评分、可视化编辑器）
   - 文件位置: 第1223-1563行
   - 图片引用: `diagrams/seq_dom_parsing.png`
   - Git提交: 0557526

### 🎉 第一类场景（3.1.1 数据采集）全部完成！

---

## Session 2 进度 (2025-10-26)

### ✅ 已完成 (4个时序图 - 3.1.2任务管理场景全部完成)

1. **3.1.2.1 定时任务创建与调度流程** ✅（重构）
   - 场景描述: 用户创建定时任务，配置Cron表达式，系统调度执行
   - 参与组件: 8个（Web UI, Gateway, Master, PostgreSQL, Scheduler, Redis, etcd）
   - PlantUML代码: 完整重构（约160行，新增分布式锁、状态机等）
   - 关键技术点: 6个方面（Cron解析、etcd分布式锁、调度器状态机、优先级队列、状态同步、高可用架构）
   - 文件位置: DataFusion技术方案设计.md 第665-731行
   - 图片引用: `diagrams/seq_create_task.png` (293KB)
   - Git提交: 待提交

2. **3.1.2.2 任务执行流程** ✅（重构）
   - 场景描述: 调度器推送任务到队列，Worker消费并执行完整数据采集流程
   - 参与组件: 10个（Scheduler, RabbitMQ, Worker, Collector, Processor, Storage, PostgreSQL, Target DB, Monitor, Redis）
   - PlantUML代码: 完整重构（约302行，新增健康检查、优先级队列、实时推送等）
   - 关键技术点: 6个方面（优先级队列、Worker健康检查、状态机、实时推送、资源管理、失败恢复）
   - 文件位置: DataFusion技术方案设计.md 第733-809行
   - 图片引用: `diagrams/seq_execute_task.png` (412KB)
   - Git提交: 待提交

3. **3.1.2.3 手动触发任务执行流程** ✅（新增）
   - 场景描述: 用户点击"立即执行"，WebSocket实时推送日志和状态
   - 参与组件: 9个（Web UI, WebSocket Server, Gateway, Master, PostgreSQL, Redis, RabbitMQ, Worker, Monitor）
   - PlantUML代码: 全新时序图（约240行）
   - 关键技术点: 6个方面（WebSocket推送、任务去重、高优先级队列、并发控制、实时日志流、权限审计）
   - 文件位置: DataFusion技术方案设计.md 第811-888行
   - 图片引用: `diagrams/seq_manual_trigger.png` (428KB)
   - Git提交: 待提交

4. **3.1.2.4 任务失败重试与告警流程** ✅（新增）
   - 场景描述: 任务失败后的自动重试和多渠道告警通知
   - 参与组件: 11个（Worker, PostgreSQL, Redis, Monitor, Alert Manager, Rule Engine, RabbitMQ, Email, SMS, DingTalk, WeChat）
   - PlantUML代码: 全新时序图（约250行）
   - 关键技术点: 6个方面（指数退避算法、重试策略、规则引擎、多渠道通知、告警去重、降级熔断）
   - 文件位置: DataFusion技术方案设计.md 第890-994行
   - 图片引用: `diagrams/seq_retry_alert.png` (435KB)
   - Git提交: 待提交

### 🎉 第二类场景（3.1.2 任务管理）全部完成！

**Session 2 成果统计**:
- 时序图: 4个（2个重构 + 2个新增）
- PlantUML代码: 约950行
- Markdown文档: 约335行
- PNG图片: 4张（合计1.5MB）
- 总体进度: 从29%提升至57%

---

## Session 3 待完成任务

### 📋 3.1.3 数据处理场景（3个）

#### 3.1.3.1 数据清洗规则配置与执行流程
- **场景描述**: 用户为数据源配置清洗规则，Worker自动应用
- **关键流程**: 配置规则 → 测试预览 → 保存 → Worker执行时应用
- **关键技术点**: 清洗规则DSL（expr引擎）、管道式处理

#### 3.1.3.2 数据查询与导出流程
- **场景描述**: 用户查询采集数据，应用筛选，导出CSV
- **关键流程**: 查询参数 → 分页查询 → 应用筛选 → 生成CSV → 返回下载链接
- **关键技术点**: 高效分页、流式导出、临时文件清理

#### 3.1.3.3 错误数据标记与重新采集流程
- **场景描述**: 用户标记错误数据，触发重新采集
- **关键流程**: 标记错误 → 创建补采任务 → Worker重新采集 → 更新记录
- **关键技术点**: 数据版本管理、幂等性保证

---

## Session 4 待完成任务

### 📋 3.1.3 数据处理场景（3个）

#### 3.1.3.1 数据清洗规则配置与执行流程
- **场景描述**: 用户为数据源配置清洗规则，Worker自动应用
- **关键流程**: 配置规则 → 测试预览 → 保存 → Worker执行时应用
- **关键技术点**: 清洗规则DSL（expr引擎）、管道式处理

#### 3.1.3.2 数据查询与导出流程
- **场景描述**: 用户查询采集数据，应用筛选，导出CSV
- **关键流程**: 查询参数 → 分页查询 → 应用筛选 → 生成CSV → 返回下载链接
- **关键技术点**: 高效分页、流式导出、临时文件清理

#### 3.1.3.3 错误数据标记与重新采集流程
- **场景描述**: 用户标记错误数据，触发重新采集
- **关键流程**: 标记错误 → 创建补采任务 → Worker重新采集 → 更新记录
- **关键技术点**: 数据版本管理、幂等性保证

---

## Session 5 待完成任务

### 📋 3.1.4 系统集成场景（3个）

#### 3.1.4.1 MCP协议资源发现与数据查询流程
- **场景描述**: AI应用通过MCP协议发现资源和查询数据
- **关键流程**: resources/list → data/query → 返回标准JSON
- **关键技术点**: MCP Go SDK集成、标准化数据格式

#### 3.1.4.2 MCP数据订阅与实时推送流程
- **场景描述**: AI应用订阅数据更新，自动推送新数据
- **关键流程**: subscribe → 订阅管理 → 数据采集完成 → WebSocket推送
- **关键技术点**: 订阅管理（Redis）、推送条件匹配、WebSocket连接池

#### 3.1.4.3 移动端任务监控与推送通知流程
- **场景描述**: 移动端接收任务失败推送，快速重试
- **关键流程**: 注册设备Token → 任务失败 → Firebase/APNs推送 → 点击跳转 → 手动重试
- **关键技术点**: Firebase Cloud Messaging、APNs集成、Deep Link

---

## 如何继续下一个Session

### 方式1: 直接继续（推荐）
在新session中直接说：
```
继续完成DataFusion技术方案设计文档3.1节的扩展任务。
请查看 design/TODO_3.1_expansion.md 了解当前进度，
从3.1.1.2 数据库数据源配置与同步流程开始。
```

### 方式2: 提供上下文
```
我正在扩展DataFusion技术方案设计文档的3.1节，
已完成3.1.1.1网页RPA采集时序图。
现在需要继续添加3.1.1.2数据库同步时序图。
详细计划见design/TODO_3.1_expansion.md。
```

---

## 注意事项

1. **保持PlantUML格式一致**
   - 使用 `!theme plain` 和白色背景
   - 使用 `autonumber` 自动编号
   - 使用 `note` 添加关键说明
   - 参与者命名清晰（如 "Master\nDataSource Service"）

2. **每个时序图包含的要素**
   - 场景描述（2-3句话）
   - 参与组件列表
   - 时序图（PlantUML代码块）
   - 图片引用（`![场景名](diagrams/xxx.png)`）
   - 关键技术点（6个左右，有序列表）

3. **插入位置**
   - 所有时序图插入到 3.1 节
   - 3.2 节（核心模块设计）保持不变
   - 最后一行应该是 `---` 分隔符

4. **文件备份**
   - 每完成一个session后提交Git
   - commit message格式: `feat: 添加3.1.x场景时序图(进度x/14)`

---

## 预计工作量

- **每个时序图**: 约120-200行（含PlantUML代码、说明、技术点）
- **总预计**: 1500-2000行
- **Session划分**:
  - Session 1: 已完成1个，剩余约150行可完成3个 = 共4个（3.1.1完成）
  - Session 2: 4个（3.1.2完成）
  - Session 3: 3个（3.1.3完成）
  - Session 4: 3个（3.1.4完成）

---

## 最后更新
- **日期**: 2025-10-26
- **最后完成**: Session 2 - 3.1.2 任务管理场景（4个时序图全部完成）
- **下一步**: Session 3 - 3.1.3 数据处理场景（3个时序图）
- **当前进度**: 8/14 (57%)
- **里程碑**:
  - ✅ 3.1.1 数据采集场景（4/4完成）
  - ✅ 3.1.2 任务管理场景（4/4完成）
  - ⏳ 3.1.3 数据处理场景（0/3待完成）
  - ⏳ 3.1.4 系统集成场景（0/3待完成）
- **Session 2成果**:
  - 重构2个时序图（seq_create_task, seq_execute_task）
  - 新增2个时序图（seq_manual_trigger, seq_retry_alert）
  - 新增335行技术文档
  - 生成4张PNG图片（合计1.5MB）
