package model

import (
	"time"

	"gorm.io/gorm"
)

// CommunityComment 커뮤니티 댓글 모델
type CommunityComment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 댓글 기본 정보
	Content string `gorm:"type:text;not null" json:"content"` // 댓글 내용

	// 작성자 정보
	UserID uint `gorm:"not null;index" json:"user_id"` // 작성자 ID
	User   User `gorm:"foreignKey:UserID" json:"user"` // 작성자 정보

	// 게시글 정보
	PostID uint          `gorm:"not null;index" json:"post_id"` // 게시글 ID
	Post   CommunityPost `gorm:"foreignKey:PostID" json:"-"`    // 게시글 정보

	// 대댓글 (계층 구조)
	ParentID *uint               `gorm:"index" json:"parent_id,omitempty"`     // 부모 댓글 ID (대댓글일 경우)
	Parent   *CommunityComment   `gorm:"foreignKey:ParentID" json:"-"`         // 부모 댓글
	Replies  []CommunityComment  `gorm:"foreignKey:ParentID" json:"replies,omitempty"` // 대댓글 목록

	// QnA 관련
	IsAnswer bool `gorm:"default:false" json:"is_answer"` // 답변 여부 (QnA 카테고리)
	IsAccepted bool `gorm:"default:false" json:"is_accepted"` // 채택 여부

	// 통계
	LikeCount int `gorm:"default:0" json:"like_count"` // 좋아요 수

	// 관계
	Likes []CommentLike `gorm:"foreignKey:CommentID" json:"-"` // 좋아요 목록
}

func (CommunityComment) TableName() string {
	return "community_comments"
}

// CommentLike 댓글 좋아요 모델
type CommentLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	CommentID uint `gorm:"not null;index:idx_comment_user_like,unique" json:"comment_id"` // 댓글 ID
	UserID    uint `gorm:"not null;index:idx_comment_user_like,unique" json:"user_id"`    // 사용자 ID

	Comment CommunityComment `gorm:"foreignKey:CommentID" json:"-"`
	User    User             `gorm:"foreignKey:UserID" json:"-"`
}

func (CommentLike) TableName() string {
	return "comment_likes"
}

// CreateCommentRequest 댓글 생성 요청
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1"`
	PostID   uint   `json:"post_id" binding:"required"`
	ParentID *uint  `json:"parent_id,omitempty"` // 대댓글일 경우
	IsAnswer bool   `json:"is_answer,omitempty"` // QnA 답변 여부
}

// UpdateCommentRequest 댓글 수정 요청
type UpdateCommentRequest struct {
	Content *string `json:"content,omitempty" binding:"omitempty,min=1"`
}

// CommentListQuery 댓글 목록 조회 쿼리
type CommentListQuery struct {
	PostID    uint   `form:"post_id" binding:"required"`
	ParentID  *uint  `form:"parent_id"` // null이면 최상위 댓글만
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=created_at like_count"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}
