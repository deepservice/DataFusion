package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/datafusion/worker/internal/models"
)

// FileStorage 文件存储
type FileStorage struct {
	basePath string
}

// NewFileStorage 创建文件存储
func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{basePath: basePath}
}

// Type 返回存储类型
func (f *FileStorage) Type() string {
	return "file"
}

// Store 存储数据到文件
func (f *FileStorage) Store(ctx context.Context, config *models.StorageConfig, data []map[string]interface{}) error {
	if len(data) == 0 {
		log.Println("没有数据需要存储")
		return nil
	}

	log.Printf("开始存储数据到文件，数据量: %d", len(data))

	// 创建目录
	dirPath := filepath.Join(f.basePath, config.Database)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成文件名（使用时间戳）
	filename := fmt.Sprintf("%s_%s.json", config.Table, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(dirPath, filename)

	// 写入文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	log.Printf("数据存储完成，文件: %s", filePath)
	return nil
}
