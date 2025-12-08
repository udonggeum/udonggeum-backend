package model

import (
	"time"

	"gorm.io/gorm"
)

// StoreReview 매장 리뷰 모델
type StoreReview struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 리뷰 기본 정보
	StoreID   uint   `gorm:"not null;index" json:"store_id"`     // 매장 ID
	Store     Store  `gorm:"foreignKey:StoreID" json:"store,omitempty"` // 매장 정보
	UserID    uint   `gorm:"not null;index" json:"user_id"`      // 작성자 ID
	User      User   `gorm:"foreignKey:UserID" json:"user"`      // 작성자 정보
	Rating    int    `gorm:"not null" json:"rating"`             // 평점 (1-5)
	Content   string `gorm:"type:text;not null" json:"content"`  // 리뷰 내용

	// 이미지
	ImageURLs []string `gorm:"type:text[]" json:"image_urls,omitempty"` // 리뷰 이미지 URL 배열

	// 방문자 리뷰 여부
	IsVisitor bool `gorm:"default:false" json:"is_visitor"` // 방문자 리뷰 (실제 방문 인증)

	// 통계
	LikeCount int `gorm:"default:0" json:"like_count"` // 좋아요 수

	// 관계
	Likes []ReviewLike `gorm:"foreignKey:ReviewID" json:"-"` // 좋아요 목록
}

func (StoreReview) TableName() string {
	return "store_reviews"
}

// ReviewLike 리뷰 좋아요 모델
type ReviewLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	ReviewID uint `gorm:"not null;index:idx_review_user_like,unique" json:"review_id"` // 리뷰 ID
	UserID   uint `gorm:"not null;index:idx_review_user_like,unique" json:"user_id"`   // 사용자 ID

	Review StoreReview `gorm:"foreignKey:ReviewID" json:"-"`
	User   User        `gorm:"foreignKey:UserID" json:"-"`
}

func (ReviewLike) TableName() string {
	return "review_likes"
}
