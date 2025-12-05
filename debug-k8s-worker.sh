#!/bin/bash

set -e

echo "=========================================="
echo "DataFusion Worker 问题排查脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取 Pod 名称
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')
WORKER_POD=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}')

echo -e "${GREEN}PostgreSQL Pod: $PG_POD${NC}"
echo -e "${GREEN}Worker Pod: $WORKER_POD${NC}"
echo ""

# 1. 查看任务执行记录（包含错误信息）
echo "=========================================="
echo "1. 任务执行记录（最近 10 条）"
echo "=========================================="
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT 
    id, 
    task_id, 
    status, 
    records_collected,
    start_time,
    LEFT(error_message, 100) as error_msg
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 10;
"
echo ""

# 2. 查看失败的任务详细错误
echo "=========================================="
echo "2. 失败任务的详细错误信息"
echo "=========================================="
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT 
    id, 
    task_id, 
    status, 
    start_time,
    error_message
FROM task_executions 
WHERE status = 'failed'
ORDER BY start_time DESC 
LIMIT 5;
"
echo ""

# 3. 查看 test_posts 表的数据
echo "=========================================="
echo "3. test_posts 表中的数据"
echo "=========================================="
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT COUNT(*) as total_records FROM test_posts;
"
echo ""

kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT id, LEFT(title, 50) as title, created_at 
FROM test_posts 
ORDER BY created_at DESC 
LIMIT 5;
"
echo ""

# 4. 查看 Worker 最近的日志（最近 50 行）
echo "=========================================="
echo "4. Worker 最近的日志（最近 50 行）"
echo "=========================================="
kubectl logs --tail=50 -n datafusion $WORKER_POD
echo ""

# 5. 检查主键冲突
echo "=========================================="
echo "5. 检查是否有主键冲突"
echo "=========================================="
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT id, COUNT(*) as count 
FROM test_posts 
GROUP BY id 
HAVING COUNT(*) > 1;
"
echo ""

# 6. 查看表结构
echo "=========================================="
echo "6. test_posts 表结构"
echo "=========================================="
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT
    column_name,
    data_type,
    character_maximum_length,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'test_posts'
ORDER BY ordinal_position;
"
echo ""

# 6.1 查看表的约束信息
echo "表约束（主键、外键等）："
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT
    con.conname AS constraint_name,
    con.contype AS constraint_type,
    CASE con.contype
        WHEN 'p' THEN 'PRIMARY KEY'
        WHEN 'f' THEN 'FOREIGN KEY'
        WHEN 'u' THEN 'UNIQUE'
        WHEN 'c' THEN 'CHECK'
        ELSE con.contype::text
    END AS type_description
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
WHERE rel.relname = 'test_posts';
"
echo ""

echo "=========================================="
echo "排查完成"
echo "=========================================="
echo ""
echo "💡 提示："
echo "  - 如果看到 'duplicate key value' 错误，说明是主键冲突"
echo "  - 如果看到 'connection refused' 错误，说明是数据库连接问题"
echo "  - 如果看到其他错误，请查看 Worker 日志详情"
echo ""
