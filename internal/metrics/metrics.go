package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 指标收集器
type Metrics struct {
	// 任务执行指标
	TaskExecutionTotal   *prometheus.CounterVec
	TaskExecutionSuccess *prometheus.CounterVec
	TaskExecutionFailure *prometheus.CounterVec
	TaskDuration         *prometheus.HistogramVec

	// 数据采集指标
	DataRecordsCollected *prometheus.CounterVec
	DataRecordsStored    *prometheus.CounterVec
	DataRecordsFailed    *prometheus.CounterVec

	// 数据处理指标
	DataCleaningTotal    *prometheus.CounterVec
	DataCleaningDuration *prometheus.HistogramVec
	DataDeduplicationTotal *prometheus.CounterVec
	DataDuplicatesRemoved  *prometheus.CounterVec

	// 存储指标
	StorageOperationTotal    *prometheus.CounterVec
	StorageOperationDuration *prometheus.HistogramVec
	StorageErrors            *prometheus.CounterVec

	// Worker 状态指标
	RunningTasks      prometheus.Gauge
	WorkerStartTime   prometheus.Gauge
	WorkerUptime      prometheus.Gauge
	TaskQueueLength   prometheus.Gauge

	// 数据库连接池指标
	DBConnectionsActive prometheus.Gauge
	DBConnectionsIdle   prometheus.Gauge
	DBConnectionsTotal  prometheus.Gauge

	// 缓存指标
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec

	// 错误和重试指标
	ErrorTotal       *prometheus.CounterVec
	RetryTotal       *prometheus.CounterVec
	RetrySuccess     *prometheus.CounterVec
	RetryExhausted   *prometheus.CounterVec
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	m := &Metrics{
		// 任务执行指标
		TaskExecutionTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_task_execution_total",
				Help: "Total number of task executions",
			},
			[]string{"task_name", "task_type", "worker"},
		),
		TaskExecutionSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_task_execution_success_total",
				Help: "Total number of successful task executions",
			},
			[]string{"task_name", "task_type", "worker"},
		),
		TaskExecutionFailure: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_task_execution_failure_total",
				Help: "Total number of failed task executions",
			},
			[]string{"task_name", "task_type", "worker"},
		),
		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "datafusion_task_duration_seconds",
				Help:    "Task execution duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
			},
			[]string{"task_name", "task_type", "worker"},
		),

		// 数据采集指标
		DataRecordsCollected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_records_collected_total",
				Help: "Total number of data records collected",
			},
			[]string{"task_name", "task_type", "source_type", "worker"},
		),
		DataRecordsStored: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_records_stored_total",
				Help: "Total number of data records stored",
			},
			[]string{"task_name", "storage_type", "worker"},
		),
		DataRecordsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_records_failed_total",
				Help: "Total number of data records failed to store",
			},
			[]string{"task_name", "storage_type", "worker"},
		),

		// 数据处理指标
		DataCleaningTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_cleaning_total",
				Help: "Total number of data cleaning operations",
			},
			[]string{"task_name", "rule_type", "worker"},
		),
		DataCleaningDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "datafusion_data_cleaning_duration_seconds",
				Help:    "Data cleaning duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"task_name", "worker"},
		),
		DataDeduplicationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_deduplication_total",
				Help: "Total number of data deduplication operations",
			},
			[]string{"task_name", "strategy", "worker"},
		),
		DataDuplicatesRemoved: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_data_duplicates_removed_total",
				Help: "Total number of duplicate records removed",
			},
			[]string{"task_name", "strategy", "worker"},
		),

		// 存储指标
		StorageOperationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_storage_operation_total",
				Help: "Total number of storage operations",
			},
			[]string{"operation", "storage_type", "worker"},
		),
		StorageOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "datafusion_storage_operation_duration_seconds",
				Help:    "Storage operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "storage_type", "worker"},
		),
		StorageErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_storage_errors_total",
				Help: "Total number of storage errors",
			},
			[]string{"operation", "storage_type", "error_type", "worker"},
		),

		// Worker 状态指标
		RunningTasks: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_running_tasks",
				Help: "Number of currently running tasks",
			},
		),
		WorkerStartTime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_worker_start_time_seconds",
				Help: "Unix timestamp when the worker started",
			},
		),
		WorkerUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_worker_uptime_seconds",
				Help: "Worker uptime in seconds",
			},
		),
		TaskQueueLength: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_task_queue_length",
				Help: "Number of tasks in the queue",
			},
		),

		// 数据库连接池指标
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		DBConnectionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datafusion_db_connections_total",
				Help: "Total number of database connections",
			},
		),

		// 缓存指标
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "worker"},
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "worker"},
		),
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "datafusion_cache_size",
				Help: "Current cache size",
			},
			[]string{"cache_type", "worker"},
		),

		// 错误和重试指标
		ErrorTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_errors_total",
				Help: "Total number of errors",
			},
			[]string{"error_type", "component", "worker"},
		),
		RetryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_retry_total",
				Help: "Total number of retry attempts",
			},
			[]string{"task_name", "worker"},
		),
		RetrySuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_retry_success_total",
				Help: "Total number of successful retries",
			},
			[]string{"task_name", "worker"},
		),
		RetryExhausted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datafusion_retry_exhausted_total",
				Help: "Total number of exhausted retries",
			},
			[]string{"task_name", "worker"},
		),
	}

	// 设置 Worker 启动时间
	m.WorkerStartTime.SetToCurrentTime()

	// 启动 uptime 更新
	go m.updateUptime()

	return m
}

