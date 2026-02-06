package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// LoadTestConfig 负载测试配置
type LoadTestConfig struct {
	BaseURL         string
	ConcurrentUsers int
	TestDuration    time.Duration
	RequestsPerUser int
	AuthToken       string
}

// TestResult 测试结果
type TestResult struct {
	TotalRequests     int
	SuccessRequests   int
	FailedRequests    int
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	MinLatency        time.Duration
	RequestsPerSecond float64
	Errors            []string
}

// LoadTester 负载测试器
type LoadTester struct {
	config *LoadTestConfig
	client *http.Client
}

// NewLoadTester 创建负载测试器
func NewLoadTester(config *LoadTestConfig) *LoadTester {
	return &LoadTester{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RunAPILoadTest 运行API负载测试
func (lt *LoadTester) RunAPILoadTest(t *testing.T) *TestResult {
	var wg sync.WaitGroup
	var mu sync.Mutex

	result := &TestResult{
		MinLatency: time.Hour, // 初始化为很大的值
		Errors:     make([]string, 0),
	}

	startTime := time.Now()

	// 启动并发用户
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			lt.runUserSession(userID, result, &mu)
		}(i)
	}

	wg.Wait()

	totalDuration := time.Since(startTime)

	// 计算统计信息
	if result.TotalRequests > 0 {
		result.RequestsPerSecond = float64(result.TotalRequests) / totalDuration.Seconds()
		result.AverageLatency = result.AverageLatency / time.Duration(result.TotalRequests)
	}

	return result
}

// runUserSession 运行单个用户会话
func (lt *LoadTester) runUserSession(userID int, result *TestResult, mu *sync.Mutex) {
	for i := 0; i < lt.config.RequestsPerUser; i++ {
		// 测试不同的API端点
		endpoints := []string{
			"/api/v1/tasks",
			"/api/v1/datasources",
			"/api/v1/stats/overview",
			"/api/v1/executions",
		}

		for _, endpoint := range endpoints {
			latency, err := lt.makeRequest(endpoint)

			mu.Lock()
			result.TotalRequests++

			if err != nil {
				result.FailedRequests++
				result.Errors = append(result.Errors, fmt.Sprintf("User %d: %v", userID, err))
			} else {
				result.SuccessRequests++
				result.AverageLatency += latency

				if latency > result.MaxLatency {
					result.MaxLatency = latency
				}
				if latency < result.MinLatency {
					result.MinLatency = latency
				}
			}
			mu.Unlock()
		}

		// 模拟用户思考时间
		time.Sleep(100 * time.Millisecond)
	}
}

// makeRequest 发送HTTP请求
func (lt *LoadTester) makeRequest(endpoint string) (time.Duration, error) {
	url := lt.config.BaseURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// 添加认证头
	if lt.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+lt.config.AuthToken)
	}

	start := time.Now()
	resp, err := lt.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return latency, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return latency, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return latency, nil
}

