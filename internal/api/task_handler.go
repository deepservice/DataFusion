package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskHandler struct {
	db     *sql.DB
	dataDB *sql.DB // datafusion_data 数据库连接，用于数据预览
	log    *logger.Logger
}

func NewTaskHandler(db *sql.DB, dataDB *sql.DB, log *logger.Logger) *TaskHandler {
	return &TaskHandler{db: db, dataDB: dataDB, log: log}
}

type Task struct {
	ID               int64           `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description"`
	Type             string          `json:"type"` // web-rpa, api, database
	DataSourceID     int64           `json:"data_source_id"`
	Cron             *string         `json:"cron"`
	NextRunTime      *time.Time      `json:"next_run_time"`
	Status           string          `json:"status"` // enabled, disabled
	Replicas         int             `json:"replicas"`
	ExecutionTimeout int             `json:"execution_timeout"`
	MaxRetries       int             `json:"max_retries"`
	Config           *json.RawMessage `json:"config"` // JSON配置
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// List 获取任务列表
func (h *TaskHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	// 支持 page_size 和 limit 两种参数名
	pageSizeStr := c.DefaultQuery("page_size", "")
	if pageSizeStr == "" {
		pageSizeStr = c.DefaultQuery("limit", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 20
	}
	status := c.Query("status")
	search := c.Query("search")

	offset := (page - 1) * pageSize

	query := `SELECT id, name, description, type, data_source_id, cron, next_run_time, status,
	          replicas, execution_timeout, max_retries, config, created_at, updated_at
	          FROM collection_tasks WHERE 1=1`
	countQuery := "SELECT COUNT(*) FROM collection_tasks WHERE 1=1"
	args := []interface{}{}
	countArgs := []interface{}{}
	argIdx := 1

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		countQuery += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		countArgs = append(countArgs, status)
		argIdx++
	}

	if search != "" {
		query += " AND (name ILIKE $" + strconv.Itoa(argIdx) + " OR description ILIKE $" + strconv.Itoa(argIdx) + ")"
		countQuery += " AND (name ILIKE $" + strconv.Itoa(argIdx) + " OR description ILIKE $" + strconv.Itoa(argIdx) + ")"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		countArgs = append(countArgs, searchPattern)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.log.Error("查询任务列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.Type, &task.DataSourceID,
			&task.Cron, &task.NextRunTime, &task.Status, &task.Replicas, &task.ExecutionTimeout, &task.MaxRetries,
			&task.Config, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			h.log.Error("扫描任务数据失败", zap.Error(err))
			continue
		}
		tasks = append(tasks, task)
	}

	// 获取总数
	var total int
	h.db.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"items": tasks,
		"pagination": gin.H{
			"page":  page,
			"limit": pageSize,
			"total": total,
		},
	})
}

