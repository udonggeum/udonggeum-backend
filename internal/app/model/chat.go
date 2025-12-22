package model

import (
	"time"

	"gorm.io/gorm"
)

// ChatRoomType 채팅방 타입
type ChatRoomType string

const (
	ChatRoomTypeSale  ChatRoomType = "SALE"  // 판매글 기반 채팅
	ChatRoomTypeStore ChatRoomType = "STORE" // 가게 문의 채팅
)

// ChatRoom 채팅방 모델
// 1:1 채팅방을 나타냄 (판매글 또는 매장 기반)
type ChatRoom struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Type      ChatRoomType   `gorm:"type:varchar(10);not null;index" json:"type"` // SALE or STORE

	// 참여자
	User1ID   uint           `gorm:"not null;index" json:"user1_id"` // 판매자/매장주인
	User2ID   uint           `gorm:"not null;index" json:"user2_id"` // 구매자/문의자
	User1     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user1,omitempty"`
	User2     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user2,omitempty"`

	// 관련 리소스 (둘 중 하나만 존재)
	ProductID *uint          `gorm:"index" json:"product_id,omitempty"` // 판매글 ID (SALE 타입)
	StoreID   *uint          `gorm:"index" json:"store_id,omitempty"`   // 매장 ID (STORE 타입)

	// 마지막 메시지 정보 (채팅방 목록에서 활용)
	LastMessageID      *uint      `json:"last_message_id,omitempty"`
	LastMessageContent string     `gorm:"type:text" json:"last_message_content,omitempty"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`

	// 읽지 않은 메시지 수 (각 사용자별)
	User1UnreadCount int `gorm:"default:0" json:"user1_unread_count"`
	User2UnreadCount int `gorm:"default:0" json:"user2_unread_count"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Virtual fields (GORM에서 자동 로드하지 않음)
	Messages []Message `gorm:"foreignKey:ChatRoomID" json:"messages,omitempty"`
}

func (ChatRoom) TableName() string {
	return "chat_rooms"
}

// Message 메시지 모델
type Message struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	ChatRoomID uint           `gorm:"not null;index" json:"chat_room_id"`
	ChatRoom   ChatRoom       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	SenderID   uint           `gorm:"not null;index" json:"sender_id"`
	Sender     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"sender,omitempty"`

	Content    string         `gorm:"type:text;not null" json:"content"` // 메시지 내용

	// 메시지 타입 (확장성)
	MessageType string        `gorm:"type:varchar(20);default:'TEXT'" json:"message_type"` // TEXT, IMAGE, FILE 등

	// 읽음 처리
	IsRead     bool           `gorm:"default:false;index" json:"is_read"`
	ReadAt     *time.Time     `json:"read_at,omitempty"`

	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Message) TableName() string {
	return "messages"
}

// ChatRoomWithUnread 채팅방 + 현재 사용자의 읽지 않은 메시지 수
type ChatRoomWithUnread struct {
	ChatRoom
	UnreadCount int `json:"unread_count"` // 현재 사용자의 읽지 않은 메시지 수
}
