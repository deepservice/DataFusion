package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ValidationError 配置验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// Error 实现error接口
func (e ValidationError) Error() string {
	return fmt.Sprintf("配置验证失败 [%s]: %s (当前值: %s)", e.Field, e.Message, e.Value)
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

// ConfigValidator 配置验证器
type ConfigValidator struct {
	errors []ValidationError
}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errors: make([]ValidationError, 0),
	}
}

// ValidateConfig 验证完整配置
func (v *ConfigValidator) ValidateConfig(config *APIServerConfig) *ValidationResult {
	v.errors = make([]ValidationError, 0)

	// 验证服务器配置
	v.validateServerConfig(&config.Server)

	// 验证认证配置
	v.validateAuthConfig(&config.Auth)

	// 验证数据库配置
	v.validateDatabaseConfig(&config.Database)

	// 验证日志配置
	v.validateLogConfig(&config.Log)

	return &ValidationResult{
		Valid:  len(v.errors) == 0,
		Errors: v.errors,
	}
}

// validateServerConfig 验证服务器配置
func (v *ConfigValidator) validateServerConfig(config *ServerConfig) {
	// 验证端口
	if config.Port < 1 || config.Port > 65535 {
		v.addError("server.port", "端口必须在1-65535范围内", strconv.Itoa(config.Port))
	}

	// 验证模式
	validModes := []string{"debug", "release", "test"}
	if !v.contains(validModes, config.Mode) {
		v.addError("server.mode", "模式必须是debug、release或test之一", config.Mode)
	}

	// 验证超时时间
	if config.ReadTimeout < 1 || config.ReadTimeout > 300 {
		v.addError("server.read_timeout", "读取超时必须在1-300秒范围内", strconv.Itoa(config.ReadTimeout))
	}

	if config.WriteTimeout < 1 || config.WriteTimeout > 300 {
		v.addError("server.write_timeout", "写入超时必须在1-300秒范围内", strconv.Itoa(config.WriteTimeout))
	}
}

// validateAuthConfig 验证认证配置
func (v *ConfigValidator) validateAuthConfig(config *AuthConfig) {
	// 验证JWT配置
	if config.JWT.SecretKey == "" {
		v.addError("auth.jwt.secret_key", "JWT密钥不能为空", "")
	} else if len(config.JWT.SecretKey) < 32 {
		v.addError("auth.jwt.secret_key", "JWT密钥长度至少32个字符", fmt.Sprintf("%d字符", len(config.JWT.SecretKey)))
	}

	// 验证Token持续时间
	if config.JWT.TokenDuration != "" {
		if _, err := time.ParseDuration(config.JWT.TokenDuration); err != nil {
			v.addError("auth.jwt.token_duration", "无效的时间格式", config.JWT.TokenDuration)
		}
	}

	// 验证密码策略
	if config.Password.MinLength < 6 || config.Password.MinLength > 128 {
		v.addError("auth.password.min_length", "密码最小长度必须在6-128范围内", strconv.Itoa(config.Password.MinLength))
	}
}

// validateDatabaseConfig 验证数据库配置
func (v *ConfigValidator) validateDatabaseConfig(config *DBConfig) {
	pg := &config.PostgreSQL

	// 验证主机
	if pg.Host == "" {
		v.addError("database.postgresql.host", "数据库主机不能为空", "")
	} else if pg.Host != "localhost" && net.ParseIP(pg.Host) == nil {
		// 如果不是localhost，检查是否为有效IP
		if _, err := net.LookupHost(pg.Host); err != nil {
			v.addError("database.postgresql.host", "无效的主机名或IP地址", pg.Host)
		}
	}

	// 验证端口
	if pg.Port < 1 || pg.Port > 65535 {
		v.addError("database.postgresql.port", "数据库端口必须在1-65535范围内", strconv.Itoa(pg.Port))
	}

	// 验证用户名
	if pg.User == "" {
		v.addError("database.postgresql.user", "数据库用户名不能为空", "")
	}

	// 验证数据库名
	if pg.Database == "" {
		v.addError("database.postgresql.database", "数据库名不能为空", "")
	}

	// 验证SSL模式
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	if !v.contains(validSSLModes, pg.SSLMode) {
		v.addError("database.postgresql.sslmode", "无效的SSL模式", pg.SSLMode)
	}

	// 验证连接池配置
	if pg.MaxOpenConns < 1 || pg.MaxOpenConns > 100 {
		v.addError("database.postgresql.max_open_conns", "最大连接数必须在1-100范围内", strconv.Itoa(pg.MaxOpenConns))
	}

	if pg.MaxIdleConns < 1 || pg.MaxIdleConns > pg.MaxOpenConns {
		v.addError("database.postgresql.max_idle_conns", "最大空闲连接数必须在1到最大连接数范围内", strconv.Itoa(pg.MaxIdleConns))
	}

	if pg.ConnMaxLifetime < 60 || pg.ConnMaxLifetime > 3600 {
		v.addError("database.postgresql.conn_max_lifetime", "连接最大生命周期必须在60-3600秒范围内", strconv.Itoa(pg.ConnMaxLifetime))
	}
}

// validateLogConfig 验证日志配置
func (v *ConfigValidator) validateLogConfig(config *LogConfig) {
	// 验证日志级别
	validLevels := []string{"debug", "info", "warn", "error"}
	if !v.contains(validLevels, config.Level) {
		v.addError("log.level", "日志级别必须是debug、info、warn或error之一", config.Level)
	}

	// 验证日志格式
	validFormats := []string{"json", "console"}
	if !v.contains(validFormats, config.Format) {
		v.addError("log.format", "日志格式必须是json或console之一", config.Format)
	}
}

// addError 添加验证错误
func (v *ConfigValidator) addError(field, message, value string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// contains 检查切片是否包含指定值
func (v *ConfigValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidateConfigFile 验证配置文件
func ValidateConfigFile(configPath string) (*ValidationResult, error) {
	config, err := LoadAPIServerConfig(configPath)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:   "config_file",
					Message: "配置文件加载失败: " + err.Error(),
					Value:   configPath,
				},
			},
		}, err
	}

	validator := NewConfigValidator()
	return validator.ValidateConfig(config), nil
}

// GetConfigRecommendations 获取配置建议
func GetConfigRecommendations(config *APIServerConfig) []string {
	var recommendations []string

	// 生产环境建议
	if config.Server.Mode == "debug" {
		recommendations = append(recommendations, "生产环境建议将server.mode设置为release")
	}

	// 安全建议
	if strings.Contains(config.Auth.JWT.SecretKey, "default") || strings.Contains(config.Auth.JWT.SecretKey, "change") {
		recommendations = append(recommendations, "建议更改默认的JWT密钥")
	}

	if config.Database.PostgreSQL.Password == "postgres" {
		recommendations = append(recommendations, "建议更改默认的数据库密码")
	}

	// 性能建议
	if config.Database.PostgreSQL.MaxOpenConns < 10 {
		recommendations = append(recommendations, "建议增加数据库最大连接数以提高性能")
	}

	// 日志建议
	if config.Log.Level == "debug" && config.Server.Mode == "release" {
		recommendations = append(recommendations, "生产环境建议将日志级别设置为info或warn")
	}

	return recommendations
}
