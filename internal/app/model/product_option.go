package model

import (
	"time"

	"gorm.io/gorm"
)

type ProductOption struct {
	ID              uint           `gorm:"primarykey" json:"id"`              // 고유 옵션 ID
	ProductID       uint           `gorm:"index;not null" json:"product_id"`  // 소속 상품 ID
	Name            string         `gorm:"not null" json:"name"`              // 옵션 그룹명
	Value           string         `gorm:"not null" json:"value"`             // 옵션 값
	AdditionalPrice float64        `gorm:"default:0" json:"additional_price"` // 추가 금액
	StockQuantity   int            `gorm:"default:0" json:"stock_quantity"`   // 옵션 재고
	ImageURL        string         `json:"image_url"`                         // 옵션 이미지
	IsDefault       bool           `gorm:"default:false" json:"is_default"`   // 기본 옵션 여부
	CreatedAt       time.Time      `json:"created_at"`                        // 생성 시각
	UpdatedAt       time.Time      `json:"updated_at"`                        // 수정 시각
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                    // 삭제 시각(소프트 삭제)

	Product Product `gorm:"foreignKey:ProductID" json:"-"` // 소속 상품 정보
}

func (ProductOption) TableName() string {
	return "product_options"
}
