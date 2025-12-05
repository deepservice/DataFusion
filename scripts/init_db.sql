-- DataFusion 数据库初始化脚本

-- 创建数据库
CREATE DATABASE datafusion_control;
CREATE DATABASE datafusion_data;

-- 连接到 datafusion_control 数据库
\c datafusion_control;

-- 创建任务配置表
CREATE TABLE IF NOT EXISTS collection_tasks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- web-rpa, api, database
    status VARCHAR(50) DEFAULT 'enabled',  -- enabled, disabled
    cron VARCHAR(100),
    next_run_time TIMESTAMP,
    replicas INT DEFAULT 1,
    execution_timeout INT DEFAULT 3600,
    max_retries INT DEFAULT 3,
    config TEXT NOT NULL,  -- JSON 配置
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 创建任务执行记录表
CREATE TABLE IF NOT EXISTS task_executions (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT REFERENCES collection_tasks(id),
    worker_pod VARCHAR(255),
    status VARCHAR(50),  -- running, success, failed
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    records_collected INT DEFAULT 0,
    error_message TEXT,
    retry_count INT DEFAULT 0
);

-- 创建索引
CREATE INDEX idx_next_run_time ON collection_tasks(next_run_time);
CREATE INDEX idx_task_executions_status ON task_executions(task_id, status);
CREATE INDEX idx_task_executions_start_time ON task_executions(start_time DESC);

-- 连接到 datafusion_data 数据库
\c datafusion_data;

-- 创建示例数据表（用于存储采集的数据）
CREATE TABLE IF NOT EXISTS articles (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500),
    content TEXT,
    author VARCHAR(100),
    publish_time TIMESTAMP,
    source_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    product_name VARCHAR(255),
    price DECIMAL(10, 2),
    stock INT,
    category VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 输出提示
SELECT 'Database initialization completed!' AS message;
