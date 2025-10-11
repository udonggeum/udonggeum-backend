# 로깅 가이드

## 개요

UDONGGEUM 백엔드는 구조화된 로깅을 위해 **zerolog**를 사용합니다. 이를 통해 로그 레벨별 필터링, JSON 포맷, 컨텍스트 기반 로깅이 가능합니다.

## 로그 레벨

| 레벨 | 용도 | 예시 |
|------|------|------|
| **Debug** | 개발 중 디버깅 정보 | 변수 값, 내부 상태 |
| **Info** | 일반 정보 메시지 | 서버 시작, 요청 완료 |
| **Warn** | 경고 (에러는 아님) | 재시도, deprecated 기능 사용 |
| **Error** | 에러 발생 (복구 가능) | DB 연결 실패, 요청 처리 실패 |
| **Fatal** | 치명적 에러 (프로그램 종료) | 설정 로드 실패, 필수 리소스 없음 |

## 로그 포맷

### Console 포맷 (개발 환경)
```
2025-10-11T15:30:45+09:00 INF Starting UDONGGEUM Backend Server environment=development port=8080
2025-10-11T15:30:45+09:00 INF Database connection established successfully max_idle_conns=10 max_open_conns=100
2025-10-11T15:30:46+09:00 INF Incoming request method=POST path=/api/v1/auth/login ip=127.0.0.1
2025-10-11T15:30:46+09:00 INF Request completed status_code=200 latency_ms=45 body_size=256
```

### JSON 포맷 (프로덕션 환경)
```json
{
  "level": "info",
  "time": "2025-10-11T15:30:45+09:00",
  "caller": "cmd/server/main.go:37",
  "message": "Starting UDONGGEUM Backend Server",
  "environment": "production",
  "port": "8080"
}
```

## 사용 방법

### 1. 기본 로깅

```go
import "github.com/ikkim/udonggeum-backend/pkg/logger"

// Info 로그
logger.Info("User logged in successfully")

// 필드와 함께 로깅
logger.Info("User logged in", map[string]interface{}{
    "user_id": 123,
    "email": "user@example.com",
    "ip": "192.168.1.1",
})

// 에러 로깅
err := doSomething()
if err != nil {
    logger.Error("Failed to process request", err, map[string]interface{}{
        "user_id": userID,
        "operation": "create_order",
    })
}
```

### 2. 컨텍스트 기반 로깅

반복적인 필드를 자동으로 포함시키려면 컨텍스트 로거를 사용하세요:

```go
// 컨텍스트 로거 생성
contextLogger := logger.WithContext(map[string]interface{}{
    "user_id": 123,
    "request_id": "abc-123",
})

// 이후 로그에 자동으로 user_id, request_id 포함됨
contextLogger.Info("Processing order")
contextLogger.Info("Order completed", map[string]interface{}{
    "order_id": 456,
    "amount": 50000,
})
```

### 3. HTTP 핸들러에서 로깅

Gin 컨텍스트에서 로거를 가져올 수 있습니다:

```go
func (ctrl *ProductController) GetProducts(c *gin.Context) {
    log := middleware.GetLoggerFromContext(c)

    log.Info("Fetching products")

    products, err := ctrl.productService.GetAllProducts()
    if err != nil {
        log.Error("Failed to fetch products", err)
        c.JSON(500, gin.H{"error": "Internal server error"})
        return
    }

    log.Info("Products fetched successfully", map[string]interface{}{
        "count": len(products),
    })
    c.JSON(200, gin.H{"products": products})
}
```

### 4. 각 레벨별 사용 예시

```go
// Debug - 개발 중 상세 정보
logger.Debug("Checking product stock", map[string]interface{}{
    "product_id": 123,
    "current_stock": 50,
    "requested_quantity": 5,
})

// Info - 정상 동작
logger.Info("Order created successfully", map[string]interface{}{
    "order_id": 789,
    "user_id": 123,
    "total_amount": 100000,
})

// Warn - 주의 필요 (에러는 아님)
logger.Warn("Low stock detected", map[string]interface{}{
    "product_id": 123,
    "current_stock": 2,
    "threshold": 5,
})

// Error - 에러 발생
logger.Error("Failed to process payment", err, map[string]interface{}{
    "order_id": 789,
    "payment_method": "card",
})

// Fatal - 프로그램 종료 (사용 주의!)
logger.Fatal("Failed to load configuration", err)
```

## 설정

### 환경변수

`.env` 파일에서 로그 레벨을 제어할 수 있습니다:

```env
# 개발 환경 - debug 로그 활성화
SERVER_ENVIRONMENT=development

# 프로덕션 환경 - info 이상만 로깅
SERVER_ENVIRONMENT=production
```

### 코드에서 초기화

`cmd/server/main.go`에서 로거를 초기화합니다:

```go
logger.Initialize(logger.Config{
    Level:       "info",    // debug, info, warn, error, fatal
    Format:      "console", // console (예쁜 출력) 또는 json (파싱 가능)
    EnableColor: true,      // 콘솔 색상 활성화
})
```

