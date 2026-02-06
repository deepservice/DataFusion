package api

import (
	"net/http"
	"strconv"

	"github.com/datafusion/worker/internal/backup"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// BackupHandler 备份管理处理器
type BackupHandler struct {
	pgBackup  *backup.PostgresBackup
	scheduler *backup.BackupScheduler
	log       *logger.Logger
}

// NewBackupHandler 创建备份管理处理器
func NewBackupHandler(pgBackup *backup.PostgresBackup, scheduler *backup.BackupScheduler, log *logger.Logger) *BackupHandler {
	return &BackupHandler{
		pgBackup:  pgBackup,
		scheduler: scheduler,
		log:       log,
	}
}

// CreateBackup 创建备份
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var options backup.BackupOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的备份选项"})
		return
	}

	// 设置默认值
	if options.OutputDir == "" {
		options.OutputDir = "backups"
	}

	result, err := h.pgBackup.CreateBackup(options)
	if err != nil {
		h.log.WithError(err).Error("创建备份失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "创建备份失败",
			"result": result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "备份创建成功",
		"result":  result,
	})
}

// ListBackups 获取备份列表
func (h *BackupHandler) ListBackups(c *gin.Context) {
	backupDir := c.DefaultQuery("dir", "backups")

	backups, err := h.pgBackup.ListBackups(backupDir)
	if err != nil {
		h.log.WithError(err).Error("获取备份列表失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取备份列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"count":   len(backups),
	})
}

// RestoreBackup 恢复备份
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	var options backup.RestoreOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的恢复选项"})
		return
	}

	if options.BackupFile == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备份文件路径不能为空"})
		return
	}

	// 验证备份文件
	if err := h.pgBackup.ValidateBackup(options.BackupFile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备份文件验证失败: " + err.Error()})
		return
	}

	result, err := h.pgBackup.RestoreBackup(options)
	if err != nil {
		h.log.WithError(err).Error("恢复备份失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "恢复备份失败",
			"result": result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "备份恢复成功",
		"result":  result,
	})
}

// DeleteBackup 删除备份
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	backupPath := c.Query("path")
	if backupPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备份文件路径不能为空"})
		return
	}

	if err := h.pgBackup.DeleteBackup(backupPath); err != nil {
		h.log.WithError(err).Error("删除备份失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除备份失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "备份删除成功"})
}

// ValidateBackup 验证备份文件
func (h *BackupHandler) ValidateBackup(c *gin.Context) {
	backupPath := c.Query("path")
	if backupPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备份文件路径不能为空"})
		return
	}

	if err := h.pgBackup.ValidateBackup(backupPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "备份文件验证通过",
	})
}

// GetSchedulerStatus 获取调度器状态
func (h *BackupHandler) GetSchedulerStatus(c *gin.Context) {
	status := h.scheduler.GetStatus()
	c.JSON(http.StatusOK, status)
}

// UpdateSchedulerConfig 更新调度器配置
func (h *BackupHandler) UpdateSchedulerConfig(c *gin.Context) {
	var config backup.SchedulerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的调度器配置"})
		return
	}

	if err := h.scheduler.UpdateConfig(&config); err != nil {
		h.log.WithError(err).Error("更新调度器配置失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新调度器配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "调度器配置更新成功"})
}

// TriggerBackup 手动触发备份
func (h *BackupHandler) TriggerBackup(c *gin.Context) {
	result, err := h.scheduler.TriggerBackup()
	if err != nil {
		h.log.WithError(err).Error("手动触发备份失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "手动触发备份失败",
			"result": result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "备份触发成功",
		"result":  result,
	})
}

// GetBackupHistory 获取备份历史
func (h *BackupHandler) GetBackupHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	history := h.scheduler.GetBackupHistory()

	// 限制返回数量
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

// GetBackupStats 获取备份统计信息
func (h *BackupHandler) GetBackupStats(c *gin.Context) {
	backupDir := c.DefaultQuery("dir", "backups")

	backups, err := h.pgBackup.ListBackups(backupDir)
	if err != nil {
		h.log.WithError(err).Error("获取备份统计失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取备份统计失败"})
		return
	}

	var totalSize int64
	var compressedCount int
	var oldestBackup, newestBackup *backup.BackupInfo

	for i, b := range backups {
		totalSize += b.Size
		if b.Compressed {
			compressedCount++
		}

		if oldestBackup == nil || b.ModTime.Before(oldestBackup.ModTime) {
			oldestBackup = &backups[i]
		}

		if newestBackup == nil || b.ModTime.After(newestBackup.ModTime) {
			newestBackup = &backups[i]
		}
	}

	stats := gin.H{
		"total_backups":    len(backups),
		"total_size":       totalSize,
		"compressed_count": compressedCount,
		"backup_dir":       backupDir,
	}

	if oldestBackup != nil {
		stats["oldest_backup"] = oldestBackup
	}

	if newestBackup != nil {
		stats["newest_backup"] = newestBackup
	}

	c.JSON(http.StatusOK, stats)
}
