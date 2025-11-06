package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Test представляет информацию о тесте
type Test struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // pending, running, completed, failed, stopped
	ProxyCount  int       `json:"proxy_count"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// TestResult представляет результаты теста
type TestResult struct {
	TestID       string        `json:"test_id"`
	TotalProxies int           `json:"total_proxies"`
	Successful   int           `json:"successful"`
	Failed       int           `json:"failed"`
	SuccessRate  float64       `json:"success_rate"`
	AverageLatency string      `json:"average_latency"`
	WorkingProxies []ProxyInfo `json:"working_proxies"`
}

// ProxyInfo представляет информацию о прокси
type ProxyInfo struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Latency  string `json:"latency"`
	Rank     int    `json:"rank"`
}

// APIServer представляет API сервер
type APIServer struct {
	tests    map[string]*Test
	results  map[string]*TestResult
	mu       sync.RWMutex
}

// NewAPIServer создает новый API сервер
func NewAPIServer() *APIServer {
	return &APIServer{
		tests:   make(map[string]*Test),
		results: make(map[string]*TestResult),
	}
}

// JSONResponse отправляет JSON ответ
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HealthHandler обрабатывает health check
func (s *APIServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"service":   "proxy-test-api",
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// StatusHandler обрабатывает статус системы
func (s *APIServer) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	response := map[string]interface{}{
		"system":       "proxy-test-api",
		"status":       "running",
		"active_tests": len(s.tests),
		"total_results": len(s.results),
		"timestamp":    time.Now().Format(time.RFC3339),
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// ConfigHandler обрабатывает конфигурацию
func (s *APIServer) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	config := map[string]interface{}{
		"xray": map[string]interface{}{
			"start_port": 10000,
			"log_level": "error",
		},
		"proxy": map[string]interface{}{
			"check_method": "ip",
			"ip_check_url": "https://api.ipify.org?format=text",
			"timeout": 30,
			"simulate_latency": false,
		},
		"api": map[string]interface{}{
			"port": 8080,
			"max_concurrent_tests": 5,
		},
	}
	
	response := map[string]interface{}{
		"config": config,
		"last_updated": time.Now().Format(time.RFC3339),
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// StartTestHandler обрабатывает запуск теста
func (s *APIServer) StartTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Name       string `json:"name"`
		ProxyCount int    `json:"proxy_count"`
		Timeout    int    `json:"timeout"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Валидация
	if request.ProxyCount <= 0 || request.ProxyCount > 100 {
		http.Error(w, "proxy_count must be between 1 and 100", http.StatusBadRequest)
		return
	}
	
	if request.Timeout <= 0 || request.Timeout > 300 {
		http.Error(w, "timeout must be between 1 and 300 seconds", http.StatusBadRequest)
		return
	}
	
	// Создаем тест
	testID := generateTestID()
	test := &Test{
		ID:         testID,
		Name:       request.Name,
		Status:     "running",
		ProxyCount: request.ProxyCount,
		StartedAt:  time.Now(),
	}
	
	s.mu.Lock()
	s.tests[testID] = test
	s.mu.Unlock()
	
	// Запускаем тест в горутине
	go s.runTest(testID, request.ProxyCount)
	
	response := map[string]interface{}{
		"test_id":    testID,
		"status":     "started",
		"message":    "Test started successfully",
		"started_at": test.StartedAt.Format(time.RFC3339),
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// GetTestStatusHandler обрабатывает статус теста
func (s *APIServer) GetTestStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	testID := r.URL.Path[len("/api/v1/tests/"):]
	
	s.mu.RLock()
	test, exists := s.tests[testID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Test not found", http.StatusNotFound)
		return
	}
	
	completedAt := ""
	if !test.CompletedAt.IsZero() {
		completedAt = test.CompletedAt.Format(time.RFC3339)
	}
	
	response := map[string]interface{}{
		"test_id":      test.ID,
		"name":         test.Name,
		"status":       test.Status,
		"proxy_count":  test.ProxyCount,
		"started_at":   test.StartedAt.Format(time.RFC3339),
		"completed_at": completedAt,
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// GetResultsHandler обрабатывает результаты теста
func (s *APIServer) GetResultsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	testID := r.URL.Path[len("/api/v1/results/"):]
	
	s.mu.RLock()
	result, exists := s.results[testID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Results not found", http.StatusNotFound)
		return
	}
	
	JSONResponse(w, http.StatusOK, result)
}

// GetWorkingProxiesHandler обрабатывает список рабочих прокси
func (s *APIServer) GetWorkingProxiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	testID := r.URL.Path[len("/api/v1/results/"):]
	testID = testID[:len(testID)-len("/working")]
	
	s.mu.RLock()
	result, exists := s.results[testID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Results not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"test_id":       testID,
		"working_proxies": result.WorkingProxies,
		"count":         len(result.WorkingProxies),
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// runTest запускает тест (заглушка для демонстрации)
func (s *APIServer) runTest(testID string, proxyCount int) {
	// Имитируем выполнение теста
	time.Sleep(5 * time.Second)
	
	// Создаем фиктивные результаты
	workingProxies := []ProxyInfo{
		{
			Name:     "🇳🇱[openproxylist.com] ss-NL",
			Protocol: "shadowsocks",
			Server:   "45.87.175.28",
			Port:     8080,
			Latency:  "1.108s",
			Rank:     1,
		},
		{
			Name:     "🇬🇧GB-141.98.101.178-3885",
			Protocol: "shadowsocks",
			Server:   "141.98.101.178",
			Port:     443,
			Latency:  "1.256s",
			Rank:     2,
		},
	}
	
	result := &TestResult{
		TestID:       testID,
		TotalProxies: proxyCount,
		Successful:   len(workingProxies),
		Failed:       proxyCount - len(workingProxies),
		SuccessRate:  float64(len(workingProxies)) / float64(proxyCount) * 100,
		AverageLatency: "1.182s",
		WorkingProxies: workingProxies,
	}
	
	s.mu.Lock()
	s.results[testID] = result
	
	// Обновляем статус теста
	if test, exists := s.tests[testID]; exists {
		test.Status = "completed"
		test.CompletedAt = time.Now()
	}
	s.mu.Unlock()
}

// generateTestID генерирует уникальный ID теста
func generateTestID() string {
	return "test_" + time.Now().Format("20060102150405")
}

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	server := NewAPIServer()
	
	// Настраиваем маршруты
	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", server.HealthHandler)
	
	// API routes
	mux.HandleFunc("/api/v1/status", server.StatusHandler)
	mux.HandleFunc("/api/v1/config", server.ConfigHandler)
	mux.HandleFunc("/api/v1/tests", server.StartTestHandler)
	mux.HandleFunc("/api/v1/tests/", server.GetTestStatusHandler)
	mux.HandleFunc("/api/v1/results/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/results/" {
			http.NotFound(w, r)
			return
		}
		
		if r.URL.Path[len(r.URL.Path)-8:] == "/working" {
			server.GetWorkingProxiesHandler(w, r)
		} else {
			server.GetResultsHandler(w, r)
		}
	})
	
	// Добавляем CORS middleware
	handler := CORSMiddleware(mux)
	
	log.Println("🚀 Proxy Test API server starting on :8080")
	log.Println("🔍 Health check: http://localhost:8080/health")
	log.Println("📊 System status: http://localhost:8080/api/v1/status")
	
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}