package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(r *gin.Engine, db *sql.DB, log *zap.Logger) {
	// 健康检查
	r.GET("/healthz", HealthCheck)
	r.GET("/readyz", ReadyCheck(db))

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 任务管理
		tasks := v1.Group("/tasks")
		{
			taskHandler := NewTaskHandler(db, log)
			tasks.GET("", taskHandler.List)
			tasks.GET("/:id", taskHandler.Get)
			tasks.POST("", taskHandler.Create)
			tasks.PUT("/:id", taskHandler.Update)
			tasks.DELETE("/:id", taskHandler.Delete)
			tasks.POST("/:id/run", taskHandler.Run)
			tasks.POST("/:id/stop", taskHandler.Stop)
		}

		// 数据源管理
		datasources := v1.Group("/datasources")
		{
			dsHandler := NewDataSourceHandler(db, log)
			datasources.GET("", dsHandler.List)
			datasources.GET("/:id", dsHandler.Get)
			datasources.POST("", dsHandler.Create)
			datasources.PUT("/:id", dsHandler.Update)
			datasources.DELETE("/:id", dsHandler.Delete)
			datasources.POST("/:id/test", dsHandler.TestConnection)
		}

		// 清洗规则管理
		rules := v1.Group("/cleaning-rules")
		{
			ruleHandler := NewCleaningRuleHandler(db, log)
			rules.GET("", ruleHandler.List)
			rules.GET("/:id", ruleHandler.Get)
			rules.POST("", ruleHandler.Create)
			rules.PUT("/:id", ruleHandler.Update)
			rules.DELETE("/:id", ruleHandler.Delete)
		}

		// 执行历史
		executions := v1.Group("/executions")
		{
			execHandler := NewExecutionHandler(db, log)
			executions.GET("", execHandler.List)
			executions.GET("/:id", execHandler.Get)
			executions.GET("/task/:task_id", execHandler.ListByTask)
		}

		// 统计信息
		stats := v1.Group("/stats")
		{
			statsHandler := NewStatsHandler(db, log)
			stats.GET("/overview", statsHandler.Overview)
			stats.GET("/tasks", statsHandler.TaskStats)
		}
	}
}
