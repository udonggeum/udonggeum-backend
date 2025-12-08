package model

import (
	"time"

	"gorm.io/gorm"
)

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

	// 매입 가능 여부 필드
	BuyingGold     bool `gorm:"default:false" json:"buying_gold"`     // 금 매입 가능 여부
	BuyingPlatinum bool `gorm:"default:false" json:"buying_platinum"` // 백금 매입 가능 여부
	BuyingSilver   bool `gorm:"default:false" json:"buying_silver"`   // 은 매입 가능 여부

	CreatedAt   time.Time      `json:"created_at"`                           // 생성 시각
	UpdatedAt   time.Time      `json:"updated_at"`                           // 수정 시각
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`                       // 삭제 시각(소프트 삭제)

	Products []Product `gorm:"foreignKey:StoreID" json:"products,omitempty"` // 보유 상품 목록

	CategoryCounts map[ProductCategory]int `gorm:"-" json:"category_counts,omitempty"` // 카테고리별 상품 수
	TotalProducts  int                     `gorm:"-" json:"total_products,omitempty"`  // 전체 상품 수
}

func (Store) TableName() string {
	return "stores"
}
