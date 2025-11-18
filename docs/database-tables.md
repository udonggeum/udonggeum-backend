# Database Tables

## users

| Column       | Type      | Constraints                          | Description           |
| ------------ | --------- | ------------------------------------ | --------------------- |
| id           | uint      | primary key                          | 사용자 ID              |
| email        | string    | unique, not null                     | 이메일                 |
| password_hash| string    | not null                             | 비밀번호 해시          |
| name         | string    | not null                             | 이름                   |
| phone        | string    | nullable                             | 전화번호               |
| role         | varchar(20) | default `'user'`                   | 권한                   |
| created_at   | timestamp | auto-managed                        | 생성 시각              |
| updated_at   | timestamp | auto-managed                        | 수정 시각              |
| deleted_at   | timestamp | indexed, soft delete                 | 삭제 시각(소프트 삭제) |

**Relations**
- Has many `orders` (users.id → orders.user_id)
- Has many `cart_items` (users.id → cart_items.user_id)
- Has many `stores` (users.id → stores.user_id)

**Enums**
- `role`: `user`, `admin`

## stores

| Column       | Type        | Constraints            | Description           |
| ------------ | ----------- | ---------------------- | --------------------- |
| id           | uint        | primary key            | 고유 매장 ID           |
| user_id      | uint        | not null, indexed      | 매장 소유자 ID         |
| name         | string      | not null               | 매장명                 |
| region       | string      | indexed, not null      | 시·도                  |
| district     | string      | indexed, not null      | 구·군                  |
| address      | text        |                        | 상세 주소              |
| phone_number | varchar(30) |                        | 연락처                 |
| image_url    | string      |                        | 매장 이미지            |
| description  | text        |                        | 매장 소개              |
| created_at   | timestamp   | auto-managed           | 생성 시각              |
| updated_at   | timestamp   | auto-managed           | 수정 시각              |
| deleted_at   | timestamp   | indexed, soft delete   | 삭제 시각(소프트 삭제) |

**Relations**
- Belongs to `users` (stores.user_id → users.id)
- Has many `products` (stores.id → products.store_id)
- Referenced by `orders` via `pickup_store_id`
- Referenced by `order_items` via `store_id`

## products

| Column           | Type           | Constraints                                         | Description           |
| ---------------- | -------------- | --------------------------------------------------- | --------------------- |
| id               | uint           | primary key                                         | 고유 상품 ID           |
| name             | string         | not null                                            | 상품명                 |
| description      | text           |                                                     | 상품 설명              |
| price            | float          | not null                                            | 기본 판매가            |
| weight           | float          |                                                     | 중량(그램 등)          |
| purity           | string         |                                                     | 금속 순도              |
| category         | varchar(50)    |                                                     | 상품 카테고리          |
| stock_quantity   | int            | default `0`                                         | 기본 재고 수량         |
| image_url        | string         |                                                     | 대표 이미지 경로       |
| store_id         | uint           | not null, indexed                                   | 소속 매장 ID           |
| view_count       | int            | default `0`                                         | 조회수                 |
| created_at       | timestamp      | auto-managed                                        | 생성 시각              |
| updated_at       | timestamp      | auto-managed                                        | 수정 시각              |
| deleted_at       | timestamp      | indexed, soft delete                                | 삭제 시각(소프트 삭제) |

**Relations**
- Belongs to `stores` (products.store_id → stores.id)
- Has many `order_items` (products.id → order_items.product_id)
- Has many `cart_items` (products.id → cart_items.product_id)
- Has many `product_options` (products.id → product_options.product_id)

**Enums**
- `category`: `gold`, `silver`, `jewelry`

## product_options

| Column           | Type      | Constraints            | Description           |
| ---------------- | --------- | ---------------------- | --------------------- |
| id               | uint      | primary key            | 고유 옵션 ID           |
| product_id       | uint      | not null, indexed      | 소속 상품 ID           |
| name             | string    | not null               | 옵션 그룹명            |
| value            | string    | not null               | 옵션 값                |
| additional_price | float     | default `0`            | 추가 금액              |
| stock_quantity   | int       | default `0`            | 옵션 재고              |
| image_url        | string    |                        | 옵션 이미지            |
| is_default       | bool      | default `false`        | 기본 옵션 여부         |
| created_at       | timestamp | auto-managed           | 생성 시각              |
| updated_at       | timestamp | auto-managed           | 수정 시각              |
| deleted_at       | timestamp | indexed, soft delete   | 삭제 시각(소프트 삭제) |

