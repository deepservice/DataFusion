package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// APIServerConfig API服务器配置
type APIServerConfig struct {
	Server   ServerConfig `yaml:"server"`
	Auth     AuthConfig   `yaml:"auth"`
	Database DBConfig     `yaml:"database"`
	Cache    CacheConfig  `yaml:"cache"`
	Log      LogConfig    `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `yaml:"port"`
	Mode         string `yaml:"mode"` // debug, release
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWT      JWTConfig      `yaml:"jwt"`
	Password PasswordConfig `yaml:"password"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey     string `yaml:"secret_key"`
	TokenDuration string `yaml:"token_duration"`
}

// PasswordConfig 密码策略配置
type PasswordConfig struct {
	MinLength      int  `yaml:"min_length"`
	RequireUpper   bool `yaml:"require_upper"`
	RequireLower   bool `yaml:"require_lower"`
	RequireDigit   bool `yaml:"require_digit"`
	RequireSpecial bool `yaml:"require_special"`
}

// DBConfig 数据库配置
type DBConfig struct {
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	SSLMode         string `yaml:"sslmode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, console
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type   string       `yaml:"type"` // redis, memory, hybrid
	Redis  RedisConfig  `yaml:"redis"`
	Memory MemoryConfig `yaml:"memory"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
	CleanupInterval string `yaml:"cleanup_interval"`
}

// GetJWTDuration 获取JWT Token持续时间
func (c *AuthConfig) GetJWTDuration() time.Duration {
	if c.JWT.TokenDuration == "" {
		return 24 * time.Hour // 默认24小时
	}

	duration, err := time.ParseDuration(c.JWT.TokenDuration)
	if err != nil {
		return 24 * time.Hour // 解析失败时使用默认值
	}

	return duration
}

// LoadAPIServerConfig 加载API服务器配置
func LoadAPIServerConfig(path string) (*APIServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg APIServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "console"
	}

	// 认证配置默认值
	if cfg.Auth.JWT.SecretKey == "" {
		cfg.Auth.JWT.SecretKey = "datafusion-default-secret-change-in-production"
	}
	if cfg.Auth.JWT.TokenDuration == "" {
		cfg.Auth.JWT.TokenDuration = "24h"
	}
	if cfg.Auth.Password.MinLength == 0 {
		cfg.Auth.Password.MinLength = 8
	}

	// 缓存配置默认值
	if cfg.Cache.Type == "" {
		cfg.Cache.Type = "memory"
	}
	if cfg.Cache.Redis.Host == "" {
		cfg.Cache.Redis.Host = "localhost"
	}
	if cfg.Cache.Redis.Port == 0 {
		cfg.Cache.Redis.Port = 6379
	}
	if cfg.Cache.Redis.PoolSize == 0 {
		cfg.Cache.Redis.PoolSize = 10
	}
	if cfg.Cache.Memory.CleanupInterval == "" {
		cfg.Cache.Memory.CleanupInterval = "10m"
	}

	return &cfg, nil
}
