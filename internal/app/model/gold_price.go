package model

import (
	"time"

	"gorm.io/gorm"
)

// GoldPriceType 금 시세 유형
type GoldPriceType string

const (
	Gold24K  GoldPriceType = "24K"      // 24K 금
	Gold18K  GoldPriceType = "18K"      // 18K 금
	Gold14K  GoldPriceType = "14K"      // 14K 금
	Platinum GoldPriceType = "Platinum" // 백금
	Silver   GoldPriceType = "Silver"   // 은
)

// GoldPrice 금 시세 정보
type GoldPrice struct {
	ID          uint          `gorm:"primarykey" json:"id"`                   // 고유 ID
	Type        GoldPriceType `gorm:"type:varchar(10);not null" json:"type"`  // 금 유형 (24K, 18K, 14K, Platinum, Silver)
	BuyPrice    float64       `gorm:"not null" json:"buy_price"`              // 매입가 (원/g)
	SellPrice   float64       `gorm:"not null" json:"sell_price"`             // 매도가 (원/g)
	Source      string        `gorm:"type:varchar(100)" json:"source"`        // 시세 출처
	SourceDate  time.Time     `json:"source_date"`                            // 시세 기준 시각
	Description string        `gorm:"type:text" json:"description,omitempty"` // 추가 설명
	CreatedAt   time.Time     `json:"created_at"`                             // 생성 시각
	UpdatedAt   time.Time     `json:"updated_at"`                             // 수정 시각
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`                        // 삭제 시각(소프트 삭제)
}

func (GoldPrice) TableName() string {
	return "gold_prices"
}

// GoldPriceResponse API 응답용 금 시세 정보
type GoldPriceResponse struct {
	Type              GoldPriceType `json:"type"`                          // 금 유형
	BuyPrice          float64       `json:"buy_price"`                     // 매입가 (원/g)
	SellPrice         float64       `json:"sell_price"`                    // 매도가 (원/g)
	Source            string        `json:"source"`                        // 시세 출처
	SourceDate        string        `json:"source_date"`                   // 시세 기준 시각
	Description       string        `json:"description,omitempty"`         // 추가 설명
	UpdatedAt         string        `json:"updated_at"`                    // 업데이트 시각

	// 전일 대비 변동률 필드
	PreviousDayPrice  *float64      `json:"previous_day_price,omitempty"`  // 전일 종가
	ChangeAmount      *float64      `json:"change_amount,omitempty"`       // 전일 대비 변동 금액 (원)
	ChangePercent     *float64      `json:"change_percent,omitempty"`      // 전일 대비 변동률 (%)
}

// GoldPriceHistoryItem 과거 시세 이력 아이템
type GoldPriceHistoryItem struct {
	Date      string  `json:"date"`       // 날짜 (YYYY-MM-DD)
	SellPrice float64 `json:"sell_price"` // 매도가 (원/g)
	BuyPrice  float64 `json:"buy_price"`  // 매입가 (원/g)
}
