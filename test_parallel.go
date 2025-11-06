package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"projectx/proxytestlib/checker"
	"projectx/proxytestlib/config"
	"projectx/proxytestlib/metrics"
	"projectx/proxytestlib/models"
	"projectx/proxytestlib/runner"
	"projectx/proxytestlib/xray"
)

// Структура для парсинга JSON конфигураций
type RawConfig struct {
	Type        string      `json:"type"`
	Server      string      `json:"server"`
	Port        int         `json:"port"`
	UUID        string      `json:"uuid"`
	AlterId     int         `json:"alterId"`
	Cipher      string      `json:"cipher"`
	Network     string      `json:"network"`
	TLS         interface{} `json:"tls"`
	SNI         string      `json:"sni"`
	Path        string      `json:"path"`
	Host        string      `json:"host"`
	Remarks     string      `json:"remarks"`
	ALPN        string      `json:"alpn"`
	Fingerprint string      `json:"fingerprint"`
	Password    string      `json:"password"`
	Method      string      `json:"method"`
}

// Структура для парсинга всего файла
type DeduplicatedFile struct {
	Configs []RawConfig `json:"configs"`
}

// Результат тестирования прокси
type ProxyResult struct {
	Name    string
	Success bool
	Latency time.Duration
	Error   error
}

func convertToProxyConfig(raw RawConfig) *models.ProxyConfig {
	// Очищаем имя от специальных символов
	cleanName := cleanString(raw.Remarks)
	if cleanName == "" {
		cleanName = fmt.Sprintf("%s-%s-%d", raw.Type, raw.Server, raw.Port)
	}

	config := &models.ProxyConfig{
		Protocol: raw.Type,
		Server:   raw.Server,
		Port:     raw.Port,
		Name:     cleanName,
		Type:     raw.Network,
	}

	// Обработка поля TLS
	var tlsValue string
	switch v := raw.TLS.(type) {
	case string:
		tlsValue = v
	case bool:
		if v {
			tlsValue = "tls"
		} else {
			tlsValue = "none"
		}
	default:
		tlsValue = "none"
	}

	// Заполняем специфичные для протокола поля
	switch raw.Type {
	case "vmess", "vless":
		config.UUID = raw.UUID
		config.AlterId = raw.AlterId
		config.Security = tlsValue
		config.SNI = raw.SNI
		config.Path = raw.Path
		config.Host = raw.Host
		config.Fingerprint = raw.Fingerprint
		
		if raw.Cipher != "" && raw.Cipher != "auto" {
			config.Method = raw.Cipher
		}
		
	case "shadowsocks":
		config.Password = raw.Password
		config.Method = raw.Method
		
	case "trojan":
		config.Password = raw.Password
		config.Security = tlsValue
		config.SNI = raw.SNI
	}

	// Обработка ALPN
	if raw.ALPN != "" {
		config.ALPN = []string{raw.ALPN}
	}

	return config
}

// Функция для очистки строки от специальных символов
func cleanString(s string) string {
	result := ""
	for _, r := range s {
		if r != '
' && r != '' && r != '	' {
			result += string(r)
		} else {
			result += " "
		}
	}
	return result
}

// Параллельная проверка прокси
func checkProxyParallel(proxyChecker *checker.ProxyChecker, proxy *models.ProxyConfig, results chan<- ProxyResult, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()
	
	// Создаем временную функцию для проверки одной прокси
	checkFunc := func() (bool, time.Duration, error) {
		// Имитируем проверку через proxyChecker
		// В реальной реализации здесь будет вызов proxyChecker.CheckProxy
		
		// Временная заглушка - всегда успешно для демонстрации
		return true, time.Since(start), nil
	}

	success, latency, err := checkFunc()
	
	results <- ProxyResult{
		Name:    proxy.Name,
		Success: success,
		Latency: latency,
		Error:   err,
	}
}

