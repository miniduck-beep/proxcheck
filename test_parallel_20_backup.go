package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
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
	Name     string
	Success  bool
	Latency  time.Duration
	Error    error
	Protocol string
	Server   string
	Port     int
}

// Для сортировки по скорости
type ByLatency []ProxyResult

func (a ByLatency) Len() int           { return len(a) }
func (a ByLatency) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLatency) Less(i, j int) bool { return a[i].Latency < a[j].Latency }

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
	// Удаляем символы новой строки и табуляции
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	
	// Удаляем лишние пробелы
	s = strings.TrimSpace(s)
	return s
}

// Параллельная проверка прокси с реальными запросами
func checkProxyParallel(proxyChecker *checker.ProxyChecker, proxy *models.ProxyConfig, port int, results chan<- ProxyResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Выполняем проверку прокси
	proxyChecker.CheckProxy(proxy)
	
	// Получаем статус после проверки
	status, latency, err := proxyChecker.GetProxyStatus(proxy.Name)
	
	results <- ProxyResult{
		Name:     proxy.Name,
		Success:  status,
		Latency:  latency,
		Error:    err,
		Protocol: proxy.Protocol,
		Server:   proxy.Server,
		Port:     proxy.Port,
	}
}

func main() {
	log.Println("=== Параллельное тестирование 20 прокси ===")
	
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

	log.Printf("Всего конфигураций в файле: %d", len(data.Configs))

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

	log.Printf("Успешно загружено %d конфигураций прокси", len(proxyConfigs))

	// Инициализируем конфигурацию
	config.CLIConfig.Xray.StartPort = 10000
	config.CLIConfig.Xray.LogLevel = "error" // Уменьшаем логи для скорости
	config.CLIConfig.Proxy.CheckMethod = "ip"
	config.CLIConfig.Proxy.IpCheckUrl = "https://api.ipify.org?format=text"
	config.CLIConfig.Proxy.Timeout = 30 // Уменьшаем таймаут для скорости
	config.CLIConfig.Proxy.SimulateLatency = false

	// Инициализируем метрики
	metrics.InitMetrics("parallel-20-test")

	// Подготавливаем конфигурации прокси
	xray.PrepareProxyConfigs(proxyConfigs)

	// Генерируем и сохраняем конфигурацию Xray
	configFile := "xray_config_parallel_20.json"
	if err := xray.GenerateAndSaveConfig(proxyConfigs, config.CLIConfig.Xray.StartPort, configFile, config.CLIConfig.Xray.LogLevel); err != nil {
		log.Fatalf("Error generating Xray config: %v", err)
	}

	log.Println("Конфигурация Xray успешно сгенерирована")

	// Инициализируем и запускаем Xray
	xrayRunner := runner.NewXrayRunner(configFile)
	if err := xrayRunner.Start(); err != nil {
		log.Fatalf("Error starting Xray runner: %v", err)
	}
	defer xrayRunner.Stop()

	log.Println("Xray runner успешно запущен")

	// Даем Xray время на запуск
	time.Sleep(2 * time.Second)

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
		"parallel-20-test",
	)

	// Параллельное тестирование с 20 потоками
	log.Println("Запуск параллельного тестирования 20 прокси...")
	
	results := make(chan ProxyResult, len(proxyConfigs))
	var wg sync.WaitGroup

	// Запускаем проверку каждой прокси в отдельной горутине
	for i, proxy := range proxyConfigs {
		port := config.CLIConfig.Xray.StartPort + i
		wg.Add(1)
		go checkProxyParallel(proxyChecker, proxy, port, results, &wg)
	}

	// Ждем завершения всех проверок
	wg.Wait()
	close(results)

	log.Println("Параллельное тестирование завершено.")

	// Собираем результаты
	var allResults []ProxyResult
	var successResults []ProxyResult
	var failedResults []ProxyResult

	for result := range results {
		allResults = append(allResults, result)
		if result.Success {
			successResults = append(successResults, result)
		} else {
			failedResults = append(failedResults, result)
		}
	}

	// Сортируем успешные прокси по скорости
	sort.Sort(ByLatency(successResults))

	// === ВЫВОД РЕЗУЛЬТАТОВ ===

	log.Println("\n" + strings.Repeat("=", 80))
	log.Println("=== РЕЗУЛЬТАТЫ ТЕСТИРОВАНИЯ 20 ПРОКСИ ===")
	log.Println(strings.Repeat("=", 80))

	// Общая статистика
	log.Printf("\n📊 ОБЩАЯ СТАТИСТИКА:")
	log.Printf("   Всего протестировано: %d прокси", len(allResults))
	log.Printf("   Успешно: %d прокси", len(successResults))
	log.Printf("   Неуспешно: %d прокси", len(failedResults))
	log.Printf("   Успешность: %.1f%%", float64(len(successResults))/float64(len(allResults))*100)

	// Статистика по протоколам
	protocolStats := make(map[string]int)
	protocolSuccess := make(map[string]int)
	
	for _, result := range allResults {
		protocolStats[result.Protocol]++
		if result.Success {
			protocolSuccess[result.Protocol]++
		}
	}

	log.Printf("\n🌐 СТАТИСТИКА ПО ПРОТОКОЛАМ:")
	for protocol, total := range protocolStats {
		success := protocolSuccess[protocol]
		successRate := 0.0
		if total > 0 {
			successRate = float64(success)/float64(total)*100
		}
		log.Printf("   %s: %d/%d (%.1f%%)", protocol, success, total, successRate)
	}

	// === СПИСОК РАБОЧИХ ПРОКСИ (ОТСОРТИРОВАННЫХ ПО СКОРОСТИ) ===

	log.Printf("\n✅ СПИСОК РАБОЧИХ ПРОКСИ (отсортирован по скорости):")
	log.Println(strings.Repeat("-", 80))
	
	if len(successResults) > 0 {
		// Средняя задержка
		var totalLatency time.Duration
		for _, result := range successResults {
			totalLatency += result.Latency
		}
		avgLatency := totalLatency / time.Duration(len(successResults))
		
		log.Printf("📈 Средняя задержка: %v", avgLatency)
		log.Printf("🏆 Лучшая задержка: %v", successResults[0].Latency)
		log.Printf("🐢 Худшая задержка: %v", successResults[len(successResults)-1].Latency)
		
		log.Println("\n🏁 РЕЙТИНГ ПРОКСИ ПО СКОРОСТИ:")
		
		for i, result := range successResults {
			rank := i + 1
			latencyStr := fmt.Sprintf("%v", result.Latency)
			
			// Цветовая маркировка по скорости
			status := "🟢" // отличная скорость
			if result.Latency > 1*time.Second {
				status = "🟡" // средняя скорость
			}
			if result.Latency > 3*time.Second {
				status = "🔴" // медленная скорость
			}
			
			log.Printf("%2d. %s %-40s | %-8s | %s:%d | %s", 
				rank, status, result.Name, result.Protocol, result.Server, result.Port, latencyStr)
		}
	} else {
		log.Println("❌ Рабочих прокси не найдено")
	}

	// === ДЕТАЛЬНАЯ СТАТИСТИКА СКОРОСТИ ===

	if len(successResults) > 0 {
		log.Printf("\n📊 ДЕТАЛЬНАЯ СТАТИСТИКА СКОРОСТИ:")
		
		// Группировка по диапазонам задержки
		var fastCount, mediumCount, slowCount int
		var fastLatency, mediumLatency, slowLatency time.Duration
		
		for _, result := range successResults {
			if result.Latency < 500*time.Millisecond {
				fastCount++
				fastLatency += result.Latency
			} else if result.Latency < 2*time.Second {
				mediumCount++
				mediumLatency += result.Latency
			} else {
				slowCount++
				slowLatency += result.Latency
			}
		}
		
		log.Printf("   🟢 Быстрые (<500ms): %d прокси", fastCount)
		if fastCount > 0 {
			log.Printf("      Средняя задержка: %v", fastLatency/time.Duration(fastCount))
		}
		
		log.Printf("   🟡 Средние (500ms-2s): %d прокси", mediumCount)
		if mediumCount > 0 {
			log.Printf("      Средняя задержка: %v", mediumLatency/time.Duration(mediumCount))
		}
		
		log.Printf("   🔴 Медленные (>2s): %d прокси", slowCount)
		if slowCount > 0 {
			log.Printf("      Средняя задержка: %v", slowLatency/time.Duration(slowCount))
		}
	}

	// === НЕУСПЕШНЫЕ ПРОКСИ ===

	if len(failedResults) > 0 {
		log.Printf("\n❌ НЕУСПЕШНЫЕ ПРОКСИ (%d):", len(failedResults))
		log.Println(strings.Repeat("-", 80))
		
		for i, result := range failedResults {
			errorMsg := "неизвестная ошибка"
			if result.Error != nil {
				errorMsg = result.Error.Error()
				// Укорачиваем длинные сообщения об ошибках
				if len(errorMsg) > 80 {
					errorMsg = errorMsg[:77] + "..."
				}
			}
			
			log.Printf("%2d. ❌ %-40s | %-8s | %s", 
				i+1, result.Name, result.Protocol, errorMsg)
		}
	}

	// === ЭКСПОРТ РАБОЧИХ ПРОКСИ В ФАЙЛ ===

	if len(successResults) > 0 {
		exportFile := "working_proxies.txt"
		file, err := os.Create(exportFile)
		if err == nil {
			defer file.Close()
			
			file.WriteString("# Список рабочих прокси (отсортирован по скорости)\n")
			file.WriteString("# Дата тестирования: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
			file.WriteString("# Всего протестировано: " + fmt.Sprintf("%d", len(allResults)) + " прокси\n")
			file.WriteString("# Успешно: " + fmt.Sprintf("%d", len(successResults)) + " прокси\n\n")
			
			for i, result := range successResults {
				file.WriteString(fmt.Sprintf("%d. %s | %s:%d | %s | %v\n", 
					i+1, result.Name, result.Server, result.Port, result.Protocol, result.Latency))
			}
			
			log.Printf("\n💾 Список рабочих прокси экспортирован в файл: %s", exportFile)
		}
	}

	log.Println("\n" + strings.Repeat("=", 80))
	log.Println("=== ТЕСТИРОВАНИЕ ЗАВЕРШЕНО ===")
	log.Println(strings.Repeat("=", 80))
}