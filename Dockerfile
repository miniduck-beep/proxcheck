# Этап 1: Сборка
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем только необходимые файлы
COPY go.mod go.sum ./
COPY cmd/api/main.go ./cmd/api/
COPY xray_config_vless_100.txt .

# Загружаем зависимости
RUN go mod tidy
RUN go mod download

# Устанавливаем Xray
RUN apk add --no-cache wget unzip &&     wget -O /usr/local/bin/xray.zip "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip" &&     unzip /usr/local/bin/xray.zip -d /usr/local/bin/xray-temp &&     mv /usr/local/bin/xray-temp/xray /usr/local/bin/xray &&     mv /usr/local/bin/xray-temp/geoip.dat /usr/local/bin/geoip.dat &&     mv /usr/local/bin/xray-temp/geosite.dat /usr/local/bin/geosite.dat &&     rm -rf /usr/local/bin/xray-temp /usr/local/bin/xray.zip &&     chmod +x /usr/local/bin/xray

# Собираем приложение
RUN cd cmd/api && go build -o /app/api-server

# Этап 2: Финальный образ
FROM alpine:latest

WORKDIR /app

# Копируем бинарник и файл с прокси
COPY --from=builder /app/api-server .
COPY --from=builder /app/xray_config_vless_100.txt .
COPY --from=builder /usr/local/bin/xray /usr/local/bin/xray
COPY --from=builder /usr/local/bin/geoip.dat /usr/local/bin/geoip.dat
COPY --from=builder /usr/local/bin/geosite.dat /usr/local/bin/geosite.dat

EXPOSE 8080

CMD ["./api-server"]
