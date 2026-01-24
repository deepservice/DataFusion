package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CleaningRuleHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewCleaningRuleHandler(db *sql.DB, log *logger.Logger) *CleaningRuleHandler {
	return &CleaningRuleHandler{db: db, log: log}
}

type CleaningRule struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RuleType    string    `json:"rule_type"` // trim, remove_html, regex, etc.
	Config      string    `json:"config"`    // JSON配置
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// List 获取清洗规则列表
func (h *CleaningRuleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	offset := (page - 1) * pageSize

	rows, err := h.db.Query(`SELECT id, name, description, rule_type, config, created_at, updated_at 
	                         FROM cleaning_rules ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		h.log.Error("查询清洗规则列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	rules := []CleaningRule{}
	for rows.Next() {
		var rule CleaningRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.RuleType,
			&rule.Config, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			h.log.Error("扫描清洗规则数据失败", zap.Error(err))
			continue
		}
		rules = append(rules, rule)
	}

	var total int
	h.db.QueryRow("SELECT COUNT(*) FROM cleaning_rules").Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":      rules,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个清洗规则
func (h *CleaningRuleHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var rule CleaningRule
	err := h.db.QueryRow(`SELECT id, name, description, rule_type, config, created_at, updated_at 
	                      FROM cleaning_rules WHERE id = $1`, id).
		Scan(&rule.ID, &rule.Name, &rule.Description, &rule.RuleType,
			&rule.Config, &rule.CreatedAt, &rule.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "清洗规则不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询清洗规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// Create 创建清洗规则
func (h *CleaningRuleHandler) Create(c *gin.Context) {
	var rule CleaningRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.db.QueryRow(`INSERT INTO cleaning_rules (name, description, rule_type, config) 
	    VALUES ($1, $2, $3, $4) RETURNING id`,
		rule.Name, rule.Description, rule.RuleType, rule.Config).Scan(&rule.ID)

	if err != nil {
		h.log.Error("创建清洗规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	h.log.Info("创建清洗规则成功", zap.Int64("rule_id", rule.ID), zap.String("name", rule.Name))
	c.JSON(http.StatusCreated, rule)
}

// Update 更新清洗规则
func (h *CleaningRuleHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var rule CleaningRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec(`UPDATE cleaning_rules SET 
	    name=$1, description=$2, rule_type=$3, config=$4, updated_at=NOW() 
	    WHERE id=$5`,
		rule.Name, rule.Description, rule.RuleType, rule.Config, id)

	if err != nil {
		h.log.Error("更新清洗规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "清洗规则不存在"})
		return
	}

	h.log.Info("更新清洗规则成功", zap.String("rule_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Delete 删除清洗规则
func (h *CleaningRuleHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("DELETE FROM cleaning_rules WHERE id=$1", id)
	if err != nil {
		h.log.Error("删除清洗规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "清洗规则不存在"})
		return
	}

	h.log.Info("删除清洗规则成功", zap.String("rule_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
