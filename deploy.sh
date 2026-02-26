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
    echo "  port-forward  启动服务端口转发"
    echo ""
    echo "选项:"
    echo "  -h, --help    显示帮助信息"
    echo "  --clean       部署前清理现有资源（包括删除本地镜像）"
    echo ""
    echo "示例:"
    echo "  $0 all                # 部署完整系统"
    echo "  $0 api-server         # 只部署 API Server"
    echo "  $0 worker             # 只部署 Worker"
    echo "  $0 web                # 只部署 Web 前端"
    echo "  $0 --clean all        # 清理后部署完整系统"
    echo "  $0 port-forward       # 启动端口转发（环境就绪时）"
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
    local K8S_ENV
    K8S_ENV=$(detect_k8s_env)
    
    echo -e "${YELLOW}检测到 Kubernetes 环境: ${K8S_ENV}${NC}"
    
    case $K8S_ENV in
        kind)
            echo -e "${YELLOW}加载镜像到 kind 集群...${NC}"
            kind load docker-image "$IMAGE_NAME" --name dev
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

    # 删除 Kubernetes 命名空间
    if kubectl get namespace datafusion &>/dev/null; then
        echo -e "${YELLOW}删除命名空间 datafusion...${NC}"
        kubectl delete namespace datafusion --ignore-not-found=true

        # 等待命名空间完全删除
        echo -e "${YELLOW}等待命名空间删除完成...${NC}"
        local TIMEOUT=60
        local ELAPSED=0
        while kubectl get namespace datafusion &>/dev/null && [ $ELAPSED -lt $TIMEOUT ]; do
            sleep 2
            ELAPSED=$((ELAPSED + 2))
            echo -n "."
        done
        echo ""

        if kubectl get namespace datafusion &>/dev/null; then
            echo -e "${YELLOW}⚠️  命名空间删除超时，继续执行...${NC}"
        else
            echo -e "${GREEN}✅ 命名空间已删除${NC}"
        fi
    else
        echo -e "${GREEN}✅ 命名空间不存在，跳过删除${NC}"
    fi

    # 删除本地 Docker 镜像
    echo -e "${YELLOW}删除本地 Docker 镜像...${NC}"
    docker rmi datafusion/api-server:latest 2>/dev/null || true
    docker rmi datafusion-worker:latest 2>/dev/null || true
    docker rmi datafusion/web:latest 2>/dev/null || true

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

    # 检查并加载 PostgreSQL 镜像
    local POSTGRES_IMAGE="postgres:14-alpine"
    echo -e "${YELLOW}检查 PostgreSQL 镜像...${NC}"

    if ! docker image inspect "$POSTGRES_IMAGE" &>/dev/null; then
        echo -e "${YELLOW}本地未找到 PostgreSQL 镜像，正在拉取...${NC}"
        docker pull "$POSTGRES_IMAGE"
        echo -e "${GREEN}✅ PostgreSQL 镜像拉取完成${NC}"
    else
        echo -e "${GREEN}✅ PostgreSQL 镜像已存在${NC}"
    fi

    # 加载镜像到集群
    load_image_to_cluster "$POSTGRES_IMAGE"

    kubectl apply -f k8s/postgres-init-scripts.yaml
    kubectl apply -f k8s/postgresql.yaml

    echo -e "${YELLOW}等待 PostgreSQL 启动...${NC}"
    kubectl wait --for=condition=ready pod -l app=postgresql -n datafusion --timeout=120s
    echo -e "${GREEN}✅ PostgreSQL 部署成功${NC}"

    # 初始化数据库
    init_database
    echo ""
}

