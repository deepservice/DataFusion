package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"context"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/models"
)

// PostgresDB PostgreSQL 数据库包装器
type PostgresDB struct {
	*sql.DB
}

// NewPostgresDBFromConfig 从 DatabaseConfig 创建 PostgreSQL 连接
func NewPostgresDBFromConfig(cfg config.DatabaseConfig) (*PostgresDB, error) {
	pgConfig := config.PostgreSQLConfig{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.Database,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}
	
	db, err := NewPostgresDB(pgConfig)
	if err != nil {
		return nil, err
	}
	
	return &PostgresDB{DB: db}, nil
}

// ParseTaskConfig 解析任务配置 JSON
func ParseTaskConfig(configJSON string) (*models.TaskConfig, error) {
	var config models.TaskConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析任务配置失败: %w", err)
	}
	return &config, nil
}

// GetPendingTasks 获取待执行的任务
func (db *PostgresDB) GetPendingTasks(workerType string) ([]models.CollectionTask, error) {
	query := `
		SELECT id, name, type, status, cron, next_run_time, replicas, 
		       execution_timeout, max_retries, config, created_at, updated_at
		FROM collection_tasks 
		WHERE status = 'enabled' 
		AND type = $1 
		AND (next_run_time IS NULL OR next_run_time <= NOW())
		ORDER BY next_run_time ASC
		LIMIT 10
	`
	
	rows, err := db.Query(query, workerType)
	if err != nil {
		return nil, fmt.Errorf("查询待执行任务失败: %w", err)
	}
	defer rows.Close()
	
	var tasks []models.CollectionTask
	for rows.Next() {
		var task models.CollectionTask
		err := rows.Scan(
			&task.ID, &task.Name, &task.Type, &task.Status, &task.Cron,
			&task.NextRunTime, &task.Replicas, &task.ExecutionTimeout,
			&task.MaxRetries, &task.Config, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描任务数据失败: %w", err)
		}
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// LockTask 锁定任务
func (db *PostgresDB) LockTask(taskID int64, workerPod string) error {
	query := `
		UPDATE collection_tasks 
		SET next_run_time = NOW() + INTERVAL '1 hour'
		WHERE id = $1 AND status = 'enabled'
	`
	
	_, err := db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("锁定任务失败: %w", err)
	}
	
	return nil
}

// CreateExecution 创建执行记录
func (db *PostgresDB) CreateExecution(taskID int64, workerPod string) (int64, error) {
	query := `
		INSERT INTO task_executions (task_id, worker_pod, status, start_time)
		VALUES ($1, $2, 'running', NOW())
		RETURNING id
	`
	
	var executionID int64
	err := db.QueryRow(query, taskID, workerPod).Scan(&executionID)
	if err != nil {
		return 0, fmt.Errorf("创建执行记录失败: %w", err)
	}
	
	return executionID, nil
}

// TryLockTask 尝试锁定任务
func (db *PostgresDB) TryLockTask(taskID int64, workerPod string) (bool, error) {
	query := `
		UPDATE collection_tasks 
		SET next_run_time = NOW() + INTERVAL '1 hour'
		WHERE id = $1 AND status = 'enabled' AND (next_run_time IS NULL OR next_run_time <= NOW())
	`
	
	result, err := db.Exec(query, taskID)
	if err != nil {
		return false, fmt.Errorf("锁定任务失败: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("获取影响行数失败: %w", err)
	}
	
	return rowsAffected > 0, nil
}

// UnlockTask 解锁任务
func (db *PostgresDB) UnlockTask(taskID int64) error {
	query := `
		UPDATE collection_tasks 
		SET next_run_time = NOW()
		WHERE id = $1
	`
	
	_, err := db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("解锁任务失败: %w", err)
	}
	
	return nil
}

// UpdateTaskNextRunTime 更新任务下次执行时间
func (db *PostgresDB) UpdateTaskNextRunTime(taskID int64, nextRunTime string) error {
	query := `
		UPDATE collection_tasks 
		SET next_run_time = $1
		WHERE id = $2
	`
	
	_, err := db.Exec(query, nextRunTime, taskID)
	if err != nil {
		return fmt.Errorf("更新任务执行时间失败: %w", err)
	}
	
	return nil
}

// UpdateExecutionWithContext 更新执行记录（兼容旧接口）
func (db *PostgresDB) UpdateExecution(ctx context.Context, execution *models.TaskExecution) error {
	return db.UpdateExecutionStatus(execution.ID, execution.Status, execution.RecordsCollected, execution.ErrorMessage)
}

// UpdateExecutionStatus 更新执行记录状态
func (db *PostgresDB) UpdateExecutionStatus(executionID int64, status string, recordsCollected int, errorMessage string) error {
	query := `
		UPDATE task_executions 
		SET status = $1, end_time = NOW(), records_collected = $2, error_message = $3
		WHERE id = $4
	`
	
	_, err := db.Exec(query, status, recordsCollected, errorMessage, executionID)
	if err != nil {
		return fmt.Errorf("更新执行记录失败: %w", err)
	}
	
	return nil
}