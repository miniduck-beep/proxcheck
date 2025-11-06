package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// APIClient представляет клиент для работы с API
type APIClient struct {
	BaseURL string
	Client  *http.Client
	Verbose bool
}

// NewAPIClient создает новый клиент
func NewAPIClient(baseURL string, verbose bool) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Verbose: verbose,
	}
}

// Health проверяет статус API
func (c *APIClient) Health() (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
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
func (c *APIClient) StartTest(name string, proxyCount int, configFile string, startPort int) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"name":        name,
		"proxy_count": proxyCount,
		"timeout":     30,
	}
	
	if configFile != "" {
		request["config_file"] = configFile
	}
	
	if startPort > 0 {
		request["start_port"] = startPort
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	
	if c.Verbose {
		fmt.Printf("📤 Sending request: %s\n", string(jsonData))
	}
	
	resp, err := c.Client.Post(c.BaseURL+"/api/v1/tests", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to start test: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("start test failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
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

// ListTests получает список всех тестов
func (c *APIClient) ListTests() (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/tests/")
	if err != nil {
		return nil, fmt.Errorf("failed to list tests: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tests failed with status: %d", resp.StatusCode)
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

// ExportResults экспортирует результаты в файл
func (c *APIClient) ExportResults(testID string) (map[string]interface{}, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/v1/results/" + testID + "/export")
	if err != nil {
		return nil, fmt.Errorf("failed to export results: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("export results failed with status: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	return result, nil
}

// printHeader выводит заголовок
func printHeader(title string) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n", strings.Repeat("=", 60))
}

// printSection выводит секцию
func printSection(title string) {
	fmt.Printf("\n%s\n", strings.Repeat("-", 40))
	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n", strings.Repeat("-", 40))
}

// printSuccess выводит успешное сообщение
func printSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// printError выводит сообщение об ошибке
func printError(message string) {
	fmt.Printf("❌ %s\n", message)
}

// printInfo выводит информационное сообщение
func printInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// printWarning выводит предупреждение
func printWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// main функция клиента
func main() {
	// Парсим аргументы командной строки
	port := flag.Int("port", 9090, "API server port")
	host := flag.String("host", "localhost", "API server host")
	verbose := flag.Bool("verbose", false, "Verbose output")
	action := flag.String("action", "demo", "Action to perform: health, status, config, list, test, results, export")
	testName := flag.String("name", "", "Test name")
	proxyCount := flag.Int("count", 20, "Number of proxies to test")
	configFile := flag.String("config", "", "Config file path")
	startPort := flag.Int("start-port", 20000, "Start port for Xray")
	testID := flag.String("test-id", "", "Test ID for results/export")
	
	flag.Parse()
	
	baseURL := fmt.Sprintf("http://%s:%d", *host, *port)
	client := NewAPIClient(baseURL, *verbose)
	
	printHeader("🚀 Proxy Test API Client")
	fmt.Printf("Server: %s\n", baseURL)
	fmt.Printf("Action: %s\n", *action)
	
	switch *action {
	case "health":
		checkHealth(client)
	case "status":
		getStatus(client)
	case "config":
		getConfig(client)
	case "list":
		listTests(client)
	case "test":
		runTest(client, *testName, *proxyCount, *configFile, *startPort)
	case "results":
		getResults(client, *testID)
	case "export":
		exportResults(client, *testID)
	case "demo":
		runDemo(client, *testName, *proxyCount, *configFile, *startPort)
	default:
		printError("Unknown action: " + *action)
		printInfo("Available actions: health, status, config, list, test, results, export, demo")
	}
}

// checkHealth проверяет здоровье API
func checkHealth(client *APIClient) {
	printSection("🔍 Health Check")
	
	health, err := client.Health()
	if err != nil {
		printError(fmt.Sprintf("Health check failed: %v", err))
		return
	}
	
	printSuccess("API is healthy")
	fmt.Printf("Status: %s\n", health["status"])
	fmt.Printf("Version: %s\n", health["version"])
	fmt.Printf("Port: %v\n", health["port"])
	fmt.Printf("Data Directory: %s\n", health["data_dir"])
}

// getStatus получает статус системы
func getStatus(client *APIClient) {
	printSection("📊 System Status")
	
	status, err := client.GetStatus()
	if err != nil {
		printError(fmt.Sprintf("Failed to get status: %v", err))
		return
	}
	
	printSuccess("System status retrieved")
	fmt.Printf("System: %s\n", status["system"])
	fmt.Printf("Status: %s\n", status["status"])
	fmt.Printf("Port: %v\n", status["port"])
	fmt.Printf("Active Tests: %v\n", status["active_tests"])
	fmt.Printf("Total Tests: %v\n", status["total_tests"])
	fmt.Printf("Total Results: %v\n", status["total_results"])
	
	if activeTests, ok := status["active_test_ids"].([]interface{}); ok && len(activeTests) > 0 {
		fmt.Printf("Active Test IDs:\n")
		for _, id := range activeTests {
			fmt.Printf("  - %s\n", id)
		}
	}
}

// getConfig получает конфигурацию
func getConfig(client *APIClient) {
	printSection("⚙️ Configuration")
	
	config, err := client.GetConfig()
	if err != nil {
		printError(fmt.Sprintf("Failed to get config: %v", err))
		return
	}
	
	printSuccess("Configuration loaded")
	
	if configData, ok := config["config"].(map[string]interface{}); ok {
		if files, ok := configData["files"].(map[string]interface{}); ok {
			fmt.Printf("Config File: %s\n", files["config_file"])
			fmt.Printf("Config Exists: %v\n", files["config_exists"])
			fmt.Printf("Config Size: %v bytes\n", files["config_size"])
		}
		
		if api, ok := configData["api"].(map[string]interface{}); ok {
			fmt.Printf("API Port: %v\n", api["port"])
			fmt.Printf("Data Directory: %s\n", api["data_directory"])
		}
		
		if xray, ok := configData["xray"].(map[string]interface{}); ok {
			fmt.Printf("Xray Start Port: %v\n", xray["start_port"])
			fmt.Printf("Xray Log Level: %s\n", xray["log_level"])
		}
	}
}

// listTests получает список тестов
func listTests(client *APIClient) {
	printSection("📋 List Tests")
	
	list, err := client.ListTests()
	if err != nil {
		printError(fmt.Sprintf("Failed to list tests: %v", err))
		return
	}
	
	if tests, ok := list["tests"].([]interface{}); ok {
		fmt.Printf("Total Tests: %v\n", list["count"])
		
		if len(tests) == 0 {
			printInfo("No tests found")
			return
		}
		
		for i, test := range tests {
			if t, ok := test.(map[string]interface{}); ok {
				fmt.Printf("\n%d. %s\n", i+1, t["id"])
				fmt.Printf("   Name: %s\n", t["name"])
				fmt.Printf("   Status: %s\n", t["status"])
				fmt.Printf("   Proxy Count: %v\n", t["proxy_count"])
				fmt.Printf("   Started: %s\n", t["started_at"])
				
				if completed, ok := t["completed_at"]; ok && completed != "" {
					fmt.Printf("   Completed: %s\n", completed)
				}
			}
		}
	}
}

// runTest запускает тест
func runTest(client *APIClient, name string, count int, configFile string, startPort int) {
	printSection("🚀 Start Test")
	
	if name == "" {
		name = "test-" + time.Now().Format("20060102-150405")
	}
	
	if configFile == "" {
		configFile = "/Users/t/zapret/test_xray_finish/deduplicated.json"
	}
	
	fmt.Printf("Test Name: %s\n", name)
	fmt.Printf("Proxy Count: %d\n", count)
	fmt.Printf("Config File: %s\n", configFile)
	fmt.Printf("Start Port: %d\n", startPort)
	
	result, err := client.StartTest(name, count, configFile, startPort)
	if err != nil {
		printError(fmt.Sprintf("Failed to start test: %v", err))
		return
	}
	
	printSuccess("Test started successfully")
	fmt.Printf("Test ID: %s\n", result["test_id"])
	fmt.Printf("Status: %s\n", result["status"])
	fmt.Printf("Config File: %s\n", result["config_file"])
	fmt.Printf("Start Port: %v\n", result["start_port"])
	fmt.Printf("Started At: %s\n", result["started_at"])
	
	printInfo("Test is running in background...")
	printInfo("Use '--action results --test-id " + result["test_id"].(string) + "' to check results")
}

// getResults получает результаты теста
func getResults(client *APIClient, testID string) {
	if testID == "" {
		printError("Test ID is required")
		printInfo("Use --test-id parameter or run --action list to see available tests")
		return
	}
	
	printSection("📈 Get Results")
	fmt.Printf("Test ID: %s\n", testID)
	
	results, err := client.GetResults(testID)
	if err != nil {
		printError(fmt.Sprintf("Failed to get results: %v", err))
		return
	}
	
	printSuccess("Results retrieved")
	fmt.Printf("Test ID: %s\n", results["test_id"])
	fmt.Printf("Total Proxies: %v\n", results["total_proxies"])
	fmt.Printf("Successful: %v\n", results["successful"])
	fmt.Printf("Failed: %v\n", results["failed"])
	fmt.Printf("Success Rate: %.1f%%\n", results["success_rate"])
	fmt.Printf("Average Latency: %s\n", results["average_latency"])
	fmt.Printf("Test Duration: %s\n", results["test_duration"])
	
	// Получаем рабочие прокси
	working, err := client.GetWorkingProxies(testID)
	if err == nil {
		if proxies, ok := working["working_proxies"].([]interface{}); ok && len(proxies) > 0 {
			fmt.Printf("\n🏆 Working Proxies (%d):\n", len(proxies))
			for i, proxy := range proxies {
				if p, ok := proxy.(map[string]interface{}); ok {
					fmt.Printf("   %d. %s (%s) - %s\n", 
						i+1, p["name"], p["protocol"], p["latency"])
				}
			}
		}
	}
}

// exportResults экспортирует результаты
func exportResults(client *APIClient, testID string) {
	if testID == "" {
		printError("Test ID is required")
		return
	}
	
	printSection("💾 Export Results")
	fmt.Printf("Test ID: %s\n", testID)
	
	result, err := client.ExportResults(testID)
	if err != nil {
		printError(fmt.Sprintf("Failed to export results: %v", err))
		return
	}
	
	printSuccess("Results exported successfully")
	fmt.Printf("Export File: %s\n", result["export_file"])
	fmt.Printf("Message: %s\n", result["message"])
}

// runDemo запускает демо-сценарий
func runDemo(client *APIClient, name string, count int, configFile string, startPort int) {
	printSection("🎯 Demo Scenario")
	
	// 1. Health check
	printSection("1. Health Check")
	checkHealth(client)
	
	// 2. System status
	printSection("2. System Status")
	getStatus(client)
	
	// 3. Configuration
	printSection("3. Configuration")
	getConfig(client)
	
	// 4. Start test
	printSection("4. Start Test")
	runTest(client, name, count, configFile, startPort)
	
	// 5. Wait and get results
	printSection("5. Waiting for results...")
	
	// Получаем последний тест
	list, err := client.ListTests()
	if err != nil {
		printError("Failed to get test list: " + err.Error())
		return
	}
	
	var lastTestID string
	if tests, ok := list["tests"].([]interface{}); ok && len(tests) > 0 {
		if lastTest, ok := tests[0].(map[string]interface{}); ok {
			lastTestID = lastTest["id"].(string)
		}
	}
	
	if lastTestID == "" {
		printError("No test found")
		return
	}
	
	// Ждем завершения теста
	for i := 0; i < 10; i++ {
		status, err := client.GetTestStatus(lastTestID)
		if err != nil {
			printError("Failed to get test status: " + err.Error())
			break
		}
		
		fmt.Printf("⏳ Test status: %s\n", status["status"])
		
		if status["status"] == "completed" {
			printSuccess("Test completed!")
			break
		}
		
		time.Sleep(2 * time.Second)
	}
	
	// 6. Get results
	printSection("6. Final Results")
	getResults(client, lastTestID)
	
	// 7. Export results
	printSection("7. Export Results")
	exportResults(client, lastTestID)
	
	printHeader("🎉 Demo Completed!")
}

// Вспомогательная функция для strings.Repeat
func strings.Repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}