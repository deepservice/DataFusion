package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/logger"
)

// PostgresBackup PostgreSQL备份管理器
type PostgresBackup struct {
	config *config.PostgreSQLConfig
	log    *logger.Logger
}

// NewPostgresBackup 创建PostgreSQL备份管理器
func NewPostgresBackup(config *config.PostgreSQLConfig, log *logger.Logger) *PostgresBackup {
	return &PostgresBackup{
		config: config,
		log:    log,
	}
}

// BackupOptions 备份选项
type BackupOptions struct {
	OutputDir     string   `json:"output_dir"`     // 输出目录
	Filename      string   `json:"filename"`       // 文件名（可选）
	Compress      bool     `json:"compress"`       // 是否压缩
	SchemaOnly    bool     `json:"schema_only"`    // 仅备份结构
	DataOnly      bool     `json:"data_only"`      // 仅备份数据
	Tables        []string `json:"tables"`         // 指定表（可选）
	ExcludeTables []string `json:"exclude_tables"` // 排除表（可选）
}

// BackupResult 备份结果
type BackupResult struct {
	Success   bool      `json:"success"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Duration  string    `json:"duration"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     string    `json:"error,omitempty"`
}

// CreateBackup 创建备份
func (pb *PostgresBackup) CreateBackup(options BackupOptions) (*BackupResult, error) {
	startTime := time.Now()

	result := &BackupResult{
		StartTime: startTime,
	}

	// 生成备份文件名
	if options.Filename == "" {
		timestamp := startTime.Format("20060102_150405")
		options.Filename = fmt.Sprintf("datafusion_backup_%s.sql", timestamp)
	}

	// 确保输出目录存在
	if options.OutputDir == "" {
		options.OutputDir = "backups"
	}

	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		result.Error = fmt.Sprintf("创建备份目录失败: %v", err)
		return result, err
	}

	backupPath := filepath.Join(options.OutputDir, options.Filename)

	// 如果需要压缩，添加.gz扩展名
	if options.Compress && !strings.HasSuffix(backupPath, ".gz") {
		backupPath += ".gz"
	}

	pb.log.WithFields(map[string]interface{}{
		"database": pb.config.Database,
		"output":   backupPath,
	}).Info("开始创建数据库备份")

	// 构建pg_dump命令
	cmd := pb.buildPgDumpCommand(backupPath, options)

	// 设置环境变量
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", pb.config.Password))

	// 执行备份
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime).String()
	result.Filename = backupPath

	if err != nil {
		result.Error = fmt.Sprintf("备份失败: %v, 输出: %s", err, string(output))
		pb.log.WithError(err).WithFields(map[string]interface{}{
			"output": string(output),
		}).Error("数据库备份失败")
		return result, err
	}

	// 获取文件大小
	if info, err := os.Stat(backupPath); err == nil {
		result.Size = info.Size()
	}

	result.Success = true

	pb.log.WithFields(map[string]interface{}{
		"file":     backupPath,
		"size":     result.Size,
		"duration": result.Duration,
	}).Info("数据库备份完成")

	return result, nil
}

// buildPgDumpCommand 构建pg_dump命令
func (pb *PostgresBackup) buildPgDumpCommand(outputPath string, options BackupOptions) *exec.Cmd {
	args := []string{
		"-h", pb.config.Host,
		"-p", fmt.Sprintf("%d", pb.config.Port),
		"-U", pb.config.User,
		"-d", pb.config.Database,
		"--verbose",
		"--no-password",
	}

	// 添加选项
	if options.SchemaOnly {
		args = append(args, "--schema-only")
	} else if options.DataOnly {
		args = append(args, "--data-only")
	}

	// 指定表
	for _, table := range options.Tables {
		args = append(args, "-t", table)
	}

	// 排除表
	for _, table := range options.ExcludeTables {
		args = append(args, "-T", table)
	}

	// 输出选项
	if options.Compress {
		args = append(args, "-Z", "9") // 最高压缩级别
		args = append(args, "-f", outputPath)
	} else {
		args = append(args, "-f", outputPath)
	}

	return exec.Command("pg_dump", args...)
}

// RestoreOptions 恢复选项
type RestoreOptions struct {
	BackupFile    string   `json:"backup_file"`    // 备份文件路径
	CleanFirst    bool     `json:"clean_first"`    // 恢复前清理
	CreateDB      bool     `json:"create_db"`      // 创建数据库
	Tables        []string `json:"tables"`         // 指定表
	ExcludeTables []string `json:"exclude_tables"` // 排除表
}

