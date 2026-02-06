package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvConfig 环境变量配置管理
type EnvConfig struct {
	prefix string
}

// NewEnvConfig 创建环境变量配置管理器
func NewEnvConfig(prefix string) *EnvConfig {
	return &EnvConfig{
		prefix: strings.ToUpper(prefix),
	}
}

// GetString 获取字符串环境变量
func (e *EnvConfig) GetString(key, defaultValue string) string {
	envKey := e.prefix + "_" + strings.ToUpper(key)
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// GetInt 获取整数环境变量
func (e *EnvConfig) GetInt(key string, defaultValue int) int {
	envKey := e.prefix + "_" + strings.ToUpper(key)
	if value := os.Getenv(envKey); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetBool 获取布尔环境变量
func (e *EnvConfig) GetBool(key string, defaultValue bool) bool {
	envKey := e.prefix + "_" + strings.ToUpper(key)
	if value := os.Getenv(envKey); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDuration 获取时间间隔环境变量
func (e *EnvConfig) GetDuration(key string, defaultValue time.Duration) time.Duration {
	envKey := e.prefix + "_" + strings.ToUpper(key)
	if value := os.Getenv(envKey); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadFromEnv 从环境变量加载配置到现有配置结构
func LoadFromEnv(cfg *APIServerConfig) {
	env := NewEnvConfig("DATAFUSION")

	// 服务器配置
	cfg.Server.Port = env.GetInt("SERVER_PORT", cfg.Server.Port)
	cfg.Server.Mode = env.GetString("SERVER_MODE", cfg.Server.Mode)
	cfg.Server.ReadTimeout = env.GetInt("SERVER_READ_TIMEOUT", cfg.Server.ReadTimeout)
	cfg.Server.WriteTimeout = env.GetInt("SERVER_WRITE_TIMEOUT", cfg.Server.WriteTimeout)

	// 认证配置
	cfg.Auth.JWT.SecretKey = env.GetString("JWT_SECRET_KEY", cfg.Auth.JWT.SecretKey)
	cfg.Auth.JWT.TokenDuration = env.GetString("JWT_TOKEN_DURATION", cfg.Auth.JWT.TokenDuration)

	cfg.Auth.Password.MinLength = env.GetInt("PASSWORD_MIN_LENGTH", cfg.Auth.Password.MinLength)
	cfg.Auth.Password.RequireUpper = env.GetBool("PASSWORD_REQUIRE_UPPER", cfg.Auth.Password.RequireUpper)
	cfg.Auth.Password.RequireLower = env.GetBool("PASSWORD_REQUIRE_LOWER", cfg.Auth.Password.RequireLower)
	cfg.Auth.Password.RequireDigit = env.GetBool("PASSWORD_REQUIRE_DIGIT", cfg.Auth.Password.RequireDigit)
	cfg.Auth.Password.RequireSpecial = env.GetBool("PASSWORD_REQUIRE_SPECIAL", cfg.Auth.Password.RequireSpecial)

	// 数据库配置
	cfg.Database.PostgreSQL.Host = env.GetString("DB_HOST", cfg.Database.PostgreSQL.Host)
	cfg.Database.PostgreSQL.Port = env.GetInt("DB_PORT", cfg.Database.PostgreSQL.Port)
	cfg.Database.PostgreSQL.User = env.GetString("DB_USER", cfg.Database.PostgreSQL.User)
	cfg.Database.PostgreSQL.Password = env.GetString("DB_PASSWORD", cfg.Database.PostgreSQL.Password)
	cfg.Database.PostgreSQL.Database = env.GetString("DB_NAME", cfg.Database.PostgreSQL.Database)
	cfg.Database.PostgreSQL.SSLMode = env.GetString("DB_SSLMODE", cfg.Database.PostgreSQL.SSLMode)
	cfg.Database.PostgreSQL.MaxOpenConns = env.GetInt("DB_MAX_OPEN_CONNS", cfg.Database.PostgreSQL.MaxOpenConns)
	cfg.Database.PostgreSQL.MaxIdleConns = env.GetInt("DB_MAX_IDLE_CONNS", cfg.Database.PostgreSQL.MaxIdleConns)
	cfg.Database.PostgreSQL.ConnMaxLifetime = env.GetInt("DB_CONN_MAX_LIFETIME", cfg.Database.PostgreSQL.ConnMaxLifetime)

	// 日志配置
	cfg.Log.Level = env.GetString("LOG_LEVEL", cfg.Log.Level)
	cfg.Log.Format = env.GetString("LOG_FORMAT", cfg.Log.Format)
}

// GetEnvExample 获取环境变量示例
func GetEnvExample() map[string]string {
	return map[string]string{
		"DATAFUSION_SERVER_PORT":            "8080",
		"DATAFUSION_SERVER_MODE":            "debug",
		"DATAFUSION_JWT_SECRET_KEY":         "your-secret-key",
		"DATAFUSION_JWT_TOKEN_DURATION":     "24h",
		"DATAFUSION_DB_HOST":                "localhost",
		"DATAFUSION_DB_PORT":                "5432",
		"DATAFUSION_DB_USER":                "postgres",
		"DATAFUSION_DB_PASSWORD":            "postgres",
		"DATAFUSION_DB_NAME":                "datafusion_control",
		"DATAFUSION_LOG_LEVEL":              "info",
		"DATAFUSION_LOG_FORMAT":             "console",
		"DATAFUSION_PASSWORD_MIN_LENGTH":    "8",
		"DATAFUSION_PASSWORD_REQUIRE_UPPER": "true",
	}
}

// ValidateEnv 验证环境变量配置
func ValidateEnv() []string {
	var errors []string
	env := NewEnvConfig("DATAFUSION")

	// 检查必需的环境变量
	requiredVars := map[string]string{
		"JWT_SECRET_KEY": "JWT密钥不能为空",
		"DB_PASSWORD":    "数据库密码不能为空",
	}

	for key, message := range requiredVars {
		if env.GetString(key, "") == "" {
			errors = append(errors, message)
		}
	}

	// 检查端口范围
	port := env.GetInt("SERVER_PORT", 8080)
	if port < 1 || port > 65535 {
		errors = append(errors, "服务器端口必须在1-65535范围内")
	}

	// 检查JWT密钥长度
	jwtSecret := env.GetString("JWT_SECRET_KEY", "")
	if len(jwtSecret) < 32 {
		errors = append(errors, "JWT密钥长度至少32个字符")
	}

	return errors
}
