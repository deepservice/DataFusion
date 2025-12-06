package unit

import (
	"context"
	"testing"

	"github.com/datafusion/worker/internal/collector"
	"github.com/datafusion/worker/internal/models"
)

func TestAPICollector(t *testing.T) {
	t.Run("创建 API 采集器", func(t *testing.T) {
		c := collector.NewAPICollector(30)
		if c == nil {
			t.Fatal("创建 API 采集器失败")
		}
		if c.Type() != "api" {
			t.Errorf("采集器类型错误，期望 'api'，得到 '%s'", c.Type())
		}
	})

	t.Run("API 采集配置验证", func(t *testing.T) {
		c := collector.NewAPICollector(30)
		config := &models.DataSourceConfig{
			Type:   "api",
			URL:    "",
			Method: "GET",
		}

		ctx := context.Background()
		_, err := c.Collect(ctx, config)
		if err == nil {
			t.Error("应该返回 URL 为空的错误")
		}
	})
}

func TestRPACollector(t *testing.T) {
	t.Run("创建 RPA 采集器", func(t *testing.T) {
		c := collector.NewRPACollector(true, 60)
		if c == nil {
			t.Fatal("创建 RPA 采集器失败")
		}
		if c.Type() != "web-rpa" {
			t.Errorf("采集器类型错误，期望 'web-rpa'，得到 '%s'", c.Type())
		}
	})
}

func TestDBCollector(t *testing.T) {
	t.Run("创建数据库采集器", func(t *testing.T) {
		c := collector.NewDBCollector(30)
		if c == nil {
			t.Fatal("创建数据库采集器失败")
		}
		if c.Type() != "database" {
			t.Errorf("采集器类型错误，期望 'database'，得到 '%s'", c.Type())
		}
	})

	t.Run("数据库配置验证", func(t *testing.T) {
		c := collector.NewDBCollector(30)
		config := &models.DataSourceConfig{
			Type:     "database",
			DBConfig: nil,
		}

		ctx := context.Background()
		_, err := c.Collect(ctx, config)
		if err == nil {
			t.Error("应该返回数据库配置为空的错误")
		}
	})
}

func TestCollectorFactory(t *testing.T) {
	t.Run("创建采集器工厂", func(t *testing.T) {
		factory := collector.NewCollectorFactory()
		if factory == nil {
			t.Fatal("创建采集器工厂失败")
		}
	})

	t.Run("注册和获取采集器", func(t *testing.T) {
		factory := collector.NewCollectorFactory()
		
		// 注册 API 采集器
		apiCollector := collector.NewAPICollector(30)
		factory.Register(apiCollector)

		// 获取采集器
		c, ok := factory.Get("api")
		if !ok {
			t.Error("获取 API 采集器失败")
		}
		if c.Type() != "api" {
			t.Errorf("采集器类型错误，期望 'api'，得到 '%s'", c.Type())
		}
	})

	t.Run("获取不存在的采集器", func(t *testing.T) {
		factory := collector.NewCollectorFactory()
		
		_, ok := factory.Get("nonexistent")
		if ok {
			t.Error("不应该获取到不存在的采集器")
		}
	})
}
