# 우동금(UDONGGEUM) 백엔드

우리동네 금은방 기반 커머스 플랫폼 백엔드 API

## 프로젝트 개요

- **프로젝트명**: 우동금 (UDONGGEUM)
- **목적**: 금은방 상품 거래를 위한 REST API 서비스
- **기술 스택**: Go, Gin, PostgreSQL, GORM, JWT

## 주요 기능

- 사용자 인증 (회원가입/로그인) - JWT 기반
- 파일 업로드 (이미지)
- 상품 관리 (CRUD)
- 장바구니 관리
- 주문 처리
- 결제 처리 (Mock)

## 기술 스택

- **언어**: Go 1.21+
- **웹 프레임워크**: Gin
- **데이터베이스**: PostgreSQL
- **ORM**: GORM
- **인증**: JWT (golang-jwt/jwt)
- **환경 변수 관리**: godotenv

## 프로젝트 구조

```
udonggeum-backend/
├── cmd/
│   └── server/
│       └── main.go              # 서버 엔트리 포인트
├── config/
│   └── config.go                # 설정 관리
├── internal/
│   ├── app/
│   │   ├── controller/          # HTTP 핸들러
│   │   ├── service/             # 비즈니스 로직
│   │   ├── repository/          # 데이터베이스 접근
│   │   └── model/               # 데이터 모델
│   ├── db/
│   │   ├── database.go          # DB 연결
│   │   └── migrate.go           # 마이그레이션
│   ├── middleware/
│   │   └── auth_middleware.go  # JWT 인증
│   └── router/
│       └── router.go            # 라우팅 설정
├── pkg/
│   └── util/
│       ├── jwt.go               # JWT 유틸리티
│       └── password.go          # 비밀번호 해싱
├── .env.example                 # 환경 변수 예시
├── .gitignore
├── go.mod
├── go.sum
├── WIREFRAME.md                 # 프로젝트 와이어프레임
└── README.md
```

## 시작하기

### 사전 요구사항

- Go 1.21 이상
- PostgreSQL 12 이상
- Git

### 설치 및 실행

#### 1. 저장소 클론

```bash
git clone https://github.com/ikkim/udonggeum-backend.git
cd udonggeum-backend
```

#### 2. 의존성 설치

```bash
go mod download
```

#### 3. 환경 변수 설정

`.env.example` 파일을 복사하여 `.env` 파일을 생성하고 설정을 수정합니다.

```bash
cp .env.example .env
```

`.env` 파일 예시:

```env
# Server Configuration
SERVER_PORT=8080
GIN_MODE=debug

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=udonggeum
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-key-change-this-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

#### 4. PostgreSQL 데이터베이스 생성

```bash
# PostgreSQL 접속
psql -U postgres

# 데이터베이스 생성
CREATE DATABASE udonggeum;

# 종료
\q
```

#### 5. 서버 실행

```bash
# 개발 모드로 실행
go run cmd/server/main.go

# 또는 빌드 후 실행
go build -o bin/server cmd/server/main.go
./bin/server
```

서버가 성공적으로 시작되면 다음과 같은 메시지가 출력됩니다:

```
Server starting on :8080
```

#### 6. 헬스 체크

```bash
curl http://localhost:8080/health
```

응답:

```json
{
  "status": "healthy",
  "message": "UDONGGEUM API is running"
}
```

## API 엔드포인트

### 인증 (Authentication)

#### 회원가입
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "홍길동",
  "phone": "010-1234-5678"
}
```

#### 로그인
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### 내 정보 조회
```http
GET /api/v1/auth/me
Authorization: Bearer {access_token}
```

### 파일 업로드 (Upload)

#### 이미지 업로드
```http
POST /api/v1/upload/image
Authorization: Bearer {access_token}
Content-Type: multipart/form-data

file: [이미지 파일]
```

응답:
```json
{
  "message": "File uploaded successfully",
  "url": "http://localhost:8080/uploads/21c19f6f-0483-4e00-bbce-e2e94c0631f4.jpg",
  "filename": "21c19f6f-0483-4e00-bbce-e2e94c0631f4.jpg",
  "size_bytes": 1683987
}
```

