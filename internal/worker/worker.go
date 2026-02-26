package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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

		// 无论成功或失败，都更新下次执行时间，避免重复轮询
		// 注意：不调用 UnlockTask（它会将 next_run_time 设为 NOW()，导致立即重新拾取）
		if err := w.updateNextRunTime(ctx, &task); err != nil {
			log.Printf("更新下次执行时间失败: %v", err)
		}
	}
}

// parseTaskConfig 解析任务配置
func (w *Worker) parseTaskConfig(configJSON string) (*models.TaskConfig, error) {
	return database.ParseTaskConfig(configJSON)
}

// resolveTaskConfig 解析任务配置：优先使用 task.Config，为空时从数据源自动构建
func (w *Worker) resolveTaskConfig(task *models.CollectionTask) (*models.TaskConfig, error) {
	// 如果任务有完整配置，直接使用
	if task.Config != nil && *task.Config != "" {
		return w.parseTaskConfig(*task.Config)
	}

	// 从关联的数据源自动构建配置
	if task.DataSourceID == 0 {
		return nil, fmt.Errorf("任务没有配置且未关联数据源")
	}

	dsType, dsConfigJSON, err := w.db.GetDataSourceConfig(task.DataSourceID)
	if err != nil {
		return nil, fmt.Errorf("获取数据源配置失败: %w", err)
	}

	// 解析数据源配置JSON
	var dsConfig map[string]interface{}
	if err := json.Unmarshal([]byte(dsConfigJSON), &dsConfig); err != nil {
		return nil, fmt.Errorf("解析数据源配置JSON失败: %w", err)
	}

	// 构建 DataSourceConfig
	url, _ := dsConfig["url"].(string)
	if url == "" {
		url, _ = dsConfig["endpoint"].(string) // 兼容旧数据源配置（使用 endpoint 字段）
	}
	method, _ := dsConfig["method"].(string)
	if method == "" {
		method = "GET"
	}

	// 转换 headers
	headers := map[string]string{}
	if h, ok := dsConfig["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	// 转换 selectors
	selectors := map[string]string{}
	if s, ok := dsConfig["selectors"].(map[string]interface{}); ok {
		for k, v := range s {
			if str, ok := v.(string); ok {
				selectors[k] = str
			}
		}
	}

	// 转换 rpa_config（登录配置、动态动作等）
	var rpaConf *models.RPAConfig
	if rpaRaw, ok := dsConfig["rpa_config"].(map[string]interface{}); ok {
		rpaBytes, _ := json.Marshal(rpaRaw)
		var rc models.RPAConfig
		if err := json.Unmarshal(rpaBytes, &rc); err == nil {
			rpaConf = &rc
		}
	}

	// 根据任务类型映射到采集器类型
	collectorType := dsType
	if task.Type == "web-rpa" {
		collectorType = "web-rpa"
	} else if task.Type == "api" {
		collectorType = "api"
	} else if task.Type == "database" {
		collectorType = "database"
	}

	taskConfig := &models.TaskConfig{
		DataSource: models.DataSourceConfig{
			Type:      collectorType,
			URL:       url,
			Method:    method,
			Headers:   headers,
			Selectors: selectors,
			RPAConfig: rpaConf,
		},
		Processor: models.ProcessorConfig{},
		Storage: models.StorageConfig{
			Target:   "postgresql",
			Database: "datafusion_data",
			Table:    fmt.Sprintf("collected_%s_%d", strings.ReplaceAll(task.Type, "-", "_"), task.ID),
		},
	}

	log.Printf("自动构建任务配置: 数据源=%s, URL=%s, 存储表=%s",
		collectorType, url, taskConfig.Storage.Table)

	return taskConfig, nil
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
	if task.Cron == nil || *task.Cron == "" {
		// 没有cron表达式的一次性任务，清空next_run_time防止重复执行
		return w.db.ClearTaskNextRunTime(task.ID)
	}

	cronExpr := strings.TrimSpace(*task.Cron)
	// 移除 Quartz 风格的 '?' 字符（替换为 '*'）
	cronExpr = strings.ReplaceAll(cronExpr, "?", "*")

	// 尝试6字段（含秒）解析，失败则回退到5字段
	parser6 := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser6.Parse(cronExpr)
	if err != nil {
		parser5 := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err = parser5.Parse(cronExpr)
		if err != nil {
			return fmt.Errorf("解析 Cron 表达式失败: %w", err)
		}
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