// RestoreResult 恢复结果
type RestoreResult struct {
	Success   bool      `json:"success"`
	Duration  string    `json:"duration"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     string    `json:"error,omitempty"`
}

// RestoreBackup 恢复备份
func (pb *PostgresBackup) RestoreBackup(options RestoreOptions) (*RestoreResult, error) {
	startTime := time.Now()

	result := &RestoreResult{
		StartTime: startTime,
	}

	// 检查备份文件是否存在
	if _, err := os.Stat(options.BackupFile); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("备份文件不存在: %s", options.BackupFile)
		return result, err
	}

	pb.log.Info("开始恢复数据库备份")

	// 构建恢复命令
	var cmd *exec.Cmd
	if strings.HasSuffix(options.BackupFile, ".gz") {
		// 压缩文件，使用gunzip + psql
		cmd = pb.buildRestoreCompressedCommand(options)
	} else {
		// 普通SQL文件，使用psql
		cmd = pb.buildRestoreCommand(options)
	}

	// 设置环境变量
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", pb.config.Password))

	// 执行恢复
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime).String()

	if err != nil {
		result.Error = fmt.Sprintf("恢复失败: %v, 输出: %s", err, string(output))
		pb.log.WithError(err).Error("数据库恢复失败")
		return result, err
	}

	result.Success = true

	pb.log.Info("数据库恢复完成")

	return result, nil
}

// buildRestoreCommand 构建恢复命令
func (pb *PostgresBackup) buildRestoreCommand(options RestoreOptions) *exec.Cmd {
	args := []string{
		"-h", pb.config.Host,
		"-p", fmt.Sprintf("%d", pb.config.Port),
		"-U", pb.config.User,
		"-d", pb.config.Database,
		"-f", options.BackupFile,
		"--no-password",
	}

	if options.CleanFirst {
		// 注意：这个选项需要谨慎使用
		args = append(args, "-c")
	}

	return exec.Command("psql", args...)
}

// buildRestoreCompressedCommand 构建压缩文件恢复命令
func (pb *PostgresBackup) buildRestoreCompressedCommand(options RestoreOptions) *exec.Cmd {
	// 使用管道: gunzip -c backup.sql.gz | psql
	psqlArgs := []string{
		"-h", pb.config.Host,
		"-p", fmt.Sprintf("%d", pb.config.Port),
		"-U", pb.config.User,
		"-d", pb.config.Database,
		"--no-password",
	}

	// 创建管道命令
	gunzipCmd := exec.Command("gunzip", "-c", options.BackupFile)
	psqlCmd := exec.Command("psql", psqlArgs...)

	// 连接管道
	psqlCmd.Stdin, _ = gunzipCmd.StdoutPipe()

	// 启动gunzip
	gunzipCmd.Start()

	return psqlCmd
}

// ListBackups 列出备份文件
func (pb *PostgresBackup) ListBackups(backupDir string) ([]BackupInfo, error) {
	if backupDir == "" {
		backupDir = "backups"
	}

	var backups []BackupInfo

	// 检查目录是否存在
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return backups, nil
	}

	// 读取目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".sql") && !strings.HasSuffix(name, ".sql.gz") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		backup := BackupInfo{
			Filename:   name,
			Path:       filepath.Join(backupDir, name),
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			Compressed: strings.HasSuffix(name, ".gz"),
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

// BackupInfo 备份信息
type BackupInfo struct {
	Filename   string    `json:"filename"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	Compressed bool      `json:"compressed"`
}

// DeleteBackup 删除备份文件
func (pb *PostgresBackup) DeleteBackup(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("删除备份文件失败: %v", err)
	}

	pb.log.Info("备份文件已删除")

	return nil
}

// ValidateBackup 验证备份文件
func (pb *PostgresBackup) ValidateBackup(backupPath string) error {
	// 检查文件是否存在
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("备份文件不存在: %v", err)
	}

	// 检查文件大小
	if info.Size() == 0 {
		return fmt.Errorf("备份文件为空")
	}

	// 检查文件格式
	if strings.HasSuffix(backupPath, ".gz") {
		// 验证gzip文件
		cmd := exec.Command("gunzip", "-t", backupPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("备份文件损坏（gzip验证失败）: %v", err)
		}
	}

	return nil
}
