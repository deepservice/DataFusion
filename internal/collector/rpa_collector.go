package collector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/datafusion/worker/internal/models"
)

// RPACollector RPA 采集器（基于 Chromedp）
type RPACollector struct {
	headless bool
	timeout  time.Duration
}

// NewRPACollector 创建 RPA 采集器
func NewRPACollector(headless bool, timeout int) *RPACollector {
	return &RPACollector{
		headless: headless,
		timeout:  time.Duration(timeout) * time.Second,
	}
}

// Type 返回采集器类型
func (r *RPACollector) Type() string {
	return "web-rpa"
}

// Collect 执行数据采集
func (r *RPACollector) Collect(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error) {
	log.Printf("开始 RPA 采集: %s", config.URL)

	// 创建 Chrome 上下文
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 设置超时
	chromeCtx, cancel = context.WithTimeout(chromeCtx, r.timeout)
	defer cancel()

	// 访问页面并获取 HTML
	var htmlContent string
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(config.URL),
		chromedp.WaitReady("body"),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("访问页面失败: %w", err)
	}

	log.Printf("页面加载成功，开始解析数据")

	// 解析 HTML
	return r.parseHTML(htmlContent, config.Selectors)
}

// parseHTML 解析 HTML 内容
func (r *RPACollector) parseHTML(html string, selectors map[string]string) ([]map[string]interface{}, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	var results []map[string]interface{}

	// 假设有一个列表容器选择器
	listSelector, ok := selectors["_list"]
	if !ok {
		// 如果没有列表选择器，则提取单条数据
		item := make(map[string]interface{})
		for field, selector := range selectors {
			if field == "_list" {
				continue
			}
			value := doc.Find(selector).First().Text()
			item[field] = value
		}
		results = append(results, item)
		return results, nil
	}

	// 遍历列表项
	doc.Find(listSelector).Each(func(i int, s *goquery.Selection) {
		item := make(map[string]interface{})
		for field, selector := range selectors {
			if field == "_list" {
				continue
			}
			value := s.Find(selector).Text()
			item[field] = value
		}
		results = append(results, item)
	})

	log.Printf("解析完成，提取到 %d 条数据", len(results))
	return results, nil
}
