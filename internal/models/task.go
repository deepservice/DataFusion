package models

import "time"

// CollectionTask 采集任务
type CollectionTask struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // web-rpa, api, database
	Status          string    `json:"status"` // enabled, disabled
	Cron            string    `json:"cron"`
	NextRunTime     time.Time `json:"next_run_time"`
	Replicas        int       `json:"replicas"`
	ExecutionTimeout int      `json:"execution_timeout"`
	MaxRetries      int       `json:"max_retries"`
	Config          string    `json:"config"` // JSON 配置
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID               int64     `json:"id"`
	TaskID           int64     `json:"task_id"`
	WorkerPod        string    `json:"worker_pod"`
	Status           string    `json:"status"` // running, success, failed
	StartTime        time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	RecordsCollected int       `json:"records_collected"`
	ErrorMessage     string    `json:"error_message"`
	RetryCount       int       `json:"retry_count"`
}

// TaskConfig 任务配置
type TaskConfig struct {
	DataSource DataSourceConfig `json:"data_source"`
	Processor  ProcessorConfig  `json:"processor"`
	Storage    StorageConfig    `json:"storage"`
}

// DataSourceConfig 数据源配置
type DataSourceConfig struct {
	Type       string                 `json:"type"` // web-rpa, api, database
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Headers    map[string]string      `json:"headers"`
	Selectors  map[string]string      `json:"selectors"`
	RPAConfig  *RPAConfig             `json:"rpa_config,omitempty"`
	APIConfig  *APIConfig             `json:"api_config,omitempty"`
	DBConfig   *DBConfig              `json:"db_config,omitempty"`
}

// RPAConfig RPA 配置
type RPAConfig struct {
	BrowserType  string `json:"browser_type"`
	Headless     bool   `json:"headless"`
	WaitStrategy string `json:"wait_strategy"`
	Timeout      int    `json:"timeout"`
}

// APIConfig API 配置
type APIConfig struct {
	AuthType string            `json:"auth_type"`
	AuthData map[string]string `json:"auth_data"`
	Timeout  int               `json:"timeout"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Query    string `json:"query"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	CleaningRules  []CleaningRule  `json:"cleaning_rules"`
	TransformRules []TransformRule `json:"transform_rules"`
}

// CleaningRule 清洗规则
type CleaningRule struct {
	Name       string `json:"name"`
	Field      string `json:"field"`
	Type       string `json:"type"` // regex, trim, remove_html, etc.
	Pattern    string `json:"pattern"`
	Replacement string `json:"replacement"`
}

// TransformRule 转换规则
type TransformRule struct {
	Name       string `json:"name"`
	SourceField string `json:"source_field"`
	TargetField string `json:"target_field"`
	Type       string `json:"type"` // convert, format, etc.
}

// StorageConfig 存储配置
type StorageConfig struct {
	Target   string            `json:"target"` // postgresql, mongodb, file
	Database string            `json:"database"`
	Table    string            `json:"table"`
	Mapping  map[string]string `json:"mapping"`
}
