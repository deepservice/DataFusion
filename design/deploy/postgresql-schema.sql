-- DataFusion Deployment常驻Worker架构 - PostgreSQL数据库Schema
-- 创建日期: 2025-11-26
-- 用途: 存储任务配置、执行记录和分布式锁信息

-- ========================================
-- 创建数据库（如果不存在）
-- ========================================
-- CREATE DATABASE datafusion_control;
-- \c datafusion_control;

-- ========================================
-- 1. 任务配置表（collection_tasks）
-- ========================================
-- 用途：存储CollectionTask CR的配置信息
-- 说明：Controller将CR配置同步到此表，Worker从此表读取任务配置

CREATE TABLE IF NOT EXISTS collection_tasks (
    -- 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 任务标识
    name VARCHAR(200) NOT NULL,

    -- Kubernetes CR关联信息
    cr_namespace VARCHAR(63) NOT NULL,
    cr_name VARCHAR(253) NOT NULL,
    cr_uid VARCHAR(36) UNIQUE NOT NULL,  -- Kubernetes资源的UID，保证唯一性
    cr_generation BIGINT NOT NULL DEFAULT 0,  -- 检测配置变更，对应metadata.generation

    -- Collector配置
    collector_type VARCHAR(50) NOT NULL,  -- api, web-rpa, database
    collector_config JSONB NOT NULL,  -- 采集器配置（JSON格式）

    -- 调度配置
    schedule_cron VARCHAR(100),  -- cron表达式，如: "*/5 * * * *" 表示每5分钟
    schedule_timezone VARCHAR(50) DEFAULT 'UTC',  -- 时区
    enabled BOOLEAN NOT NULL DEFAULT true,  -- 是否启用

    -- 数据处理配置
    parsing_rules JSONB,  -- 解析规则（JSON格式）
    cleaning_rules JSONB,  -- 清洗规则（JSON格式）

    -- 存储配置
    storage_type VARCHAR(50) NOT NULL,  -- postgresql, elasticsearch, s3等
    storage_config JSONB NOT NULL,  -- 存储配置（JSON格式）

    -- 运行时状态
    next_run_time TIMESTAMP WITH TIME ZONE,  -- 下次执行时间，Worker轮询此字段
    last_success_time TIMESTAMP WITH TIME ZONE,  -- 最后成功执行时间
    last_failure_time TIMESTAMP WITH TIME ZONE,  -- 最后失败时间

    -- 统计信息
    total_runs BIGINT NOT NULL DEFAULT 0,  -- 总执行次数
    successful_runs BIGINT NOT NULL DEFAULT 0,  -- 成功次数
    failed_runs BIGINT NOT NULL DEFAULT 0,  -- 失败次数

    -- 元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- 索引
    CONSTRAINT collection_tasks_cr_unique UNIQUE (cr_namespace, cr_name)
);

-- 创建索引：用于Worker查询到期任务
CREATE INDEX IF NOT EXISTS idx_collection_tasks_enabled_next_run
    ON collection_tasks (enabled, next_run_time)
    WHERE enabled = true;

-- 创建索引：用于按采集器类型查询
CREATE INDEX IF NOT EXISTS idx_collection_tasks_collector_type
    ON collection_tasks (collector_type, enabled)
    WHERE enabled = true;

-- 创建索引：用于按CR UID查询
CREATE INDEX IF NOT EXISTS idx_collection_tasks_cr_uid
    ON collection_tasks (cr_uid);

COMMENT ON TABLE collection_tasks IS '任务配置表，存储CollectionTask CR的配置信息';
COMMENT ON COLUMN collection_tasks.id IS '任务ID（UUID）';
COMMENT ON COLUMN collection_tasks.cr_uid IS 'Kubernetes CR的UID，保证唯一性';
COMMENT ON COLUMN collection_tasks.cr_generation IS 'CR的generation字段，用于检测配置变更';
COMMENT ON COLUMN collection_tasks.next_run_time IS '下次执行时间，Worker轮询此字段获取到期任务';

-- ========================================
-- 2. 任务执行记录表（task_executions）
-- ========================================
-- 用途：记录每次任务执行的详细信息
-- 说明：Worker执行任务前创建记录，执行完成后更新状态

