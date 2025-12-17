package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	ID          uint           `gorm:"primarykey" json:"id"`          // 고유 매장 ID
	UserID      uint           `gorm:"not null;index" json:"user_id"` // 매장 소유자 ID
	User        User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"owner,omitempty"`
	Name        string         `gorm:"not null" json:"name"`                 // 매장명
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
