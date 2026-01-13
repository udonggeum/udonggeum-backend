package model

import (
	"time"

	"gorm.io/gorm"
)

// StoreVerification 매장 인증 정보 모델
type StoreVerification struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 외래 키
	StoreID uint  `gorm:"uniqueIndex;not null" json:"store_id"` // 매장 ID (1:1 관계)
	Store   Store `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	// 인증 정보
	BusinessLicenseURL string `gorm:"type:text;not null" json:"business_license_url"` // 사업자등록증 이미지 URL
	Status             string `gorm:"type:varchar(20);default:'pending';index" json:"status"` // pending, approved, rejected
	SubmittedAt        *time.Time `json:"submitted_at,omitempty"`                             // 제출 일시
	ReviewedAt         *time.Time `json:"reviewed_at,omitempty"`                              // 검토 완료 일시
	ReviewedBy         *uint      `json:"reviewed_by,omitempty"`                              // 검토한 관리자 ID
	RejectionReason    string     `gorm:"type:text" json:"rejection_reason,omitempty"`        // 반려 사유

	// 추적 정보 (보안/로그용)
	IPAddress string `gorm:"type:varchar(50)" json:"ip_address,omitempty"` // 제출자 IP
	UserAgent string `gorm:"type:text" json:"user_agent,omitempty"`        // 제출자 User-Agent
}

func (StoreVerification) TableName() string {
	return "store_verifications"
}

// VerificationStatus 상수 정의
const (
	VerificationStatusPending  = "pending"  // 검토 대기
	VerificationStatusApproved = "approved" // 승인됨
	VerificationStatusRejected = "rejected" // 반려됨
)
