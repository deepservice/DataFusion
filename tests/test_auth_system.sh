#!/bin/bash

# DataFusion Authentication System Test Script
# 测试认证系统的基本功能

set -e

API_BASE="http://localhost:8080/api/v1"
ADMIN_TOKEN=""

echo "🚀 开始测试 DataFusion 认证系统..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local headers=$6

    echo -e "\n${YELLOW}测试: $description${NC}"
    echo "请求: $method $endpoint"
    
    if [ -n "$headers" ]; then
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data" \
                "$API_BASE$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -H "$headers" \
                "$API_BASE$endpoint")
        fi
    else
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -H "Content-Type: application/json" \
                -d "$data" \
                "$API_BASE$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                "$API_BASE$endpoint")
        fi
    fi
    
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$status_code" = "$expected_status" ]; then
        echo -e "${GREEN}✅ 成功 (状态码: $status_code)${NC}"
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo "响应: $body" | jq . 2>/dev/null || echo "响应: $body"
        fi
    else
        echo -e "${RED}❌ 失败 (期望: $expected_status, 实际: $status_code)${NC}"
        echo "响应: $body"
        return 1
    fi
}

# 等待服务启动
wait_for_service() {
    echo "等待 API 服务启动..."
    for i in {1..30}; do
        if curl -s http://localhost:8080/healthz > /dev/null 2>&1; then
            echo -e "${GREEN}✅ API 服务已启动${NC}"
            return 0
        fi
        echo "等待中... ($i/30)"
        sleep 2
    done
    echo -e "${RED}❌ API 服务启动超时${NC}"
    exit 1
}

# 检查服务是否运行
check_service() {
    if ! curl -s http://localhost:8080/healthz > /dev/null 2>&1; then
        echo -e "${RED}❌ API 服务未运行，请先启动服务:${NC}"
        echo "  cd $(pwd)"
        echo "  ./api-server"
        exit 1
    fi
}

# 主测试流程
main() {
    echo "检查 API 服务状态..."
    check_service
    
    echo -e "\n${YELLOW}=== 1. 健康检查测试 ===${NC}"
    test_endpoint "GET" "/healthz" "" "200" "健康检查"
    
    echo -e "\n${YELLOW}=== 2. 未认证访问测试 ===${NC}"
    test_endpoint "GET" "/tasks" "" "401" "未认证访问任务列表"
    
    echo -e "\n${YELLOW}=== 3. 用户登录测试 ===${NC}"
    login_data='{"username":"admin","password":"admin123"}'
    test_endpoint "POST" "/auth/login" "$login_data" "200" "管理员登录"
    
    # 提取 token
    login_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$login_data" \
        "$API_BASE/auth/login")
    
    ADMIN_TOKEN=$(echo "$login_response" | jq -r '.token' 2>/dev/null || echo "")
    
    if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
        echo -e "${RED}❌ 无法获取登录 token${NC}"
        echo "登录响应: $login_response"
        exit 1
    fi
    
    echo -e "${GREEN}✅ 成功获取 token: ${ADMIN_TOKEN:0:20}...${NC}"
    
    echo -e "\n${YELLOW}=== 4. 认证访问测试 ===${NC}"
    auth_header="Authorization: Bearer $ADMIN_TOKEN"
    test_endpoint "GET" "/auth/profile" "" "200" "获取用户信息" "$auth_header"
    
    echo -e "\n${YELLOW}=== 5. 权限访问测试 ===${NC}"
    test_endpoint "GET" "/tasks" "" "200" "获取任务列表" "$auth_header"
    test_endpoint "GET" "/datasources" "" "200" "获取数据源列表" "$auth_header"
    test_endpoint "GET" "/users" "" "200" "获取用户列表（管理员权限）" "$auth_header"
    
    echo -e "\n${YELLOW}=== 6. 用户管理测试 ===${NC}"
    create_user_data='{"username":"testuser","password":"Test123456","email":"test@example.com","role":"user"}'
    test_endpoint "POST" "/users" "$create_user_data" "201" "创建测试用户" "$auth_header"
    
    echo -e "\n${YELLOW}=== 7. API 密钥管理测试 ===${NC}"
    create_apikey_data='{"name":"测试密钥","description":"用于测试的API密钥","permissions":["read"]}'
    test_endpoint "POST" "/api-keys" "$create_apikey_data" "201" "创建API密钥" "$auth_header"
    
    echo -e "\n${YELLOW}=== 8. 角色权限测试 ===${NC}"
    test_endpoint "GET" "/roles" "" "200" "获取角色列表" "$auth_header"
    
    echo -e "\n${YELLOW}=== 9. 错误处理测试 ===${NC}"
    wrong_login_data='{"username":"admin","password":"wrongpassword"}'
    test_endpoint "POST" "/auth/login" "$wrong_login_data" "401" "错误密码登录"
    
    invalid_token_header="Authorization: Bearer invalid_token"
    test_endpoint "GET" "/auth/profile" "" "401" "无效token访问" "$invalid_token_header"
    
    echo -e "\n${GREEN}🎉 所有认证系统测试完成！${NC}"
    echo -e "\n${YELLOW}测试总结:${NC}"
    echo "✅ 健康检查正常"
    echo "✅ 用户认证功能正常"
    echo "✅ JWT Token 生成和验证正常"
    echo "✅ 权限控制正常"
    echo "✅ 用户管理功能正常"
    echo "✅ API密钥管理功能正常"
    echo "✅ 错误处理正常"
    
    echo -e "\n${YELLOW}下一步建议:${NC}"
    echo "1. 启动 Web 界面开发"
    echo "2. 完善权限细粒度控制"
    echo "3. 添加更多测试用例"
    echo "4. 实现 OAuth 集成"
}

# 运行测试
main "$@"