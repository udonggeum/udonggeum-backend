package repository

import (
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"gorm.io/gorm"
)

type ChatRepository interface {
	// ChatRoom operations
	CreateChatRoom(room *model.ChatRoom) error
	GetChatRoomByID(id uint) (*model.ChatRoom, error)
	GetChatRoomByIDWithUsers(id uint) (*model.ChatRoom, error)
	FindExistingChatRoom(user1ID, user2ID uint, roomType model.ChatRoomType, resourceID *uint) (*model.ChatRoom, error)
	GetUserChatRooms(userID uint, limit, offset int) ([]model.ChatRoom, int64, error)
	UpdateChatRoomLastMessage(roomID uint, messageID uint, content string, timestamp time.Time) error
	IncrementUnreadCount(roomID uint, recipientID uint) error
	ResetUnreadCount(roomID uint, userID uint) error
	LeaveChatRoom(roomID uint, userID uint) error        // 채팅방 나가기
	RejoinChatRoom(roomID uint, userID uint) error       // 채팅방 다시 참여
	DeleteChatRoomIfBothLeft(roomID uint) error          // 양쪽 모두 나간 경우 삭제

	// Message operations
	CreateMessage(message *model.Message) error
	GetMessageByID(id uint) (*model.Message, error)
	GetChatRoomMessages(roomID uint, limit, offset int) ([]model.Message, int64, error)
	MarkMessagesAsRead(roomID uint, recipientID uint) error
	GetUnreadMessageCount(roomID uint, userID uint) (int64, error)
	SearchMessages(userID uint, keyword string, limit, offset int) ([]model.Message, int64, error) // 메시지 검색
	UpdateMessage(messageID uint, content string) error                                              // 메시지 수정
	DeleteMessage(messageID uint, deletedBy uint) error                                              // 메시지 삭제
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

// CreateChatRoom 채팅방 생성
func (r *chatRepository) CreateChatRoom(room *model.ChatRoom) error {
	return r.db.Create(room).Error
}

// GetChatRoomByID 채팅방 ID로 조회
func (r *chatRepository) GetChatRoomByID(id uint) (*model.ChatRoom, error) {
	var room model.ChatRoom
	if err := r.db.First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// GetChatRoomByIDWithUsers 채팅방 ID로 조회 (사용자 정보 포함)
func (r *chatRepository) GetChatRoomByIDWithUsers(id uint) (*model.ChatRoom, error) {
	var room model.ChatRoom
	if err := r.db.Preload("User1").Preload("User2").First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// FindExistingChatRoom 기존 채팅방 찾기 (중복 생성 방지)
func (r *chatRepository) FindExistingChatRoom(user1ID, user2ID uint, roomType model.ChatRoomType, resourceID *uint) (*model.ChatRoom, error) {
	var room model.ChatRoom
	query := r.db.Where("type = ?", roomType)

	// 두 사용자의 조합으로 찾기 (순서 무관)
	query = query.Where(
		"(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)",
		user1ID, user2ID, user2ID, user1ID,
	)

	// 리소스 ID 조건 추가
	if roomType == model.ChatRoomTypeSale && resourceID != nil {
		query = query.Where("product_id = ?", *resourceID)
	} else if roomType == model.ChatRoomTypeStore && resourceID != nil {
		query = query.Where("store_id = ?", *resourceID)
	}

	if err := query.First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 찾지 못함 (정상)
		}
		return nil, err
	}

	return &room, nil
}

// GetUserChatRooms 사용자의 채팅방 목록 조회 (나간 방 제외)
func (r *chatRepository) GetUserChatRooms(userID uint, limit, offset int) ([]model.ChatRoom, int64, error) {
	var rooms []model.ChatRoom
	var total int64

	query := r.db.Model(&model.ChatRoom{}).
		Where(
			r.db.Where("user1_id = ? AND user1_left_at IS NULL", userID).
				Or("user2_id = ? AND user2_left_at IS NULL", userID),
		).
		Preload("User1").
		Preload("User2")

	// 총 개수
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 최신 메시지 순으로 정렬
	if err := query.
		Order("last_message_at DESC NULLS LAST, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rooms).Error; err != nil {
		return nil, 0, err
	}

	return rooms, total, nil
}

// UpdateChatRoomLastMessage 채팅방의 마지막 메시지 정보 업데이트
func (r *chatRepository) UpdateChatRoomLastMessage(roomID uint, messageID uint, content string, timestamp time.Time) error {
	return r.db.Model(&model.ChatRoom{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"last_message_id":      messageID,
			"last_message_content": content,
			"last_message_at":      timestamp,
		}).Error
}

// IncrementUnreadCount 읽지 않은 메시지 수 증가
func (r *chatRepository) IncrementUnreadCount(roomID uint, recipientID uint) error {
	var room model.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return err
	}

	// recipientID가 user1이면 user1_unread_count 증가, user2이면 user2_unread_count 증가
	if room.User1ID == recipientID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user1_unread_count", gorm.Expr("user1_unread_count + 1")).Error
	} else if room.User2ID == recipientID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user2_unread_count", gorm.Expr("user2_unread_count + 1")).Error
	}

	return nil
}

