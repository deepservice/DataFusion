# DataFusion 项目待办事项

## 📋 当前状态

- ✅ 产品需求文档已完成
- ✅ 技术设计文档已完成（P0和P1级别）
- ✅ README文档已更新
- ⏸️ 开发阶段准备中

---

## 🎯 短期任务（1-2周）

### 设计评审
- [ ] 组织技术评审会议
  - [ ] 邀请开发团队成员
  - [ ] 准备评审材料（PPT/文档）
  - [ ] 确认架构设计方案
  - [ ] 收集反馈意见

### 数据库设计
- [ ] 评审数据库表结构
  - [ ] `collection_tasks` 表
  - [ ] `task_executions` 表
  - [ ] `data_sources` 表
  - [ ] `cleaning_rules` 表
  - [ ] `field_mappings` 表
  - [ ] `mcp_subscriptions` 表
- [ ] 创建数据库初始化脚本
  - [ ] `init-control-db.sql`
  - [ ] `init-data-db.sql`
  - [ ] `create-task-tables.sql`
  - [ ] `create-mcp-tables.sql`

### 原型验证
- [ ] 实现Worker Pool核心逻辑原型
  - [ ] Worker轮询机制
  - [ ] PostgreSQL分布式锁
  - [ ] 任务争抢逻辑
- [ ] 实现任务调度原型
  - [ ] Cron表达式解析
  - [ ] next_run_time计算
  - [ ] 任务状态更新

---

## 🚀 中期任务（1个月）

### Operator开发
- [ ] 初始化Operator项目（Kubebuilder）
- [ ] 实现CRD定义
  - [ ] CollectionTask CRD
  - [ ] DataSource CRD
  - [ ] CleaningRule CRD
- [ ] 实现Controller
  - [ ] CollectionTaskController
  - [ ] DataSourceController
  - [ ] CleaningRuleController
- [ ] 实现Reconcile逻辑
  - [ ] syncCRToDB
  - [ ] syncDBToStatus
  - [ ] reconcileHPA
- [ ] 实现Admission Webhook
  - [ ] Validation Webhook
  - [ ] Conversion Webhook（可选）

### Worker开发
- [ ] 实现RPA采集器
  - [ ] Puppeteer/Playwright集成
  - [ ] 浏览器池管理
  - [ ] 页面渲染和数据提取
  - [ ] 反爬虫策略
- [ ] 实现API采集器
  - [ ] HTTP客户端封装
  - [ ] 多种认证方式支持
  - [ ] 自动分页处理
  - [ ] JSONPath数据提取
- [ ] 实现数据库采集器
  - [ ] 数据库连接池
  - [ ] SQL查询执行
  - [ ] 增量同步逻辑
  - [ ] 字段映射

### 数据处理引擎
- [ ] 实现数据解析器
  - [ ] HTML解析（CSS选择器/XPath）
  - [ ] JSON解析（JSONPath）
  - [ ] XML解析
  - [ ] CSV解析
- [ ] 实现数据清洗引擎
  - [ ] 清洗规则DSL设计
  - [ ] expr表达式引擎集成
  - [ ] 预置清洗规则
  - [ ] 自定义清洗规则
- [ ] 实现数据存储模块
  - [ ] PostgreSQL写入
  - [ ] MongoDB写入
  - [ ] 文件写入（JSON/CSV）
  - [ ] 批量写入优化

### MCP Server开发
- [ ] 实现MCP协议处理器
  - [ ] resources/list
  - [ ] resources/read
  - [ ] tools/list
  - [ ] tools/call
- [ ] 实现资源映射器
  - [ ] CollectionTask → MCP Resource
  - [ ] DataSource → MCP Resource
- [ ] 实现查询引擎
  - [ ] SQL查询构建器
  - [ ] 过滤和分页
  - [ ] 结果格式化
- [ ] 实现订阅管理器
  - [ ] WebSocket连接池
  - [ ] 事件分发器
  - [ ] 订阅持久化

### 测试
- [ ] 单元测试
  - [ ] Controller单元测试
  - [ ] Worker单元测试
  - [ ] MCP Server单元测试
- [ ] 集成测试
  - [ ] Operator + Worker集成测试
  - [ ] Worker + PostgreSQL集成测试
  - [ ] MCP Server + Worker集成测试
- [ ] 端到端测试
  - [ ] 完整采集流程测试
  - [ ] 故障恢复测试
  - [ ] 性能测试

---

## 📱 长期任务（2-3个月）

