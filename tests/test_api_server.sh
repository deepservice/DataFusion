#!/bin/bash

# DataFusion API Server 测试脚本

API_URL="http://localhost:8081"

echo "========================================="
echo "DataFusion API Server 测试"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "${YELLOW}测试: ${description}${NC}"
    echo "请求: $method $endpoint"
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}✓ 成功 (HTTP $http_code)${NC}"
        echo "响应: $body" | jq '.' 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ 失败 (HTTP $http_code)${NC}"
        echo "响应: $body"
    fi
    echo ""
}

# 1. 健康检查
echo "========================================="
echo "1. 健康检查"
echo "========================================="
test_endpoint "GET" "/healthz" "" "服务健康检查"
test_endpoint "GET" "/readyz" "" "服务就绪检查"

# 2. 数据源管理
echo "========================================="
echo "2. 数据源管理"
echo "========================================="
test_endpoint "GET" "/api/v1/datasources" "" "获取数据源列表"
test_endpoint "GET" "/api/v1/datasources/1" "" "获取单个数据源"

# 创建新数据源
datasource_data='{
  "name": "测试API数据源",
  "type": "api",
  "config": "{\"url\":\"https://api.example.com/data\",\"method\":\"GET\"}",
  "description": "测试用API数据源",
  "status": "active"
}'
test_endpoint "POST" "/api/v1/datasources" "$datasource_data" "创建数据源"

# 3. 清洗规则管理
echo "========================================="
echo "3. 清洗规则管理"
echo "========================================="
test_endpoint "GET" "/api/v1/cleaning-rules" "" "获取清洗规则列表"
test_endpoint "GET" "/api/v1/cleaning-rules/1" "" "获取单个清洗规则"

# 创建新清洗规则
rule_data='{
  "name": "测试正则规则",
  "description": "测试用正则替换规则",
  "rule_type": "regex",
  "config": "{\"pattern\":\"[0-9]+\",\"replacement\":\"***\"}"
}'
test_endpoint "POST" "/api/v1/cleaning-rules" "$rule_data" "创建清洗规则"

# 4. 任务管理
echo "========================================="
echo "4. 任务管理"
echo "========================================="
test_endpoint "GET" "/api/v1/tasks" "" "获取任务列表"
test_endpoint "GET" "/api/v1/tasks?status=enabled" "" "获取启用的任务"

# 创建新任务
task_data='{
  "name": "测试采集任务",
  "description": "测试用Web采集任务",
  "type": "web-rpa",
  "data_source_id": 1,
  "cron": "0 * * * *",
  "status": "enabled",
  "replicas": 1,
  "execution_timeout": 3600,
  "max_retries": 3,
  "config": "{\"url\":\"https://example.com\",\"selectors\":{\"title\":\".title\"}}"
}'
test_endpoint "POST" "/api/v1/tasks" "$task_data" "创建任务"

# 获取任务详情
test_endpoint "GET" "/api/v1/tasks/1" "" "获取任务详情"

# 5. 执行历史
echo "========================================="
echo "5. 执行历史"
echo "========================================="
test_endpoint "GET" "/api/v1/executions" "" "获取执行历史列表"
test_endpoint "GET" "/api/v1/executions?status=success" "" "获取成功的执行记录"

# 6. 统计信息
echo "========================================="
echo "6. 统计信息"
echo "========================================="
test_endpoint "GET" "/api/v1/stats/overview" "" "获取系统概览"
test_endpoint "GET" "/api/v1/stats/tasks" "" "获取任务统计"

echo "========================================="
echo "测试完成！"
echo "========================================="