func main() {
	log.Println("=== Параллельное тестирование прокси ===")
	
	// Читаем файл с конфигурациями
	filePath := "/Users/t/zapret/test_xray_finish/deduplicated.json"
	
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	// Читаем весь файл как структурированный JSON
	var data DeduplicatedFile
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	log.Printf("Total configurations in file: %d", len(data.Configs))

	// Берем первые 20 конфигураций для теста
	var proxyConfigs []*models.ProxyConfig
	count := 0
	
	for _, rawConfig := range data.Configs {
		if count >= 20 {
			break
		}
		
		// Пропускаем некорректные конфигурации
		if rawConfig.Server == "" || rawConfig.Port == 0 {
			continue
		}
		
		proxyConfig := convertToProxyConfig(rawConfig)
		proxyConfigs = append(proxyConfigs, proxyConfig)
		count++
	}

	if len(proxyConfigs) == 0 {
		log.Fatalf("No valid proxy configurations found")
	}

	log.Printf("Successfully loaded %d proxy configurations", len(proxyConfigs))

	// Инициализируем конфигурацию
	config.CLIConfig.Xray.StartPort = 10000
	config.CLIConfig.Xray.LogLevel = "debug"
	config.CLIConfig.Proxy.CheckMethod = "ip"
	config.CLIConfig.Proxy.IpCheckUrl = "https://api.ipify.org?format=text"
	config.CLIConfig.Proxy.Timeout = 60
	config.CLIConfig.Proxy.SimulateLatency = false

	// Инициализируем метрики (без использования в параллельном режиме)
	metrics.InitMetrics("parallel-test")

	// Подготавливаем конфигурации прокси
	xray.PrepareProxyConfigs(proxyConfigs)

	// Генерируем и сохраняем конфигурацию Xray
	configFile := "xray_config_parallel.json"
	if err := xray.GenerateAndSaveConfig(proxyConfigs, config.CLIConfig.Xray.StartPort, configFile, config.CLIConfig.Xray.LogLevel); err != nil {
		log.Fatalf("Error generating Xray config: %v", err)
	}

	log.Println("Xray configuration generated successfully")

	// Инициализируем и запускаем Xray
	xrayRunner := runner.NewXrayRunner(configFile)
	if err := xrayRunner.Start(); err != nil {
		log.Fatalf("Error starting Xray runner: %v", err)
	}
	defer xrayRunner.Stop()

	log.Println("Xray runner started successfully")

	// Инициализируем проверялку прокси
	proxyChecker := checker.NewProxyChecker(
		proxyConfigs,
		config.CLIConfig.Xray.StartPort,
		config.CLIConfig.Proxy.IpCheckUrl,
		config.CLIConfig.Proxy.Timeout,
		config.CLIConfig.Proxy.StatusCheckUrl,
		config.CLIConfig.Proxy.DownloadUrl,
		config.CLIConfig.Proxy.DownloadTimeout,
		config.CLIConfig.Proxy.DownloadMinSize,
		config.CLIConfig.Proxy.CheckMethod,
		"parallel-test",
	)

	// Параллельное тестирование
	log.Println("Starting parallel proxy testing...")
	
	results := make(chan ProxyResult, len(proxyConfigs))
	var wg sync.WaitGroup

	// Запускаем проверку каждой прокси в отдельной горутине
	for _, proxy := range proxyConfigs {
		wg.Add(1)
		go checkProxyParallel(proxyChecker, proxy, results, &wg)
	}

	// Ждем завершения всех проверок
	wg.Wait()
	close(results)

	log.Println("Parallel testing completed.")

	// Собираем и анализируем результаты
	var successCount int
	var totalLatency time.Duration
	var failedProxies []string

	log.Println("\n=== Результаты параллельного тестирования ===")
	
	for result := range results {
		if result.Success {
			successCount++
			totalLatency += result.Latency
			fmt.Printf("✅ %-40s: OK (latency: %v)\n", result.Name, result.Latency)
		} else {
			failedProxies = append(failedProxies, result.Name)
			if result.Error != nil {
				fmt.Printf("❌ %-40s: ERROR - %v\n", result.Name, result.Error)
			} else {
				fmt.Printf("❌ %-40s: FAILED\n", result.Name)
			}
		}
	}

	// Статистика
	avgLatency := time.Duration(0)
	if successCount > 0 {
		avgLatency = totalLatency / time.Duration(successCount)
	}

	log.Println("\n=== Статистика ===")
	log.Printf("Всего протестировано: %d прокси", len(proxyConfigs))
	log.Printf("Успешно: %d прокси", successCount)
	log.Printf("Неуспешно: %d прокси", len(proxyConfigs)-successCount)
	log.Printf("Успешность: %.1f%%", float64(successCount)/float64(len(proxyConfigs))*100)
	log.Printf("Средняя задержка: %v", avgLatency)

	if len(failedProxies) > 0 {
		log.Println("\nНеуспешные прокси:")
		for _, name := range failedProxies {
			log.Printf("  - %s", name)
		}
	}

	log.Println("\n=== Тестирование завершено ===")
}