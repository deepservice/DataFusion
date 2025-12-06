package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBStorage MongoDB 存储实现
type MongoDBStorage struct {
	pool       *Pool
	config     *Config
	collection string
}

// NewMongoDBStorage 创建 MongoDB 存储
func NewMongoDBStorage(config *Config) (*MongoDBStorage, error) {
	pool, err := NewPool(config)
	if err != nil {
		return nil, fmt.Errorf("创建连接池失败: %w", err)
	}

	return &MongoDBStorage{
		pool:       pool,
		config:     config,
		collection: config.Collection,
	}, nil
}

// Type 返回存储类型
func (m *MongoDBStorage) Type() string {
	return "mongodb"
}

// Store 存储数据
func (m *MongoDBStorage) Store(ctx context.Context, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	log.Printf("开始存储数据到 MongoDB，共 %d 条", len(data))

	coll, err := m.pool.GetCollection()
	if err != nil {
		return fmt.Errorf("获取集合失败: %w", err)
	}

	// 添加时间戳
	now := time.Now()
	documents := make([]interface{}, 0, len(data))
	for _, item := range data {
		doc := make(map[string]interface{})
		for k, v := range item {
			doc[k] = v
		}
		doc["_created_at"] = now
		doc["_updated_at"] = now
		documents = append(documents, doc)
	}

	// 批量插入
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	opts := options.InsertMany().SetOrdered(false) // 允许部分失败
	result, err := coll.InsertMany(ctx, documents, opts)
	if err != nil {
		// 检查是否是部分成功
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("部分数据已存在，成功插入 %d 条", len(result.InsertedIDs))
			return nil
		}
		return fmt.Errorf("插入数据失败: %w", err)
	}

	log.Printf("成功存储 %d 条数据到 MongoDB", len(result.InsertedIDs))
	return nil
}

// Query 查询数据
func (m *MongoDBStorage) Query(ctx context.Context, filter map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	coll, err := m.pool.GetCollection()
	if err != nil {
		return nil, fmt.Errorf("获取集合失败: %w", err)
	}

	// 构建查询选项
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	opts.SetSort(bson.D{{Key: "_created_at", Value: -1}}) // 按创建时间倒序

	// 执行查询
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}
	defer cursor.Close(ctx)

	// 解析结果
	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("解析查询结果失败: %w", err)
	}

	return results, nil
}

// Update 更新数据
func (m *MongoDBStorage) Update(ctx context.Context, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	coll, err := m.pool.GetCollection()
	if err != nil {
		return 0, fmt.Errorf("获取集合失败: %w", err)
	}

	// 添加更新时间
	update["_updated_at"] = time.Now()

	// 构建更新文档
	updateDoc := bson.M{"$set": update}

	// 执行更新
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	result, err := coll.UpdateMany(ctx, filter, updateDoc)
	if err != nil {
		return 0, fmt.Errorf("更新数据失败: %w", err)
	}

	return result.ModifiedCount, nil
}

// Delete 删除数据
func (m *MongoDBStorage) Delete(ctx context.Context, filter map[string]interface{}) (int64, error) {
	coll, err := m.pool.GetCollection()
	if err != nil {
		return 0, fmt.Errorf("获取集合失败: %w", err)
	}

	// 执行删除
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("删除数据失败: %w", err)
	}

	return result.DeletedCount, nil
}

// Count 统计数据
func (m *MongoDBStorage) Count(ctx context.Context, filter map[string]interface{}) (int64, error) {
	coll, err := m.pool.GetCollection()
	if err != nil {
		return 0, fmt.Errorf("获取集合失败: %w", err)
	}

	// 执行统计
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("统计数据失败: %w", err)
	}

	return count, nil
}

// CreateIndex 创建索引
func (m *MongoDBStorage) CreateIndex(ctx context.Context, keys map[string]int, unique bool) error {
	coll, err := m.pool.GetCollection()
	if err != nil {
		return fmt.Errorf("获取集合失败: %w", err)
	}

	// 构建索引模型
	indexKeys := bson.D{}
	for key, order := range keys {
		indexKeys = append(indexKeys, bson.E{Key: key, Value: order})
	}

	indexModel := mongo.IndexModel{
		Keys:    indexKeys,
		Options: options.Index().SetUnique(unique),
	}

	// 创建索引
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	indexName, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	log.Printf("成功创建索引: %s", indexName)
	return nil
}

// Close 关闭存储
func (m *MongoDBStorage) Close() error {
	return m.pool.Close()
}
