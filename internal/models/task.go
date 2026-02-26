package models

import "time"

// CollectionTask 采集任务
type CollectionTask struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"` // web-rpa, api, database
	Status          string     `json:"status"` // enabled, disabled
	DataSourceID    int64      `json:"data_source_id"`
	Cron            *string    `json:"cron"`
	NextRunTime     *time.Time `json:"next_run_time"`
	Replicas        int        `json:"replicas"`
	ExecutionTimeout int       `json:"execution_timeout"`
	MaxRetries      int        `json:"max_retries"`
	Config          *string    `json:"config"` // JSON 配置
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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

// RPALoginConfig 登录配置
type RPALoginConfig struct {
	URL              string `json:"url,omitempty"`            // 登录页URL，为空则使用主URL
	UsernameSelector string `json:"username_selector"`        // 用户名输入框选择器
	PasswordSelector string `json:"password_selector"`        // 密码输入框选择器
	SubmitSelector   string `json:"submit_selector"`          // 提交按钮选择器
	Username         string `json:"username"`                 // 用户名
	Password         string `json:"password"`                 // 密码
	WaitAfter        string `json:"wait_after,omitempty"`     // 登录成功后等待的元素选择器
	CheckSelector    string `json:"check_selector,omitempty"` // 检测会话有效性的元素选择器（不存在则认为已过期）
}

// RPAPageAction 页面动作（搜索/筛选/点击等）
type RPAPageAction struct {
	Type     string `json:"type"`              // input, click, select, wait
	Selector string `json:"selector"`          // 目标元素选择器
	Value    string `json:"value,omitempty"`   // 输入值（input/select时使用）
	WaitFor  string `json:"wait_for,omitempty"` // 动作完成后等待某元素出现
	WaitMs   int    `json:"wait_ms,omitempty"` // 等待毫秒数（type=wait时使用）
}

// RPACookieParam 手动配置的 Cookie 参数（用于短信/扫码等无法自动登录的场景）
type RPACookieParam struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain,omitempty"` // 可选，未填则从 URL 自动推断
	Path   string `json:"path,omitempty"`   // 可选，默认 "/"
}

// RPAConfig RPA 配置
type RPAConfig struct {
	BrowserType  string          `json:"browser_type"`
	Headless     bool            `json:"headless"`
	WaitStrategy string          `json:"wait_strategy"`
	Timeout      int             `json:"timeout"`
	Login        *RPALoginConfig  `json:"login,omitempty"`   // 登录配置（用户名/密码）
	Actions      []RPAPageAction  `json:"actions,omitempty"` // 页面动作序列
	// Cookie 注入（适用于短信验证码、扫码登录等无法自动模拟的场景）
	InitialCookies []*RPACookieParam `json:"initial_cookies,omitempty"` // 手动指定初始 Cookie 列表
	CookieString   string            `json:"cookie_string,omitempty"`   // 浏览器 Cookie 字符串（格式：name=val; name2=val2）
	CheckSelector  string            `json:"check_selector,omitempty"`  // 会话有效性检测选择器（元素不存在则报错提示重新配置 Cookie）
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
