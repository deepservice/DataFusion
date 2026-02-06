package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCacheItem 内存缓存项
type MemoryCacheItem struct {
	Value      interface{}
	Expiration int64
}

// IsExpired 检查是否过期
func (item *MemoryCacheItem) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	items   map[string]*MemoryCacheItem
	mu      sync.RWMutex
	janitor *janitor
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	mc := &MemoryCache{
		items: make(map[string]*MemoryCacheItem),
	}

	// 启动清理协程
	if cleanupInterval > 0 {
		mc.janitor = &janitor{
			Interval: cleanupInterval,
			stop:     make(chan bool),
		}
		go mc.janitor.Run(mc)
	}

	return mc
}

// Set 设置缓存
func (mc *MemoryCache) Set(key string, value interface{}, expiration time.Duration) error {
	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items[key] = &MemoryCacheItem{
		Value:      value,
		Expiration: exp,
	}

	return nil
}

// Get 获取缓存
func (mc *MemoryCache) Get(key string, dest interface{}) error {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return ErrCacheNotFound
	}

	if item.IsExpired() {
		mc.Delete(key)
		return ErrCacheNotFound
	}

	// 使用JSON进行类型转换
	data, err := json.Marshal(item.Value)
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("反序列化缓存数据失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.items, key)
	return nil
}

// Exists 检查缓存是否存在
func (mc *MemoryCache) Exists(key string) (bool, error) {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return false, nil
	}

	if item.IsExpired() {
		mc.Delete(key)
		return false, nil
	}

	return true, nil
}

// SetWithTTL 设置带TTL的缓存
func (mc *MemoryCache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	return mc.Set(key, value, ttl)
}

// GetTTL 获取缓存TTL
func (mc *MemoryCache) GetTTL(key string) (time.Duration, error) {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return 0, ErrCacheNotFound
	}

	if item.Expiration == 0 {
		return 0, nil // 永不过期
	}

	remaining := time.Duration(item.Expiration - time.Now().UnixNano())
	if remaining <= 0 {
		mc.Delete(key)
		return 0, ErrCacheNotFound
	}

	return remaining, nil
}

// Increment 递增计数器
func (mc *MemoryCache) Increment(key string) (int64, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, found := mc.items[key]
	if !found {
		mc.items[key] = &MemoryCacheItem{
			Value:      int64(1),
			Expiration: 0,
		}
		return 1, nil
	}

	if item.IsExpired() {
		mc.items[key] = &MemoryCacheItem{
			Value:      int64(1),
			Expiration: 0,
		}
		return 1, nil
	}

	// 尝试转换为int64
	switch v := item.Value.(type) {
	case int64:
		v++
		item.Value = v
		return v, nil
	case int:
		v++
		item.Value = int64(v)
		return int64(v), nil
	case float64:
		v++
		item.Value = int64(v)
		return int64(v), nil
	default:
		return 0, fmt.Errorf("无法递增非数字类型的值")
	}
}

// IncrementWithExpire 递增计数器并设置过期时间
func (mc *MemoryCache) IncrementWithExpire(key string, expiration time.Duration) (int64, error) {
	val, err := mc.Increment(key)
	if err != nil {
		return 0, err
	}

	// 设置过期时间
	if expiration > 0 {
		mc.mu.Lock()
		if item, found := mc.items[key]; found {
			item.Expiration = time.Now().Add(expiration).UnixNano()
		}
		mc.mu.Unlock()
	}

	return val, nil
}

// SetHash 设置哈希字段（内存缓存中使用嵌套map模拟）
func (mc *MemoryCache) SetHash(key, field string, value interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, found := mc.items[key]
	if !found {
		hashMap := make(map[string]interface{})
		hashMap[field] = value
		mc.items[key] = &MemoryCacheItem{
			Value:      hashMap,
			Expiration: 0,
		}
		return nil
	}

	if item.IsExpired() {
		hashMap := make(map[string]interface{})
		hashMap[field] = value
		mc.items[key] = &MemoryCacheItem{
			Value:      hashMap,
			Expiration: 0,
		}
		return nil
	}

	// 检查是否为map类型
	if hashMap, ok := item.Value.(map[string]interface{}); ok {
		hashMap[field] = value
	} else {
		return fmt.Errorf("键 %s 不是哈希类型", key)
	}

	return nil
}

// GetHash 获取哈希字段
func (mc *MemoryCache) GetHash(key, field string, dest interface{}) error {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return ErrCacheNotFound
	}

	if item.IsExpired() {
		mc.Delete(key)
		return ErrCacheNotFound
	}

	hashMap, ok := item.Value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("键 %s 不是哈希类型", key)
	}

	value, found := hashMap[field]
	if !found {
		return ErrCacheNotFound
	}

	// 使用JSON进行类型转换
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化哈希数据失败: %w", err)
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("反序列化哈希数据失败: %w", err)
	}

	return nil
}

