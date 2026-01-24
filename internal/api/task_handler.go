package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewTaskHandler(db *sql.DB, log *logger.Logger) *TaskHandler {
	return &TaskHandler{db: db, log: log}
}

type Task struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"` // web-rpa, api, database
	DataSourceID    int64     `json:"data_source_id"`
	Cron            string    `json:"cron"`
	Status          string    `json:"status"` // enabled, disabled
	Replicas        int       `json:"replicas"`
	ExecutionTimeout int      `json:"execution_timeout"`
	MaxRetries      int       `json:"max_retries"`
	Config          string    `json:"config"` // JSON配置
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// List 获取任务列表
func (h *TaskHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	offset := (page - 1) * pageSize

	query := `SELECT id, name, description, type, data_source_id, cron, status, 
	          replicas, execution_timeout, max_retries, config, created_at, updated_at 
	          FROM collection_tasks WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = $1"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
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
			&task.Cron, &task.Status, &task.Replicas, &task.ExecutionTimeout, &task.MaxRetries,
			&task.Config, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			h.log.Error("扫描任务数据失败", zap.Error(err))
			continue
		}
		tasks = append(tasks, task)
	}

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM collection_tasks WHERE 1=1"
	if status != "" {
		countQuery += " AND status = '" + status + "'"
	}
	h.db.QueryRow(countQuery).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  tasks,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// Get 获取单个任务
func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var task Task
	err := h.db.QueryRow(`SELECT id, name, description, type, data_source_id, cron, status, 
	                      replicas, execution_timeout, max_retries, config, created_at, updated_at 
	                      FROM collection_tasks WHERE id = $1`, id).
		Scan(&task.ID, &task.Name, &task.Description, &task.Type, &task.DataSourceID,
			&task.Cron, &task.Status, &task.Replicas, &task.ExecutionTimeout, &task.MaxRetries,
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

	err := h.db.QueryRow(`INSERT INTO collection_tasks 
	    (name, description, type, data_source_id, cron, status, replicas, execution_timeout, max_retries, config) 
	    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		task.Name, task.Description, task.Type, task.DataSourceID, task.Cron, task.Status,
		task.Replicas, task.ExecutionTimeout, task.MaxRetries, task.Config).Scan(&task.ID)

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

	result, err := h.db.Exec(`UPDATE collection_tasks SET 
	    name=$1, description=$2, type=$3, data_source_id=$4, cron=$5, status=$6, 
	    replicas=$7, execution_timeout=$8, max_retries=$9, config=$10, updated_at=NOW() 
	    WHERE id=$11`,
		task.Name, task.Description, task.Type, task.DataSourceID, task.Cron, task.Status,
		task.Replicas, task.ExecutionTimeout, task.MaxRetries, task.Config, id)

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

// Run 手动运行任务
func (h *TaskHandler) Run(c *gin.Context) {
	id := c.Param("id")

	// 更新next_run_time为当前时间，让Worker立即执行
	_, err := h.db.Exec("UPDATE collection_tasks SET next_run_time=NOW() WHERE id=$1", id)
	if err != nil {
		h.log.Error("触发任务失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "触发失败"})
		return
	}

	h.log.Info("手动触发任务", zap.String("task_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "任务已触发"})
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
