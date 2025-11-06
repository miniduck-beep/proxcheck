package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"projectx/proxytestlib/checker"
	"projectx/proxytestlib/config"
	"projectx/proxytestlib/models"
	"projectx/proxytestlib/runner"
	"projectx/proxytestlib/xray"
)

// Структура для парсинга JSON конфигураций
type RawConfig struct {
	Type      string      `json:"type"`
	Server    string      `json:"server"`
	Port      int         `json:"port"`
	UUID      string      `json:"uuid"`
	AlterId   int         `json:"alterId"`
	Cipher    string      `json:"cipher"`
	Network   string      `json:"network"`
	TLS       interface{} `json:"tls"` // Может быть строкой или булевым значением
	SNI       string      `json:"sni"`
	Path      string      `json:"path"`
	Host      string      `json:"host"`
	Remarks   string      `json:"remarks"`
	ALPN      string      `json:"alpn"`
	Fingerprint string   `json:"fingerprint"`
	Password  string      `json:"password"`
	Method    string      `json:"method"`
}

type DeduplicatedFile struct {
	Configs []RawConfig `json:"configs"`
}

func convertToProxyConfig(raw RawConfig) *models.ProxyConfig {
	config := &models.ProxyConfig{
		Protocol: raw.Type,
		Server:   raw.Server,
		Port:     raw.Port,
		Name:     raw.Remarks,
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

func main() {
	// Читаем файл с конфигурациями
	filePath := "/Users/t/zapret/test_xray_finish/deduplicated.json"
	
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	// Читаем весь файл как структурированный JSON
	var data struct {
		Configs []RawConfig `json:"configs"`
	}
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	// Берем первые 100 конфигураций
	var proxyConfigs []*models.ProxyConfig
	count := 0
	
	for _, rawConfig := range data.Configs {
		if count >= 100 {
			break
		}
		
		// Пропускаем некорректные конфигурации
		if rawConfig.Server == "" || rawConfig.Port == 0 {
			continue
		}
		
		proxyConfig := convertToProxyConfig(rawConfig)
		proxyConfigs = append(proxyConfigs, proxyConfig)
		count++
		
		if count%10 == 0 {
			log.Printf("Processed %d configurations...", count)
		}
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

	// Подготавливаем конфигурации прокси
	xray.PrepareProxyConfigs(proxyConfigs)

	// Генерируем и сохраняем конфигурацию Xray
	configFile := "xray_config_deduplicated.json"
	if err := xray.GenerateAndSaveConfig(proxyConfigs, config.CLIConfig.Xray.StartPort, configFile, config.CLIConfig.Xray.LogLevel); err != nil {
		log.Fatalf("Error generating Xray config: %v", err)
	}

	// Инициализируем и запускаем Xray
	xrayRunner := runner.NewXrayRunner(configFile)
	if err := xrayRunner.Start(); err != nil {
		log.Fatalf("Error starting Xray runner: %v", err)
	}
	defer xrayRunner.Stop()

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
	for i, proxy := range proxyConfigs {
		status, latency, err := proxyChecker.GetProxyStatus(proxy.Name)
		if err != nil {
			fmt.Printf("%2d. %-40s: ERROR - %v\n", i+1, proxy.Name, err)
		} else {
			statusStr := "FAIL"
			if status {
				statusStr = "OK"
			}
			fmt.Printf("%2d. %-40s: %-4s (latency: %v)\n", i+1, proxy.Name, statusStr, latency)
		}
	}
}