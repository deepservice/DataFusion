package unit

import (
	"testing"

	"github.com/datafusion/worker/internal/storage"
)

func TestStorageFactory(t *testing.T) {
	t.Run("创建存储工厂", func(t *testing.T) {
		factory := storage.NewStorageFactory()
		if factory == nil {
			t.Fatal("创建存储工厂失败")
		}
	})

	t.Run("注册和获取存储", func(t *testing.T) {
		factory := storage.NewStorageFactory()
		
		// 注册文件存储
		fileStorage := storage.NewFileStorage("./test_data")
		factory.Register(fileStorage)

		// 获取存储
		s, ok := factory.Get("file")
		if !ok {
			t.Error("获取文件存储失败")
		}
		if s.Type() != "file" {
			t.Errorf("存储类型错误，期望 'file'，得到 '%s'", s.Type())
		}
	})

	t.Run("获取不存在的存储", func(t *testing.T) {
		factory := storage.NewStorageFactory()
		
		_, ok := factory.Get("nonexistent")
		if ok {
			t.Error("不应该获取到不存在的存储")
		}
	})
}

func TestFileStorage(t *testing.T) {
	t.Run("创建文件存储", func(t *testing.T) {
		s := storage.NewFileStorage("./test_data")
		if s == nil {
			t.Fatal("创建文件存储失败")
		}
		if s.Type() != "file" {
			t.Errorf("存储类型错误，期望 'file'，得到 '%s'", s.Type())
		}
	})
}
