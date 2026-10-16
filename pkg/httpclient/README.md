---
title: "HTTP Client Package"
description: "Resilient, generic, and structured-logged HTTP client for Go services in core-kit"
status: "RELEASED"
last_updated: "2026-09-29"
tier: "package"
service: "core-kit"
related_code:
  - pkg/httpclient/client.go
---

# HTTP Client Package (`pkg/httpclient`)

Package `httpclient` cung cấp HTTP client tái sử dụng, resilient và type-safe cho toàn bộ hệ thống Golang microservices trong `core-kit`.

Package được tích hợp trực tiếp với middleware **[`github.com/dangnmh/transport`](https://github.com/dangnmh/transport)** để tự động ghi log có cấu trúc (structured JSON logging) cho mọi request và response ở tầng `http.RoundTripper`.

---

## 1. Tính Năng Chính

- **Transport-Level Logging tự động (`dangnmh/transport`)**:
  - Tự động ghi nhận `method`, `url` (path), `status`, `latency`, `headers`, `req_body`, `res_body`.
  - Tích hợp với `log/slog` thông qua `logger.Get()`.
  - Tự động che giấu (redact) các header nhạy cảm: `Authorization`, `Cookie`, `Set-Cookie`.
  - Hỗ trợ phân loại log level theo HTTP status:
    - 2xx / 3xx $\rightarrow$ `INFO`
    - 4xx $\rightarrow$ `WARN`
    - 5xx / Network Error $\rightarrow$ `ERROR`
- **Go 1.27.1 Generic Methods on Structs**:
  - `Get[T](ctx, client, path)` / `client.GetParsed[T](ctx, path)`: Tự động unmarshal JSON response vào struct con trỏ `*T`.
  - `Post[T](ctx, client, path, reqBody)` / `client.PostParsed[T](ctx, path, reqBody)`: Serialize `reqBody` thành JSON và deserialize response vào `*T`.
- **Retry & Backoff**:
  - Tự động thử lại khi gặp lỗi mạng hoặc HTTP 5xx với khoảng nghỉ tùy biến.

---

## 2. Cách Sử Dụng

### Khởi tạo Client mặc định (Đã bật sẵn Log)
```go
import (
    "context"
    "time"
    "github.com/dangnmh/core-kit/pkg/httpclient"
)

client := httpclient.New(
    httpclient.WithBaseURL("http://localhost:3200"),
    httpclient.WithTimeout(5 * time.Second),
)

// Gửi request - Tự động được ghi log có cấu trúc qua dangnmh/transport
var data MyResponse
err := client.GetJSON(ctx, "/api/status", &data)
```

### Tùy Biến Hoặc Tắt Logging
```go
// Tắt logging cho các tác vụ tải dữ liệu lớn / streaming
clientNoLog := httpclient.New(
    httpclient.WithBaseURL("http://localhost:3200"),
    httpclient.WithoutTransportLog(),
)

// Tùy biến tùy chọn logging của dangnmh/transport
import "github.com/dangnmh/transport"

clientCustomLog := httpclient.New(
    httpclient.WithBaseURL("http://localhost:3200"),
    httpclient.WithTransportLog(
        transport.LogOptionMaxLogBodySize(1024), // Giới hạn kích thước log body
        transport.LogOptionQueryParams(true),    // Log query params riêng
    ),
)
```

---

## 3. Mẫu Log JSON Xuất Ra
```json
{
  "timestamp": "2026-09-23T10:14:45.375426817Z",
  "severity": "INFO",
  "message": "HTTP Request",
  "service": "api-service",
  "method": "POST",
  "url": "/api/actions",
  "headers": {
    "Content-Type": ["application/json"],
    "Authorization": ["[REDACTED]"]
  },
  "req_body": "{\"prompt\":\"...\"}",
  "status": 200,
  "latency": 624735,
  "res_body": "{\"response\":\"...\"}"
}
```

