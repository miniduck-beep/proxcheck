# Proxy Test API

REST API для управления тестированием прокси-серверов.

## 🚀 Быстрый старт

### Запуск API сервера

```bash
cd cmd/api
go mod tidy
go run main.go
```

Сервер запустится на `http://localhost:8080`

### Проверка работоспособности

```bash
curl http://localhost:8080/health
```

**Ответ:**
```json
{
  "status": "healthy",
  "timestamp": 1698636649,
  "version": "1.0.0",
  "service": "proxy-test-api"
}
```

## 📚 Документация

- **OpenAPI документация:** http://localhost:8080/docs/openapi.yaml
- **Swagger UI:** Можно открыть через Swagger Editor или Postman

## 🔌 Основные эндпоинты

### Health & Status
- `GET /health` - Проверка состояния сервера
- `GET /api/v1/status` - Детальный статус системы
- `GET /api/v1/config` - Конфигурация системы

### Управление тестами
- `POST /api/v1/tests` - Запуск нового теста
- `GET /api/v1/tests/{id}` - Статус теста
- `DELETE /api/v1/tests/{id}` - Остановка теста

### Результаты
- `GET /api/v1/results/{id}` - Результаты теста
- `GET /api/v1/results/{id}/working` - Список рабочих прокси
- `POST /api/v1/results/{id}/export` - Экспорт результатов

## 📋 Примеры использования

### Запуск теста

```bash
curl -X POST http://localhost:8080/api/v1/tests \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nightly-test",
    "proxy_count": 20,
    "timeout": 30
  }'
```

**Ответ:**
```json
{
  "test_id": "test_20251030053049",
  "status": "started",
  "message": "Test started successfully",
  "started_at": "2025-10-30T05:30:49Z"
}
```

### Получение статуса теста

```bash
curl http://localhost:8080/api/v1/tests/test_20251030053049
```

**Ответ:**
```json
{
  "test_id": "test_20251030053049",
  "name": "nightly-test",
  "status": "running",
  "proxy_count": 20,
  "started_at": "2025-10-30T05:30:49Z"
}
```

### Получение результатов

```bash
curl http://localhost:8080/api/v1/results/test_20251030053049
```

**Ответ:**
```json
{
  "test_id": "test_20251030053049",
  "total_proxies": 20,
  "successful": 4,
  "failed": 16,
  "success_rate": 20.0,
  "average_latency": "1.182s",
  "working_proxies": [
    {
      "name": "🇳🇱[openproxylist.com] ss-NL",
      "protocol": "shadowsocks",
      "server": "45.87.175.28",
      "port": 8080,
      "latency": "1.108s",
      "rank": 1
    }
  ]
}
```

## 🛠️ Использование клиента

Включен пример клиента для тестирования API:

```bash
cd cmd/api
go run client.go
```

Клиент автоматически:
1. Проверяет здоровье API
2. Запускает тест
3. Мониторит статус
4. Получает результаты

## 🔧 Настройка

### Конфигурация по умолчанию

API использует следующие настройки по умолчанию:

```yaml
xray:
  start_port: 10000
  log_level: error
proxy:
  check_method: ip
  ip_check_url: https://api.ipify.org?format=text
  timeout: 30
  simulate_latency: false
api:
  port: 8080
  max_concurrent_tests: 5
```

### Переменные окружения

Планируется поддержка переменных окружения для конфигурации:

```bash
export API_PORT=8080
export XRAY_START_PORT=10000
export PROXY_TIMEOUT=30
```

## 🏗️ Архитектура

### Компоненты

- **Gin Web Framework** - HTTP сервер и маршрутизация
- **In-memory хранилище** - Временное хранение тестов и результатов
- **Асинхронная обработка** - Тесты запускаются в горутинах

### Структура данных

```go
type Test struct {
    ID          string
    Name        string
    Status      string // pending, running, completed, failed, stopped
    ProxyCount  int
    StartedAt   time.Time
    CompletedAt time.Time
}

type TestResult struct {
    TestID       string
    TotalProxies int
    Successful   int
    Failed       int
    SuccessRate  float64
    AverageLatency string
    WorkingProxies []ProxyInfo
}
```

## 🔮 Планы развития

### Ближайшие улучшения

1. **Интеграция с proxytestlib** - Реальное тестирование вместо заглушек
2. **Постоянное хранилище** - База данных для результатов
3. **Аутентификация** - Защита API
4. **Мониторинг** - Prometheus метрики
5. **Документация** - Swagger UI

### Дополнительные функции

- Пакетное тестирование
- Планирование тестов (cron)
- Уведомления (email, webhook)
- Графический интерфейс

## 🐛 Отладка

### Логирование

API использует стандартное логирование Gin. Для включения детальных логов:

```go
gin.SetMode(gin.DebugMode)
```

### Мониторинг

Проверьте доступные эндпоинты:

```bash
# Health check
curl http://localhost:8080/health

# System status
curl http://localhost:8080/api/v1/status

# Configuration
curl http://localhost:8080/api/v1/config
```

## 📄 Лицензия

MIT License