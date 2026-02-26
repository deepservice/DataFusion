package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
	"github.com/datafusion/worker/internal/models"
)

// PostgresStorage PostgreSQL 存储
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage 创建 PostgreSQL 存储
func NewPostgresStorage(host string, port int, user, password, database, sslMode string) (*PostgresStorage, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

// Type 返回存储类型
func (p *PostgresStorage) Type() string {
	return "postgresql"
}

// Store 存储数据
func (p *PostgresStorage) Store(ctx context.Context, config *models.StorageConfig, data []map[string]interface{}) error {
	if len(data) == 0 {
		log.Println("没有数据需要存储")
		return nil
	}

	log.Printf("开始存储数据到 PostgreSQL，表: %s，数据量: %d", config.Table, len(data))

	// 自动创建表（如果不存在）
	if err := p.ensureTable(ctx, config.Table, data[0]); err != nil {
		return fmt.Errorf("自动创建表失败: %w", err)
	}

	// 构建插入语句
	fields := make([]string, 0)
	for field := range data[0] {
		// 应用字段映射
		if mappedField, ok := config.Mapping[field]; ok {
			fields = append(fields, mappedField)
		} else {
			fields = append(fields, field)
		}
	}

	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// 使用 ON CONFLICT DO NOTHING 来忽略主键冲突
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		config.Table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	// 批量插入
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("准备语句失败: %w", err)
	}
	defer stmt.Close()

	successCount := 0
	duplicateCount := 0
	errorCount := 0
	
	for _, record := range data {
		values := make([]interface{}, len(fields))
		for i, field := range fields {
			// 反向查找原始字段名
			originalField := field
			for k, v := range config.Mapping {
				if v == field {
					originalField = k
					break
				}
			}
			values[i] = record[originalField]
		}

		result, execErr := stmt.ExecContext(ctx, values...)
		if execErr != nil {
			log.Printf("插入数据失败: %v, 数据: %v", execErr, record)
			errorCount++
			continue
		}
		
		// 检查是否实际插入了数据
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			successCount++
		} else {
			// ON CONFLICT DO NOTHING 导致没有插入（数据重复）
			duplicateCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	log.Printf("数据存储完成，成功: %d 条，重复: %d 条，失败: %d 条", successCount, duplicateCount, errorCount)
	
	// 只要有数据成功插入或者是重复数据，就认为是成功的
	// 只有全部失败才返回错误
	if successCount == 0 && duplicateCount == 0 && errorCount > 0 {
		return fmt.Errorf("所有数据插入失败")
	}
	
	return nil
}

// ensureTable 确保目标表存在，不存在则根据数据字段自动创建
func (p *PostgresStorage) ensureTable(ctx context.Context, table string, sample map[string]interface{}) error {
	// 检查表是否存在
	var exists bool
	err := p.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
	if err != nil {
		return fmt.Errorf("检查表是否存在失败: %w", err)
	}
	if exists {
		return nil
	}

	// 构建 CREATE TABLE 语句，所有字段用 TEXT 类型
	cols := []string{"id SERIAL PRIMARY KEY", "collected_at TIMESTAMP DEFAULT NOW()"}
	for field := range sample {
		cols = append(cols, fmt.Sprintf("%s TEXT", field))
	}

	query := fmt.Sprintf("CREATE TABLE %s (%s)", table, strings.Join(cols, ", "))
	log.Printf("自动创建表: %s", query)

	if _, err := p.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}
	return nil
}

// Close 关闭数据库连接
func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
