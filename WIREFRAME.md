# 우동금(UDONGGEUM) 프로젝트 와이어프레임

## 1. 시스템 아키텍처 개요

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend Client                         │
│                   (React/Vue/Mobile App)                     │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP/HTTPS (REST API)
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                   API Gateway Layer                          │
│                    (Gin Framework)                           │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │            Middleware Layer                        │    │
│  │  • CORS                                            │    │
│  │  • JWT Authentication                              │    │
│  │  • Request Logging                                 │    │
│  │  • Error Handling                                  │    │
│  └────────────────────────────────────────────────────┘    │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                  Application Layer                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Controller   │  │ Controller   │  │ Controller   │     │
│  │   (Auth)     │  │  (Product)   │  │   (Order)    │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │              │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐     │
│  │   Service    │  │   Service    │  │   Service    │     │
│  │   (Auth)     │  │  (Product)   │  │   (Order)    │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │              │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐     │
│  │ Repository   │  │ Repository   │  │ Repository   │     │
│  │   (User)     │  │  (Product)   │  │   (Order)    │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │              │
└─────────┼──────────────────┼──────────────────┼──────────────┘
          │                  │                  │
┌─────────▼──────────────────▼──────────────────▼──────────────┐
│                  Database Layer                               │
│                 PostgreSQL + GORM                             │
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  users   │  │ products │  │  orders  │  │order_items│   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└───────────────────────────────────────────────────────────────┘
```

## 2. API 엔드포인트 구조

### 2.1 인증 (Authentication)
```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
GET  /api/v1/auth/me [Protected]
```

### 2.2 상품 (Products)
```
GET    /api/v1/products           # 전체 상품 목록 조회
GET    /api/v1/products/:id       # 상품 상세 조회
POST   /api/v1/products [Admin]   # 상품 등록
PUT    /api/v1/products/:id [Admin] # 상품 수정
DELETE /api/v1/products/:id [Admin] # 상품 삭제
```

### 2.3 장바구니 (Cart)
```
GET    /api/v1/cart [Protected]      # 장바구니 조회
POST   /api/v1/cart [Protected]      # 장바구니 추가
PUT    /api/v1/cart/:id [Protected]  # 장바구니 수정
DELETE /api/v1/cart/:id [Protected]  # 장바구니 삭제
```

### 2.4 주문 (Orders)
```
GET  /api/v1/orders [Protected]       # 주문 내역 조회
GET  /api/v1/orders/:id [Protected]   # 주문 상세 조회
POST /api/v1/orders [Protected]       # 주문 생성
```

### 2.5 결제 (Payment)
```
POST /api/v1/payment/prepare [Protected]  # 결제 준비
POST /api/v1/payment/complete [Protected] # 결제 완료
POST /api/v1/payment/cancel [Protected]   # 결제 취소
```

## 3. 데이터베이스 ERD

```
┌─────────────────────────┐
│        users            │
├─────────────────────────┤
│ id (PK)                 │
│ email (unique)          │
│ password_hash           │
│ name                    │
│ phone                   │
│ role (user/admin)       │
│ created_at              │
│ updated_at              │
└───────────┬─────────────┘
            │
            │ 1:N
            │
┌───────────▼─────────────┐
│       orders            │
├─────────────────────────┤
│ id (PK)                 │
│ user_id (FK)            │
│ total_amount            │
│ status                  │
│ payment_status          │
│ shipping_address        │
│ created_at              │
│ updated_at              │
└───────────┬─────────────┘
            │
            │ 1:N
            │
┌───────────▼─────────────┐          ┌─────────────────────────┐
│     order_items         │   N:1    │       products          │
├─────────────────────────┤◄─────────├─────────────────────────┤
│ id (PK)                 │          │ id (PK)                 │
│ order_id (FK)           │          │ name                    │
│ product_id (FK)         │          │ description             │
│ quantity                │          │ price                   │
│ price                   │          │ weight                  │
│ created_at              │          │ purity (금 순도)        │
└─────────────────────────┘          │ category                │
                                     │ stock_quantity          │
                                     │ image_url               │
                                     │ created_at              │
                                     │ updated_at              │
                                     └───────────┬─────────────┘
                                                 │
                                                 │ 1:N
                                                 │
                                     ┌───────────▼─────────────┐
                                     │      cart_items         │
                                     ├─────────────────────────┤
                                     │ id (PK)                 │
                                     │ user_id (FK)            │
                                     │ product_id (FK)         │
                                     │ quantity                │
                                     │ created_at              │
                                     │ updated_at              │
                                     └─────────────────────────┘
