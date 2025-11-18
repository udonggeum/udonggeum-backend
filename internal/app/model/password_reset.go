package model

import (
	"time"
)

type PasswordReset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`                        // 비밀번호 재설정 ID
	Email     string    `gorm:"size:255;not null;index" json:"email"`        // 이메일
	Token     string    `gorm:"size:255;not null;unique;index" json:"-"`     // 재설정 토큰 (노출 금지)
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`                  // 만료 시각
	Used      bool      `gorm:"default:false" json:"used"`                   // 사용 여부
	CreatedAt time.Time `json:"created_at"`                                  // 생성 시각
}

func (PasswordReset) TableName() string {
	return "password_resets"
}
