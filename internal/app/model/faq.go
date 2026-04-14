package model

import (
	"time"

	"gorm.io/gorm"
)

// FAQTarget FAQ 대상 유형
type FAQTarget string

const (
	FAQTargetUser  FAQTarget = "user"  // 일반 사용자
	FAQTargetOwner FAQTarget = "owner" // 금은방 사장님
)

// FAQ 자주 묻는 질문
type FAQ struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Target    FAQTarget      `gorm:"type:varchar(10);not null;default:'user'" json:"target"`
	Question  string         `gorm:"type:text;not null" json:"question"`
	Answer    string         `gorm:"type:text;not null" json:"answer"`
	SortOrder int            `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (FAQ) TableName() string {
	return "faqs"
}
