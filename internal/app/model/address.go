package model

import (
	"time"

	"gorm.io/gorm"
)

type Address struct {
	ID            uint           `gorm:"primaryKey" json:"id"`                       // 배송지 ID
	UserID        uint           `gorm:"not null;index" json:"user_id"`              // 사용자 ID
	Name          string         `gorm:"size:100;not null" json:"name"`              // 배송지명 (예: "집", "회사")
	Recipient     string         `gorm:"size:100;not null" json:"recipient"`         // 수령인
	Phone         string         `gorm:"size:30;not null" json:"phone"`              // 전화번호
	ZipCode       string         `gorm:"size:10" json:"zip_code"`                    // 우편번호
	Address       string         `gorm:"type:text;not null" json:"address"`          // 주소
	DetailAddress string         `gorm:"type:text" json:"detail_address"`            // 상세주소
	IsDefault     bool           `gorm:"default:false" json:"is_default"`            // 기본 배송지 여부
	CreatedAt     time.Time      `json:"created_at"`                                 // 생성 시각
	UpdatedAt     time.Time      `json:"updated_at"`                                 // 수정 시각
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`                             // 삭제 시각(소프트 삭제)
}

func (Address) TableName() string {
	return "addresses"
}
