package model

import (
	"time"

	"gorm.io/gorm"
)

// Tag represents a predefined tag that can be associated with stores
// 매장에 연결할 수 있는 사전 정의된 태그
type Tag struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Name      string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"` // 태그 이름 (예: "24K 취급")
	Category  string         `gorm:"type:varchar(20)" json:"category"`                  // 카테고리 (예: "서비스", "상품", "특징")
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Tag) TableName() string {
	return "tags"
}

// StoreTag represents the many-to-many relationship between stores and tags
// 매장과 태그의 다대다 관계
type StoreTag struct {
	StoreID   uint      `gorm:"primaryKey;index" json:"store_id"`
	TagID     uint      `gorm:"primaryKey;index" json:"tag_id"`
	Store     Store     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Tag       Tag       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"tag,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (StoreTag) TableName() string {
	return "store_tags"
}
