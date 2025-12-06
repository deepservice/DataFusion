-- 示例：MongoDB 存储任务配置

-- 1. API 采集 + MongoDB 存储
INSERT INTO collection_tasks (
    name,
    type,
    status,
    cron,
    next_run_time,
    replicas,
    execution_timeout,
    max_retries,
    config,
    created_at,
    updated_at
) VALUES (
    'api-to-mongodb',
    'api',
    'enabled',
    '0 */1 * * *',  -- 每小时执行一次
    NOW() + INTERVAL '1 minute',
    1,
    300,
    3,
    '{
        "data_source": {
            "type": "api",
            "url": "https://api.example.com/data",
            "method": "GET",
            "api_config": {
                "timeout": 30
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "trim_name",
                    "field": "name",
                    "type": "trim"
                }
            ],
            "deduplication": {
                "strategy": "content_hash",
                "cache_size": 10000,
                "enable_logging": true
            }
        },
        "storage": {
            "target": "mongodb",
            "database": "datafusion",
            "collection": "api_data",
            "config": {
                "uri": "mongodb://localhost:27017",
                "timeout": 30,
                "max_pool_size": 100
            }
        }
    }',
    NOW(),
    NOW()
);

-- 2. 数据库采集 + 去重 + MongoDB 存储
INSERT INTO collection_tasks (
    name,
    type,
    status,
    cron,
    next_run_time,
    replicas,
    execution_timeout,
    max_retries,
    config,
    created_at,
    updated_at
) VALUES (
    'mysql-to-mongodb-dedup',
    'database',
    'enabled',
    '0 */6 * * *',  -- 每 6 小时执行一次
    NOW() + INTERVAL '1 minute',
    1,
    600,
    3,
    '{
        "data_source": {
            "type": "database",
            "db_config": {
                "host": "mysql-server",
                "port": 3306,
                "user": "datafusion",
                "password": "password123",
                "database": "app_db",
                "query": "SELECT id, name, email, created_at FROM users WHERE created_at > DATE_SUB(NOW(), INTERVAL 6 HOUR)"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "trim_name",
                    "field": "name",
                    "type": "trim"
                },
                {
                    "name": "validate_email",
                    "field": "email",
                    "type": "email_validate"
                }
            ],
            "deduplication": {
                "strategy": "field_based",
                "fields": ["id", "email"],
                "cache_size": 50000,
                "enable_logging": true
            }
        },
        "storage": {
            "target": "mongodb",
            "database": "datafusion",
            "collection": "users",
            "config": {
                "uri": "mongodb://localhost:27017",
                "timeout": 30
            }
        }
    }',
    NOW(),
    NOW()
);

-- 3. 时间窗口去重示例
INSERT INTO collection_tasks (
    name,
    type,
    status,
    cron,
    next_run_time,
    replicas,
    execution_timeout,
    max_retries,
    config,
    created_at,
    updated_at
) VALUES (
    'web-scraping-time-window-dedup',
    'web-rpa',
    'enabled',
    '0 * * * *',  -- 每小时执行一次
    NOW() + INTERVAL '1 minute',
    1,
    900,
    3,
    '{
        "data_source": {
            "type": "web-rpa",
            "url": "https://example.com/news",
            "selectors": {
                "title": ".news-title",
                "content": ".news-content",
                "date": ".news-date"
            },
            "rpa_config": {
                "browser_type": "chromium",
                "headless": true,
                "timeout": 60
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "remove_html",
                    "field": "content",
                    "type": "remove_html"
                },
                {
                    "name": "normalize_whitespace",
                    "field": "content",
                    "type": "normalize_whitespace"
                }
            ],
            "deduplication": {
                "strategy": "time_window",
                "time_window": "24h",
                "cache_size": 10000,
                "enable_logging": true
            }
        },
        "storage": {
            "target": "mongodb",
            "database": "datafusion",
            "collection": "news",
            "config": {
                "uri": "mongodb://localhost:27017"
            }
        }
    }',
    NOW(),
    NOW()
);

-- 查询所有 MongoDB 存储任务
SELECT 
    id,
    name,
    type,
    status,
    cron,
    next_run_time
FROM collection_tasks
WHERE config::jsonb->'storage'->>'target' = 'mongodb'
ORDER BY created_at DESC;

-- 查询带去重配置的任务
SELECT 
    id,
    name,
    type,
    config::jsonb->'processor'->'deduplication'->>'strategy' as dedup_strategy
FROM collection_tasks
WHERE config::jsonb->'processor'->'deduplication' IS NOT NULL
ORDER BY created_at DESC;
