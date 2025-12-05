# 修复主键冲突问题

## 🐛 问题描述

在 Kubernetes 环境中运行 Worker 时，发现：
- 第一次执行成功，数据正常保存到 PostgreSQL
- 后续执行都失败，`task_executions` 表中状态为 `failed`
- 错误原因：主键冲突（duplicate key）

## 🔍 根本原因

测试任务每次都从同一个 API 获取相同的数据（id: 1-5），由于 `test_posts` 表的主键是 `id`，第二次插入时会发生主键冲突，导致：

1. PostgreSQL 抛出 `duplicate key value violates unique constraint` 错误
2. 事务回滚，所有数据都不保存
3. Worker 将任务标记为 `failed`

## ✅ 解决方案

### 修改 1: 使用 `ON CONFLICT DO NOTHING`

在 `internal/storage/postgres_storage.go` 中，修改 INSERT 语句：

```go
// 修改前
query := fmt.Sprintf(
    "INSERT INTO %s (%s) VALUES (%s)",
    config.Table,
    strings.Join(fields, ", "),
    strings.Join(placeholders, ", "),
)

// 修改后
query := fmt.Sprintf(
    "INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
    config.Table,
    strings.Join(fields, ", "),
    strings.Join(placeholders, ", "),
)
```

**效果**：当遇到主键冲突时，PostgreSQL 会忽略该条记录，而不是抛出错误。

### 修改 2: 改进错误处理和日志

```go
// 统计插入结果
successCount := 0      // 成功插入的记录数
duplicateCount := 0    // 重复的记录数
errorCount := 0        // 真正失败的记录数

// 检查每条记录的插入结果
result, execErr := stmt.ExecContext(ctx, values...)
if execErr != nil {
    errorCount++
    continue
}

rowsAffected, _ := result.RowsAffected()
if rowsAffected > 0 {
    successCount++
} else {
    duplicateCount++  // ON CONFLICT DO NOTHING 导致没有插入
}

// 改进的日志输出
log.Printf("数据存储完成，成功: %d 条，重复: %d 条，失败: %d 条", 
    successCount, duplicateCount, errorCount)

// 只有全部失败才返回错误
if successCount == 0 && duplicateCount == 0 && errorCount > 0 {
    return fmt.Errorf("所有数据插入失败")
}
```

**效果**：
- 区分"重复数据"和"真正的错误"
- 只要有数据成功插入或者是重复数据，就认为任务成功
- 只有全部失败才标记为失败

### 修改 3: 优化事务处理

```go
// 修改前
defer tx.Rollback()

// 修改后
defer func() {
    if err != nil {
        tx.Rollback()
    }
}()
```

**效果**：只在真正出错时才回滚事务。

## 🚀 应用修复

### 方法 1: 使用更新脚本（推荐）

```bash
./update-k8s.sh
```

这个脚本会：
1. ✅ 重新构建 Docker 镜像
2. ✅ 重启 Worker Pod
3. ✅ 可选：清理旧的测试数据
4. ✅ 显示新的日志

### 方法 2: 手动更新

```bash
# 1. 重新构建镜像
docker build -t datafusion-worker:latest .

# 2. 删除旧的 Worker Pod（会自动重启）
kubectl delete pod -l app=datafusion-worker -n datafusion

# 3. 等待新 Pod 就绪
kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s

# 4. 查看日志
kubectl logs -f -l app=datafusion-worker -n datafusion
```

## ✅ 验证修复

### 1. 运行调试脚本

```bash
./debug-k8s.sh
```

查看：
- 任务执行记录（应该看到 `success` 状态）
- 详细的日志输出（应该看到"重复: X 条"）

### 2. 查看 Worker 日志

```bash
kubectl logs -f -l app=datafusion-worker -n datafusion
```

**修复前的日志**：
```
插入数据失败: pq: duplicate key value violates unique constraint "test_posts_pkey"
数据存储完成，成功: 0 条，失败: 5 条
数据存储失败: 提交事务失败
```

**修复后的日志**：
```
数据存储完成，成功: 0 条，重复: 5 条，失败: 0 条
任务执行完成: K8S测试-API采集, 耗时: 2.5s, 数据量: 5
```

### 3. 查看执行记录

```bash
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT id, status, records_collected, LEFT(error_message, 50) as error 
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 5;
"
```

**预期结果**：
```
 id | status  | records_collected | error 
----+---------+-------------------+-------
  5 | success |                 5 | 
  4 | success |                 5 | 
  3 | success |                 5 | 
  2 | success |                 5 | 
  1 | success |                 5 | 
```

## 📊 修复效果对比

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| 第一次执行 | ✅ 成功 | ✅ 成功 |
| 后续执行 | ❌ 失败 | ✅ 成功 |
| 错误信息 | duplicate key | 无错误 |
| 数据保存 | 第一次后不再保存 | 新数据正常保存 |
| 执行状态 | failed | success |

## 🎯 适用场景

这个修复适用于：

1. **幂等性要求**：多次执行相同任务不会产生重复数据
2. **增量采集**：只保存新数据，忽略已存在的数据
3. **定时任务**：周期性执行，避免重复插入

## 💡 其他解决方案

### 方案 1: 使用 UPSERT（ON CONFLICT DO UPDATE）

如果需要更新已存在的数据：

```sql
INSERT INTO test_posts (id, title, body, user_id) 
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE 
SET title = EXCLUDED.title,
    body = EXCLUDED.body,
    user_id = EXCLUDED.user_id,
    created_at = NOW();
```

### 方案 2: 先删除后插入

```sql
DELETE FROM test_posts WHERE id IN (1, 2, 3, 4, 5);
INSERT INTO test_posts (id, title, body, user_id) VALUES ...;
```

### 方案 3: 使用唯一的时间戳

修改表结构，使用复合主键：

```sql
CREATE TABLE test_posts (
    id INT,
    title VARCHAR(500),
    body TEXT,
    user_id INT,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);
```

## 📚 相关文档

- [PostgreSQL ON CONFLICT 文档](https://www.postgresql.org/docs/current/sql-insert.html)
- [K8S_DEPLOYMENT_GUIDE.md](K8S_DEPLOYMENT_GUIDE.md)
- [debug-k8s.sh](debug-k8s.sh) - 问题排查脚本

## 🎉 总结

通过使用 `ON CONFLICT DO NOTHING`，我们实现了：

1. ✅ 优雅处理主键冲突
2. ✅ 区分重复数据和真正的错误
3. ✅ 正确记录任务执行状态
4. ✅ 提供详细的日志信息

现在 Worker 可以正确处理重复数据，不会将其标记为失败！

---

**修复日期**: 2025-12-04  
**影响范围**: PostgreSQL 存储模块  
**状态**: ✅ 已修复
