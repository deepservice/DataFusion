#!/bin/bash

# DataFusion 统一部署脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 显示帮助信息
show_help() {
    echo "DataFusion 部署脚本"
    echo ""
    echo "用法:"
    echo "  $0 [选项] <组件>"
    echo ""
    echo "组件:"
    echo "  api-server    部署 API Server"
    echo "  worker        部署 Worker"
    echo "  web           部署 Web 前端"
    echo "  all           部署完整系统（API Server + Worker + Web）"
    echo ""
    echo "选项:"
    echo "  -h, --help    显示帮助信息"
    echo "  --clean       部署前清理现有资源"
    echo ""
    echo "示例:"
    echo "  $0 all                # 部署完整系统"
    echo "  $0 api-server         # 只部署 API Server"
    echo "  $0 worker             # 只部署 Worker"
    echo "  $0 web                # 只部署 Web 前端"
    echo "  $0 --clean all        # 清理后部署完整系统"
}

# 检查依赖
check_dependencies() {
    echo -e "${BLUE}检查依赖...${NC}"
    
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl 未安装${NC}"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ docker 未安装${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ 依赖检查通过${NC}"
    echo ""
}

# 检测 Kubernetes 环境类型
detect_k8s_env() {
    if kubectl config current-context | grep -q "kind"; then
        echo "kind"
    elif kubectl config current-context | grep -q "minikube"; then
        echo "minikube"
    else
        echo "other"
    fi
}

# 加载镜像到 Kubernetes 集群
load_image_to_cluster() {
    local IMAGE_NAME=$1
    local K8S_ENV=$(detect_k8s_env)
    
    echo -e "${YELLOW}检测到 Kubernetes 环境: ${K8S_ENV}${NC}"
    
    case $K8S_ENV in
        kind)
            echo -e "${YELLOW}加载镜像到 kind 集群...${NC}"
            kind load docker-image "$IMAGE_NAME"
            echo -e "${GREEN}✅ 镜像已加载到 kind 集群${NC}"
            ;;
        minikube)
            echo -e "${YELLOW}加载镜像到 minikube...${NC}"
            minikube image load "$IMAGE_NAME"
            echo -e "${GREEN}✅ 镜像已加载到 minikube${NC}"
            ;;
        *)
            echo -e "${YELLOW}⚠️  非 kind/minikube 环境，跳过镜像加载${NC}"
            echo -e "${YELLOW}   如需使用本地镜像，请手动推送到镜像仓库${NC}"
            ;;
    esac
}

# 清理资源
clean_resources() {
    echo -e "${YELLOW}清理现有资源...${NC}"
    kubectl delete namespace datafusion --ignore-not-found=true
    echo -e "${GREEN}✅ 清理完成${NC}"
    echo ""
}

# 创建命名空间
create_namespace() {
    echo -e "${YELLOW}创建命名空间...${NC}"
    kubectl create namespace datafusion --dry-run=client -o yaml | kubectl apply -f -
    echo -e "${GREEN}✅ 命名空间已就绪${NC}"
    echo ""
}

# 部署 PostgreSQL
deploy_postgresql() {
    echo -e "${YELLOW}部署 PostgreSQL...${NC}"
    kubectl apply -f k8s/postgres-init-scripts.yaml
    kubectl apply -f k8s/postgresql.yaml
    
    echo -e "${YELLOW}等待 PostgreSQL 启动...${NC}"
    kubectl wait --for=condition=ready pod -l app=postgresql -n datafusion --timeout=120s
    echo -e "${GREEN}✅ PostgreSQL 部署成功${NC}"
    echo ""
}

# 部署 API Server
deploy_api_server() {
    echo -e "${YELLOW}构建 API Server 镜像...${NC}"
    docker build -f Dockerfile.api-server -t datafusion/api-server:latest .
    echo -e "${GREEN}✅ API Server 镜像构建完成${NC}"
    
    # 加载镜像到集群
    load_image_to_cluster "datafusion/api-server:latest"
    
    echo -e "${YELLOW}部署 API Server...${NC}"
    kubectl apply -f k8s/api-server-deployment.yaml
    
    echo -e "${YELLOW}等待 API Server 启动...${NC}"
    kubectl wait --for=condition=ready pod -l app=api-server -n datafusion --timeout=120s
    echo -e "${GREEN}✅ API Server 部署成功${NC}"
    echo ""
}

# 部署 Worker
deploy_worker() {
    echo -e "${YELLOW}构建 Worker 镜像...${NC}"
    docker build -t datafusion-worker:latest .
    echo -e "${GREEN}✅ Worker 镜像构建完成${NC}"
    
    # 加载镜像到集群
    load_image_to_cluster "datafusion-worker:latest"
    
    echo -e "${YELLOW}部署 Worker...${NC}"
    kubectl apply -f k8s/worker-config.yaml
    kubectl apply -f k8s/worker.yaml
    
    echo -e "${YELLOW}等待 Worker 启动...${NC}"
    kubectl wait --for=condition=ready pod -l app=datafusion-worker -n datafusion --timeout=120s
    echo -e "${GREEN}✅ Worker 部署成功${NC}"
    echo ""
}

