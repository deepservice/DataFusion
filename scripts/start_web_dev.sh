#!/bin/bash

# DataFusion Web 开发环境启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 DataFusion Web 开发环境启动脚本${NC}"
echo "=================================================="

# 检查 Node.js
check_nodejs() {
    echo -e "\n${YELLOW}检查 Node.js 环境...${NC}"
    
    if ! command -v node &> /dev/null; then
        echo -e "${RED}❌ Node.js 未安装${NC}"
        echo "请安装 Node.js: https://nodejs.org/"
        exit 1
    fi
    
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}✅ Node.js 版本: $NODE_VERSION${NC}"
    
    if ! command -v npm &> /dev/null; then
        echo -e "${RED}❌ npm 未安装${NC}"
        exit 1
    fi
    
    NPM_VERSION=$(npm --version)
    echo -e "${GREEN}✅ npm 版本: $NPM_VERSION${NC}"
}

# 检查 API 服务器
check_api_server() {
    echo -e "\n${YELLOW}检查 API 服务器状态...${NC}"
    
    if curl -s http://localhost:8080/healthz > /dev/null 2>&1; then
        echo -e "${GREEN}✅ API 服务器运行正常${NC}"
    else
        echo -e "${YELLOW}⚠️  API 服务器未运行${NC}"
        echo "请先启动 API 服务器:"
        echo "  cd $(dirname $0)/.."
        echo "  ./api-server"
        echo ""
        echo -e "${YELLOW}是否继续启动 Web 开发服务器？ (y/n)${NC}"
        read -r response
        if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            exit 1
        fi
    fi
}

# 安装依赖
install_dependencies() {
    echo -e "\n${YELLOW}安装前端依赖...${NC}"
    
    cd web
    
    if [ ! -d "node_modules" ]; then
        echo "首次安装，这可能需要几分钟..."
        npm install
    else
        echo "检查依赖更新..."
        npm install
    fi
    
    echo -e "${GREEN}✅ 依赖安装完成${NC}"
}

# 启动开发服务器
start_dev_server() {
    echo -e "\n${YELLOW}启动 Web 开发服务器...${NC}"
    
    echo -e "${GREEN}开发服务器启动成功！${NC}"
    echo ""
    echo -e "${BLUE}访问地址:${NC}"
    echo "  本地: http://localhost:3000"
    echo "  网络: http://$(hostname -I | awk '{print $1}'):3000"
    echo ""
    echo -e "${BLUE}默认登录信息:${NC}"
    echo "  用户名: admin"
    echo "  密码: Admin@123"
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 停止开发服务器${NC}"
    echo "=================================================="
    
    # 设置环境变量
    export REACT_APP_API_BASE_URL=http://localhost:8080/api/v1
    export BROWSER=none  # 防止自动打开浏览器
    
    # 启动开发服务器
    npm start
}

# 显示帮助信息
show_help() {
    echo -e "\n${BLUE}开发提示:${NC}"
    echo ""
    echo "1. 代码热重载:"
    echo "   修改代码后会自动重新编译和刷新浏览器"
    echo ""
    echo "2. 开发工具:"
    echo "   - React Developer Tools"
    echo "   - Redux DevTools (如果使用)"
    echo ""
    echo "3. 构建生产版本:"
    echo "   npm run build"
    echo ""
    echo "4. 代码检查:"
    echo "   npm run lint"
    echo ""
    echo "5. 类型检查:"
    echo "   npx tsc --noEmit"
    echo ""
}

# 主函数
main() {
    echo -e "${BLUE}开始设置 DataFusion Web 开发环境...${NC}"
    
    check_nodejs
    check_api_server
    install_dependencies
    show_help
    
    echo -e "\n${GREEN}🎉 环境检查完成！${NC}"
    echo -e "\n${YELLOW}是否立即启动开发服务器？ (y/n)${NC}"
    read -r response
    
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        start_dev_server
    else
        echo -e "\n${BLUE}手动启动命令:${NC}"
        echo "  cd web"
        echo "  npm start"
    fi
}

# 错误处理
trap 'echo -e "\n${RED}❌ 脚本执行被中断${NC}"; exit 1' INT

# 检查是否在正确的目录
if [ ! -f "web/package.json" ]; then
    echo -e "${RED}❌ 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 运行主函数
main "$@"