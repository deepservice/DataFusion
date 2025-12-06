package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datafusion/worker/internal/api"
	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/database"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config/api-server.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log := logger.NewLogger(cfg.Log.Level, cfg.Log.Format)
	defer log.Sync()

	log.Info("启动 DataFusion API Server", zap.String("version", "v1.0.0"))

	// 初始化数据库连接
	db, err := database.NewPostgresDB(cfg.Database.PostgreSQL)
	if err != nil {
		log.Fatal("数据库连接失败", zap.Error(err))
	}
	defer db.Close()

	// 初始化路由
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(api.LoggerMiddleware(log))
	router.Use(api.CORSMiddleware())

	// 注册路由
	api.RegisterRoutes(router, db, log)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		log.Info("API Server 启动", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭 API Server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("服务器关闭失败", zap.Error(err))
	}

	log.Info("API Server 已关闭")
}