## HTTP 요청 로깅

모든 HTTP 요청은 자동으로 로깅됩니다:

```
2025-10-11T15:30:46+09:00 INF Incoming request request_id=20251011153046.123 method=POST path=/api/v1/orders ip=127.0.0.1 user_agent="Mozilla/5.0..."
2025-10-11T15:30:46+09:00 INF Request completed request_id=20251011153046.123 status_code=201 latency_ms=145 body_size=512
```

### 로깅 정보:
- `request_id`: 요청 고유 ID (추적용)
- `method`: HTTP 메서드 (GET, POST, etc.)
- `path`: 요청 경로
- `ip`: 클라이언트 IP
- `status_code`: 응답 상태 코드
- `latency_ms`: 요청 처리 시간 (밀리초)
- `body_size`: 응답 바디 크기

### 상태 코드별 로그 레벨:
- **200-399**: Info 레벨
- **400-499**: Warn 레벨 (클라이언트 에러)
- **500-599**: Error 레벨 (서버 에러)

## 베스트 프랙티스

### ✅ 좋은 예

```go
// 1. 구조화된 필드 사용
logger.Info("Order processed", map[string]interface{}{
    "order_id": order.ID,
    "user_id": user.ID,
    "amount": order.TotalAmount,
    "payment_status": order.PaymentStatus,
})

// 2. 에러와 함께 컨텍스트 제공
if err != nil {
    logger.Error("Failed to create order", err, map[string]interface{}{
        "user_id": userID,
        "cart_items": len(items),
    })
    return err
}

// 3. 민감 정보 제외
logger.Info("User registered", map[string]interface{}{
    "user_id": user.ID,
    "email": user.Email,
    // ❌ "password": user.Password (절대 로깅하지 말 것!)
})

// 4. 적절한 로그 레벨 사용
logger.Debug("Cache miss, fetching from DB") // 개발용
logger.Info("User logged in successfully")   // 일반 정보
logger.Warn("Rate limit approaching")        // 경고
logger.Error("Database query failed", err)   // 에러
```

### ❌ 나쁜 예

```go
// 1. 문자열 포맷팅 사용 (검색 어려움)
logger.Info(fmt.Sprintf("Order %d processed for user %d", orderID, userID))
// 👉 대신 구조화된 필드 사용

// 2. 민감 정보 로깅
logger.Info("Login attempt", map[string]interface{}{
    "password": password,        // ❌ 절대 안됨!
    "credit_card": cardNumber,   // ❌ 절대 안됨!
})

// 3. 과도한 로깅
for _, item := range items {
    logger.Debug("Processing item", map[string]interface{}{
        "item_id": item.ID,  // ❌ 루프 안에서 로깅 지양
    })
}
// 👉 대신 요약 정보 로깅
logger.Debug("Processing items", map[string]interface{}{
    "count": len(items),
})

// 4. 부적절한 로그 레벨
logger.Error("User not found", nil)  // ❌ Error는 예상치 못한 에러용
logger.Info("Database connection failed", err)  // ❌ Error 레벨 사용해야 함
```

## 로그 모니터링

### 개발 환경
콘솔에서 직접 확인:
```bash
make run
# 또는
go run cmd/server/main.go
```

### 프로덕션 환경

JSON 로그는 다양한 도구로 파싱 가능합니다:

**1. jq로 필터링:**
```bash
# 에러 로그만 보기
./server 2>&1 | jq 'select(.level == "error")'

# 특정 사용자 로그만 보기
./server 2>&1 | jq 'select(.user_id == 123)'

# 느린 요청 찾기 (100ms 이상)
./server 2>&1 | jq 'select(.latency_ms > 100)'
```

**2. 로그 수집 시스템:**
- **ELK Stack**: Elasticsearch + Logstash + Kibana
- **Loki**: Grafana Loki + Promtail
- **CloudWatch**: AWS CloudWatch Logs
- **Datadog**: Datadog Logs

## 문제 해결

### 로그가 너무 많아요
```go
// .env 파일에서 로그 레벨 상향
SERVER_ENVIRONMENT=production  // debug 로그 비활성화
```

### 로그가 보이지 않아요
```go
// 로거 초기화 확인
logger.Initialize(logger.Config{
    Level: "debug",  // 모든 로그 보기
    Format: "console",
    EnableColor: true,
})
```

### JSON 로그로 변경하고 싶어요
```go
// main.go에서
logger.Initialize(logger.Config{
    Level: "info",
    Format: "json",  // console 대신 json
    EnableColor: false,
})
```

## 예제

전체 예제는 다음 파일을 참고하세요:
- [pkg/logger/logger.go](pkg/logger/logger.go) - 로거 구현
- [internal/middleware/logging_middleware.go](internal/middleware/logging_middleware.go) - HTTP 로깅 미들웨어
- [cmd/server/main.go](cmd/server/main.go) - 로거 초기화 및 사용

---

**참고**: 로그는 디버깅과 모니터링의 핵심입니다. 적절한 로그 레벨과 구조화된 필드를 사용하여 시스템을 효과적으로 관리하세요!
