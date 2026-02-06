package cache

import (
	"fmt"
	"time"

	"github.com/datafusion/worker/internal/config"
	"github.com/datafusion/worker/internal/logger"
	"go.uber.org/zap"
)

// CacheFactory 缓存工厂
type CacheFactory struct {
	log *logger.Logger
}

// NewCacheFactory 创建缓存工厂
func NewCacheFactory(log *logger.Logger) *CacheFactory {
	return &CacheFactory{
		log: log,
	}
}

// CreateCache 根据配置创建缓存实例
func (cf *CacheFactory) CreateCache(cfg *config.CacheConfig) (Cache, error) {
	switch cfg.Type {
	case "redis":
		return cf.createRedisCache(&cfg.Redis)
	case "memory":
		return cf.createMemoryCache(&cfg.Memory)
	case "hybrid":
		return cf.createHybridCache(&cfg.Redis, &cfg.Memory)
	default:
		cf.log.Warn("未知的缓存类型，使用内存缓存", zap.String("type", cfg.Type))
		return cf.createMemoryCache(&cfg.Memory)
	}
}

// createRedisCache 创建Redis缓存
func (cf *CacheFactory) createRedisCache(cfg *config.RedisConfig) (Cache, error) {
	redisConfig := &RedisConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}

	cache := NewRedisCache(redisConfig, cf.log)

	// 测试连接
	if err := cache.Ping(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	cf.log.Info("Redis缓存初始化成功", zap.String("host", cfg.Host), zap.Int("port", cfg.Port))
	return cache, nil
}

// createMemoryCache 创建内存缓存
func (cf *CacheFactory) createMemoryCache(cfg *config.MemoryConfig) (Cache, error) {
	cleanupInterval, err := time.ParseDuration(cfg.CleanupInterval)
	if err != nil {
		cf.log.Warn("解析清理间隔失败，使用默认值", zap.String("interval", cfg.CleanupInterval), zap.Error(err))
		cleanupInterval = 10 * time.Minute
	}

	cache := NewMemoryCache(cleanupInterval)
	cf.log.Info("内存缓存初始化成功", zap.Duration("cleanup_interval", cleanupInterval))
	return cache, nil
}

// createHybridCache 创建混合缓存（Redis + Memory）
func (cf *CacheFactory) createHybridCache(redisCfg *config.RedisConfig, memoryCfg *config.MemoryConfig) (Cache, error) {
	// 创建Redis缓存作为主缓存
	primaryCache, err := cf.createRedisCache(redisCfg)
	if err != nil {
		cf.log.Warn("Redis缓存创建失败，降级为纯内存缓存", zap.Error(err))
		return cf.createMemoryCache(memoryCfg)
	}

	// 创建内存缓存作为二级缓存
	secondaryCache, err := cf.createMemoryCache(memoryCfg)
	if err != nil {
		cf.log.Warn("内存缓存创建失败，使用纯Redis缓存", zap.Error(err))
		return primaryCache, nil
	}

	// 创建缓存管理器
	cacheManager := NewCacheManager(primaryCache, secondaryCache)
	cf.log.Info("混合缓存初始化成功", zap.String("primary", "redis"), zap.String("secondary", "memory"))
	return cacheManager, nil
}

// CreateCacheWithFallback 创建带降级的缓存
func (cf *CacheFactory) CreateCacheWithFallback(cfg *config.CacheConfig) Cache {
	cache, err := cf.CreateCache(cfg)
	if err != nil {
		cf.log.Error("缓存创建失败，降级为内存缓存", zap.Error(err))

		// 降级为内存缓存
		fallbackCache := NewMemoryCache(10 * time.Minute)
		cf.log.Info("降级缓存初始化成功", zap.String("type", "memory"))
		return fallbackCache
	}

	return cache
}
