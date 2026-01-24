#!/bin/bash

# DataFusion Docker 快速启动脚本

set -e

echo "=========================================="
echo "DataFusion Docker 快速启动"
echo "=========================================="
echo ""

# 检查 Docker
echo "1. 检查 Docker..."
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi
echo "✅ Docker 已安装"
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

# 启动 PostgreSQL 容器
echo "3. 启动 PostgreSQL 容器..."
if docker ps -a --format 'table {{.Names}}' | grep -q "datafusion-postgres"; then
    echo "📦 PostgreSQL 容器已存在，检查状态..."
    if docker ps --format 'table {{.Names}}' | grep -q "datafusion-postgres"; then
        echo "✅ PostgreSQL 容器正在运行"
    else
        echo "🔄 启动现有的 PostgreSQL 容器..."
        docker start datafusion-postgres
        sleep 10
    fi
else
    echo "🚀 创建并启动 PostgreSQL 容器..."
    docker run -d \
        --name datafusion-postgres \
        -e POSTGRES_PASSWORD=postgres \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_DB=postgres \
        -p 5432:5432 \
        postgres:14
    
    echo "⏳ 等待 PostgreSQL 启动（30秒）..."
    sleep 30
fi
echo ""

# 初始化数据库
echo "4. 初始化数据库..."
echo "创建数据库..."
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_control;" 2>/dev/null || echo "数据库 datafusion_control 已存在"
docker exec -i datafusion-postgres psql -U postgres -c "CREATE DATABASE datafusion_data;" 2>/dev/null || echo "数据库 datafusion_data 已存在"

echo "初始化控制面数据库..."
docker exec -i datafusion-postgres psql -U postgres -d datafusion_control < scripts/init_control_db.sql

echo "初始化数据面数据库..."
docker exec -i datafusion-postgres psql -U postgres -d datafusion_data < scripts/init_db.sql
echo "✅ 数据库初始化完成"
echo ""

# 编译 API Server
echo "5. 编译 API Server..."
go build -o bin/api-server ./cmd/api-server
echo "✅ 编译完成"
echo ""

echo "=========================================="
echo "✅ 环境准备完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 启动 API Server: ./bin/api-server"
echo "2. 测试 API: curl http://localhost:8081/healthz"
echo "3. 运行完整测试: ./tests/test_api_server.sh"
echo ""
echo "管理 PostgreSQL 容器："
echo "  查看状态: docker ps"
echo "  查看日志: docker logs datafusion-postgres"
echo "  停止容器: docker stop datafusion-postgres"
echo "  重启容器: docker start datafusion-postgres"
echo ""