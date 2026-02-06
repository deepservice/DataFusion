package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/datafusion/worker/internal/auth"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler API密钥管理处理器
type APIKeyHandler struct {
	db        *sql.DB
	log       *logger.Logger
	apiKeyMgr *auth.APIKeyManager
}

// NewAPIKeyHandler 创建API密钥管理处理器
func NewAPIKeyHandler(db *sql.DB, log *logger.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		db:        db,
		log:       log,
		apiKeyMgr: auth.NewAPIKeyManager(),
	}
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expires_at"` // ISO 8601 格式
}

// APIKeyResponse API密钥响应
type APIKeyResponse struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expires_at"`
	LastUsedAt  *string  `json:"last_used_at"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
}

// CreateAPIKeyResponse 创建API密钥响应
type CreateAPIKeyResponse struct {
	APIKey  string         `json:"api_key"` // 只在创建时返回
	KeyInfo APIKeyResponse `json:"key_info"`
}

// ListAPIKeys 获取当前用户的API密钥列表
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	query := `
		SELECT id, name, description, permissions, expires_at, last_used_at, status, created_at
		FROM api_keys 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		h.log.WithError(err).Error("查询API密钥列表失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询API密钥列表失败"})
		return
	}
	defer rows.Close()

	var apiKeys []APIKeyResponse
	for rows.Next() {
		var key APIKeyResponse
		var permissionsJSON []byte
		var expiresAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.Name, &key.Description, &permissionsJSON,
			&expiresAt, &lastUsedAt, &key.Status, &key.CreatedAt,
		)
		if err != nil {
			h.log.WithError(err).Error("扫描API密钥数据失败")
			continue
		}

		// 解析权限JSON
		if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
			h.log.WithError(err).Error("解析权限JSON失败")
			key.Permissions = []string{}
		}

		// 处理时间字段
		if expiresAt.Valid {
			expiresAtStr := expiresAt.Time.Format(time.RFC3339)
			key.ExpiresAt = &expiresAtStr
		}
		if lastUsedAt.Valid {
			lastUsedAtStr := lastUsedAt.Time.Format(time.RFC3339)
			key.LastUsedAt = &lastUsedAtStr
		}

		apiKeys = append(apiKeys, key)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": apiKeys})
}

// CreateAPIKey 创建API密钥
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的过期时间格式"})
			return
		}
		expiresAt = &parsedTime
	}

	// 设置默认权限
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"}
	}

	// 创建API密钥
	keyInfo, apiKey, err := h.apiKeyMgr.CreateAPIKey(
		userID, req.Name, req.Description, req.Permissions, expiresAt,
	)
	if err != nil {
		h.log.WithError(err).Error("创建API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建API密钥失败"})
		return
	}

	// 序列化权限
	permissionsJSON, err := json.Marshal(keyInfo.Permissions)
	if err != nil {
		h.log.WithError(err).Error("序列化权限失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建API密钥失败"})
		return
	}

	// 保存到数据库
	query := `
		INSERT INTO api_keys (user_id, key_hash, name, description, permissions, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err = h.db.QueryRow(
		query, keyInfo.UserID, keyInfo.KeyHash, keyInfo.Name,
		keyInfo.Description, permissionsJSON, keyInfo.ExpiresAt, keyInfo.Status,
	).Scan(&keyInfo.ID, &keyInfo.CreatedAt)

	if err != nil {
		h.log.WithError(err).Error("保存API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存API密钥失败"})
		return
	}

	// 构建响应
	response := CreateAPIKeyResponse{
		APIKey: apiKey,
		KeyInfo: APIKeyResponse{
			ID:          keyInfo.ID,
			Name:        keyInfo.Name,
			Description: keyInfo.Description,
			Permissions: keyInfo.Permissions,
			Status:      keyInfo.Status,
			CreatedAt:   keyInfo.CreatedAt.Format(time.RFC3339),
		},
	}

	if keyInfo.ExpiresAt != nil {
		expiresAtStr := keyInfo.ExpiresAt.Format(time.RFC3339)
		response.KeyInfo.ExpiresAt = &expiresAtStr
	}

	c.JSON(http.StatusCreated, response)
}

// GetAPIKey 获取单个API密钥信息
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的API密钥ID"})
		return
	}

	var key APIKeyResponse
	var permissionsJSON []byte
	var expiresAt, lastUsedAt sql.NullTime

	query := `
		SELECT id, name, description, permissions, expires_at, last_used_at, status, created_at
		FROM api_keys 
		WHERE id = $1 AND user_id = $2
	`

	err = h.db.QueryRow(query, id, userID).Scan(
		&key.ID, &key.Name, &key.Description, &permissionsJSON,
		&expiresAt, &lastUsedAt, &key.Status, &key.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "API密钥不存在"})
			return
		}
		h.log.WithError(err).Error("查询API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询API密钥失败"})
		return
	}

	// 解析权限JSON
	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		h.log.WithError(err).Error("解析权限JSON失败")
		key.Permissions = []string{}
	}

	// 处理时间字段
	if expiresAt.Valid {
		expiresAtStr := expiresAt.Time.Format(time.RFC3339)
		key.ExpiresAt = &expiresAtStr
	}
	if lastUsedAt.Valid {
		lastUsedAtStr := lastUsedAt.Time.Format(time.RFC3339)
		key.LastUsedAt = &lastUsedAtStr
	}

	c.JSON(http.StatusOK, key)
}

// UpdateAPIKeyRequest 更新API密钥请求
type UpdateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateAPIKey 更新API密钥
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的API密钥ID"})
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 构建更新查询
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != "" {
		setParts = append(setParts, "name = $"+strconv.Itoa(argIndex))
		args = append(args, req.Name)
		argIndex++
	}

	if req.Description != "" {
		setParts = append(setParts, "description = $"+strconv.Itoa(argIndex))
		args = append(args, req.Description)
		argIndex++
	}

	if len(req.Permissions) > 0 {
		permissionsJSON, err := json.Marshal(req.Permissions)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的权限格式"})
			return
		}
		setParts = append(setParts, "permissions = $"+strconv.Itoa(argIndex))
		args = append(args, permissionsJSON)
		argIndex++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有提供更新字段"})
		return
	}

	args = append(args, id, userID)

	query := "UPDATE api_keys SET " + setParts[0]
	for i := 1; i < len(setParts); i++ {
		query += ", " + setParts[i]
	}
	query += " WHERE id = $" + strconv.Itoa(argIndex) + " AND user_id = $" + strconv.Itoa(argIndex+1)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		h.log.WithError(err).Error("更新API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新API密钥失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API密钥不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API密钥更新成功"})
}

// RevokeAPIKey 撤销API密钥
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的API密钥ID"})
		return
	}

	query := `
		UPDATE api_keys 
		SET status = 'revoked' 
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`

	result, err := h.db.Exec(query, id, userID)
	if err != nil {
		h.log.WithError(err).Error("撤销API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "撤销API密钥失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API密钥不存在或已被撤销"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API密钥撤销成功"})
}

// DeleteAPIKey 删除API密钥
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的API密钥ID"})
		return
	}

	query := "DELETE FROM api_keys WHERE id = $1 AND user_id = $2"
	result, err := h.db.Exec(query, id, userID)
	if err != nil {
		h.log.WithError(err).Error("删除API密钥失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除API密钥失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API密钥不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API密钥删除成功"})
}
