# 금광산 커뮤니티 API 문서

## 개요

우동금 플랫폼의 커뮤니티 기능 "금광산"을 위한 REST API 문서입니다.

### 카테고리 구조

```
금광산 (Community)
├── 금거래 (gold_trade)
│   ├── 금 매수 (sell_gold) - 일반 사용자: 내 금 팔기
│   └── 금 매입 (buy_gold)  - 금은방 사장님: 금 매입 홍보
├── 금소식 (gold_news)
│   ├── 뉴스 (news)
│   ├── 후기 (review)
│   └── 팁 (tip)
└── QnA (qna)
    ├── 질문 (question)
    └── FAQ (faq) - 관리자만 작성 가능
```

### 권한 체계

- **일반 사용자 (user)**
  - 금 매수 글 작성 가능 (내 금 팔기)
  - 금소식 글 작성 가능
  - QnA 질문/답변 작성 가능

- **금은방 사장님 (admin)**
  - 일반 사용자 권한 + 추가:
  - 금 매입 홍보 글 작성 가능 (매장 ID 필수)
  - FAQ 작성 가능
  - 모든 게시글/댓글 관리 가능

## API 엔드포인트

Base URL: `/api/v1/community`

### 1. 게시글 (Posts)

#### 1.1 게시글 목록 조회

```http
GET /api/v1/community/posts
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| category | string | No | 카테고리 필터 (`gold_trade`, `gold_news`, `qna`) |
| type | string | No | 게시글 타입 필터 |
| status | string | No | 상태 필터 (`active`, `inactive`) |
| user_id | integer | No | 작성자 ID 필터 |
| store_id | integer | No | 매장 ID 필터 |
| is_answered | boolean | No | 답변 완료 여부 (QnA) |
| search | string | No | 검색어 (제목+내용) |
| page | integer | No | 페이지 번호 (default: 1) |
| page_size | integer | No | 페이지 크기 (default: 20, max: 100) |
| sort_by | string | No | 정렬 기준 (`created_at`, `view_count`, `like_count`, `comment_count`) |
| sort_order | string | No | 정렬 순서 (`asc`, `desc`) |

**Response Example:**

```json
{
  "data": [
    {
      "id": 1,
      "title": "24K 금반지 판매합니다",
      "content": "순도 99.9% 금반지 5돈 판매합니다...",
      "category": "gold_trade",
      "type": "sell_gold",
      "status": "active",
      "user_id": 2,
      "user": {
        "id": 2,
        "name": "홍길동",
        "email": "hong@example.com"
      },
      "gold_type": "24K",
      "weight": 18.75,
      "price": 3500000,
      "location": "서울 강남구",
      "view_count": 123,
      "like_count": 5,
      "comment_count": 3,
      "image_urls": ["https://..."],
      "created_at": "2025-12-08T10:00:00Z",
      "updated_at": "2025-12-08T10:00:00Z"
    }
  ],
  "total": 45,
  "page": 1,
  "page_size": 20
}
```

#### 1.2 게시글 상세 조회

```http
GET /api/v1/community/posts/{id}
```

**Response:**

```json
{
  "data": {
    "id": 1,
    "title": "24K 금반지 판매합니다",
    "content": "...",
    "category": "gold_trade",
    "type": "sell_gold",
    "user": { ... },
    "store": null,
    "comments": [
      {
        "id": 1,
        "content": "연락 드렸습니다!",
        "user": { ... },
        "parent_id": null,
        "replies": [],
        "like_count": 2,
        "created_at": "..."
      }
    ],
    "view_count": 124,
    "like_count": 5,
    "comment_count": 3
  },
  "is_liked": false
}
```

#### 1.3 게시글 작성

```http
POST /api/v1/community/posts
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "title": "24K 금반지 판매합니다",
  "content": "순도 99.9% 금반지 5돈 판매합니다. 깨끗한 상태입니다.",
  "category": "gold_trade",
  "type": "sell_gold",
  "gold_type": "24K",
  "weight": 18.75,
  "price": 3500000,
  "location": "서울 강남구",
  "image_urls": ["https://cdn.udonggeum.com/images/..."]
}
```

**금은방 사장님 매입 글 예시:**

```json
{
  "title": "금 고가 매입합니다",
  "content": "모든 금 제품 최고가 매입! 방문 환영합니다.",
  "category": "gold_trade",
  "type": "buy_gold",
  "store_id": 1,
  "gold_type": "24K, 18K, 14K",
  "price": 0,
  "location": "서울 강남구 테헤란로 231"
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "title": "24K 금반지 판매합니다",
  "content": "...",
  "category": "gold_trade",
  "type": "sell_gold",
  "user_id": 2,
  "status": "active",
  "created_at": "2025-12-08T10:00:00Z"
}
```

#### 1.4 게시글 수정

```http
PUT /api/v1/community/posts/{id}
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "title": "수정된 제목",
  "content": "수정된 내용",
  "status": "active",
  "price": 3300000
}
```

**Response:** `200 OK`

#### 1.5 게시글 삭제

```http
DELETE /api/v1/community/posts/{id}
Authorization: Bearer {token}
```

**Response:** `204 No Content`

#### 1.6 게시글 좋아요 토글

```http
POST /api/v1/community/posts/{id}/like
Authorization: Bearer {token}
```

**Response:**

```json
{
  "is_liked": true
}
```

---

### 2. 댓글 (Comments)

#### 2.1 댓글 목록 조회

```http
GET /api/v1/community/comments?post_id={post_id}
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| post_id | integer | Yes | 게시글 ID |
| parent_id | integer | No | 부모 댓글 ID (null이면 최상위 댓글만) |
| page | integer | No | 페이지 번호 (default: 1) |
| page_size | integer | No | 페이지 크기 (default: 50, max: 100) |
| sort_by | string | No | 정렬 기준 (`created_at`, `like_count`) |
| sort_order | string | No | 정렬 순서 (`asc`, `desc`) |

