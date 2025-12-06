package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/datafusion/worker/internal/collector"
	"github.com/datafusion/worker/internal/models"
	"github.com/datafusion/worker/internal/processor"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("测试数据库采集器和增强清洗功能")
	fmt.Println("==========================================")
	fmt.Println()

	// 测试 MySQL 采集
	testMySQLCollection()

	// 测试 PostgreSQL 采集
	testPostgreSQLCollection()

	// 测试增强清洗规则
	testEnhancedCleaning()

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("✅ 测试完成")
	fmt.Println("==========================================")
}

func testMySQLCollection() {
	fmt.Println("1. 测试 MySQL 数据采集")
	fmt.Println("------------------------------------------")

	// 创建数据库采集器
	dbCollector := collector.NewDBCollector(30)

	// 配置 MySQL 数据源
	config := &models.DataSourceConfig{
		Type: "database",
		DBConfig: &models.DBConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "test_user",
			Password: "test_password",
			Database: "test_db",
			Query:    "SELECT id, name, email FROM users LIMIT 10",
		},
	}

	// 执行采集
	ctx := context.Background()
	data, err := dbCollector.Collect(ctx, config)
	if err != nil {
		fmt.Printf("❌ MySQL 采集失败: %v\n", err)
		fmt.Println("   (这是预期的，因为没有实际的 MySQL 服务器)")
	} else {
		fmt.Printf("✅ MySQL 采集成功，获取 %d 条数据\n", len(data))
		if len(data) > 0 {
			jsonData, _ := json.MarshalIndent(data[0], "", "  ")
			fmt.Printf("   示例数据: %s\n", string(jsonData))
		}
	}

	fmt.Println()
}

func testPostgreSQLCollection() {
	fmt.Println("2. 测试 PostgreSQL 数据采集")
	fmt.Println("------------------------------------------")

	// 创建数据库采集器
	dbCollector := collector.NewDBCollector(30)

	// 配置 PostgreSQL 数据源
	config := &models.DataSourceConfig{
		Type: "database",
		DBConfig: &models.DBConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "datafusion",
			Query:    "SELECT id, name, status FROM collection_tasks LIMIT 5",
		},
	}

	// 执行采集
	ctx := context.Background()
	data, err := dbCollector.Collect(ctx, config)
	if err != nil {
		fmt.Printf("❌ PostgreSQL 采集失败: %v\n", err)
		fmt.Println("   (如果数据库配置正确，应该能成功)")
	} else {
		fmt.Printf("✅ PostgreSQL 采集成功，获取 %d 条数据\n", len(data))
		if len(data) > 0 {
			jsonData, _ := json.MarshalIndent(data[0], "", "  ")
			fmt.Printf("   示例数据: %s\n", string(jsonData))
		}
	}

	fmt.Println()
}

func testEnhancedCleaning() {
	fmt.Println("3. 测试增强清洗规则")
	fmt.Println("------------------------------------------")

	// 准备测试数据
	testData := []map[string]interface{}{
		{
			"name":        "  John Doe  ",
			"email":       "JOHN.DOE@EXAMPLE.COM",
			"phone":       "13812345678",
			"price":       "1,234.56",
			"description": "<p>This is a <b>test</b> product</p>",
			"website":     "www.example.com",
			"date":        "2024-12-04 10:30:00",
		},
	}

	// 定义清洗规则
	rules := []models.CleaningRule{
		{
			Name:  "trim_name",
			Field: "name",
			Type:  "trim",
		},
		{
			Name:  "validate_email",
			Field: "email",
			Type:  "email_validate",
		},
		{
			Name:  "format_phone",
			Field: "phone",
			Type:  "phone_format",
		},
		{
			Name:  "format_price",
			Field: "price",
			Type:  "number_format",
		},
		{
			Name:  "remove_html",
			Field: "description",
			Type:  "remove_html",
		},
		{
			Name:  "normalize_whitespace",
			Field: "description",
			Type:  "normalize_whitespace",
		},
		{
			Name:  "normalize_url",
			Field: "website",
			Type:  "url_normalize",
		},
		{
			Name:    "format_date",
			Field:   "date",
			Type:    "date_format",
			Pattern: "2006-01-02",
		},
	}

	// 创建处理器配置
	processorConfig := &models.ProcessorConfig{
		CleaningRules: rules,
	}

	// 创建处理器
	proc := processor.NewProcessor(processorConfig)

	// 执行清洗
	cleaned, err := proc.Process(testData)
	if err != nil {
		fmt.Printf("❌ 数据清洗失败: %v\n", err)
	} else {
		fmt.Println("✅ 数据清洗成功")
		fmt.Println("\n原始数据:")
		jsonData, _ := json.MarshalIndent(testData[0], "", "  ")
		fmt.Println(string(jsonData))
		
		fmt.Println("\n清洗后数据:")
		jsonData, _ = json.MarshalIndent(cleaned[0], "", "  ")
		fmt.Println(string(jsonData))
	}

	fmt.Println()
}
