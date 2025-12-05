#!/bin/bash

echo "=========================================="
echo "DataFusion Worker 快速验证"
echo "=========================================="
echo ""

# 检查 PostgreSQL
echo "步骤 1: 检查 PostgreSQL..."
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL 已安装"
    
    # 测试连接
    if psql -U postgres -c "SELECT 1;" &> /dev/null; then
        echo "✅ PostgreSQL 连接成功"
    else
        echo "❌ PostgreSQL 连接失败，请检查："
        echo "   1. PostgreSQL 服务是否运行"
        echo "   2. 用户名密码是否正确"
        echo ""
        echo "尝试启动 PostgreSQL:"
        echo "   sudo systemctl start postgresql"
        exit 1
    fi
else
    echo "❌ PostgreSQL 未安装"
    exit 1
fi

echo ""
echo "步骤 2: 创建数据库..."

# 创建数据库（忽略已存在的错误）
psql -U postgres -c "CREATE DATABASE datafusion_control;" 2>/dev/null || echo "   数据库 datafusion_control 已存在"
psql -U postgres -c "CREATE DATABASE datafusion_data;" 2>/dev/null || echo "   数据库 datafusion_data 已存在"

echo "✅ 数据库准备完成"

echo ""
echo "步骤 3: 初始化表结构..."
psql -U postgres -d datafusion_control -f scripts/init_db.sql > /dev/null 2>&1
echo "✅ 表结构初始化完成"

echo ""
echo "步骤 4: 插入测试任务（API 采集）..."

# 插入一个简单的 API 测试任务
psql -U postgres -d datafusion_control << 'EOF'
-- 删除旧的测试任务
DELETE FROM collection_tasks WHERE name LIKE '快速验证%';

-- 插入新的测试任务
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '快速验证-API测试',
    'api',
    'enabled',
    '*/1 * * * *',
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://jsonplaceholder.typicode.com/users?_limit=3",
            "method": "GET",
            "headers": {},
            "selectors": {
                "_data_path": "",
                "id": "id",
                "name": "name",
                "email": "email",
                "username": "username"
            }
        },
        "processor": {
            "cleaning_rules": [
                {"field": "name", "type": "trim"},
                {"field": "email", "type": "lowercase"}
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "file",
            "database": "test_output",
            "table": "users",
            "mapping": {}
        }
    }'
);

SELECT id, name, type, status, next_run_time FROM collection_tasks WHERE name LIKE '快速验证%';
EOF

echo "✅ 测试任务插入完成"

echo ""
echo "步骤 5: 编译 Worker..."
go build -o bin/worker cmd/worker/main.go
if [ $? -eq 0 ]; then
    echo "✅ Worker 编译成功"
else
    echo "❌ Worker 编译失败"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ 环境准备完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 启动 Worker:"
echo "   ./bin/worker -config config/worker.yaml"
echo ""
echo "2. 等待 1 分钟后，查看采集结果:"
echo "   ls -lh data/test_output/"
echo "   cat data/test_output/users_*.json | jq ."
echo ""
echo "3. 查看执行记录:"
echo "   psql -U postgres -d datafusion_control -c \"SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 5;\""
echo ""
