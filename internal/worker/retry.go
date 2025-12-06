package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/datafusion/worker/internal/models"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  5 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
	}
}

// calculateDelay 计算重试延迟（指数退避）
func (p *RetryPolicy) calculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialDelay
	}

	// 指数退避: delay = initialDelay * (backoffFactor ^ attempt)
	delay := float64(p.InitialDelay) * pow(p.BackoffFactor, float64(attempt))
	
	// 限制最大延迟
	if delay > float64(p.MaxDelay) {
		return p.MaxDelay
	}
	
	return time.Duration(delay)
}

// pow 计算幂次方
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// shouldRetry 判断是否应该重试
func (p *RetryPolicy) shouldRetry(attempt int, err error) bool {
	if err == nil {
		return false
	}
	
	if attempt >= p.MaxRetries {
		return false
	}
	
	// 可以根据错误类型判断是否应该重试
	// 例如：网络错误应该重试，配置错误不应该重试
	return true
}

// executeWithRetry 带重试的任务执行
func (w *Worker) executeWithRetry(ctx context.Context, task *models.CollectionTask) error {
	policy := DefaultRetryPolicy()
	
	// 如果任务配置了最大重试次数，使用任务配置
	if task.MaxRetries > 0 {
		policy.MaxRetries = task.MaxRetries
	}
	
	var lastErr error
	
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// 第一次尝试不延迟，后续尝试需要延迟
		if attempt > 0 {
			delay := policy.calculateDelay(attempt - 1)
			log.Printf("任务 %s (ID: %d) 第 %d 次重试，延迟 %v", 
				task.Name, task.ID, attempt, delay)
			
			select {
			case <-ctx.Done():
				return fmt.Errorf("任务被取消: %w", ctx.Err())
			case <-time.After(delay):
				// 延迟结束，继续执行
			}
		}
		
		// 执行任务
		log.Printf("任务 %s (ID: %d) 开始执行 (尝试 %d/%d)", 
			task.Name, task.ID, attempt+1, policy.MaxRetries+1)
		
		err := w.executeTaskOnce(ctx, task, attempt)
		
		if err == nil {
			// 执行成功
			if attempt > 0 {
				log.Printf("任务 %s (ID: %d) 重试成功", task.Name, task.ID)
			}
			return nil
		}
		
		lastErr = err
		
		// 判断是否应该重试
		if !policy.shouldRetry(attempt, err) {
			log.Printf("任务 %s (ID: %d) 不应该重试: %v", task.Name, task.ID, err)
			break
		}
		
		log.Printf("任务 %s (ID: %d) 执行失败: %v", task.Name, task.ID, err)
	}
	
	// 所有重试都失败
	return fmt.Errorf("任务执行失败，已重试 %d 次: %w", policy.MaxRetries, lastErr)
}

// executeTaskOnce 执行一次任务（不包含重试逻辑）
func (w *Worker) executeTaskOnce(ctx context.Context, task *models.CollectionTask, retryCount int) error {
	startTime := time.Now()

	// 添加任务级别的超时控制
	timeout := time.Duration(task.ExecutionTimeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second // 默认 5 分钟
	}
	
	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 创建执行记录
	execution := &models.TaskExecution{
		TaskID:     task.ID,
		WorkerPod:  w.podName,
		Status:     "running",
		StartTime:  startTime,
		RetryCount: retryCount,
	}

	execID, err := w.db.CreateExecution(taskCtx, execution)
	if err != nil {
		return fmt.Errorf("创建执行记录失败: %w", err)
	}
	execution.ID = execID

	log.Printf("开始执行任务: %s (执行ID: %d, 重试次数: %d, 超时: %v)", 
		task.Name, execID, retryCount, timeout)

	// 解析任务配置
	taskConfig, err := w.parseTaskConfig(task.Config)
	if err != nil {
		w.finishExecution(ctx, execution, "failed", 0, fmt.Sprintf("解析任务配置失败: %v", err))
		return fmt.Errorf("解析任务配置失败: %w", err)
	}

	// 1. 数据采集
	collectedData, err := w.collectData(taskCtx, &taskConfig.DataSource)
	if err != nil {
		w.finishExecution(taskCtx, execution, "failed", 0, fmt.Sprintf("数据采集失败: %v", err))
		return fmt.Errorf("数据采集失败: %w", err)
	}

	// 2. 数据处理
	processedData, err := w.processData(collectedData, &taskConfig.Processor)
	if err != nil {
		w.finishExecution(taskCtx, execution, "failed", len(collectedData), fmt.Sprintf("数据处理失败: %v", err))
		return fmt.Errorf("数据处理失败: %w", err)
	}

	// 3. 数据存储
	if err := w.storeData(taskCtx, &taskConfig.Storage, processedData); err != nil {
		w.finishExecution(taskCtx, execution, "failed", len(processedData), fmt.Sprintf("数据存储失败: %v", err))
		return fmt.Errorf("数据存储失败: %w", err)
	}

	// 4. 更新下次执行时间
	if err := w.updateNextRunTime(taskCtx, task); err != nil {
		log.Printf("更新下次执行时间失败: %v", err)
	}

	// 完成执行
	w.finishExecution(taskCtx, execution, "success", len(processedData), "")
	
	log.Printf("任务执行完成: %s, 耗时: %v, 数据量: %d", task.Name, time.Since(startTime), len(processedData))
	
	return nil
}
