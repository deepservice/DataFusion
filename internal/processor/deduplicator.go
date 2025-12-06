package processor

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// DeduplicationStrategy 去重策略
type DeduplicationStrategy string

const (
	// StrategyContentHash 基于内容哈希去重
	StrategyContentHash DeduplicationStrategy = "content_hash"
	// StrategyFieldBased 基于字段去重
	StrategyFieldBased DeduplicationStrategy = "field_based"
	// StrategyTimeWindow 基于时间窗口去重
	StrategyTimeWindow DeduplicationStrategy = "time_window"
)

// DeduplicatorConfig 去重器配置
type DeduplicatorConfig struct {
	Strategy      DeduplicationStrategy `json:"strategy"`
	Fields        []string              `json:"fields"`         // 用于 field_based 策略
	TimeWindow    time.Duration         `json:"time_window"`    // 用于 time_window 策略
	CacheSize     int                   `json:"cache_size"`     // 缓存大小
	EnableLogging bool                  `json:"enable_logging"` // 是否记录去重日志
}

// Deduplicator 数据去重器
type Deduplicator struct {
	config    *DeduplicatorConfig
	cache     map[string]time.Time // 哈希 -> 时间戳
	mu        sync.RWMutex
	stats     *DeduplicationStats
	cleanupCh chan struct{}
}

// DeduplicationStats 去重统计
type DeduplicationStats struct {
	TotalProcessed int64
	Duplicates     int64
	Unique         int64
	mu             sync.RWMutex
}

// NewDeduplicator 创建去重器
func NewDeduplicator(config *DeduplicatorConfig) *Deduplicator {
	if config.CacheSize == 0 {
		config.CacheSize = 10000 // 默认缓存 10000 条
	}
	if config.TimeWindow == 0 {
		config.TimeWindow = 24 * time.Hour // 默认 24 小时
	}

	d := &Deduplicator{
		config:    config,
		cache:     make(map[string]time.Time),
		stats:     &DeduplicationStats{},
		cleanupCh: make(chan struct{}),
	}

	// 启动定期清理
	go d.startCleanup()

	return d
}

// Deduplicate 执行去重
func (d *Deduplicator) Deduplicate(data []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(data) == 0 {
		return data, nil
	}

	if d.config.EnableLogging {
		log.Printf("开始去重，输入数据 %d 条", len(data))
	}

	var result []map[string]interface{}
	duplicateCount := 0

	for _, item := range data {
		d.stats.IncrementTotal()

		isDuplicate, err := d.isDuplicate(item)
		if err != nil {
			return nil, fmt.Errorf("检查重复失败: %w", err)
		}

		if isDuplicate {
			duplicateCount++
			d.stats.IncrementDuplicates()
			continue
		}

		result = append(result, item)
		d.stats.IncrementUnique()
	}

	if d.config.EnableLogging {
		log.Printf("去重完成，输出数据 %d 条，去除重复 %d 条", len(result), duplicateCount)
	}

	return result, nil
}

// isDuplicate 检查是否重复
func (d *Deduplicator) isDuplicate(item map[string]interface{}) (bool, error) {
	var key string
	var err error

	switch d.config.Strategy {
	case StrategyContentHash:
		key, err = d.generateContentHash(item)
	case StrategyFieldBased:
		key, err = d.generateFieldHash(item)
	case StrategyTimeWindow:
		key, err = d.generateContentHash(item)
	default:
		return false, fmt.Errorf("未知的去重策略: %s", d.config.Strategy)
	}

	if err != nil {
		return false, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查缓存
	if timestamp, exists := d.cache[key]; exists {
		// 对于时间窗口策略，检查时间
		if d.config.Strategy == StrategyTimeWindow {
			if time.Since(timestamp) < d.config.TimeWindow {
				return true, nil
			}
			// 超过时间窗口，删除旧记录
			delete(d.cache, key)
		} else {
			return true, nil
		}
	}

	// 添加到缓存
	d.cache[key] = time.Now()

	// 检查缓存大小
	if len(d.cache) > d.config.CacheSize {
		d.evictOldest()
	}

	return false, nil
}

// generateContentHash 生成内容哈希
func (d *Deduplicator) generateContentHash(item map[string]interface{}) (string, error) {
	// 序列化为 JSON
	jsonData, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %w", err)
	}

	// 计算 MD5 哈希
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%x", hash), nil
}

// generateFieldHash 基于指定字段生成哈希
func (d *Deduplicator) generateFieldHash(item map[string]interface{}) (string, error) {
	if len(d.config.Fields) == 0 {
		return "", fmt.Errorf("field_based 策略需要指定字段")
	}

	// 提取指定字段
	fieldData := make(map[string]interface{})
	for _, field := range d.config.Fields {
		if value, exists := item[field]; exists {
			fieldData[field] = value
		}
	}

	// 序列化并计算哈希
	jsonData, err := json.Marshal(fieldData)
	if err != nil {
		return "", fmt.Errorf("序列化字段数据失败: %w", err)
	}

	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%x", hash), nil
}

// evictOldest 清除最旧的缓存项
func (d *Deduplicator) evictOldest() {
	// 找到最旧的项
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, timestamp := range d.cache {
		if first || timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(d.cache, oldestKey)
	}
}

// startCleanup 启动定期清理
func (d *Deduplicator) startCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.cleanup()
		case <-d.cleanupCh:
			return
		}
	}
}

// cleanup 清理过期缓存
func (d *Deduplicator) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// 找出过期的键
	for key, timestamp := range d.cache {
		if now.Sub(timestamp) > d.config.TimeWindow {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 删除过期键
	for _, key := range expiredKeys {
		delete(d.cache, key)
	}

	if len(expiredKeys) > 0 && d.config.EnableLogging {
		log.Printf("清理过期缓存 %d 条", len(expiredKeys))
	}
}

// GetStats 获取统计信息
func (d *Deduplicator) GetStats() DeduplicationStats {
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()

	return DeduplicationStats{
		TotalProcessed: d.stats.TotalProcessed,
		Duplicates:     d.stats.Duplicates,
		Unique:         d.stats.Unique,
	}
}

// ResetStats 重置统计信息
func (d *Deduplicator) ResetStats() {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()

	d.stats.TotalProcessed = 0
	d.stats.Duplicates = 0
	d.stats.Unique = 0
}

// ClearCache 清空缓存
func (d *Deduplicator) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache = make(map[string]time.Time)
	if d.config.EnableLogging {
		log.Println("已清空去重缓存")
	}
}

// Close 关闭去重器
func (d *Deduplicator) Close() {
	close(d.cleanupCh)
}

// IncrementTotal 增加总处理数
func (s *DeduplicationStats) IncrementTotal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalProcessed++
}

// IncrementDuplicates 增加重复数
func (s *DeduplicationStats) IncrementDuplicates() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Duplicates++
}

// IncrementUnique 增加唯一数
func (s *DeduplicationStats) IncrementUnique() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Unique++
}

// GetDuplicationRate 获取重复率
func (s *DeduplicationStats) GetDuplicationRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalProcessed == 0 {
		return 0
	}
	return float64(s.Duplicates) / float64(s.TotalProcessed) * 100
}
