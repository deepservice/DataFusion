#!/bin/bash

set -e

echo "=========================================="
echo "DataFusion Worker 强制更新部署"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 步骤 1: 下载依赖
echo "=========================================="
echo "步骤 1: 下载依赖"
echo "=========================================="

echo -e "${YELLOW}正在下载 Go 模块依赖...${NC}"
go mod tidy

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 依赖下载成功${NC}"
else
    echo -e "${RED}❌ 依赖下载失败${NC}"
    exit 1
fi

echo ""

# 步骤 2: 编译
echo "=========================================="
echo "步骤 2: 编译 Worker"
echo "=========================================="

echo -e "${YELLOW}正在编译 Worker...${NC}"
go build -o worker cmd/worker/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 编译成功${NC}"
    ls -lh worker
else
    echo -e "${RED}❌ 编译失败${NC}"
    exit 1
fi

echo ""

# 步骤 3: 删除旧镜像
echo "=========================================="
echo "步骤 3: 删除旧镜像"
echo "=========================================="

echo -e "${YELLOW}删除旧的 Docker 镜像...${NC}"
docker rmi datafusion-worker:latest 2>/dev/null || echo "旧镜像不存在，跳过"

echo ""

# 步骤 4: 构建新镜像
echo "=========================================="
echo "步骤 4: 构建新镜像"
echo "=========================================="

VERSION_TAG="v2.0-$(date +%Y%m%d%H%M%S)"
echo -e "${BLUE}版本标签: $VERSION_TAG${NC}"

echo -e "${YELLOW}正在构建 Docker 镜像...${NC}"
docker build --no-cache -t datafusion-worker:latest -t datafusion-worker:$VERSION_TAG .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Docker 镜像构建成功${NC}"
    docker images | grep datafusion-worker | head -3
else
    echo -e "${RED}❌ Docker 镜像构建失败${NC}"
    exit 1
fi

echo ""

# 步骤 5: 更新 K8S 配置
echo "=========================================="
echo "步骤 5: 更新 K8S 配置"
echo "=========================================="

echo -e "${YELLOW}应用最新的 K8S 配置...${NC}"
kubectl apply -f k8s/worker.yaml

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ K8S 配置更新成功${NC}"
else
    echo -e "${RED}❌ K8S 配置更新失败${NC}"
    exit 1
fi

echo ""

# 步骤 6: 强制重启 Pod
echo "=========================================="
echo "步骤 6: 强制重启 Pod"
echo "=========================================="

echo -e "${YELLOW}删除旧的 Worker Pod（强制重新拉取镜像）...${NC}"
kubectl delete pod -l app=datafusion-worker -n datafusion

echo -e "${YELLOW}等待新的 Worker Pod 启动...${NC}"
sleep 5
kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Worker Pod 重启成功${NC}"
else
    echo -e "${RED}❌ Worker Pod 重启失败${NC}"
    kubectl get pods -n datafusion -l app=datafusion-worker
    exit 1
fi

echo ""

# 步骤 7: 验证健康检查
echo "=========================================="
echo "步骤 7: 验证健康检查"
echo "=========================================="

WORKER_POD=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}')
echo -e "${BLUE}Worker Pod: $WORKER_POD${NC}"

# 等待服务启动
echo -e "${YELLOW}等待服务启动（15秒）...${NC}"
sleep 15

echo ""
echo -e "${YELLOW}检查 /healthz 端点...${NC}"
kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:8080/healthz 2>&1

echo ""
echo -e "${YELLOW}检查 /readyz 端点...${NC}"
kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:8080/readyz 2>&1

echo ""
echo -e "${YELLOW}检查 /metrics 端点（前20行）...${NC}"
kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:9090/metrics 2>&1 | head -20

echo ""

# 步骤 8: 查看日志
echo "=========================================="
echo "步骤 8: 查看 Worker 日志"
echo "=========================================="

kubectl logs --tail=50 -n datafusion $WORKER_POD

echo ""
echo "=========================================="
echo -e "${GREEN}✅ 强制更新完成！${NC}"
echo "=========================================="
echo ""

echo "📊 部署信息："
echo "  版本: $VERSION_TAG"
echo "  Pod: $WORKER_POD"
echo "  命名空间: datafusion"
echo "  镜像拉取策略: Always（强制重新拉取）"
echo ""

echo "🔍 验证命令："
echo "  查看 Pod 状态: kubectl get pod -n datafusion $WORKER_POD"
echo "  查看 Pod 详情: kubectl describe pod -n datafusion $WORKER_POD"
echo "  查看实时日志: kubectl logs -f -n datafusion $WORKER_POD"
echo ""

echo "🔍 监控端点："
echo "  健康检查: kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:8080/healthz"
echo "  就绪检查: kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:8080/readyz"
echo "  Prometheus: kubectl exec -n datafusion $WORKER_POD -- wget -q -O- http://localhost:9090/metrics"
echo ""

echo "🎉 DataFusion Worker v2.0 强制更新成功！"
echo ""
