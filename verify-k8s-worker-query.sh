#!/bin/bash

set -e

echo "=========================================="
echo "DataFusion Worker K8S 验证脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取 PostgreSQL Pod 名称
PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')
WORKER_POD=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}')

if [ -z "$PG_POD" ]; then
    echo -e "${RED}❌ PostgreSQL Pod 未找到${NC}"
    exit 1
fi

if [ -z "$WORKER_POD" ]; then
    echo -e "${RED}❌ Worker Pod 未找到${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 找到 PostgreSQL Pod: $PG_POD${NC}"
echo -e "${GREEN}✅ 找到 Worker Pod: $WORKER_POD${NC}"
echo ""

# 验证 1: 检查 Pods 状态
echo "=========================================="
echo "验证 1: 检查 Pods 状态"
echo "=========================================="
kubectl get pods -n datafusion
echo ""

# 验证 2: 检查数据库连接
echo "=========================================="
echo "验证 2: 检查数据库连接"
echo "=========================================="
echo -e "${YELLOW}测试数据库连接...${NC}"
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT 1;" > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库连接正常${NC}"
else
    echo -e "${RED}❌ 数据库连接失败${NC}"
    exit 1
fi
echo ""

# 验证 3: 检查任务配置
echo "=========================================="
echo "验证 3: 检查任务配置"
echo "=========================================="
echo -e "${BLUE}任务列表:${NC}"
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "SELECT id, name, type, status, next_run_time FROM collection_tasks;"
echo ""

# 验证 4: 查看 Worker 日志
echo "=========================================="
echo "验证 4: 查看 Worker 日志（最近 20 行）"
echo "=========================================="
kubectl logs --tail=20 -n datafusion $WORKER_POD
echo ""

# 验证 5: 等待任务执行
echo "=========================================="
echo "验证 5: 等待任务执行"
echo "=========================================="
echo -e "${YELLOW}等待 1 分钟，让 Worker 执行任务...${NC}"

for i in {1..6}; do
    echo -n "."
    sleep 10
done
echo ""
echo -e "${GREEN}✅ 等待完成${NC}"
echo ""

# 验证 6: 检查任务执行记录
echo "=========================================="
echo "验证 6: 检查任务执行记录"
echo "=========================================="
echo -e "${BLUE}任务执行历史:${NC}"
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "
SELECT 
    id, 
    task_id, 
    worker_pod, 
    status, 
    records_collected, 
    start_time,
    EXTRACT(EPOCH FROM (end_time - start_time)) as duration_seconds
FROM task_executions 
ORDER BY start_time DESC 
LIMIT 5;
"
echo ""

# 验证 7: 检查采集的数据
echo "=========================================="
echo "验证 7: 检查采集的数据"
echo "=========================================="
echo -e "${BLUE}采集的数据（test_posts 表）:${NC}"
kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "
SELECT 
    id, 
    LEFT(title, 50) as title, 
    user_id,
    created_at 
FROM test_posts 
ORDER BY created_at DESC 
LIMIT 10;
"
echo ""

# 验证 8: 统计信息
echo "=========================================="
echo "验证 8: 统计信息"
echo "=========================================="

TASK_COUNT=$(kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -t -c "SELECT COUNT(*) FROM collection_tasks;")
EXEC_COUNT=$(kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -t -c "SELECT COUNT(*) FROM task_executions;")
DATA_COUNT=$(kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -t -c "SELECT COUNT(*) FROM test_posts;")
SUCCESS_COUNT=$(kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -t -c "SELECT COUNT(*) FROM task_executions WHERE status='success';")

echo -e "${BLUE}📊 统计数据:${NC}"
echo "  任务总数: $TASK_COUNT"
echo "  执行次数: $EXEC_COUNT"
echo "  成功次数: $SUCCESS_COUNT"
echo "  采集数据: $DATA_COUNT 条"
echo ""

# 验证结果
echo "=========================================="
echo "验证结果"
echo "=========================================="

if [ "$DATA_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ 验证成功！${NC}"
    echo -e "${GREEN}   - Worker 正常运行${NC}"
    echo -e "${GREEN}   - 任务执行成功${NC}"
    echo -e "${GREEN}   - 数据已保存到 PostgreSQL${NC}"
    echo ""
    echo -e "${BLUE}📝 采集到 $DATA_COUNT 条数据${NC}"
else
    echo -e "${YELLOW}⚠️  暂未采集到数据${NC}"
    echo -e "${YELLOW}   请检查:${NC}"
    echo "   1. Worker 日志是否有错误"
    echo "   2. 任务是否已执行"
    echo "   3. 网络连接是否正常"
fi

echo ""
echo "=========================================="
echo "常用命令"
echo "=========================================="
echo ""
echo "📝 实时查看 Worker 日志:"
echo "  kubectl logs -f -n datafusion $WORKER_POD"
echo ""
echo "📝 手动触发任务执行:"
echo "  kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c \"UPDATE collection_tasks SET next_run_time = NOW() WHERE id = 1;\""
echo ""
echo "📝 查看完整数据:"
echo "  kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c 'SELECT * FROM test_posts;'"
echo ""
echo "📝 进入 PostgreSQL 交互式终端:"
echo "  kubectl exec -it -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control"
echo ""
