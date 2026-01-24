package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/datafusion/worker/internal/collector"
	"github.com/datafusion/worker/internal/models"
	"github.com/datafusion/worker/internal/processor"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DataFusion Worker 简单验证测试")
	fmt.Println("========================================")
	fmt.Println()

	// 测试 API 采集器
	testAPICollector()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ 测试完成！")
	fmt.Println("========================================")
}

func testAPICollector() {
	fmt.Println("测试 1: API 采集器")
	fmt.Println("------------------------------------------")

	// 创建 API 采集器
	apiCollector := collector.NewAPICollector(30)

	// 配置数据源
	config := &models.DataSourceConfig{
		Type:    "api",
		URL:     "https://jsonplaceholder.typicode.com/users?_limit=3",
		Method:  "GET",
		Headers: map[string]string{},
		Selectors: map[string]string{
			"_data_path": "",
			"id":         "id",
			"name":       "name",
			"email":      "email",
			"username":   "username",
		},
	}

	fmt.Printf("📡 正在采集数据: %s\n", config.URL)

	// 执行采集
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := apiCollector.Collect(ctx, config)
	if err != nil {
		log.Fatalf("❌ 采集失败: %v", err)
	}

	fmt.Printf("✅ 采集成功！获取到 %d 条数据\n\n", len(data))

	// 显示原始数据
	fmt.Println("📋 原始数据:")
	for i, record := range data {
		fmt.Printf("  [%d] %v\n", i+1, record)
	}

	// 测试数据处理
	fmt.Println("\n📝 应用数据清洗规则...")

	processorConfig := &models.ProcessorConfig{
		CleaningRules: []models.CleaningRule{
			{Field: "name", Type: "trim"},
			{Field: "email", Type: "email_validate"},
		},
	}

	proc := processor.NewProcessor(processorConfig)
	processedData, err := proc.Process(data)
	if err != nil {
		log.Fatalf("❌ 处理失败: %v", err)
	}

	fmt.Printf("✅ 处理完成！有效数据 %d 条\n\n", len(processedData))

	// 显示处理后的数据
	fmt.Println("📋 处理后的数据:")
	for i, record := range processedData {
		jsonData, _ := json.MarshalIndent(record, "  ", "  ")
		fmt.Printf("  [%d] %s\n", i+1, string(jsonData))
	}
}
