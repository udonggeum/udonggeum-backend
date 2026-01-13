package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// StringArray는 PostgreSQL의 TEXT[] 또는 JSON 배열을 처리하기 위한 커스텀 타입
type StringArray []string

// Value는 database/sql/driver.Valuer 인터페이스 구현
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan은 database/sql.Scanner 인터페이스 구현
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArray")
	}

	return json.Unmarshal(bytes, s)
}

type Store struct {
	ID          uint           `gorm:"primarykey" json:"id"`     // 고유 매장 ID
	UserID      *uint          `gorm:"index" json:"user_id"`     // 매장 소유자 ID (nullable - 비관리매장은 null)
	User        User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"owner,omitempty"`
	Name        string         `gorm:"not null" json:"name"`                 // 매장명
	Slug        string         `gorm:"uniqueIndex" json:"slug"`              // URL용 고유 식별자 (SEO)
	Region      string         `gorm:"index;not null" json:"region"`         // 시·도
	District    string         `gorm:"index;not null" json:"district"`       // 구·군
	Address     string         `gorm:"type:text" json:"address"`             // 상세 주소
	Latitude    *float64       `gorm:"type:decimal(10,8)" json:"latitude"`   // 위도 (WGS84)
	Longitude   *float64       `gorm:"type:decimal(11,8)" json:"longitude"`  // 경도 (WGS84)
	PhoneNumber string         `gorm:"type:varchar(30)" json:"phone_number"` // 연락처
	ImageURL    string         `json:"image_url"`                            // 매장 이미지
	Description string         `gorm:"type:text" json:"description"`         // 매장 소개
	OpenTime    string         `gorm:"type:varchar(10)" json:"open_time"`    // 오픈 시간 (예: "09:00")
	CloseTime   string         `gorm:"type:varchar(10)" json:"close_time"`   // 마감 시간 (예: "20:00")

	// 2단계 검증 시스템
	IsManaged   bool       `gorm:"default:false;index" json:"is_managed"`    // 관리매장 여부 (소유자가 있는 매장)
	IsVerified  bool       `gorm:"default:false;index" json:"is_verified"`   // 인증 매장 여부 (사업자등록증 검증 완료)
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`                    // 인증 완료 일시

	// 사업자 정보 (1:1 관계 - 별도 테이블로 관리)
	BusinessRegistration *BusinessRegistration `gorm:"foreignKey:StoreID" json:"business_registration,omitempty"`

	// 인증 정보 (1:1 관계)
	Verification *StoreVerification `gorm:"foreignKey:StoreID" json:"verification,omitempty"`

	// 매장 태그 (Many-to-Many 관계)
	Tags []Tag `gorm:"many2many:store_tags;" json:"tags,omitempty"`

	// [Deprecated] 매입 가능 여부 필드 - tags로 이관 예정
	BuyingGold     bool `gorm:"default:false" json:"buying_gold,omitempty"`     // 금 매입 가능 여부
	BuyingPlatinum bool `gorm:"default:false" json:"buying_platinum,omitempty"` // 백금 매입 가능 여부
	BuyingSilver   bool `gorm:"default:false" json:"buying_silver,omitempty"`   // 은 매입 가능 여부

	CreatedAt time.Time      `json:"created_at"` // 생성 시각
	UpdatedAt time.Time      `json:"updated_at"` // 수정 시각
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 삭제 시각(소프트 삭제)
}

func (Store) TableName() string {
	return "stores"
}

// StoreLike 매장 좋아요 모델
type StoreLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	StoreID uint `gorm:"not null;index:idx_store_user_like,unique" json:"store_id"` // 매장 ID
	UserID  uint `gorm:"not null;index:idx_store_user_like,unique" json:"user_id"`  // 사용자 ID

	Store Store `gorm:"foreignKey:StoreID" json:"-"`
	User  User  `gorm:"foreignKey:UserID" json:"-"`
}

func (StoreLike) TableName() string {
	return "store_likes"
}

// generateSlug는 매장명과 지역 정보로 URL용 slug를 생성합니다
func generateSlug(district, name string) string {
	// 공백을 하이픈으로 변경
	slug := fmt.Sprintf("%s-%s", district, name)

	// 특수문자 제거 (한글, 영문, 숫자, 하이픈만 허용)
	reg := regexp.MustCompile(`[^\p{L}\p{N}-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 연속된 하이픈을 하나로
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 앞뒤 하이픈 제거
	slug = strings.Trim(slug, "-")

	// 소문자로 변환 (영문만)
	slug = strings.ToLower(slug)

	return slug
}

// BeforeCreate는 매장 생성 전에 slug를 자동 생성합니다
func (s *Store) BeforeCreate(tx *gorm.DB) error {
	if s.Slug == "" {
		baseSlug := generateSlug(s.District, s.Name)
		slug := baseSlug

		// 중복 체크 및 숫자 붙이기
		counter := 1
		for {
			var count int64
			if err := tx.Model(&Store{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
				return err
			}

			if count == 0 {
				break
			}

			counter++
			slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		}

		s.Slug = slug
	}
	return nil
}

// BeforeUpdate는 매장 수정 시 이름이나 지역이 변경되면 slug를 재생성합니다
func (s *Store) BeforeUpdate(tx *gorm.DB) error {
	// 기존 매장 정보 조회
	var oldStore Store
	if err := tx.First(&oldStore, s.ID).Error; err != nil {
		return err
	}

	// 이름이나 지역이 변경되었는지 확인
	if s.Name != oldStore.Name || s.District != oldStore.District {
		baseSlug := generateSlug(s.District, s.Name)
		slug := baseSlug

		// 중복 체크 (자기 자신은 제외)
		counter := 1
		for {
			var count int64
			if err := tx.Model(&Store{}).Where("slug = ? AND id != ?", slug, s.ID).Count(&count).Error; err != nil {
				return err
			}

			if count == 0 {
				break
			}

			counter++
			slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		}

		s.Slug = slug
	}
	return nil
}