// TestTaskCreationLoad 测试任务创建负载
func TestTaskCreationLoad(t *testing.T) {
	config := &LoadTestConfig{
		BaseURL:         "http://localhost:8080",
		ConcurrentUsers: 10,
		TestDuration:    30 * time.Second,
		RequestsPerUser: 5,
		AuthToken:       getTestAuthToken(t),
	}

	tester := NewLoadTester(config)
	result := tester.RunTaskCreationLoad(t)

	// 验证结果
	if result.SuccessRequests == 0 {
		t.Fatal("没有成功的请求")
	}

	if result.AverageLatency > 5*time.Second {
		t.Errorf("平均延迟过高: %v", result.AverageLatency)
	}

	if float64(result.FailedRequests)/float64(result.TotalRequests) > 0.05 {
		t.Errorf("失败率过高: %.2f%%", float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	}

	t.Logf("负载测试结果:")
	t.Logf("  总请求数: %d", result.TotalRequests)
	t.Logf("  成功请求: %d", result.SuccessRequests)
	t.Logf("  失败请求: %d", result.FailedRequests)
	t.Logf("  平均延迟: %v", result.AverageLatency)
	t.Logf("  最大延迟: %v", result.MaxLatency)
	t.Logf("  最小延迟: %v", result.MinLatency)
	t.Logf("  QPS: %.2f", result.RequestsPerSecond)
}

// RunTaskCreationLoad 运行任务创建负载测试
func (lt *LoadTester) RunTaskCreationLoad(t *testing.T) *TestResult {
	var wg sync.WaitGroup
	var mu sync.Mutex

	result := &TestResult{
		MinLatency: time.Hour,
		Errors:     make([]string, 0),
	}

	startTime := time.Now()

	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < lt.config.RequestsPerUser; j++ {
				latency, err := lt.createTestTask(userID, j)

				mu.Lock()
				result.TotalRequests++

				if err != nil {
					result.FailedRequests++
					result.Errors = append(result.Errors, fmt.Sprintf("User %d Task %d: %v", userID, j, err))
				} else {
					result.SuccessRequests++
					result.AverageLatency += latency

					if latency > result.MaxLatency {
						result.MaxLatency = latency
					}
					if latency < result.MinLatency {
						result.MinLatency = latency
					}
				}
				mu.Unlock()

				time.Sleep(200 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	totalDuration := time.Since(startTime)

	if result.TotalRequests > 0 {
		result.RequestsPerSecond = float64(result.TotalRequests) / totalDuration.Seconds()
		result.AverageLatency = result.AverageLatency / time.Duration(result.TotalRequests)
	}

	return result
}

// createTestTask 创建测试任务
func (lt *LoadTester) createTestTask(userID, taskID int) (time.Duration, error) {
	taskData := map[string]interface{}{
		"name":        fmt.Sprintf("LoadTest-User%d-Task%d", userID, taskID),
		"description": "Load test task",
		"type":        "api",
		"config": map[string]interface{}{
			"url":    "https://api.example.com/data",
			"method": "GET",
		},
		"schedule": "0 */5 * * * *", // 每5分钟执行一次
	}

	jsonData, err := json.Marshal(taskData)
	if err != nil {
		return 0, err
	}

	url := lt.config.BaseURL + "/api/v1/tasks"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if lt.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+lt.config.AuthToken)
	}

	start := time.Now()
	resp, err := lt.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return latency, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return latency, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return latency, nil
}

// TestCachePerformance 测试缓存性能
func TestCachePerformance(t *testing.T) {
	config := &LoadTestConfig{
		BaseURL:         "http://localhost:8080",
		ConcurrentUsers: 20,
		RequestsPerUser: 10,
		AuthToken:       getTestAuthToken(t),
	}

	tester := NewLoadTester(config)

	// 测试缓存命中性能
	cacheHitLatency := tester.testCacheHitPerformance(t)

	// 测试缓存未命中性能
	cacheMissLatency := tester.testCacheMissPerformance(t)

	t.Logf("缓存命中平均延迟: %v", cacheHitLatency)
	t.Logf("缓存未命中平均延迟: %v", cacheMissLatency)

	// 缓存命中应该明显快于缓存未命中
	if cacheHitLatency >= cacheMissLatency {
		t.Error("缓存命中延迟应该小于缓存未命中延迟")
	}
}

// testCacheHitPerformance 测试缓存命中性能
func (lt *LoadTester) testCacheHitPerformance(t *testing.T) time.Duration {
	// 先预热缓存
	lt.makeRequest("/api/v1/stats/overview")
	time.Sleep(100 * time.Millisecond)

	var totalLatency time.Duration
	requests := 50

	for i := 0; i < requests; i++ {
		latency, err := lt.makeRequest("/api/v1/stats/overview")
		if err != nil {
			t.Logf("缓存命中测试请求失败: %v", err)
			continue
		}
		totalLatency += latency
	}

	return totalLatency / time.Duration(requests)
}

// testCacheMissPerformance 测试缓存未命中性能
func (lt *LoadTester) testCacheMissPerformance(t *testing.T) time.Duration {
	var totalLatency time.Duration
	requests := 10

	for i := 0; i < requests; i++ {
		// 每次请求不同的数据以避免缓存命中
		endpoint := fmt.Sprintf("/api/v1/tasks?page=%d", i)
		latency, err := lt.makeRequest(endpoint)
		if err != nil {
			t.Logf("缓存未命中测试请求失败: %v", err)
			continue
		}
		totalLatency += latency

		// 等待一段时间确保不会命中缓存
		time.Sleep(50 * time.Millisecond)
	}

	return totalLatency / time.Duration(requests)
}

// getTestAuthToken 获取测试用的认证令牌
func getTestAuthToken(t *testing.T) string {
	// 这里应该实现获取测试令牌的逻辑
	// 可以通过登录API获取，或者使用预设的测试令牌
	return "test-token-for-load-testing"
}

// BenchmarkAPIEndpoints 基准测试API端点
func BenchmarkAPIEndpoints(b *testing.B) {
	config := &LoadTestConfig{
		BaseURL:   "http://localhost:8080",
		AuthToken: "test-token",
	}

	tester := NewLoadTester(config)

	endpoints := []string{
		"/api/v1/tasks",
		"/api/v1/datasources",
		"/api/v1/stats/overview",
	}

	for _, endpoint := range endpoints {
		b.Run(endpoint, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := tester.makeRequest(endpoint)
				if err != nil {
					b.Errorf("请求失败: %v", err)
				}
			}
		})
	}
}

// TestMemoryUsage 测试内存使用情况
func TestMemoryUsage(t *testing.T) {
	// 这里可以添加内存使用监控的测试
	// 例如监控API服务器在负载测试期间的内存使用情况
	t.Skip("内存使用测试需要额外的监控工具")
}
