-- VSCode PostgreSQL 插件常用查询
-- 切换到 datafusion_control 数据库后执行

-- 1. 查看所有采集任务
SELECT id, name, type, status, cron, next_run_time, replicas
FROM collection_tasks
ORDER BY id;

-- 2. 查看任务执行历史（最近10条）
SELECT
    te.id,
    ct.name AS task_name,
    te.status,
    te.records_collected,
    te.start_time,
    te.end_time,
    te.error_message,
    te.worker_pod
FROM task_executions te
LEFT JOIN collection_tasks ct ON te.task_id = ct.id
ORDER BY te.start_time DESC
LIMIT 10;

-- 3. 查看任务执行统计
SELECT
    ct.name,
    COUNT(*) AS total_executions,
    SUM(CASE WHEN te.status = 'success' THEN 1 ELSE 0 END) AS success_count,
    SUM(CASE WHEN te.status = 'failed' THEN 1 ELSE 0 END) AS failed_count,
    SUM(te.records_collected) AS total_records
FROM task_executions te
LEFT JOIN collection_tasks ct ON te.task_id = ct.id
GROUP BY ct.name;

-- 4. 查看正在运行的任务
SELECT
    ct.name,
    te.worker_pod,
    te.start_time,
    NOW() - te.start_time AS running_duration
FROM task_executions te
JOIN collection_tasks ct ON te.task_id = ct.id
WHERE te.status = 'running'
ORDER BY te.start_time;

-- ============================================
-- 切换到 datafusion_data 数据库后执行
-- ============================================

-- 5. 查看采集到的测试数据
SELECT * FROM test_posts
ORDER BY created_at DESC
LIMIT 20;

-- 6. 统计采集数据量
SELECT
    COUNT(*) AS total_records,
    MIN(created_at) AS first_record,
    MAX(created_at) AS last_record
FROM test_posts;

-- 7. 按用户统计
SELECT
    user_id,
    COUNT(*) AS post_count
FROM test_posts
GROUP BY user_id
ORDER BY post_count DESC;
