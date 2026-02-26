package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DataSourceHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewDataSourceHandler(db *sql.DB, log *logger.Logger) *DataSourceHandler {
	return &DataSourceHandler{db: db, log: log}
}

type DataSource struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // web, api, database
	Config      string    `json:"config"` // JSON配置
	Description *string   `json:"description"`
	Status      string    `json:"status"` // active, inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PageElement 页面元素，用于结构预览
type PageElement struct {
	Selector string `json:"selector"`
	Tag      string `json:"tag"`
	ID       string `json:"id,omitempty"`
	Class    string `json:"class,omitempty"`
	Text     string `json:"text"`
}

// List 获取数据源列表
func (h *DataSourceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	dsType := c.Query("type")

	offset := (page - 1) * pageSize

	query := `SELECT id, name, type, config, description, status, created_at, updated_at
	          FROM data_sources WHERE 1=1`
	countQuery := "SELECT COUNT(*) FROM data_sources WHERE 1=1"
	args := []any{}
	countArgs := []any{}
	argIdx := 1

	if dsType != "" {
		query += " AND type = $" + strconv.Itoa(argIdx)
		countQuery += " AND type = $" + strconv.Itoa(argIdx)
		args = append(args, dsType)
		countArgs = append(countArgs, dsType)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.log.Error("查询数据源列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	datasources := []DataSource{}
	for rows.Next() {
		var ds DataSource
		err := rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Config, &ds.Description,
			&ds.Status, &ds.CreatedAt, &ds.UpdatedAt)
		if err != nil {
			h.log.Error("扫描数据源数据失败", zap.Error(err))
			continue
		}
		datasources = append(datasources, ds)
	}

	var total int
	h.db.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":      datasources,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个数据源
func (h *DataSourceHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var ds DataSource
	err := h.db.QueryRow(`SELECT id, name, type, config, description, status, created_at, updated_at
	                      FROM data_sources WHERE id = $1`, id).
		Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Config, &ds.Description,
			&ds.Status, &ds.CreatedAt, &ds.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, ds)
}

// Create 创建数据源
func (h *DataSourceHandler) Create(c *gin.Context) {
	var ds DataSource
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ds.Status == "" {
		ds.Status = "active"
	}

	err := h.db.QueryRow(`INSERT INTO data_sources (name, type, config, description, status)
	    VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		ds.Name, ds.Type, ds.Config, ds.Description, ds.Status).Scan(&ds.ID)

	if err != nil {
		h.log.Error("创建数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	h.log.Info("创建数据源成功", zap.Int64("datasource_id", ds.ID), zap.String("name", ds.Name))
	c.JSON(http.StatusCreated, ds)
}

// Update 更新数据源
func (h *DataSourceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var ds DataSource
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec(`UPDATE data_sources SET
	    name=$1, type=$2, config=$3, description=$4, status=$5, updated_at=NOW()
	    WHERE id=$6`,
		ds.Name, ds.Type, ds.Config, ds.Description, ds.Status, id)

	if err != nil {
		h.log.Error("更新数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	h.log.Info("更新数据源成功", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Delete 删除数据源
func (h *DataSourceHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("DELETE FROM data_sources WHERE id=$1", id)
	if err != nil {
		h.log.Error("删除数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	h.log.Info("删除数据源成功", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestConnection 测试数据源连接
func (h *DataSourceHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	h.log.Info("测试数据源连接", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "连接测试成功",
	})
}

// PreviewPageStructure 预览页面/API 结构
// - Web 类型：返回 CSS 选择器列表
// - API 类型（JSON 响应）：返回顶级字段列表
func (h *DataSourceHandler) PreviewPageStructure(c *gin.Context) {
	id := c.Param("id")

	// 从数据库获取数据源配置和类型
	var configStr, dsType string
	err := h.db.QueryRow(`SELECT config, type FROM data_sources WHERE id = $1`, id).Scan(&configStr, &dsType)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 解析 config JSON
	var dsConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &dsConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数据源配置格式错误"})
		return
	}

	// 从 url 或 endpoint 字段取 URL（兼容旧配置）
	pageURL, _ := dsConfig["url"].(string)
	if pageURL == "" {
		pageURL, _ = dsConfig["endpoint"].(string)
	}
	if pageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数据源未配置 URL（url 或 endpoint 字段）"})
		return
	}

	h.log.Info("预览结构", zap.String("type", dsType), zap.String("url", pageURL))

	// 发送 HTTP 请求
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL 格式错误: " + err.Error()})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/html, */*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	// 转发自定义 headers
	if hdrs, ok := dsConfig["headers"].(map[string]interface{}); ok {
		for k, v := range hdrs {
			if s, ok := v.(string); ok && s != "" {
				req.Header.Set(k, s)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法访问地址: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// 限制读取大小为 2MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应内容失败"})
		return
	}

	contentType := resp.Header.Get("Content-Type")

	// JSON 响应：提取字段结构（适用于 API 类型）
	if strings.Contains(contentType, "application/json") || looksLikeJSON(body) {
		fields := extractJSONFields(body)
		c.JSON(http.StatusOK, gin.H{
			"status":        "success",
			"url":           pageURL,
			"title":         "API 响应字段",
			"response_type": "json",
			"elements":      fields,
		})
		return
	}

	// HTML 响应：提取 CSS 选择器（适用于 Web 类型）
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析页面失败"})
		return
	}

	pageTitle := strings.TrimSpace(doc.Find("title").Text())
	doc.Find("script, style, noscript").Remove()
	elements := extractPageElements(doc)

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"url":           pageURL,
		"title":         pageTitle,
		"response_type": "html",
		"elements":      elements,
	})
}

// looksLikeJSON 粗略判断响应体是否是 JSON
func looksLikeJSON(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

// extractJSONFields 从 JSON 响应中提取字段路径和样本值，用于 API 数据源预览
// 支持顶层对象字段、数组中第一个元素的字段
func extractJSONFields(body []byte) []PageElement {
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return []PageElement{{
			Selector: "(raw)",
			Tag:      "text",
			Text:     truncate(strings.TrimSpace(string(body)), 200),
		}}
	}

	var elements []PageElement
	extractFieldsRecursive("", root, &elements, 0)
	return elements
}

// extractFieldsRecursive 递归提取 JSON 字段（最多 3 层，80 个字段）
func extractFieldsRecursive(prefix string, node interface{}, elements *[]PageElement, depth int) {
	if len(*elements) >= 80 || depth > 3 {
		return
	}
	switch v := node.(type) {
	case map[string]interface{}:
		for key, val := range v {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			valStr := jsonValuePreview(val)
			tag := jsonTypeName(val)
			*elements = append(*elements, PageElement{
				Selector: path,
				Tag:      tag,
				Text:     valStr,
			})
			// 递归对象/数组
			if depth < 2 {
				extractFieldsRecursive(path, val, elements, depth+1)
			}
		}
	case []interface{}:
		if len(v) > 0 {
			itemPath := prefix + "[0]"
			if prefix == "" {
				itemPath = "[0]"
			}
			extractFieldsRecursive(itemPath, v[0], elements, depth+1)
		}
	}
}

// jsonValuePreview 将 JSON 值格式化为预览字符串
func jsonValuePreview(v interface{}) string {
	switch val := v.(type) {
	case string:
		return truncate(val, 150)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case map[string]interface{}:
		b, _ := json.Marshal(val)
		return truncate(string(b), 150)
	case []interface{}:
		return fmt.Sprintf("[%d 项]", len(val))
	default:
		b, _ := json.Marshal(val)
		return truncate(string(b), 150)
	}
}

// jsonTypeName 返回 JSON 值的类型名
func jsonTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

// truncate 截断字符串到指定长度
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractPageElements 从页面中提取有意义的元素及其 CSS 选择器
func extractPageElements(doc *goquery.Document) []PageElement {
	var elements []PageElement
	seen := map[string]bool{} // 去重，同选择器只保留第一个

	// 按优先级遍历有意义的元素
	doc.Find("h1,h2,h3,h4,h5,h6,title,p,article,section,main,div[id],div[class],span[id],span[class],a[href],img[src],ul,ol,table,[id],[class]").Each(func(i int, s *goquery.Selection) {
		if len(elements) >= 80 {
			return
		}

		tag := goquery.NodeName(s)
		rawText := strings.TrimSpace(s.Text())
		// 跳过空内容或过短（可能是图标、按钮等）
		if len(rawText) < 2 {
			return
		}

		// 截断文本
		textPreview := rawText
		if len(textPreview) > 150 {
			textPreview = textPreview[:150] + "..."
		}

		elID, _ := s.Attr("id")
		elClass, _ := s.Attr("class")

		// 构建 CSS 选择器
		selector := buildCSSSelector(tag, elID, elClass)
		if selector == "" || selector == "div" || selector == "span" {
			return // 跳过无意义的通用标签
		}

		// 去重：同选择器只保留第一条（文本最长的）
		if seen[selector] {
			return
		}
		seen[selector] = true

		// 清理 class（只取前两个，避免太长）
		displayClass := ""
		if elClass != "" {
			parts := strings.Fields(elClass)
			if len(parts) > 2 {
				displayClass = strings.Join(parts[:2], " ") + "..."
			} else {
				displayClass = elClass
			}
		}

		elements = append(elements, PageElement{
			Selector: selector,
			Tag:      tag,
			ID:       elID,
			Class:    displayClass,
			Text:     textPreview,
		})
	})

	return elements
}

// buildCSSSelector 构建 CSS 选择器
func buildCSSSelector(tag, id, class string) string {
	// 有 ID 优先使用 #id（最精确）
	if id != "" && !strings.ContainsAny(id, " \t\n.:") {
		return "#" + id
	}
	// 有 class 使用 tag.first-class
	if class != "" {
		classes := strings.Fields(class)
		for _, cls := range classes {
			// 跳过空的或含特殊字符的 class
			if cls == "" || strings.ContainsAny(cls, ".:[]{}()") {
				continue
			}
			return tag + "." + cls
		}
	}
	// 标题/语义标签直接用 tag
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6", "article", "main", "section", "p":
		return tag
	}
	return ""
}
