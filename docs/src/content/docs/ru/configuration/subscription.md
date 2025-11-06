---
title: Формат подписки
description: Варианты и примеры формата подписки
---

Xray Checker поддерживает четыре различных формата для конфигурации прокси. Для настройки используйте [переменную окружения](/ru/configuration/envs#subscription_url) `SUBSCRIPTION_URL`.

Подробнее о методах проверки прокси читайте в разделе [методы проверки](/ru/configuration/check-methods).

### 1. URL подписки (По умолчанию)

Стандартный URL подписки, возвращающий Base64-кодированный список прокси-ссылок.

Пример:

```bash
SUBSCRIPTION_URL="https://example.com/subscription"
```

Требования:

- HTTPS URL
- Возвращает Base64-кодированное содержимое
- Содержимое - это прокси-URL, разделенные переносом строки
- Поддерживает стандартные заголовки User-Agent

Отправляемые заголовки:

```
Accept: */*
User-Agent: Xray-Checker
```

### 2. Строка Base64

Прямая Base64-кодированная строка, содержащая ссылки конфигурации прокси.

Пример:

```bash
SUBSCRIPTION_URL="dmxlc3M6Ly91dWlkQGV4YW1wbGUuY29tOjQ0MyVlbmNyeXB0aW9uPW5vbmUmc2VjdXJpdHk9dGxzI3Byb3h5MQ=="
```

Формат содержимого (до кодирования):

```
vless://uuid@example.com:443?encryption=none&security=tls#proxy1
trojan://password@example.com:443?security=tls#proxy2
vmess://base64encodedconfig
ss://base64encodedconfig
```

### 3. JSON-файл V2Ray

Один JSON-файл конфигурации в формате V2Ray/Xray.

Пример:

```bash
SUBSCRIPTION_URL="file:///path/to/config.json"
```

Формат файла:

```json
{
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "example.com",
            "port": 443,
            "users": [
              {
                "id": "uuid",
                "encryption": "none"
              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls"
      }
    }
  ]
}
```

### 4. Папка с конфигурациями

Директория, содержащая несколько JSON-файлов конфигурации V2Ray/Xray.

Пример:

```bash
SUBSCRIPTION_URL="folder:///path/to/configs"
```

Требования:

- Директория должна содержать .json файлы
- Каждый файл следует формату JSON V2Ray
- Файлы обрабатываются в алфавитном порядке
- Некорректные файлы пропускаются с предупреждением
