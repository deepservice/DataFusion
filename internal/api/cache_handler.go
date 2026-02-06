package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/datafusion/worker/internal/cache"
	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheHandler 缓存管理处理器
type CacheHandler struct {
	cache cache.Cache
	log   *logger.Logger
}

// NewCacheHandler 创建缓存处理器
func NewCacheHandler(cache cache.Cache, log *logger.Logger) *CacheHandler {
	return &CacheHandler{
		cache: cache,
		log:   log,
	}
}

// GetCacheStats 获取缓存统计信息
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.cache.GetStats()
	if err != nil {
		h.log.WithError(err).Error("获取缓存统计信息失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取缓存统计信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// FlushCache 清空缓存
func (h *CacheHandler) FlushCache(c *gin.Context) {
	err := h.cache.FlushAll()
	if err != nil {
		h.log.WithError(err).Error("清空缓存失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "清空缓存失败",
		})
		return
	}

	h.log.Info("缓存已清空")
	c.JSON(http.StatusOK, gin.H{
		"message": "缓存已清空",
	})
}

// GetCacheKey 获取缓存值
func (h *CacheHandler) GetCacheKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缓存键不能为空",
		})
		return
	}

	var value interface{}
	err := h.cache.Get(key, &value)
	if err != nil {
		if err == cache.ErrCacheNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "缓存键不存在",
			})
			return
		}

		h.log.WithError(err).Error("获取缓存值失败", zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取缓存值失败",
		})
		return
	}

	// 获取TTL
	ttl, _ := h.cache.GetTTL(key)

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
		"ttl":   ttl.Seconds(),
	})
}

// SetCacheKey 设置缓存值
func (h *CacheHandler) SetCacheKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缓存键不能为空",
		})
		return
	}

	var req struct {
		Value interface{} `json:"value" binding:"required"`
		TTL   int         `json:"ttl"` // 秒数，0表示永不过期
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误",
		})
		return
	}

	var expiration time.Duration
	if req.TTL > 0 {
		expiration = time.Duration(req.TTL) * time.Second
	}

	err := h.cache.Set(key, req.Value, expiration)
	if err != nil {
		h.log.WithError(err).Error("设置缓存值失败", zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "设置缓存值失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "缓存设置成功",
		"key":     key,
		"ttl":     req.TTL,
	})
}

// DeleteCacheKey 删除缓存键
func (h *CacheHandler) DeleteCacheKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缓存键不能为空",
		})
		return
	}

	err := h.cache.Delete(key)
	if err != nil {
		h.log.WithError(err).Error("删除缓存键失败", zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除缓存键失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "缓存键删除成功",
		"key":     key,
	})
}

// CheckCacheKey 检查缓存键是否存在
func (h *CacheHandler) CheckCacheKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缓存键不能为空",
		})
		return
	}

	exists, err := h.cache.Exists(key)
	if err != nil {
		h.log.WithError(err).Error("检查缓存键存在性失败", zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "检查缓存键存在性失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":    key,
		"exists": exists,
	})
}

// IncrementCounter 递增计数器
func (h *CacheHandler) IncrementCounter(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缓存键不能为空",
		})
		return
	}

	// 获取可选的过期时间
	ttlStr := c.Query("ttl")
	var value int64
	var err error

	if ttlStr != "" {
		ttl, parseErr := strconv.Atoi(ttlStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "TTL参数格式错误",
			})
			return
		}

		expiration := time.Duration(ttl) * time.Second
		value, err = h.cache.IncrementWithExpire(key, expiration)
	} else {
		value, err = h.cache.Increment(key)
	}

	if err != nil {
		h.log.WithError(err).Error("递增计数器失败", zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "递增计数器失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
	})
}

// PingCache 检查缓存连接
func (h *CacheHandler) PingCache(c *gin.Context) {
	err := h.cache.Ping()
	if err != nil {
		h.log.WithError(err).Error("缓存连接检查失败")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  "缓存连接失败",
			"status": "unhealthy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "缓存连接正常",
	})
}
