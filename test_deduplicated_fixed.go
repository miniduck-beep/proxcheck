package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	TLS         interface{} `json:"tls"` // Может быть строкой или булевым значением
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

func convertToProxyConfig(raw RawConfig) *models.ProxyConfig {
	// Очищаем имя от специальных символов, которые могут сломать JSON
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

	// Обработка поля TLS (может быть строкой или булевым значением)
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
	// Удаляем символы новой строки и табуляции
	result := ""
	for _, r := range s {
		if r != '\n' && r != '\r' && r != '\t' {
			result += string(r)
		} else {
			result += " "
		}
	}
	
	// Удаляем лишние пробелы
	return result
}

func main() {
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

	// Берем первые 10 конфигураций для теста
	var proxyConfigs []*models.ProxyConfig
	count := 0
	
	for _, rawConfig := range data.Configs {
		if count >= 10 {
			break
		}
		
		// Пропускаем некорректные конфигурации
		if rawConfig.Server == "" || rawConfig.Port == 0 {
			log.Printf("Skipping invalid config: %s", rawConfig.Remarks)
			continue
		}
		
		proxyConfig := convertToProxyConfig(rawConfig)
		proxyConfigs = append(proxyConfigs, proxyConfig)
		count++
		
		log.Printf("Loaded config %d: %s (%s://%s:%d)", 
			count, rawConfig.Remarks, rawConfig.Type, rawConfig.Server, rawConfig.Port)
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

	// Инициализируем метрики
	metrics.InitMetrics("deduplicated-test")

	// Подготавливаем конфигурации прокси
	xray.PrepareProxyConfigs(proxyConfigs)

	// Генерируем и сохраняем конфигурацию Xray
	configFile := "xray_config_deduplicated.json"
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
		"", // instance пустой
	)

	// Выполняем проверку
	log.Println("Starting proxy check for deduplicated configurations...")
	proxyChecker.CheckAllProxies()
	log.Println("Proxy check completed.")

	// Выводим статистику
	log.Println("\n=== Test Results ===")
	successCount := 0
	for i, proxy := range proxyConfigs {
		status, latency, err := proxyChecker.GetProxyStatus(proxy.Name)
		if err != nil {
			fmt.Printf("%2d. %-40s: ERROR - %v\n", i+1, proxy.Name, err)
		} else {
			statusStr := "FAIL"
			if status {
				statusStr = "OK"
				successCount++
			}
			fmt.Printf("%2d. %-40s: %-4s (latency: %v)\n", i+1, proxy.Name, statusStr, latency)
		}
	}

	log.Printf("\nSuccess rate: %d/%d (%.1f%%)", successCount, len(proxyConfigs), 
		float64(successCount)/float64(len(proxyConfigs))*100)
}