package model

import (
	"time"

	"gorm.io/gorm"
)

// BusinessRegistration 사업자 등록 정보 모델
// 매장 등록 시 사업자 인증에 사용되며, 한 번 등록 후 수정 불가
type BusinessRegistration struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 외래 키
	StoreID uint  `gorm:"not null;uniqueIndex" json:"store_id"` // 매장 ID (1:1 관계)
	Store   Store `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	// 사업자 정보
	BusinessNumber     string `gorm:"type:varchar(10);not null;index" json:"business_number"`      // 사업자등록번호 (10자리, 하이픈 제외)
	BusinessStartDate  string `gorm:"type:varchar(8);not null" json:"business_start_date"`         // 개업일자 (YYYYMMDD)
	RepresentativeName string `gorm:"type:varchar(100);not null" json:"representative_name"`       // 대표자명
	BusinessStatus     string `gorm:"type:varchar(20)" json:"business_status,omitempty"`           // 사업자 상태 (계속사업자/휴업자/폐업자)
	TaxType            string `gorm:"type:varchar(20)" json:"tax_type,omitempty"`                  // 과세 유형 (일반과세자/간이과세자)
	IsVerified         bool   `gorm:"default:false;not null" json:"is_verified"`                   // 사업자 인증 여부
	VerificationDate   *time.Time `json:"verification_date,omitempty"`                              // 인증 일시
}

func (BusinessRegistration) TableName() string {
	return "business_registrations"
}
