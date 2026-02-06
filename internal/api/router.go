package api

import (
	"database/sql"

	"github.com/datafusion/worker/internal/auth"
	"github.com/datafusion/worker/internal/cache"
	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(r *gin.Engine, db *sql.DB, log *logger.Logger, cfg *config.APIServerConfig, cacheInstance cache.Cache) {
	// 创建JWT管理器和RBAC
	jwtManager := auth.NewJWTManager(cfg.Auth.JWT.SecretKey, cfg.Auth.GetJWTDuration())
	rbac := auth.NewRBAC()

	// 健康检查（无需认证）
	r.GET("/healthz", HealthCheck)
	r.GET("/readyz", ReadyCheck(db))

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证相关路由（无需认证）
		authHandler := NewAuthHandler(db, log, jwtManager)
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/refresh", authHandler.RefreshToken)

		// 需要认证的路由
		authenticated := v1.Group("")
		authenticated.Use(auth.AuthMiddleware(jwtManager))
		{
			// 用户信息
			authenticated.GET("/auth/profile", authHandler.GetProfile)
			authenticated.PUT("/auth/profile", authHandler.UpdateProfile)
			authenticated.POST("/auth/change-password", authHandler.ChangePassword)
			authenticated.POST("/auth/logout", authHandler.Logout)

			// API密钥管理
			apiKeyHandler := NewAPIKeyHandler(db, log)
			apiKeys := authenticated.Group("/api-keys")
			{
				apiKeys.GET("", apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
				apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
				apiKeys.POST("/:id/revoke", apiKeyHandler.RevokeAPIKey)
				apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
			}

			// 用户管理（仅管理员）
			userHandler := NewUserHandler(db, log)
			users := authenticated.Group("/users")
			users.Use(auth.RequireRole("admin"))
			{
				users.GET("", userHandler.ListUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.POST("/:id/reset-password", userHandler.ResetPassword)
			}

			// 角色信息
			authenticated.GET("/roles", userHandler.GetRoles)

			// 任务管理
			tasks := authenticated.Group("/tasks")
			tasks.Use(auth.RequirePermission(rbac, "tasks", "read"))
			{
				taskHandler := NewTaskHandler(db, log)
				tasks.GET("", taskHandler.List)
				tasks.GET("/:id", taskHandler.Get)

				// 写操作需要写权限
				writeGroup := tasks.Group("")
				writeGroup.Use(auth.RequirePermission(rbac, "tasks", "write"))
				{
					writeGroup.POST("", taskHandler.Create)
					writeGroup.PUT("/:id", taskHandler.Update)
					writeGroup.POST("/:id/run", taskHandler.Run)
					writeGroup.POST("/:id/stop", taskHandler.Stop)
				}

				// 删除操作需要删除权限
				deleteGroup := tasks.Group("")
				deleteGroup.Use(auth.RequirePermission(rbac, "tasks", "delete"))
				{
					deleteGroup.DELETE("/:id", taskHandler.Delete)
				}
			}

			// 数据源管理
			datasources := authenticated.Group("/datasources")
			datasources.Use(auth.RequirePermission(rbac, "datasources", "read"))
			{
				dsHandler := NewDataSourceHandler(db, log)
				datasources.GET("", dsHandler.List)
				datasources.GET("/:id", dsHandler.Get)

				// 写操作需要写权限
				writeGroup := datasources.Group("")
				writeGroup.Use(auth.RequirePermission(rbac, "datasources", "write"))
				{
					writeGroup.POST("", dsHandler.Create)
					writeGroup.PUT("/:id", dsHandler.Update)
					writeGroup.POST("/:id/test", dsHandler.TestConnection)
				}

				// 删除操作需要删除权限
				deleteGroup := datasources.Group("")
				deleteGroup.Use(auth.RequirePermission(rbac, "datasources", "delete"))
				{
					deleteGroup.DELETE("/:id", dsHandler.Delete)
				}
			}

			// 清洗规则管理
			rules := authenticated.Group("/cleaning-rules")
			rules.Use(auth.RequirePermission(rbac, "cleaning-rules", "read"))
			{
				ruleHandler := NewCleaningRuleHandler(db, log)
				rules.GET("", ruleHandler.List)
				rules.GET("/:id", ruleHandler.Get)

				// 写操作需要写权限
				writeGroup := rules.Group("")
				writeGroup.Use(auth.RequirePermission(rbac, "cleaning-rules", "write"))
				{
					writeGroup.POST("", ruleHandler.Create)
					writeGroup.PUT("/:id", ruleHandler.Update)
				}

				// 删除操作需要删除权限
				deleteGroup := rules.Group("")
				deleteGroup.Use(auth.RequirePermission(rbac, "cleaning-rules", "delete"))
				{
					deleteGroup.DELETE("/:id", ruleHandler.Delete)
				}
			}

			// 执行历史
			executions := authenticated.Group("/executions")
			executions.Use(auth.RequirePermission(rbac, "executions", "read"))
			{
				execHandler := NewExecutionHandler(db, log)
				executions.GET("", execHandler.List)
				executions.GET("/:id", execHandler.Get)
				executions.GET("/task/:task_id", execHandler.ListByTask)
			}

			// 统计信息
			stats := authenticated.Group("/stats")
			stats.Use(auth.RequirePermission(rbac, "stats", "read"))
			{
				statsHandler := NewStatsHandler(db, log)
				stats.GET("/overview", statsHandler.Overview)
				stats.GET("/tasks", statsHandler.TaskStats)
			}

			// 配置管理（仅管理员）
			configHandler := NewConfigHandler(nil, log) // TODO: 传入动态配置实例
			configGroup := authenticated.Group("/config")
			configGroup.Use(auth.RequireRole("admin"))
			{
				configGroup.GET("", configHandler.GetConfig)
				configGroup.POST("/validate", configHandler.ValidateConfig)
				configGroup.PUT("", configHandler.UpdateConfig)
				configGroup.POST("/reload", configHandler.ReloadConfig)
				configGroup.GET("/schema", configHandler.GetConfigSchema)
				configGroup.GET("/status", configHandler.GetConfigStatus)
			}

			// 备份管理（仅管理员）
			backupHandler := NewBackupHandler(nil, nil, log) // TODO: 传入备份实例
			backupGroup := authenticated.Group("/backup")
			backupGroup.Use(auth.RequireRole("admin"))
			{
				backupGroup.POST("", backupHandler.CreateBackup)
				backupGroup.GET("/list", backupHandler.ListBackups)
				backupGroup.POST("/restore", backupHandler.RestoreBackup)
				backupGroup.DELETE("", backupHandler.DeleteBackup)
				backupGroup.GET("/validate", backupHandler.ValidateBackup)
				backupGroup.GET("/stats", backupHandler.GetBackupStats)
				backupGroup.GET("/history", backupHandler.GetBackupHistory)

				// 调度器管理
				backupGroup.GET("/scheduler/status", backupHandler.GetSchedulerStatus)
				backupGroup.PUT("/scheduler/config", backupHandler.UpdateSchedulerConfig)
				backupGroup.POST("/scheduler/trigger", backupHandler.TriggerBackup)
			}
			// 缓存管理（仅管理员）
			cacheHandler := NewCacheHandler(cacheInstance, log)
			cacheGroup := authenticated.Group("/cache")
			cacheGroup.Use(auth.RequireRole("admin"))
			{
				cacheGroup.GET("/stats", cacheHandler.GetCacheStats)
				cacheGroup.POST("/flush", cacheHandler.FlushCache)
				cacheGroup.GET("/ping", cacheHandler.PingCache)
				cacheGroup.GET("/keys/:key", cacheHandler.GetCacheKey)
				cacheGroup.PUT("/keys/:key", cacheHandler.SetCacheKey)
				cacheGroup.DELETE("/keys/:key", cacheHandler.DeleteCacheKey)
				cacheGroup.HEAD("/keys/:key", cacheHandler.CheckCacheKey)
				cacheGroup.POST("/counters/:key/incr", cacheHandler.IncrementCounter)
			}
		}
	}
}
