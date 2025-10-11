package model

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string
type PaymentStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipping  OrderStatus = "shipping"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"

	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Order struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	TotalAmount     float64        `gorm:"not null" json:"total_amount"`
	Status          OrderStatus    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentStatus   PaymentStatus  `gorm:"type:varchar(20);default:'pending'" json:"payment_status"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	TotalPrice      float64        `gorm:"not null" json:"total_price"`

	// Relationships
	User       User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order_items,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	OrderID   uint           `gorm:"not null;index" json:"order_id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	Quantity  int            `gorm:"not null" json:"quantity"`
	Price     float64        `gorm:"not null" json:"price"` // 주문 당시 가격
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Order   Order   `gorm:"foreignKey:OrderID" json:"-"`
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