// ResetUnreadCount 읽지 않은 메시지 수 초기화
func (r *chatRepository) ResetUnreadCount(roomID uint, userID uint) error {
	var room model.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return err
	}

	if room.User1ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user1_unread_count", 0).Error
	} else if room.User2ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user2_unread_count", 0).Error
	}

	return nil
}

// CreateMessage 메시지 생성
func (r *chatRepository) CreateMessage(message *model.Message) error {
	return r.db.Create(message).Error
}

// GetMessageByID 메시지 ID로 조회
func (r *chatRepository) GetMessageByID(id uint) (*model.Message, error) {
	var message model.Message
	if err := r.db.Preload("Sender").First(&message, id).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

// GetChatRoomMessages 채팅방의 메시지 목록 조회
func (r *chatRepository) GetChatRoomMessages(roomID uint, limit, offset int) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	query := r.db.Model(&model.Message{}).
		Where("chat_room_id = ?", roomID).
		Preload("Sender")

	// 총 개수
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 시간 순으로 정렬 (오래된 메시지부터)
	if err := query.
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// MarkMessagesAsRead 채팅방의 메시지를 읽음 처리
func (r *chatRepository) MarkMessagesAsRead(roomID uint, recipientID uint) error {
	now := time.Now()
	return r.db.Model(&model.Message{}).
		Where("chat_room_id = ? AND sender_id != ? AND is_read = ?", roomID, recipientID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// GetUnreadMessageCount 읽지 않은 메시지 수 조회
func (r *chatRepository) GetUnreadMessageCount(roomID uint, userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Message{}).
		Where("chat_room_id = ? AND sender_id != ? AND is_read = ?", roomID, userID, false).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// LeaveChatRoom 채팅방 나가기 (soft delete for specific user)
func (r *chatRepository) LeaveChatRoom(roomID uint, userID uint) error {
	var room model.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return err
	}

	now := time.Now()
	if room.User1ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user1_left_at", now).Error
	} else if room.User2ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user2_left_at", now).Error
	}

	return nil
}

// RejoinChatRoom 채팅방 다시 참여 (나간 상태 해제)
func (r *chatRepository) RejoinChatRoom(roomID uint, userID uint) error {
	var room model.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return err
	}

	if room.User1ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user1_left_at", nil).Error
	} else if room.User2ID == userID {
		return r.db.Model(&model.ChatRoom{}).
			Where("id = ?", roomID).
			Update("user2_left_at", nil).Error
	}

	return nil
}

// DeleteChatRoomIfBothLeft 양쪽 사용자 모두 나간 경우 채팅방 삭제
func (r *chatRepository) DeleteChatRoomIfBothLeft(roomID uint) error {
	var room model.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return err
	}

	// 양쪽 모두 나갔으면 실제 삭제 (hard delete)
	if room.User1LeftAt != nil && room.User2LeftAt != nil {
		return r.db.Unscoped().Delete(&model.ChatRoom{}, roomID).Error
	}

	return nil
}

// SearchMessages 사용자의 모든 채팅방에서 메시지 검색
func (r *chatRepository) SearchMessages(userID uint, keyword string, limit, offset int) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	// 사용자가 참여한 채팅방 ID 목록 가져오기
	var roomIDs []uint
	if err := r.db.Model(&model.ChatRoom{}).
		Where("(user1_id = ? AND user1_left_at IS NULL) OR (user2_id = ? AND user2_left_at IS NULL)", userID, userID).
		Pluck("id", &roomIDs).Error; err != nil {
		return nil, 0, err
	}

	if len(roomIDs) == 0 {
		return []model.Message{}, 0, nil
	}

	// 키워드로 메시지 검색
	query := r.db.Model(&model.Message{}).
		Where("chat_room_id IN ?", roomIDs).
		Where("content LIKE ?", "%"+keyword+"%").
		Preload("Sender").
		Preload("ChatRoom")

	// 전체 개수 조회
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 메시지 조회 (최신순)
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// UpdateMessage 메시지 수정
func (r *chatRepository) UpdateMessage(messageID uint, content string) error {
	now := time.Now()
	return r.db.Model(&model.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"content":    content,
			"is_edited":  true,
			"edited_at":  now,
			"updated_at": now,
		}).Error
}

// DeleteMessage 메시지 삭제 (soft delete)
func (r *chatRepository) DeleteMessage(messageID uint, deletedBy uint) error {
	return r.db.Model(&model.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_by": deletedBy,
			"content":    "삭제된 메시지입니다",
		}).Error
}
