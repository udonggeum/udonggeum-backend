package model

import (
	"time"

	"gorm.io/gorm"
)

type ProductCategory string // 상품 카테고리 타입
type ProductMaterial string // 상품 재질 타입

const (
	CategoryRing     ProductCategory = "반지"  // 반지 제품
	CategoryBracelet ProductCategory = "팔찌"  // 팔찌 제품
	CategoryNecklace ProductCategory = "목걸이" // 목걸이 제품
	CategoryEarring  ProductCategory = "귀걸이" // 귀걸이 제품
	CategoryOther    ProductCategory = "기타"  // 기타 제품
)

const (
	MaterialGold   ProductMaterial = "금"  // 금 재질
	MaterialSilver ProductMaterial = "은"  // 은 재질
	MaterialOther  ProductMaterial = "기타" // 기타 재질
)

type Product struct {
	ID              uint            `gorm:"primarykey" json:"id"`                                                 // 고유 상품 ID
	Name            string          `gorm:"not null" json:"name"`                                                 // 상품명
	Description     string          `gorm:"type:text" json:"description"`                                         // 상품 설명
	Price           float64         `gorm:"not null" json:"price"`                                                // 기본 판매가
	Weight          float64         `json:"weight"`                                                               // 중량(그램 등)
	Purity          string          `json:"purity"`                                                               // 금속 순도
	Category        ProductCategory `gorm:"type:varchar(50)" json:"category"`                                     // 상품 카테고리
	Material        ProductMaterial `gorm:"type:varchar(50)" json:"material"`                                     // 상품 재질
	StockQuantity   int             `gorm:"default:0" json:"stock_quantity"`                                      // 기본 재고 수량
	ImageURL        string          `json:"image_url"`                                                            // 대표 이미지 경로
	StoreID         uint            `gorm:"not null;index" json:"store_id"`                                       // 소속 매장 ID
	Store           Store           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"store,omitempty"` // 매장 정보
	PopularityScore float64         `gorm:"default:0" json:"popularity_score"`                                    // 인기 점수
	ViewCount       int             `gorm:"default:0" json:"view_count"`                                          // 조회수
	CreatedAt       time.Time       `json:"created_at"`                                                           // 생성 시각
	UpdatedAt       time.Time       `json:"updated_at"`                                                           // 수정 시각
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"-"`                                                       // 삭제 시각(소프트 삭제)

	OrderItems []OrderItem     `gorm:"foreignKey:ProductID" json:"-"`                                             // 주문 항목 목록
	CartItems  []CartItem      `gorm:"foreignKey:ProductID" json:"-"`                                             // 장바구니 항목 목록
	Options    []ProductOption `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"options,omitempty"` // 상품 옵션 목록
}

func (Product) TableName() string {
	return "products"
}
