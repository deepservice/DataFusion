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
	"github.com/datafusion/worker/internal/storage"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("DataFusion Worker 完整流程测试")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 数据采集
	fmt.Println("步骤 1: 数据采集")
	fmt.Println("------------------------------------------")
	
	apiCollector := collector.NewAPICollector(30)
	
	dataSourceConfig := &models.DataSourceConfig{
		Type:   "api",
		URL:    "https://jsonplaceholder.typicode.com/posts?_limit=5",
		Method: "GET",
		Selectors: map[string]string{
			"id":     "id",
			"title":  "title",
			"body":   "body",
			"userId": "userId",
		},
	}
	
	fmt.Printf("📡 采集数据: %s\n", dataSourceConfig.URL)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	data, err := apiCollector.Collect(ctx, dataSourceConfig)
	if err != nil {
		log.Fatalf("❌ 采集失败: %v", err)
	}
	
	fmt.Printf("✅ 采集成功！获取到 %d 条数据\n\n", len(data))

	// 2. 数据处理
	fmt.Println("步骤 2: 数据处理")
	fmt.Println("------------------------------------------")
	
	processorConfig := &models.ProcessorConfig{
		CleaningRules: []models.CleaningRule{
			{Field: "title", Type: "trim"},
			{Field: "body", Type: "trim"},
		},
	}
	
	proc := processor.NewProcessor(processorConfig)
	processedData, err := proc.Process(data)
	if err != nil {
		log.Fatalf("❌ 处理失败: %v", err)
	}
	
	fmt.Printf("✅ 处理完成！有效数据 %d 条\n\n", len(processedData))

	// 3. 数据存储
	fmt.Println("步骤 3: 数据存储")
	fmt.Println("------------------------------------------")
	
	fileStorage := storage.NewFileStorage("./data")
	
	storageConfig := &models.StorageConfig{
		Target:   "file",
		Database: "test_output",
		Table:    "posts",
	}
	
	err = fileStorage.Store(ctx, storageConfig, processedData)
	if err != nil {
		log.Fatalf("❌ 存储失败: %v", err)
	}
	
	fmt.Println("✅ 数据已保存到文件")
	fmt.Println()

	// 4. 显示结果
	fmt.Println("步骤 4: 查看结果")
	fmt.Println("------------------------------------------")
	fmt.Println("📁 数据文件位置: ./data/test_output/posts_*.json")
	fmt.Println()
	fmt.Println("查看数据:")
	fmt.Println("  cat data/test_output/posts_*.json | jq .")
	fmt.Println()
	
	// 显示前 2 条数据
	fmt.Println("📋 前 2 条数据预览:")
	for i := 0; i < 2 && i < len(processedData); i++ {
		jsonData, _ := json.MarshalIndent(processedData[i], "  ", "  ")
		fmt.Printf("  [%d] %s\n", i+1, string(jsonData))
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ 完整流程测试成功！")
	fmt.Println("========================================")
}
