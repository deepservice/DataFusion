#!/bin/bash

# DataFusion 性能测试脚本
# 用于测试系统在不同负载下的性能表现

set -e

# 配置
API_SERVER_URL="http://localhost:8080"
TEST_DURATION="60s"
CONCURRENT_USERS="50"
RAMP_UP_TIME="10s"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖工具..."
    
    # 检查k6
    if ! command -v k6 &> /dev/null; then
        log_error "k6 未安装，请先安装 k6"
        log_info "安装命令: brew install k6 (macOS) 或访问 https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
    
    # 检查curl
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi
    
    # 检查jq
    if ! command -v jq &> /dev/null; then
        log_warning "jq 未安装，某些功能可能受限"
    fi
    
    log_success "依赖检查完成"
}

# 检查API服务器状态
check_api_server() {
    log_info "检查API服务器状态..."
    
    if curl -s -f "${API_SERVER_URL}/healthz" > /dev/null; then
        log_success "API服务器运行正常"
    else
        log_error "API服务器不可访问: ${API_SERVER_URL}"
        log_info "请确保API服务器正在运行"
        exit 1
    fi
}

# 获取认证令牌
get_auth_token() {
    log_info "获取认证令牌..."
    
    # 尝试登录获取令牌
    local login_response=$(curl -s -X POST "${API_SERVER_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' || echo "")
    
    if [ -n "$login_response" ] && command -v jq &> /dev/null; then
        AUTH_TOKEN=$(echo "$login_response" | jq -r '.data.token // empty')
        if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
            log_success "认证令牌获取成功"
            return 0
        fi
    fi
    
    log_warning "无法获取认证令牌，将使用匿名访问"
    AUTH_TOKEN=""
}

# 创建k6测试脚本
create_k6_script() {
    local script_name="$1"
    local test_type="$2"
    
    cat > "${script_name}" << EOF
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// 自定义指标
export let errorRate = new Rate('errors');

// 测试配置
export let options = {
    stages: [
        { duration: '${RAMP_UP_TIME}', target: ${CONCURRENT_USERS} },
        { duration: '${TEST_DURATION}', target: ${CONCURRENT_USERS} },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<2000'], // 95%的请求应在2秒内完成
        http_req_failed: ['rate<0.05'],    // 错误率应低于5%
        errors: ['rate<0.05'],
    },
};

// 基础URL和认证
const BASE_URL = '${API_SERVER_URL}';
const AUTH_TOKEN = '${AUTH_TOKEN}';

// 请求头
const headers = {
    'Content-Type': 'application/json',
};

if (AUTH_TOKEN) {
    headers['Authorization'] = \`Bearer \${AUTH_TOKEN}\`;
}

// 测试场景
export default function() {
    let response;
    
    switch('${test_type}') {
        case 'api_load':
            testAPILoad();
            break;
        case 'task_crud':
            testTaskCRUD();
            break;
        case 'cache_performance':
            testCachePerformance();
            break;
        default:
            testAPILoad();
    }
    
    sleep(1);
}

// API负载测试
function testAPILoad() {
    const endpoints = [
        '/api/v1/tasks',
        '/api/v1/datasources', 
        '/api/v1/stats/overview',
        '/api/v1/executions',
    ];
    
    endpoints.forEach(endpoint => {
        let response = http.get(\`\${BASE_URL}\${endpoint}\`, { headers });
        
        let success = check(response, {
            'status is 200': (r) => r.status === 200,
            'response time < 2000ms': (r) => r.timings.duration < 2000,
        });
        
        errorRate.add(!success);
    });
}

// 任务CRUD测试
function testTaskCRUD() {
    // 创建任务
    let taskData = {
        name: \`LoadTest-Task-\${Math.random().toString(36).substr(2, 9)}\`,
        description: 'Load test task',
        type: 'api',
        config: {
            url: 'https://api.example.com/data',
            method: 'GET'
        },
        schedule: '0 */5 * * * *'
    };
    
    let createResponse = http.post(\`\${BASE_URL}/api/v1/tasks\`, 
        JSON.stringify(taskData), { headers });
    
    let createSuccess = check(createResponse, {
        'task created': (r) => r.status === 201,
    });
    
    if (createSuccess && createResponse.json('data.id')) {
        let taskId = createResponse.json('data.id');
        
        // 获取任务
        let getResponse = http.get(\`\${BASE_URL}/api/v1/tasks/\${taskId}\`, { headers });
        check(getResponse, {
            'task retrieved': (r) => r.status === 200,
        });
        
        // 更新任务
        taskData.description = 'Updated load test task';
        let updateResponse = http.put(\`\${BASE_URL}/api/v1/tasks/\${taskId}\`,
            JSON.stringify(taskData), { headers });
        check(updateResponse, {
            'task updated': (r) => r.status === 200,
        });
        
        // 删除任务
        let deleteResponse = http.del(\`\${BASE_URL}/api/v1/tasks/\${taskId}\`, null, { headers });
        check(deleteResponse, {
            'task deleted': (r) => r.status === 200,
        });
    }
    
    errorRate.add(!createSuccess);
}

// 缓存性能测试
function testCachePerformance() {
    // 第一次请求（缓存未命中）
    let firstResponse = http.get(\`\${BASE_URL}/api/v1/stats/overview\`, { headers });
    let firstLatency = firstResponse.timings.duration;
    
    // 第二次请求（缓存命中）
    let secondResponse = http.get(\`\${BASE_URL}/api/v1/stats/overview\`, { headers });
    let secondLatency = secondResponse.timings.duration;
    
    check(secondResponse, {
        'cache hit faster': () => secondLatency < firstLatency,
        'cache hit under 100ms': () => secondLatency < 100,
    });
}
EOF
}

# 运行API负载测试
run_api_load_test() {
    log_info "运行API负载测试..."
    
    local script_file="k6_api_load_test.js"
    create_k6_script "$script_file" "api_load"
    
    log_info "测试配置:"
    log_info "  - 并发用户: ${CONCURRENT_USERS}"
    log_info "  - 测试时长: ${TEST_DURATION}"
    log_info "  - 预热时间: ${RAMP_UP_TIME}"
    
    k6 run --out json=api_load_results.json "$script_file"
    
    if [ $? -eq 0 ]; then
        log_success "API负载测试完成"
    else
        log_error "API负载测试失败"
    fi
    
    rm -f "$script_file"
}

# 运行任务CRUD测试
run_task_crud_test() {
    log_info "运行任务CRUD性能测试..."
    
    local script_file="k6_task_crud_test.js"
    create_k6_script "$script_file" "task_crud"
    
    k6 run --out json=task_crud_results.json "$script_file"
    
    if [ $? -eq 0 ]; then
        log_success "任务CRUD测试完成"
    else
        log_error "任务CRUD测试失败"
    fi
    
    rm -f "$script_file"
}

# 运行缓存性能测试
run_cache_performance_test() {
    log_info "运行缓存性能测试..."
    
    local script_file="k6_cache_test.js"
    create_k6_script "$script_file" "cache_performance"
    
    k6 run --out json=cache_results.json "$script_file"
    
    if [ $? -eq 0 ]; then
        log_success "缓存性能测试完成"
    else
        log_error "缓存性能测试失败"
    fi
    
    rm -f "$script_file"
}

# 运行数据库性能测试
run_database_performance_test() {
    log_info "运行数据库性能测试..."
    
    # 运行Go测试
    cd "$(dirname "$0")/.."
    go test -v ./tests/performance/ -run TestDatabasePerformance -timeout 5m
    
    if [ $? -eq 0 ]; then
        log_success "数据库性能测试完成"
    else
        log_error "数据库性能测试失败"
    fi
}

# 生成性能报告
generate_performance_report() {
    log_info "生成性能测试报告..."
    
    local report_file="performance_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# DataFusion 性能测试报告

**测试时间**: $(date)
**测试配置**:
- 并发用户: ${CONCURRENT_USERS}
- 测试时长: ${TEST_DURATION}
- API服务器: ${API_SERVER_URL}

## 测试结果摘要

EOF

    # 如果有jq，解析JSON结果
    if command -v jq &> /dev/null; then
        for result_file in *_results.json; do
            if [ -f "$result_file" ]; then
                echo "### $(basename "$result_file" .json)" >> "$report_file"
                echo "" >> "$report_file"
                
                # 提取关键指标
                local avg_duration=$(jq -r '.metrics.http_req_duration.avg // "N/A"' "$result_file")
                local p95_duration=$(jq -r '.metrics.http_req_duration.p95 // "N/A"' "$result_file")
                local error_rate=$(jq -r '.metrics.http_req_failed.rate // "N/A"' "$result_file")
                local rps=$(jq -r '.metrics.http_reqs.rate // "N/A"' "$result_file")
                
                echo "- 平均响应时间: ${avg_duration}ms" >> "$report_file"
                echo "- P95响应时间: ${p95_duration}ms" >> "$report_file"
                echo "- 错误率: ${error_rate}" >> "$report_file"
                echo "- 每秒请求数: ${rps}" >> "$report_file"
                echo "" >> "$report_file"
            fi
        done
    fi
    
    cat >> "$report_file" << EOF

## 建议

1. **响应时间优化**: 如果P95响应时间超过2秒，考虑优化数据库查询或增加缓存
2. **错误率控制**: 错误率应保持在5%以下
3. **并发处理**: 系统应能稳定处理${CONCURRENT_USERS}个并发用户
4. **缓存效果**: 缓存命中应显著降低响应时间

## 详细数据

详细的测试数据请查看对应的JSON文件：
$(ls -1 *_results.json 2>/dev/null | sed 's/^/- /' || echo "- 无详细数据文件")

EOF

    log_success "性能报告已生成: $report_file"
}

# 清理测试文件
cleanup() {
    log_info "清理测试文件..."
    rm -f k6_*.js
    rm -f *_results.json
}

# 主函数
main() {
    echo "=================================="
    echo "DataFusion 性能测试工具"
    echo "=================================="
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --url)
                API_SERVER_URL="$2"
                shift 2
                ;;
            --duration)
                TEST_DURATION="$2"
                shift 2
                ;;
            --users)
                CONCURRENT_USERS="$2"
                shift 2
                ;;
            --cleanup)
                cleanup
                exit 0
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --url URL          API服务器URL (默认: http://localhost:8080)"
                echo "  --duration TIME    测试持续时间 (默认: 60s)"
                echo "  --users NUM        并发用户数 (默认: 50)"
                echo "  --cleanup          清理测试文件"
                echo "  --help             显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行测试流程
    check_dependencies
    check_api_server
    get_auth_token
    
    log_info "开始性能测试..."
    
    # 运行各种测试
    run_api_load_test
    run_task_crud_test
    run_cache_performance_test
    run_database_performance_test
    
    # 生成报告
    generate_performance_report
    
    log_success "所有性能测试完成！"
    log_info "查看性能报告: performance_report_*.md"
}

# 捕获退出信号进行清理
trap cleanup EXIT

# 运行主函数
main "$@"