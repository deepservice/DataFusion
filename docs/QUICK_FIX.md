# 快速修复指南

## 🐛 问题

Worker 第一次执行成功，后续执行都失败，原因是主键冲突。

## ✅ 快速修复（3 步）

### 第 1 步：应用代码修复

代码已经修复完成，包括：
- ✅ 使用 `ON CONFLICT DO NOTHING` 处理主键冲突
- ✅ 区分重复数据和真正的错误
- ✅ 改进日志输出

### 第 2 步：重新部署

```bash
./update-k8s.sh
```

按提示选择是否清理旧数据（建议选择 `y`）。

### 第 3 步：验证修复

```bash
# 等待 2 分钟后运行
./verify-k8s.sh
```

## 📊 预期结果

### 修复前

```
任务执行记录:
 id | status | records_collected | error_message
----+--------+-------------------+------------------
  3 | failed |                 5 | 数据存储失败: ...
  2 | failed |                 5 | 数据存储失败: ...
  1 | success|                 5 | 
```

### 修复后

```
任务执行记录:
 id | status  | records_collected | error_message
----+---------+-------------------+---------------
  5 | success |                 5 | 
  4 | success |                 5 | 
  3 | success |                 5 | 
  2 | success |                 5 | 
  1 | success |                 5 | 
```

### Worker 日志

**修复前**：
```
❌ 插入数据失败: duplicate key value violates unique constraint
❌ 数据存储失败
```

**修复后**：
```
✅ 数据存储完成，成功: 0 条，重复: 5 条，失败: 0 条
✅ 任务执行完成
```

## 🔍 排查问题

如果还有问题，运行：

```bash
./debug-k8s.sh
```

这会显示：
- 任务执行记录
- 详细错误信息
- Worker 日志
- 数据库中的数据

## 📝 详细说明

查看 [FIX_DUPLICATE_KEY_ISSUE.md](FIX_DUPLICATE_KEY_ISSUE.md) 了解：
- 问题的根本原因
- 详细的修复方案
- 其他解决方案

---

**快速修复，立即生效！** 🚀
