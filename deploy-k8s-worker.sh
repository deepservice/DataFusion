#!/bin/bash

set -e

echo "=========================================="
echo "DataFusion Worker K8S 部署脚本"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查 kubectl
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✅ kubectl 已安装${NC}"

# 检查 docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ docker 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✅ docker 已安装${NC}"
echo ""

# 步骤 1: 构建 Docker 镜像
echo "=========================================="
echo "步骤 1: 构建 Docker 镜像"
echo "=========================================="

echo -e "${YELLOW}正在构建 Worker 镜像...${NC}"
docker build -t datafusion-worker:latest .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Docker 镜像构建成功${NC}"
else
    echo -e "${RED}❌ Docker 镜像构建失败${NC}"
    exit 1
fi

echo ""

# 步骤 2: 创建命名空间
echo "=========================================="
echo "步骤 2: 创建 Kubernetes 命名空间"
echo "=========================================="

kubectl apply -f k8s/namespace.yaml
echo -e "${GREEN}✅ 命名空间创建完成${NC}"
echo ""

# 步骤 3: 部署 PostgreSQL
echo "=========================================="
echo "步骤 3: 部署 PostgreSQL"
echo "=========================================="

kubectl apply -f k8s/postgres-init-scripts.yaml
kubectl apply -f k8s/postgresql.yaml

echo -e "${YELLOW}等待 PostgreSQL 启动...${NC}"
kubectl wait --for=condition=ready pod -l app=postgresql -n datafusion --timeout=120s

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ PostgreSQL 部署成功${NC}"
else
    echo -e "${RED}❌ PostgreSQL 部署失败${NC}"
    exit 1
fi

echo ""

# 步骤 4: 部署 Worker
echo "=========================================="
echo "步骤 4: 部署 Worker"
echo "=========================================="

kubectl apply -f k8s/worker-config.yaml
kubectl apply -f k8s/worker.yaml

echo -e "${YELLOW}等待 Worker 启动...${NC}"
kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Worker 部署成功${NC}"
else
    echo -e "${RED}❌ Worker 部署失败${NC}"
    exit 1
fi

echo ""

# 步骤 5: 显示部署状态
echo "=========================================="
echo "步骤 5: 部署状态"
echo "=========================================="

echo ""
echo "📦 Pods 状态:"
kubectl get pods -n datafusion

echo ""
echo "🔧 Services 状态:"
kubectl get svc -n datafusion

echo ""
echo "=========================================="
echo -e "${GREEN}✅ 部署完成！${NC}"
echo "=========================================="
echo ""

echo "📝 查看 Worker 日志:"
echo "  kubectl logs -f -l app=datafusion-worker -n datafusion"
echo ""

echo "📝 查看 PostgreSQL 日志:"
echo "  kubectl logs -f -l app=postgresql -n datafusion"
echo ""

echo "📝 进入 PostgreSQL 容器:"
echo "  kubectl exec -it -n datafusion \$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U datafusion -d datafusion_control"
echo ""

echo "📝 查看任务执行记录:"
echo "  kubectl exec -it -n datafusion \$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U datafusion -d datafusion_control -c 'SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 5;'"
echo ""

echo "📝 查看采集的数据:"
echo "  kubectl exec -it -n datafusion \$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- psql -U datafusion -d datafusion_data -c 'SELECT * FROM test_posts;'"
echo ""

echo "🗑️  清理部署:"
echo "  kubectl delete namespace datafusion"
echo ""
