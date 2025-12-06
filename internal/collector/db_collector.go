package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/datafusion/worker/internal/models"
)

// DBCollector 数据库采集器
type DBCollector struct {
	timeout time.Duration
}

// NewDBCollector 创建数据库采集器
func NewDBCollector(timeout int) *DBCollector {
	return &DBCollector{
		timeout: time.Duration(timeout) * time.Second,
	}
}

// Type 返回采集器类型
func (d *DBCollector) Type() string {
	return "database"
}

// Collect 执行数据采集
func (d *DBCollector) Collect(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error) {
	log.Printf("开始数据库采集: %s", config.URL)

	// 解析数据库配置
	dbConfig := config.DBConfig
	if dbConfig == nil {
		return nil, fmt.Errorf("数据库配置为空")
	}

	// 建立数据库连接
	db, err := d.connectDB(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	// 设置连接超时
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// 执行查询
	rows, err := db.QueryContext(ctx, dbConfig.Query)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	// 解析结果
	results, err := d.parseRows(rows)
	if err != nil {
		return nil, fmt.Errorf("解析查询结果失败: %w", err)
	}

	log.Printf("数据库采集完成，获取到 %d 条数据", len(results))
	return results, nil
}

// connectDB 建立数据库连接
func (d *DBCollector) connectDB(config *models.DBConfig) (*sql.DB, error) {
	var dsn string
	var driverName string

	// 根据端口判断数据库类型
	switch config.Port {
	case 3306:
		// MySQL
		driverName = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
			config.User, config.Password, config.Host, config.Port, config.Database)
	case 5432:
		// PostgreSQL
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, config.Port, config.User, config.Password, config.Database)
	default:
		return nil, fmt.Errorf("不支持的数据库端口: %d", config.Port)
	}

	// 打开数据库连接
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// parseRows 解析查询结果
func (d *DBCollector) parseRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 获取列类型
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 构建结果映射
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			
			// 处理 NULL 值
			if val == nil {
				row[col] = nil
				continue
			}

			// 类型转换
			row[col] = d.convertValue(val, columnTypes[i])
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// convertValue 转换数据库值为合适的 Go 类型
func (d *DBCollector) convertValue(val interface{}, colType *sql.ColumnType) interface{} {
	// 处理字节数组
	if b, ok := val.([]byte); ok {
		return string(b)
	}

	// 处理时间类型
	if t, ok := val.(time.Time); ok {
		return t.Format(time.RFC3339)
	}

	return val
}
