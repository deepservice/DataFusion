package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/datafusion/worker/internal/logger"
	"github.com/go-redis/redis/v8"
)

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
	log    *logger.Logger
	ctx    context.Context
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(config *RedisConfig, log *logger.Logger) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	return &RedisCache{
		client: rdb,
		log:    log,
		ctx:    context.Background(),
	}
}

// Set 设置缓存
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	err = r.client.Set(r.ctx, key, data, expiration).Err()
	if err != nil {
		r.log.WithError(err).Error("设置Redis缓存失败")
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// Get 获取缓存
func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheNotFound
		}
		r.log.WithError(err).Error("获取Redis缓存失败")
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("反序列化缓存数据失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (r *RedisCache) Delete(key string) error {
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		r.log.WithError(err).Error("删除Redis缓存失败")
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	return nil
}

// Exists 检查缓存是否存在
func (r *RedisCache) Exists(key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		r.log.WithError(err).Error("检查Redis缓存存在性失败")
		return false, fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	return count > 0, nil
}

// SetWithTTL 设置带TTL的缓存
func (r *RedisCache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	return r.Set(key, value, ttl)
}

// GetTTL 获取缓存TTL
func (r *RedisCache) GetTTL(key string) (time.Duration, error) {
	ttl, err := r.client.TTL(r.ctx, key).Result()
	if err != nil {
		r.log.WithError(err).Error("获取Redis缓存TTL失败")
		return 0, fmt.Errorf("获取缓存TTL失败: %w", err)
	}

	return ttl, nil
}

// Increment 递增计数器
func (r *RedisCache) Increment(key string) (int64, error) {
	val, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		r.log.WithError(err).Error("递增Redis计数器失败")
		return 0, fmt.Errorf("递增计数器失败: %w", err)
	}

	return val, nil
}

// IncrementWithExpire 递增计数器并设置过期时间
func (r *RedisCache) IncrementWithExpire(key string, expiration time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(r.ctx, key)
	pipe.Expire(r.ctx, key, expiration)

	_, err := pipe.Exec(r.ctx)
	if err != nil {
		r.log.WithError(err).Error("递增Redis计数器并设置过期时间失败")
		return 0, fmt.Errorf("递增计数器并设置过期时间失败: %w", err)
	}

	return incrCmd.Val(), nil
}

// SetHash 设置哈希字段
func (r *RedisCache) SetHash(key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化哈希数据失败: %w", err)
	}

	err = r.client.HSet(r.ctx, key, field, data).Err()
	if err != nil {
		r.log.WithError(err).Error("设置Redis哈希失败")
		return fmt.Errorf("设置哈希失败: %w", err)
	}

	return nil
}

// GetHash 获取哈希字段
func (r *RedisCache) GetHash(key, field string, dest interface{}) error {
	data, err := r.client.HGet(r.ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheNotFound
		}
		r.log.WithError(err).Error("获取Redis哈希失败")
		return fmt.Errorf("获取哈希失败: %w", err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("反序列化哈希数据失败: %w", err)
	}

	return nil
}

// GetAllHash 获取所有哈希字段
func (r *RedisCache) GetAllHash(key string) (map[string]string, error) {
	data, err := r.client.HGetAll(r.ctx, key).Result()
	if err != nil {
		r.log.WithError(err).Error("获取Redis所有哈希字段失败")
		return nil, fmt.Errorf("获取所有哈希字段失败: %w", err)
	}

	return data, nil
}

// DeleteHash 删除哈希字段
func (r *RedisCache) DeleteHash(key, field string) error {
	err := r.client.HDel(r.ctx, key, field).Err()
	if err != nil {
		r.log.WithError(err).Error("删除Redis哈希字段失败")
		return fmt.Errorf("删除哈希字段失败: %w", err)
	}

	return nil
}

// SetList 设置列表
func (r *RedisCache) SetList(key string, values ...interface{}) error {
	// 先清空列表
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("清空列表失败: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	// 序列化所有值
	serializedValues := make([]interface{}, len(values))
	for i, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("序列化列表数据失败: %w", err)
		}
		serializedValues[i] = data
	}

	err = r.client.RPush(r.ctx, key, serializedValues...).Err()
	if err != nil {
		r.log.WithError(err).Error("设置Redis列表失败")
		return fmt.Errorf("设置列表失败: %w", err)
	}

	return nil
}

// GetList 获取列表
func (r *RedisCache) GetList(key string, start, stop int64) ([]string, error) {
	data, err := r.client.LRange(r.ctx, key, start, stop).Result()
	if err != nil {
		r.log.WithError(err).Error("获取Redis列表失败")
		return nil, fmt.Errorf("获取列表失败: %w", err)
	}

	return data, nil
}

// PushList 向列表添加元素
func (r *RedisCache) PushList(key string, values ...interface{}) error {
	serializedValues := make([]interface{}, len(values))
	for i, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("序列化列表数据失败: %w", err)
		}
		serializedValues[i] = data
	}

	err := r.client.RPush(r.ctx, key, serializedValues...).Err()
	if err != nil {
		r.log.WithError(err).Error("向Redis列表添加元素失败")
		return fmt.Errorf("向列表添加元素失败: %w", err)
	}

	return nil
}

// PopList 从列表弹出元素
func (r *RedisCache) PopList(key string) (string, error) {
	data, err := r.client.LPop(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrCacheNotFound
		}
		r.log.WithError(err).Error("从Redis列表弹出元素失败")
		return "", fmt.Errorf("从列表弹出元素失败: %w", err)
	}

	return data, nil
}

// GetStats 获取缓存统计信息
func (r *RedisCache) GetStats() (map[string]interface{}, error) {
	info, err := r.client.Info(r.ctx, "memory", "stats").Result()
	if err != nil {
		r.log.WithError(err).Error("获取Redis统计信息失败")
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}

	// 解析INFO命令的输出
	stats := make(map[string]interface{})
	stats["info"] = info

	// 获取数据库大小
	dbSize, err := r.client.DBSize(r.ctx).Result()
	if err == nil {
		stats["db_size"] = dbSize
	}

	return stats, nil
}

// Ping 检查连接
func (r *RedisCache) Ping() error {
	_, err := r.client.Ping(r.ctx).Result()
	if err != nil {
		r.log.WithError(err).Error("Redis连接检查失败")
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	return nil
}

// Close 关闭连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// FlushAll 清空所有缓存（谨慎使用）
func (r *RedisCache) FlushAll() error {
	err := r.client.FlushAll(r.ctx).Err()
	if err != nil {
		r.log.WithError(err).Error("清空Redis缓存失败")
		return fmt.Errorf("清空缓存失败: %w", err)
	}

	r.log.Info("已清空所有Redis缓存")
	return nil
}