```

## 4. 주요 기능 플로우

### 4.1 회원가입 플로우
```
Client                Controller           Service              Repository          DB
  │                       │                   │                     │                │
  ├─ POST /auth/register ─►                   │                     │                │
  │                       ├─ RegisterUser() ──►                     │                │
  │                       │                   ├─ ValidateEmail()    │                │
  │                       │                   ├─ HashPassword()     │                │
  │                       │                   ├─ CreateUser() ──────►                │
  │                       │                   │                     ├─ INSERT ───────►
  │                       │                   │                     ◄─ User ─────────┤
  │                       │                   ◄─ User ──────────────┤                │
  │                       │                   ├─ GenerateJWT()      │                │
  │                       ◄─ JWT Token ───────┤                     │                │
  ◄─ 201 Created + Token ─┤                   │                     │                │
```

### 4.2 로그인 플로우
```
Client                Controller           Service              Repository          DB
  │                       │                   │                     │                │
  ├─ POST /auth/login ────►                   │                     │                │
  │                       ├─ Login() ─────────►                     │                │
  │                       │                   ├─ FindByEmail() ─────►                │
  │                       │                   │                     ├─ SELECT ───────►
  │                       │                   │                     ◄─ User ─────────┤
  │                       │                   ◄─ User ──────────────┤                │
  │                       │                   ├─ VerifyPassword()   │                │
  │                       │                   ├─ GenerateJWT()      │                │
  │                       ◄─ JWT Token ───────┤                     │                │
  ◄─ 200 OK + Token ──────┤                   │                     │                │
```

### 4.3 상품 조회 플로우
```
Client                Controller           Service              Repository          DB
  │                       │                   │                     │                │
  ├─ GET /products ───────►                   │                     │                │
  │                       ├─ GetProducts() ───►                     │                │
  │                       │                   ├─ FindAll() ─────────►                │
  │                       │                   │                     ├─ SELECT ───────►
  │                       │                   │                     ◄─ Products ─────┤
  │                       │                   ◄─ Products ──────────┤                │
  │                       ◄─ Products ────────┤                     │                │
  ◄─ 200 OK + Products ───┤                   │                     │                │
```

### 4.4 주문 생성 플로우
```
Client            Controller        Service         Repository        Payment Service    DB
  │                  │                │                 │                    │            │
  ├─ POST /orders ───►                │                 │                    │            │
  │  [JWT Token]     │                │                 │                    │            │
  │                  ├─ Auth Middleware ────────────────►                    │            │
  │                  │                │                 │                    │            │
  │                  ├─ CreateOrder() ─►                │                    │            │
  │                  │                ├─ ValidateCart() │                    │            │
  │                  │                ├─ CheckStock() ──►                    │            │
  │                  │                │                 ├─ SELECT products ──►            │
  │                  │                │                 ◄─ Products ─────────┤            │
  │                  │                ├─ CalculateTotal()                    │            │
  │                  │                ├─ CreateOrder() ──►                   │            │
  │                  │                │                 ├─ INSERT order ─────►            │
  │                  │                │                 ├─ INSERT order_items►            │
  │                  │                │                 ◄─ Order ────────────┤            │
  │                  │                ◄─ Order ─────────┤                    │            │
  │                  ◄─ Order ────────┤                 │                    │            │
  ◄─ 201 Created ────┤                │                 │                    │            │
```

### 4.5 결제 플로우
```
Client            Controller        Service         Payment Gateway     Repository        DB
  │                  │                │                    │                │             │
  ├─ POST /payment ──►                │                    │                │             │
  │    /complete     │                │                    │                │             │
  │  [JWT + OrderID] │                │                    │                │             │
  │                  ├─ ProcessPayment()                   │                │             │
  │                  │                ├─ GetOrder() ───────►                │             │
  │                  │                │                    │                ├─ SELECT ────►
  │                  │                │                    │                ◄─ Order ─────┤
  │                  │                ├─ CallPaymentAPI() ─►                │             │
  │                  │                │                    ├─ Process       │             │
  │                  │                │                    ◄─ Success ──────┤             │
  │                  │                ├─ UpdateOrderStatus()                │             │
  │                  │                │                    │                ├─ UPDATE ────►
  │                  │                ├─ UpdateStock() ────►                │             │
  │                  │                │                    │                ├─ UPDATE ────►
  │                  ◄─ PaymentResult ┤                    │                │             │
  ◄─ 200 OK ─────────┤                │                    │                │             │
