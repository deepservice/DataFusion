package cache

import (
	"errors"
	"time"
)

// 缓存错误定义
var (
	ErrCacheNotFound = errors.New("缓存未找到")
	ErrCacheExpired  = errors.New("缓存已过期")
)

// Cache 缓存接口
type Cache interface {
	// 基础操作
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, dest interface{}) error
	Delete(key string) error
	Exists(key string) (bool, error)

	// TTL操作
	SetWithTTL(key string, value interface{}, ttl time.Duration) error
	GetTTL(key string) (time.Duration, error)

	// 计数器操作
	Increment(key string) (int64, error)
	IncrementWithExpire(key string, expiration time.Duration) (int64, error)

	// 哈希操作
	SetHash(key, field string, value interface{}) error
	GetHash(key, field string, dest interface{}) error
	GetAllHash(key string) (map[string]string, error)
	DeleteHash(key, field string) error

	// 列表操作
	SetList(key string, values ...interface{}) error
	GetList(key string, start, stop int64) ([]string, error)
	PushList(key string, values ...interface{}) error
	PopList(key string) (string, error)

	// 管理操作
	GetStats() (map[string]interface{}, error)
	Ping() error
	Close() error
	FlushAll() error
}

// CacheManager 缓存管理器
type CacheManager struct {
	primary   Cache
	secondary Cache // 可选的二级缓存
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(primary Cache, secondary ...Cache) *CacheManager {
	cm := &CacheManager{
		primary: primary,
	}

	if len(secondary) > 0 {
		cm.secondary = secondary[0]
	}

	return cm
}

// Set 设置缓存（写入主缓存和二级缓存）
func (cm *CacheManager) Set(key string, value interface{}, expiration time.Duration) error {
	// 写入主缓存
	err := cm.primary.Set(key, value, expiration)
	if err != nil {
		return err
	}

	// 写入二级缓存（如果存在）
	if cm.secondary != nil {
		// 二级缓存失败不影响主流程
		cm.secondary.Set(key, value, expiration)
	}

	return nil
}

// Get 获取缓存（优先从主缓存获取，失败时从二级缓存获取）
func (cm *CacheManager) Get(key string, dest interface{}) error {
	// 先从主缓存获取
	err := cm.primary.Get(key, dest)
	if err == nil {
		return nil
	}

	// 如果主缓存失败且有二级缓存，尝试从二级缓存获取
	if cm.secondary != nil && err == ErrCacheNotFound {
		err = cm.secondary.Get(key, dest)
		if err == nil {
			// 从二级缓存获取成功，回写到主缓存
			cm.primary.Set(key, dest, time.Hour) // 默认1小时过期
			return nil
		}
	}

	return err
}

// Delete 删除缓存（从所有缓存中删除）
func (cm *CacheManager) Delete(key string) error {
	// 删除主缓存
	err := cm.primary.Delete(key)

	// 删除二级缓存
	if cm.secondary != nil {
		cm.secondary.Delete(key) // 忽略错误
	}

	return err
}

// Exists 检查缓存是否存在
func (cm *CacheManager) Exists(key string) (bool, error) {
	return cm.primary.Exists(key)
}

// 其他方法直接委托给主缓存
func (cm *CacheManager) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	return cm.primary.SetWithTTL(key, value, ttl)
}

func (cm *CacheManager) GetTTL(key string) (time.Duration, error) {
	return cm.primary.GetTTL(key)
}

func (cm *CacheManager) Increment(key string) (int64, error) {
	return cm.primary.Increment(key)
}

func (cm *CacheManager) IncrementWithExpire(key string, expiration time.Duration) (int64, error) {
	return cm.primary.IncrementWithExpire(key, expiration)
}

func (cm *CacheManager) SetHash(key, field string, value interface{}) error {
	return cm.primary.SetHash(key, field, value)
}

func (cm *CacheManager) GetHash(key, field string, dest interface{}) error {
	return cm.primary.GetHash(key, field, dest)
}

func (cm *CacheManager) GetAllHash(key string) (map[string]string, error) {
	return cm.primary.GetAllHash(key)
}

func (cm *CacheManager) DeleteHash(key, field string) error {
	return cm.primary.DeleteHash(key, field)
}

func (cm *CacheManager) SetList(key string, values ...interface{}) error {
	return cm.primary.SetList(key, values...)
}

func (cm *CacheManager) GetList(key string, start, stop int64) ([]string, error) {
	return cm.primary.GetList(key, start, stop)
}

func (cm *CacheManager) PushList(key string, values ...interface{}) error {
	return cm.primary.PushList(key, values...)
}

func (cm *CacheManager) PopList(key string) (string, error) {
	return cm.primary.PopList(key)
}

func (cm *CacheManager) GetStats() (map[string]interface{}, error) {
	return cm.primary.GetStats()
}

func (cm *CacheManager) Ping() error {
	return cm.primary.Ping()
}

func (cm *CacheManager) Close() error {
	err := cm.primary.Close()
	if cm.secondary != nil {
		cm.secondary.Close()
	}
	return err
}

func (cm *CacheManager) FlushAll() error {
	err := cm.primary.FlushAll()
	if cm.secondary != nil {
		cm.secondary.FlushAll()
	}
	return err
}

// CacheKey 缓存键生成器
type CacheKey struct {
	prefix string
}

// NewCacheKey 创建缓存键生成器
func NewCacheKey(prefix string) *CacheKey {
	return &CacheKey{prefix: prefix}
}

// Key 生成缓存键
func (ck *CacheKey) Key(parts ...string) string {
	key := ck.prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// UserKey 用户相关缓存键
func (ck *CacheKey) UserKey(userID int64, suffix string) string {
	return ck.Key("user", string(rune(userID)), suffix)
}

// TaskKey 任务相关缓存键
func (ck *CacheKey) TaskKey(taskID int64, suffix string) string {
	return ck.Key("task", string(rune(taskID)), suffix)
}

// StatsKey 统计相关缓存键
func (ck *CacheKey) StatsKey(suffix string) string {
	return ck.Key("stats", suffix)
}

// ConfigKey 配置相关缓存键
func (ck *CacheKey) ConfigKey(suffix string) string {
	return ck.Key("config", suffix)
}
