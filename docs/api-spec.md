# UDONGGEUM API 명세

본 문서는 UDONGGEUM 백엔드(`api/v1`)에서 제공하는 주요 REST API를 정리한 명세입니다.  
모든 API는 `Content-Type: application/json`을 사용하며, 보호된 엔드포인트는 `Authorization: Bearer <AccessToken>` 헤더가 필요합니다.

## 공통 응답 형식

| 필드 | 타입 | 설명 |
| --- | --- | --- |
| `message` | string | 작업 성공 메시지 (성공 시) |
| `error` | string | 에러 메시지 (실패 시) |
| `details` | string | 추가 에러 정보 (선택적) |

---

## 인증 (Auth)

### 회원가입
`POST /api/v1/auth/register`

요청 필드:
- `email` *(string, required)*  
- `password` *(string, required, 최소 8자)*  
- `name` *(string, required)*  
- `phone` *(string, optional)*

응답 (201):
```json
{
  "message": "User registered successfully",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "Test User",
    "phone": "010-1234-5678",
    "role": "user"
  },
  "tokens": {
    "access_token": "<JWT>",
    "refresh_token": "<JWT>"
  }
}
```

### 로그인
`POST /api/v1/auth/login`

요청:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

응답 (200): 회원가입과 동일한 `user`, `tokens` 구조.

### 내 정보 조회
`GET /api/v1/auth/me` *(인증 필요)*

응답 (200):
```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "Test User",
    "phone": "010-1234-5678",
    "role": "user",
    "created_at": "...",
    "updated_at": "..."
  }
}
```

---

## 매장 (Stores)

### 매장 목록
`GET /api/v1/stores`

쿼리 파라미터:
- `region` *(string, optional)*  
- `district` *(string, optional)*  
- `search` *(string, optional)*  
- `include_products` *(bool, optional, default=false)*

응답 (200):
```json
{
  "count": 2,
  "stores": [
    {
      "id": 1,
      "user_id": 7,
      "name": "강동 우동금 주얼리",
      "region": "서울특별시",
      "district": "강동구",
      "address": "...",
      "phone_number": "02-1234-5678",
      "image_url": "...",
      "description": "...",
      "owner": {
        "id": 7,
        "email": "admin@example.com",
        "name": "관리자"
      },
      "products": [
        {
          "id": 10,
          "name": "24K 순금 반지",
          "price": 950000,
          "stock_quantity": 12,
          "category": "gold"
        }
      ]
    }
  ]
}
```

### 매장 지역 목록
`GET /api/v1/stores/locations`

응답:
```json
{
  "count": 3,
  "locations": [
    { "region": "서울특별시", "district": "강동구", "store_count": 2 },
    { "region": "서울특별시", "district": "강남구", "store_count": 1 }
  ]
}
```

### 매장 상세
`GET /api/v1/stores/:id`

- `include_products` 쿼리로 상품 목록 포함 여부 제어.
- 응답에는 매장을 소유한 관리자 계정 정보(`user_id`, `owner`)가 함께 제공됩니다.

### 매장 생성 *(관리자 전용)*
`POST /api/v1/stores`

- 헤더에 유효한 관리자 Access Token이 필요합니다.
- 요청 본문:
  ```json
  {
    "name": "신규 매장",
    "region": "서울특별시",
    "district": "송파구",
    "address": "서울시 송파구 ...",
    "phone_number": "02-0000-0000",
    "image_url": "https://...",
    "description": "매장 소개"
  }
  ```
- 호출자의 사용자 ID가 자동으로 `user_id`에 매핑됩니다.
- 성공 시 (201):
  ```json
  {
    "message": "Store created successfully",
    "store": {
      "id": 3,
      "user_id": 7,
      "name": "신규 매장",
      "region": "서울특별시",
      "district": "송파구",
      "address": "...",
      "phone_number": "02-0000-0000",
      "image_url": "...",
      "description": "...",
      "created_at": "...",
      "updated_at": "..."
    }
  }
  ```

