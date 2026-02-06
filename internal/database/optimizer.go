package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/datafusion/worker/internal/cache"
	"github.com/datafusion/worker/internal/logger"
	"go.uber.org/zap"
)

// QueryOptimizer 数据库查询优化器
type QueryOptimizer struct {
	db    *sql.DB
	cache cache.Cache
	log   *logger.Logger
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *sql.DB, cache cache.Cache, log *logger.Logger) *QueryOptimizer {
	return &QueryOptimizer{
		db:    db,
		cache: cache,
		log:   log,
	}
}

// CachedQuery 带缓存的查询
func (qo *QueryOptimizer) CachedQuery(ctx context.Context, cacheKey string, cacheTTL time.Duration, query string, args ...interface{}) (*sql.Rows, error) {
	// 尝试从缓存获取
	var cachedResult []map[string]interface{}
	err := qo.cache.Get(cacheKey, &cachedResult)
	if err == nil {
		qo.log.Debug("从缓存获取查询结果", zap.String("cache_key", cacheKey))
		return qo.rowsFromCache(cachedResult), nil
	}

	// 缓存未命中，执行数据库查询
	start := time.Now()
	rows, err := qo.db.QueryContext(ctx, query, args...)
	if err != nil {
		qo.log.WithError(err).Error("数据库查询失败", zap.String("query", query))
		return nil, err
	}

	// 记录查询时间
	duration := time.Since(start)
	qo.log.Debug("数据库查询完成", zap.Duration("duration", duration), zap.String("query", qo.sanitizeQuery(query)))

	// 如果查询时间较长，记录慢查询
	if duration > 1*time.Second {
		qo.log.Warn("慢查询检测", zap.Duration("duration", duration), zap.String("query", qo.sanitizeQuery(query)))
	}

	// 将结果缓存（异步）
	go func() {
		defer rows.Close()
		result, err := qo.rowsToSlice(rows)
		if err != nil {
			qo.log.WithError(err).Error("转换查询结果失败")
			return
		}

		err = qo.cache.Set(cacheKey, result, cacheTTL)
		if err != nil {
			qo.log.WithError(err).Error("缓存查询结果失败", zap.String("cache_key", cacheKey))
		} else {
			qo.log.Debug("查询结果已缓存", zap.String("cache_key", cacheKey), zap.Duration("ttl", cacheTTL))
		}
	}()

	// 重新执行查询返回给调用者（因为上面的rows已经被消费了）
	return qo.db.QueryContext(ctx, query, args...)
}

// CachedQueryRow 带缓存的单行查询
func (qo *QueryOptimizer) CachedQueryRow(ctx context.Context, cacheKey string, cacheTTL time.Duration, query string, args ...interface{}) *sql.Row {
	// 尝试从缓存获取
	var cachedResult map[string]interface{}
	err := qo.cache.Get(cacheKey, &cachedResult)
	if err == nil {
		qo.log.Debug("从缓存获取单行查询结果", zap.String("cache_key", cacheKey))
		return qo.rowFromCache(cachedResult)
	}

	// 缓存未命中，执行数据库查询
	start := time.Now()
	row := qo.db.QueryRowContext(ctx, query, args...)

	// 记录查询时间
	duration := time.Since(start)
	qo.log.Debug("数据库单行查询完成", zap.Duration("duration", duration), zap.String("query", qo.sanitizeQuery(query)))

	// 异步缓存结果
	go func() {
		// 这里需要重新查询来获取结果进行缓存
		rows, err := qo.db.QueryContext(ctx, query, args...)
		if err != nil {
			return
		}
		defer rows.Close()

		if rows.Next() {
			result, err := qo.rowToMap(rows)
			if err != nil {
				return
			}

			err = qo.cache.Set(cacheKey, result, cacheTTL)
			if err != nil {
				qo.log.WithError(err).Error("缓存单行查询结果失败", zap.String("cache_key", cacheKey))
			}
		}
	}()

	return row
}

// BatchInsert 批量插入优化
func (qo *QueryOptimizer) BatchInsert(ctx context.Context, table string, columns []string, values [][]interface{}, batchSize int) error {
	if len(values) == 0 {
		return nil
	}

	// 构建批量插入SQL
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// 分批处理
	for i := 0; i < len(values); i += batchSize {
		end := i + batchSize
		if end > len(values) {
			end = len(values)
		}

		batch := values[i:end]

		// 构建当前批次的SQL
		valueClauses := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*len(columns))

		for j, row := range batch {
			// 更新占位符索引
			batchPlaceholders := make([]string, len(columns))
			for k := range batchPlaceholders {
				batchPlaceholders[k] = fmt.Sprintf("$%d", j*len(columns)+k+1)
			}
			valueClauses[j] = "(" + strings.Join(batchPlaceholders, ", ") + ")"
			args = append(args, row...)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			table,
			strings.Join(columns, ", "),
			strings.Join(valueClauses, ", "))

		start := time.Now()
		_, err := qo.db.ExecContext(ctx, query, args...)
		if err != nil {
			qo.log.WithError(err).Error("批量插入失败", zap.String("table", table), zap.Int("batch_size", len(batch)))
			return err
		}

		duration := time.Since(start)
		qo.log.Debug("批量插入完成", zap.String("table", table), zap.Int("batch_size", len(batch)), zap.Duration("duration", duration))
	}

	return nil
}