**Response:**

```json
{
  "data": [
    {
      "id": 1,
      "content": "연락 드렸습니다!",
      "user": {
        "id": 3,
        "name": "김철수",
        "email": "kim@example.com"
      },
      "post_id": 1,
      "parent_id": null,
      "is_answer": false,
      "is_accepted": false,
      "like_count": 2,
      "replies": [
        {
          "id": 2,
          "content": "네, 확인했습니다!",
          "user": { ... },
          "parent_id": 1,
          "like_count": 0
        }
      ],
      "created_at": "2025-12-08T11:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 50
}
```

#### 2.2 댓글 작성

```http
POST /api/v1/community/comments
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "content": "연락 드렸습니다!",
  "post_id": 1,
  "parent_id": null,
  "is_answer": false
}
```

**대댓글 작성:**

```json
{
  "content": "답변 감사합니다",
  "post_id": 1,
  "parent_id": 1
}
```

**QnA 답변 작성:**

```json
{
  "content": "24K 금반지는 순도가 99.9%로...",
  "post_id": 5,
  "is_answer": true
}
```

**Response:** `201 Created`

#### 2.3 댓글 수정

```http
PUT /api/v1/community/comments/{id}
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "content": "수정된 댓글 내용"
}
```

**Response:** `200 OK`

#### 2.4 댓글 삭제

```http
DELETE /api/v1/community/comments/{id}
Authorization: Bearer {token}
```

**Response:** `204 No Content`

#### 2.5 댓글 좋아요 토글

```http
POST /api/v1/community/comments/{id}/like
Authorization: Bearer {token}
```

**Response:**

```json
{
  "is_liked": true
}
```

---

### 3. QnA 특수 기능

#### 3.1 답변 채택

```http
POST /api/v1/community/posts/{post_id}/accept/{comment_id}
Authorization: Bearer {token}
```

**설명:** QnA 게시글 작성자만 답변을 채택할 수 있습니다.

**Response:**

```json
{
  "message": "answer accepted successfully"
}
```

---

## 에러 응답

### 400 Bad Request

```json
{
  "error": "invalid request parameters"
}
```

### 401 Unauthorized

```json
{
  "error": "unauthorized"
}
```

### 403 Forbidden

```json
{
  "error": "permission denied"
}
```

### 404 Not Found

```json
{
  "error": "post not found"
}
```

---

## 사용 예시

### 예시 1: 금거래 - 일반 사용자가 금 판매 글 작성

```bash
curl -X POST https://api.udonggeum.com/api/v1/community/posts \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "18K 금목걸이 판매합니다",
    "content": "사용감 거의 없는 18K 금목걸이입니다. 3돈 정도 됩니다.",
    "category": "gold_trade",
    "type": "sell_gold",
    "gold_type": "18K",
    "weight": 11.25,
    "price": 2000000,
    "location": "부산 해운대구",
    "image_urls": ["https://cdn.udonggeum.com/images/necklace.jpg"]
  }'
```

### 예시 2: 금거래 - 사장님이 매입 홍보 글 작성

```bash
curl -X POST https://api.udonggeum.com/api/v1/community/posts \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "서울 강남 금은방 - 금 최고가 매입",
    "content": "모든 금 제품 최고가로 매입합니다! 방문 환영합니다.",
    "category": "gold_trade",
    "type": "buy_gold",
    "store_id": 1,
    "gold_type": "24K, 18K, 14K",
    "location": "서울 강남구 테헤란로 231"
  }'
```

### 예시 3: 금소식 - 금 시세 정보 공유

```bash
curl -X POST https://api.udonggeum.com/api/v1/community/posts \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "2025년 1월 금 시세 전망",
    "content": "새해 들어 금 시세가 상승세를 보이고 있습니다...",
    "category": "gold_news",
    "type": "news"
  }'
```

### 예시 4: QnA - 질문 작성 및 답변 채택

**질문 작성:**