**제약사항:**
- 지원 파일 형식: JPG, JPEG, PNG, GIF, WEBP
- 최대 파일 크기: 5MB
- 인증 필요 (Bearer Token)

**사용 예시 (curl):**
```bash
curl -X POST http://localhost:8080/api/v1/upload/image \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/image.jpg"
```

### 상품 (Products)

#### 전체 상품 조회
```http
GET /api/v1/products
```

#### 상품 필터(카테고리/재질) 조회
```http
GET /api/v1/products/filters
```

#### 상품 상세 조회
```http
GET /api/v1/products/:id
```

#### 상품 생성 (관리자 전용)
```http
POST /api/v1/products
Authorization: Bearer {admin_access_token}
Content-Type: application/json

{
  "name": "24K 골드바 100g",
  "description": "순도 99.99% 24K 골드바",
  "price": 8500000,
  "weight": 100,
  "purity": "24K",
  "category": "기타",
  "material": "금",
  "stock_quantity": 10,
  "image_url": "https://example.com/image.jpg",
  "store_id": 1
}
```

### 장바구니 (Cart)

#### 장바구니 조회
```http
GET /api/v1/cart
Authorization: Bearer {access_token}
```

#### 장바구니 추가
```http
POST /api/v1/cart
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "product_id": 1,
  "quantity": 2
}
```

#### 장바구니 수정
```http
PUT /api/v1/cart/:id
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "quantity": 3
}
```

#### 장바구니 삭제
```http
DELETE /api/v1/cart/:id
Authorization: Bearer {access_token}
```

### 주문 (Orders)

#### 주문 생성
```http
POST /api/v1/orders
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "shipping_address": "서울시 강남구 테헤란로 123"
}
```

#### 주문 내역 조회
```http
GET /api/v1/orders
Authorization: Bearer {access_token}
```

#### 주문 상세 조회
```http
GET /api/v1/orders/:id
Authorization: Bearer {access_token}
```

## 데이터베이스 스키마

### Users 테이블
- id (PK)
- email (unique)
- password_hash
- name
- phone
- role (user/admin)
- created_at, updated_at, deleted_at

### Products 테이블
- id (PK)
- name
- description
- price
- weight
- purity
- category
- stock_quantity
- image_url
- created_at, updated_at, deleted_at

### Orders 테이블
- id (PK)
- user_id (FK)
- total_amount
- status
- payment_status
- shipping_address
- created_at, updated_at, deleted_at

### OrderItems 테이블
- id (PK)
- order_id (FK)
- product_id (FK)
- quantity
- price
- created_at, deleted_at

### CartItems 테이블
- id (PK)
- user_id (FK)
- product_id (FK)
- quantity
- created_at, updated_at, deleted_at

## 개발

### 코드 포맷팅

```bash
go fmt ./...
```

### 테스트 실행

```bash
go test ./...
```

### 빌드

```bash
# Linux/Mac
go build -o bin/server cmd/server/main.go

# Windows
go build -o bin/server.exe cmd/server/main.go
```

## 배포

### Docker를 사용한 배포 (선택사항)

Dockerfile을 작성하여 컨테이너화할 수 있습니다.

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o server cmd/server/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["./server"]
```

## 보안 고려사항

- JWT Secret은 반드시 강력한 비밀키로 설정하세요
- 프로덕션 환경에서는 HTTPS를 사용하세요
- 데이터베이스 비밀번호를 안전하게 관리하세요
- .env 파일은 절대 Git에 커밋하지 마세요

## 트러블슈팅

### Go 설치 확인
```bash
go version
```

### PostgreSQL 연결 테스트
```bash
psql -h localhost -U postgres -d udonggeum
```

### 포트 충돌 확인
```bash
# Linux/Mac
lsof -i :8080

# Windows
netstat -ano | findstr :8080
```

## 라이선스

MIT License

## 기여

프로젝트에 기여하고 싶으시다면 Pull Request를 보내주세요.

## 문의

문제가 발생하거나 질문이 있으시면 Issue를 생성해주세요.

test
