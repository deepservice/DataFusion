package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/models"
)

type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB 创建 PostgreSQL 连接
func NewPostgresDB(cfg config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

// GetPendingTasks 获取待执行的任务
func (p *PostgresDB) GetPendingTasks(ctx context.Context, workerType string) ([]*models.CollectionTask, error) {
	query := `
		SELECT id, name, type, status, cron, next_run_time, replicas, 
		       execution_timeout, max_retries, config, created_at, updated_at
		FROM collection_tasks
		WHERE next_run_time <= $1
		  AND status = 'enabled'
		  AND type = $2
		  AND (SELECT COUNT(*) FROM task_executions 
		       WHERE task_id = collection_tasks.id 
		         AND status = 'running') < replicas
		ORDER BY next_run_time ASC
		LIMIT 10
	`

	rows, err := p.db.QueryContext(ctx, query, time.Now(), workerType)
	if err != nil {
		return nil, fmt.Errorf("查询待执行任务失败: %w", err)
	}
	defer rows.Close()

	var tasks []*models.CollectionTask
	for rows.Next() {
		var task models.CollectionTask
		if err := rows.Scan(
			&task.ID, &task.Name, &task.Type, &task.Status, &task.Cron,
			&task.NextRunTime, &task.Replicas, &task.ExecutionTimeout,
			&task.MaxRetries, &task.Config, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描任务数据失败: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// TryLockTask 尝试获取任务锁
func (p *PostgresDB) TryLockTask(ctx context.Context, taskID int64) (bool, error) {
	var locked bool
	err := p.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", taskID).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("获取任务锁失败: %w", err)
	}
	return locked, nil
}

// UnlockTask 释放任务锁
func (p *PostgresDB) UnlockTask(ctx context.Context, taskID int64) error {
	_, err := p.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", taskID)
	if err != nil {
		return fmt.Errorf("释放任务锁失败: %w", err)
	}
	return nil
}

// CreateExecution 创建任务执行记录
func (p *PostgresDB) CreateExecution(ctx context.Context, exec *models.TaskExecution) (int64, error) {
	query := `
		INSERT INTO task_executions (task_id, worker_pod, status, start_time, retry_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64
	err := p.db.QueryRowContext(ctx, query,
		exec.TaskID, exec.WorkerPod, exec.Status, exec.StartTime, exec.RetryCount,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("创建执行记录失败: %w", err)
	}

	return id, nil
}

// UpdateExecution 更新任务执行记录
func (p *PostgresDB) UpdateExecution(ctx context.Context, exec *models.TaskExecution) error {
	query := `
		UPDATE task_executions
		SET status = $1, end_time = $2, records_collected = $3, error_message = $4
		WHERE id = $5
	`

	_, err := p.db.ExecContext(ctx, query,
		exec.Status, exec.EndTime, exec.RecordsCollected, exec.ErrorMessage, exec.ID,
	)

	if err != nil {
		return fmt.Errorf("更新执行记录失败: %w", err)
	}

	return nil
}

// UpdateTaskNextRunTime 更新任务下次执行时间
func (p *PostgresDB) UpdateTaskNextRunTime(ctx context.Context, taskID int64, nextRunTime time.Time) error {
	query := `UPDATE collection_tasks SET next_run_time = $1, updated_at = $2 WHERE id = $3`
	_, err := p.db.ExecContext(ctx, query, nextRunTime, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("更新任务下次执行时间失败: %w", err)
	}
	return nil
}

// ParseTaskConfig 解析任务配置
func ParseTaskConfig(configJSON string) (*models.TaskConfig, error) {
	var config models.TaskConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析任务配置失败: %w", err)
	}
	return &config, nil
}

// Close 关闭数据库连接
func (p *PostgresDB) Close() error {
	return p.db.Close()
}