```bash
curl -X POST https://api.udonggeum.com/api/v1/community/posts \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "24K와 18K 금의 차이가 뭔가요?",
    "content": "금을 처음 사려고 하는데, 24K와 18K의 차이를 모르겠습니다.",
    "category": "qna",
    "type": "question"
  }'
```

**답변 작성:**

```bash
curl -X POST https://api.v1/community/comments \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "24K는 순금(99.9%), 18K는 금 75% + 합금 25%입니다...",
    "post_id": 10,
    "is_answer": true
  }'
```

**답변 채택:**

```bash
curl -X POST https://api.udonggeum.com/api/v1/community/posts/10/accept/5 \
  -H "Authorization: Bearer {token}"
```

### 예시 5: 게시글 목록 필터링

**금거래 - 매수 글만 조회:**

```bash
curl "https://api.udonggeum.com/api/v1/community/posts?category=gold_trade&type=sell_gold&page=1&page_size=20"
```

**특정 매장의 매입 글 조회:**

```bash
curl "https://api.udonggeum.com/api/v1/community/posts?category=gold_trade&type=buy_gold&store_id=1"
```

**미답변 QnA 조회:**

```bash
curl "https://api.udonggeum.com/api/v1/community/posts?category=qna&is_answered=false"
```

**검색:**

```bash
curl "https://api.udonggeum.com/api/v1/community/posts?search=금반지&page=1"
```

---

## 데이터베이스 스키마

### community_posts

| Column | Type | Description |
|--------|------|-------------|
| id | uint | 게시글 ID (PK) |
| title | varchar(200) | 제목 |
| content | text | 내용 |
| category | varchar(20) | 카테고리 |
| type | varchar(20) | 게시글 타입 |
| status | varchar(20) | 상태 |
| user_id | uint | 작성자 ID (FK) |
| store_id | uint | 매장 ID (FK, nullable) |
| gold_type | varchar(50) | 금 종류 (nullable) |
| weight | float | 중량 (g, nullable) |
| price | bigint | 가격 (원, nullable) |
| location | varchar(100) | 지역 (nullable) |
| is_answered | boolean | 답변 완료 여부 |
| accepted_answer_id | uint | 채택된 답변 ID (nullable) |
| view_count | int | 조회수 |
| like_count | int | 좋아요 수 |
| comment_count | int | 댓글 수 |
| image_urls | text[] | 이미지 URL 배열 |
| created_at | timestamp | 생성 시각 |
| updated_at | timestamp | 수정 시각 |
| deleted_at | timestamp | 삭제 시각 (soft delete) |

### community_comments

| Column | Type | Description |
|--------|------|-------------|
| id | uint | 댓글 ID (PK) |
| content | text | 댓글 내용 |
| user_id | uint | 작성자 ID (FK) |
| post_id | uint | 게시글 ID (FK) |
| parent_id | uint | 부모 댓글 ID (FK, nullable) |
| is_answer | boolean | 답변 여부 (QnA) |
| is_accepted | boolean | 채택 여부 |
| like_count | int | 좋아요 수 |
| created_at | timestamp | 생성 시각 |
| updated_at | timestamp | 수정 시각 |
| deleted_at | timestamp | 삭제 시각 (soft delete) |

### post_likes

| Column | Type | Description |
|--------|------|-------------|
| id | uint | 좋아요 ID (PK) |
| post_id | uint | 게시글 ID (FK) |
| user_id | uint | 사용자 ID (FK) |
| created_at | timestamp | 생성 시각 |

**Unique Constraint:** (post_id, user_id)

### comment_likes

| Column | Type | Description |
|--------|------|-------------|
| id | uint | 좋아요 ID (PK) |
| comment_id | uint | 댓글 ID (FK) |
| user_id | uint | 사용자 ID (FK) |
| created_at | timestamp | 생성 시각 |

**Unique Constraint:** (comment_id, user_id)

---

## 주의사항

1. **권한 관리**
   - `buy_gold` 타입 게시글은 `admin` 권한 필요
   - `faq` 타입 게시글은 `admin` 권한 필요
   - 본인 게시글/댓글만 수정/삭제 가능 (관리자 예외)

2. **데이터 검증**
   - 금거래 게시글 작성 시 금 타입, 중량, 가격은 선택 사항
   - 사장님 매입 글(`buy_gold`)은 `store_id` 필수

3. **성능 최적화**
   - 게시글 목록 조회 시 댓글은 포함되지 않음
   - 댓글은 별도 API로 조회
   - 페이지네이션 필수 사용 권장

4. **검색 기능**
   - `search` 파라미터는 제목과 내용을 모두 검색 (ILIKE 사용)
   - PostgreSQL의 Full-Text Search 활용 가능

---

## 향후 개선 사항

- [ ] 게시글 신고 기능
- [ ] 이미지 업로드 API
- [ ] 알림 기능 (댓글, 좋아요, 답변 채택)
- [ ] 통계 API (인기 게시글, 트렌드)
- [ ] 태그 시스템
- [ ] 북마크 기능
