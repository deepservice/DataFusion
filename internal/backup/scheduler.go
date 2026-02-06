package backup

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/logger"
	"github.com/robfig/cron/v3"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Enabled         bool   `json:"enabled"`           // 是否启用
	CronExpression  string `json:"cron_expression"`   // Cron表达式
	BackupDir       string `json:"backup_dir"`        // 备份目录
	RetentionDays   int    `json:"retention_days"`    // 保留天数
	MaxBackups      int    `json:"max_backups"`       // 最大备份数量
	CompressBackups bool   `json:"compress_backups"`  // 是否压缩
	NotifyOnFailure bool   `json:"notify_on_failure"` // 失败时通知
	NotifyOnSuccess bool   `json:"notify_on_success"` // 成功时通知
}

// BackupScheduler 备份调度器
type BackupScheduler struct {
	config        *SchedulerConfig
	pgBackup      *PostgresBackup
	log           *logger.Logger
	cron          *cron.Cron
	ctx           context.Context
	cancel        context.CancelFunc
	lastBackup    *BackupResult
	backupHistory []BackupResult
}

// NewBackupScheduler 创建备份调度器
func NewBackupScheduler(config *SchedulerConfig, dbConfig *config.PostgreSQLConfig, log *logger.Logger) *BackupScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &BackupScheduler{
		config:        config,
		pgBackup:      NewPostgresBackup(dbConfig, log),
		log:           log,
		cron:          cron.New(cron.WithSeconds()),
		ctx:           ctx,
		cancel:        cancel,
		backupHistory: make([]BackupResult, 0),
	}
}

// Start 启动调度器
func (bs *BackupScheduler) Start() error {
	if !bs.config.Enabled {
		bs.log.Info("备份调度器已禁用")
		return nil
	}

	// 添加定时任务
	_, err := bs.cron.AddFunc(bs.config.CronExpression, bs.performBackup)
	if err != nil {
		return fmt.Errorf("添加定时任务失败: %v", err)
	}

	// 启动cron调度器
	bs.cron.Start()

	bs.log.Info("备份调度器已启动")

	// 启动清理任务
	go bs.startCleanupTask()

	return nil
}

// Stop 停止调度器
func (bs *BackupScheduler) Stop() {
	bs.cancel()
	bs.cron.Stop()
	bs.log.Info("备份调度器已停止")
}

// performBackup 执行备份
func (bs *BackupScheduler) performBackup() {
	bs.log.Info("开始执行定时备份")

	options := BackupOptions{
		OutputDir: bs.config.BackupDir,
		Compress:  bs.config.CompressBackups,
	}

	result, err := bs.pgBackup.CreateBackup(options)
	if result != nil {
		bs.lastBackup = result
		bs.addToHistory(*result)
	}

	if err != nil {
		bs.log.WithError(err).Error("定时备份失败")
		if bs.config.NotifyOnFailure {
			bs.sendNotification("备份失败", fmt.Sprintf("定时备份失败: %v", err), "error")
		}
		return
	}

	bs.log.Info("定时备份完成")

	if bs.config.NotifyOnSuccess {
		bs.sendNotification("备份成功",
			fmt.Sprintf("备份已完成: %s (大小: %d bytes, 耗时: %s)",
				result.Filename, result.Size, result.Duration), "success")
	}
}

// startCleanupTask 启动清理任务
func (bs *BackupScheduler) startCleanupTask() {
	ticker := time.NewTicker(24 * time.Hour) // 每天检查一次
	defer ticker.Stop()

	for {
		select {
		case <-bs.ctx.Done():
			return
		case <-ticker.C:
			bs.cleanupOldBackups()
		}
	}
}

