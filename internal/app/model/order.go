package model

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string      // 주문 상태 코드
type PaymentStatus string    // 결제 상태 코드
type FulfillmentType string  // 주문 처리 유형

const (
	OrderStatusPending   OrderStatus = "pending"   // 주문 접수
	OrderStatusConfirmed OrderStatus = "confirmed" // 주문 확정
	OrderStatusShipping  OrderStatus = "shipping"  // 배송 중
	OrderStatusDelivered OrderStatus = "delivered" // 배송 완료
	OrderStatusCancelled OrderStatus = "cancelled" // 주문 취소

	PaymentStatusPending   PaymentStatus = "pending"   // 결제 대기
	PaymentStatusCompleted PaymentStatus = "completed" // 결제 완료
	PaymentStatusFailed    PaymentStatus = "failed"    // 결제 실패
	PaymentStatusRefunded  PaymentStatus = "refunded"  // 환불 완료

	FulfillmentDelivery FulfillmentType = "delivery" // 택배 배송
	FulfillmentPickup   FulfillmentType = "pickup"   // 매장 픽업
)

type Order struct {
	ID              uint            `gorm:"primarykey" json:"id"`                                                        // 주문 ID
	UserID          uint            `gorm:"not null;index" json:"user_id"`                                               // 주문자 ID
	TotalAmount     float64         `gorm:"not null" json:"total_amount"`                                                // 총 결제 금액
	TotalPrice      float64         `gorm:"not null" json:"total_price"`                                                 // 총 상품 금액
	Status          OrderStatus     `gorm:"type:varchar(20);default:'pending'" json:"status"`                            // 주문 상태
	PaymentStatus   PaymentStatus   `gorm:"type:varchar(20);default:'pending'" json:"payment_status"`                    // 결제 상태
	PaymentProvider string          `gorm:"type:varchar(50)" json:"payment_provider,omitempty"`                          // 결제 제공자 (kakaopay, card 등)
	PaymentTID      string          `gorm:"type:varchar(50);index" json:"payment_tid,omitempty"`                         // 결제 거래 ID (Kakao Pay TID)
	PaymentAID      string          `gorm:"type:varchar(50)" json:"payment_aid,omitempty"`                               // 결제 승인 ID (Kakao Pay AID)
	PaymentApprovedAt *time.Time    `json:"payment_approved_at,omitempty"`                                               // 결제 승인 시각
	FulfillmentType FulfillmentType `gorm:"type:varchar(20);default:'delivery'" json:"fulfillment_type"`                 // 이행 방식
	ShippingAddress string          `gorm:"type:text" json:"shipping_address"`                                           // 배송지 주소
	PickupStoreID   *uint           `gorm:"index" json:"pickup_store_id,omitempty"`                                      // 픽업 매장 ID
	PickupStore     *Store          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"pickup_store,omitempty"` // 픽업 매장 정보
	CreatedAt       time.Time       `json:"created_at"`                                                                  // 생성 시각
	UpdatedAt       time.Time       `json:"updated_at"`                                                                  // 수정 시각
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"-"`                                                              // 삭제 시각(소프트 삭제)

	User       User        `gorm:"foreignKey:UserID" json:"user,omitempty"`                                     // 주문자 정보
	OrderItems []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order_items,omitempty"` // 주문 항목 목록
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID              uint           `gorm:"primarykey" json:"id"`                     // 주문 항목 ID
	OrderID         uint           `gorm:"not null;index" json:"order_id"`           // 주문 ID
	ProductID       uint           `gorm:"not null;index" json:"product_id"`         // 상품 ID
	ProductOptionID *uint          `gorm:"index" json:"product_option_id,omitempty"` // 선택 옵션 ID
	StoreID         uint           `gorm:"not null;index" json:"store_id"`           // 매장 ID
	Quantity        int            `gorm:"not null" json:"quantity"`                 // 수량
	Price           float64        `gorm:"not null" json:"price"`                    // 단가
	OptionSnapshot  string         `gorm:"type:text" json:"option_snapshot"`         // 옵션 정보 스냅샷
	CreatedAt       time.Time      `json:"created_at"`                               // 생성 시각
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                           // 삭제 시각(소프트 삭제)

	Order         Order         `gorm:"foreignKey:OrderID" json:"-"`                   // 주문 정보
	Product       Product       `gorm:"foreignKey:ProductID" json:"product,omitempty"` // 상품 정보
	ProductOption ProductOption `json:"product_option,omitempty"`                      // 옵션 정보
	Store         Store         `gorm:"foreignKey:StoreID" json:"store,omitempty"`     // 매장 정보
}

func (OrderItem) TableName() string {
	return "order_items"
}
