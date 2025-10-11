package model

import (
	"time"

	"gorm.io/gorm"
)

type ProductCategory string

const (
	CategoryGold    ProductCategory = "gold"
	CategorySilver  ProductCategory = "silver"
	CategoryJewelry ProductCategory = "jewelry"
)

type Product struct {
	ID            uint            `gorm:"primarykey" json:"id"`
	Name          string          `gorm:"not null" json:"name"`
	Description   string          `gorm:"type:text" json:"description"`
	Price         float64         `gorm:"not null" json:"price"`
	Weight        float64         `json:"weight"` // 무게 (g)
	Purity        string          `json:"purity"` // 순도 (예: 24K, 18K, 999)
	Category      ProductCategory `gorm:"type:varchar(50)" json:"category"`
	StockQuantity int             `gorm:"default:0" json:"stock_quantity"`
	ImageURL      string          `json:"image_url"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     gorm.DeletedAt  `gorm:"index" json:"-"`

	// Relationships
	OrderItems []OrderItem `gorm:"foreignKey:ProductID" json:"-"`
	CartItems  []CartItem  `gorm:"foreignKey:ProductID" json:"-"`
}

func (Product) TableName() string {
	return "products"
}
