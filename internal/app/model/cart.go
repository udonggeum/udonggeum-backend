package model

import (
	"time"

	"gorm.io/gorm"
)

type CartItem struct {
	ID              uint           `gorm:"primarykey" json:"id"`                     // 장바구니 항목 ID
	UserID          uint           `gorm:"not null;index" json:"user_id"`            // 사용자 ID
	ProductID       uint           `gorm:"not null;index" json:"product_id"`         // 상품 ID
	ProductOptionID *uint          `gorm:"index" json:"product_option_id,omitempty"` // 선택 옵션 ID
	Quantity        int            `gorm:"not null;default:1" json:"quantity"`       // 수량
	CreatedAt       time.Time      `json:"created_at"`                               // 생성 시각
	UpdatedAt       time.Time      `json:"updated_at"`                               // 수정 시각
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                           // 삭제 시각(소프트 삭제)

	User          User          `gorm:"foreignKey:UserID" json:"-"`                    // 사용자 정보
	Product       Product       `gorm:"foreignKey:ProductID" json:"product,omitempty"` // 상품 정보
	ProductOption ProductOption `json:"product_option,omitempty"`                      // 옵션 정보
}

func (CartItem) TableName() string {
	return "cart_items"
}
