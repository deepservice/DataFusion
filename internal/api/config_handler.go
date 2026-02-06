package api

import (
	"net/http"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
)

// ConfigHandler 配置管理处理器
type ConfigHandler struct {
	dynamicConfig *config.DynamicConfig
	log           *logger.Logger
}

// NewConfigHandler 创建配置管理处理器
func NewConfigHandler(dynamicConfig *config.DynamicConfig, log *logger.Logger) *ConfigHandler {
	return &ConfigHandler{
		dynamicConfig: dynamicConfig,
		log:           log,
	}
}

// GetConfig 获取当前配置
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	configJSON, err := h.dynamicConfig.GetConfigJSON()
	if err != nil {
		h.log.WithError(err).Error("获取配置JSON失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置失败"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", configJSON)
}

// ValidateConfig 验证配置
func (h *ConfigHandler) ValidateConfig(c *gin.Context) {
	var newConfig config.APIServerConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置格式"})
		return
	}

	validator := config.NewConfigValidator()
	result := validator.ValidateConfig(&newConfig)

	c.JSON(http.StatusOK, gin.H{
		"valid":           result.Valid,
		"errors":          result.Errors,
		"recommendations": config.GetConfigRecommendations(&newConfig),
	})
}

// UpdateConfig 更新配置
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var newConfig config.APIServerConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置格式"})
		return
	}

	// 验证配置
	validator := config.NewConfigValidator()
	result := validator.ValidateConfig(&newConfig)

	if !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "配置验证失败",
			"errors": result.Errors,
		})
		return
	}

	// 更新配置
	if err := h.dynamicConfig.UpdateConfig(&newConfig); err != nil {
		h.log.WithError(err).Error("更新配置失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "配置更新成功",
		"recommendations": config.GetConfigRecommendations(&newConfig),
	})
}

// ReloadConfig 重新加载配置
func (h *ConfigHandler) ReloadConfig(c *gin.Context) {
	// 这里可以实现从文件重新加载配置的逻辑
	// 由于我们已经有文件监听器，这个端点主要用于手动触发重载

	currentConfig := h.dynamicConfig.GetConfig()

	// 应用环境变量覆盖
	config.LoadFromEnv(currentConfig)

	// 验证配置
	validator := config.NewConfigValidator()
	result := validator.ValidateConfig(currentConfig)

	if !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "重新加载的配置验证失败",
			"errors": result.Errors,
		})
		return
	}

	// 更新配置
	if err := h.dynamicConfig.UpdateConfig(currentConfig); err != nil {
		h.log.WithError(err).Error("重新加载配置失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重新加载配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置重新加载成功"})
}

// GetConfigSchema 获取配置结构说明
func (h *ConfigHandler) GetConfigSchema(c *gin.Context) {
	schema := map[string]interface{}{
		"server": map[string]interface{}{
			"port": map[string]interface{}{
				"type":        "integer",
				"description": "服务器监听端口",
				"range":       "1-65535",
				"default":     8080,
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "运行模式",
				"options":     []string{"debug", "release", "test"},
				"default":     "debug",
			},
			"read_timeout": map[string]interface{}{
				"type":        "integer",
				"description": "读取超时时间（秒）",
				"range":       "1-300",
				"default":     30,
			},
			"write_timeout": map[string]interface{}{
				"type":        "integer",
				"description": "写入超时时间（秒）",
				"range":       "1-300",
				"default":     30,
			},
		},
		"auth": map[string]interface{}{
			"jwt": map[string]interface{}{
				"secret_key": map[string]interface{}{
					"type":        "string",
					"description": "JWT签名密钥",
					"min_length":  32,
					"required":    true,
				},
				"token_duration": map[string]interface{}{
					"type":        "string",
					"description": "Token有效期",
					"format":      "duration",
					"default":     "24h",
				},
			},
			"password": map[string]interface{}{
				"min_length": map[string]interface{}{
					"type":        "integer",
					"description": "密码最小长度",
					"range":       "6-128",
					"default":     8,
				},
				"require_upper": map[string]interface{}{
					"type":        "boolean",
					"description": "是否要求大写字母",
					"default":     true,
				},
				"require_lower": map[string]interface{}{
					"type":        "boolean",
					"description": "是否要求小写字母",
					"default":     true,
				},
				"require_digit": map[string]interface{}{
					"type":        "boolean",
					"description": "是否要求数字",
					"default":     true,
				},
				"require_special": map[string]interface{}{
					"type":        "boolean",
					"description": "是否要求特殊字符",
					"default":     false,
				},
			},
		},
		"database": map[string]interface{}{
			"postgresql": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "数据库主机",
					"default":     "localhost",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "数据库端口",
					"range":       "1-65535",
					"default":     5432,
				},
				"user": map[string]interface{}{
					"type":        "string",
					"description": "数据库用户名",
					"required":    true,
				},
				"password": map[string]interface{}{
					"type":        "string",
					"description": "数据库密码",
					"required":    true,
					"sensitive":   true,
				},
				"database": map[string]interface{}{
					"type":        "string",
					"description": "数据库名",
					"required":    true,
				},
				"sslmode": map[string]interface{}{
					"type":        "string",
					"description": "SSL模式",
					"options":     []string{"disable", "require", "verify-ca", "verify-full"},
					"default":     "disable",
				},
			},
		},
		"log": map[string]interface{}{
			"level": map[string]interface{}{
				"type":        "string",
				"description": "日志级别",
				"options":     []string{"debug", "info", "warn", "error"},
				"default":     "info",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "日志格式",
				"options":     []string{"json", "console"},
				"default":     "console",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"schema":   schema,
		"env_vars": config.GetEnvExample(),
	})
}

// GetConfigStatus 获取配置状态
func (h *ConfigHandler) GetConfigStatus(c *gin.Context) {
	currentConfig := h.dynamicConfig.GetConfig()

	// 验证当前配置
	validator := config.NewConfigValidator()
	result := validator.ValidateConfig(currentConfig)

	// 获取环境变量验证结果
	envErrors := config.ValidateEnv()

	status := gin.H{
		"config_valid":    result.Valid,
		"env_valid":       len(envErrors) == 0,
		"errors":          result.Errors,
		"env_errors":      envErrors,
		"recommendations": config.GetConfigRecommendations(currentConfig),
	}

	if result.Valid && len(envErrors) == 0 {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusBadRequest, status)
	}
}
