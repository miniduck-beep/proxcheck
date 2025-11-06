---
title: API Reference
description: API Reference of Xray Checker
---

## Available Endpoints

Xray Checker provides several HTTP endpoints for monitoring and status checking:

### Health Check Endpoint

```http
GET /health
```

Simple health check endpoint that returns HTTP 200 if the service is running.

**Response:**

- Status: `200 OK`
- Body: `OK`

### Metrics Endpoint

```http
GET /metrics
```

Prometheus metrics endpoint providing detailed proxy status and latency information.

**Response:**

- Status: `200 OK`
- Content-Type: `text/plain; version=0.0.4`

Example metrics:

```text
# HELP xray_proxy_status Status of proxy connection (1: success, 0: failure)
# TYPE xray_proxy_status gauge
xray_proxy_status{protocol="vless",address="example.com:443",name="proxy1"} 1

# HELP xray_proxy_latency_ms Latency of proxy connection in milliseconds
# TYPE xray_proxy_latency_ms gauge
xray_proxy_latency_ms{protocol="vless",address="example.com:443",name="proxy1"} 156
```

### Proxy Status Endpoint

```http
GET /config/{index}-{protocol}-{server}-{port}
```

Individual proxy status endpoint, perfect for uptime monitoring.

**Parameters:**

- `index`: Proxy index number
- `protocol`: Protocol type (vless/vmess/trojan/shadowsocks)
- `server`: Server address
- `port`: Server port

**Response:**

- Status: `200 OK` if proxy is working
- Status: `503 Service Unavailable` if proxy is not working
- Body: `OK` or `Failed`

Example:

```bash
# Check specific proxy status
curl http://localhost:2112/config/0-vless-example.com-443
```

### Web Interface

```http
GET /
```

Returns the HTML dashboard with proxy status overview.

## Authentication

When enabled (`METRICS_PROTECTED=true`), endpoints are protected with Basic Authentication:

- Username: Configured via `METRICS_USERNAME`
- Password: Configured via `METRICS_PASSWORD`

Example with authentication:

```bash
curl -u username:password http://localhost:2112/metrics
```

## Integration Examples

### Uptime Kuma

```bash
# Add monitor with URL
http://localhost:2112/config/0-vless-example.com-443

# Add authentication if enabled
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

## Error Responses

The API returns standard HTTP status codes:

- `200 OK`: Request successful
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Authentication failed
- `404 Not Found`: Endpoint or proxy not found
- `503 Service Unavailable`: Proxy check failed