// Get 获取单个任务
func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var task Task
	err := h.db.QueryRow(`SELECT id, name, description, type, data_source_id, cron, next_run_time, status,
	                      replicas, execution_timeout, max_retries, config, created_at, updated_at
	                      FROM collection_tasks WHERE id = $1`, id).
		Scan(&task.ID, &task.Name, &task.Description, &task.Type, &task.DataSourceID,
			&task.Cron, &task.NextRunTime, &task.Status, &task.Replicas, &task.ExecutionTimeout, &task.MaxRetries,
			&task.Config, &task.CreatedAt, &task.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// Create 创建任务
func (h *TaskHandler) Create(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if task.Status == "" {
		task.Status = "enabled"
	}
	if task.Replicas == 0 {
		task.Replicas = 1
	}
	if task.ExecutionTimeout == 0 {
		task.ExecutionTimeout = 3600
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	// 处理空config的情况
	var configData interface{}
	if task.Config == nil || len(*task.Config) == 0 || string(*task.Config) == "null" {
		configData = nil
	} else {
		configData = *task.Config
	}

	err := h.db.QueryRow(`INSERT INTO collection_tasks
	    (name, description, type, data_source_id, cron, status, replicas, execution_timeout, max_retries, config)
	    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		task.Name, task.Description, task.Type, task.DataSourceID, task.Cron, task.Status,
		task.Replicas, task.ExecutionTimeout, task.MaxRetries, configData).Scan(&task.ID)

	if err != nil {
		h.log.Error("创建任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	h.log.Info("创建任务成功", zap.Int64("task_id", task.ID), zap.String("name", task.Name))
	c.JSON(http.StatusCreated, task)
}

// Update 更新任务
func (h *TaskHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理空config的情况
	var configData interface{}
	if task.Config == nil || len(*task.Config) == 0 || string(*task.Config) == "null" {
		configData = nil
	} else {
		configData = *task.Config
	}

	result, err := h.db.Exec(`UPDATE collection_tasks SET
	    name=$1, description=$2, type=$3, data_source_id=$4, cron=$5, status=$6,
	    replicas=$7, execution_timeout=$8, max_retries=$9, config=$10, updated_at=NOW()
	    WHERE id=$11`,
		task.Name, task.Description, task.Type, task.DataSourceID, task.Cron, task.Status,
		task.Replicas, task.ExecutionTimeout, task.MaxRetries, configData, id)

	if err != nil {
		h.log.Error("更新任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	h.log.Info("更新任务成功", zap.String("task_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Delete 删除任务
func (h *TaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("DELETE FROM collection_tasks WHERE id=$1", id)
	if err != nil {
		h.log.Error("删除任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	h.log.Info("删除任务成功", zap.String("task_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Run 启用并调度任务
func (h *TaskHandler) Run(c *gin.Context) {
	id := c.Param("id")

	// 启用任务并设置next_run_time为当前时间，让Worker立即调度
	_, err := h.db.Exec("UPDATE collection_tasks SET status='enabled', next_run_time=NOW() WHERE id=$1", id)
	if err != nil {
		h.log.Error("启用任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启用失败"})
		return
	}

	h.log.Info("启用并调度任务", zap.String("task_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "任务已启用并调度"})
}

// Execute 手动触发任务执行（仅设置触发条件，由 Worker 创建执行记录）
func (h *TaskHandler) Execute(c *gin.Context) {
	id := c.Param("id")

	// 验证任务存在
	var taskName string
	err := h.db.QueryRow("SELECT name FROM collection_tasks WHERE id=$1", id).Scan(&taskName)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 启用任务并设置 next_run_time=NOW()，让 Worker 在下次轮询时拾取执行
	_, err = h.db.Exec("UPDATE collection_tasks SET status='enabled', next_run_time=NOW() WHERE id=$1", id)
	if err != nil {
		h.log.Error("触发任务执行失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "触发执行失败"})
		return
	}

	h.log.Info("手动触发任务执行", zap.String("task_id", id), zap.String("task_name", taskName))
	c.JSON(http.StatusOK, gin.H{"message": "任务执行已触发"})
}

// PreviewData 预览任务采集的数据
func (h *TaskHandler) PreviewData(c *gin.Context) {
	id := c.Param("id")

	// 获取任务类型，用于拼接表名
	var taskType string
	err := h.db.QueryRow("SELECT type FROM collection_tasks WHERE id=$1", id).Scan(&taskType)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if h.dataDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}

	// 拼接表名: collected_{type}_{id}，type 中 - 替换为 _
	tableName := fmt.Sprintf("collected_%s_%s", strings.ReplaceAll(taskType, "-", "_"), id)

	// 检查表是否存在
	var exists bool
	err = h.dataDB.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tableName).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusOK, gin.H{
			"items":   []interface{}{},
			"columns": []string{},
			"pagination": gin.H{
				"page":  1,
				"limit": 20,
				"total": 0,
			},
		})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 查询总数
	var total int
	err = h.dataDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&total)
	if err != nil {
		h.log.Error("查询数据总数失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 查询数据
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY collected_at DESC LIMIT $1 OFFSET $2", tableName)
	rows, err := h.dataDB.Query(query, limit, offset)
	if err != nil {
		h.log.Error("查询数据失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据失败"})
		return
	}
	defer rows.Close()

	// 获取列名
	columnNames, err := rows.Columns()
	if err != nil {
		h.log.Error("获取列名失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 读取数据
	var items []map[string]interface{}
	for rows.Next() {
		// 为每行创建扫描目标
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		item := make(map[string]interface{})
		for i, col := range columnNames {
			val := values[i]
			// 将 []byte 转为 string
			if b, ok := val.([]byte); ok {
				s := string(b)
				// 截断超长内容用于预览
				if len(s) > 500 {
					item[col] = s[:500] + "..."
				} else {
					item[col] = s
				}
			} else if t, ok := val.(time.Time); ok {
				item[col] = t.Format("2006-01-02 15:04:05")
			} else {
				item[col] = val
			}
		}
		items = append(items, item)
	}

	if items == nil {
		items = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":   items,
		"columns": columnNames,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// Stop 停止任务
func (h *TaskHandler) Stop(c *gin.Context) {
	id := c.Param("id")

	_, err := h.db.Exec("UPDATE collection_tasks SET status='disabled' WHERE id=$1", id)
	if err != nil {
		h.log.Error("停止任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "停止失败"})
		return
	}

	h.log.Info("停止任务", zap.String("task_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "任务已停止"})
}
