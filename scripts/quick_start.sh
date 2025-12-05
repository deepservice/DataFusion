#!/bin/bash

# DataFusion Worker 快速启动脚本

set -e

echo "=========================================="
echo "DataFusion Worker 快速验证"
echo "=========================================="
echo ""

# 检查 PostgreSQL
echo "1. 检查 PostgreSQL..."
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL 未安装，请先安装 PostgreSQL"
    exit 1
fi
echo "✅ PostgreSQL 已安装"
echo ""

# 检查 Go
echo "2. 检查 Go 环境..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.21+"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Go 已安装: $GO_VERSION"
echo ""

# 下载依赖
echo "3. 下载 Go 依赖..."
go mod download
echo "✅ 依赖下载完成"
echo ""

# 初始化数据库
echo "4. 初始化数据库..."
echo "请输入 PostgreSQL 用户名 (默认: postgres):"
read -r PG_USER
PG_USER=${PG_USER:-postgres}

echo "正在创建数据库..."
psql -U "$PG_USER" -c "CREATE DATABASE datafusion_control;" 2>/dev/null || echo "数据库 datafusion_control 已存在"
psql -U "$PG_USER" -c "CREATE DATABASE datafusion_data;" 2>/dev/null || echo "数据库 datafusion_data 已存在"

echo "正在创建表结构..."
psql -U "$PG_USER" -d datafusion_control -f scripts/init_db.sql
echo "✅ 数据库初始化完成"
echo ""

# 插入测试任务
echo "5. 插入测试任务..."
psql -U "$PG_USER" -d datafusion_control -f scripts/insert_test_task.sql
echo "✅ 测试任务插入完成"
echo ""

# 编译 Worker
echo "6. 编译 Worker..."
go build -o bin/worker cmd/worker/main.go
echo "✅ 编译完成"
echo ""

echo "=========================================="
echo "✅ 环境准备完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 修改配置文件: config/worker.yaml"
echo "2. 启动 Worker: ./bin/worker -config config/worker.yaml"
echo "3. 或者使用: make run"
echo ""
echo "查看任务执行情况："
echo "  psql -U $PG_USER -d datafusion_control -c 'SELECT * FROM task_executions ORDER BY start_time DESC LIMIT 10;'"
echo ""
