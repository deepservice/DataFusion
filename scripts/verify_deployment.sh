#!/bin/bash

# DataFusion 部署验证脚本
# 用于验证所有组件是否正确部署和运行

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=========================================="
echo -e "${BLUE}DataFusion 部署验证${NC}"
echo "=========================================="
echo ""

# 1. 检查命名空间
echo -e "${YELLOW}1. 检查命名空间...${NC}"
if kubectl get namespace datafusion &>/dev/null; then
    echo -e "${GREEN}   ✅ 命名空间存在${NC}"
else
    echo -e "${RED}   ❌ 命名空间不存在${NC}"
    exit 1
fi
echo ""

# 2. 检查 Pods
echo -e "${YELLOW}2. 检查 Pods 状态...${NC}"
PODS=$(kubectl get pods -n datafusion --no-headers 2>/dev/null || echo "")
if [ -z "$PODS" ]; then
    echo -e "${RED}   ❌ 没有找到任何 Pod${NC}"
    exit 1
fi

echo "$PODS" | while read -r line; do
    POD_NAME=$(echo "$line" | awk '{print $1}')
    READY=$(echo "$line" | awk '{print $2}')
    STATUS=$(echo "$line" | awk '{print $3}')

    if [ "$STATUS" = "Running" ] && [[ "$READY" =~ ^1/1$ ]]; then
        echo -e "${GREEN}   ✅ $POD_NAME: $STATUS ($READY)${NC}"
    else
        echo -e "${RED}   ❌ $POD_NAME: $STATUS ($READY)${NC}"
    fi
done
echo ""

# 3. 检查 Services
echo -e "${YELLOW}3. 检查 Services...${NC}"
SERVICES=$(kubectl get svc -n datafusion --no-headers 2>/dev/null || echo "")
if [ -z "$SERVICES" ]; then
    echo -e "${RED}   ❌ 没有找到任何 Service${NC}"
else
    echo "$SERVICES" | while read -r line; do
        SVC_NAME=$(echo "$line" | awk '{print $1}')
        echo -e "${GREEN}   ✅ $SVC_NAME${NC}"
    done
fi
echo ""

# 4. 检查 PostgreSQL 数据库
echo -e "${YELLOW}4. 检查 PostgreSQL 数据库...${NC}"
POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$POSTGRES_POD" ]; then
    echo -e "${RED}   ❌ PostgreSQL Pod 不存在${NC}"
else
    # 检查数据库连接
    if kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -c "SELECT 1" &>/dev/null; then
        echo -e "${GREEN}   ✅ 数据库连接正常${NC}"

        # 检查表
        TABLES_COUNT=$(kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null || echo "0")

        if [ "$TABLES_COUNT" -gt 0 ]; then
            echo -e "${GREEN}   ✅ 数据库表已创建 (共 $TABLES_COUNT 个表)${NC}"
        else
            echo -e "${RED}   ❌ 数据库表未创建${NC}"
        fi

        # 检查管理员用户
        ADMIN_EXISTS=$(kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -tAc "SELECT EXISTS(SELECT 1 FROM users WHERE username = 'admin');" 2>/dev/null || echo "f")
        if [ "$ADMIN_EXISTS" = "t" ]; then
            echo -e "${GREEN}   ✅ 管理员用户已创建${NC}"
        else
            echo -e "${YELLOW}   ⚠️  管理员用户不存在${NC}"
        fi
    else
        echo -e "${RED}   ❌ 数据库连接失败${NC}"
    fi
fi
echo ""

# 5. 检查 Worker 健康状态
echo -e "${YELLOW}5. 检查 Worker 状态...${NC}"
WORKER_POD=$(kubectl get pod -n datafusion -l app=datafusion-worker -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$WORKER_POD" ]; then
    echo -e "${RED}   ❌ Worker Pod 不存在${NC}"
else
    echo -e "${GREEN}   ✅ Worker Pod 存在${NC}"
fi
echo ""

# 6. 检查 API Server 健康状态
echo -e "${YELLOW}6. 检查 API Server 状态...${NC}"
API_POD=$(kubectl get pod -n datafusion -l app=api-server -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$API_POD" ]; then
    echo -e "${YELLOW}   ⚠️  API Server Pod 不存在${NC}"
else
    echo -e "${GREEN}   ✅ API Server Pod 存在${NC}"
fi
echo ""

# 总结
echo "=========================================="
echo -e "${GREEN}验证完成！${NC}"
echo "=========================================="
echo ""

echo -e "${BLUE}快速访问命令：${NC}"
echo "  查看所有 Pods:      kubectl get pods -n datafusion"
echo "  查看 Worker 日志:   kubectl logs -f -l app=datafusion-worker -n datafusion"
echo "  查看 API 日志:      kubectl logs -f -l app=api-server -n datafusion"
echo "  启动端口转发:       ./deploy.sh port-forward"
echo ""
