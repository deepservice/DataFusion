-- 示例：数据库采集任务配置

-- 1. MySQL 数据采集任务
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
    'mysql-users-collection',
    'database',
    'enabled',
    '0 */6 * * *',  -- 每 6 小时执行一次
    NOW() + INTERVAL '1 minute',
    1,
    300,  -- 5 分钟超时
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
                "query": "SELECT id, username, email, created_at FROM users WHERE created_at > DATE_SUB(NOW(), INTERVAL 6 HOUR)"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "trim_username",
                    "field": "username",
                    "type": "trim"
                },
                {
                    "name": "validate_email",
                    "field": "email",
                    "type": "email_validate"
                },
                {
                    "name": "format_date",
                    "field": "created_at",
                    "type": "date_format",
                    "pattern": "2006-01-02 15:04:05"
                }
            ]
        },
        "storage": {
            "target": "postgresql",
            "database": "datafusion",
            "table": "collected_users",
            "mapping": {
                "id": "source_user_id",
                "username": "username",
                "email": "email",
                "created_at": "registration_time"
            }
        }
    }',
    NOW(),
    NOW()
);

-- 2. PostgreSQL 数据采集任务
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
    'postgres-orders-collection',
    'database',
    'enabled',
    '0 * * * *',  -- 每小时执行一次
    NOW() + INTERVAL '1 minute',
    1,
    600,  -- 10 分钟超时
    3,
    '{
        "data_source": {
            "type": "database",
            "db_config": {
                "host": "postgres-server",
                "port": 5432,
                "user": "datafusion",
                "password": "password123",
                "database": "ecommerce",
                "query": "SELECT order_id, customer_id, total_amount, order_date, status FROM orders WHERE order_date > NOW() - INTERVAL ''1 hour''"
            }
        },
        "processor": {
            "cleaning_rules": [
                {
                    "name": "format_amount",
                    "field": "total_amount",
                    "type": "number_format"
                },
                {
                    "name": "format_date",
                    "field": "order_date",
                    "type": "date_format",
                    "pattern": "2006-01-02"
                },
                {
                    "name": "normalize_status",
                    "field": "status",
                    "type": "trim"
                }
            ]
        },
        "storage": {
            "target": "postgresql",
            "database": "datafusion",
            "table": "collected_orders",
            "mapping": {
                "order_id": "source_order_id",
                "customer_id": "customer_id",
                "total_amount": "amount",
                "order_date": "order_date",
                "status": "order_status"
            }
        }
    }',
    NOW(),
    NOW()
);

-- 3. 带增强清洗规则的数据采集任务
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
    'mysql-products-collection',
    'database',
    'enabled',
    '0 0 * * *',  -- 每天凌晨执行
    NOW() + INTERVAL '1 minute',
    1,
    900,  -- 15 分钟超时
    3,
    '{
        "data_source": {
            "type": "database",
            "db_config": {
                "host": "mysql-server",
                "port": 3306,
                "user": "datafusion",
                "password": "password123",
                "database": "shop_db",
                "query": "SELECT product_id, name, description, price, phone, website, updated_at FROM products WHERE updated_at > DATE_SUB(NOW(), INTERVAL 1 DAY)"
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
                    "name": "remove_html_description",
                    "field": "description",
                    "type": "remove_html"
                },
                {
                    "name": "normalize_whitespace",
                    "field": "description",
                    "type": "normalize_whitespace"
                },
                {
                    "name": "format_price",
                    "field": "price",
                    "type": "number_format"
                },
                {
                    "name": "format_phone",
                    "field": "phone",
                    "type": "phone_format"
                },
                {
                    "name": "normalize_url",
                    "field": "website",
                    "type": "url_normalize"
                },
                {
                    "name": "format_date",
                    "field": "updated_at",
                    "type": "date_format",
                    "pattern": "2006-01-02 15:04:05"
                }
            ]
        },
        "storage": {
            "target": "postgresql",
            "database": "datafusion",
            "table": "collected_products",
            "mapping": {
                "product_id": "source_product_id",
                "name": "product_name",
                "description": "product_description",
                "price": "price",
                "phone": "contact_phone",
                "website": "website_url",
                "updated_at": "last_updated"
            }
        }
    }',
    NOW(),
    NOW()
);

-- 查询所有数据库采集任务
SELECT 
    id,
    name,
    type,
    status,
    cron,
    next_run_time,
    replicas,
    execution_timeout,
    max_retries
FROM collection_tasks
WHERE type = 'database'
ORDER BY created_at DESC;
