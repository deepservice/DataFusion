package storage

import (
	"context"

	"github.com/datafusion/worker/internal/models"
)

// Storage 存储接口
type Storage interface {
	// Store 存储数据
	Store(ctx context.Context, config *models.StorageConfig, data []map[string]interface{}) error
	
	// Type 返回存储类型
	Type() string
}

// StorageFactory 存储工厂
type StorageFactory struct {
	storages map[string]Storage
}

// NewStorageFactory 创建存储工厂
func NewStorageFactory() *StorageFactory {
	return &StorageFactory{
		storages: make(map[string]Storage),
	}
}

// Register 注册存储
func (f *StorageFactory) Register(storage Storage) {
	f.storages[storage.Type()] = storage
}

// Get 获取存储
func (f *StorageFactory) Get(storageType string) (Storage, bool) {
	storage, ok := f.storages[storageType]
	return storage, ok
}