### 前端开发
- [ ] 编写前端技术设计文档
  - [ ] 前端架构设计
  - [ ] 组件划分
  - [ ] 状态管理方案（Vuex/Pinia）
  - [ ] 路由设计
  - [ ] API封装
- [ ] 实现核心页面
  - [ ] 数据源管理页面
  - [ ] 任务管理页面
  - [ ] 任务执行监控页面
  - [ ] 数据查看页面
  - [ ] 清洗规则配置页面
- [ ] 实现高级功能
  - [ ] 页面可视化展示（DOM树、元素高亮）
  - [ ] 智能字段识别
  - [ ] 实时日志查看
  - [ ] 数据导出

### 移动端开发
- [ ] 编写移动端技术设计文档
  - [ ] 技术选型（PWA/原生App/React Native）
  - [ ] 响应式布局设计
  - [ ] 推送通知实现
  - [ ] 离线支持
- [ ] 实现核心功能
  - [ ] 任务监控
  - [ ] 任务控制
  - [ ] 告警处理
  - [ ] 数据查看
- [ ] 实现推送通知
  - [ ] Web Push API集成
  - [ ] 通知分级管理
  - [ ] 通知历史记录

### 部署和运维
- [ ] 创建Helm Chart
  - [ ] datafusion-operator Chart
  - [ ] datafusion-worker Chart
  - [ ] datafusion-mcp Chart
  - [ ] 完整的values.yaml配置
- [ ] 编写部署文档
  - [ ] 安装指南
  - [ ] 配置指南
  - [ ] 升级指南
  - [ ] 故障排查指南
- [ ] 监控和告警
  - [ ] Prometheus指标定义
  - [ ] Grafana Dashboard设计
  - [ ] AlertManager告警规则
  - [ ] 日志采集配置（ELK）

### 文档完善
- [ ] 用户文档
  - [ ] 快速开始指南
  - [ ] 用户手册
  - [ ] 最佳实践
  - [ ] FAQ
- [ ] 开发者文档
  - [ ] 开发环境搭建
  - [ ] 代码结构说明
  - [ ] API参考
  - [ ] 插件开发指南
- [ ] 运维文档
  - [ ] 部署架构说明
  - [ ] 性能调优指南
  - [ ] 备份恢复方案
  - [ ] 安全加固指南

---

## 🔮 未来规划

### 功能增强
- [ ] 智能字段识别算法优化
- [ ] 数据质量评分系统
- [ ] 自动化数据清洗建议
- [ ] 数据血缘追踪
- [ ] 数据版本管理

### 性能优化
- [ ] 分布式采集优化
- [ ] 数据库连接池优化
- [ ] 缓存策略优化
- [ ] 批量写入优化

### 扩展性
- [ ] 插件化架构完善
- [ ] 自定义采集器插件
- [ ] 自定义解析器插件
- [ ] 自定义存储器插件

### AI集成增强
- [ ] MCP协议语义查询
- [ ] 自然语言查询支持
- [ ] 数据推荐系统
- [ ] 智能任务调度

---

## 📝 注意事项

### 开发优先级
1. **P0**: Operator + Worker核心功能（必须完成）
2. **P1**: MCP Server + 基础监控（强烈建议）
3. **P2**: 前端 + 移动端（后续迭代）
4. **P3**: 高级功能和优化（长期规划）

### 技术债务
- [ ] 前端设计文档缺失（待补充）
- [ ] 移动端设计文档缺失（待补充）
- [ ] 智能字段识别算法设计缺失（待补充）
- [ ] 数据质量保证机制设计不够详细（待补充）

### 风险管理
- [ ] 定期技术评审（每2周）
- [ ] 代码质量检查（每次PR）
- [ ] 性能测试（每个迭代）
- [ ] 安全审计（上线前）

---

## 📊 进度跟踪

### 里程碑

- **M1: 设计完成** (✅ 已完成)
  - 产品需求文档
  - 技术设计文档
  - README文档

- **M2: 核心功能开发** (⏸️ 准备中)
  - Operator开发
  - Worker开发
  - 数据处理引擎

- **M3: MCP集成** (⏸️ 待开始)
  - MCP Server开发
  - 客户端SDK
  - 集成测试

- **M4: 前端开发** (⏸️ 待开始)
  - 前端设计
  - 核心页面实现
  - 高级功能实现

- **M5: 生产就绪** (⏸️ 待开始)
  - 性能优化
  - 安全加固
  - 文档完善
  - 生产部署

---

**最后更新**: 2025-12-04  
**下次评审**: 待定