**Relations**
- Belongs to `products` (product_options.product_id → products.id)
- Referenced by `cart_items` and `order_items` via `product_option_id`

## orders

| Column            | Type           | Constraints                                     | Description           |
| ----------------- | -------------- | ----------------------------------------------- | --------------------- |
| id                | uint           | primary key                                     | 주문 ID                |
| user_id           | uint           | not null, indexed                               | 주문자 ID              |
| total_amount      | float          | not null                                        | 총 결제 금액           |
| total_price       | float          | not null                                        | 총 상품 금액           |
| status            | varchar(20)    | default `'pending'`                             | 주문 상태              |
| payment_status    | varchar(20)    | default `'pending'`                             | 결제 상태              |
| fulfillment_type  | varchar(20)    | default `'delivery'`                            | 이행 방식              |
| shipping_address  | text           |                                                 | 배송지 주소            |
| pickup_store_id   | uint           | indexed, nullable                               | 픽업 매장 ID           |
| created_at        | timestamp      | auto-managed                                    | 생성 시각              |
| updated_at        | timestamp      | auto-managed                                    | 수정 시각              |
| deleted_at        | timestamp      | indexed, soft delete                            | 삭제 시각(소프트 삭제) |

**Relations**
- Belongs to `users` (orders.user_id → users.id)
- Belongs to `stores` via `pickup_store_id`
- Has many `order_items` (orders.id → order_items.order_id)

**Enums**
- `status`: `pending`, `confirmed`, `shipping`, `delivered`, `cancelled`
- `payment_status`: `pending`, `completed`, `failed`, `refunded`
- `fulfillment_type`: `delivery`, `pickup`

## order_items

| Column             | Type      | Constraints          | Description           |
| ------------------ | --------- | -------------------- | --------------------- |
| id                 | uint      | primary key          | 주문 항목 ID           |
| order_id           | uint      | not null, indexed    | 주문 ID                |
| product_id         | uint      | not null, indexed    | 상품 ID                |
| product_option_id  | uint      | indexed, nullable    | 선택 옵션 ID           |
| store_id           | uint      | not null, indexed    | 매장 ID                |
| quantity           | int       | not null             | 수량                   |
| price              | float     | not null             | 단가                   |
| option_snapshot    | text      |                      | 옵션 정보 스냅샷       |
| created_at         | timestamp | auto-managed         | 생성 시각              |
| deleted_at         | timestamp | indexed, soft delete | 삭제 시각(소프트 삭제) |

**Relations**
- Belongs to `orders` (order_items.order_id → orders.id)
- Belongs to `products` (order_items.product_id → products.id)
- Optional belongs to `product_options` via `product_option_id`
- Belongs to `stores` (order_items.store_id → stores.id)

## cart_items

| Column             | Type      | Constraints            | Description           |
| ------------------ | --------- | ---------------------- | --------------------- |
| id                 | uint      | primary key            | 장바구니 항목 ID        |
| user_id            | uint      | not null, indexed      | 사용자 ID               |
| product_id         | uint      | not null, indexed      | 상품 ID                 |
| product_option_id  | uint      | indexed, nullable      | 선택 옵션 ID            |
| quantity           | int       | not null, default `1`  | 수량                    |
| created_at         | timestamp | auto-managed           | 생성 시각               |
| updated_at         | timestamp | auto-managed           | 수정 시각               |
| deleted_at         | timestamp | indexed, soft delete   | 삭제 시각(소프트 삭제)  |

**Relations**
- Belongs to `users` (cart_items.user_id → users.id)
- Belongs to `products` (cart_items.product_id → products.id)
- Optional belongs to `product_options` via `product_option_id`
