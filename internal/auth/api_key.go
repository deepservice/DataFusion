package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// APIKey API密钥结构
type APIKey struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	KeyHash     string     `json:"-"` // 不在JSON中显示
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	Status      string     `json:"status"` // active, revoked
	CreatedAt   time.Time  `json:"created_at"`
}

// APIKeyManager API密钥管理器
type APIKeyManager struct {
	// 可以添加数据库连接等依赖
}

// NewAPIKeyManager 创建API密钥管理器
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{}
}

// GenerateAPIKey 生成API密钥
func (m *APIKeyManager) GenerateAPIKey() (string, string, error) {
	// 生成32字节随机数据
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("生成随机数据失败: %w", err)
	}

	// 创建API密钥（以df_开头，便于识别）
	apiKey := "df_" + hex.EncodeToString(randomBytes)

	// 计算哈希值用于存储
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	return apiKey, keyHash, nil
}

// HashAPIKey 计算API密钥的哈希值
func (m *APIKeyManager) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKey 验证API密钥格式
func (m *APIKeyManager) ValidateAPIKey(apiKey string) error {
	if len(apiKey) < 10 {
		return fmt.Errorf("API密钥长度不足")
	}

	if apiKey[:3] != "df_" {
		return fmt.Errorf("无效的API密钥格式")
	}

	return nil
}

// CreateAPIKey 创建API密钥记录
func (m *APIKeyManager) CreateAPIKey(userID int64, name, description string, permissions []string, expiresAt *time.Time) (*APIKey, string, error) {
	// 生成API密钥
	apiKey, keyHash, err := m.GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	// 创建API密钥记录
	key := &APIKey{
		UserID:      userID,
		Name:        name,
		Description: description,
		KeyHash:     keyHash,
		Permissions: permissions,
		ExpiresAt:   expiresAt,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	return key, apiKey, nil
}

// IsExpired 检查API密钥是否过期
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive 检查API密钥是否激活
func (k *APIKey) IsActive() bool {
	return k.Status == "active" && !k.IsExpired()
}

// UpdateLastUsed 更新最后使用时间
func (k *APIKey) UpdateLastUsed() {
	now := time.Now()
	k.LastUsedAt = &now
}

// Revoke 撤销API密钥
func (k *APIKey) Revoke() {
	k.Status = "revoked"
}

// HasPermission 检查API密钥是否有指定权限
func (k *APIKey) HasPermission(permission string) bool {
	for _, p := range k.Permissions {
		if p == "*" || p == permission {
			return true
		}
	}
	return false
}

// APIKeyMiddleware API密钥认证中间件
func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从头部获取API密钥
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 也可以从查询参数获取
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(401, gin.H{"error": "缺少API密钥"})
			c.Abort()
			return
		}

		// 验证API密钥格式
		manager := NewAPIKeyManager()
		if err := manager.ValidateAPIKey(apiKey); err != nil {
			c.JSON(401, gin.H{"error": "无效的API密钥格式"})
			c.Abort()
			return
		}

		// 计算哈希值
		keyHash := manager.HashAPIKey(apiKey)

		// TODO: 从数据库查询API密钥信息
		// 这里需要实现数据库查询逻辑
		// key, err := db.GetAPIKeyByHash(keyHash)
		// if err != nil {
		//     c.JSON(401, gin.H{"error": "无效的API密钥"})
		//     c.Abort()
		//     return
		// }

		// if !key.IsActive() {
		//     c.JSON(401, gin.H{"error": "API密钥已过期或被撤销"})
		//     c.Abort()
		//     return
		// }

		// 更新最后使用时间
		// key.UpdateLastUsed()
		// db.UpdateAPIKey(key)

		// 将API密钥信息存储到上下文
		c.Set("api_key_hash", keyHash)
		// c.Set("api_key_user_id", key.UserID)
		// c.Set("api_key_permissions", key.Permissions)

		c.Next()
	}
}
