// API 响应基础类型
export interface ApiResponse<T = any> {
  success?: boolean;
  message?: string;
  data?: T;
  error?: string;
}

// 分页响应类型
export interface PaginatedResponse<T> {
  items: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
  };
}

// 用户相关类型
export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  status: string;
  auth_type: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  expires_at: string;
  user: User;
}

// 任务相关类型
export interface Task {
  id: number;
  name: string;
  description: string;
  type: string;
  data_source_id: number;
  cron: string;
  next_run_time: string;
  status: string;
  replicas: number;
  execution_timeout: number;
  max_retries: number;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface TaskExecution {
  id: number;
  task_id: number;
  task_name?: string;
  worker_pod: string;
  status: string;
  start_time: string;
  end_time?: string;
  records_collected: number;
  error_message?: string;
  retry_count: number;
  created_at: string;
}

// 数据源相关类型
export interface DataSource {
  id: number;
  name: string;
  type: string;
  config: Record<string, any>;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// 清洗规则相关类型
export interface CleaningRule {
  id: number;
  name: string;
  description: string;
  rule_type: string;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
}

// API 密钥相关类型
export interface ApiKey {
  id: number;
  name: string;
  description: string;
  permissions: string[];
  expires_at?: string;
  last_used_at?: string;
  status: string;
  created_at: string;
}

export interface CreateApiKeyResponse {
  api_key: string;
  key_info: ApiKey;
}

// 配置相关类型
export interface ServerConfig {
  port: number;
  mode: string;
  read_timeout: number;
  write_timeout: number;
}

export interface AuthConfig {
  jwt: {
    secret_key: string;
    token_duration: string;
  };
  password: {
    min_length: number;
    require_upper: boolean;
    require_lower: boolean;
    require_digit: boolean;
    require_special: boolean;
  };
}

export interface DatabaseConfig {
  postgresql: {
    host: string;
    port: number;
    user: string;
    password: string;
    database: string;
    sslmode: string;
    max_open_conns: number;
    max_idle_conns: number;
    conn_max_lifetime: number;
  };
}

export interface LogConfig {
  level: string;
  format: string;
}

export interface SystemConfig {
  server: ServerConfig;
  auth: AuthConfig;
  database: DatabaseConfig;
  log: LogConfig;
}

// 备份相关类型
export interface BackupInfo {
  filename: string;
  path: string;
  size: number;
  mod_time: string;
  compressed: boolean;
}

export interface BackupResult {
  success: boolean;
  filename: string;
  size: number;
  duration: string;
  start_time: string;
  end_time: string;
  error?: string;
}

export interface BackupOptions {
  output_dir?: string;
  filename?: string;
  compress?: boolean;
  schema_only?: boolean;
  data_only?: boolean;
  tables?: string[];
  exclude_tables?: string[];
}

export interface SchedulerConfig {
  enabled: boolean;
  cron_expression: string;
  backup_dir: string;
  retention_days: number;
  max_backups: number;
  compress_backups: boolean;
  notify_on_failure: boolean;
  notify_on_success: boolean;
}

// 统计相关类型
export interface SystemStats {
  total_tasks: number;
  active_tasks: number;
  total_executions: number;
  success_rate: number;
  total_data_sources: number;
  total_users: number;
}

// 角色相关类型
export interface Role {
  name: string;
  description: string;
  permissions: Array<{
    resource: string;
    action: string;
  }>;
}

// 表单相关类型
export interface FormField {
  name: string;
  label: string;
  type: 'input' | 'password' | 'select' | 'textarea' | 'number' | 'switch' | 'date';
  required?: boolean;
  options?: Array<{ label: string; value: any }>;
  placeholder?: string;
  rules?: any[];
}

// 菜单相关类型
export interface MenuItem {
  key: string;
  label: string;
  icon?: React.ReactNode;
  children?: MenuItem[];
  path?: string;
  permission?: string;
}