```

## 5. 컴포넌트 상세 설명

### 5.1 Controller Layer
- **역할**: HTTP 요청 수신, 검증, 응답 반환
- **책임**:
  - Request 파싱 및 유효성 검사
  - Service 계층 호출
  - Response 포맷팅
  - HTTP 상태 코드 설정

### 5.2 Service Layer
- **역할**: 비즈니스 로직 처리
- **책임**:
  - 비즈니스 규칙 적용
  - 트랜잭션 관리
  - 여러 Repository 조율
  - 외부 서비스 통합

### 5.3 Repository Layer
- **역할**: 데이터베이스 접근
- **책임**:
  - CRUD 연산
  - 쿼리 최적화
  - 데이터 매핑

### 5.4 Middleware
- **JWT Authentication**: 토큰 검증 및 사용자 인증
- **CORS**: Cross-Origin 요청 처리
- **Logging**: 요청/응답 로깅
- **Error Handling**: 통합 에러 처리

## 6. 보안 고려사항

```
┌─────────────────────────────────────────────────────────┐
│                    Security Layers                       │
├─────────────────────────────────────────────────────────┤
│ 1. HTTPS/TLS Encryption                                 │
│    └─ 전송 중 데이터 암호화                               │
│                                                          │
│ 2. JWT Token Authentication                             │
│    ├─ Access Token (15분)                               │
│    └─ Refresh Token (7일)                               │
│                                                          │
│ 3. Password Security                                    │
│    ├─ bcrypt 해싱 (cost factor: 12)                    │
│    └─ Salt 자동 생성                                     │
│                                                          │
│ 4. Input Validation                                     │
│    ├─ Request Body 검증                                 │
│    └─ SQL Injection 방지 (GORM Prepared Statements)    │
│                                                          │
│ 5. Rate Limiting                                        │
│    └─ API 호출 제한 (IP 기반)                            │
│                                                          │
│ 6. CORS Policy                                          │
│    └─ 허용된 도메인만 접근                                │
└─────────────────────────────────────────────────────────┘
```

## 7. 환경 설정 구조

```
Development Environment
├─ .env.development
│  ├─ DB_HOST=localhost
│  ├─ DB_PORT=5432
│  ├─ JWT_SECRET=dev_secret
│  └─ LOG_LEVEL=debug
│
Production Environment
├─ .env.production
│  ├─ DB_HOST=prod-db-server
│  ├─ DB_PORT=5432
│  ├─ JWT_SECRET=prod_secure_secret
│  └─ LOG_LEVEL=info
│
Testing Environment
└─ .env.test
   ├─ DB_HOST=localhost
   ├─ DB_PORT=5433
   └─ LOG_LEVEL=error
```

## 8. 배포 아키텍처 (선택사항)

```
┌──────────────────────────────────────────────────────────┐
│                    Load Balancer                         │
│                      (Nginx)                             │
└─────────────┬────────────────┬───────────────────────────┘
              │                │
    ┌─────────▼─────┐    ┌────▼──────────┐
    │  App Server 1 │    │  App Server 2 │
    │   (Go + Gin)  │    │   (Go + Gin)  │
    └─────────┬─────┘    └────┬──────────┘
              │                │
              └────────┬───────┘
                       │
              ┌────────▼─────────┐
              │   PostgreSQL     │
              │   (Primary DB)   │
              └──────────────────┘
```

## 9. 개발 우선순위

### Phase 1: 기본 인프라
1. 프로젝트 구조 설정
2. 데이터베이스 연결 및 마이그레이션
3. 기본 라우터 설정

### Phase 2: 인증 시스템
1. 회원가입/로그인 API
2. JWT 미들웨어 구현
3. 사용자 관리 기능

### Phase 3: 상품 관리
1. 상품 CRUD API
2. 상품 이미지 처리
3. 재고 관리

### Phase 4: 주문 시스템
1. 장바구니 기능
2. 주문 생성 API
3. 주문 내역 조회

### Phase 5: 결제 통합
1. 결제 API 연동
2. 결제 상태 관리
3. 결제 취소/환불

### Phase 6: 최적화 및 배포
1. 성능 최적화
2. 에러 처리 강화
3. 배포 준비

## 10. 테스트 전략

```
┌─────────────────────────────────────────────┐
│              Test Pyramid                    │
├─────────────────────────────────────────────┤
│                                              │
│              ▲                               │
│             ╱ ╲  E2E Tests                   │
│            ╱   ╲ (API 통합 테스트)           │
│           ╱─────╲                            │
│          ╱       ╲                           │
│         ╱ Integration╲                       │
│        ╱   Tests      ╲                      │
│       ╱ (Service Layer)╲                     │
│      ╱─────────────────╲                    │
│     ╱                   ╲                    │
│    ╱   Unit Tests        ╲                   │
│   ╱  (Repository, Utils)  ╲                  │
│  ╱─────────────────────────╲                 │
│                                              │
└─────────────────────────────────────────────┘
```

---

## 참고사항

- 이 와이어프레임은 초기 설계 문서이며, 개발 과정에서 조정될 수 있습니다.
- 모든 API는 RESTful 원칙을 따릅니다.
- 데이터베이스 스키마는 GORM AutoMigrate를 통해 자동 생성됩니다.
- 프로덕션 환경에서는 추가적인 보안 및 성능 최적화가 필요합니다.
