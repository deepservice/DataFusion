package unit

import (
	"testing"
	"time"

	"github.com/datafusion/worker/internal/models"
	"github.com/datafusion/worker/internal/processor"
)

func TestEnhancedCleaner(t *testing.T) {
	t.Run("Trim 清洗", func(t *testing.T) {
		rules := []models.CleaningRule{
			{
				Name:  "trim_name",
				Field: "name",
				Type:  "trim",
			},
		}

		cleaner := processor.NewEnhancedCleaner(rules)
		data := []map[string]interface{}{
			{"name": "  Alice  "},
		}

		result, err := cleaner.Clean(data)
		if err != nil {
			t.Fatalf("清洗失败: %v", err)
		}

		if result[0]["name"] != "Alice" {
			t.Errorf("Trim 清洗失败，期望 'Alice'，得到 '%v'", result[0]["name"])
		}
	})

	t.Run("移除 HTML 标签", func(t *testing.T) {
		rules := []models.CleaningRule{
			{
				Name:  "remove_html",
				Field: "content",
				Type:  "remove_html",
			},
		}

		cleaner := processor.NewEnhancedCleaner(rules)
		data := []map[string]interface{}{
			{"content": "<p>Hello <b>World</b></p>"},
		}

		result, err := cleaner.Clean(data)
		if err != nil {
			t.Fatalf("清洗失败: %v", err)
		}

		if result[0]["content"] != "Hello World" {
			t.Errorf("HTML 清洗失败，期望 'Hello World'，得到 '%v'", result[0]["content"])
		}
	})

	t.Run("数字格式化", func(t *testing.T) {
		rules := []models.CleaningRule{
			{
				Name:  "format_price",
				Field: "price",
				Type:  "number_format",
			},
		}

		cleaner := processor.NewEnhancedCleaner(rules)
		data := []map[string]interface{}{
			{"price": "1,234.56"},
		}

		result, err := cleaner.Clean(data)
		if err != nil {
			t.Fatalf("清洗失败: %v", err)
		}

		price, ok := result[0]["price"].(float64)
		if !ok {
			t.Errorf("数字格式化失败，结果不是 float64 类型")
		}
		if price != 1234.56 {
			t.Errorf("数字格式化失败，期望 1234.56，得到 %v", price)
		}
	})

	t.Run("URL 规范化", func(t *testing.T) {
		rules := []models.CleaningRule{
			{
				Name:  "normalize_url",
				Field: "website",
				Type:  "url_normalize",
			},
		}

		cleaner := processor.NewEnhancedCleaner(rules)
		data := []map[string]interface{}{
			{"website": "example.com"},
		}

		result, err := cleaner.Clean(data)
		if err != nil {
			t.Fatalf("清洗失败: %v", err)
		}

		if result[0]["website"] != "https://example.com" {
			t.Errorf("URL 规范化失败，期望 'https://example.com'，得到 '%v'", result[0]["website"])
		}
	})
}

func TestDeduplicator(t *testing.T) {
	t.Run("内容哈希去重", func(t *testing.T) {
		config := &processor.DeduplicatorConfig{
			Strategy:      processor.StrategyContentHash,
			CacheSize:     1000,
			EnableLogging: false,
		}

		dedup := processor.NewDeduplicator(config)
		defer dedup.Close()

		data := []map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
			{"id": 1, "name": "Alice"}, // 重复
		}

		result, err := dedup.Deduplicate(data)
		if err != nil {
			t.Fatalf("去重失败: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("去重失败，期望 2 条记录，得到 %d 条", len(result))
		}

		stats := dedup.GetStats()
		if stats.Duplicates != 1 {
			t.Errorf("重复统计错误，期望 1，得到 %d", stats.Duplicates)
		}
	})

	t.Run("字段去重", func(t *testing.T) {
		config := &processor.DeduplicatorConfig{
			Strategy:      processor.StrategyFieldBased,
			Fields:        []string{"email"},
			CacheSize:     1000,
			EnableLogging: false,
		}

		dedup := processor.NewDeduplicator(config)
		defer dedup.Close()

		data := []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Alice Smith", "email": "alice@example.com"}, // 邮箱重复
		}

		result, err := dedup.Deduplicate(data)
		if err != nil {
			t.Fatalf("去重失败: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("去重失败，期望 2 条记录，得到 %d 条", len(result))
		}
	})

	t.Run("时间窗口去重", func(t *testing.T) {
		config := &processor.DeduplicatorConfig{
			Strategy:      processor.StrategyTimeWindow,
			TimeWindow:    1 * time.Second,
			CacheSize:     1000,
			EnableLogging: false,
		}

		dedup := processor.NewDeduplicator(config)
		defer dedup.Close()

		data := []map[string]interface{}{
			{"id": 1, "name": "Alice"},
		}

		// 第一次处理
		result1, _ := dedup.Deduplicate(data)
		if len(result1) != 1 {
			t.Errorf("第一次处理失败，期望 1 条记录，得到 %d 条", len(result1))
		}

		// 立即再次处理（应该被去重）
		result2, _ := dedup.Deduplicate(data)
		if len(result2) != 0 {
			t.Errorf("立即再次处理应该被去重，期望 0 条记录，得到 %d 条", len(result2))
		}

		// 等待超过时间窗口
		time.Sleep(1100 * time.Millisecond)

		// 再次处理（应该不被去重）
		result3, _ := dedup.Deduplicate(data)
		if len(result3) != 1 {
			t.Errorf("超过时间窗口后应该不被去重，期望 1 条记录，得到 %d 条", len(result3))
		}
	})

	t.Run("去重统计", func(t *testing.T) {
		config := &processor.DeduplicatorConfig{
			Strategy:      processor.StrategyContentHash,
			CacheSize:     1000,
			EnableLogging: false,
		}

		dedup := processor.NewDeduplicator(config)
		defer dedup.Close()

		data := []map[string]interface{}{
			{"id": 1},
			{"id": 2},
			{"id": 1}, // 重复
			{"id": 3},
		}

		dedup.Deduplicate(data)

		stats := dedup.GetStats()
		if stats.TotalProcessed != 4 {
			t.Errorf("总处理数错误，期望 4，得到 %d", stats.TotalProcessed)
		}
		if stats.Duplicates != 1 {
			t.Errorf("重复数错误，期望 1，得到 %d", stats.Duplicates)
		}
		if stats.Unique != 3 {
			t.Errorf("唯一数错误，期望 3，得到 %d", stats.Unique)
		}

		rate := stats.GetDuplicationRate()
		if rate != 25.0 {
			t.Errorf("重复率错误，期望 25.0，得到 %f", rate)
		}
	})
}

func TestProcessor(t *testing.T) {
	t.Run("创建处理器", func(t *testing.T) {
		config := &models.ProcessorConfig{
			CleaningRules: []models.CleaningRule{},
		}

		proc := processor.NewProcessor(config)
		if proc == nil {
			t.Fatal("创建处理器失败")
		}
	})

	t.Run("处理空数据", func(t *testing.T) {
		config := &models.ProcessorConfig{}
		proc := processor.NewProcessor(config)

		data := []map[string]interface{}{}
		result, err := proc.Process(data)
		if err != nil {
			t.Fatalf("处理失败: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("处理空数据失败，期望 0 条记录，得到 %d 条", len(result))
		}
	})
}
