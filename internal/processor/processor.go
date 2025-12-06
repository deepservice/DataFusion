package processor

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/datafusion/worker/internal/models"
)

// Processor 数据处理器
type Processor struct {
	config *models.ProcessorConfig
}

// NewProcessor 创建数据处理器
func NewProcessor(config *models.ProcessorConfig) *Processor {
	return &Processor{config: config}
}

// Process 处理数据
func (p *Processor) Process(data []map[string]interface{}) ([]map[string]interface{}, error) {
	if p.config == nil {
		return data, nil
	}

	log.Printf("开始数据处理，共 %d 条数据", len(data))

	// 使用增强清洗器
	if len(p.config.CleaningRules) > 0 {
		enhancedCleaner := NewEnhancedCleaner(p.config.CleaningRules)
		cleaned, err := enhancedCleaner.Clean(data)
		if err != nil {
			return nil, fmt.Errorf("增强清洗失败: %w", err)
		}
		data = cleaned
	}

	// 应用转换规则
	var processed []map[string]interface{}
	for _, record := range data {
		transformed, err := p.applyTransformRules(record)
		if err != nil {
			log.Printf("转换数据失败: %v", err)
			continue
		}
		processed = append(processed, transformed)
	}

	log.Printf("数据处理完成，有效数据 %d 条", len(processed))
	return processed, nil
}

// applyCleaningRules 应用清洗规则
func (p *Processor) applyCleaningRules(record map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range record {
		result[k] = v
	}

	for _, rule := range p.config.CleaningRules {
		value, ok := result[rule.Field]
		if !ok {
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			continue
		}

		cleaned, err := p.applyCleaningRule(strValue, &rule)
		if err != nil {
			return nil, fmt.Errorf("应用清洗规则失败: %w", err)
		}

		result[rule.Field] = cleaned
	}

	return result, nil
}

// applyCleaningRule 应用单个清洗规则
func (p *Processor) applyCleaningRule(value string, rule *models.CleaningRule) (string, error) {
	switch rule.Type {
	case "trim":
		return strings.TrimSpace(value), nil

	case "remove_html":
		// 简单的 HTML 标签移除
		re := regexp.MustCompile(`<[^>]*>`)
		return re.ReplaceAllString(value, ""), nil

	case "regex":
		if rule.Pattern == "" {
			return value, nil
		}
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return "", fmt.Errorf("编译正则表达式失败: %w", err)
		}
		return re.ReplaceAllString(value, rule.Replacement), nil

	case "lowercase":
		return strings.ToLower(value), nil

	case "uppercase":
		return strings.ToUpper(value), nil

	default:
		return value, nil
	}
}

// applyTransformRules 应用转换规则
func (p *Processor) applyTransformRules(record map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range record {
		result[k] = v
	}

	for _, rule := range p.config.TransformRules {
		value, ok := result[rule.SourceField]
		if !ok {
			continue
		}

		// 简单的字段映射
		if rule.TargetField != "" && rule.TargetField != rule.SourceField {
			result[rule.TargetField] = value
			delete(result, rule.SourceField)
		}
	}

	return result, nil
}
