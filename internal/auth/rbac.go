package auth

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// Permission 权限定义
type Permission struct {
	Resource string `json:"resource"` // 资源名称，如 "tasks", "datasources"
	Action   string `json:"action"`   // 操作名称，如 "read", "write", "delete"
}

// Role 角色定义
type Role struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// RBAC 基于角色的访问控制
type RBAC struct {
	roles map[string]*Role
}

// NewRBAC 创建RBAC实例
func NewRBAC() *RBAC {
	rbac := &RBAC{
		roles: make(map[string]*Role),
	}

	// 初始化默认角色
	rbac.initDefaultRoles()

	return rbac
}

// initDefaultRoles 初始化默认角色
func (r *RBAC) initDefaultRoles() {
	// 管理员角色 - 拥有所有权限
	adminRole := &Role{
		Name:        "admin",
		Description: "系统管理员，拥有所有权限",
		Permissions: []Permission{
			{"*", "*"}, // 通配符表示所有资源的所有操作
		},
	}
	r.roles["admin"] = adminRole

	// 操作员角色 - 可以管理任务和数据源
	operatorRole := &Role{
		Name:        "operator",
		Description: "操作员，可以管理任务和数据源",
		Permissions: []Permission{
			{"tasks", "read"},
			{"tasks", "write"},
			{"tasks", "delete"},
			{"datasources", "read"},
			{"datasources", "write"},
			{"datasources", "delete"},
			{"executions", "read"},
			{"cleaning-rules", "read"},
			{"cleaning-rules", "write"},
			{"stats", "read"},
		},
	}
	r.roles["operator"] = operatorRole

	// 查看者角色 - 只能查看信息
	viewerRole := &Role{
		Name:        "viewer",
		Description: "查看者，只能查看信息",
		Permissions: []Permission{
			{"tasks", "read"},
			{"datasources", "read"},
			{"executions", "read"},
			{"cleaning-rules", "read"},
			{"stats", "read"},
		},
	}
	r.roles["viewer"] = viewerRole

	// 普通用户角色 - 基础权限
	userRole := &Role{
		Name:        "user",
		Description: "普通用户，基础权限",
		Permissions: []Permission{
			{"tasks", "read"},
			{"executions", "read"},
			{"stats", "read"},
		},
	}
	r.roles["user"] = userRole
}

// AddRole 添加角色
func (r *RBAC) AddRole(role *Role) {
	r.roles[role.Name] = role
}

// GetRole 获取角色
func (r *RBAC) GetRole(roleName string) (*Role, bool) {
	role, exists := r.roles[roleName]
	return role, exists
}

// HasPermission 检查用户是否有指定权限
func (r *RBAC) HasPermission(roleName, resource, action string) bool {
	role, exists := r.roles[roleName]
	if !exists {
		return false
	}

	for _, permission := range role.Permissions {
		// 检查通配符权限
		if permission.Resource == "*" && permission.Action == "*" {
			return true
		}

		// 检查资源通配符
		if permission.Resource == "*" && permission.Action == action {
			return true
		}

		// 检查操作通配符
		if permission.Resource == resource && permission.Action == "*" {
			return true
		}

		// 检查精确匹配
		if permission.Resource == resource && permission.Action == action {
			return true
		}
	}

	return false
}

// GetAllRoles 获取所有角色
func (r *RBAC) GetAllRoles() map[string]*Role {
	return r.roles
}

// ValidateRole 验证角色是否存在
func (r *RBAC) ValidateRole(roleName string) error {
	if _, exists := r.roles[roleName]; !exists {
		return fmt.Errorf("角色 '%s' 不存在", roleName)
	}
	return nil
}

// GetResourceFromPath 从API路径提取资源名称
func GetResourceFromPath(path string) string {
	// 移除前缀 /api/v1/
	path = strings.TrimPrefix(path, "/api/v1/")

	// 提取第一个路径段作为资源名称
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// GetActionFromMethod 从HTTP方法获取操作类型
func GetActionFromMethod(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "read"
	case "POST":
		return "write"
	case "PUT", "PATCH":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return "read"
	}
}

// RequirePermission 权限检查中间件工厂
func RequirePermission(rbac *RBAC, resource, action string) func(c *gin.Context) {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(401, gin.H{"error": "未找到用户角色信息"})
			c.Abort()
			return
		}

		role := userRole.(string)
		if !rbac.HasPermission(role, resource, action) {
			c.JSON(403, gin.H{
				"error": fmt.Sprintf("权限不足：需要 %s:%s 权限", resource, action),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