# 部署 Web 前端
deploy_web() {
    echo -e "${YELLOW}构建 Web 前端镜像...${NC}"
    docker build -t datafusion/web:latest ./web
    echo -e "${GREEN}✅ Web 前端镜像构建完成${NC}"
    
    # 加载镜像到集群
    load_image_to_cluster "datafusion/web:latest"
    
    echo -e "${YELLOW}部署 Web 前端...${NC}"
    kubectl apply -f k8s/web-deployment.yaml
    
    echo -e "${YELLOW}等待 Web 前端启动...${NC}"
    kubectl wait --for=condition=ready pod -l app=datafusion-web -n datafusion --timeout=120s
    echo -e "${GREEN}✅ Web 前端部署成功${NC}"
    echo ""
}

# 显示部署状态
show_status() {
    echo "=========================================="
    echo "部署状态"
    echo "=========================================="
    echo ""
    
    echo "📦 Pods:"
    kubectl get pods -n datafusion
    echo ""
    
    echo "🔧 Services:"
    kubectl get svc -n datafusion
    echo ""
    
    if kubectl get ingress -n datafusion &>/dev/null; then
        echo "🌐 Ingress:"
        kubectl get ingress -n datafusion
        echo ""
    fi
}

# 显示访问信息
show_access_info() {
    echo "=========================================="
    echo "访问信息"
    echo "=========================================="
    echo ""
    
    if [[ "$DEPLOY_WEB" == "true" ]]; then
        echo "🌐 Web 管理界面:"
        echo "  内部访问: http://datafusion-web-service.datafusion.svc.cluster.local"
        echo "  端口转发: kubectl port-forward -n datafusion svc/datafusion-web-service 3000:80"
        echo "  然后访问: http://localhost:3000"
        echo "  默认账户: admin / admin123"
        echo ""
    fi
    
    if [[ "$DEPLOY_API_SERVER" == "true" ]]; then
        echo "🔗 API Server:"
        echo "  内部访问: http://api-server-service.datafusion.svc.cluster.local:8080"
        echo "  端口转发: kubectl port-forward -n datafusion svc/api-server-service 8081:8080"
        echo "  然后访问: http://localhost:8081"
        echo ""
    fi
    
    echo "📝 常用命令:"
    if [[ "$DEPLOY_WEB" == "true" ]]; then
        echo "  查看 Web 日志: kubectl logs -f -l app=datafusion-web -n datafusion"
    fi
    echo "  查看 Worker 日志: kubectl logs -f -l app=datafusion-worker -n datafusion"
    echo "  查看 API Server 日志: kubectl logs -f -l app=api-server -n datafusion"
    echo "  查看 PostgreSQL 日志: kubectl logs -f -l app=postgresql -n datafusion"
    echo ""
    
    echo "🗑️  清理部署:"
    echo "  kubectl delete namespace datafusion"
    echo ""
}

# 测试健康检查
test_health() {
    if [[ "$DEPLOY_API_SERVER" == "true" ]]; then
        echo -e "${YELLOW}测试 API Server 健康检查...${NC}"
        kubectl port-forward -n datafusion svc/api-server-service 8081:8080 &
        PF_PID=$!
        sleep 3
        
        if curl -s http://localhost:8081/healthz | grep -q "ok"; then
            echo -e "${GREEN}✅ API Server 健康检查通过${NC}"
        else
            echo -e "${RED}❌ API Server 健康检查失败${NC}"
        fi
        
        kill $PF_PID 2>/dev/null || true
        echo ""
    fi
}

# 主函数
main() {
    local CLEAN=false
    local DEPLOY_API_SERVER=false
    local DEPLOY_WORKER=false
    local DEPLOY_WEB=false
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --clean)
                CLEAN=true
                shift
                ;;
            api-server)
                DEPLOY_API_SERVER=true
                shift
                ;;
            worker)
                DEPLOY_WORKER=true
                shift
                ;;
            web)
                DEPLOY_WEB=true
                shift
                ;;
            all)
                DEPLOY_API_SERVER=true
                DEPLOY_WORKER=true
                DEPLOY_WEB=true
                shift
                ;;
            *)
                echo -e "${RED}未知参数: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 检查是否指定了组件
    if [[ "$DEPLOY_API_SERVER" == "false" && "$DEPLOY_WORKER" == "false" && "$DEPLOY_WEB" == "false" ]]; then
        echo -e "${RED}请指定要部署的组件${NC}"
        show_help
        exit 1
    fi
    
    echo "=========================================="
    echo "DataFusion Kubernetes 部署"
    echo "=========================================="
    echo ""
    
    # 检查依赖
    check_dependencies
    
    # 清理资源（如果指定）
    if [[ "$CLEAN" == "true" ]]; then
        clean_resources
    fi
    
    # 创建命名空间
    create_namespace
    
    # 部署 PostgreSQL（Worker 需要）
    if [[ "$DEPLOY_WORKER" == "true" ]]; then
        deploy_postgresql
    fi
    
    # 部署 API Server
    if [[ "$DEPLOY_API_SERVER" == "true" ]]; then
        deploy_api_server
    fi
    
    # 部署 Worker
    if [[ "$DEPLOY_WORKER" == "true" ]]; then
        deploy_worker
    fi
    
    # 部署 Web 前端
    if [[ "$DEPLOY_WEB" == "true" ]]; then
        deploy_web
    fi
    
    # 显示状态
    show_status
    
    # 测试健康检查
    test_health
    
    # 显示访问信息
    show_access_info
    
    echo "=========================================="
    echo -e "${GREEN}✅ 部署完成！${NC}"
    echo "=========================================="
}

# 执行主函数
main "$@"