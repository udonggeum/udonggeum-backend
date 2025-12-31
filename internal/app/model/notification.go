package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeNewSellPost  NotificationType = "new_sell_post"
	NotificationTypePostComment  NotificationType = "post_comment"
	NotificationTypeStoreLiked   NotificationType = "store_liked"
)

type NotificationRange string

const (
	NotificationRangeDistrict   NotificationRange = "district"
	NotificationRangeRegion     NotificationRange = "region"
	NotificationRangeNationwide NotificationRange = "nationwide"
)

// Notification 알림 모델
type Notification struct {
	ID        uint             `gorm:"primarykey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`

	// 알림 받을 사용자
	UserID uint  `gorm:"not null;index" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 알림 타입
	Type NotificationType `gorm:"type:varchar(50);not null;index" json:"type"`

	// 알림 내용
	Title   string `gorm:"type:text;not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`
	Link    string `gorm:"type:text;not null" json:"link"`

	// 상태
	IsRead bool `gorm:"default:false;index" json:"is_read"`

	// 관련 데이터 (nullable)
	RelatedPostID  *uint `gorm:"index" json:"related_post_id,omitempty"`
	RelatedStoreID *uint `gorm:"index" json:"related_store_id,omitempty"`
	RelatedUserID  *uint `gorm:"index" json:"related_user_id,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}

// NotificationSettings 사용자별 알림 설정
type NotificationSettings struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 사용자
	UserID uint  `gorm:"uniqueIndex;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 금 판매글 알림
	SellPostNotification bool              `gorm:"default:true" json:"sell_post_notification"`
	SellPostRange        NotificationRange `gorm:"type:varchar(20);default:'district'" json:"sell_post_range"`          // Deprecated: SelectedRegions 사용
	SelectedRegions      pq.StringArray    `gorm:"type:text[];default:'{}';not null" json:"selected_regions"` // 선택한 지역 목록 (예: ["서울 강남구", "서울 서초구"])

	// 댓글 알림
	CommentNotification bool `gorm:"default:true" json:"comment_notification"`

	// 찜 알림
	LikeNotification bool `gorm:"default:true" json:"like_notification"`
}

func (NotificationSettings) TableName() string {
	return "notification_settings"
}
