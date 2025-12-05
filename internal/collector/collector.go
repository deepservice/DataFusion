package collector

import (
	"context"

	"github.com/datafusion/worker/internal/models"
)

// Collector 数据采集器接口
type Collector interface {
	// Collect 执行数据采集
	Collect(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error)
	
	// Type 返回采集器类型
	Type() string
}

// CollectorFactory 采集器工厂
type CollectorFactory struct {
	collectors map[string]Collector
}

// NewCollectorFactory 创建采集器工厂
func NewCollectorFactory() *CollectorFactory {
	return &CollectorFactory{
		collectors: make(map[string]Collector),
	}
}

// Register 注册采集器
func (f *CollectorFactory) Register(collector Collector) {
	f.collectors[collector.Type()] = collector
}

// Get 获取采集器
func (f *CollectorFactory) Get(collectorType string) (Collector, bool) {
	collector, ok := f.collectors[collectorType]
	return collector, ok
}
