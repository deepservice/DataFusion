#!/bin/bash

# DataFusion Authentication Demo Startup Script
# 启动认证系统演示

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 DataFusion 认证系统演示启动脚本${NC}"
echo "=================================================="

# 检查依赖
check_dependencies() {
    echo -e "\n${YELLOW}检查依赖...${NC}"
    
    # 检查 PostgreSQL
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}❌ PostgreSQL 未安装${NC}"
        echo "请安装 PostgreSQL: https://www.postgresql.org/download/"
        exit 1
    fi
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go 未安装${NC}"
        echo "请安装 Go: https://golang.org/dl/"
        exit 1
    fi
    
    # 检查 jq (用于JSON处理)
    if ! command -v jq &> /dev/null; then
        echo -e "${YELLOW}⚠️  jq 未安装，建议安装以获得更好的测试体验${NC}"
        echo "安装命令: sudo apt-get install jq (Ubuntu) 或 brew install jq (macOS)"
    fi
    
    echo -e "${GREEN}✅ 依赖检查完成${NC}"
}

# 设置数据库
setup_database() {
    echo -e "\n${YELLOW}设置数据库...${NC}"
    
    # 检查数据库是否存在
    DB_EXISTS=$(psql -U postgres -lqt | cut -d \| -f 1 | grep -w datafusion_control | wc -l)
    
    if [ $DB_EXISTS -eq 0 ]; then
        echo "创建数据库 datafusion_control..."
        createdb -U postgres datafusion_control
    else
        echo "数据库 datafusion_control 已存在"
    fi
    
    # 初始化数据库结构
    echo "初始化数据库结构..."
    psql -U postgres -d datafusion_control -f scripts/init_control_db.sql > /dev/null
    
    echo -e "${GREEN}✅ 数据库设置完成${NC}"
}

# 编译项目
build_project() {
    echo -e "\n${YELLOW}编译项目...${NC}"
    
    go mod tidy
    go build -o api-server ./cmd/api-server
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ 项目编译成功${NC}"
    else
        echo -e "${RED}❌ 项目编译失败${NC}"
        exit 1
    fi
}

# 启动 API 服务器
start_api_server() {
    echo -e "\n${YELLOW}启动 API 服务器...${NC}"
    
    # 检查端口是否被占用
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
        echo -e "${RED}❌ 端口 8080 已被占用${NC}"
        echo "请停止占用端口的进程或修改配置文件中的端口"
        exit 1
    fi
    
    echo "启动 DataFusion API Server..."
    echo "配置文件: config/api-server.yaml"
    echo "监听端口: 8080"
    echo ""
    echo -e "${GREEN}服务器启动成功！${NC}"
    echo ""
    echo -e "${BLUE}API 端点:${NC}"
    echo "  健康检查: http://localhost:8080/healthz"
    echo "  登录接口: http://localhost:8080/api/v1/auth/login"
    echo "  API 文档: 查看 docs/ 目录"
    echo ""
    echo -e "${BLUE}默认管理员账户:${NC}"
    echo "  用户名: admin"
    echo "  密码: admin123"
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 停止服务器${NC}"
    echo "=================================================="
    
    # 启动服务器
    ./api-server
}

# 显示测试命令
show_test_commands() {
    echo -e "\n${BLUE}测试命令示例:${NC}"
    echo ""
    echo "1. 健康检查:"
    echo "   curl http://localhost:8080/healthz"
    echo ""
    echo "2. 用户登录:"
    echo '   curl -X POST http://localhost:8080/api/v1/auth/login \'
    echo '        -H "Content-Type: application/json" \'
    echo '        -d '"'"'{"username":"admin","password":"admin123"}'"'"
    echo ""
    echo "3. 运行完整测试:"
    echo "   ./tests/test_auth_system.sh"
    echo ""
}

# 主函数
main() {
    echo -e "${BLUE}开始设置 DataFusion 认证系统演示环境...${NC}"
    
    check_dependencies
    setup_database
    build_project
    
    echo -e "\n${GREEN}🎉 设置完成！${NC}"
    show_test_commands
    
    echo -e "\n${YELLOW}是否立即启动 API 服务器？ (y/n)${NC}"
    read -r response
    
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        start_api_server
    else
        echo -e "\n${BLUE}手动启动命令:${NC}"
        echo "  ./api-server"
        echo ""
        echo -e "${BLUE}测试命令:${NC}"
        echo "  ./tests/test_auth_system.sh"
    fi
}

# 错误处理
trap 'echo -e "\n${RED}❌ 脚本执行被中断${NC}"; exit 1' INT

# 运行主函数
main "$@"