# 初始化数据库
init_database() {
    echo -e "${YELLOW}检查并初始化数据库...${NC}"

    local POSTGRES_POD
    POSTGRES_POD=$(kubectl get pod -n datafusion -l app=postgresql -o jsonpath='{.items[0].metadata.name}')

    if [ -z "$POSTGRES_POD" ]; then
        echo -e "${RED}❌ 找不到 PostgreSQL Pod${NC}"
        return 1
    fi

    # 1. 检查并创建 datafusion_data 数据库
    echo -e "${YELLOW}检查数据面数据库 (datafusion_data)...${NC}"
    local DATA_DB_EXISTS
    DATA_DB_EXISTS=$(kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d postgres -tAc "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'datafusion_data');" 2>/dev/null || echo "f")

    if [ "$DATA_DB_EXISTS" = "f" ]; then
        echo -e "${YELLOW}创建 datafusion_data 数据库...${NC}"
        if kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d postgres -c "CREATE DATABASE datafusion_data;" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ datafusion_data 数据库创建成功${NC}"
        else
            echo -e "${RED}❌ datafusion_data 数据库创建失败${NC}"
            return 1
        fi
    else
        echo -e "${GREEN}✅ datafusion_data 数据库已存在${NC}"
    fi

    # 2. 检查并初始化控制面数据库 (datafusion_control)
    echo -e "${YELLOW}检查控制面数据库表 (datafusion_control)...${NC}"
    local TABLE_EXISTS
    TABLE_EXISTS=$(kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -tAc "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'collection_tasks');" 2>/dev/null || echo "false")

    if [ "$TABLE_EXISTS" = "t" ]; then
        echo -e "${GREEN}✅ 控制面数据库表已存在，跳过初始化${NC}"
        return 0
    fi

    echo -e "${YELLOW}控制面数据库表不存在，执行初始化脚本...${NC}"

    # 执行初始化脚本
    if kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -f /docker-entrypoint-initdb.d/01-init-tables.sql > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 表结构创建成功${NC}"
    else
        echo -e "${RED}❌ 表结构创建失败${NC}"
        return 1
    fi

    # 插入测试数据
    if kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -f /docker-entrypoint-initdb.d/02-insert-test-data.sql > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 测试数据插入成功${NC}"
    else
        echo -e "${YELLOW}⚠️  测试数据插入失败（可能已存在）${NC}"
    fi

    # 验证表是否创建成功
    local TABLES_COUNT
    TABLES_COUNT=$(kubectl exec -n datafusion "$POSTGRES_POD" -- psql -U datafusion -d datafusion_control -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null || echo "0")

    if [ "$TABLES_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✅ 数据库初始化完成（共 $TABLES_COUNT 个表）${NC}"
    else
        echo -e "${RED}❌ 数据库初始化失败${NC}"
        return 1
    fi
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
        echo "  默认账户: admin / Admin@123"
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

# 启动端口转发
start_port_forward() {
    echo "========================================="
    echo -e "${BLUE}启动服务端口转发${NC}"
    echo "========================================="
    echo ""
    
    # 检查命名空间是否存在
    if ! kubectl get namespace datafusion &>/dev/null; then
        echo -e "${RED}❌ 命名空间 'datafusion' 不存在，请先部署服务${NC}"
        return 1
    fi
    
    # PostgreSQL 端口转发
    if kubectl get svc postgres-service -n datafusion &>/dev/null; then
        echo -e "${YELLOW}启动 PostgreSQL 端口转发 (localhost:5432 -> postgres-service:5432)...${NC}"
        kubectl port-forward -n datafusion svc/postgres-service 5432:5432 &
        PF_PID_PG=$!
        echo -e "${GREEN}✅ PostgreSQL 已发布到 localhost:5432${NC}"
        echo ""
    fi
    
    # API Server 端口转发
    if kubectl get svc api-server-service -n datafusion &>/dev/null; then
        echo -e "${YELLOW}启动 API Server 端口转发 (localhost:8081 -> api-server-service:8080)...${NC}"
        kubectl port-forward -n datafusion svc/api-server-service 8081:8080 &
        PF_PID_API=$!
        echo -e "${GREEN}✅ API Server 已发布到 localhost:8081${NC}"
        echo ""
    fi
    
    # Worker 端口转发（查找 worker 服务）
    WORKER_SERVICE=$(kubectl get svc -n datafusion 2>/dev/null | grep -i worker | awk '{print $1}' | head -1)
    if [[ -n "$WORKER_SERVICE" ]]; then
        # 获取 worker 服务的第一个端口
        WORKER_PORT=$(kubectl get svc "$WORKER_SERVICE" -n datafusion -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
        if [[ -n "$WORKER_PORT" ]]; then
            echo -e "${YELLOW}启动 Worker 端口转发 (localhost:9090 -> $WORKER_SERVICE:$WORKER_PORT)...${NC}"
            kubectl port-forward -n datafusion svc/"$WORKER_SERVICE" 9090:"$WORKER_PORT" &
            PF_PID_WORKER=$!
            echo -e "${GREEN}✅ Worker 已发布到 localhost:9090${NC}"
            echo ""
        fi
    fi
    
    # Web 前端端口转发
    if kubectl get svc datafusion-web-service -n datafusion &>/dev/null; then
        echo -e "${YELLOW}启动 Web 前端端口转发 (localhost:3000 -> datafusion-web-service:80)...${NC}"
        kubectl port-forward -n datafusion svc/datafusion-web-service 3000:80 &
        PF_PID_WEB=$!
        echo -e "${GREEN}✅ Web 前端已发布到 localhost:3000${NC}"
        echo ""
    fi
    
    echo "========================================="
    echo -e "${GREEN}✅ 端口转发已启动${NC}"
    echo "========================================="
    echo ""
    echo -e "${BLUE}访问地址：${NC}"
    if [[ -n "${PF_PID_WEB}" ]]; then
        echo "  🌐 Web 管理界面: http://localhost:3000"
        echo "     默认账户: admin / Admin@123"
    fi
    if [[ -n "${PF_PID_API}" ]]; then
        echo "  🔗 API Server: http://localhost:8081"
    fi
    if [[ -n "${PF_PID_WORKER}" ]]; then
        echo "  ⚙️  Worker 服务: localhost:9090"
    fi
    if [[ -n "${PF_PID_PG}" ]]; then
        echo "  🗄️  PostgreSQL: localhost:5432"
    fi
    echo ""
    echo -e "${YELLOW}提示：${NC}端口转发将在后台运行。要停止转发，请按 Ctrl+C 或执行："
    echo "  kill \$PF_PID_PG \$PF_PID_API \$PF_PID_WORKER \$PF_PID_WEB 2>/dev/null"
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
    local START_PORT_FORWARD=false
    local ONLY_PORT_FORWARD=false
    
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
                START_PORT_FORWARD=true
                shift
                ;;
            port-forward)
                ONLY_PORT_FORWARD=true
                shift
                ;;
            *)
                echo -e "${RED}未知参数: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 处理仅启动端口转发的情况
    if [[ "$ONLY_PORT_FORWARD" == "true" ]]; then
        start_port_forward
        return
    fi
    
    # 处理仅清理资源的情况
    if [[ "$CLEAN" == "true" && "$DEPLOY_API_SERVER" == "false" && "$DEPLOY_WORKER" == "false" && "$DEPLOY_WEB" == "false" ]]; then
        echo "=========================================="
        echo "DataFusion Kubernetes 清理"
        echo "=========================================="
        echo ""
        
        check_dependencies
        clean_resources
        
        echo "=========================================="
        echo -e "${GREEN}✅ 清理完成！${NC}"
        echo "=========================================="
        return
    fi
    
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
    echo ""
    
    # 自动启动端口转发（all 部署时）
    if [[ "$START_PORT_FORWARD" == "true" ]]; then
        start_port_forward
    fi
}

# 执行主函数
main "$@"