CREATE TABLE IF NOT EXISTS task_executions (
    -- 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 关联任务
    task_id UUID NOT NULL REFERENCES collection_tasks(id) ON DELETE CASCADE,

    -- 执行状态
    status VARCHAR(20) NOT NULL,  -- running, success, failed
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_seconds NUMERIC(10, 3),  -- 执行耗时（秒）

    -- 执行者信息
    worker_pod VARCHAR(255) NOT NULL,  -- 执行该任务的Pod名称
    worker_node VARCHAR(255),  -- 执行节点名称

    -- 执行结果
    records_fetched INTEGER DEFAULT 0,  -- 采集记录数
    records_stored INTEGER DEFAULT 0,  -- 存储记录数
    error_message TEXT,  -- 错误信息
    error_code VARCHAR(50),  -- 错误代码

    -- 重试信息
    retry_count INTEGER NOT NULL DEFAULT 0,  -- 重试次数
    is_retry BOOLEAN NOT NULL DEFAULT false,  -- 是否是重试执行
    original_execution_id UUID,  -- 原始执行ID（如果是重试）

    -- 元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引：用于查询任务的执行历史
CREATE INDEX IF NOT EXISTS idx_task_executions_task_id_start_time
    ON task_executions (task_id, start_time DESC);

-- 创建索引：用于查询特定状态的执行记录
CREATE INDEX IF NOT EXISTS idx_task_executions_status
    ON task_executions (status, start_time DESC);

-- 创建索引：用于查询特定Worker的执行记录
CREATE INDEX IF NOT EXISTS idx_task_executions_worker_pod
    ON task_executions (worker_pod, start_time DESC);

-- 创建索引：用于按时间范围查询
CREATE INDEX IF NOT EXISTS idx_task_executions_start_time
    ON task_executions (start_time DESC);

COMMENT ON TABLE task_executions IS '任务执行记录表，记录每次任务执行的详细信息';
COMMENT ON COLUMN task_executions.status IS '执行状态: running-运行中, success-成功, failed-失败';
COMMENT ON COLUMN task_executions.worker_pod IS '执行该任务的Pod名称，用于故障排查';
COMMENT ON COLUMN task_executions.duration_seconds IS '执行耗时（秒），用于性能分析';

-- ========================================
-- 3. 分布式锁表（task_locks）
-- ========================================
-- 用途：多副本Worker场景下保证任务不重复执行
-- 说明：使用PostgreSQL的FOR UPDATE SKIP LOCKED实现分布式锁

CREATE TABLE IF NOT EXISTS task_locks (
    -- 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 关联任务（每个任务只有一把锁）
    task_id UUID UNIQUE NOT NULL REFERENCES collection_tasks(id) ON DELETE CASCADE,

    -- 锁信息
    locked_at TIMESTAMP WITH TIME ZONE,  -- 锁定时间
    locked_by VARCHAR(255),  -- 持有锁的Pod名称
    lock_version BIGINT NOT NULL DEFAULT 0,  -- 锁版本号，每次加锁递增

    -- 元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引：用于查询被锁定的任务
CREATE INDEX IF NOT EXISTS idx_task_locks_locked_by
    ON task_locks (locked_by, locked_at)
    WHERE locked_by IS NOT NULL;

COMMENT ON TABLE task_locks IS '分布式锁表，保证多副本Worker场景下任务不重复执行';
COMMENT ON COLUMN task_locks.task_id IS '任务ID，每个任务对应唯一一把锁';
COMMENT ON COLUMN task_locks.locked_by IS '持有锁的Pod名称，用于排查锁泄漏';
COMMENT ON COLUMN task_locks.lock_version IS '锁版本号，每次加锁递增，用于检测锁冲突';

-- ========================================
-- 4. 采集数据表（collected_data）- 可选
-- ========================================
-- 用途：直接存储采集的数据（可选，也可存储到外部系统）
-- 说明：使用data_hash字段实现去重

CREATE TABLE IF NOT EXISTS collected_data (
    -- 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 关联任务
    task_id UUID NOT NULL REFERENCES collection_tasks(id) ON DELETE CASCADE,

    -- 数据内容
    data JSONB NOT NULL,  -- 采集的数据（JSON格式）
    data_hash VARCHAR(64) NOT NULL,  -- 数据哈希值，用于去重（SHA256）

    -- 采集信息
    collected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    execution_id UUID REFERENCES task_executions(id),  -- 关联执行记录

    -- 元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建唯一索引：用于去重（同一任务的相同数据只存储一次）
CREATE UNIQUE INDEX IF NOT EXISTS uk_collected_data_task_data_hash
    ON collected_data (task_id, data_hash);

-- 创建索引：用于按任务查询数据
CREATE INDEX IF NOT EXISTS idx_collected_data_task_id_collected_at
    ON collected_data (task_id, collected_at DESC);

-- 创建索引：用于按时间范围查询
CREATE INDEX IF NOT EXISTS idx_collected_data_collected_at
    ON collected_data (collected_at DESC);

-- 创建GIN索引：用于JSONB数据查询
CREATE INDEX IF NOT EXISTS idx_collected_data_data_gin
    ON collected_data USING GIN (data);

COMMENT ON TABLE collected_data IS '采集数据表（可选），直接存储采集的数据';
COMMENT ON COLUMN collected_data.data_hash IS '数据哈希值（SHA256），用于去重';
COMMENT ON COLUMN collected_data.data IS 'JSONB格式的采集数据，支持复杂查询';

-- ========================================
-- 5. 触发器：自动更新updated_at
-- ========================================

-- 创建更新时间戳函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为collection_tasks表创建触发器
CREATE TRIGGER update_collection_tasks_updated_at
    BEFORE UPDATE ON collection_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为task_locks表创建触发器
CREATE TRIGGER update_task_locks_updated_at
    BEFORE UPDATE ON task_locks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- 6. 视图：任务执行统计
-- ========================================

-- 创建任务执行统计视图
CREATE OR REPLACE VIEW v_task_execution_stats AS
SELECT
    t.id AS task_id,
    t.name AS task_name,
    t.collector_type,
    t.enabled,
    t.total_runs,
    t.successful_runs,
    t.failed_runs,
    CASE
        WHEN t.total_runs > 0
        THEN ROUND((t.successful_runs::NUMERIC / t.total_runs::NUMERIC) * 100, 2)
        ELSE 0
    END AS success_rate,
    t.last_success_time,
    t.last_failure_time,
    t.next_run_time,
    (
        SELECT AVG(duration_seconds)
        FROM task_executions
        WHERE task_id = t.id AND status = 'success'
    ) AS avg_duration_seconds,
    (
        SELECT COUNT(*)
        FROM task_executions
        WHERE task_id = t.id
          AND status = 'running'
          AND start_time > NOW() - INTERVAL '1 hour'
    ) AS running_count_last_hour
FROM collection_tasks t;

COMMENT ON VIEW v_task_execution_stats IS '任务执行统计视图，提供任务成功率、平均耗时等统计信息';

-- ========================================
-- 7. 示例数据（用于测试）
-- ========================================

-- 插入示例任务配置
-- INSERT INTO collection_tasks (
--     name, cr_namespace, cr_name, cr_uid, cr_generation,
--     collector_type, collector_config,
--     schedule_cron, storage_type, storage_config,
--     next_run_time
-- ) VALUES (
--     'product-scraper',
--     'datafusion',
--     'product-scraper',
--     '550e8400-e29b-41d4-a716-446655440000',
--     1,
--     'web-rpa',
--     '{"url": "https://example.com/products", "wait_selector": ".product-item"}',
--     '*/5 * * * *',
--     'postgresql',
--     '{"table": "products", "batch_size": 100}',
--     NOW()
-- );

-- 为示例任务创建锁记录
-- INSERT INTO task_locks (task_id)
-- SELECT id FROM collection_tasks WHERE name = 'product-scraper';

-- ========================================
-- 8. 权限管理（生产环境建议）
-- ========================================

-- 创建专用用户（如果不存在）
-- CREATE USER datafusion_worker WITH PASSWORD 'strong_password_here';

-- 授予必要权限
-- GRANT CONNECT ON DATABASE datafusion_control TO datafusion_worker;
-- GRANT SELECT, INSERT, UPDATE ON collection_tasks TO datafusion_worker;
-- GRANT SELECT, INSERT, UPDATE ON task_executions TO datafusion_worker;
-- GRANT SELECT, UPDATE ON task_locks TO datafusion_worker;
-- GRANT SELECT, INSERT ON collected_data TO datafusion_worker;

-- ========================================
-- 9. 维护SQL（日常运维）
-- ========================================

-- 清理30天前的执行记录
-- DELETE FROM task_executions WHERE start_time < NOW() - INTERVAL '30 days';

-- 清理90天前的采集数据
-- DELETE FROM collected_data WHERE collected_at < NOW() - INTERVAL '90 days';

-- 检查死锁（锁定超过1小时的任务）
-- SELECT
--     l.task_id,
--     t.name,
--     l.locked_by,
--     l.locked_at,
--     NOW() - l.locked_at AS lock_duration
-- FROM task_locks l
-- JOIN collection_tasks t ON l.task_id = t.id
-- WHERE l.locked_at < NOW() - INTERVAL '1 hour';

-- 释放死锁
-- UPDATE task_locks SET locked_at = NULL, locked_by = NULL
-- WHERE locked_at < NOW() - INTERVAL '1 hour';

-- ========================================
-- Schema创建完成
-- ========================================

-- 验证表创建
SELECT
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) AS column_count,
    (SELECT COUNT(*) FROM pg_indexes WHERE tablename = t.table_name) AS index_count
FROM information_schema.tables t
WHERE table_schema = 'public'
  AND table_name IN ('collection_tasks', 'task_executions', 'task_locks', 'collected_data')
ORDER BY table_name;
