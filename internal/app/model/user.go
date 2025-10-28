package model

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string // 사용자 권한 타입

const (
	RoleUser  UserRole = "user"  // 일반 사용자 권한
	RoleAdmin UserRole = "admin" // 관리자 권한
)

type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`                        // 사용자 ID
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`           // 이메일
	PasswordHash string         `gorm:"not null" json:"-"`                           // 비밀번호 해시
	Name         string         `gorm:"not null" json:"name"`                        // 이름
	Phone        string         `json:"phone"`                                       // 전화번호
	Role         UserRole       `gorm:"type:varchar(20);default:'user'" json:"role"` // 권한
	CreatedAt    time.Time      `json:"created_at"`                                  // 생성 시각
	UpdatedAt    time.Time      `json:"updated_at"`                                  // 수정 시각
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                              // 삭제 시각(소프트 삭제)

	Orders    []Order    `gorm:"foreignKey:UserID" json:"orders,omitempty"`     // 사용자 주문
	CartItems []CartItem `gorm:"foreignKey:UserID" json:"cart_items,omitempty"` // 장바구니 항목
	Stores    []Store    `gorm:"foreignKey:UserID" json:"stores,omitempty"`     // 소유 매장 목록
}

func (User) TableName() string {
	return "users"
}