### 매장 수정 *(관리자 전용, 소유자 한정)*
`PUT /api/v1/stores/:id`

- 요청 본문은 생성과 동일합니다.
- 해당 매장의 `user_id`와 현재 토큰의 사용자 ID가 일치하지 않으면 `403 Insufficient permissions`가 반환됩니다.

### 매장 삭제 *(관리자 전용, 소유자 한정)*
`DELETE /api/v1/stores/:id`

- 성공 시 `200 OK` 와 `{"message":"Store deleted successfully"}` 반환.
- 소유자가 아닌 경우 `403`, 존재하지 않는 경우 `404`.

---

## 상품 (Products)

### 상품 목록 필터링
`GET /api/v1/products`

쿼리 파라미터:
- `region`, `district` *(string)* : 매장 위치 필터
- `category` *(enum: gold|silver|jewelry)*
- `store_id` *(number)*
- `search` *(string)* : 이름/설명 검색
- `sort` *(string)* : `popularity`(default) / `price_asc` / `price_desc` / `latest`
- `popular_only` *(bool)*
- `include_options` *(bool)*
- `page`, `page_size` *(number)* : 페이지네이션

응답:
```json
{
  "count": 12,
  "page_size": 20,
  "offset": 0,
  "products": [
    {
      "id": 1,
      "name": "24K 순금 반지",
      "price": 950000,
      "category": "gold",
      "stock_quantity": 12,
      "popularity_score": 92,
      "store": {
        "id": 1,
        "name": "강동 우동금 주얼리"
      },
      "options": [
        { "id": 101, "name": "사이즈", "value": "9호", "additional_price": 0, "stock_quantity": 4, "is_default": true }
      ]
    }
  ]
}
```

### 인기 상품
`GET /api/v1/products/popular?category=gold&region=서울특별시&district=강동구&limit=4`

### 상품 상세
`GET /api/v1/products/:id`

### 상품 생성/수정/삭제 *(관리자 권한 필요)*
- `POST /api/v1/products`
- `PUT /api/v1/products/:id`
- `DELETE /api/v1/products/:id`

요청 필드:
| 필드 | 타입 | 필수 | 설명 |
| --- | --- | --- | --- |
| `name` | string | Y | |
| `price` | number | Y | 0보다 커야 함 |
| `category` | string | Y | `gold`, `silver`, `jewelry` |
| `store_id` | number | Y* | 상품을 보유한 매장 (생성 시 필수) |
| `stock_quantity` | number | N | 기본값 0 |
| `description`, `weight`, `purity`, `image_url`, `popularity_score` | optional |

- **권한 규칙**
  - 상품 생성은 본인이 소유한 매장(`stores.user_id = token user`)에 대해서만 가능합니다.
  - 상품 수정은 소유자가 동일한 경우에만 가능하며, `store_id`를 다른 매장으로 변경할 수 없습니다.
  - 상품 삭제 역시 매장 소유자만 수행할 수 있습니다.
  - 위 조건을 만족하지 않으면 `403 Insufficient permissions` 에러가 반환됩니다.

---

## 장바구니 (Cart)

모든 엔드포인트 인증 필요.

### 장바구니 조회
`GET /api/v1/cart`

응답:
```json
{
  "count": 1,
  "total": 1900000,
  "cart_items": [
    {
      "id": 11,
      "quantity": 2,
      "product": {
        "id": 1,
        "name": "24K 순금 반지",
        "price": 950000,
        "store": { "id": 1, "name": "강동 우동금 주얼리" }
      },
      "product_option": {
        "id": 101,
        "name": "사이즈",
        "value": "11호",
        "additional_price": 20000
      }
    }
  ]
}
```

### 장바구니 담기
`POST /api/v1/cart`

