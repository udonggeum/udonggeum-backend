package model

import (
	"time"

	"gorm.io/gorm"
)

type WishlistItem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`                     // 찜 항목 ID
	UserID    uint           `gorm:"not null;index" json:"user_id"`            // 사용자 ID
	ProductID uint           `gorm:"not null;index" json:"product_id"`         // 상품 ID
	CreatedAt time.Time      `json:"created_at"`                               // 생성 시각
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                           // 삭제 시각(소프트 삭제)

	// Associations (loaded with Preload)
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"` // 상품 정보
}

func (WishlistItem) TableName() string {
	return "wishlist_items"
}
