package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/datafusion/worker/internal/database"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	db *database.PostgresDB
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *database.PostgresDB) *HealthChecker {
	return &HealthChecker{db: db}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// HealthzHandler 健康检查端点 (/healthz)
// 检查服务是否存活
func (h *HealthChecker) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	status := &HealthStatus{
		Status:    "ok",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// ReadyzHandler 就绪检查端点 (/readyz)
// 检查服务是否准备好接收请求
func (h *HealthChecker) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	status := &HealthStatus{
		Status:    "ok",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// 检查数据库连接
	if err := h.checkDatabase(); err != nil {
		status.Status = "error"
		status.Checks["database"] = fmt.Sprintf("failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
		return
	}
	status.Checks["database"] = "ok"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// checkDatabase 检查数据库连接
func (h *HealthChecker) checkDatabase() error {
	// 尝试查询数据库
	_, err := h.db.GetPendingTasks("health-check")
	return err
}

// StartHealthServer 启动健康检查服务器
func StartHealthServer(port int, checker *HealthChecker) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", checker.HealthzHandler)
	mux.HandleFunc("/readyz", checker.ReadyzHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return server.ListenAndServe()
}