요청:
```json
{
  "product_id": 1,
  "product_option_id": 101, // 선택 사항 (없으면 null)
  "quantity": 2
}
```

응답 (201):
```json
{ "message": "Item added to cart successfully" }
```

에러 코드:
- 401: 인증 실패
- 404: 상품/옵션 없음 (`Product not found`, `Invalid product option`)
- 400: 재고 부족 (`Insufficient stock`)

### 장바구니 수량 변경
`PUT /api/v1/cart/:cart_item_id`

요청:
```json
{ "quantity": 3 }
```

### 장바구니 아이템 삭제
`DELETE /api/v1/cart/:cart_item_id`

### 장바구니 비우기
`DELETE /api/v1/cart`

---

## 주문 (Orders)

### 주문 생성
`POST /api/v1/orders`

요청 필드:
| 필드 | 타입 | 필수 | 설명 |
| --- | --- | --- | --- |
| `shipping_address` | string | 조건부 | 배송 주문일 때 필수 |
| `fulfillment_type` | string | N | `delivery`(default) / `pickup` |
| `pickup_store_id` | number | 조건부 | 픽업 주문일 때 필수 (미지정 시 장바구니 첫 상품의 매장 사용) |

응답 (201):
```json
{
  "message": "Order created successfully",
  "order": {
    "id": 21,
    "user_id": 1,
    "total_amount": 1900000,
    "fulfillment_type": "delivery",
    "shipping_address": "서울시 강남구 ...",
    "order_items": [
      {
        "id": 33,
        "product_id": 1,
        "product_option_id": 101,
        "store_id": 1,
        "quantity": 2,
        "price": 960000,
        "option_snapshot": "사이즈: 11호"
      }
    ]
  }
}
```

에러:
- 400: 장바구니 비어있음 (`Cart is empty`)
- 400: 재고 부족 (`Insufficient stock for one or more items`)
- 400: 주문 방식 오류 (`Invalid fulfillment selection`)
- 400: 옵션 오류 (`Invalid product option`)

### 주문 목록
`GET /api/v1/orders`

### 주문 상세
`GET /api/v1/orders/:id`

### 주문/결제 상태 변경 *(관리자)*
- `PUT /api/v1/orders/:id/status`  
- `PUT /api/v1/orders/:id/payment`

요청:
```json
{ "status": "confirmed" } // 또는 payment_status: completed
```

---

## 응답 예시 요약

| 엔드포인트 | 정상 코드 | 주요 에러 코드 |
| --- | --- | --- |
| `POST /auth/register` | 201 | 400 (유효성), 409 (중복) |
| `POST /auth/login` | 200 | 401 (인증 실패) |
| `GET /stores` | 200 | - |
| `GET /products` | 200 | - |
| `POST /products` | 201 | 401, 403, 400 |
| `POST /cart` | 201 | 401, 404, 400 |
| `POST /orders` | 201 | 400, 401 |
| `GET /orders` | 200 | 401 |

---

## 인증 흐름 요약

1. `POST /auth/register` → 회원 생성 + 토큰 발급  
2. `POST /auth/login` → Access/Refresh 토큰 발급  
3. 보호된 API 접근 시 `Authorization: Bearer <AccessToken>` 헤더 전달  
4. 토큰 만료 시(추후 구현 예정) Refresh Token을 사용해 재발급  

---

## 용어

- **Access Token**: 15분 기본 유효기간 JWT  
- **Refresh Token**: 7일 기본 유효기간 JWT  
- **Fulfillment Type**: `delivery`(배송) 또는 `pickup`(매장 픽업)  
- **Product Option**: 사이즈, 길이 등 추가 금액이 붙을 수 있는 상품 옵션  
- **Store**: 상품이 속한 오프라인 매장 정보  

---

## 버전 정보

- API 버전: `v1`  
- 문서 버전: 2025-10-20 (작성일 기준)  
- 담당: Backend Team (@ikkim)
