-- DataFusion Control Plane Database Schema
-- 控制面数据库初始化脚本

-- 创建数据库（如果不存在）
-- CREATE DATABASE datafusion_control;

-- 连接到数据库
-- \c datafusion_control;

-- 1. 数据源表
CREATE TABLE IF NOT EXISTS data_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,  -- web, api, database
    config JSONB NOT NULL,       -- 数据源配置（JSON格式）
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',  -- active, inactive
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_data_sources_type ON data_sources(type);
CREATE INDEX idx_data_sources_status ON data_sources(status);

-- 2. 清洗规则表
CREATE TABLE IF NOT EXISTS cleaning_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL,  -- trim, remove_html, regex, etc.
    config JSONB NOT NULL,            -- 规则配置（JSON格式）
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_cleaning_rules_type ON cleaning_rules(rule_type);

-- 3. 采集任务表
CREATE TABLE IF NOT EXISTS collection_tasks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    type VARCHAR(50) NOT NULL,        -- web-rpa, api, database
    data_source_id BIGINT REFERENCES data_sources(id) ON DELETE CASCADE,
    cron VARCHAR(100),                -- Cron表达式
    next_run_time TIMESTAMP,          -- 下次执行时间
    status VARCHAR(50) DEFAULT 'enabled',  -- enabled, disabled
    replicas INT DEFAULT 1,           -- 并发执行数
    execution_timeout INT DEFAULT 3600,  -- 执行超时（秒）
    max_retries INT DEFAULT 3,        -- 最大重试次数
    config JSONB,                     -- 任务配置（JSON格式）
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_collection_tasks_status ON collection_tasks(status);
CREATE INDEX idx_collection_tasks_next_run_time ON collection_tasks(next_run_time);
CREATE INDEX idx_collection_tasks_data_source ON collection_tasks(data_source_id);

-- 4. 任务执行记录表
CREATE TABLE IF NOT EXISTS task_executions (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT REFERENCES collection_tasks(id) ON DELETE CASCADE,
    worker_pod VARCHAR(255),          -- 执行的Worker Pod名称
    status VARCHAR(50) NOT NULL,      -- running, success, failed
    start_time TIMESTAMP DEFAULT NOW(),
    end_time TIMESTAMP,
    records_collected INT DEFAULT 0,  -- 采集的记录数
    error_message TEXT,               -- 错误信息
    retry_count INT DEFAULT 0,        -- 重试次数
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_task_executions_task_id ON task_executions(task_id);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_start_time ON task_executions(start_time DESC);

-- 5. 任务-清洗规则关联表
CREATE TABLE IF NOT EXISTS task_cleaning_rules (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT REFERENCES collection_tasks(id) ON DELETE CASCADE,
    cleaning_rule_id BIGINT REFERENCES cleaning_rules(id) ON DELETE CASCADE,
    field_name VARCHAR(255),          -- 应用规则的字段名
    order_index INT DEFAULT 0,        -- 规则执行顺序
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(task_id, cleaning_rule_id, field_name)
);

CREATE INDEX idx_task_cleaning_rules_task ON task_cleaning_rules(task_id);
CREATE INDEX idx_task_cleaning_rules_rule ON task_cleaning_rules(cleaning_rule_id);

-- 6. 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255),       -- 本地用户密码哈希
    email VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',  -- admin, user
    auth_type VARCHAR(50) DEFAULT 'local',  -- local, oauth
    oauth_provider VARCHAR(50),       -- github, google, etc.
    oauth_id VARCHAR(255),            -- OAuth用户ID
    status VARCHAR(50) DEFAULT 'active',  -- active, inactive
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id);

-- 7. API密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),                -- API Key名称
    description TEXT,
    permissions JSONB,                -- 权限配置
    expires_at TIMESTAMP,             -- 过期时间
    last_used_at TIMESTAMP,           -- 最后使用时间
    status VARCHAR(50) DEFAULT 'active',  -- active, revoked
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- 8. 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 插入默认管理员用户（密码: admin123）
INSERT INTO users (username, password_hash, email, role, auth_type, status)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin@datafusion.io', 'admin', 'local', 'active')
ON CONFLICT (username) DO NOTHING;

-- 插入示例数据源
INSERT INTO data_sources (name, type, config, description, status)
VALUES 
    ('示例网页数据源', 'web', '{"url": "https://example.com", "method": "GET"}', '示例网页数据源', 'active'),
    ('示例API数据源', 'api', '{"url": "https://api.example.com/data", "method": "GET"}', '示例API数据源', 'active'),
    ('示例数据库数据源', 'database', '{"type": "postgresql", "host": "localhost", "port": 5432}', '示例数据库数据源', 'active')
ON CONFLICT (name) DO NOTHING;

-- 插入示例清洗规则
INSERT INTO cleaning_rules (name, description, rule_type, config)
VALUES 
    ('去除空白', '去除字符串首尾空白', 'trim', '{}'),
    ('移除HTML标签', '移除HTML标签', 'remove_html', '{}'),
    ('日期格式化', '统一日期格式', 'date_format', '{"format": "2006-01-02 15:04:05"}'),
    ('数字格式化', '格式化数字', 'number_format', '{"decimals": 2}')
ON CONFLICT (name) DO NOTHING;

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加更新时间触发器
CREATE TRIGGER update_data_sources_updated_at BEFORE UPDATE ON data_sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cleaning_rules_updated_at BEFORE UPDATE ON cleaning_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_collection_tasks_updated_at BEFORE UPDATE ON collection_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 完成
SELECT 'DataFusion Control Database initialized successfully!' as message;
