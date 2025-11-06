---
title: Справочник по API
description: Справочник по API Xray Checker
---

## Доступные эндпоинты

Xray Checker предоставляет несколько HTTP-эндпоинтов для мониторинга и проверки статуса:

### Эндпоинт проверки работоспособности

```http
GET /health
```

Простой эндпоинт проверки работоспособности, который возвращает HTTP 200, если сервис работает.

**Ответ:**

- Статус: `200 OK`
- Тело: `OK`

### Эндпоинт метрик

```http
GET /metrics
```

Эндпоинт метрик Prometheus, предоставляющий подробную информацию о статусе прокси и задержках.

**Ответ:**

- Статус: `200 OK`
- Content-Type: `text/plain; version=0.0.4`

Пример метрик:

```text
# HELP xray_proxy_status Статус прокси-соединения (1: успешно, 0: неудача)
# TYPE xray_proxy_status gauge
xray_proxy_status{protocol="vless",address="example.com:443",name="proxy1"} 1

# HELP xray_proxy_latency_ms Задержка прокси-соединения в миллисекундах
# TYPE xray_proxy_latency_ms gauge
xray_proxy_latency_ms{protocol="vless",address="example.com:443",name="proxy1"} 156
```

### Эндпоинт статуса прокси

```http
GET /config/{index}-{protocol}-{server}-{port}
```

Эндпоинт статуса отдельного прокси, идеально подходит для мониторинга доступности.

**Параметры:**

- `index`: Номер индекса прокси
- `protocol`: Тип протокола (vless/vmess/trojan/shadowsocks)
- `server`: Адрес сервера
- `port`: Порт сервера

**Ответ:**

- Статус: `200 OK` если прокси работает
- Статус: `503 Service Unavailable` если прокси не работает
- Тело: `OK` или `Failed`

Пример:

```bash
# Проверка статуса конкретного прокси
curl http://localhost:2112/config/0-vless-example.com-443
```

### Веб-интерфейс

```http
GET /
```

Возвращает HTML-панель с обзором статуса прокси.

## Аутентификация

При включении (`METRICS_PROTECTED=true`), эндпоинты защищены Basic Authentication:

- Имя пользователя: Настраивается через `METRICS_USERNAME`
- Пароль: Настраивается через `METRICS_PASSWORD`

Пример с аутентификацией:

```bash
curl -u username:password http://localhost:2112/metrics
```

## Примеры интеграции

### Uptime Kuma

```bash
# Добавить монитор с URL
http://localhost:2112/config/0-vless-example.com-443

# Добавить аутентификацию, если включена
http://username:password@localhost:2112/config/0-vless-example.com-443
```

### Prometheus

```yaml
scrape_configs:
  - job_name: "xray-checker"
    metrics_path: "/metrics"
    basic_auth:
      username: "username"
      password: "password"
    static_configs:
      - targets: ["localhost:2112"]
```

## Коды ошибок

API возвращает стандартные HTTP-коды статуса:

- `200 OK`: Запрос успешен
- `401 Unauthorized`: Требуется аутентификация
- `403 Forbidden`: Ошибка аутентификации
- `404 Not Found`: Эндпоинт или прокси не найден
- `503 Service Unavailable`: Проверка прокси не удалась
