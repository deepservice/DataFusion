package api

import (
	"database/sql"
	"net/http"

	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatsHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewStatsHandler(db *sql.DB, log *logger.Logger) *StatsHandler {
	return &StatsHandler{db: db, log: log}
}

// Overview 获取系统概览统计
func (h *StatsHandler) Overview(c *gin.Context) {
	var stats struct {
		TotalTasks       int `json:"total_tasks"`
		EnabledTasks     int `json:"enabled_tasks"`
		TotalExecutions  int `json:"total_executions"`
		SuccessExecutions int `json:"success_executions"`
		FailedExecutions int `json:"failed_executions"`
		RunningExecutions int `json:"running_executions"`
		TotalRecords     int `json:"total_records"`
	}

	// 任务统计
	h.db.QueryRow("SELECT COUNT(*) FROM collection_tasks").Scan(&stats.TotalTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM collection_tasks WHERE status='enabled'").Scan(&stats.EnabledTasks)

	// 执行统计
	h.db.QueryRow("SELECT COUNT(*) FROM task_executions").Scan(&stats.TotalExecutions)
	h.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE status='success'").Scan(&stats.SuccessExecutions)
	h.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE status='failed'").Scan(&stats.FailedExecutions)
	h.db.QueryRow("SELECT COUNT(*) FROM task_executions WHERE status='running'").Scan(&stats.RunningExecutions)

	// 采集记录统计
	h.db.QueryRow("SELECT COALESCE(SUM(records_collected), 0) FROM task_executions WHERE status='success'").Scan(&stats.TotalRecords)

	c.JSON(http.StatusOK, stats)
}

// TaskStats 获取任务统计信息
func (h *StatsHandler) TaskStats(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT 
			t.id,
			t.name,
			t.type,
			t.status,
			COUNT(e.id) as total_runs,
			SUM(CASE WHEN e.status='success' THEN 1 ELSE 0 END) as success_runs,
			SUM(CASE WHEN e.status='failed' THEN 1 ELSE 0 END) as failed_runs,
			COALESCE(SUM(e.records_collected), 0) as total_records,
			MAX(e.start_time) as last_run_time
		FROM collection_tasks t
		LEFT JOIN task_executions e ON t.id = e.task_id
		GROUP BY t.id, t.name, t.type, t.status
		ORDER BY t.id
	`)
	if err != nil {
		h.log.Error("查询任务统计失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type TaskStat struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		Status       string `json:"status"`
		TotalRuns    int    `json:"total_runs"`
		SuccessRuns  int    `json:"success_runs"`
		FailedRuns   int    `json:"failed_runs"`
		TotalRecords int    `json:"total_records"`
		LastRunTime  *string `json:"last_run_time"`
	}

	stats := []TaskStat{}
	for rows.Next() {
		var stat TaskStat
		err := rows.Scan(&stat.ID, &stat.Name, &stat.Type, &stat.Status,
			&stat.TotalRuns, &stat.SuccessRuns, &stat.FailedRuns,
			&stat.TotalRecords, &stat.LastRunTime)
		if err != nil {
			h.log.Error("扫描任务统计数据失败", zap.Error(err))
			continue
		}
		stats = append(stats, stat)
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}
