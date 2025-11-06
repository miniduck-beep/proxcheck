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
	
	return nil
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

// Example использования клиента
func main() {
	client := NewAPIClient("http://localhost:8080")
	
	// Проверяем здоровье API
	fmt.Println("🔍 Checking API health...")
	if err := client.Health(); err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		return
	}
	fmt.Println("✅ API is healthy")
	
	// Запускаем тест
	fmt.Println("\n🚀 Starting new test...")
	testID, err := client.StartTest("api-test", 10)
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
		
		fmt.Printf("Status: %s, Progress: checking...\n", status["status"])
		
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
	
	fmt.Printf("Results: %+v\n", results)
}