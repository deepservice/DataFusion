package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/datafusion/worker/internal/auth"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	db          *sql.DB
	log         *logger.Logger
	jwtManager  *auth.JWTManager
	passwordMgr *auth.PasswordManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *sql.DB, log *logger.Logger, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		db:          db,
		log:         log,
		jwtManager:  jwtManager,
		passwordMgr: auth.NewPasswordManager(),
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询用户
	var user UserInfo
	var passwordHash string
	var status string

	query := `
		SELECT id, username, email, role, password_hash, status 
		FROM users 
		WHERE username = $1 AND auth_type = 'local'
	`

	err := h.db.QueryRow(query, req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Role, &passwordHash, &status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		h.log.WithError(err).Error("查询用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	// 检查用户状态
	if status != "active" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户账户已被禁用"})
		return
	}

	// 验证密码
	if err := h.passwordMgr.VerifyPassword(passwordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT Token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		h.log.WithError(err).Error("生成Token失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成访问令牌失败"})
		return
	}

	// 返回登录成功响应
	response := LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24小时过期
		User:      user,
	}

	c.JSON(http.StatusOK, response)
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// RefreshToken 刷新访问令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证旧Token
	claims, err := h.jwtManager.VerifyToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的访问令牌"})
		return
	}

	// 检查用户是否仍然有效
	var status string
	query := "SELECT status FROM users WHERE id = $1"
	err = h.db.QueryRow(query, claims.UserID).Scan(&status)
	if err != nil || status != "active" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户账户无效"})
		return
	}

	// 生成新Token
	newToken, err := h.jwtManager.GenerateToken(claims.UserID, claims.Username, claims.Role)
	if err != nil {
		h.log.WithError(err).Error("生成新Token失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成新访问令牌失败"})
		return
	}

	response := LoginResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile 获取用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	var user UserInfo
	query := `
		SELECT id, username, email, role 
		FROM users 
		WHERE id = $1 AND status = 'active'
	`

	err := h.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		h.log.WithError(err).Error("查询用户信息失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Email string `json:"email"`
}

// UpdateProfile 更新用户信息
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 更新用户信息
	query := `
		UPDATE users 
		SET email = $1, updated_at = NOW() 
		WHERE id = $2 AND status = 'active'
	`

	result, err := h.db.Exec(query, req.Email, userID)
	if err != nil {
		h.log.WithError(err).Error("更新用户信息失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户信息失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户信息更新成功"})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := auth.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 查询当前密码哈希
	var currentPasswordHash string
	query := "SELECT password_hash FROM users WHERE id = $1 AND status = 'active'"
	err := h.db.QueryRow(query, userID).Scan(&currentPasswordHash)
	if err != nil {
		h.log.WithError(err).Error("查询用户密码失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	// 验证旧密码
	if err := h.passwordMgr.VerifyPassword(currentPasswordHash, req.OldPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}

	// 哈希新密码
	newPasswordHash, err := h.passwordMgr.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新密码
	updateQuery := `
		UPDATE users 
		SET password_hash = $1, updated_at = NOW() 
		WHERE id = $2
	`

	_, err = h.db.Exec(updateQuery, newPasswordHash, userID)
	if err != nil {
		h.log.WithError(err).Error("更新密码失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT是无状态的，客户端删除token即可
	// 这里可以记录登出日志
	if userID, exists := auth.GetCurrentUserID(c); exists {
		h.log.WithFields(map[string]interface{}{"user_id": userID}).Info("用户登出")
	}

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}