// GetAllHash 获取所有哈希字段
func (mc *MemoryCache) GetAllHash(key string) (map[string]string, error) {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return nil, ErrCacheNotFound
	}

	if item.IsExpired() {
		mc.Delete(key)
		return nil, ErrCacheNotFound
	}

	hashMap, ok := item.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("键 %s 不是哈希类型", key)
	}

	result := make(map[string]string)
	for k, v := range hashMap {
		data, err := json.Marshal(v)
		if err != nil {
			continue
		}
		result[k] = string(data)
	}

	return result, nil
}

// DeleteHash 删除哈希字段
func (mc *MemoryCache) DeleteHash(key, field string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, found := mc.items[key]
	if !found {
		return nil
	}

	if item.IsExpired() {
		delete(mc.items, key)
		return nil
	}

	hashMap, ok := item.Value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("键 %s 不是哈希类型", key)
	}

	delete(hashMap, field)
	return nil
}

// SetList 设置列表（内存缓存中使用slice）
func (mc *MemoryCache) SetList(key string, values ...interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items[key] = &MemoryCacheItem{
		Value:      values,
		Expiration: 0,
	}

	return nil
}

// GetList 获取列表
func (mc *MemoryCache) GetList(key string, start, stop int64) ([]string, error) {
	mc.mu.RLock()
	item, found := mc.items[key]
	mc.mu.RUnlock()

	if !found {
		return nil, ErrCacheNotFound
	}

	if item.IsExpired() {
		mc.Delete(key)
		return nil, ErrCacheNotFound
	}

	list, ok := item.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("键 %s 不是列表类型", key)
	}

	// 处理索引
	length := int64(len(list))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []string{}, nil
	}

	result := make([]string, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		data, err := json.Marshal(list[i])
		if err != nil {
			continue
		}
		result = append(result, string(data))
	}

	return result, nil
}

// PushList 向列表添加元素
func (mc *MemoryCache) PushList(key string, values ...interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, found := mc.items[key]
	if !found {
		mc.items[key] = &MemoryCacheItem{
			Value:      values,
			Expiration: 0,
		}
		return nil
	}

	if item.IsExpired() {
		mc.items[key] = &MemoryCacheItem{
			Value:      values,
			Expiration: 0,
		}
		return nil
	}

	list, ok := item.Value.([]interface{})
	if !ok {
		return fmt.Errorf("键 %s 不是列表类型", key)
	}

	list = append(list, values...)
	item.Value = list

	return nil
}

// PopList 从列表弹出元素
func (mc *MemoryCache) PopList(key string) (string, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, found := mc.items[key]
	if !found {
		return "", ErrCacheNotFound
	}

	if item.IsExpired() {
		delete(mc.items, key)
		return "", ErrCacheNotFound
	}

	list, ok := item.Value.([]interface{})
	if !ok {
		return "", fmt.Errorf("键 %s 不是列表类型", key)
	}

	if len(list) == 0 {
		return "", ErrCacheNotFound
	}

	// 弹出第一个元素
	value := list[0]
	item.Value = list[1:]

	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("序列化列表数据失败: %w", err)
	}

	return string(data), nil
}

// GetStats 获取缓存统计信息
func (mc *MemoryCache) GetStats() (map[string]interface{}, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := map[string]interface{}{
		"type":       "memory",
		"item_count": len(mc.items),
	}

	// 计算过期项数量
	expiredCount := 0
	for _, item := range mc.items {
		if item.IsExpired() {
			expiredCount++
		}
	}
	stats["expired_count"] = expiredCount

	return stats, nil
}

// Ping 检查连接（内存缓存总是可用）
func (mc *MemoryCache) Ping() error {
	return nil
}

// Close 关闭缓存
func (mc *MemoryCache) Close() error {
	if mc.janitor != nil {
		mc.janitor.stop <- true
	}
	return nil
}

// FlushAll 清空所有缓存
func (mc *MemoryCache) FlushAll() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items = make(map[string]*MemoryCacheItem)
	return nil
}

// deleteExpired 删除过期项
func (mc *MemoryCache) deleteExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for key, item := range mc.items {
		if item.IsExpired() {
			delete(mc.items, key)
		}
	}
}

// janitor 清理协程
type janitor struct {
	Interval time.Duration
	stop     chan bool
}

// Run 运行清理协程
func (j *janitor) Run(mc *MemoryCache) {
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.deleteExpired()
		case <-j.stop:
			return
		}
	}
}
