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

type ExecutionHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewExecutionHandler(db *sql.DB, log *logger.Logger) *ExecutionHandler {
	return &ExecutionHandler{db: db, log: log}
}

type Execution struct {
	ID               int64      `json:"id"`
	TaskID           int64      `json:"task_id"`
	TaskName         *string    `json:"task_name"`
	WorkerPod        *string    `json:"worker_pod"`
	Status           string     `json:"status"` // running, success, failed
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	RecordsCollected int        `json:"records_collected"`
	ErrorMessage     *string    `json:"error_message"`
	RetryCount       int        `json:"retry_count"`
}

// List 获取执行历史列表
func (h *ExecutionHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSizeStr := c.DefaultQuery("page_size", "")
	if pageSizeStr == "" {
		pageSizeStr = c.DefaultQuery("limit", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 20
	}
	status := c.Query("status")

	offset := (page - 1) * pageSize

	query := `SELECT e.id, e.task_id, t.name as task_name, e.worker_pod, e.status,
	          e.start_time, e.end_time, e.records_collected, e.error_message, e.retry_count
	          FROM task_executions e
	          LEFT JOIN collection_tasks t ON e.task_id = t.id
	          WHERE 1=1`
	countQuery := "SELECT COUNT(*) FROM task_executions WHERE 1=1"
	args := []any{}
	countArgs := []any{}
	argIdx := 1

	if status != "" {
		query += " AND e.status = $" + strconv.Itoa(argIdx)
		countQuery += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		countArgs = append(countArgs, status)
		argIdx++
	}

	query += " ORDER BY e.start_time DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.log.Error("查询执行历史失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	executions := []Execution{}
	for rows.Next() {
		var exec Execution
		err := rows.Scan(&exec.ID, &exec.TaskID, &exec.TaskName, &exec.WorkerPod, &exec.Status,
			&exec.StartTime, &exec.EndTime, &exec.RecordsCollected, &exec.ErrorMessage, &exec.RetryCount)
		if err != nil {
			h.log.Error("扫描执行历史数据失败", zap.Error(err))
			continue
		}
		executions = append(executions, exec)
	}

	var total int
	h.db.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"items": executions,
		"pagination": gin.H{
			"page":  page,
			"limit": pageSize,
			"total": total,
		},
	})
}

// Get 获取单个执行记录
func (h *ExecutionHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var exec Execution
	err := h.db.QueryRow(`SELECT e.id, e.task_id, t.name as task_name, e.worker_pod, e.status,
	                      e.start_time, e.end_time, e.records_collected, e.error_message, e.retry_count
	                      FROM task_executions e
	                      LEFT JOIN collection_tasks t ON e.task_id = t.id
	                      WHERE e.id = $1`, id).
		Scan(&exec.ID, &exec.TaskID, &exec.TaskName, &exec.WorkerPod, &exec.Status,
			&exec.StartTime, &exec.EndTime, &exec.RecordsCollected, &exec.ErrorMessage, &exec.RetryCount)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行记录不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询执行记录失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, exec)
}

// ListByTask 获取指定任务的执行历史
func (h *ExecutionHandler) ListByTask(c *gin.Context) {
	taskID := c.Param("task_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSizeStr := c.DefaultQuery("page_size", "")
	if pageSizeStr == "" {
		pageSizeStr = c.DefaultQuery("limit", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	rows, err := h.db.Query(`SELECT e.id, e.task_id, t.name as task_name, e.worker_pod, e.status,
	                         e.start_time, e.end_time, e.records_collected, e.error_message, e.retry_count
	                         FROM task_executions e
	                         LEFT JOIN collection_tasks t ON e.task_id = t.id
	                         WHERE e.task_id = $1
	                         ORDER BY e.start_time DESC LIMIT $2 OFFSET $3`,
		taskID, pageSize, offset)
	if err != nil {
		h.log.Error("查询任务执行历史失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	executions := []Execution{}
	for rows.Next() {
		var exec Execution
		err := rows.Scan(&exec.ID, &exec.TaskID, &exec.TaskName, &exec.WorkerPod, &exec.Status,
			&exec.StartTime, &exec.EndTime, &exec.RecordsCollected, &exec.ErrorMessage, &exec.RetryCount)
		if err != nil {
			h.log.Error("扫描执行历史数据失败", zap.Error(err))
			continue
		}
		executions = append(executions, exec)
	}

	var total int
	h.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE task_id = $1", taskID).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"items": executions,
		"pagination": gin.H{
			"page":  page,
			"limit": pageSize,
			"total": total,
		},
	})
}