// cleanupOldBackups 清理旧备份
func (bs *BackupScheduler) cleanupOldBackups() {
	bs.log.Info("开始清理旧备份文件")

	backups, err := bs.pgBackup.ListBackups(bs.config.BackupDir)
	if err != nil {
		bs.log.WithError(err).Error("获取备份列表失败")
		return
	}

	if len(backups) == 0 {
		return
	}

	// 按修改时间排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	var toDelete []BackupInfo
	now := time.Now()

	// 根据保留天数删除
	if bs.config.RetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -bs.config.RetentionDays)
		for _, backup := range backups {
			if backup.ModTime.Before(cutoff) {
				toDelete = append(toDelete, backup)
			}
		}
	}

	// 根据最大备份数量删除
	if bs.config.MaxBackups > 0 && len(backups) > bs.config.MaxBackups {
		excess := backups[bs.config.MaxBackups:]
		toDelete = append(toDelete, excess...)
	}

	// 删除文件
	for _, backup := range toDelete {
		if err := bs.pgBackup.DeleteBackup(backup.Path); err != nil {
			bs.log.WithError(err).Error("删除备份文件失败")
		} else {
			bs.log.Info("已删除旧备份文件")
		}
	}

	if len(toDelete) > 0 {
		bs.log.Info("备份清理完成")
	}
}

// addToHistory 添加到历史记录
func (bs *BackupScheduler) addToHistory(result BackupResult) {
	bs.backupHistory = append(bs.backupHistory, result)

	// 保持历史记录不超过100条
	if len(bs.backupHistory) > 100 {
		bs.backupHistory = bs.backupHistory[1:]
	}
}

// GetLastBackup 获取最后一次备份结果
func (bs *BackupScheduler) GetLastBackup() *BackupResult {
	return bs.lastBackup
}

// GetBackupHistory 获取备份历史
func (bs *BackupScheduler) GetBackupHistory() []BackupResult {
	return bs.backupHistory
}

// GetStatus 获取调度器状态
func (bs *BackupScheduler) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":         bs.config.Enabled,
		"cron_expression": bs.config.CronExpression,
		"backup_dir":      bs.config.BackupDir,
		"retention_days":  bs.config.RetentionDays,
		"max_backups":     bs.config.MaxBackups,
		"compress":        bs.config.CompressBackups,
	}

	if bs.lastBackup != nil {
		status["last_backup"] = bs.lastBackup
	}

	// 获取下次执行时间
	entries := bs.cron.Entries()
	if len(entries) > 0 {
		status["next_run"] = entries[0].Next
	}

	return status
}

// UpdateConfig 更新配置
func (bs *BackupScheduler) UpdateConfig(newConfig *SchedulerConfig) error {
	// 停止当前调度
	bs.cron.Stop()

	// 更新配置
	bs.config = newConfig

	// 如果启用，重新启动
	if newConfig.Enabled {
		return bs.Start()
	}

	return nil
}

// TriggerBackup 手动触发备份
func (bs *BackupScheduler) TriggerBackup() (*BackupResult, error) {
	bs.log.Info("手动触发备份")

	options := BackupOptions{
		OutputDir: bs.config.BackupDir,
		Compress:  bs.config.CompressBackups,
	}

	result, err := bs.pgBackup.CreateBackup(options)
	if result != nil {
		bs.lastBackup = result
		bs.addToHistory(*result)
	}

	return result, err
}

// sendNotification 发送通知
func (bs *BackupScheduler) sendNotification(title, message, level string) {
	// 这里可以实现各种通知方式：邮件、Slack、钉钉等
	bs.log.Info("发送备份通知")

	// TODO: 实现具体的通知逻辑
}

// GetDefaultSchedulerConfig 获取默认调度器配置
func GetDefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Enabled:         false,
		CronExpression:  "0 0 2 * * *", // 每天凌晨2点
		BackupDir:       "backups",
		RetentionDays:   30, // 保留30天
		MaxBackups:      10, // 最多10个备份
		CompressBackups: true,
		NotifyOnFailure: true,
		NotifyOnSuccess: false,
	}
}
