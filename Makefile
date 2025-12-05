.PHONY: help build run test clean init-db docker-build

help:
	@echo "DataFusion Worker - Makefile 命令"
	@echo ""
	@echo "  make build       - 编译 Worker"
	@echo "  make run         - 运行 Worker"
	@echo "  make test        - 运行测试"
	@echo "  make clean       - 清理编译文件"
	@echo "  make init-db     - 初始化数据库"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo ""

build:
	@echo "编译 Worker..."
	go build -o bin/worker cmd/worker/main.go
	@echo "编译完成: bin/worker"

run:
	@echo "启动 Worker..."
	go run cmd/worker/main.go -config config/worker.yaml

test:
	@echo "运行测试..."
	go test -v ./...

clean:
	@echo "清理编译文件..."
	rm -rf bin/
	rm -rf data/
	@echo "清理完成"

init-db:
	@echo "初始化数据库..."
	psql -U postgres -f scripts/init_db.sql
	@echo "数据库初始化完成"

insert-test-task:
	@echo "插入测试任务..."
	psql -U postgres -f scripts/insert_test_task.sql
	@echo "测试任务插入完成"

docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t datafusion-worker:latest .
	@echo "Docker 镜像构建完成"

deps:
	@echo "下载依赖..."
	go mod download
	go mod tidy
	@echo "依赖下载完成"
