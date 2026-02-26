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

	return true
}

// executeWithRetry 带重试的任务执行（只创建一条执行记录）
func (w *Worker) executeWithRetry(ctx context.Context, task *models.CollectionTask) error {
	policy := DefaultRetryPolicy()

	// 如果任务配置了最大重试次数，使用任务配置
	if task.MaxRetries > 0 {
		policy.MaxRetries = task.MaxRetries
	}

	// 创建唯一的执行记录
	startTime := time.Now()
	execID, err := w.db.CreateExecution(task.ID, w.podName)
	if err != nil {
		return fmt.Errorf("创建执行记录失败: %w", err)
	}

	execution := &models.TaskExecution{
		ID:        execID,
		TaskID:    task.ID,
		WorkerPod: w.podName,
		Status:    "running",
		StartTime: startTime,
	}

	log.Printf("创建执行记录: 任务=%s, 执行ID=%d", task.Name, execID)

	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// 第一次尝试不延迟，后续尝试需要延迟
		if attempt > 0 {
			delay := policy.calculateDelay(attempt - 1)
			log.Printf("任务 %s (ID: %d) 第 %d 次重试，延迟 %v",
				task.Name, task.ID, attempt, delay)

			select {
			case <-ctx.Done():
				w.finishExecution(ctx, execution, "failed", 0, "任务被取消")
				return fmt.Errorf("任务被取消: %w", ctx.Err())
			case <-time.After(delay):
				// 延迟结束，继续执行
			}
		}

		// 执行任务
		log.Printf("任务 %s (ID: %d) 开始执行 (尝试 %d/%d, 执行ID: %d)",
			task.Name, task.ID, attempt+1, policy.MaxRetries+1, execID)

		recordCount, err := w.executeTaskOnce(ctx, task, execution, attempt)

		if err == nil {
			// 执行成功
			w.finishExecution(ctx, execution, "success", recordCount, "")
			log.Printf("任务执行完成: %s, 耗时: %v, 数据量: %d", task.Name, time.Since(startTime), recordCount)
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

	// 所有重试都失败，标记执行记录为失败
	w.finishExecution(ctx, execution, "failed", 0, fmt.Sprintf("重试 %d 次后失败: %v", policy.MaxRetries, lastErr))
	return fmt.Errorf("任务执行失败，已重试 %d 次: %w", policy.MaxRetries, lastErr)
}

// executeTaskOnce 执行一次任务（不创建执行记录，不管理执行状态）
// 返回采集的记录数和错误
func (w *Worker) executeTaskOnce(ctx context.Context, task *models.CollectionTask, execution *models.TaskExecution, retryCount int) (int, error) {
	// 添加任务级别的超时控制
	timeout := time.Duration(task.ExecutionTimeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second // 默认 5 分钟
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 更新执行记录的重试次数
	execution.RetryCount = retryCount

	// 解析任务配置：优先使用 task.Config，为空时从数据源自动构建
	taskConfig, err := w.resolveTaskConfig(task)
	if err != nil {
		return 0, fmt.Errorf("解析任务配置失败: %w", err)
	}

	// 1. 数据采集
	collectedData, err := w.collectData(taskCtx, &taskConfig.DataSource)
	if err != nil {
		return 0, fmt.Errorf("数据采集失败: %w", err)
	}

	// 2. 数据处理
	processedData, err := w.processData(collectedData, &taskConfig.Processor)
	if err != nil {
		return len(collectedData), fmt.Errorf("数据处理失败: %w", err)
	}

	// 3. 数据存储
	if err := w.storeData(taskCtx, &taskConfig.Storage, processedData); err != nil {
		return len(processedData), fmt.Errorf("数据存储失败: %w", err)
	}

	return len(processedData), nil
}
