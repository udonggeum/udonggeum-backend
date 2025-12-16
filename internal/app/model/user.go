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
	Nickname     string         `gorm:"uniqueIndex;not null" json:"nickname"`        // 닉네임 (자동 생성, 수정 가능)
	Phone        string         `json:"phone"`                                       // 전화번호 (숫자만, 예: 01012345678)
	ProfileImage string         `json:"profile_image"`                               // 프로필 이미지 URL
	Address      string         `json:"address"`                                     // 주소
	Role         UserRole       `gorm:"type:varchar(20);default:'user'" json:"role"` // 권한
	StoreID      *uint          `gorm:"index" json:"store_id,omitempty"`             // 대표 매장 ID (사장님용)
	CreatedAt    time.Time      `json:"created_at"`                                  // 생성 시각
	UpdatedAt    time.Time      `json:"updated_at"`                                  // 수정 시각
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                              // 삭제 시각(소프트 삭제)

	Store  *Store  `gorm:"foreignKey:StoreID" json:"store,omitempty"`     // 대표 매장 (사장님용)
	Stores []Store `gorm:"foreignKey:UserID" json:"stores,omitempty"`     // 소유 매장 목록 (Admin만 관리)
}

func (User) TableName() string {
	return "users"
}
