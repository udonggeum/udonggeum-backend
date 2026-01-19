# Database Migrations

## 개요

이 디렉토리는 데이터베이스 스키마 변경을 위한 migration 파일들을 포함합니다.

## 최신 Migration: Partial Unique Indexes (2026-01-19)

### 목적
Soft delete와 unique constraint의 충돌 문제를 해결하기 위해 partial unique index를 적용합니다.

### 문제 상황
- Soft delete는 데이터를 실제로 삭제하지 않고 `deleted_at` 필드만 설정
- 기존 unique constraint는 삭제된 데이터도 포함하여 중복 체크
- 결과: 삭제된 데이터의 unique 값을 재사용할 수 없음

**예시:**
```sql
-- 사업자번호 "1234567890"로 매장 생성
INSERT INTO stores (business_number) VALUES ('1234567890');

-- 매장 삭제 (soft delete)
UPDATE stores SET deleted_at = NOW() WHERE business_number = '1234567890';

-- 같은 사업자번호로 새 매장 생성 시도
INSERT INTO stores (business_number) VALUES ('1234567890');
-- ❌ ERROR: duplicate key value violates unique constraint
```

### 해결 방법
PostgreSQL의 partial unique index를 사용하여 `deleted_at IS NULL`인 행만 unique 체크

```sql
CREATE UNIQUE INDEX idx_stores_business_number
ON stores(business_number)
WHERE deleted_at IS NULL;  -- 삭제되지 않은 행만 체크
```

### 적용 대상

#### 1. Stores 테이블
- `business_number`: 상가업소번호/사업자번호
- `slug`: URL용 고유 식별자

#### 2. Users 테이블
- `email`: 이메일 주소
- `nickname`: 닉네임

#### 3. Business Registrations 테이블
- `store_id`: 매장 ID (1:1 관계)

## Migration 실행 방법

### 1. 사전 준비

**중복 데이터 확인:**
```sql
-- 활성 매장 중 중복 business_number 확인
SELECT business_number, COUNT(*)
FROM stores
WHERE deleted_at IS NULL
GROUP BY business_number
HAVING COUNT(*) > 1;

-- 활성 사용자 중 중복 email 확인
SELECT email, COUNT(*)
FROM users
WHERE deleted_at IS NULL
GROUP BY email
HAVING COUNT(*) > 1;

-- 활성 사용자 중 중복 nickname 확인
SELECT nickname, COUNT(*)
FROM users
WHERE deleted_at IS NULL
GROUP BY nickname
HAVING COUNT(*) > 1;

-- 활성 business_registration 중 중복 store_id 확인
SELECT store_id, COUNT(*)
FROM business_registrations
WHERE deleted_at IS NULL
GROUP BY store_id
HAVING COUNT(*) > 1;
```

**중요:** 중복 데이터가 있으면 migration이 실패합니다. 먼저 중복 데이터를 정리해야 합니다.

### 2. Migration 실행

```bash
# PostgreSQL에 연결
psql -U your_username -d your_database

# Migration 파일 실행
\i db/migrations/20260119_add_partial_unique_indexes.sql

# 또는 직접 실행
psql -U your_username -d your_database -f db/migrations/20260119_add_partial_unique_indexes.sql
```

### 3. 검증

```sql
-- 생성된 인덱스 확인
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename IN ('stores', 'users', 'business_registrations')
    AND indexname LIKE 'idx_%'
ORDER BY tablename, indexname;

-- 예상 결과:
-- idx_business_registrations_store_id
-- idx_stores_business_number
-- idx_stores_slug
-- idx_users_email
-- idx_users_nickname
```

### 4. 테스트

```sql
-- 테스트 1: 매장 생성 → 삭제 → 재생성
BEGIN;

-- 매장 생성
INSERT INTO stores (business_number, name, region, district, slug, created_at, updated_at)
VALUES ('9999999999', '테스트매장', '서울', '강남구', 'test-store', NOW(), NOW())
RETURNING id;

-- 매장 삭제 (soft delete)
UPDATE stores SET deleted_at = NOW() WHERE business_number = '9999999999';

-- 같은 사업자번호로 재생성 (성공해야 함)
INSERT INTO stores (business_number, name, region, district, slug, created_at, updated_at)
VALUES ('9999999999', '새로운매장', '서울', '강남구', 'new-store', NOW(), NOW());

-- 정리
ROLLBACK;
```

## 코드 변경 사항

### 1. GORM 모델 태그 수정