// PreparedStatement 预编译语句缓存
type PreparedStatement struct {
	stmt *sql.Stmt
	uses int64
}

var preparedStmts = make(map[string]*PreparedStatement)

// GetPreparedStatement 获取预编译语句
func (qo *QueryOptimizer) GetPreparedStatement(query string) (*sql.Stmt, error) {
	if ps, exists := preparedStmts[query]; exists {
		ps.uses++
		return ps.stmt, nil
	}

	stmt, err := qo.db.Prepare(query)
	if err != nil {
		return nil, err
	}

	preparedStmts[query] = &PreparedStatement{
		stmt: stmt,
		uses: 1,
	}

	qo.log.Debug("创建预编译语句", zap.String("query", qo.sanitizeQuery(query)))
	return stmt, nil
}

// AnalyzeSlowQueries 分析慢查询
func (qo *QueryOptimizer) AnalyzeSlowQueries(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		SELECT query, calls, total_time, mean_time, rows
		FROM pg_stat_statements 
		WHERE mean_time > 1000 
		ORDER BY mean_time DESC 
		LIMIT 20
	`

	rows, err := qo.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return qo.rowsToSlice(rows)
}

// GetTableStats 获取表统计信息
func (qo *QueryOptimizer) GetTableStats(ctx context.Context, tableName string) (map[string]interface{}, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			n_tup_ins as inserts,
			n_tup_upd as updates,
			n_tup_del as deletes,
			n_live_tup as live_tuples,
			n_dead_tup as dead_tuples,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_user_tables 
		WHERE tablename = $1
	`

	row := qo.db.QueryRowContext(ctx, query, tableName)
	return qo.scanRowToMap(row)
}

// OptimizeTable 优化表（VACUUM ANALYZE）
func (qo *QueryOptimizer) OptimizeTable(ctx context.Context, tableName string) error {
	query := fmt.Sprintf("VACUUM ANALYZE %s", tableName)

	start := time.Now()
	_, err := qo.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	duration := time.Since(start)
	qo.log.Info("表优化完成", zap.String("table", tableName), zap.Duration("duration", duration))
	return nil
}

// 辅助方法

// rowsToSlice 将查询结果转换为切片
func (qo *QueryOptimizer) rowsToSlice(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

// rowToMap 将单行结果转换为map
func (qo *QueryOptimizer) rowToMap(rows *sql.Rows) (map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	err = rows.Scan(valuePtrs...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	return result, nil
}

// scanRowToMap 将sql.Row扫描到map
func (qo *QueryOptimizer) scanRowToMap(row *sql.Row) (map[string]interface{}, error) {
	// 这里需要根据具体的查询结构来实现
	// 简化实现，实际使用时需要根据查询字段调整
	var schemaname, tablename sql.NullString
	var inserts, updates, deletes, liveTuples, deadTuples sql.NullInt64
	var lastVacuum, lastAutovacuum, lastAnalyze, lastAutoanalyze sql.NullTime

	err := row.Scan(&schemaname, &tablename, &inserts, &updates, &deletes,
		&liveTuples, &deadTuples, &lastVacuum, &lastAutovacuum,
		&lastAnalyze, &lastAutoanalyze)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"schemaname":       schemaname.String,
		"tablename":        tablename.String,
		"inserts":          inserts.Int64,
		"updates":          updates.Int64,
		"deletes":          deletes.Int64,
		"live_tuples":      liveTuples.Int64,
		"dead_tuples":      deadTuples.Int64,
		"last_vacuum":      lastVacuum.Time,
		"last_autovacuum":  lastAutovacuum.Time,
		"last_analyze":     lastAnalyze.Time,
		"last_autoanalyze": lastAutoanalyze.Time,
	}

	return result, nil
}

// rowsFromCache 从缓存数据创建虚拟Rows（简化实现）
func (qo *QueryOptimizer) rowsFromCache(data []map[string]interface{}) *sql.Rows {
	// 这是一个简化的实现，实际项目中可能需要更复杂的处理
	// 或者改变缓存策略，直接缓存处理后的结果而不是原始查询结果
	return nil
}

// rowFromCache 从缓存数据创建虚拟Row（简化实现）
func (qo *QueryOptimizer) rowFromCache(data map[string]interface{}) *sql.Row {
	// 这是一个简化的实现，实际项目中可能需要更复杂的处理
	return nil
}

// sanitizeQuery 清理查询字符串用于日志
func (qo *QueryOptimizer) sanitizeQuery(query string) string {
	// 移除多余的空白字符
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")

	// 限制长度
	if len(query) > 200 {
		query = query[:200] + "..."
	}

	return query
}
