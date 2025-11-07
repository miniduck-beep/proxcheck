package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

// Test представляет информацию о тесте
type Test struct {
	ID          string
	Name        string
	Status      string // pending, running, completed, failed
	ProxyCount  int
	StartedAt   time.Time
	CompletedAt time.Time
}

// TestResult представляет результаты теста
type TestResult struct {
	TestID         string
	TotalProxies   int
	Successful     int
	Failed         int
	SuccessRate    float64
	AverageLatency string
	WorkingProxies []ProxyInfo
}

// ProxyInfo представляет информацию о прокси
type ProxyInfo struct {
	Name     string
	Protocol string
	Server   string
	Port     int
	Latency  string
	Rank     int
}

// VLESSConfig содержит параметры для VLESS прокси
type VLESSConfig struct {
	UUID        string
	Address     string
	Port        int
	Flow        string
	Encryption  string
	Network     string
	TLS         bool
	SNI         string
	Fingerprint string
	Path        string
	ServiceName string // Для gRPC
	Mode        string // Для gRPC (multi или gun)
	Host        string // Для WebSocket
	Type        string // Для WebSocket (none, http, ws)
	Headers     map[string]string
	Fragment    string // Исходный фрагмент URL
}

// TestRequest определяет структуру для входящих запросов на тест
type TestRequest struct {
	Name       string            `json:"name"`
	ProxyCount int               `json:"proxy_count"`
	Timeout    int               `json:"timeout"`
	Configs    []json.RawMessage `json:"configs"`
}

// In-memory хранилище для демонстрации
var (
	tests   = make(map[string]*Test)
	results = make(map[string]*TestResult)
	mu      sync.Mutex
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
			"version":   "1.1.0", // Incremented version
			"service":   "proxy-test-api",
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/status", getStatus)
		api.POST("/tests", startTest)
		api.GET("/tests/:id", getTestStatus)
		api.GET("/results/:id", getResults)
	}

	log.Println("🚀 Proxy Test API server starting on :8080")
	log.Fatal(r.Run(":8080"))
}

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// getStatus возвращает статус системы
func getStatus(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"system":        "proxy-test-api",
		"status":        "running",
		"active_tests":  len(tests),
		"total_results": len(results),
		"timestamp":     time.Now().Format(time.RFC3339),
	})
}

// startTest запускает новый тест
func startTest(c *gin.Context) {
	var request TestRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if len(request.Configs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "configs array cannot be empty"})
		return
	}

	if request.ProxyCount <= 0 || request.ProxyCount > len(request.Configs) {
		request.ProxyCount = len(request.Configs)
	}

	if request.Timeout <= 0 {
		request.Timeout = 30 // default timeout
	}

	testID := generateTestID()
	test := &Test{
		ID:         testID,
		Name:       request.Name,
		Status:     "running",
		ProxyCount: request.ProxyCount,
		StartedAt:  time.Now(),
	}

	mu.Lock()
	tests[testID] = test
	mu.Unlock()

	go runTest(testID, request.Configs, request.ProxyCount, request.Timeout)

	c.JSON(http.StatusOK, gin.H{
		"test_id":    testID,
		"status":     "started",
		"message":    "Test started successfully",
		"started_at": test.StartedAt.Format(time.RFC3339),
	})
}

// getTestStatus возвращает статус теста
func getTestStatus(c *gin.Context) {
	testID := c.Param("id")
	mu.Lock()
	defer mu.Unlock()
	test, exists := tests[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test not found", "test_id": testID})
		return
	}
	c.JSON(http.StatusOK, test)
}

// getResults возвращает результаты теста
func getResults(c *gin.Context) {
	testID := c.Param("id")
	mu.Lock()
	defer mu.Unlock()
	result, exists := results[testID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Results not found", "test_id": testID})
		return
	}
	c.JSON(http.StatusOK, result)
}

// runTest запускает тест
func runTest(testID string, configs []json.RawMessage, proxyCount int, timeout int) {
	log.Printf("Starting test %s with %d proxies", testID, proxyCount)

	var (
		workingProxies []ProxyInfo
		successful     int
		failed         int
		totalLatency   time.Duration
		wg             sync.WaitGroup
		proxyResults   = make(chan ProxyInfo, len(configs))
		muResults      sync.Mutex
	)

	for i, rawConfig := range configs {
		if i >= proxyCount {
			break
		}
		wg.Add(1)
		go func(index int, config json.RawMessage) {
			defer wg.Done()

			var proxyURL string
			if err := json.Unmarshal(config, &proxyURL); err != nil {
				log.Printf("Error unmarshaling config for test %s: %v", testID, err)
				muResults.Lock()
				failed++
				muResults.Unlock()
				return
			}

			vlessConfig, err := ParseVLESSConfig(proxyURL)
			if err != nil {
				log.Printf("Proxy %d (%s) failed to parse: %v", index+1, proxyURL, err)
				muResults.Lock()
				failed++
				muResults.Unlock()
				return
			}

			latency, err := testProxy(proxyURL, time.Duration(timeout)*time.Second)
			if err != nil {
				log.Printf("Proxy %d (%s) failed: %v", index+1, proxyURL, err)
				muResults.Lock()
				failed++
				muResults.Unlock()
				return
			}

			log.Printf("Proxy %d (%s) successful, latency: %s", index+1, proxyURL, latency)
			proxyInfo := ProxyInfo{
				Name:     vlessConfig.Fragment,
				Protocol: "vless",
				Server:   vlessConfig.Address,
				Port:     vlessConfig.Port,
				Latency:  latency.String(),
				Rank:     index + 1,
			}
			proxyResults <- proxyInfo
			muResults.Lock()
			successful++
			totalLatency += latency
			muResults.Unlock()
		}(i, rawConfig)
	}

	wg.Wait()
	close(proxyResults)

	for res := range proxyResults {
		workingProxies = append(workingProxies, res)
	}

	averageLatency := "N/A"
	if successful > 0 {
		averageLatency = (totalLatency / time.Duration(successful)).String()
	}

	successRate := 0.0
	if proxyCount > 0 {
		successRate = float64(successful) / float64(proxyCount) * 100
	}

	mu.Lock()
	results[testID] = &TestResult{
		TestID:         testID,
		TotalProxies:   proxyCount,
		Successful:     successful,
		Failed:         proxyCount - successful,
		SuccessRate:    successRate,
		AverageLatency: averageLatency,
		WorkingProxies: workingProxies,
	}
	if test, exists := tests[testID]; exists {
		test.Status = "completed"
		test.CompletedAt = time.Now()
	}
	mu.Unlock()

	log.Printf("Test %s completed. Successful: %d, Failed: %d", testID, successful, proxyCount-successful)
}

