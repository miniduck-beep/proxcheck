package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Test представляет информацию о тесте
type Test struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // pending, running, completed, failed
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

// In-memory хранилище для демонстрации
var (
	tests    = make(map[string]*Test)
	results  = make(map[string]*TestResult)
)

func main() {
	r := gin.Default()
	
	// Middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
			"service":   "proxy-test-api",
		})
	})
	
	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/status", getStatus)
		api.GET("/config", getConfig)
		api.POST("/tests", startTest)
		api.GET("/tests/:id", getTestStatus)
		api.DELETE("/tests/:id", stopTest)
		api.GET("/results/:id", getResults)
		api.GET("/results/:id/working", getWorkingProxies)
		api.POST("/results/:id/export", exportResults)
	}
	
	// Serve OpenAPI docs
	r.Static("/docs", "./docs")
	
	log.Println("🚀 Proxy Test API server starting on :8080")
	log.Println("📚 API Documentation: http://localhost:8080/docs")
	log.Println("🔍 Health check: http://localhost:8080/health")
	
	log.Fatal(r.Run(":8080"))
}

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// getStatus возвращает статус системы
func getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"system": "proxy-test-api",
		"status": "running",
		"uptime": "0s", // TODO: calculate actual uptime
		"active_tests": len(tests),
		"total_results": len(results),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// getConfig возвращает текущую конфигурацию
func getConfig(c *gin.Context) {
	config := gin.H{
		"xray": gin.H{
			"start_port": 10000,
			"log_level": "error",
		},
		"proxy": gin.H{
			"check_method": "ip",
			"ip_check_url": "https://api.ipify.org?format=text",
			"timeout": 30,
			"simulate_latency": false,
		},
		"api": gin.H{
			"port": 8080,
			"max_concurrent_tests": 5,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"config": config,
		"last_updated": time.Now().Format(time.RFC3339),
	})
}

// startTest запускает новый тест
func startTest(c *gin.Context) {
	var request struct {
		Name       string `json:"name"`
		ProxyCount int    `json:"proxy_count"`
		Timeout    int    `json:"timeout"`
	}
	
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}
	
	// Валидация
	if request.ProxyCount <= 0 || request.ProxyCount > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "proxy_count must be between 1 and 100",
		})
		return
	}
	
	if request.Timeout <= 0 || request.Timeout > 300 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "timeout must be between 1 and 300 seconds",
		})
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
	
	tests[testID] = test
	
	// Запускаем тест в горутине (заглушка для демонстрации)
	go runTest(testID, request.ProxyCount)
	
	c.JSON(http.StatusOK, gin.H{
		"test_id": testID,
		"status":  "started",
		"message": "Test started successfully",
		"started_at": test.StartedAt.Format(time.RFC3339),
	})
}

// getTestStatus возвращает статус теста
func getTestStatus(c *gin.Context) {
	testID := c.Param("id")
	
	test, exists := tests[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Test not found",
			"test_id": testID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"test_id": test.ID,
		"name":    test.Name,
		"status":  test.Status,
		"proxy_count": test.ProxyCount,
		"started_at": test.StartedAt.Format(time.RFC3339),
		"completed_at": "",
	})
}

// stopTest останавливает тест
func stopTest(c *gin.Context) {
	testID := c.Param("id")
	
	test, exists := tests[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Test not found",
			"test_id": testID,
		})
		return
	}
	
	test.Status = "stopped"
	test.CompletedAt = time.Now()
	
	c.JSON(http.StatusOK, gin.H{
		"test_id": test.ID,
		"status":  "stopped",
		"message": "Test stopped successfully",
		"stopped_at": test.CompletedAt.Format(time.RFC3339),
	})
}

// getResults возвращает результаты теста
func getResults(c *gin.Context) {
	testID := c.Param("id")
	
	result, exists := results[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Results not found",
			"test_id": testID,
		})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// getWorkingProxies возвращает список рабочих прокси
func getWorkingProxies(c *gin.Context) {
	testID := c.Param("id")
	
	result, exists := results[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Results not found",
			"test_id": testID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"test_id": testID,
		"working_proxies": result.WorkingProxies,
		"count": len(result.WorkingProxies),
	})
}

// exportResults экспортирует результаты в файл
func exportResults(c *gin.Context) {
	testID := c.Param("id")
	
	result, exists := results[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Results not found",
			"test_id": testID,
		})
		return
	}
	
	// Генерируем текстовый файл с результатами
	content := "# Proxy Test Results\n"
	content += "# Test ID: " + testID + "\n"
	content += "# Date: " + time.Now().Format("2006-01-02 15:04:05") + "\n"
	content += "# Total Proxies: " + string(rune(result.TotalProxies)) + "\n"
	content += "# Successful: " + string(rune(result.Successful)) + "\n"
	content += "# Success Rate: " + string(rune(result.SuccessRate)) + "%\n\n"
	
	for i, proxy := range result.WorkingProxies {
		content += fmt.Sprintf("%d. %s | %s:%d | %s | %s\n", 
			i+1, proxy.Name, proxy.Server, proxy.Port, proxy.Protocol, proxy.Latency)
	}
	
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", "attachment; filename=proxies_"+testID+".txt")
	c.String(http.StatusOK, content)
}

// runTest запускает тест (заглушка для демонстрации)
func runTest(testID string, proxyCount int) {
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
	
	results[testID] = result
	
	// Обновляем статус теста
	if test, exists := tests[testID]; exists {
		test.Status = "completed"
		test.CompletedAt = time.Now()
	}
}

// generateTestID генерирует уникальный ID теста
func generateTestID() string {
	return "test_" + time.Now().Format("20060102150405")
}

