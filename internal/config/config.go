package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config Worker 配置
type Config struct {
	WorkerType   string        `yaml:"worker_type"`   // rpa-collector, api-collector, db-collector
	PollInterval time.Duration `yaml:"poll_interval"` // 轮询间隔
	Database     DatabaseConfig `yaml:"database"`
	Collector    CollectorConfig `yaml:"collector"`
	Storage      StorageConfig `yaml:"storage"`
}

// DatabaseConfig 数据库配置（PostgreSQL）
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	RPA RPAConfig `yaml:"rpa"`
	API APIConfig `yaml:"api"`
}

// RPAConfig RPA 采集器配置
type RPAConfig struct {
	BrowserType string `yaml:"browser_type"` // chromium, firefox
	Headless    bool   `yaml:"headless"`
	Timeout     int    `yaml:"timeout"` // 秒
}

// APIConfig API 采集器配置
type APIConfig struct {
	Timeout int `yaml:"timeout"` // 秒
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type     string         `yaml:"type"` // postgresql, mongodb, file
	Database DatabaseConfig `yaml:"database"`
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if cfg.Collector.RPA.Timeout == 0 {
		cfg.Collector.RPA.Timeout = 30
	}
	if cfg.Collector.API.Timeout == 0 {
		cfg.Collector.API.Timeout = 30
	}

	return &cfg, nil
}
