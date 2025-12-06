package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Pool MongoDB 连接池
type Pool struct {
	client *mongo.Client
	config *Config
	mu     sync.RWMutex
	closed bool
}

// NewPool 创建连接池
func NewPool(config *Config) (*Pool, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建客户端选项
	clientOpts := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetMaxConnIdleTime(config.MaxConnIdleTime).
		SetConnectTimeout(config.Timeout)

	// 创建客户端
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("连接 MongoDB 失败: %w", err)
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB 连接测试失败: %w", err)
	}

	return &Pool{
		client: client,
		config: config,
		closed: false,
	}, nil
}

// GetClient 获取客户端
func (p *Pool) GetClient() (*mongo.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("连接池已关闭")
	}

	return p.client, nil
}

// GetDatabase 获取数据库
func (p *Pool) GetDatabase() (*mongo.Database, error) {
	client, err := p.GetClient()
	if err != nil {
		return nil, err
	}

	return client.Database(p.config.Database), nil
}

// GetCollection 获取集合
func (p *Pool) GetCollection() (*mongo.Collection, error) {
	db, err := p.GetDatabase()
	if err != nil {
		return nil, err
	}

	return db.Collection(p.config.Collection), nil
}

// Close 关闭连接池
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("关闭 MongoDB 连接失败: %w", err)
	}

	p.closed = true
	return nil
}

// IsClosed 检查是否已关闭
func (p *Pool) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}
