package processor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/datafusion/worker/internal/models"
)

// EnhancedCleaner 增强的数据清洗器
type EnhancedCleaner struct {
	rules []models.CleaningRule
}

// NewEnhancedCleaner 创建增强清洗器
func NewEnhancedCleaner(rules []models.CleaningRule) *EnhancedCleaner {
	return &EnhancedCleaner{
		rules: rules,
	}
}

// Clean 执行数据清洗
func (c *EnhancedCleaner) Clean(data []map[string]interface{}) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(data))

	for _, item := range data {
		cleaned := make(map[string]interface{})
		
		// 复制原始数据
		for k, v := range item {
			cleaned[k] = v
		}

		// 应用清洗规则
		for _, rule := range c.rules {
			if err := c.applyRule(cleaned, rule); err != nil {
				return nil, fmt.Errorf("应用规则 %s 失败: %w", rule.Name, err)
			}
		}

		result = append(result, cleaned)
	}

	return result, nil
}

// applyRule 应用单个清洗规则
func (c *EnhancedCleaner) applyRule(data map[string]interface{}, rule models.CleaningRule) error {
	value, exists := data[rule.Field]
	if !exists {
		return nil // 字段不存在，跳过
	}

	// 转换为字符串
	strValue := fmt.Sprintf("%v", value)

	var result interface{}
	var err error

	switch rule.Type {
	case "trim":
		result = c.cleanTrim(strValue)
	case "remove_html":
		result = c.cleanRemoveHTML(strValue)
	case "regex":
		result, err = c.cleanRegex(strValue, rule.Pattern, rule.Replacement)
	case "normalize_whitespace":
		result = c.cleanNormalizeWhitespace(strValue)
	case "remove_special_chars":
		result = c.cleanRemoveSpecialChars(strValue)
	case "date_format":
		result, err = c.cleanDateFormat(strValue, rule.Pattern)
	case "number_format":
		result, err = c.cleanNumberFormat(strValue)
	case "email_validate":
		result, err = c.cleanEmailValidate(strValue)
	case "phone_format":
		result, err = c.cleanPhoneFormat(strValue)
	case "url_normalize":
		result = c.cleanURLNormalize(strValue)
	default:
		return fmt.Errorf("未知的清洗规则类型: %s", rule.Type)
	}

	if err != nil {
		return err
	}

	data[rule.Field] = result
	return nil
}

// cleanTrim 去除首尾空白
func (c *EnhancedCleaner) cleanTrim(value string) string {
	return strings.TrimSpace(value)
}

// cleanRemoveHTML 移除 HTML 标签
func (c *EnhancedCleaner) cleanRemoveHTML(value string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(value, "")
}

// cleanRegex 正则表达式替换
func (c *EnhancedCleaner) cleanRegex(value, pattern, replacement string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return value, fmt.Errorf("编译正则表达式失败: %w", err)
	}
	return re.ReplaceAllString(value, replacement), nil
}

// cleanNormalizeWhitespace 规范化空白字符
func (c *EnhancedCleaner) cleanNormalizeWhitespace(value string) string {
	// 替换多个空白字符为单个空格
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(value, " "))
}

// cleanRemoveSpecialChars 移除特殊字符
func (c *EnhancedCleaner) cleanRemoveSpecialChars(value string) string {
	var result strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// cleanDateFormat 日期格式化
func (c *EnhancedCleaner) cleanDateFormat(value, targetFormat string) (string, error) {
	// 尝试多种常见日期格式
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		time.RFC3339,
		time.RFC1123,
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, value)
		if err == nil {
			break
		}
	}

	if err != nil {
		return value, fmt.Errorf("无法解析日期: %s", value)
	}

	// 如果没有指定目标格式，使用 ISO 8601
	if targetFormat == "" {
		targetFormat = "2006-01-02"
	}

	return parsedTime.Format(targetFormat), nil
}

// cleanNumberFormat 数字格式化
func (c *EnhancedCleaner) cleanNumberFormat(value string) (float64, error) {
	// 移除常见的数字分隔符
	cleaned := strings.ReplaceAll(value, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.TrimSpace(cleaned)

	// 尝试解析为浮点数
	num, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析数字: %s", value)
	}

	return num, nil
}

// cleanEmailValidate 邮箱验证和规范化
func (c *EnhancedCleaner) cleanEmailValidate(value string) (string, error) {
	// 简单的邮箱正则
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	cleaned := strings.TrimSpace(strings.ToLower(value))
	
	if !re.MatchString(cleaned) {
		return value, fmt.Errorf("无效的邮箱地址: %s", value)
	}

	return cleaned, nil
}

// cleanPhoneFormat 电话号码格式化
func (c *EnhancedCleaner) cleanPhoneFormat(value string) (string, error) {
	// 移除所有非数字字符
	re := regexp.MustCompile(`\D`)
	cleaned := re.ReplaceAllString(value, "")

	// 验证长度（中国手机号 11 位）
	if len(cleaned) != 11 {
		return value, fmt.Errorf("无效的电话号码长度: %s", value)
	}

	// 格式化为 xxx-xxxx-xxxx
	return fmt.Sprintf("%s-%s-%s", cleaned[0:3], cleaned[3:7], cleaned[7:11]), nil
}

// cleanURLNormalize URL 规范化
func (c *EnhancedCleaner) cleanURLNormalize(value string) string {
	// 移除首尾空白
	cleaned := strings.TrimSpace(value)

	// 确保有协议
	if !strings.HasPrefix(cleaned, "http://") && !strings.HasPrefix(cleaned, "https://") {
		cleaned = "https://" + cleaned
	}

	// 移除尾部斜杠
	cleaned = strings.TrimSuffix(cleaned, "/")

	return cleaned
}
