package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/datafusion/worker/internal/collector"
	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/database"
	"github.com/datafusion/worker/internal/models"
	"github.com/datafusion/worker/internal/processor"
	"github.com/datafusion/worker/internal/storage"
)

// Worker 工作节点
type Worker struct {
	config            *config.Config
	db                *database.PostgresDB
	collectorFactory  *collector.CollectorFactory
	storageFactory    *storage.StorageFactory
	podName           string
}

// NewWorker 创建 Worker
func NewWorker(cfg *config.Config) (*Worker, error) {
	// 连接数据库
	db, err := database.NewPostgresDBFromConfig(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 创建采集器工厂
	collectorFactory := collector.NewCollectorFactory()
	
	// 注册 RPA 采集器
	rpaCollector := collector.NewRPACollector(cfg.Collector.RPA.Headless, cfg.Collector.RPA.Timeout)
	collectorFactory.Register(rpaCollector)
	
	// 注册 API 采集器
	apiCollector := collector.NewAPICollector(cfg.Collector.API.Timeout)
	collectorFactory.Register(apiCollector)
	
	// 注册数据库采集器
	dbCollector := collector.NewDBCollector(cfg.Collector.API.Timeout) // 使用相同的超时配置
	collectorFactory.Register(dbCollector)

	// 创建存储工厂
	storageFactory := storage.NewStorageFactory()
	
	// 注册文件存储
	fileStorage := storage.NewFileStorage("./data")
	storageFactory.Register(fileStorage)
	
	// 注册 PostgreSQL 存储（如果配置了）
	if cfg.Storage.Type == "postgresql" {
		pgStorage, err := storage.NewPostgresStorage(
			cfg.Storage.Database.Host,
			cfg.Storage.Database.Port,
			cfg.Storage.Database.User,
			cfg.Storage.Database.Password,
			cfg.Storage.Database.Database,
			cfg.Storage.Database.SSLMode,
		)
		if err != nil {
			log.Printf("警告: 创建 PostgreSQL 存储失败: %v", err)
		} else {
			storageFactory.Register(pgStorage)
		}
	}

	// 获取 Pod 名称
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = fmt.Sprintf("worker-%d", time.Now().Unix())
	}

	return &Worker{
		config:           cfg,
		db:               db,
		collectorFactory: collectorFactory,
		storageFactory:   storageFactory,
		podName:          podName,
	}, nil
}

// Start 启动 Worker
func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Worker 启动: %s, 类型: %s", w.podName, w.config.WorkerType)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// 立即执行一次
	w.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker 收到停止信号")
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

// poll 轮询待执行任务
func (w *Worker) poll(ctx context.Context) {
	// 查询待执行任务
	tasks, err := w.db.GetPendingTasks(w.config.WorkerType)
	if err != nil {
		log.Printf("查询待执行任务失败: %v", err)
		return
	}

	if len(tasks) == 0 {
		log.Printf("没有待执行任务")
		return
	}

	log.Printf("发现 %d 个待执行任务", len(tasks))

	// 处理每个任务
	for _, task := range tasks {
		// 尝试获取任务锁
		locked, err := w.db.TryLockTask(task.ID, w.podName)
		if err != nil {
			log.Printf("获取任务锁失败: %v", err)
			continue
		}

		if !locked {
			log.Printf("任务 %s (ID: %d) 已被其他 Worker 锁定", task.Name, task.ID)
			continue
		}

		log.Printf("成功锁定任务 %s (ID: %d)，开始执行", task.Name, task.ID)

		// 执行任务（带重试）
		if err := w.executeWithRetry(ctx, &task); err != nil {
			log.Printf("任务执行最终失败: %v", err)
		}

		// 释放锁
		if err := w.db.UnlockTask(task.ID); err != nil {
			log.Printf("释放任务锁失败: %v", err)
		}
	}
}

// parseTaskConfig 解析任务配置
func (w *Worker) parseTaskConfig(configJSON string) (*models.TaskConfig, error) {
	return database.ParseTaskConfig(configJSON)
}

// collectData 采集数据
func (w *Worker) collectData(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error) {
	col, ok := w.collectorFactory.Get(config.Type)
	if !ok {
		return nil, fmt.Errorf("不支持的采集器类型: %s", config.Type)
	}

	return col.Collect(ctx, config)
}

// processData 处理数据
func (w *Worker) processData(data []map[string]interface{}, config *models.ProcessorConfig) ([]map[string]interface{}, error) {
	proc := processor.NewProcessor(config)
	return proc.Process(data)
}

// storeData 存储数据
func (w *Worker) storeData(ctx context.Context, config *models.StorageConfig, data []map[string]interface{}) error {
	stor, ok := w.storageFactory.Get(config.Target)
	if !ok {
		return fmt.Errorf("不支持的存储类型: %s", config.Target)
	}

	return stor.Store(ctx, config, data)
}

// updateNextRunTime 更新下次执行时间
func (w *Worker) updateNextRunTime(ctx context.Context, task *models.CollectionTask) error {
	if task.Cron == "" {
		return nil
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(task.Cron)
	if err != nil {
		return fmt.Errorf("解析 Cron 表达式失败: %w", err)
	}

	nextRunTime := schedule.Next(time.Now())
	return w.db.UpdateTaskNextRunTime(task.ID, nextRunTime.Format("2006-01-02 15:04:05"))
}

// finishExecution 完成执行
func (w *Worker) finishExecution(ctx context.Context, execution *models.TaskExecution, status string, recordsCollected int, errorMsg string) {
	endTime := time.Now()
	execution.Status = status
	execution.EndTime = &endTime
	execution.RecordsCollected = recordsCollected
	execution.ErrorMessage = errorMsg

	if err := w.db.UpdateExecution(ctx, execution); err != nil {
		log.Printf("更新执行记录失败: %v", err)
	}
}

// GetDB 获取数据库连接（用于健康检查）
func (w *Worker) GetDB() *database.PostgresDB {
	return w.db
}

// Shutdown 优雅关闭 Worker
func (w *Worker) Shutdown(ctx context.Context) error {
	log.Println("开始优雅关闭 Worker...")
	
	// 这里可以添加等待正在运行的任务完成的逻辑
	// 目前简单实现，直接返回
	
	log.Println("Worker 优雅关闭完成")
	return nil
}
