package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/datafusion/worker/internal/models"
	"github.com/tidwall/gjson"
)

// APICollector API 采集器
type APICollector struct {
	client *resty.Client
}

// NewAPICollector 创建 API 采集器
func NewAPICollector(timeout int) *APICollector {
	client := resty.New()
	client.SetTimeout(time.Duration(timeout) * time.Second)
	return &APICollector{client: client}
}

// Type 返回采集器类型
func (a *APICollector) Type() string {
	return "api"
}

// Collect 执行数据采集
func (a *APICollector) Collect(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error) {
	log.Printf("开始 API 采集: %s", config.URL)

	// 构建请求
	req := a.client.R().SetContext(ctx)

	// 设置请求头
	if config.Headers != nil {
		req.SetHeaders(config.Headers)
	}

	// 发送请求
	var resp *resty.Response
	var err error

	switch config.Method {
	case "POST":
		resp, err = req.Post(config.URL)
	case "GET":
		fallthrough
	default:
		resp, err = req.Get(config.URL)
	}

	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API 返回错误状态码: %d", resp.StatusCode())
	}

	log.Printf("API 请求成功，状态码: %d，响应大小: %d bytes", resp.StatusCode(), len(resp.Body()))

	// 解析 JSON 响应
	return a.parseJSON(resp.Body(), config.Selectors)
}

// parseJSON 解析 JSON 响应
func (a *APICollector) parseJSON(body []byte, selectors map[string]string) ([]map[string]interface{}, error) {
	// 获取数据路径
	dataPath, ok := selectors["_data_path"]
	if !ok || dataPath == "" {
		// 如果没有指定路径，直接解析整个 JSON
		dataPath = "@this"
	}

	// 使用 gjson 提取数据
	result := gjson.GetBytes(body, dataPath)
	if !result.Exists() {
		return nil, fmt.Errorf("数据路径 %s 不存在", dataPath)
	}

	var results []map[string]interface{}

	// 如果是数组
	if result.IsArray() {
		for _, item := range result.Array() {
			record := make(map[string]interface{})
			for field, path := range selectors {
				if field == "_data_path" {
					continue
				}
				value := item.Get(path)
				if value.Exists() {
					record[field] = value.Value()
				}
			}
			results = append(results, record)
		}
	} else {
		// 单条数据
		record := make(map[string]interface{})
		for field, path := range selectors {
			if field == "_data_path" {
				continue
			}
			value := result.Get(path)
			if value.Exists() {
				record[field] = value.Value()
			}
		}
		results = append(results, record)
	}

	log.Printf("解析完成，提取到 %d 条数据", len(results))
	return results, nil
}
