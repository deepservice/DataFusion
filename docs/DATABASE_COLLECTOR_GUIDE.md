# 数据库采集器使用指南

## 快速开始

### 1. 基本配置

```json
{
  "data_source": {
    "type": "database",
    "db_config": {
      "host": "数据库主机",
      "port": 端口号,
      "user": "用户名",
      "password": "密码",
      "database": "数据库名",
      "query": "SQL 查询语句"
    }
  }
}
```

### 2. MySQL 示例

```json
{
  "data_source": {
    "type": "database",
    "db_config": {
      "host": "mysql-server",
      "port": 3306,
      "user": "datafusion",
      "password": "password123",
      "database": "app_db",
      "query": "SELECT * FROM users WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)"
    }
  }
}
```

### 3. PostgreSQL 示例

```json
{
  "data_source": {
    "type": "database",
    "db_config": {
      "host": "postgres-server",
      "port": 5432,
      "user": "datafusion",
      "password": "password123",
      "database": "app_db",
      "query": "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '1 hour'"
    }
  }
}
```

## 增强清洗规则

### 基础清洗

#### trim - 去除空白
```json
{
  "name": "trim_field",
  "field": "name",
  "type": "trim"
}
```

#### remove_html - 移除 HTML
```json
{
  "name": "clean_description",
  "field": "description",
  "type": "remove_html"
}
```

#### normalize_whitespace - 规范化空白
```json
{
  "name": "normalize_text",
  "field": "content",
  "type": "normalize_whitespace"
}
```

### 数据验证

#### email_validate - 邮箱验证
```json
{
  "name": "validate_email",
  "field": "email",
  "type": "email_validate"
}
```
- 验证邮箱格式
- 转换为小写
- 去除空白

#### phone_format - 电话格式化
```json
{
  "name": "format_phone",
  "field": "phone",
  "type": "phone_format"
}
```
- 输入: `13812345678`
- 输出: `138-1234-5678`

### 格式化

#### date_format - 日期格式化
```json
{
  "name": "format_date",
  "field": "created_at",
  "type": "date_format",
  "pattern": "2006-01-02"
}
```

支持的输入格式：
- `2006-01-02`
- `2006/01/02`
- `02-01-2006`
- `2006-01-02 15:04:05`
- RFC3339
- RFC1123

#### number_format - 数字格式化
```json
{
  "name": "format_price",
  "field": "price",
  "type": "number_format"
}
```
- 输入: `"1,234.56"`
- 输出: `1234.56`

#### url_normalize - URL 规范化
```json
{
  "name": "normalize_url",
  "field": "website",
  "type": "url_normalize"
}
```
- 输入: `"www.example.com"`
- 输出: `"https://www.example.com"`

## 完整示例

### 用户数据采集

```sql
INSERT INTO collection_tasks (
    name, type, status, cron, 
    next_run_time, replicas, execution_timeout, max_retries, 
    config, created_at, updated_at
) VALUES (
    'user-data-collection',
    'database',
    'enabled',
    '0 */6 * * *',
    NOW() + INTERVAL '1 minute',
    1,
    300,
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
                "query": "SELECT id, username, email, phone, created_at FROM users WHERE created_at > DATE_SUB(NOW(), INTERVAL 6 HOUR)"
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
                    "name": "format_phone",
                    "field": "phone",
                    "type": "phone_format"
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
                "phone": "phone",
                "created_at": "registration_time"
            }
        }
    }',
    NOW(),
    NOW()
);
```

## 常见问题

### 1. 连接失败

**问题**: `连接数据库失败: dial tcp: i/o timeout`

**解决**:
- 检查网络连通性
- 验证主机名和端口
- 检查防火墙规则
- 确认数据库服务运行中

### 2. 认证失败

**问题**: `Access denied for user`

**解决**:
- 验证用户名和密码
- 检查用户权限
- 确认数据库访问权限

### 3. 查询超时

**问题**: `context deadline exceeded`

**解决**:
- 优化 SQL 查询
- 添加索引
- 增加超时时间
- 限制返回数据量

### 4. 清洗失败

**问题**: `无法解析日期: xxx`

**解决**:
- 检查数据格式
- 使用正确的 pattern
- 处理 NULL 值
- 添加数据验证

## 性能优化

### 1. 查询优化

```sql
-- ❌ 不好：全表扫描
SELECT * FROM users;

-- ✅ 好：使用索引和时间范围
SELECT id, name, email 
FROM users 
WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)
AND status = 'active'
LIMIT 1000;
```

### 2. 连接池配置

```go
db.SetMaxOpenConns(10)      // 最大连接数
db.SetMaxIdleConns(5)       // 最大空闲连接
db.SetConnMaxLifetime(time.Hour)  // 连接最大生命周期
```

### 3. 批量处理

```sql
-- 使用 LIMIT 控制数据量
SELECT * FROM large_table 
WHERE id > :last_id 
ORDER BY id 
LIMIT 1000;
```

## 安全建议

### 1. 密码管理

```yaml
# 使用 Kubernetes Secret
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
type: Opaque
data:
  username: base64_encoded_username
  password: base64_encoded_password
```

### 2. 最小权限原则

```sql
-- 创建只读用户
CREATE USER 'datafusion_readonly'@'%' IDENTIFIED BY 'password';
GRANT SELECT ON app_db.* TO 'datafusion_readonly'@'%';
FLUSH PRIVILEGES;
```

### 3. 网络隔离

- 使用 VPC/私有网络
- 配置安全组规则
- 启用 SSL/TLS 连接

## 监控和调试

### 1. 查看执行日志

```bash
kubectl logs -n datafusion -l app=datafusion-worker -f | grep "数据库采集"
```

### 2. 查看任务执行记录

```sql
SELECT 
    te.id,
    ct.name,
    te.status,
    te.records_collected,
    te.error_message,
    te.start_time,
    te.end_time
FROM task_executions te
JOIN collection_tasks ct ON te.task_id = ct.id
WHERE ct.type = 'database'
ORDER BY te.start_time DESC
LIMIT 20;
```

### 3. 测试查询

```bash
# MySQL
mysql -h <host> -u <user> -p<password> <database> -e "YOUR_QUERY"

# PostgreSQL
psql -h <host> -U <user> -d <database> -c "YOUR_QUERY"
```

## 最佳实践

1. **增量采集**: 使用时间戳字段进行增量查询
2. **数据验证**: 使用清洗规则验证数据质量
3. **错误处理**: 配置合理的重试次数
4. **监控告警**: 监控任务执行状态
5. **定期维护**: 清理历史执行记录
