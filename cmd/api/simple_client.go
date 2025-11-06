package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient представляет клиент для работы с API
type APIClient struct {
	BaseURL string
	Client  *http.Client
}

// NewAPIClient создает новый клиент
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Health проверяет статус API
func (c *APIClient) Health() error {
	resp, err := c.Client.Get(c.BaseURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}
	
	fmt.Printf("✅ Health check: %s\n", result["status"])
	return nil
}

// GetStatus получает статус системы
func (c *APIClient) GetStatus() (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// GetConfig получает конфигурацию
func (c *APIClient) GetConfig() (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/config")
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get config failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// StartTest запускает новый тест
func (c *APIClient) StartTest(name string, proxyCount int) (string, error) {
	request := map[string]interface{}{
		"name":        name,
		"proxy_count": proxyCount,
		"timeout":     30,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}
	
	resp, err := c.Client.Post(c.BaseURL+"/api/v1/tests", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to start test: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("start test failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}
	
	testID, ok := result["test_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response: test_id not found")
	}
	
	return testID, nil
}

// GetTestStatus получает статус теста
func (c *APIClient) GetTestStatus(testID string) (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/tests/" + testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test status: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get test status failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// GetResults получает результаты теста
func (c *APIClient) GetResults(testID string) (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/results/" + testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get results failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// GetWorkingProxies получает список рабочих прокси
func (c *APIClient) GetWorkingProxies(testID string) (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/results/" + testID + "/working")
	if err != nil {
		return nil, fmt.Errorf("failed to get working proxies: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get working proxies failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// Example использования клиента
func main() {
	client := NewAPIClient("http://localhost:8080")
	
	fmt.Println("🚀 Proxy Test API Client")
	fmt.Println(strings.Repeat("=", 40))
	
	// Проверяем здоровье API
	fmt.Println("\n🔍 Checking API health...")
	if err := client.Health(); err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		fmt.Println("💡 Make sure the API server is running on localhost:8080")
		return
	}
	
	// Получаем статус системы
	fmt.Println("\n📊 Getting system status...")
	status, err := client.GetStatus()
	if err != nil {
		fmt.Printf("❌ Failed to get status: %v\n", err)
	} else {
		fmt.Printf("✅ System: %s\n", status["system"])
		fmt.Printf("✅ Status: %s\n", status["status"])
		fmt.Printf("✅ Active tests: %v\n", status["active_tests"])
		fmt.Printf("✅ Total results: %v\n", status["total_results"])
	}
	
	// Получаем конфигурацию
	fmt.Println("\n⚙️ Getting configuration...")
	config, err := client.GetConfig()
	if err != nil {
		fmt.Printf("❌ Failed to get config: %v\n", err)
	} else {
		fmt.Printf("✅ Configuration loaded\n")
	}
	
	// Запускаем тест
	fmt.Println("\n🚀 Starting new test...")
	testID, err := client.StartTest("api-demo-test", 10)
	if err != nil {
		fmt.Printf("❌ Failed to start test: %v\n", err)
		return
	}
	fmt.Printf("✅ Test started with ID: %s\n", testID)
	
	// Мониторим статус теста
	fmt.Println("\n📊 Monitoring test status...")
	for i := 0; i < 10; i++ {
		status, err := client.GetTestStatus(testID)
		if err != nil {
			fmt.Printf("❌ Failed to get status: %v\n", err)
			break
		}
		
		fmt.Printf("⏳ Status: %s, Progress: checking...\n", status["status"])
		
		if status["status"] == "completed" {
			fmt.Println("✅ Test completed!")
			break
		}
		
		time.Sleep(2 * time.Second)
	}
	
	// Получаем результаты
	fmt.Println("\n📈 Getting test results...")
	results, err := client.GetResults(testID)
	if err != nil {
		fmt.Printf("❌ Failed to get results: %v\n", err)
		return
	}
	
	fmt.Printf("\n📊 Test Results:\n")
	fmt.Printf("   Total proxies: %v\n", results["total_proxies"])
	fmt.Printf("   Successful: %v\n", results["successful"])
	fmt.Printf("   Failed: %v\n", results["failed"])
	fmt.Printf("   Success rate: %.1f%%\n", results["success_rate"])
	fmt.Printf("   Average latency: %v\n", results["average_latency"])
	
	// Получаем рабочие прокси
	fmt.Println("\n✅ Getting working proxies...")
	working, err := client.GetWorkingProxies(testID)
	if err != nil {
		fmt.Printf("❌ Failed to get working proxies: %v\n", err)
		return
	}
	
	if proxies, ok := working["working_proxies"].([]interface{}); ok {
		fmt.Printf("\n🏆 Working Proxies (%d):\n", len(proxies))
		for i, proxy := range proxies {
			if p, ok := proxy.(map[string]interface{}); ok {
				fmt.Printf("   %d. %s (%s) - %s\n", 
					i+1, p["name"], p["protocol"], p["latency"])
			}
		}
	}
	
	fmt.Println("\n🎉 API client demo completed successfully!")
}

// Вспомогательная функция для strings.Repeat
func strings.Repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}