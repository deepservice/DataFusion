package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/datafusion/worker/internal/logger"
)

// DynamicConfig 动态配置管理器
type DynamicConfig struct {
	mu       sync.RWMutex
	config   *APIServerConfig
	watchers []ConfigWatcher
	log      *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// ConfigWatcher 配置变更监听器
type ConfigWatcher interface {
	OnConfigChange(oldConfig, newConfig *APIServerConfig) error
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Time     time.Time   `json:"time"`
}

// NewDynamicConfig 创建动态配置管理器
func NewDynamicConfig(config *APIServerConfig, log *logger.Logger) *DynamicConfig {
	ctx, cancel := context.WithCancel(context.Background())

	return &DynamicConfig{
		config:   config,
		log:      log,
		ctx:      ctx,
		cancel:   cancel,
		watchers: make([]ConfigWatcher, 0),
	}
}

// GetConfig 获取当前配置（线程安全）
func (dc *DynamicConfig) GetConfig() *APIServerConfig {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// 返回配置的深拷贝
	configCopy := *dc.config
	return &configCopy
}

// UpdateConfig 更新配置
func (dc *DynamicConfig) UpdateConfig(newConfig *APIServerConfig) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	oldConfig := dc.config

	// 验证新配置
	if err := dc.validateConfig(newConfig); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 更新配置
	dc.config = newConfig

	// 通知所有监听器
	for _, watcher := range dc.watchers {
		if err := watcher.OnConfigChange(oldConfig, newConfig); err != nil {
			dc.log.WithError(err).Error("配置变更通知失败")
		}
	}

	dc.log.Info("配置已更新")
	return nil
}

// AddWatcher 添加配置监听器
func (dc *DynamicConfig) AddWatcher(watcher ConfigWatcher) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.watchers = append(dc.watchers, watcher)
}

// RemoveWatcher 移除配置监听器
func (dc *DynamicConfig) RemoveWatcher(watcher ConfigWatcher) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	for i, w := range dc.watchers {
		if w == watcher {
			dc.watchers = append(dc.watchers[:i], dc.watchers[i+1:]...)
			break
		}
	}
}

// StartFileWatcher 启动文件监听器
func (dc *DynamicConfig) StartFileWatcher(configPath string) error {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
		defer ticker.Stop()

		var lastModTime time.Time

		for {
			select {
			case <-dc.ctx.Done():
				return
			case <-ticker.C:
				if modTime, err := getFileModTime(configPath); err == nil {
					if !lastModTime.IsZero() && modTime.After(lastModTime) {
						dc.log.Info("检测到配置文件变更，重新加载配置")
						if err := dc.reloadFromFile(configPath); err != nil {
							dc.log.WithError(err).Error("重新加载配置失败")
						}
					}
					lastModTime = modTime
				}
			}
		}
	}()

	return nil
}

// reloadFromFile 从文件重新加载配置
func (dc *DynamicConfig) reloadFromFile(configPath string) error {
	newConfig, err := LoadAPIServerConfig(configPath)
	if err != nil {
		return err
	}

	// 应用环境变量覆盖
	LoadFromEnv(newConfig)

	return dc.UpdateConfig(newConfig)
}

// validateConfig 验证配置
func (dc *DynamicConfig) validateConfig(config *APIServerConfig) error {
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("无效的服务器端口: %d", config.Server.Port)
	}

	if config.Auth.JWT.SecretKey == "" {
		return fmt.Errorf("JWT密钥不能为空")
	}

	if len(config.Auth.JWT.SecretKey) < 32 {
		return fmt.Errorf("JWT密钥长度至少32个字符")
	}

	if config.Auth.Password.MinLength < 6 {
		return fmt.Errorf("密码最小长度不能小于6")
	}

	return nil
}

// GetConfigJSON 获取配置的JSON表示
func (dc *DynamicConfig) GetConfigJSON() ([]byte, error) {
	config := dc.GetConfig()

	// 隐藏敏感信息
	configCopy := *config
	configCopy.Auth.JWT.SecretKey = "***"
	configCopy.Database.PostgreSQL.Password = "***"

	return json.MarshalIndent(configCopy, "", "  ")
}

// Stop 停止动态配置管理器
func (dc *DynamicConfig) Stop() {
	dc.cancel()
}

// getFileModTime 获取文件修改时间
func getFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// LoggerConfigWatcher 日志配置监听器
type LoggerConfigWatcher struct {
	log *logger.Logger
}

// NewLoggerConfigWatcher 创建日志配置监听器
func NewLoggerConfigWatcher(log *logger.Logger) *LoggerConfigWatcher {
	return &LoggerConfigWatcher{log: log}
}

// OnConfigChange 处理配置变更
func (w *LoggerConfigWatcher) OnConfigChange(oldConfig, newConfig *APIServerConfig) error {
	// 检查日志配置是否变更
	if oldConfig.Log.Level != newConfig.Log.Level || oldConfig.Log.Format != newConfig.Log.Format {
		w.log.Info("日志配置已变更")

		// 这里可以实现日志级别的动态调整
		// 由于zap logger的限制，完整的动态调整需要重新创建logger
	}

	return nil
}