// testProxy тестирует один прокси
func testProxy(proxyURL string, timeout time.Duration) (time.Duration, error) {
	xrayConfig, err := GenerateXrayConfig(proxyURL)
	if err != nil {
		return 0, fmt.Errorf("failed to generate Xray config: %w", err)
	}

	configFile, err := os.CreateTemp("", "xray-config-*.json")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp config file: %w", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(xrayConfig); err != nil {
		return 0, fmt.Errorf("failed to write Xray config: %w", err)
	}
	configFile.Close()

	cmd := exec.Command("xray", "-c", configFile.Name())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start Xray: %w", err)
	}
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill Xray process: %v", err)
		}
		cmd.Wait()
	}()

	time.Sleep(2 * time.Second) // Даем Xray время на запуск

	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "socks5",
				Host:   "127.0.0.1:10808", // Локальный порт Xray из шаблона
			}),
		},
	}

	start := time.Now()
	resp, err := client.Get("http://www.google.com/generate_204") // Более надежный URL для проверки
	if err != nil {
		return 0, fmt.Errorf("failed to connect via proxy, Xray stderr: %s, error: %w", stderr.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return time.Since(start), nil
}

// GenerateXrayConfig генерирует конфигурацию Xray для VLESS прокси
func GenerateXrayConfig(vlessURL string) (string, error) {
	config, err := ParseVLESSConfig(vlessURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse VLESS URL: %w", err)
	}

	tmpl, err := template.New("xrayConfig").Parse(xrayTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Xray template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute Xray template: %w", err)
	}

	return buf.String(), nil
}

// ParseVLESSConfig парсит VLESS URL и возвращает VLESSConfig
func ParseVLESSConfig(vlessURL string) (*VLESSConfig, error) {
	u, err := url.Parse(vlessURL)
	if err != nil {
		return nil, fmt.Errorf("invalid VLESS URL: %w", err)
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	uuid := u.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("VLESS UUID not found in URL")
	}

	addressParts := strings.SplitN(u.Host, ":", 2)
	if len(addressParts) != 2 {
		return nil, fmt.Errorf("invalid VLESS host:port format")
	}
	address := addressParts[0]
	port, err := strconv.Atoi(addressParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	query := u.Query()
	config := &VLESSConfig{
		UUID:        uuid,
		Address:     address,
		Port:        port,
		Fragment:    u.Fragment,
		Encryption:  "none",
		Flow:        query.Get("flow"),
		Network:     query.Get("type"),
		TLS:         query.Get("security") == "tls",
		SNI:         query.Get("sni"),
		Fingerprint: query.Get("fp"),
		Path:        query.Get("path"),
		Host:        query.Get("host"),
		ServiceName: query.Get("serviceName"),
		Mode:        query.Get("mode"),
	}
	if config.SNI == "" {
		config.SNI = config.Host
	}

	return config, nil
}

// xrayTemplate - шаблон конфигурации Xray для VLESS
const xrayTemplate = `{
    "log": {
        "loglevel": "warning"
    },
    "inbounds": [
        {
            "port": 10808,
            "protocol": "socks",
            "settings": {
                "auth": "noauth",
                "udp": true
            }
        }
    ],
    "outbounds": [
        {
            "protocol": "vless",
            "settings": {
                "vnext": [
                    {
                        "address": "{{.Address}}",
                        "port": {{.Port}},
                        "users": [
                            {
                                "id": "{{.UUID}}",
                                "encryption": "none",
                                "flow": "{{.Flow}}"
                            }
                        ]
                    }
                ]
            },
            "streamSettings": {
                "network": "{{.Network}}",
                "security": "{{if .TLS}}tls{{else}}none{{end}}",
                "tlsSettings": {
                    "serverName": "{{.SNI}}",
                    "fingerprint": "{{.Fingerprint}}"
                },
                "wsSettings": {
                    "path": "{{.Path}}",
                    "headers": {
                        "Host": "{{.Host}}"
                    }
                },
                "grpcSettings": {
                    "serviceName": "{{.ServiceName}}",
                    "multiMode": {{if eq .Mode "multi"}}true{{else}}false{{end}}
                }
            }
        }
    ]
}`

// generateTestID генерирует уникальный ID теста
func generateTestID() string {
	return "test_" + time.Now().Format("20060102150405")
}