**Before:**
```go
type Store struct {
    BusinessNumber string `gorm:"uniqueIndex;type:varchar(50)"`
    Slug           string `gorm:"uniqueIndex"`
}

type User struct {
    Email    string `gorm:"uniqueIndex;not null"`
    Nickname string `gorm:"uniqueIndex;not null"`
}

type BusinessRegistration struct {
    StoreID uint `gorm:"not null;uniqueIndex"`
}
```

**After:**
```go
type Store struct {
    BusinessNumber string `gorm:"type:varchar(50);index"`  // DB partial index로 관리
    Slug           string `gorm:"index"`  // DB partial index로 관리
}

type User struct {
    Email    string `gorm:"index;not null"`  // DB partial index로 관리
    Nickname string `gorm:"index;not null"`  // DB partial index로 관리
}

type BusinessRegistration struct {
    StoreID uint `gorm:"not null;index"`  // DB partial index로 관리
}
```

### 2. Store 삭제 로직 개선

연관 데이터(BusinessRegistration)도 함께 soft delete하도록 트랜잭션 처리:

```go
func (s *storeService) DeleteStore(userID uint, storeID uint) error {
    tx := s.db.Begin()

    // Store soft delete
    tx.Delete(&Store{}, storeID)

    // BusinessRegistration soft delete
    tx.Where("store_id = ?", storeID).Delete(&BusinessRegistration{})

    return tx.Commit().Error
}
```

## 주의사항

### 1. Soft Delete 복구 시 주의

삭제된 데이터를 복구할 때 같은 unique 값을 가진 활성 데이터가 있으면 충돌합니다.

```go
// 복구 전 충돌 체크 필요
func RestoreStore(storeID uint) error {
    var deletedStore Store
    db.Unscoped().First(&deletedStore, storeID)

    // 같은 business_number를 가진 활성 매장 확인
    var activeStore Store
    err := db.Where("business_number = ?", deletedStore.BusinessNumber).
           First(&activeStore).Error

    if err == nil {
        return fmt.Errorf("충돌: 해당 사업자번호를 사용하는 활성 매장이 있습니다")
    }

    // 복구
    db.Model(&deletedStore).Update("deleted_at", nil)
    return nil
}
```

### 2. 닉네임/이메일 재사용 정책 (선택적)

민감한 식별자는 재사용 방지 기간을 설정할 수 있습니다:

```go
func CheckNicknameAvailability(nickname string) error {
    // 30일 이내 삭제된 닉네임은 재사용 불가
    var recentlyDeleted User
    err := db.Unscoped().
           Where("nickname = ? AND deleted_at IS NOT NULL", nickname).
           Where("deleted_at > ?", time.Now().Add(-30*24*time.Hour)).
           First(&recentlyDeleted).Error

    if err == nil {
        return fmt.Errorf("이 닉네임은 30일 후에 사용 가능합니다")
    }
    return nil
}
```

### 3. 삭제된 데이터 조회

삭제된 데이터를 조회할 때는 partial index를 사용할 수 없으므로 성능에 주의:

```sql
-- 활성 데이터: 인덱스 사용 (빠름)
SELECT * FROM stores WHERE business_number = '1234567890' AND deleted_at IS NULL;

-- 삭제된 데이터: 인덱스 미사용 (느림)
SELECT * FROM stores WHERE business_number = '1234567890' AND deleted_at IS NOT NULL;
```

필요시 삭제된 데이터용 별도 인덱스 생성:
```sql
CREATE INDEX idx_stores_deleted_business_number
ON stores(business_number)
WHERE deleted_at IS NOT NULL;
```

## 롤백

만약 문제가 발생하면 다음과 같이 롤백할 수 있습니다:

```sql
-- Partial unique indexes 제거
DROP INDEX IF EXISTS idx_stores_business_number;
DROP INDEX IF EXISTS idx_stores_slug;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_nickname;
DROP INDEX IF EXISTS idx_business_registrations_store_id;

-- 기존 unique indexes 재생성
CREATE UNIQUE INDEX stores_business_number_key ON stores(business_number);
CREATE UNIQUE INDEX stores_slug_key ON stores(slug);
CREATE UNIQUE INDEX users_email_key ON users(email);
CREATE UNIQUE INDEX users_nickname_key ON users(nickname);
CREATE UNIQUE INDEX business_registrations_store_id_key ON business_registrations(store_id);
```

**주의:** 롤백 시 삭제된 데이터에 중복이 있으면 unique index 생성이 실패할 수 있습니다.

## 문의

Migration 관련 문의사항은 개발팀에 문의해주세요.
