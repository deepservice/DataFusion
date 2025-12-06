package mongodb

import (
	"fmt"
	"time"
)

// Config MongoDB 配置
type Config struct {
	URI            string        `json:"uri"`
	Database       string        `json:"database"`
	Collection     string        `json:"collection"`
	Timeout        time.Duration `json:"timeout"`
	MaxPoolSize    uint64        `json:"max_pool_size"`
	MinPoolSize    uint64        `json:"min_pool_size"`
	MaxConnIdleTime time.Duration `json:"max_conn_idle_time"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		URI:             "mongodb://localhost:27017",
		Database:        "datafusion",
		Collection:      "collected_data",
		Timeout:         30 * time.Second,
		MaxPoolSize:     100,
		MinPoolSize:     10,
		MaxConnIdleTime: 5 * time.Minute,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("MongoDB URI 不能为空")
	}
	if c.Database == "" {
		return fmt.Errorf("数据库名不能为空")
	}
	if c.Collection == "" {
		return fmt.Errorf("集合名不能为空")
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxPoolSize == 0 {
		c.MaxPoolSize = 100
	}
	if c.MinPoolSize == 0 {
		c.MinPoolSize = 10
	}
	return nil
}
