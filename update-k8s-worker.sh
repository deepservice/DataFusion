#!/bin/bash

set -e

echo "=========================================="
echo "DataFusion Worker 更新部署脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 步骤 1: 重新构建 Docker 镜像
echo "=========================================="
echo "步骤 1: 重新构建 Docker 镜像"
echo "=========================================="

echo -e "${YELLOW}正在构建新的 Worker 镜像...${NC}"
docker build -t datafusion-worker:latest .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Docker 镜像构建成功${NC}"
else
    echo -e "${RED}❌ Docker 镜像构建失败${NC}"
    exit 1
fi

echo ""

# 步骤 2: 重启 Worker Pod
echo "=========================================="
echo "步骤 2: 重启 Worker Pod"
echo "=========================================="

echo -e "${YELLOW}删除旧的 Worker Pod...${NC}"
kubectl delete pod -l app=datafusion-worker -n datafusion

echo -e "${YELLOW}等待新的 Worker Pod 启动...${NC}"
kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Worker Pod 重启成功${NC}"
else
    echo -e "${RED}❌ Worker Pod 重启失败${NC}"
    exit 1
fi

echo ""

# 步骤 3: 清理旧的执行记录（可选）
echo "=========================================="
echo "步骤 3: 清理测试数据（可选）"
echo "=========================================="

PG_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

read -p "是否清理旧的测试数据？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}清理测试数据...${NC}"
    
    # 清理 test_posts 表
    kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_data -c "TRUNCATE TABLE test_posts;"
    
    # 清理执行记录
    kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "DELETE FROM task_executions;"
    
    # 重置任务的下次执行时间
    kubectl exec -n datafusion $PG_POD -- psql -U datafusion -d datafusion_control -c "UPDATE collection_tasks SET next_run_time = NOW();"
    
    echo -e "${GREEN}✅ 测试数据已清理${NC}"
else
    echo -e "${YELLOW}跳过清理${NC}"
fi

echo ""

# 步骤 4: 查看新的 Worker 日志
echo "=========================================="
echo "步骤 4: 查看新的 Worker 日志"
echo "=========================================="

WORKER_POD=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}')
echo -e "${YELLOW}Worker Pod: $WORKER_POD${NC}"
echo ""

kubectl logs --tail=30 -n datafusion $WORKER_POD

echo ""
echo "=========================================="
echo -e "${GREEN}✅ 更新完成！${NC}"
echo "=========================================="
echo ""

echo "📝 后续操作："
echo "  1. 等待 2 分钟让任务执行"
echo "  2. 运行验证脚本: ./verify-k8s.sh"
echo "  3. 查看实时日志: kubectl logs -f -n datafusion $WORKER_POD"
echo ""
