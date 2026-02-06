package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/datafusion/worker/internal/auth"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	db          *sql.DB
	log         *logger.Logger
	passwordMgr *auth.PasswordManager
	rbac        *auth.RBAC
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(db *sql.DB, log *logger.Logger) *UserHandler {
	return &UserHandler{
		db:          db,
		log:         log,
		passwordMgr: auth.NewPasswordManager(),
		rbac:        auth.NewRBAC(),
	}
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Role     string `json:"role" binding:"required"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	AuthType  string `json:"auth_type"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// 搜索参数
	search := c.Query("search")

	// 构建查询
	var query string
	var args []interface{}

	if search != "" {
		query = `
			SELECT id, username, email, role, status, auth_type, created_at, updated_at
			FROM users 
			WHERE username ILIKE $1 OR email ILIKE $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{"%" + search + "%", limit, offset}
	} else {
		query = `
			SELECT id, username, email, role, status, auth_type, created_at, updated_at
			FROM users 
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.log.WithError(err).Error("查询用户列表失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户列表失败"})
		return
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var user UserResponse
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Role,
			&user.Status, &user.AuthType, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			h.log.WithError(err).Error("扫描用户数据失败")
			continue
		}
		users = append(users, user)
	}

	// 获取总数
	var total int64
	countQuery := "SELECT COUNT(*) FROM users"
	if search != "" {
		countQuery += " WHERE username ILIKE $1 OR email ILIKE $1"
		h.db.QueryRow(countQuery, "%"+search+"%").Scan(&total)
	} else {
		h.db.QueryRow(countQuery).Scan(&total)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetUser 获取单个用户
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var user UserResponse
	query := `
		SELECT id, username, email, role, status, auth_type, created_at, updated_at
		FROM users 
		WHERE id = $1
	`

	err = h.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Role,
		&user.Status, &user.AuthType, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		h.log.WithError(err).Error("查询用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证角色
	if err := h.rbac.ValidateRole(req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	err := h.db.QueryRow(checkQuery, req.Username).Scan(&exists)
	if err != nil {
		h.log.WithError(err).Error("检查用户名失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 哈希密码
	passwordHash, err := h.passwordMgr.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建用户
	query := `
		INSERT INTO users (username, password_hash, email, role, auth_type, status)
		VALUES ($1, $2, $3, $4, 'local', 'active')
		RETURNING id, created_at, updated_at
	`

	var user UserResponse
	err = h.db.QueryRow(query, req.Username, passwordHash, req.Email, req.Role).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		h.log.WithError(err).Error("创建用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	user.Username = req.Username
	user.Email = req.Email
	user.Role = req.Role
	user.Status = "active"
	user.AuthType = "local"

	c.JSON(http.StatusCreated, user)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证角色（如果提供）
	if req.Role != "" {
		if err := h.rbac.ValidateRole(req.Role); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// 验证状态（如果提供）
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户状态"})
		return
	}

	// 构建更新查询
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Email != "" {
		setParts = append(setParts, "email = $"+strconv.Itoa(argIndex))
		args = append(args, req.Email)
		argIndex++
	}

	if req.Role != "" {
		setParts = append(setParts, "role = $"+strconv.Itoa(argIndex))
		args = append(args, req.Role)
		argIndex++
	}

	if req.Status != "" {
		setParts = append(setParts, "status = $"+strconv.Itoa(argIndex))
		args = append(args, req.Status)
		argIndex++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有提供更新字段"})
		return
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := "UPDATE users SET " + setParts[0]
	for i := 1; i < len(setParts); i++ {
		query += ", " + setParts[i]
	}
	query += " WHERE id = $" + strconv.Itoa(argIndex)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		h.log.WithError(err).Error("更新用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户更新成功"})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 检查是否为当前用户
	currentUserID, _ := auth.GetCurrentUserID(c)
	if currentUserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账户"})
		return
	}

	// 软删除：将状态设置为inactive
	query := "UPDATE users SET status = 'inactive', updated_at = NOW() WHERE id = $1"
	result, err := h.db.Exec(query, id)
	if err != nil {
		h.log.WithError(err).Error("删除用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// ResetPassword 重置用户密码（管理员功能）
func (h *UserHandler) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 哈希新密码
	passwordHash, err := h.passwordMgr.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新密码
	query := `
		UPDATE users 
		SET password_hash = $1, updated_at = NOW() 
		WHERE id = $2 AND status = 'active'
	`

	result, err := h.db.Exec(query, passwordHash, id)
	if err != nil {
		h.log.WithError(err).Error("重置密码失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在或已被禁用"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
}

// GetRoles 获取所有角色
func (h *UserHandler) GetRoles(c *gin.Context) {
	roles := h.rbac.GetAllRoles()

	roleList := make([]gin.H, 0, len(roles))
	for _, role := range roles {
		roleList = append(roleList, gin.H{
			"name":        role.Name,
			"description": role.Description,
			"permissions": role.Permissions,
		})
	}

	c.JSON(http.StatusOK, gin.H{"roles": roleList})
}
