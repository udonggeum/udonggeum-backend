package model

import (
	"time"

	"gorm.io/gorm"
)

// ChatRoomType 채팅방 타입
type ChatRoomType string

const (
	ChatRoomTypeStore    ChatRoomType = "STORE"     // 사용자가 매장에 일반 문의
	ChatRoomTypeSellGold ChatRoomType = "SELL_GOLD" // 사용자 판매글에 내가 문의 (사용자의 금 판매글)
	ChatRoomTypeBuyGold  ChatRoomType = "BUY_GOLD"  // 내 구매글에 사용자가 문의 (매장의 금 매입 홍보글)
	ChatRoomTypeSale     ChatRoomType = "SALE"      // Deprecated: SELL_GOLD 또는 BUY_GOLD 사용
)

// ChatRoom 채팅방 모델
// 1:1 채팅방을 나타냄 (판매글 또는 매장 기반)
type ChatRoom struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Type      ChatRoomType   `gorm:"type:varchar(10);not null;index" json:"type"` // SALE or STORE

	// 참여자
	User1ID   uint           `gorm:"not null;index:idx_user1_last_msg,priority:1;index" json:"user1_id"` // 판매자/매장주인
	User2ID   uint           `gorm:"not null;index:idx_user2_last_msg,priority:1;index" json:"user2_id"` // 구매자/문의자
	User1     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user1,omitempty"`
	User2     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user2,omitempty"`

	// 관련 리소스 (둘 중 하나만 존재)
	ProductID *uint          `gorm:"index" json:"product_id,omitempty"` // 판매글 ID (SALE 타입)
	Product   *CommunityPost `gorm:"foreignKey:ProductID" json:"product,omitempty"` // 판매글 정보
	StoreID   *uint          `gorm:"index" json:"store_id,omitempty"`   // 매장 ID (STORE 타입)
	Store     *Store         `gorm:"foreignKey:StoreID" json:"store,omitempty"` // 매장 정보

	// 마지막 메시지 정보 (채팅방 목록에서 활용)
	LastMessageID      *uint      `json:"last_message_id,omitempty"`
	LastMessageContent string     `gorm:"type:text" json:"last_message_content,omitempty"`
	LastMessageAt      *time.Time `gorm:"index:idx_user1_last_msg,priority:2;index:idx_user2_last_msg,priority:2" json:"last_message_at,omitempty"` // 목록 정렬 최적화

	// 읽지 않은 메시지 수 (각 사용자별)
	User1UnreadCount int `gorm:"default:0" json:"user1_unread_count"`
	User2UnreadCount int `gorm:"default:0" json:"user2_unread_count"`

	// 채팅방 나가기 (soft delete)
	User1LeftAt *time.Time `json:"user1_left_at,omitempty"` // user1이 나간 시간
	User2LeftAt *time.Time `json:"user2_left_at,omitempty"` // user2가 나간 시간

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
	ChatRoomID uint           `gorm:"not null;index:idx_room_created,priority:1;index" json:"chat_room_id"`
	ChatRoom   ChatRoom       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	SenderID   uint           `gorm:"not null;index:idx_room_unread,priority:3;index" json:"sender_id"`
	Sender     User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"sender,omitempty"`

	Content    string         `gorm:"type:text;not null" json:"content"` // 메시지 내용

	// 메시지 타입 (확장성)
	MessageType string        `gorm:"type:varchar(20);default:'TEXT'" json:"message_type"` // TEXT, IMAGE, FILE 등
	FileURL     string        `gorm:"type:text" json:"file_url,omitempty"`                 // 파일/이미지 URL (IMAGE, FILE 타입일 때)
	FileName    string        `gorm:"type:varchar(255)" json:"file_name,omitempty"`        // 원본 파일명

	// 수정/삭제 정보
	IsEdited   bool       `gorm:"default:false" json:"is_edited"`        // 수정 여부
	EditedAt   *time.Time `json:"edited_at,omitempty"`                   // 수정 시간
	IsDeleted  bool       `gorm:"default:false" json:"is_deleted"`       // 삭제 여부 (soft delete)
	DeletedBy  *uint      `json:"deleted_by,omitempty"`                  // 삭제한 사용자 ID

	// 읽음 처리
	IsRead     bool           `gorm:"default:false;index:idx_room_unread,priority:2;index" json:"is_read"`
	ReadAt     *time.Time     `json:"read_at,omitempty"`

	CreatedAt  time.Time      `gorm:"index:idx_room_created,priority:2" json:"created_at"` // 메시지 목록 정렬 최적화
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
