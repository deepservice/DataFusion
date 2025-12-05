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

	log.Println("收到退出信号，正在关闭 Worker...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("Worker 已关闭")
}