// updateUptime 定期更新 uptime
func (m *Metrics) updateUptime() {
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.WorkerUptime.Set(time.Since(startTime).Seconds())
	}
}

// RecordTaskExecution 记录任务执行
func (m *Metrics) RecordTaskExecution(taskName, taskType, status, worker string, duration time.Duration, recordsCollected int) {
	m.TaskExecutionTotal.WithLabelValues(taskName, taskType, worker).Inc()
	
	if status == "success" {
		m.TaskExecutionSuccess.WithLabelValues(taskName, taskType, worker).Inc()
	} else {
		m.TaskExecutionFailure.WithLabelValues(taskName, taskType, worker).Inc()
	}
	
	m.TaskDuration.WithLabelValues(taskName, taskType, worker).Observe(duration.Seconds())
}

// RecordDataCollected 记录数据采集
func (m *Metrics) RecordDataCollected(taskName, taskType, sourceType, worker string, count int) {
	m.DataRecordsCollected.WithLabelValues(taskName, taskType, sourceType, worker).Add(float64(count))
}

// RecordDataStored 记录数据存储
func (m *Metrics) RecordDataStored(taskName, storageType, worker string, count int) {
	m.DataRecordsStored.WithLabelValues(taskName, storageType, worker).Add(float64(count))
}

// RecordStorageOperation 记录存储操作
func (m *Metrics) RecordStorageOperation(operation, storageType, worker string, duration time.Duration) {
	m.StorageOperationTotal.WithLabelValues(operation, storageType, worker).Inc()
	m.StorageOperationDuration.WithLabelValues(operation, storageType, worker).Observe(duration.Seconds())
}

// RecordDeduplication 记录去重操作
func (m *Metrics) RecordDeduplication(taskName, strategy, worker string, totalRecords, duplicates int) {
	m.DataDeduplicationTotal.WithLabelValues(taskName, strategy, worker).Inc()
	m.DataDuplicatesRemoved.WithLabelValues(taskName, strategy, worker).Add(float64(duplicates))
}

// RecordError 记录错误
func (m *Metrics) RecordError(errorType, component, worker string) {
	m.ErrorTotal.WithLabelValues(errorType, component, worker).Inc()
}

// RecordRetry 记录重试
func (m *Metrics) RecordRetry(taskName, worker string, success bool) {
	m.RetryTotal.WithLabelValues(taskName, worker).Inc()
	if success {
		m.RetrySuccess.WithLabelValues(taskName, worker).Inc()
	}
}

// SetRunningTasks 设置当前运行任务数
func (m *Metrics) SetRunningTasks(count int) {
	m.RunningTasks.Set(float64(count))
}

// StartMetricsServer 启动指标服务器
func StartMetricsServer(port int) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
