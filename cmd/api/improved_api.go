package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	ConfigFile  string    `json:"config_file"`
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
	TestDuration string        `json:"test_duration"`
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

// TestRequest представляет запрос на запуск теста
type TestRequest struct {
	Name        string `json:"name"`
	ProxyCount  int    `json:"proxy_count"`
	Timeout     int    `json:"timeout"`
	ConfigFile  string `json:"config_file"`
	StartPort   int    `json:"start_port"`
}

// APIServer представляет API сервер
type APIServer struct {
	tests    map[string]*Test
	results  map[string]*TestResult
	mu       sync.RWMutex
	port     int
	dataDir  string
}

// NewAPIServer создает новый API сервер
func NewAPIServer(port int, dataDir string) *APIServer {
	// Создаем директорию для данных если не существует
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: Could not create data directory: %v", err)
	}
	
	return &APIServer{
		tests:   make(map[string]*Test),
		results: make(map[string]*TestResult),
		port:    port,
		dataDir: dataDir,
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
		"port":      s.port,
		"data_dir":  s.dataDir,
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
	
	// Получаем список активных тестов
	activeTests := []string{}
	for id, test := range s.tests {
		if test.Status == "running" {
			activeTests = append(activeTests, id)
		}
	}
	
	response := map[string]interface{}{
		"system":       "proxy-test-api",
		"status":       "running",
		"port":         s.port,
		"active_tests": len(activeTests),
		"total_tests":  len(s.tests),
		"total_results": len(s.results),
		"active_test_ids": activeTests,
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
	
	// Проверяем существование файла конфигураций
	configFile := "/Users/t/zapret/test_xray_finish/deduplicated.json"
	fileInfo, err := os.Stat(configFile)
	configExists := err == nil
	configSize := int64(0)
	if configExists {
		configSize = fileInfo.Size()
	}
	
	config := map[string]interface{}{
		"xray": map[string]interface{}{
			"start_port": 20000,
			"log_level": "error",
		},
		"proxy": map[string]interface{}{
			"check_method": "ip",
			"ip_check_url": "https://api.ipify.org?format=text",
			"timeout": 30,
			"simulate_latency": false,
		},
		"api": map[string]interface{}{
			"port": s.port,
			"max_concurrent_tests": 5,
			"data_directory": s.dataDir,
		},
		"files": map[string]interface{}{
			"config_file": configFile,
			"config_exists": configExists,
			"config_size": configSize,
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
	
	var request TestRequest
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Устанавливаем значения по умолчанию
	if request.Name == "" {
		request.Name = "test-" + time.Now().Format("20060102-150405")
	}
	
	if request.ProxyCount <= 0 || request.ProxyCount > 100 {
		request.ProxyCount = 20
	}
	
	if request.Timeout <= 0 || request.Timeout > 300 {
		request.Timeout = 30
	}
	
	if request.ConfigFile == "" {
		request.ConfigFile = "/Users/t/zapret/test_xray_finish/deduplicated.json"
	}
	
	if request.StartPort <= 0 {
		request.StartPort = 20000
	}
	
	// Проверяем существование файла конфигураций
	if _, err := os.Stat(request.ConfigFile); os.IsNotExist(err) {
		http.Error(w, "Config file not found: "+request.ConfigFile, http.StatusBadRequest)
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
		ConfigFile: request.ConfigFile,
	}
	
	s.mu.Lock()
	s.tests[testID] = test
	s.mu.Unlock()
	
	// Запускаем тест в горутине
	go s.runTest(testID, request)
	
	response := map[string]interface{}{
		"test_id":     testID,
		"name":        test.Name,
		"status":      "started",
		"proxy_count": test.ProxyCount,
		"config_file": test.ConfigFile,
		"start_port":  request.StartPort,
		"message":     "Test started successfully",
		"started_at":  test.StartedAt.Format(time.RFC3339),
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
		http.Error(w, "Test not found: "+testID, http.StatusNotFound)
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
		"config_file":  test.ConfigFile,
		"started_at":   test.StartedAt.Format(time.RFC3339),
		"completed_at": completedAt,
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// ListTestsHandler обрабатывает список тестов
func (s *APIServer) ListTestsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	testsList := []map[string]interface{}{}
	for _, test := range s.tests {
		testInfo := map[string]interface{}{
			"id":          test.ID,
			"name":        test.Name,
			"status":      test.Status,
			"proxy_count": test.ProxyCount,
			"started_at":  test.StartedAt.Format(time.RFC3339),
		}
		
		if !test.CompletedAt.IsZero() {
			testInfo["completed_at"] = test.CompletedAt.Format(time.RFC3339)
		}
		
		testsList = append(testsList, testInfo)
	}
	
	response := map[string]interface{}{
		"tests": testsList,
		"count": len(testsList),
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
		http.Error(w, "Results not found: "+testID, http.StatusNotFound)
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
		http.Error(w, "Results not found: "+testID, http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"test_id":        testID,
		"working_proxies": result.WorkingProxies,
		"count":          len(result.WorkingProxies),
		"success_rate":   result.SuccessRate,
		"average_latency": result.AverageLatency,
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// ExportResultsHandler экспортирует результаты в файл
func (s *APIServer) ExportResultsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	testID := r.URL.Path[len("/api/v1/results/"):]
	testID = testID[:len(testID)-len("/export")]
	
	s.mu.RLock()
	result, exists := s.results[testID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Results not found: "+testID, http.StatusNotFound)
		return
	}
	
	// Создаем файл для экспорта
	exportFile := filepath.Join(s.dataDir, "proxies_"+testID+".txt")
	content := "# Proxy Test Results\n"
	content += "# Test ID: " + testID + "\n"
	content += "# Date: " + time.Now().Format("2006-01-02 15:04:05") + "\n"
	content += "# Total Proxies: " + fmt.Sprintf("%d", result.TotalProxies) + "\n"
	content += "# Successful: " + fmt.Sprintf("%d", result.Successful) + "\n"
	content += "# Success Rate: " + fmt.Sprintf("%.1f", result.SuccessRate) + "%\n"
	content += "# Average Latency: " + result.AverageLatency + "\n\n"
	
	for i, proxy := range result.WorkingProxies {
		content += fmt.Sprintf("%d. %s | %s:%d | %s | %s\n", 
			i+1, proxy.Name, proxy.Server, proxy.Port, proxy.Protocol, proxy.Latency)
	}
	
	// Сохраняем файл
	if err := os.WriteFile(exportFile, []byte(content), 0644); err != nil {
		http.Error(w, "Failed to export results: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"test_id": testID,
		"export_file": exportFile,
		"message": "Results exported successfully",
	}
	
	JSONResponse(w, http.StatusOK, response)
}

// runTest запускает тест (заглушка для демонстрации)
func (s *APIServer) runTest(testID string, request TestRequest) {
	startTime := time.Now()
	
	// Имитируем выполнение теста
	time.Sleep(5 * time.Second)
	
	// Создаем фиктивные результаты на основе реальных данных
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
		{
			Name:     "🇱🇹LT-45.87.175.197-0285",
			Protocol: "shadowsocks",
			Server:   "45.87.175.197",
			Port:     8080,
			Latency:  "2.965s",
			Rank:     3,
		},
	}
	
	duration := time.Since(startTime)
	
	result := &TestResult{
		TestID:       testID,
		TotalProxies: request.ProxyCount,
		Successful:   len(workingProxies),
		Failed:       request.ProxyCount - len(workingProxies),
		SuccessRate:  float64(len(workingProxies)) / float64(request.ProxyCount) * 100,
		AverageLatency: "1.776s",
		WorkingProxies: workingProxies,
		TestDuration:  duration.String(),
	}
	
	s.mu.Lock()
	s.results[testID] = result
	
	// Обновляем статус теста
	if test, exists := s.tests[testID]; exists {
		test.Status = "completed"
		test.CompletedAt = time.Now()
	}
	s.mu.Unlock()
	
	log.Printf("Test %s completed in %s", testID, duration)
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
	// Парсим аргументы командной строки
	port := flag.Int("port", 9090, "Port to run the API server on")
	dataDir := flag.String("data-dir", "/tmp/proxy-test-api", "Directory for storing test data")
	help := flag.Bool("help", false, "Show help")
	
	flag.Parse()
	
	if *help {
		fmt.Println("Proxy Test API Server")
		fmt.Println("Usage:")
		fmt.Println("  ./api_server [options]")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ./api_server -port 9090 -data-dir ./data")
		fmt.Println("  ./api_server -port 8080")
		return
	}
	
	server := NewAPIServer(*port, *dataDir)
	
	// Настраиваем маршруты
	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", server.HealthHandler)
	
	// API routes
	mux.HandleFunc("/api/v1/status", server.StatusHandler)
	mux.HandleFunc("/api/v1/config", server.ConfigHandler)
	mux.HandleFunc("/api/v1/tests", server.StartTestHandler)
	mux.HandleFunc("/api/v1/tests/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tests/" {
			server.ListTestsHandler(w, r)
		} else {
			server.GetTestStatusHandler(w, r)
		}
	})
	
	mux.HandleFunc("/api/v1/results/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/results/" {
			http.NotFound(w, r)
			return
		}
		
		if r.URL.Path[len(r.URL.Path)-8:] == "/working" {
			server.GetWorkingProxiesHandler(w, r)
		} else if r.URL.Path[len(r.URL.Path)-7:] == "/export" {
			server.ExportResultsHandler(w, r)
		} else {
			server.GetResultsHandler(w, r)
		}
	})
	
	// Добавляем CORS middleware
	handler := CORSMiddleware(mux)
	
	addr := fmt.Sprintf(":%d", *port)
	
	log.Printf("🚀 Proxy Test API server starting on %s", addr)
	log.Printf("🔍 Health check: http://localhost:%d/health", *port)
	log.Printf("📊 System status: http://localhost:%d/api/v1/status", *port)
	log.Printf("💾 Data directory: %s", *dataDir)
	log.Printf("⚙️  Use --help for command line options")
	
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}