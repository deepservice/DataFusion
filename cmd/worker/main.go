package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/health"
	"github.com/datafusion/worker/internal/metrics"
	"github.com/datafusion/worker/internal/worker"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config/worker.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建 Worker
	w, err := worker.NewWorker(cfg)
	if err != nil {
		log.Fatalf("创建 Worker 失败: %v", err)
	}

	// 创建健康检查器
	healthChecker := health.NewHealthChecker(w.GetDB())

	// 启动健康检查服务器
	go func() {
		log.Println("启动健康检查服务器，端口: 8080")
		if err := health.StartHealthServer(8080, healthChecker); err != nil {
			log.Printf("健康检查服务器启动失败: %v", err)
		}
	}()

	// 启动指标服务器
	go func() {
		log.Println("启动指标服务器，端口: 9090")
		if err := metrics.StartMetricsServer(9090); err != nil {
			log.Printf("指标服务器启动失败: %v", err)
		}
	}()

	// 启动 Worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := w.Start(ctx); err != nil {
			log.Fatalf("Worker 启动失败: %v", err)
		}
	}()

	log.Printf("Worker 启动成功，轮询间隔: %v", cfg.PollInterval)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("收到退出信号，正在优雅关闭 Worker...")

	// 创建关闭超时上下文
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 优雅关闭 Worker
	if err := w.Shutdown(shutdownCtx); err != nil {
		log.Printf("Worker 关闭失败: %v", err)
	}

	// 取消主上下文
	cancel()
	log.Println("Worker 已关闭")
}
