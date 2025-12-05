-- 插入测试任务

\c datafusion_control;

-- 示例 1: RPA 采集任务（采集新闻文章）
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '测试-新闻文章采集',
    'web-rpa',
    'enabled',
    '*/5 * * * *',  -- 每5分钟执行一次
    NOW(),
    1,
    '{
        "data_source": {
            "type": "web-rpa",
            "url": "https://example.com/news",
            "selectors": {
                "_list": ".article-item",
                "title": ".article-title",
                "content": ".article-content",
                "author": ".author-name",
                "publish_time": ".publish-time"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "清理标题",
                    "field": "title",
                    "type": "trim"
                },
                {
                    "name": "移除HTML标签",
                    "field": "content",
                    "type": "remove_html"
                }
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "postgresql",
            "database": "datafusion_data",
            "table": "articles",
            "mapping": {
                "title": "title",
                "content": "content",
                "author": "author",
                "publish_time": "publish_time"
            }
        }
    }'
);

-- 示例 2: API 采集任务（采集产品数据）
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '测试-产品数据API采集',
    'api',
    'enabled',
    '0 */2 * * *',  -- 每2小时执行一次
    NOW(),
    1,
    '{
        "data_source": {
            "type": "api",
            "url": "https://api.example.com/products",
            "method": "GET",
            "headers": {
                "Authorization": "Bearer YOUR_API_KEY"
            },
            "selectors": {
                "_data_path": "data.items",
                "product_name": "name",
                "price": "price",
                "stock": "inventory",
                "category": "category",
                "description": "desc"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "清理产品名称",
                    "field": "product_name",
                    "type": "trim"
                }
            ],
            "transform_rules": []
        },
        "storage": {
            "target": "postgresql",
            "database": "datafusion_data",
            "table": "products",
            "mapping": {
                "product_name": "product_name",
                "price": "price",
                "stock": "stock",
                "category": "category",
                "description": "description"
            }
        }
    }'
);

-- 示例 3: 简单的文件存储任务
INSERT INTO collection_tasks (name, type, status, cron, next_run_time, replicas, config)
VALUES (
    '测试-文件存储',
    'web-rpa',
    'enabled',
    '0 0 * * *',  -- 每天凌晨执行
    NOW() + INTERVAL '1 day',
    1,
    '{
        "data_source": {
            "type": "web-rpa",
            "url": "https://example.com/data",
            "selectors": {
                "title": "h1",
                "content": ".content"
            }
        },
        "processor": {
            "cleaning_rules": [],
            "transform_rules": []
        },
        "storage": {
            "target": "file",
            "database": "exports",
            "table": "daily_data",
            "mapping": {}
        }
    }'
);

SELECT 'Test tasks inserted successfully!' AS message;
SELECT id, name, type, status, next_run_time FROM collection_tasks;
