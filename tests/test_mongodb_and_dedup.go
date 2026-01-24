package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/datafusion/worker/internal/processor"
	"github.com/datafusion/worker/internal/storage/mongodb"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("测试 MongoDB 存储和数据去重功能")
	fmt.Println("==========================================")
	fmt.Println()

	// 测试 MongoDB 存储
	testMongoDBStorage()

	// 测试数据去重
	testDeduplication()

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("✅ 测试完成")
	fmt.Println("==========================================")
}

func testMongoDBStorage() {
	fmt.Println("1. 测试 MongoDB 存储")
	fmt.Println("------------------------------------------")

	// 创建配置
	config := &mongodb.Config{
		URI:        "mongodb://localhost:27017",
		Database:   "datafusion_test",
		Collection: "test_data",
		Timeout:    30 * time.Second,
	}

	// 创建存储
	storage, err := mongodb.NewMongoDBStorage(config)
	if err != nil {
		fmt.Printf("❌ 创建 MongoDB 存储失败: %v\n", err)
		fmt.Println("   (这是预期的，因为没有实际的 MongoDB 服务器)")
		fmt.Println()
		return
	}
	defer storage.Close()

	// 准备测试数据
	testData := []map[string]interface{}{
		{
			"id":    1,
			"name":  "Test User 1",
			"email": "user1@example.com",
		},
		{
			"id":    2,
			"name":  "Test User 2",
			"email": "user2@example.com",
		},
	}

	// 存储数据
	ctx := context.Background()
	err = storage.Store(ctx, testData)
	if err != nil {
		fmt.Printf("❌ 存储数据失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功存储 %d 条数据\n", len(testData))
	}

	// 查询数据
	results, err := storage.Query(ctx, map[string]interface{}{}, 10)
	if err != nil {
		fmt.Printf("❌ 查询数据失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询到 %d 条数据\n", len(results))
		if len(results) > 0 {
			jsonData, _ := json.MarshalIndent(results[0], "", "  ")
			fmt.Printf("   示例数据: %s\n", string(jsonData))
		}
	}

	// 统计数据
	count, err := storage.Count(ctx, map[string]interface{}{})
	if err != nil {
		fmt.Printf("❌ 统计数据失败: %v\n", err)
	} else {
		fmt.Printf("✅ 总数据量: %d 条\n", count)
	}

	fmt.Println()
}

func testDeduplication() {
	fmt.Println("2. 测试数据去重")
	fmt.Println("------------------------------------------")

	// 测试内容哈希去重
	testContentHashDedup()

	// 测试字段去重
	testFieldBasedDedup()

	// 测试时间窗口去重
	testTimeWindowDedup()
}

func testContentHashDedup() {
	fmt.Println("2.1 测试内容哈希去重")
	fmt.Println("   ------------------------------------------")

	config := &processor.DeduplicatorConfig{
		Strategy:      processor.StrategyContentHash,
		CacheSize:     1000,
		EnableLogging: true,
	}

	dedup := processor.NewDeduplicator(config)
	defer dedup.Close()

	// 准备测试数据（包含重复）
	testData := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
		{"id": 1, "name": "Alice"}, // 重复
		{"id": 3, "name": "Charlie"},
		{"id": 2, "name": "Bob"}, // 重复
	}

	// 执行去重
	result, err := dedup.Deduplicate(testData)
	if err != nil {
		fmt.Printf("   ❌ 去重失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 去重成功\n")
		fmt.Printf("   输入: %d 条，输出: %d 条，去除重复: %d 条\n",
			len(testData), len(result), len(testData)-len(result))

		stats := dedup.GetStats()
		fmt.Printf("   统计: 总处理 %d，重复 %d，唯一 %d，重复率 %.2f%%\n",
			stats.TotalProcessed, stats.Duplicates, stats.Unique, stats.GetDuplicationRate())
	}

	fmt.Println()
}

func testFieldBasedDedup() {
	fmt.Println("2.2 测试字段去重")
	fmt.Println("   ------------------------------------------")

	config := &processor.DeduplicatorConfig{
		Strategy:      processor.StrategyFieldBased,
		Fields:        []string{"email"},
		CacheSize:     1000,
		EnableLogging: true,
	}

	dedup := processor.NewDeduplicator(config)
	defer dedup.Close()

	// 准备测试数据（相同邮箱但其他字段不同）
	testData := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
		{"id": 3, "name": "Alice Smith", "email": "alice@example.com"}, // 邮箱重复
		{"id": 4, "name": "Charlie", "email": "charlie@example.com"},
	}

	// 执行去重
	result, err := dedup.Deduplicate(testData)
	if err != nil {
		fmt.Printf("   ❌ 去重失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 去重成功\n")
		fmt.Printf("   输入: %d 条，输出: %d 条，去除重复: %d 条\n",
			len(testData), len(result), len(testData)-len(result))

		stats := dedup.GetStats()
		fmt.Printf("   统计: 总处理 %d，重复 %d，唯一 %d，重复率 %.2f%%\n",
			stats.TotalProcessed, stats.Duplicates, stats.Unique, stats.GetDuplicationRate())
	}

	fmt.Println()
}

func testTimeWindowDedup() {
	fmt.Println("2.3 测试时间窗口去重")
	fmt.Println("   ------------------------------------------")

	config := &processor.DeduplicatorConfig{
		Strategy:      processor.StrategyTimeWindow,
		TimeWindow:    2 * time.Second, // 2 秒窗口
		CacheSize:     1000,
		EnableLogging: true,
	}

	dedup := processor.NewDeduplicator(config)
	defer dedup.Close()

	// 准备测试数据
	testData := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
	}

	// 第一次处理
	result1, _ := dedup.Deduplicate(testData)
	fmt.Printf("   第一次处理: 输入 %d 条，输出 %d 条\n", len(testData), len(result1))

	// 立即再次处理（应该被去重）
	result2, _ := dedup.Deduplicate(testData)
	fmt.Printf("   立即再次处理: 输入 %d 条，输出 %d 条（应该被去重）\n", len(testData), len(result2))

	// 等待超过时间窗口
	fmt.Println("   等待 3 秒...")
	time.Sleep(3 * time.Second)

	// 再次处理（应该不被去重）
	result3, _ := dedup.Deduplicate(testData)
	fmt.Printf("   3 秒后再次处理: 输入 %d 条，输出 %d 条（应该不被去重）\n", len(testData), len(result3))

	stats := dedup.GetStats()
	fmt.Printf("   ✅ 时间窗口去重测试完成\n")
	fmt.Printf("   统计: 总处理 %d，重复 %d，唯一 %d\n",
		stats.TotalProcessed, stats.Duplicates, stats.Unique)

	fmt.Println()
}
